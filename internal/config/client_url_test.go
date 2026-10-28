package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetClientURLPreservesServerSettings(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path := filepath.Join(root, "nexttimer", "client.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"listen":"0.0.0.0:8080","custom_future":"keep"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := SetClientURL("http://pi.local:8080"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]any
	if err := json.Unmarshal(b, &o); err != nil {
		t.Fatal(err)
	}
	if o["url"] != "http://pi.local:8080" || o["listen"] != "0.0.0.0:8080" || o["custom_future"] != "keep" {
		t.Fatalf("settings lost: %s", b)
	}
	if actual, err := ClientURL("http://localhost:8080"); err != nil || actual != "http://pi.local:8080" {
		t.Fatalf("url read: %q %v", actual, err)
	}
	if _, err := SetClientURL("https://example.org/prefix"); err == nil {
		t.Fatal("path should be rejected")
	}
	if _, err := SetClientURL("http://user:pass@example.org"); err == nil {
		t.Fatal("credentials should be rejected")
	}
}
