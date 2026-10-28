package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestStopOwnedServerTimerOnlyActiveOwnedTimer(t *testing.T) {
	var stopped atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/state":
			_, _ = fmt.Fprint(w, `{"server_now_ms":0,"timers":[{"id":"21","status":"running"},{"id":"22","status":"running"}]}`)
		case r.Method == "POST" && r.URL.Path == "/api/timers/21/stop":
			stopped.Add(1)
			w.WriteHeader(204)
		default:
			http.Error(w, "not expected", 404)
		}
	}))
	defer ts.Close()
	old := serverURL
	SetServerURL(ts.URL)
	defer SetServerURL(old)
	if err := stopOwnedServerTimer("21"); err != nil {
		t.Fatal(err)
	}
	if stopped.Load() != 1 {
		t.Fatalf("expected one stop, got %d", stopped.Load())
	}
}

func TestStopOwnedServerTimerSkipsFinished(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/state" || r.Method != "GET" {
			http.Error(w, "no stop expected", 500)
			return
		}
		_, _ = fmt.Fprint(w, `{"timers":[{"id":"4","status":"finished"}]}`)
	}))
	defer ts.Close()
	old := serverURL
	SetServerURL(ts.URL)
	defer SetServerURL(old)
	if err := stopOwnedServerTimer("4"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("finished timer should not be stopped, calls %d", calls.Load())
	}
}

func TestConflictingExitModesRejected(t *testing.T) {
	for _, opt := range []Options{{Detach: true, KeepRunning: true}, {Local: true, KeepRunning: true}, {Local: true, Detach: true}} {
		err := RunWithClientOptions([]string{"1m", "Kaffee"}, opt)
		if err == nil {
			t.Fatalf("expected conflict error for %+v", opt)
		}
	}
	if err := RunWithClientOptions([]string{"ls"}, Options{KeepRunning: true}); err == nil || !strings.Contains(err.Error(), "direkten Timerstart") {
		t.Fatalf("unexpected keep-running error: %v", err)
	}
}

func TestAttachedSessionExitPolicy(t *testing.T) {
	var stopped atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/state" {
			_, _ = fmt.Fprint(w, `{"timers":[{"id":"99","status":"running"}]}`)
			return
		}
		if r.Method == "POST" && r.URL.Path == "/api/timers/99/stop" {
			stopped.Add(1)
			w.WriteHeader(204)
			return
		}
		http.Error(w, "unexpected", 500)
	}))
	defer ts.Close()
	old := serverURL
	SetServerURL(ts.URL)
	defer SetServerURL(old)
	if err := runAttachedSession("99", true, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if stopped.Load() != 0 {
		t.Fatal("--keep-running must not send stop")
	}
	if err := runAttachedSession("99", false, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if stopped.Load() != 1 {
		t.Fatal("default exit must stop timer")
	}
	simulatedError := fmt.Errorf("TUI failed")
	if err := runAttachedSession("99", false, func() error { return simulatedError }); err != simulatedError {
		t.Fatalf("expected original error, got %v", err)
	}
	if stopped.Load() != 2 {
		t.Fatal("TUI errors must also stop owned timer")
	}
}
