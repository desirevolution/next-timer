package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool { return &v }

func TestFavoriteOptionsPersistAndRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	original, err := s.AddTemplateConfigured(Template{Name: "Kaffee", Duration: "5m", Sound: boolPtr(false), Notify: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.StartTemplate(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Sound || !v.Notify {
		t.Fatalf("timer options lost: %+v", v)
	}
	_, err = s.UpdateTemplateConfigured(original.ID, Template{Name: "Kaffee", Duration: "5m", BrowserOnly: true, Sound: boolPtr(true), Notify: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartTemplate(original.ID); err == nil {
		t.Fatal("browser-only favorite launched on server")
	}
	restored, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got := restored.Snapshot().Templates[0]
	if !got.BrowserOnly || !enabled(got.Sound) || enabled(got.Notify) {
		t.Fatalf("wrong restored options: %+v", got)
	}
}
func TestOldTemplatesDefaultToAlertsEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	if err := os.WriteFile(path, []byte(`[{"id":"old","name":"Alt","duration":"2m"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.StartTemplate("old")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Sound || !v.Notify {
		t.Fatalf("legacy options unexpectedly disabled: %+v", v)
	}
}
func TestTimerOptionsHTTP(t *testing.T) {
	s := makeService(t)
	ts := httptest.NewServer((API{s}).Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/api/timers", "application/json", strings.NewReader(`{"name":"Eier","duration":"1m","sound":false,"notify":false}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("status %d", res.StatusCode)
	}
	var v TimerView
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	if v.Sound || v.Notify {
		t.Fatalf("wrong options %+v", v)
	}
	res2, err := http.Post(ts.URL+"/api/timers", "application/json", strings.NewReader(`{"name":"X","duration":"1m","browser_only":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != 400 {
		t.Fatalf("server accepted local-only: %d", res2.StatusCode)
	}
}
func TestTemplateHTTPOptions(t *testing.T) {
	s := makeService(t)
	ts := httptest.NewServer((API{s}).Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/api/templates", "application/json", strings.NewReader(`{"name":"Kaffee","duration":"5m","browser_only":true,"notify":false,"sound":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("add status %d", res.StatusCode)
	}
	var tmpl Template
	if err := json.NewDecoder(res.Body).Decode(&tmpl); err != nil {
		t.Fatal(err)
	}
	if !tmpl.BrowserOnly || enabled(tmpl.Notify) || !enabled(tmpl.Sound) {
		t.Fatalf("wrong template: %+v", tmpl)
	}
	req, err := http.NewRequest("PUT", ts.URL+"/api/templates/"+tmpl.ID, strings.NewReader(`{"name":"Tee","duration":"6m","browser_only":false,"notify":true,"sound":false}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 200 {
		t.Fatalf("update status %d", r2.StatusCode)
	}
	got := s.Snapshot().Templates[0]
	if got.BrowserOnly || !enabled(got.Notify) || enabled(got.Sound) {
		t.Fatalf("bad updated options: %+v", got)
	}
}
