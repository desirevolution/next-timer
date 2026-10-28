package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is shared by CLI and server; environment variables override file values.
type Config struct {
	URL                     string  `json:"url"`
	Listen                  string  `json:"listen"`
	Data                    string  `json:"data"`
	AWTRIX                  string  `json:"awtrix_url"`
	AWTRIXFinishEffect      string  `json:"awtrix_finish_effect"`
	AWTRIXFinishEffectSpeed float64 `json:"awtrix_finish_effect_speed"`
	AWTRIXFinishDuration    string  `json:"awtrix_finish_duration"`
	AWTRIXSyncLead          string  `json:"awtrix_sync_lead"`
	AlarmWebhook            string  `json:"alarm_webhook_url"`
	FinishedTTL             string  `json:"finished_ttl"`
}

func ConfigPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "nexttimer", "config.json"), nil
}
func DataPath() (string, error) {
	root := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(root, "nexttimer", "templates.json"), nil
}
func Defaults() (Config, error) {
	data, err := DataPath()
	if err != nil {
		return Config{}, err
	}
	return Config{URL: "http://127.0.0.1:8080", Listen: "127.0.0.1:8080", Data: data, FinishedTTL: "60s", AWTRIXFinishEffect: "Fireworks", AWTRIXFinishEffectSpeed: 1.0, AWTRIXFinishDuration: "5s", AWTRIXSyncLead: "100ms"}, nil
}
func Read() (Config, string, error) {
	defaults, err := Defaults()
	if err != nil {
		return Config{}, "", err
	}
	path, err := ConfigPath()
	if err != nil {
		return Config{}, "", err
	}
	c := defaults
	raw, err := os.ReadFile(path)
	if err == nil {
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		var file Config
		if err = dec.Decode(&file); err != nil {
			return Config{}, path, fmt.Errorf("config %s: %w", path, err)
		}
		// Read only defined values; zero/empty means default.
		if file.URL != "" {
			c.URL = file.URL
		}
		if file.Listen != "" {
			c.Listen = file.Listen
		}
		if file.Data != "" {
			c.Data = file.Data
		}
		if file.AWTRIX != "" {
			c.AWTRIX = file.AWTRIX
		}
		if file.AWTRIXFinishEffect != "" {
			c.AWTRIXFinishEffect = file.AWTRIXFinishEffect
		}
		if file.AWTRIXFinishEffectSpeed != 0 {
			c.AWTRIXFinishEffectSpeed = file.AWTRIXFinishEffectSpeed
		}
		if file.AWTRIXFinishDuration != "" {
			c.AWTRIXFinishDuration = file.AWTRIXFinishDuration
		}
		if file.AWTRIXSyncLead != "" {
			c.AWTRIXSyncLead = file.AWTRIXSyncLead
		}
		if file.AlarmWebhook != "" {
			c.AlarmWebhook = file.AlarmWebhook
		}
		if file.FinishedTTL != "" {
			c.FinishedTTL = file.FinishedTTL
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, path, err
	}
	if v, ok := os.LookupEnv("TIMER_URL"); ok {
		c.URL = v
	}
	if v, ok := os.LookupEnv("TIMER_LISTEN"); ok {
		c.Listen = v
	}
	if v, ok := os.LookupEnv("TIMER_DATA"); ok {
		c.Data = v
	}
	if v, ok := os.LookupEnv("AWTRIX_URL"); ok {
		c.AWTRIX = v
	}
	if v, ok := os.LookupEnv("AWTRIX_FINISH_EFFECT"); ok {
		c.AWTRIXFinishEffect = v
	}
	if v, ok := os.LookupEnv("AWTRIX_FINISH_EFFECT_SPEED"); ok {
		speed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, path, fmt.Errorf("AWTRIX_FINISH_EFFECT_SPEED: %w", err)
		}
		c.AWTRIXFinishEffectSpeed = speed
	}
	if v, ok := os.LookupEnv("AWTRIX_FINISH_DURATION"); ok {
		c.AWTRIXFinishDuration = v
	}
	if v, ok := os.LookupEnv("AWTRIX_SYNC_LEAD"); ok {
		c.AWTRIXSyncLead = v
	}
	if v, ok := os.LookupEnv("TIMER_ALARM_WEBHOOK_URL"); ok {
		c.AlarmWebhook = v
	}
	if v, ok := os.LookupEnv("TIMER_FINISHED_TTL"); ok {
		c.FinishedTTL = v
	}
	return c, path, c.Validate()
}
func (c Config) Validate() error {
	if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
		return errors.New("url muss mit http:// oder https:// beginnen")
	}
	if strings.TrimSpace(c.Listen) == "" || strings.TrimSpace(c.Data) == "" {
		return errors.New("listen und data dürfen nicht leer sein")
	}
	if c.AWTRIX != "" && !strings.HasPrefix(c.AWTRIX, "http://") && !strings.HasPrefix(c.AWTRIX, "https://") {
		return errors.New("awtrix_url muss eine HTTP(S)-URL sein")
	}
	if c.AlarmWebhook != "" {
		u, err := url.Parse(c.AlarmWebhook)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return errors.New("alarm_webhook_url muss eine gültige HTTP(S)-URL ohne Benutzerinformationen sein")
		}
	}
	if len(c.AWTRIXFinishEffect) > 64 || strings.ContainsAny(c.AWTRIXFinishEffect, "\r\n") {
		return errors.New("awtrix_finish_effect ist ungültig")
	}
	if math.IsNaN(c.AWTRIXFinishEffectSpeed) || math.IsInf(c.AWTRIXFinishEffectSpeed, 0) || c.AWTRIXFinishEffectSpeed < 0.1 || c.AWTRIXFinishEffectSpeed > 10 {
		return errors.New("awtrix_finish_effect_speed muss zwischen 0.1 und 10 liegen")
	}
	finishDuration, err := time.ParseDuration(c.AWTRIXFinishDuration)
	if err != nil || finishDuration < time.Second || finishDuration > 5*time.Minute {
		return errors.New("awtrix_finish_duration muss zwischen 1s und 5m liegen")
	}
	lead, err := time.ParseDuration(c.AWTRIXSyncLead)
	if err != nil || lead < 0 || lead > 500*time.Millisecond {
		return errors.New("awtrix_sync_lead muss zwischen 0ms und 500ms liegen")
	}
	ttl, err := time.ParseDuration(c.FinishedTTL)
	if err != nil || ttl < 5*time.Second {
		return errors.New("finished_ttl muss mindestens 5s sein")
	}
	return nil
}
func Init() (string, error) {
	c, err := Defaults()
	if err != nil {
		return "", err
	}
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(append(b, '\n'))
	e := f.Close()
	if err == nil {
		err = e
	}
	return path, err
}
