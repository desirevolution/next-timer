//go:build !lite

package client

import (
	tea "charm.land/bubbletea/v2"
	"testing"
	"time"
)

func TestDirectCLILaunchExitsOnEscapeAndQ(t *testing.T) {
	for _, k := range []string{"esc", "q"} {
		m := sampleTUI().newWatch("2")
		m.cliLaunch = true
		next, cmd := updated(m, k)
		if cmd == nil || next.mode != watchMode {
			t.Fatalf("%s should quit attached focus", k)
		}
	}
}

func TestDirectCLILaunchQuitsImmediatelyWhenFinished(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.cliLaunch = true
	s := m.snap
	s.Timers[0].Status = "finished"
	s.FocusID = "5"
	next, cmd := m.Update(snapshotMsg{snap: s})
	m = next.(tuiModel)
	if cmd == nil || m.watchID != "2" || !m.doneSince.IsZero() {
		t.Fatalf("finished CLI timer should quit immediately, not show FERTIG/follow another timer: %+v", m)
	}
	if !m.attachedTimerFinished() {
		t.Fatal("finished CLI timer not recognized")
	}
	_ = tea.Quit
}

func TestDirectCLILaunchDoesNotQuitBeforeServerConfirmsFinish(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.cliLaunch = true
	// A local zero alone is not proof the server has finished. Otherwise
	// the deferred owned-timer cleanup could turn an alarm into a stop.
	m.now = m.now.Add(10 * time.Minute)
	if m.attachedTimerFinished() {
		t.Fatal("CLI client must wait for the server's finished status")
	}
}

func TestTermdownStyleIsCurved(t *testing.T) {
	if len(termdownPaths('0')) < 20 || len(termdownPaths('7')) < 10 {
		t.Fatal("expected curved vector glyphs")
	}
	if a, b := largeClockAdaptive("00:07", 110, 29), largeClockAdaptive("00:08", 110, 29); a == b {
		t.Fatal("different times rendered identically")
	}
}
