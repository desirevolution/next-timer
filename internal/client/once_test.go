package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCLIDetachDirectAndFavorite(t *testing.T) {
	var creates atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/state":
			_, _ = fmt.Fprint(w, `{"server_now_ms":0,"templates":[{"id":"fav1","name":"Kaffee","duration":"5m"}],"timers":[]}`)
		case r.Method == "POST" && r.URL.Path == "/api/timers":
			creates.Add(1)
			_, _ = fmt.Fprint(w, `{"id":"1","name":"Kaffee","duration_ms":300000,"status":"running"}`)
		default:
			http.Error(w, "unexpected endpoint", 404)
		}
	}))
	defer ts.Close()
	old := serverURL
	defer SetServerURL(old)
	SetServerURL(ts.URL)
	if err := RunWithClientOptions([]string{"5m", "Kaffee"}, Options{Detach: true}); err != nil {
		t.Fatal(err)
	}
	if err := RunWithClientOptions([]string{"start", "Kaffee"}, Options{Detach: true}); err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 2 {
		t.Fatalf("expected 2 creations, got %d", creates.Load())
	}
	if err := RunWithClientOptions(nil, Options{Detach: true}); err == nil || !strings.Contains(err.Error(), "--detach") {
		t.Fatalf("unexpected empty command error: %v", err)
	}
}
