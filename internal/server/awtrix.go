package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Awtrix uses only AWTRIX NG notifications; it never installs or updates an app.
// A short-lived, named notification displays the primary timer and is refreshed
// once per second. It expires automatically if the server crashes. A separate
// five-second notification temporarily replaces it when a countdown finishes.
type Awtrix struct {
	base              string
	http              *http.Client
	active            bool
	lastID            string
	lastText          string
	lastDeadlineMS    int64
	lastDurationMS    int64
	lastSent          time.Time
	displayLead       time.Duration
	lastUrgent        bool
	overlayUntil      time.Time
	finishEffect      string
	finishEffectSpeed float64
	finishDuration    time.Duration
}

const awtrixLiveName = "nexttimer-live"

func NewAwtrix(base string) *Awtrix {
	return &Awtrix{
		base:              strings.TrimRight(base, "/"),
		http:              &http.Client{Timeout: 3 * time.Second},
		finishEffect:      "Fireworks",
		finishEffectSpeed: 1.0,
		finishDuration:    5 * time.Second,
		displayLead:       100 * time.Millisecond,
	}
}

func (a *Awtrix) SetFinishEffect(name string) {
	name = strings.TrimSpace(name)
	if strings.EqualFold(name, "none") || strings.EqualFold(name, "off") {
		name = ""
	}
	a.finishEffect = name
}

// SetDisplayLead compensates for the LAN/firmware delivery delay. This shifts
// only the AWTRIX text update, never the actual timer deadline or alarm.
func (a *Awtrix) SetDisplayLead(lead time.Duration) { a.displayLead = lead }

// SetFinishOptions controls only the alarm notification, not the live countdown.
// The caller validates ranges before calling this method.
func (a *Awtrix) SetFinishOptions(speed float64, duration time.Duration) {
	a.finishEffectSpeed = speed
	a.finishDuration = duration
}

func (a *Awtrix) request(ctx context.Context, method, endpoint string, payload any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+endpoint, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	// Missing named notification is a valid state after an AWTRIX restart or
	// if the user dismissed it manually.
	if method == http.MethodDelete && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("AWTRIX NG %s %s: HTTP %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (a *Awtrix) dismiss(ctx context.Context) error {
	err := a.request(ctx, http.MethodDelete, "/api/v1/notifications/"+url.PathEscape(awtrixLiveName), nil)
	if err == nil {
		a.active = false
		a.lastID = ""
		a.lastText = ""
		a.lastDeadlineMS = 0
		a.lastDurationMS = 0
		a.lastSent = time.Time{}
		a.lastUrgent = false
	}
	return err
}

func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int64(math.Ceil(d.Seconds()))
	if seconds >= 3600 {
		return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds/60)%60, seconds%60)
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func (a *Awtrix) Render(ctx context.Context, state Snapshot) {
	a.renderAt(ctx, state, time.Now())
}

