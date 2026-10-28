package client

import (
	"strings"
	"testing"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func TestWatchArgs(t *testing.T) {
	sel, compact, err := watchArgs([]string{"--compact", "hello"})
	if err != nil || !compact || sel != "hello" {
		t.Fatalf("watch args: %s %t %v", sel, compact, err)
	}
	if _, _, err = watchArgs([]string{"a", "b"}); err == nil {
		t.Fatal("multiple selector accepted")
	}
}
func TestRenderWatch(t *testing.T) {
	now := time.Now()
	timer := protocol.TimerView{ID: "x", Name: "Kaffee", Status: "running", DeadlineMS: now.Add(2 * time.Minute).UnixMilli(), DurationMS: (5 * time.Minute).Milliseconds()}
	compact := renderWatch(timer, now, true, true)
	if !strings.Contains(compact, "02:00") || !strings.Contains(compact, "Kaffee") {
		t.Fatalf("unexpected compact %q", compact)
	}
	big := renderWatch(timer, now, false, true)
	if !strings.Contains(big, "██") || !strings.Contains(big, "Kaffee") || !strings.Contains(big, "[") {
		t.Fatalf("unexpected big display %q", big)
	}
}
