package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeService(t *testing.T) *Service {
	t.Helper()
	s, err := NewService(filepath.Join(t.TempDir(), "templates.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestEarliestDeadlineIsFocus(t *testing.T) {
	s := makeService(t)
	long := s.Start("Lang", 20*time.Minute)
	short := s.Start("Kurz", 3*time.Minute)
	if got := s.Snapshot().FocusID; got != short.ID {
		t.Fatalf("focus %q; want %q", got, short.ID)
	}
	if _, err := s.Control(short.ID, "pause", 0); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().FocusID; got != long.ID {
		t.Fatalf("after pause focus %q; want %q", got, long.ID)
	}
	if _, err := s.Control(short.ID, "resume", 0); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().FocusID; got != short.ID {
		t.Fatalf("after resume focus %q; want %q", got, short.ID)
	}
	if _, err := s.Control(short.ID, "adjust", 19*time.Minute); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().FocusID; got != long.ID {
		t.Fatalf("after adjust focus %q; want %q", got, long.ID)
	}
}
func TestFinishedAndCleanup(t *testing.T) {
	s := makeService(t)
	s.finishedTTL = 20 * time.Millisecond
	timer := s.Start("Kurz", time.Second)
	s.mu.Lock()
	s.timers[timer.ID].Deadline = time.Now().Add(-time.Millisecond)
	s.mu.Unlock()
	s.Tick()
	snap := s.Snapshot()
	if snap.FocusID != "" || len(snap.Timers) != 1 || snap.Timers[0].Status != "finished" {
		t.Fatalf("not finished: %+v", snap)
	}
	time.Sleep(25 * time.Millisecond)
	s.Tick()
	if len(s.Snapshot().Timers) != 0 {
		t.Fatal("finished timer not removed")
	}
}
func TestTemplatesPersistTimersDoNot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(file, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	fav, err := s.AddTemplate("Kaffee", "4m")
	if err != nil {
		t.Fatal(err)
	}
	s.Start("Temp", 2*time.Minute)
	s2, err := NewService(file, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	snap := s2.Snapshot()
	if len(snap.Timers) != 0 || len(snap.Templates) != 1 || snap.Templates[0].ID != fav.ID {
		t.Fatalf("wrong restart state: %+v", snap)
	}
	_, err = s2.UpdateTemplate(fav.ID, "Tee", "5m")
	if err != nil {
		t.Fatal(err)
	}
	if s2.Snapshot().Templates[0].Name != "Tee" {
		t.Fatal("template not updated")
	}
}
func TestHTTP(t *testing.T) {
	s := makeService(t)
	server := httptest.NewServer((API{s}).Handler())
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/timers", "application/json", strings.NewReader(`{"name":"Test","duration":"5m"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("start status %d", resp.StatusCode)
	}
	var got TimerView
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	response, err := http.Get(server.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var snap Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if snap.FocusID != got.ID {
		t.Fatalf("API focus: %v", snap)
	}
}

// Extending the remaining duration must also increase the effective total duration.
// Otherwise all clients display 0% progress until time remaining drops below original duration.
func TestAdjustUpdatesDurationAndProgress(t *testing.T) {
	s := makeService(t)
	v := s.Start("Kaffee", 10*time.Minute)
	s.mu.Lock()
	timer := s.timers[v.ID]
	timer.Deadline = time.Now().Add(5 * time.Minute)
	s.mu.Unlock()
	increased, err := s.Control(v.ID, "adjust", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if increased.DurationMS != (20 * time.Minute).Milliseconds() {
		t.Fatalf("duration=%d expected 20m", increased.DurationMS)
	}
	if remaining := time.Until(time.UnixMilli(increased.DeadlineMS)); remaining < 14*time.Minute || remaining > 16*time.Minute {
		t.Fatalf("remaining unexpectedly %s", remaining)
	}
	// 5 elapsed of 20 total should be approx 25 percent, not 0 percent.
	progress := 1 - float64(increased.RemainingMS)/float64(increased.DurationMS)
	if progress < .24 || progress > .26 {
		t.Fatalf("progress %.4f expected ~0.25", progress)
	}
	paused, err := s.Control(v.ID, "pause", 0)
	if err != nil {
		t.Fatal(err)
	}
	reduced, err := s.Control(v.ID, "adjust", -3*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reduced.DurationMS != (17 * time.Minute).Milliseconds() {
		t.Fatalf("paused adjustment duration=%d", reduced.DurationMS)
	}
	if paused.RemainingMS-reduced.RemainingMS != (3 * time.Minute).Milliseconds() {
		t.Fatalf("paused remaining not changed by 3m")
	}
}

func TestFavoriteOrderPersistsAndRejectsInvalidIDs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(file, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddTemplate("A", "1m")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddTemplate("B", "2m")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.AddTemplate("C", "3m")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{c.ID, a.ID, b.ID}
	if err := s.ReorderTemplates(want); err != nil {
		t.Fatal(err)
	}
	if err := s.ReorderTemplates([]string{c.ID, c.ID, a.ID}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := s.ReorderTemplates([]string{c.ID, a.ID}); err == nil {
		t.Fatal("stale list accepted")
	}
	s2, err := NewService(file, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for i, tmpl := range s2.Snapshot().Templates {
		if tmpl.ID != want[i] {
			t.Fatalf("position %d: %q want %q", i, tmpl.ID, want[i])
		}
	}
}

func TestFavoriteOrderHTTP(t *testing.T) {
	s := makeService(t)
	a, _ := s.AddTemplate("A", "1m")
	b, _ := s.AddTemplate("B", "2m")
	apiServer := httptest.NewServer((API{s}).Handler())
	defer apiServer.Close()
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"ids":["` + b.ID + `","` + a.ID + `"]}`, 200},
		{`{"ids":["` + a.ID + `"]}`, 409},
	} {
		req, err := http.NewRequest("PUT", apiServer.URL+"/api/templates/order", strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.status {
			t.Fatalf("status %d want %d", res.StatusCode, tc.status)
		}
	}
	if s.Snapshot().Templates[0].ID != b.ID {
		t.Fatal("API order not applied")
	}
}
