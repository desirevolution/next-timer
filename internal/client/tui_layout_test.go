//go:build !lite

package client

import (
	"strings"
	"testing"
	"time"
)

func TestAdaptiveClockStaysInsideWidthAndHeight(t *testing.T) {
	for _, test := range []struct{ width, height int }{{120, 40}, {80, 25}, {45, 16}, {28, 8}} {
		rendered := largeClockAdaptive("04:32", test.width, test.height)
		lines := strings.Split(rendered, "\n")
		if len(lines) > test.height {
			t.Fatalf("height %d exceeds %d", len(lines), test.height)
		}
		for _, line := range lines {
			if len([]rune(line)) > test.width {
				t.Fatalf("line width exceeds %d", test.width)
			}
		}
	}
}
func TestRedFocusAndCleanOverview(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.snap.Timers[0].DeadlineMS = m.now.Add(8 * time.Second).UnixMilli()
	urgent := renderTUI(m)
	if strings.Contains(urgent, "LETZTE 10 SEKUNDEN") || !strings.Contains(urgent, "● LÄUFT") || !strings.Contains(urgent, "5 Min.") {
		t.Fatalf("focus header/status/length is wrong: %s", urgent)
	}
	if strings.Contains(urgent, "NEXTTIMER  /  FOKUS") || strings.Contains(urgent, "x Stop") {
		t.Fatalf("unwanted fullscreen controls in view: %s", urgent)
	}
	m.mode = overviewMode
	home := renderTUI(m)
	if !strings.Contains(home, "FAVORITEN") || !strings.Contains(home, "WEITERE TIMER") || !strings.Contains(home, "Kaffee") {
		t.Fatalf("incomplete dashboard: %s", home)
	}
}
func TestFocusShowsDoneThenFollowsNext(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.doneName = "Kaffee"
	m.doneSince = time.Now()
	if !strings.Contains(renderTUI(m), "F E R T I G") {
		t.Fatal("missing done message")
	}
	m.snap.FocusID = "5"
	m.doneSince = time.Now().Add(-3 * time.Second)
	m = m.advanceAfterDone()
	if m.watchID != "5" || m.mode != watchMode {
		t.Fatalf("unexpected next watch: %+v", m)
	}
}

func TestOverviewPanelsAndHighlightedSelection(t *testing.T) {
	m := sampleTUI()
	m.section = favoritesSection
	body := renderTUI(m)
	for _, part := range []string{"FAVORITEN", "WEITERE TIMER", "COUNTDOWN", "Kaffee", "1", "2"} {
		if !strings.Contains(body, part) {
			t.Fatalf("panel missing %q", part)
		}
	}
	m.section = timersSection
	m.selectedID = "5"
	detail := renderTUI(m)
	if !strings.Contains(detail, "Workshop") {
		t.Fatal("selected timer not visible in detail panel")
	}
}
func TestOverviewFallsBackWhenSmall(t *testing.T) {
	m := sampleTUI()
	m.width = 50
	m.height = 15
	body := renderTUI(m)
	if !strings.Contains(body, "FAVORITEN") {
		t.Fatal("small fallback lost favorites")
	}
}

func TestAttachedFocusNeverShowsDoneIntermission(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.cliLaunch = true
	m.doneSince = time.Now()
	m.doneName = "Kaffee"
	if strings.Contains(renderTUI(m), "F E R T I G") {
		t.Fatal("CLI focus must not display a finished intermission")
	}
}

func TestFocusShowsDurationAndLocalHint(t *testing.T) {
	m := sampleTUI().newWatch("2")
	m.snap.Timers[0].Local = true
	view := renderTUI(m)
	if !strings.Contains(view, "Kaffee") || !strings.Contains(view, "5 Min.") || !strings.Contains(view, "lokal") {
		t.Fatalf("no subtle timer kind/duration label: %s", view)
	}
}

func TestShortDurationLabel(t *testing.T) {
	for _, tc := range []struct {
		millis int64
		want   string
	}{
		{15000, "15 Sek."}, {300000, "5 Min."}, {330000, "5:30 Min."}, {3600000, "1 Std."}, {5400000, "1 Std. 30 Min."},
	} {
		if got := shortDurationLabel(tc.millis); got != tc.want {
			t.Fatalf("duration %d: %q instead of %q", tc.millis, got, tc.want)
		}
	}
}
