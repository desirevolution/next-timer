//go:build !lite

package client

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/desirevolution/next-timer/internal/protocol"
)

func sampleTUI() tuiModel {
	now := time.Now()
	return tuiModel{
		now: now, width: 90, height: 40,
		snap: protocol.Snapshot{
			ServerNowMS: now.UnixMilli(), FocusID: "2",
			Timers: []protocol.TimerView{
				{ID: "2", Name: "Kaffee", Status: "running", DeadlineMS: now.Add(5 * time.Minute).UnixMilli(), DurationMS: 300000},
				{ID: "5", Name: "Workshop", Status: "running", DeadlineMS: now.Add(20 * time.Minute).UnixMilli(), DurationMS: 1200000},
			},
			Templates: []protocol.Template{{ID: "fav", Name: "Kaffee", Duration: "5m"}, {ID: "fav2", Name: "Arbeit", Duration: "25m"}},
		},
	}
}
func key(s string) tea.KeyPressMsg {
	if s == "esc" {
		return tea.KeyPressMsg(tea.Key{Code: 27})
	}
	if s == "enter" {
		return tea.KeyPressMsg(tea.Key{Code: 13})
	}
	if s == "tab" {
		return tea.KeyPressMsg(tea.Key{Code: 9})
	}
	if s == "space" {
		return tea.KeyPressMsg(tea.Key{Code: 32})
	}
	return tea.KeyPressMsg(tea.Key{Code: []rune(s)[0], Text: s})
}
func updated(m tuiModel, k string) (tuiModel, tea.Cmd) {
	next, cmd := m.Update(key(k))
	return next.(tuiModel), cmd
}
func TestTUIFavoriteKeyStartsFullscreen(t *testing.T) {
	m, cmd := updated(sampleTUI(), "1")
	if cmd == nil || m.mode != overviewMode {
		t.Fatal("favorite key should dispatch start command")
	}
	// Simulate a successful HTTP create response.
	v := protocol.TimerView{ID: "8", Name: "Kaffee", Status: "running"}
	next, _ := m.Update(actionMsg{started: &v})
	m = next.(tuiModel)
	if m.mode != watchMode || m.watchID != "8" || !m.pendingWatch {
		t.Fatalf("wrong mode after start: %+v", m)
	}
}
func TestTUIStopNeedsConfirmation(t *testing.T) {
	m := sampleTUI()
	m.section = timersSection
	m.selectedID = "2"
	m, cmd := updated(m, "x")
	if cmd != nil || m.mode != confirmStopMode || m.confirmID != "2" {
		t.Fatalf("bad confirm: %+v", m)
	}
	m, cmd = updated(m, "esc")
	if cmd != nil || m.mode != overviewMode || m.confirmID != "" {
		t.Fatal("esc must abort stopping")
	}
	m, _ = updated(m, "x")
	m, cmd = updated(m, "enter")
	if cmd == nil || m.mode != overviewMode {
		t.Fatal("enter should dispatch stop")
	}
}
func TestTUIDoneFollowsNextPrimary(t *testing.T) {
	m := sampleTUI().newWatch("2")
	finished := m.snap
	finished.FocusID = "5"
	finished.Timers[0].Status = "finished"
	m = m.handleSnapshot(finished)
	if m.mode != watchMode || m.doneSince.IsZero() {
		t.Fatal("no finished stage")
	}
	if m.target() != nil {
		t.Fatal("finished watch must not control another timer")
	}
	m.doneSince = time.Now().Add(-3 * time.Second)
	m = m.advanceAfterDone()
	if m.mode != watchMode || m.watchID != "5" || !m.doneSince.IsZero() {
		t.Fatalf("did not follow next primary: %+v", m)
	}
}
func TestTUIDoneWithNoOtherTimerReturnsOverview(t *testing.T) {
	m := sampleTUI().newWatch("2")
	s := m.snap
	s.FocusID = ""
	s.Timers = nil
	m = m.handleSnapshot(s)
	m.doneSince = time.Now().Add(-3 * time.Second)
	m = m.advanceAfterDone()
	if m.mode != overviewMode {
		t.Fatalf("mode %v", m.mode)
	}
}
func TestTUISelectedTimerVsPrimary(t *testing.T) {
	m := sampleTUI()
	if m.target().ID != "2" {
		t.Fatal("default must control primary")
	}
	m, _ = updated(m, "down") // select Workshop
	if m.target().ID != "5" {
		t.Fatal("navigation must select other timer")
	}
	m, _ = updated(m, "tab") // select favorites: primary again
	if m.target().ID != "2" {
		t.Fatal("leaving timer list must clear selection")
	}
}
func TestTUIUrgentColorAndPaused(t *testing.T) {
	now := time.Now()
	tm := protocol.TimerView{Name: "Kaffee", Status: "running", DeadlineMS: now.Add(8 * time.Second).UnixMilli(), DurationMS: 60000}
	var b strings.Builder
	drawTimer(&b, tm, now, 90, true)
	if !strings.Contains(b.String(), "\x1b[1;31m") {
		t.Fatal("no red warning for last 10s")
	}
	b.Reset()
	tm.Status = "paused"
	tm.RemainingMS = 8000
	drawTimer(&b, tm, now, 90, true)
	if strings.Contains(b.String(), "\x1b[1;31m") {
		t.Fatal("paused countdown should not indicate active urgency")
	}
}

func TestFullScreenXIsNoLongerAStopShortcut(t *testing.T) {
	for _, attached := range []bool{false, true} {
		m := sampleTUI().newWatch("2")
		m.cliLaunch = attached
		next, cmd := updated(m, "x")
		if cmd != nil || next.mode != watchMode || next.confirmID != "" {
			t.Fatalf("fullscreen x should do nothing (attached=%v): %+v", attached, next)
		}
	}
}

func TestSnapshotClockSkewUsesRequestMidpoint(t *testing.T) {
	m := sampleTUI()
	start := time.Now()
	received := start.Add(200 * time.Millisecond)
	s := m.snap
	s.ServerNowMS = start.Add(500 * time.Millisecond).UnixMilli()
	updated, _ := m.Update(snapshotMsg{snap: s, requestedAt: start, receivedAt: received})
	m = updated.(tuiModel)
	want := time.UnixMilli(s.ServerNowMS).Sub(start.Add(100 * time.Millisecond))
	if d := m.skew - want; d < -2*time.Millisecond || d > 2*time.Millisecond {
		t.Fatalf("midpoint skew = %s, expected %s", m.skew, want)
	}
}
