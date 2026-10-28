package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAlarmWebhookOnlyOnFinish(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]any
	received := make(chan struct{}, 3)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("invalid request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Error(err)
		}
		mu.Lock()
		got = append(got, data)
		mu.Unlock()
		received <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer endpoint.Close()
	s, err := NewService(filepath.Join(t.TempDir(), "templates.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	h := NewAlarmWebhook(endpoint.URL)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { h.Run(ctx, s); close(done) }()
	// Ensure subscription is installed before generating events.
	deadline := time.Now().Add(time.Second)
	for {
		s.mu.Lock()
		n := len(s.subscribers)
		s.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("webhook not subscribed")
		}
		time.Sleep(time.Millisecond)
	}
	timer := s.Start("Kaffee", time.Minute)
	if _, err := s.Control(timer.ID, "pause", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Control(timer.ID, "resume", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Control(timer.ID, "adjust", -time.Hour); err != nil {
		t.Fatal(err)
	}
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("no finished webhook")
	}
	mu.Lock()
	n := len(got)
	p := got[0]
	mu.Unlock()
	if n != 1 || p["event"] != "timer.finished" {
		t.Fatalf("expected only finish event, got %+v", got)
	}
	if p["event_id"] == "" || p["timestamp"] == "" {
		t.Fatalf("missing id/time: %+v", p)
	}
	tm, ok := p["timer"].(map[string]any)
	if !ok || tm["id"] != timer.ID || tm["status"] != "finished" {
		t.Fatalf("bad timer payload: %+v", p)
	}
	time.Sleep(40 * time.Millisecond)
	mu.Lock()
	n = len(got)
	mu.Unlock()
	if n != 1 {
		t.Fatalf("duplicate / non-finish webhook, count %d", n)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("webhook did not shut down")
	}
}

func TestAlarmWebhookNon2xx(t *testing.T) {
	ep := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer ep.Close()
	h := NewAlarmWebhook(ep.URL)
	if err := h.send(context.Background(), TimerView{ID: "1", FinishedAtMS: time.Now().UnixMilli()}); err == nil {
		t.Fatal("HTTP 503 was ignored")
	}
}
