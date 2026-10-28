package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type awtrixCall struct {
	method, path string
	data         map[string]any
}

func TestAWTRIXNGNotificationsLifecycle(t *testing.T) {
	var mu sync.Mutex
	calls := []awtrixCall{}
	dev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				t.Errorf("decode: %v", err)
			}
		}
		mu.Lock()
		calls = append(calls, awtrixCall{r.Method, r.URL.Path, data})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer dev.Close()
	a := NewAwtrix(dev.URL)
	for _, tc := range []struct {
		remain time.Duration
		text   string
		bar    string
		bg     string
	}{
		{20 * time.Second, "#FFFFFF", "#00FF00", "#000000"},
		{8 * time.Second, "#FF0000", "#FF0000", "#280000"},
		{30 * time.Second, "#FFFFFF", "#00FF00", "#000000"},
	} {
		s := Snapshot{FocusID: "1", Timers: []TimerView{{ID: "1", Name: "Kaffee", Status: "running", DurationMS: 60000, DeadlineMS: time.Now().Add(tc.remain).UnixMilli()}}}
		a.Render(context.Background(), s)
		mu.Lock()
		c := calls[len(calls)-1]
		mu.Unlock()
		if c.path != "/api/v1/notifications" || c.method != http.MethodPost {
			t.Fatalf("unexpected NG endpoint: %+v", c)
		}
		for k, want := range map[string]any{"textColor": tc.text, "progressColor": tc.bar, "backgroundColor": tc.bg, "durationMs": float64(2500), "stack": false, "name": "nexttimer-live"} {
			if !reflect.DeepEqual(c.data[k], want) {
				t.Errorf("%v: %s: got %v, want %v", tc.remain, k, c.data[k], want)
			}
		}
	}
	// Paused/idle: dismiss only our own named notification, never 'active'.
	a.Render(context.Background(), Snapshot{})
	mu.Lock()
	last := calls[len(calls)-1]
	count := len(calls)
	mu.Unlock()
	if last.method != http.MethodDelete || last.path != "/api/v1/notifications/nexttimer-live" {
		t.Fatalf("dismiss: %+v", last)
	}
	a.Render(context.Background(), Snapshot{})
	mu.Lock()
	after := len(calls)
	mu.Unlock()
	if after != count {
		t.Fatalf("idle triggered unnecessary requests: %d vs %d", after, count)
	}
}

func TestAWTRIXNGFinishAndOverlay(t *testing.T) {
	var calls []awtrixCall
	dev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&p)
		}
		calls = append(calls, awtrixCall{r.Method, r.URL.Path, p})
		w.WriteHeader(http.StatusOK)
	}))
	defer dev.Close()
	a := NewAwtrix(dev.URL)
	snap := Snapshot{FocusID: "1", Timers: []TimerView{{ID: "1", Name: "Tee", Status: "running", DurationMS: 30000, DeadlineMS: time.Now().Add(20 * time.Second).UnixMilli()}}}
	a.Render(context.Background(), snap)
	if err := a.notifyFinished(context.Background(), TimerView{ID: "2", Name: "Kaffee"}, false); err != nil {
		t.Fatal(err)
	}
	p := calls[len(calls)-1].data
	for k, want := range map[string]any{"stack": false, "durationMs": float64(5000), "effect": "Fireworks", "effectSpeed": float64(1), "textColor": "#FF0000"} {
		if p[k] != want {
			t.Fatalf("finish: %s = %v, expected %v", k, p[k], want)
		}
	}
	a.overlayUntil = time.Now().Add(5 * time.Second)
	n := len(calls)
	a.Render(context.Background(), snap)
	if len(calls) != n {
		t.Fatal("display update interrupted the finish effect")
	}
	a.SetFinishEffect("none")
	if err := a.notifyFinished(context.Background(), TimerView{Name: "Pause"}, true); err != nil {
		t.Fatal(err)
	}
	p = calls[len(calls)-1].data
	if p["stack"] != true {
		t.Fatalf("queued finish missing: %v", p)
	}
	if _, ok := p["effect"]; ok {
		t.Fatal("effect should be omitted for none")
	}
}

func TestAWTRIXNGRejectionIncludesResponse(t *testing.T) {
	dev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"field":"effect","code":"validationFailed"}`))
	}))
	defer dev.Close()
	a := NewAwtrix(dev.URL)
	err := a.notifyFinished(context.Background(), TimerView{Name: "X"}, false)
	if err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("missing diagnostic response: %v", err)
	}
}

func TestAWTRIXNGCustomFinishOptions(t *testing.T) {
	var recorded map[string]any
	dev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notifications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&recorded); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer dev.Close()
	a := NewAwtrix(dev.URL)
	a.SetFinishEffect("ColorWaves")
	a.SetFinishOptions(0.3, 10*time.Second)
	if err := a.notifyFinished(context.Background(), TimerView{Name: "Kaffee"}, false); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{"effect": "ColorWaves", "effectSpeed": 0.3, "durationMs": float64(10000)} {
		if !reflect.DeepEqual(recorded[k], want) {
			t.Errorf("%s: got %v, want %v", k, recorded[k], want)
		}
	}
	a.SetFinishEffect("none")
	recorded = nil // Decode into a fresh map so omitted fields cannot linger from the earlier payload.
	if err := a.notifyFinished(context.Background(), TimerView{Name: "Kaffee"}, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := recorded["effectSpeed"]; ok {
		t.Error("effectSpeed should be omitted when effect is disabled")
	}
}
