package client

import (
	"testing"
	"time"
)

func TestProcessLocalLastFinished(t *testing.T) {
	s := &localStore{}
	if s.completed() != nil {
		t.Fatal("unexpected repeat candidate")
	}
	one := s.create(5*time.Minute, "One")
	if err := s.control(one.ID, "pause", ""); err != nil {
		t.Fatal(err)
	}
	if s.completed() != nil {
		t.Fatal("pause is not a completion")
	}
	if err := s.control(one.ID, "stop", ""); err != nil {
		t.Fatal(err)
	}
	if item := s.completed(); item == nil || item.Name != "One" || item.Duration != "5m0s" {
		t.Fatalf("manual stop not captured: %+v", item)
	}
	two := s.create(time.Second, "Two")
	s.mu.Lock()
	s.timers[0].deadline = time.Now().Add(2 * time.Millisecond)
	time.Sleep(4 * time.Millisecond)
	s.mu.Unlock()
	s.snapshot()
	if last := s.completed(); last.Name != "Two" {
		t.Fatalf("expiration not captured: %+v", last)
	}
	// Later observations of an older deadline must not override the latest event.
	three := s.create(time.Second, "Three")
	s.mu.Lock()
	s.timers[1].deadline = time.Now().Add(-time.Hour)
	s.mu.Unlock()
	s.snapshot()
	if last := s.completed(); last.Name != "Two" {
		t.Fatalf("old expiration overwrote new one: %+v", last)
	}
	_ = two
	_ = three
}
