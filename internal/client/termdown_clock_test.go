package client

import (
	"strings"
	"testing"
)

func TestTermdownClockAdaptiveBounds(t *testing.T) {
	for _, tc := range []struct {
		time          string
		width, height int
	}{
		{"04:32", 120, 40}, {"00:07", 80, 25}, {"01:20:45", 160, 35},
		{"12:59", 45, 15}, {"04:32", 28, 8}, {"04:32", 20, 6},
	} {
		got := largeClockAdaptive(tc.time, tc.width, tc.height)
		lines := strings.Split(got, "\n")
		if len(lines) > tc.height {
			t.Fatalf("%s: rendered height %d > %d", tc.time, len(lines), tc.height)
		}
		for _, line := range lines {
			if len([]rune(line)) > tc.width {
				t.Fatalf("%s: rendered width %d > %d", tc.time, len([]rune(line)), tc.width)
			}
		}
	}
}

func TestTermdownClockShapes(t *testing.T) {
	a := largeClockAdaptive("00:07", 110, 28)
	b := largeClockAdaptive("00:08", 110, 28)
	if a == b {
		t.Fatal("different digits rendered identically")
	}
	if !strings.Contains(a, "█") && !strings.Contains(a, "▌") {
		t.Fatal("expected filled quadrant characters")
	}
	if got := largeClockAdaptive("not-a-clock", 100, 30); got != "not-a-clock" {
		t.Fatal("invalid clock should be rendered as text")
	}
}
