package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// AlarmWebhook delivers exactly one event type: timer.finished.
// It does not block the timer engine and has intentionally no generic hooks.
type AlarmWebhook struct {
	url  string
	http *http.Client
}

func NewAlarmWebhook(url string) *AlarmWebhook {
	return &AlarmWebhook{url: url, http: &http.Client{Timeout: 4 * time.Second}}
}

func (h *AlarmWebhook) send(ctx context.Context, timer TimerView) error {
	when := time.UnixMilli(timer.FinishedAtMS).UTC().Format(time.RFC3339Nano)
	payload := struct {
		Event     string    `json:"event"`
		EventID   string    `json:"event_id"`
		Timestamp string    `json:"timestamp"`
		Timer     TimerView `json:"timer"`
	}{
		Event:     "timer.finished",
		EventID:   timer.ID + "-" + strconv.FormatInt(timer.FinishedAtMS, 10),
		Timestamp: when,
		Timer:     timer,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	response, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, response)
	}
	return nil
}

func (h *AlarmWebhook) Run(ctx context.Context, s *Service) {
	events, unsubscribe := s.Subscribe()
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Type != "finished" || ev.Timer == nil {
				continue
			}
			if err := h.send(ctx, *ev.Timer); err != nil {
				log.Printf("Timer-Alarm-WebHook (ID=%s): %v", ev.Timer.ID, err)
			}
		}
	}
}
