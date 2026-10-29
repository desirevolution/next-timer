package server

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTemplateShortcutAssignmentAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.AddTemplateConfigured(Template{Name: "Kaffee", Duration: "5m", Shortcut: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Shortcut != "1" {
		t.Fatalf("shortcut = %q", first.Shortcut)
	}
	if _, err := s.AddTemplateConfigured(Template{Name: "Tee", Duration: "3m", Shortcut: "1"}); err == nil {
		t.Fatal("duplicate shortcut across favorites accepted")
	}
	if _, err := s.AddTemplateConfigured(Template{Name: "Ungültig", Duration: "3m", Shortcut: "x"}); err == nil {
		t.Fatal("invalid shortcut accepted")
	}
	second, err := s.AddTemplateConfigured(Template{Name: "Tee", Duration: "3m", Shortcut: "0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateTemplateConfigured(first.ID, Template{Name: "Kaffee", Duration: "5m", Shortcut: "0"}); err == nil {
		t.Fatal("duplicate shortcut accepted on update")
	}
	updated, err := s.UpdateTemplateConfigured(first.ID, Template{Name: "Kaffee", Duration: "5m", Shortcut: "2"})
	if err != nil || updated.Shortcut != "2" {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	fresh, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got := fresh.Snapshot().Templates
	if len(got) != 2 || got[0].Shortcut != "2" || got[1].Shortcut != "0" {
		t.Fatalf("shortcuts did not survive a restart: %+v", got)
	}
	if err := s.DeleteTemplate(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddTemplateConfigured(Template{Name: "Neu", Duration: "1m", Shortcut: "0"}); err != nil {
		t.Fatalf("deleted favorite must release shortcut: %v", err)
	}
}