func (a *Awtrix) renderAt(ctx context.Context, state Snapshot, now time.Time) {
	if now.Before(a.overlayUntil) {
		return
	}
	var main *TimerView
	for i := range state.Timers {
		if state.Timers[i].ID == state.FocusID && state.Timers[i].Status == "running" {
			main = &state.Timers[i]
			break
		}
	}
	if main == nil {
		if a.active {
			if err := a.dismiss(ctx); err != nil {
				log.Printf("AWTRIX NG Countdown entfernen: %v", err)
			}
		}
		return
	}
	remaining := time.UnixMilli(main.DeadlineMS).Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	progress := 0
	if main.DurationMS > 0 {
		progress = int(math.Round(100 * (1 - float64(remaining.Milliseconds())/float64(main.DurationMS))))
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	// Render a fraction of a second ahead to account for HTTP delivery and the
	// device's display refresh. The actual finish event remains server-owned.
	shownRemaining := remaining - a.displayLead
	if shownRemaining < 0 {
		shownRemaining = 0
	}
	text := formatClock(shownRemaining)
	urgent := shownRemaining <= 10*time.Second
	// Run polls frequently, but send only when the displayed second changes,
	// a timer is adjusted/switched, or the notification needs a refresh.
	// This phase-locks updates to the timer deadline rather than to an
	// unrelated 1-second ticker (which used to drift by almost one second).
	if a.active && a.lastID == main.ID && a.lastText == text &&
		a.lastUrgent == urgent && a.lastDeadlineMS == main.DeadlineMS &&
		a.lastDurationMS == main.DurationMS && now.Sub(a.lastSent) < 1900*time.Millisecond {
		return
	}
	color, bar, bg := "#FFFFFF", "#00FF00", "#000000"
	if urgent {
		color, bar, bg = "#FF0000", "#FF0000", "#280000"
	}
	payload := map[string]any{
		"name":               awtrixLiveName,
		"text":               text,
		"textColor":          color,
		"backgroundColor":    bg,
		"progress":           progress,
		"progressColor":      bar,
		"progressTrackColor": "#25344B",
		"scroll":             map[string]any{"mode": "static"},
		"durationMs":         2500,
		"stack":              false,
	}
	if err := a.request(ctx, http.MethodPost, "/api/v1/notifications", payload); err != nil {
		log.Printf("AWTRIX NG Countdown aktualisieren: %v", err)
		return
	}
	if a.lastID != main.ID || a.lastUrgent != urgent {
		log.Printf("AWTRIX NG Haupttimer %s: %s, Schlussphase=%t", main.ID, text, urgent)
	}
	a.active = true
	a.lastID = main.ID
	a.lastText = text
	a.lastDeadlineMS = main.DeadlineMS
	a.lastDurationMS = main.DurationMS
	a.lastSent = now
	a.lastUrgent = urgent
}

func (a *Awtrix) notifyFinished(ctx context.Context, t TimerView, queued bool) error {
	payload := map[string]any{
		"text":       t.Name + " fertig",
		"textColor":  "#FF0000",
		"durationMs": a.finishDuration.Milliseconds(),
		"stack":      queued,
	}
	if a.finishEffect != "" {
		payload["effect"] = a.finishEffect
		payload["effectSpeed"] = a.finishEffectSpeed
	}
	return a.request(ctx, http.MethodPost, "/api/v1/notifications", payload)
}

func (a *Awtrix) Run(ctx context.Context, s *Service) {
	log.Print("AWTRIX NG Notification-Adapter gestartet")
	clean, cancelClean := context.WithTimeout(ctx, 3*time.Second)
	if err := a.dismiss(clean); err != nil {
		log.Printf("AWTRIX NG Startbereinigung: %v", err)
	}
	cancelClean()
	events, unsubscribe := s.Subscribe()
	defer unsubscribe()
	// The 40 ms wakeup is local only; HTTP requests are de-duplicated by
	// renderAt and sent approximately once per second at the actual time edge.
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if a.active {
				clean, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if err := a.dismiss(clean); err != nil {
					log.Printf("AWTRIX NG Shutdown-Bereinigung: %v", err)
				}
				cancel()
			}
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Type != "finished" || ev.Timer == nil {
				// React immediately to starts, pauses, resumes and adjustments.
				// Waiting for a periodic tick introduced avoidable delay.
				a.Render(ctx, ev.State)
				continue
			}
			now := time.Now()
			queued := now.Before(a.overlayUntil)
			if !queued {
				// The first finish notification replaces the live countdown.
				a.overlayUntil = now
			}
			if err := a.notifyFinished(ctx, *ev.Timer, queued); err != nil {
				log.Printf("AWTRIX NG Fertig-Notification: %v", err)
				continue
			}
			a.overlayUntil = a.overlayUntil.Add(a.finishDuration)
			a.active = false // The live countdown was replaced, not retained.
		case <-ticker.C:
			a.Render(ctx, s.Snapshot())
		}
	}
}
