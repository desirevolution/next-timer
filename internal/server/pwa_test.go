package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPWARequiresUserUpdateApproval(t *testing.T) {
	handler := API{Service: makeService(t)}.Handler()
	for _, path := range []string{"/", "/sw.js", "/pwa-update.js"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if !strings.Contains(w.Header().Get("Cache-Control"), "no-cache") {
			t.Errorf("%s missing no-cache header", path)
		}
		body := w.Body.String()
		if path == "/sw.js" && (!strings.Contains(body, "SKIP_WAITING") || strings.Contains(body, ".then(()=>self.skipWaiting())")) {
			t.Fatal("sw must await user approval")
		}
		if path == "/" && (!strings.Contains(body, "pwa-update.js?v=21") || !strings.Contains(body, "pwa-update-banner")) {
			t.Fatal("missing PWA update coordinator")
		}
	}
}
