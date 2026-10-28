package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// The AWTRIX renderer must change seconds at the timer's actual second edges,
// instead of drifting with an unrelated one-second ticker.
func TestAWTRIXDeadlineAlignedDisplayAndDeduplication(t *testing.T) {
	var mu sync.Mutex
	var frames []map[string]any
	device := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode: %v", err)
		}
		mu.Lock()
		frames = append(frames, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer device.Close()
	a := NewAwtrix(device.URL)
	a.SetDisplayLead(100 * time.Millisecond)
	base := time.Unix(1800000000, 0)
	deadline := base.Add(10*time.Second + 250*time.Millisecond)
	s := Snapshot{FocusID: "1", Timers: []TimerView{{ID: "1", Status: "running", Name: "Kaffee", DeadlineMS: deadline.UnixMilli(), DurationMS: 15000}}}
	// 10.25s left -> 11 shown. 150ms later is the (100ms advanced)
	// boundary for displaying 10, and the text and bar turn red together.
	a.renderAt(context.Background(), s, base)
	a.renderAt(context.Background(), s, base.Add(100*time.Millisecond))
	mu.Lock()
	if len(frames) != 1 || frames[0]["text"] != "00:11" {
		t.Fatalf("first edge and de-duplication: %+v", frames)
	}
	mu.Unlock()
	a.renderAt(context.Background(), s, base.Add(151*time.Millisecond))
	a.renderAt(context.Background(), s, base.Add(400*time.Millisecond))
	mu.Lock()
	if len(frames) != 2 || frames[1]["text"] != "00:10" || frames[1]["textColor"] != "#FF0000" {
		t.Fatalf("wrong next edge/color or duplicate: %+v", frames)
	}
	mu.Unlock()
	a.renderAt(context.Background(), s, base.Add(1151*time.Millisecond))
	mu.Lock()
	defer mu.Unlock()
	if len(frames) != 3 || frames[2]["text"] != "00:09" {
		t.Fatalf("next edge wasn't updated: %+v", frames)
	}
}

func TestAWTRIXTimerAdjustmentRefreshesSameSecond(t *testing.T) {
	var calls int
	device := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer device.Close()
	a := NewAwtrix(device.URL)
	base := time.Unix(1800000000, 0)
	s := Snapshot{FocusID: "1", Timers: []TimerView{{ID: "1", Status: "running", DeadlineMS: base.Add(25 * time.Second).UnixMilli(), DurationMS: 30000}}}
	a.renderAt(context.Background(), s, base)
	s.Timers[0].DurationMS = 60000 // adjust can change progress without changing the displayed second
	a.renderAt(context.Background(), s, base)
	if calls != 2 {
		t.Fatalf("expected timer adjustment to update, got %d calls", calls)
	}
}
