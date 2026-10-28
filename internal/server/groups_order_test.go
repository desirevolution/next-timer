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

func TestGroupReorderPersistsAndKeepsFavoriteOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddTemplateConfigured(Template{Name: "A", Duration: "1m", Group: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddTemplateConfigured(Template{Name: "B", Duration: "2m", Group: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.AddTemplateConfigured(Template{Name: "C", Duration: "3m", Group: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.AddTemplateConfigured(Template{Name: "D", Duration: "4m"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReorderTemplates([]string{c.ID, a.ID, b.ID, d.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReorderGroups([]string{"Home", "Work"}); err != nil {
		t.Fatal(err)
	}
	got := s.Snapshot()
	if len(got.Groups) != 2 || got.Groups[0] != "Home" || got.Groups[1] != "Work" {
		t.Fatalf("groups: %v", got.Groups)
	}
	want := []string{b.ID, c.ID, a.ID, d.ID}
	for i, fav := range got.Templates {
		if fav.ID != want[i] {
			t.Fatalf("favorites reordered unexpectedly: %v", got.Templates)
		}
	}
	if err := s.ReorderGroups([]string{"Home", "Home"}); err == nil {
		t.Fatal("duplicate group accepted")
	}
	if err := s.ReorderGroups([]string{"Home"}); err == nil {
		t.Fatal("incomplete group list accepted")
	}
	if err := s.ReorderGroups([]string{"Other", "Work"}); err == nil {
		t.Fatal("unknown group accepted")
	}
	after, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	persisted := after.Snapshot()
	if persisted.Groups[0] != "Home" || persisted.Groups[1] != "Work" {
		t.Fatalf("reorder not persisted %v", persisted.Groups)
	}
	ts := httptest.NewServer((API{after}).Handler())
	defer ts.Close()
	req, _ := http.NewRequest("PUT", ts.URL+"/api/groups/order", strings.NewReader(`{"groups":["Work","Home"]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var state Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.Groups[0] != "Work" {
		t.Fatalf("bad API reorder: %v", state.Groups)
	}
}

func TestFavoriteNameUniquePerGroupAndAmbiguousServerStart(t *testing.T) {
	s, err := NewService(filepath.Join(t.TempDir(), "fav.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddTemplateConfigured(Template{Name: "Pause", Duration: "5m", Group: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddTemplateConfigured(Template{Name: "pause", Duration: "10m", Group: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AddTemplateConfigured(Template{Name: "PAUSE", Duration: "3m", Group: "work"})
	if err == nil {
		t.Fatal("duplicate in same group accepted")
	}
	if _, err = s.StartTemplate("pause"); err == nil || !strings.Contains(err.Error(), "mehrdeutig") {
		t.Fatalf("name unexpectedly resolved: %v", err)
	}
	if _, err = s.StartTemplate(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.StartTemplate(b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateTemplateConfigured(b.ID, Template{Name: "Pause", Duration: "3m", Group: "Work"}); err == nil {
		t.Fatal("move collision accepted")
	}
	if _, err = s.UpdateTemplateConfigured(b.ID, Template{Name: "Pause", Duration: "3m", Group: "Home"}); err != nil {
		t.Fatal(err)
	}
}
