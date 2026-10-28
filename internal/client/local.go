package client

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

// localStore is deliberately process-local. Closing the client ends its timers;
// no network request, AWTRIX update or server-side alarm is generated.
type localStore struct {
	mu     sync.Mutex
	next   int
	timers []localEntry
}

type localEntry struct {
	timer    protocol.TimerView
	deadline time.Time
}

func isLocalID(id string) bool { return strings.HasPrefix(id, "L") && len(id) > 1 }

func (s *localStore) create(duration time.Duration, name string) protocol.TimerView {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	if strings.TrimSpace(name) == "" {
		name = "Timer"
	}
	now := time.Now()
	t := protocol.TimerView{
		ID: "L" + strconv.Itoa(s.next), Name: name, Status: "running",
		DurationMS: duration.Milliseconds(), RemainingMS: duration.Milliseconds(),
		CreatedAtMS: now.UnixMilli(), DeadlineMS: time.Unix(0, now.Add(duration).UnixNano()).UnixMilli(), Local: true,
	}
	s.timers = append(s.timers, localEntry{timer: t, deadline: time.Unix(0, now.Add(duration).UnixNano())})
	return t
}

func (s *localStore) snapshot() []protocol.TimerView {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	out := make([]protocol.TimerView, 0, len(s.timers))
	for i := range s.timers {
		entry := &s.timers[i]
		if entry.timer.Status == "running" {
			left := entry.deadline.Sub(now)
			if left <= 0 {
				entry.timer.Status = "finished"
				entry.timer.RemainingMS = 0
				entry.timer.FinishedAtMS = now.UnixMilli()
			} else {
				entry.timer.RemainingMS = left.Milliseconds()
				entry.timer.DeadlineMS = entry.deadline.UnixMilli()
			}
		}
		out = append(out, entry.timer)
	}
	return out
}

func (s *localStore) control(id, action, delta string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.timers {
		e := &s.timers[i]
		if e.timer.ID != id {
			continue
		}
		if e.timer.Status == "running" && !e.deadline.After(now) {
			e.timer.Status = "finished"
			e.timer.RemainingMS = 0
			e.timer.FinishedAtMS = now.UnixMilli()
		}
		switch action {
		case "pause":
			if e.timer.Status != "running" {
				return fmt.Errorf("Timer läuft nicht")
			}
			e.timer.RemainingMS = e.deadline.Sub(now).Milliseconds()
			e.timer.Status = "paused"
		case "resume":
			if e.timer.Status != "paused" {
				return fmt.Errorf("Timer ist nicht pausiert")
			}
			e.deadline = time.Unix(0, now.Add(time.Duration(e.timer.RemainingMS)*time.Millisecond).UnixNano())
			e.timer.DeadlineMS = e.deadline.UnixMilli()
			e.timer.Status = "running"
		case "adjust":
			if e.timer.Status != "running" && e.timer.Status != "paused" {
				return fmt.Errorf("Timer ist bereits beendet")
			}
			d, err := time.ParseDuration(delta)
			if err != nil {
				return err
			}
			rem := time.Duration(e.timer.RemainingMS) * time.Millisecond
			if e.timer.Status == "running" {
				rem = e.deadline.Sub(now)
			}
			newRemaining := rem + d
			if newRemaining <= 0 {
				e.timer.Status = "finished"
				e.timer.FinishedAtMS = now.UnixMilli()
				e.timer.RemainingMS = 0
				return nil
			}
			e.timer.DurationMS += d.Milliseconds()
			if e.timer.DurationMS < newRemaining.Milliseconds() {
				e.timer.DurationMS = newRemaining.Milliseconds()
			}
			e.timer.RemainingMS = newRemaining.Milliseconds()
			if e.timer.Status == "running" {
				e.deadline = time.Unix(0, now.Add(newRemaining).UnixNano())
				e.timer.DeadlineMS = e.deadline.UnixMilli()
			}
		case "stop":
			s.timers = append(s.timers[:i], s.timers[i+1:]...)
		default:
			return fmt.Errorf("unbekannte Aktion: %s", action)
		}
		return nil
	}
	return fmt.Errorf("lokaler Timer %s nicht gefunden", id)
}

// mergeLocal makes the shortest remaining running timer the client-side focus.
// It never changes the server's own FocusID.
func mergeLocal(remote protocol.Snapshot, local []protocol.TimerView) protocol.Snapshot {
	remote.Timers = append(remote.Timers, local...)
	var best *protocol.TimerView
	for i := range remote.Timers {
		t := &remote.Timers[i]
		if t.Status != "running" {
			continue
		}
		if best == nil || t.DeadlineMS < best.DeadlineMS || (t.DeadlineMS == best.DeadlineMS && t.ID < best.ID) {
			best = t
		}
	}
	remote.FocusID = ""
	if best != nil {
		remote.FocusID = best.ID
	}
	if remote.ServerNowMS == 0 {
		remote.ServerNowMS = time.Now().UnixMilli()
	}
	return remote
}
