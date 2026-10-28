//go:build !lite

package client

import (
	"github.com/desirevolution/next-timer/internal/protocol"
	"testing"
	"time"
)

func TestLocalQuickstartAndNoFocus(t *testing.T) {
	s := sampleTUI()
	s.locals = &localStore{}
	s, _ = updated(s, "n")
	for _, c := range "!5m Tee" {
		s, _ = updated(s, string(c))
	}
	next, cmd := updated(s, "enter")
	if cmd == nil || next.mode != overviewMode {
		t.Fatal("quickstart should run command")
	}
	msg := cmd()
	if msg.(actionMsg).err != nil {
		t.Fatalf("local quickstart: %v", msg.(actionMsg).err)
	}
	if msg.(actionMsg).started == nil || !msg.(actionMsg).started.Local {
		t.Fatal("not local")
	}
	result, _ := next.Update(msg)
	s = result.(tuiModel)
	if s.mode != watchMode {
		t.Fatal("local quickstart must open focus")
	}
	if got := len(s.locals.snapshot()); got != 1 {
		t.Fatalf("expected 1 local timer, got %d", got)
	}
}
func TestLocalFavoriteAndLocalControls(t *testing.T) {
	m := sampleTUI()
	m.locals = &localStore{}
	m.snap.Templates = []protocol.Template{{ID: "a", Name: "Lokal", Duration: "1m", BrowserOnly: true}}
	cmd := startFavoriteCmd(m.snap.Templates[0], m.locals, false, false)
	msg := cmd().(actionMsg)
	if msg.err != nil || msg.started == nil || !msg.started.Local {
		t.Fatalf("expected local favorite: %+v", msg)
	}
	tmr := msg.started
	if err := m.locals.control(tmr.ID, "adjust", "+1m"); err != nil {
		t.Fatal(err)
	}
	if s := m.locals.snapshot()[0]; s.DurationMS != 2*int64(time.Minute/time.Millisecond) {
		t.Fatalf("duration: %+v", s)
	}
	if msg := controlCmd(tmr.ID, "pause", "", m.locals)().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	if got := m.locals.snapshot()[0].Status; got != "paused" {
		t.Fatalf("pause: %s", got)
	}
}
func TestStayInOverviewAfterStart(t *testing.T) {
	m := sampleTUI()
	m.locals = &localStore{}
	tm := m.locals.create(time.Minute, "Tee")
	next, _ := m.Update(actionMsg{started: &tm, stayInOverview: true})
	if got := next.(tuiModel).mode; got != overviewMode {
		t.Fatalf("no-focus mode: %d", got)
	}
}
