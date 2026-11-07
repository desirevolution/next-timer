package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func TestQuickParserSharedCases(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "scripts", "quick-input-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Input      string `json:"input"`
		Kind       string `json:"kind"`
		DurationMS int64  `json:"duration_ms"`
		Name       string `json:"name"`
		Shortcut   string `json:"shortcut"`
		Error      bool   `json:"error"`
	}
	if err := json.Unmarshal(b, &cases); err != nil {
		t.Fatal(err)
	}
	for _, item := range cases {
		t.Run(item.Input, func(t *testing.T) {
			result, err := parseQuickInput(item.Input)
			if item.Error {
				if err == nil {
					t.Fatalf("invalid input accepted: %+v", result)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if item.Kind == "favorite" {
				if result.Shortcut != item.Shortcut {
					t.Fatalf("shortcut: %q != %q", result.Shortcut, item.Shortcut)
				}
			} else if result.Shortcut != "" || result.Duration.Milliseconds() != item.DurationMS || result.Name != item.Name {
				t.Fatalf("parsed: %+v, expected: %+v", result, item)
			}
		})
	}
}

func postQuick(t *testing.T, url, contentType, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(url+"/api/timers/quick", contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode HTTP %d: %v", resp.StatusCode, err)
	}
	return resp.StatusCode, data
}

func postStopNext(t *testing.T, url string) (int, *TimerView) {
	t.Helper()
	resp, err := http.Post(url+"/api/timers/next/stop", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		if len(body) != 0 {
			t.Fatalf("204 unexpectedly contains: %q", body)
		}
		return resp.StatusCode, nil
	}
	var view TimerView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, &view
}

func TestQuickHTTPStartsAndRejectsInvalidRequests(t *testing.T) {
	s := makeService(t)
	web := httptest.NewServer((API{s}).Handler())
	defer web.Close()
	for _, tc := range []struct {
		ctype, body string
	}{
		{"text/plain", "5m Kaffee"},
		{"text/plain; charset=utf-8", "5m Kaffee"},
		{"application/json", `{"input":"5m Kaffee"}`},
	} {
		status, body := postQuick(t, web.URL, tc.ctype, tc.body)
		if status != 201 || body["name"] != "Kaffee" || body["duration_ms"] != float64(300000) || body["status"] != "running" {
			t.Fatalf("HTTP %d: %+v", status, body)
		}
	}
	if len(s.Snapshot().Timers) != 3 {
		t.Fatal("identical quick requests must create independent timers")
	}

	invalid := []struct {
		ctype, body string
		status      int
		code        string
	}{
		{"text/plain", "???", 422, "INVALID_DURATION"},
		{"text/plain", "5mKaffee", 422, "INVALID_INPUT"},
		{"text/plain", "#12", 422, "INVALID_SHORTCUT"},
		{"text/plain", "5m " + strings.Repeat("x", 81), 422, "NAME_TOO_LONG"},
		{"text/plain", "#1", 404, "FAVORITE_NOT_FOUND"},
		{"application/json", `{"input":42}`, 422, "INVALID_INPUT"},
		{"application/json", `{}`, 422, "INVALID_INPUT"},
		{"application/json", `{"input":null}`, 422, "INVALID_INPUT"},
		{"application/json", `{"input":"5m","other":1}`, 422, "INVALID_INPUT"},
		{"application/json", `{"input":"5m"}{}`, 422, "INVALID_INPUT"},
		{"application/json", `[]`, 422, "INVALID_INPUT"},
		{"application/xml", `5m`, 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"text/plain; charset=latin1", `5m`, 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"text/plain", strings.Repeat("x", 8193), 413, "INPUT_TOO_LARGE"},
	}
	for i, tc := range invalid {
		status, got := postQuick(t, web.URL, tc.ctype, tc.body)
		if status != tc.status || got["code"] != tc.code {
			t.Errorf("case %d HTTP %d %+v; want %d %s", i, status, got, tc.status, tc.code)
		}
	}
	if len(s.Snapshot().Timers) != 3 {
		t.Fatal("invalid quick input mutated timer state")
	}
}

func TestQuickFavoriteUsesServerOptionsAndIgnoresBrowserOnly(t *testing.T) {
	s := makeService(t)
	volume := 0.25
	alarm := &protocol.BrowserAlarm{Tone: "gong", Repeat: "once", Visual: true, System: false, Volume: &volume}
	sound := false
	notify := true
	favorite, err := s.AddTemplateConfigured(Template{Name: "Brühen", Duration: "2m", Shortcut: "1", Sound: &sound, Notify: &notify, Alarm: alarm})
	if err != nil {
		t.Fatal(err)
	}
	if favorite.Shortcut != "1" {
		t.Fatal("favorite shortcut missing")
	}
	if _, err := s.AddTemplateConfigured(Template{Name: "Browserlokal", Duration: "3m", Shortcut: "0", BrowserOnly: true}); err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer((API{s}).Handler())
	defer web.Close()
	status, got := postQuick(t, web.URL, "text/plain", "#1")
	if status != 201 || got["name"] != "Brühen" || got["sound"] != false || got["notify"] != true || got["duration_ms"] != float64(120000) {
		t.Fatalf("shortcut lost options: %d %+v", status, got)
	}
	if gotAlarm, ok := got["alarm"].(map[string]any); !ok || gotAlarm["tone"] != "gong" {
		t.Fatalf("shortcut lost browser alarm: %+v", got)
	}
	status, got = postQuick(t, web.URL, "application/json", `{"input":"#0"}`)
	if status != 404 || got["code"] != "FAVORITE_NOT_FOUND" {
		t.Fatalf("browser-only template started by REST: %d %+v", status, got)
	}
	if len(s.Snapshot().Timers) != 1 {
		t.Fatal("unexpected timer created")
	}
}

func TestNextStopFindsNextRunningAndIgnoresPaused(t *testing.T) {
	s := makeService(t)
	a := s.Start("Lang", 20*time.Minute)
	b := s.Start("Kurz", 5*time.Minute)
	paused := s.Start("Pausiert", time.Minute)
	if _, err := s.Control(paused.ID, "pause", 0); err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer((API{s}).Handler())
	defer web.Close()
	for _, want := range []string{b.ID, a.ID} {
		status, got := postStopNext(t, web.URL)
		if status != 200 || got == nil || got.ID != want {
			t.Fatalf("stop next returned HTTP %d %+v, want %s", status, got, want)
		}
	}
	status, got := postStopNext(t, web.URL)
	if status != 204 || got != nil {
		t.Fatalf("paused timer must be ignored: %d %+v", status, got)
	}
	if len(s.Snapshot().Timers) != 1 || s.Snapshot().Timers[0].ID != paused.ID {
		t.Fatal("paused timer changed")
	}
}

func TestNextStopExpiresDueTimerBeforeStoppingAnother(t *testing.T) {
	s := makeService(t)
	expired := s.Start("Fällig", time.Minute)
	next := s.Start("Noch aktiv", 5*time.Minute)
	s.mu.Lock()
	s.timers[expired.ID].Deadline = time.Now().Add(-time.Second)
	s.mu.Unlock()
	web := httptest.NewServer((API{s}).Handler())
	defer web.Close()
	status, got := postStopNext(t, web.URL)
	if status != 200 || got.ID != next.ID {
		t.Fatalf("expired timer was stopped: HTTP %d %+v", status, got)
	}
	state := s.Snapshot()
	if len(state.Timers) != 1 || state.Timers[0].ID != expired.ID || state.Timers[0].Status != "finished" {
		t.Fatalf("expired timer state: %+v", state)
	}
	if state.LastFinished == nil || state.LastFinished.Name != "Noch aktiv" {
		t.Fatalf("last completion should use later stop time: %+v", state.LastFinished)
	}
	status, _ = postStopNext(t, web.URL)
	if status != 204 {
		t.Fatalf("expired-only must be no-op: %d", status)
	}
}

func TestNextStopConcurrentlyStopsEachTimerAtMostOnce(t *testing.T) {
	s := makeService(t)
	for i := 0; i < 10; i++ {
		s.Start(fmt.Sprintf("Timer %d", i), time.Duration(i+1)*time.Minute)
	}
	web := httptest.NewServer((API{s}).Handler())
	defer web.Close()
	var wg sync.WaitGroup
	results := make(chan string, 25)
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(web.URL+"/api/timers/next/stop", "text/plain", bytes.NewReader(nil))
			if err != nil {
				results <- "network error " + err.Error()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 204 {
				results <- "no-op"
				return
			}
			var timer TimerView
			if err := json.NewDecoder(resp.Body).Decode(&timer); err != nil || resp.StatusCode != 200 {
				results <- "decode failure"
				return
			}
			results <- timer.ID
		}()
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{}
	numStopped, numNoop := 0, 0
	for result := range results {
		if result == "no-op" {
			numNoop++
		} else if seen[result] {
			t.Fatalf("timer stopped twice: %s", result)
		} else {
			seen[result] = true
			numStopped++
		}
	}
	if numStopped != 10 || numNoop != 15 || len(s.Snapshot().Timers) != 0 {
		t.Fatalf("stopped=%d noop=%d, state=%+v", numStopped, numNoop, s.Snapshot())
	}
}
