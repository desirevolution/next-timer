package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXDGPaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	configFile, e := ConfigPath()
	if e != nil {
		t.Fatal(e)
	}
	if want := filepath.Join(root, "cfg", "nexttimer", "config.json"); configFile != want {
		t.Fatalf("got %s want %s", configFile, want)
	}
	data, e := DataPath()
	if e != nil {
		t.Fatal(e)
	}
	if want := filepath.Join(root, "data", "nexttimer", "templates.json"); data != want {
		t.Fatalf("got %s want %s", data, want)
	}
	t.Setenv("TIMER_URL", "http://localhost:9999")
	if _, e := Init(); e != nil {
		t.Fatal(e)
	}
	c, _, e := Read()
	if e != nil {
		t.Fatal(e)
	}
	if c.URL != "http://localhost:9999" {
		t.Fatal("environment precedence broken")
	}
	if _, e := Init(); !os.IsExist(e) {
		t.Fatalf("expected no overwrite: %v", e)
	}
}
func TestInvalidConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	p, e := ConfigPath()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(p, []byte(`{"listen":"0.0.0.0:8080","unknown":42}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, e = Read(); e == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestAWTRIXFinishEffectConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defaults, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if defaults.AWTRIXFinishEffect != "Fireworks" {
		t.Fatalf("wrong default: %q", defaults.AWTRIXFinishEffect)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"awtrix_finish_effect":"TwinklingStars"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, _, err := Read()
	if err != nil || c.AWTRIXFinishEffect != "TwinklingStars" {
		t.Fatalf("config file effect: %#v %v", c, err)
	}
	t.Setenv("AWTRIX_FINISH_EFFECT", "none")
	c, _, err = Read()
	if err != nil || c.AWTRIXFinishEffect != "none" {
		t.Fatalf("environment override: %#v %v", c, err)
	}
}

func TestAWTRIXFinishOptions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defaults, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if defaults.AWTRIXFinishEffectSpeed != 1 || defaults.AWTRIXFinishDuration != "5s" {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"awtrix_finish_effect":"ColorWaves","awtrix_finish_effect_speed":0.3,"awtrix_finish_duration":"10s"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, _, err := Read()
	if err != nil || c.AWTRIXFinishEffect != "ColorWaves" || c.AWTRIXFinishEffectSpeed != 0.3 || c.AWTRIXFinishDuration != "10s" {
		t.Fatalf("file options: %+v %v", c, err)
	}
	t.Setenv("AWTRIX_FINISH_EFFECT_SPEED", "2")
	t.Setenv("AWTRIX_FINISH_DURATION", "15s")
	c, _, err = Read()
	if err != nil || c.AWTRIXFinishEffectSpeed != 2 || c.AWTRIXFinishDuration != "15s" {
		t.Fatalf("env options: %+v %v", c, err)
	}
	for _, raw := range []string{"0", "10.1", "NaN", "not-a-number"} {
		t.Setenv("AWTRIX_FINISH_EFFECT_SPEED", raw)
		if _, _, err := Read(); err == nil {
			t.Errorf("invalid effect speed %q accepted", raw)
		}
	}
	t.Setenv("AWTRIX_FINISH_EFFECT_SPEED", "1")
	for _, raw := range []string{"0s", "banana", "10m", "-1s"} {
		t.Setenv("AWTRIX_FINISH_DURATION", raw)
		if _, _, err := Read(); err == nil {
			t.Errorf("invalid finish duration %q accepted", raw)
		}
	}
}
