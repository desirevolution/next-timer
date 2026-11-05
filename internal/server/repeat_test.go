package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func TestLastFinishedRepeatAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RepeatLast(); err == nil {
		t.Fatal("empty repeat should fail")
	}
	volume := 0.2
	settings := TimerOptions{Sound: true, Notify: false, Alarm: &protocol.BrowserAlarm{Tone: "bell", Repeat: "once", Visual: true, System: false, Volume: &volume}}
	tea := s.StartConfigured("Tee", 3*time.Minute, settings)
	workshop := s.StartConfigured("Workshop", 12*time.Minute, TimerOptions{Sound: false, Notify: true})
	// Only the latest actual end matters. A timer that finishes first but whose
	// expiry is discovered later must not replace a more recently stopped one.
	s.mu.Lock()
	s.timers[tea.ID].Deadline = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	if _, err := s.Control(workshop.ID, "stop", 0); err != nil {
		t.Fatal(err)
	}
	first := s.Snapshot().LastFinished
	if first == nil || first.Name != "Workshop" {
		t.Fatalf("stop did not record completion: %+v", first)
	}
	s.Tick()
	if got := s.Snapshot().LastFinished; got.Name != "Workshop" {
		t.Fatalf("late tick overwrote newer completion: %+v", got)
	}
	repeated, err := s.RepeatLast()
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Name != "Workshop" || repeated.DurationMS != 12*60*1000 || repeated.Sound || !repeated.Notify {
		t.Fatalf("wrong repeat: %+v", repeated)
	}
	if got := s.Snapshot().LastFinished; got.Name != "Workshop" {
		t.Fatalf("repeat should not count as finish: %+v", got)
	}
	paused := s.StartConfigured("Pause", time.Minute, settings)
	if _, err := s.Control(paused.ID, "pause", 0); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().LastFinished; got.Name != "Workshop" {
		t.Fatalf("pause is not a finish: %+v", got)
	}
	if _, err := s.Control(paused.ID, "stop", 0); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().LastFinished; got.Name != "Pause" {
		t.Fatalf("paused stop missing: %+v", got)
	}
	again, err := s.RepeatLast()
	if err != nil {
		t.Fatal(err)
	}
	if again.Name != "Pause" || again.DurationMS != 60000 || !again.Sound || again.Notify || again.Alarm == nil || again.Alarm.Tone != "bell" {
		t.Fatalf("settings lost: %+v", again)
	}
	restored, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if last := restored.Snapshot().LastFinished; last == nil || last.Name != "Pause" {
		t.Fatalf("completion not persisted: %+v", last)
	}
	if _, err := restored.RepeatLast(); err != nil {
		t.Fatalf("repeat after restart: %v", err)
	}
}

func TestRepeatHTTPAndNaturalExpiration(t *testing.T) {
	s, err := NewService(filepath.Join(t.TempDir(), "templates.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	h := API{Service: s}.Handler()
	empty := httptest.NewRecorder()
	h.ServeHTTP(empty, httptest.NewRequest(http.MethodPost, "/api/repeat", nil))
	if empty.Code != http.StatusNotFound {
		t.Fatalf("empty repeat returned %d", empty.Code)
	}
	started := s.StartConfigured("Kaffee", time.Second, TimerOptions{Sound: true, Notify: true})
	s.mu.Lock()
	s.timers[started.ID].Deadline = time.Now().Add(-time.Millisecond)
	s.mu.Unlock()
	s.Tick()
	if last := s.Snapshot().LastFinished; last == nil || last.Name != "Kaffee" {
		t.Fatalf("natural expiry not captured: %+v", last)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/repeat", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("repeat code %d: %s", rec.Code, rec.Body.String())
	}
	var actual protocol.TimerView
	if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.ID == started.ID || actual.Name != "Kaffee" || actual.Status != "running" || actual.DurationMS != 1000 {
		t.Fatalf("repeat view: %+v", actual)
	}
}
