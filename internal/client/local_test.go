package client

import (
	"strings"
	"testing"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func TestLocalTimerStartPauseResumeAdjustFinishStop(t *testing.T) {
	s := &localStore{}
	tm := s.create(50*time.Second, "Kaffee")
	if tm.ID != "L1" || !tm.Local || tm.Status != "running" {
		t.Fatalf("unexpected local timer: %+v", tm)
	}
	if err := s.control(tm.ID, "pause", ""); err != nil {
		t.Fatal(err)
	}
	snap := s.snapshot()
	if snap[0].Status != "paused" {
		t.Fatal("did not pause")
	}
	if err := s.control(tm.ID, "adjust", "+20s"); err != nil {
		t.Fatal(err)
	}
	if snap = s.snapshot(); snap[0].DurationMS != 70000 || snap[0].RemainingMS < 69000 {
		t.Fatalf("bad adjust: %+v", snap[0])
	}
	if err := s.control(tm.ID, "resume", ""); err != nil {
		t.Fatal(err)
	}
	if snap = s.snapshot(); snap[0].Status != "running" {
		t.Fatal("did not resume")
	}
	if err := s.control(tm.ID, "adjust", "-71s"); err != nil {
		t.Fatal(err)
	}
	if snap = s.snapshot(); snap[0].Status != "finished" {
		t.Fatalf("should finish on negative adjustment: %+v", snap[0])
	}
	if err := s.control(tm.ID, "stop", ""); err != nil {
		t.Fatal(err)
	}
	if got := len(s.snapshot()); got != 0 {
		t.Fatalf("expected empty local timer list, got %d", got)
	}
}
func TestLocalExpiryAndFocusSelection(t *testing.T) {
	store := &localStore{}
	a := store.create(30*time.Millisecond, "Kurz")
	b := store.create(15*time.Minute, "Lang")
	remote := protocol.Snapshot{FocusID: "server", Timers: []protocol.TimerView{{ID: "server", Name: "Remote", Status: "running", DeadlineMS: time.Now().Add(10 * time.Minute).UnixMilli()}}}
	snap := mergeLocal(remote, store.snapshot())
	if snap.FocusID != a.ID {
		t.Fatalf("short timer must be primary, got %s", snap.FocusID)
	}
	time.Sleep(45 * time.Millisecond)
	snap = mergeLocal(remote, store.snapshot())
	if snap.FocusID != "server" {
		t.Fatalf("expected server to become primary, got %s", snap.FocusID)
	}
	if snap.Timers[1].Status != "finished" {
		t.Fatal("local expiration not detected")
	}
	if len(remote.Timers) != 1 || remote.FocusID != "server" {
		t.Fatal("merge changed remote snapshot")
	}
	_ = b
}
func TestDetachLocalRejected(t *testing.T) {
	err := RunWithClientOptions([]string{"10m", "Kaffee"}, Options{Detach: true, Local: true})
	if err == nil || !strings.Contains(err.Error(), "unvereinbar") {
		t.Fatalf("expected explanatory error, got %v", err)
	}
}
