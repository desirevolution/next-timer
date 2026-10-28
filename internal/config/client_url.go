package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SetClientURL changes only the URL in the XDG config. Other server settings
// and unknown future keys are preserved; the file is atomically replaced.
func ClientConfigPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "nexttimer", "client.json"), nil
}

// ClientURL reads the client's own URL, falling back to the older shared
// config file. TIMER_URL remains the highest-priority non-flag setting.
func ClientURL(fallback string) (string, error) {
	if v, ok := os.LookupEnv("TIMER_URL"); ok {
		return v, nil
	}
	path, err := ClientConfigPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	var c struct {
		URL string `json:"url"`
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return "", fmt.Errorf("Client-Konfiguration %s: %w", path, err)
	}
	if strings.TrimSpace(c.URL) == "" {
		return fallback, nil
	}
	return strings.TrimRight(strings.TrimSpace(c.URL), "/"), nil
}

func SetClientURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("Server-URL muss http(s)://HOST[:PORT] ohne Pfad, Benutzerinfos oder Parameter sein")
	}
	path, err := ClientConfigPath()
	if err != nil {
		return "", err
	}
	obj := make(map[string]json.RawMessage)
	b, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(b, &obj); err != nil {
			return "", fmt.Errorf("bestehende Konfiguration ist ungültig: %w", err)
		}
		if obj == nil {
			return "", errors.New("Konfiguration muss ein JSON-Objekt sein")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	value, _ := json.Marshal(strings.TrimRight(raw, "/"))
	obj["url"] = value
	content, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err = tmp.Write(append(content, '\n')); err != nil {
		tmp.Close()
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}
