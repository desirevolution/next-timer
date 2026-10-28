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

func TestFavoriteGroupsLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.AddTemplateConfigured(Template{Name: "A", Duration: "3m", Group: "Arbeit"})
	b, _ := s.AddTemplateConfigured(Template{Name: "B", Duration: "5m", Group: "Küche"})
	c, _ := s.AddTemplateConfigured(Template{Name: "C", Duration: "6m", Group: "Arbeit"})
	d, _ := s.AddTemplateConfigured(Template{Name: "D", Duration: "7m"})
	check := func(wantGroups ...string) {
		t.Helper()
		snap := s.Snapshot()
		if len(snap.Groups) != len(wantGroups) {
			t.Fatalf("groups=%v want %v", snap.Groups, wantGroups)
		}
		for i, g := range wantGroups {
			if snap.Groups[i] != g {
				t.Fatalf("groups=%v want %v", snap.Groups, wantGroups)
			}
		}
	}
	check("Arbeit", "Küche")
	if got := s.Snapshot().Templates; got[0].ID != a.ID || got[1].ID != c.ID || got[2].ID != b.ID || got[3].ID != d.ID {
		t.Fatalf("bad group order: %+v", got)
	}
	if err := s.ReorderTemplates([]string{c.ID, a.ID, b.ID, d.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReorderTemplates([]string{a.ID, b.ID, c.ID, d.ID}); err == nil {
		t.Fatal("cross-group reorder accepted")
	}
	if err := s.RenameGroup("Arbeit", "Job"); err != nil {
		t.Fatal(err)
	}
	check("Job", "Küche")
	if err := s.RenameGroup("Küche", "Job"); err == nil {
		t.Fatal("collision rename accepted")
	}
	s2, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Snapshot().Templates[0].Group != "Job" {
		t.Fatal("rename not persisted")
	}
	_, err = s.UpdateTemplateConfigured(c.ID, Template{Name: "C", Duration: "6m", Group: "Küche"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTemplate(a.ID); err != nil {
		t.Fatal(err)
	}
	check("Küche")
	if err := s.DeleteTemplate(b.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTemplate(c.ID); err != nil {
		t.Fatal(err)
	}
	check()
	if len(s.Snapshot().Templates) != 1 || s.Snapshot().Templates[0].ID != d.ID {
		t.Fatal("default group not retained")
	}
}

func TestFavoriteGroupHTTPAndNoIcon(t *testing.T) {
	s := makeService(t)
	ts := httptest.NewServer((API{s}).Handler())
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/templates", "application/json", strings.NewReader(`{"name":"Tee","duration":"4m","group":"Küche"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("add %d", resp.StatusCode)
	}
	var fav Template
	if err := json.NewDecoder(resp.Body).Decode(&fav); err != nil {
		t.Fatal(err)
	}
	if fav.Group != "Küche" {
		t.Fatalf("group=%q", fav.Group)
	}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/groups/K%C3%BCche", strings.NewReader(`{"name":"Kochen"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	renamed, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer renamed.Body.Close()
	if renamed.StatusCode != 200 {
		t.Fatalf("rename %d", renamed.StatusCode)
	}
	if s.Snapshot().Templates[0].Group != "Kochen" {
		t.Fatalf("rename not applied: %+v", s.Snapshot().Templates)
	}
	// Icon input is no longer supported, including on new templates.
	response, err := http.Post(ts.URL+"/api/templates", "application/json", strings.NewReader(`{"name":"X","duration":"3m","icon":"☕"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatalf("icon accepted: %d", response.StatusCode)
	}
}

func TestOldIconsNotPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	if err := os.WriteFile(path, []byte(`[{"id":"old","name":"Alt","duration":"2m","icon":"☕"}]`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AddTemplate("Neu", "3m")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "icon") {
		t.Fatal("legacy symbol was persisted")
	}
}
