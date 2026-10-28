package client

import (
	"github.com/desirevolution/next-timer/internal/protocol"
	"testing"
)

func TestSelector(t *testing.T) {
	s := protocol.Snapshot{FocusID: "2", Timers: []protocol.TimerView{{ID: "2", Name: "Kaffee", Status: "running"}, {ID: "1", Name: "Arbeit", Status: "running"}}}
	for _, sample := range []struct{ in, want string }{{"", "2"}, {"1", "1"}, {"#1", "2"}, {"#2", "1"}, {"Arbeit", "1"}} {
		timer, err := mainTimer(s, sample.in, false)
		if err != nil || timer.ID != sample.want {
			t.Errorf("%s: %+v %v", sample.in, timer, err)
		}
	}
}
