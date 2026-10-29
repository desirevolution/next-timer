package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func TestBrowserAlarmProfileOnFavoriteAndStartedTimer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "templates.json")
	s, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	p := &protocol.BrowserAlarm{Tone: "gong", Repeat: "until_ack", Visual: true, System: false}
	fav, err := s.AddTemplateConfigured(Template{Name: "Tea", Duration: "3m", Group: "Kitchen", Alarm: p})
	if err != nil {
		t.Fatal(err)
	}
	timer, err := s.StartTemplate(fav.ID)
	if err != nil {
		t.Fatal(err)
	}
	if timer.Alarm == nil || *timer.Alarm != *p {
		t.Fatalf("Timer lost profile: %+v", timer.Alarm)
	}
	restored, err := NewService(path, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got := restored.Snapshot().Templates[0]
	if got.Alarm == nil || *got.Alarm != *p {
		t.Fatalf("Favorite lost profile: %+v", got.Alarm)
	}
	edited, err := restored.UpdateTemplateConfigured(fav.ID, Template{Name: "Tea", Duration: "3m", Group: "Kitchen", Alarm: nil})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Alarm != nil {
		t.Fatal("Profile override cannot be removed")
	}
	for _, bad := range []*protocol.BrowserAlarm{{Tone: "invalid", Repeat: "once", Visual: true}, {Tone: "bell", Repeat: "forever", Visual: true}} {
		if _, err := restored.AddTemplateConfigured(Template{Name: "Invalid", Duration: "1m", Alarm: bad}); err == nil {
			t.Fatal("Invalid profile accepted")
		}
	}
}

func TestBrowserAlarmTimerHTTP(t *testing.T) {
	ts := httptest.NewServer((API{Service: makeService(t)}).Handler())
	defer ts.Close()
	r, err := http.Post(ts.URL+"/api/timers", "application/json", strings.NewReader(`{"name":"Focus","duration":"2m","alarm":{"tone":"digital","repeat":"three","visual":false,"system":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 201 {
		t.Fatalf("status %d", r.StatusCode)
	}
	var timer TimerView
	if err := json.NewDecoder(r.Body).Decode(&timer); err != nil {
		t.Fatal(err)
	}
	if timer.Alarm == nil || timer.Alarm.Tone != "digital" || timer.Alarm.Visual || !timer.Alarm.System {
		t.Fatalf("Invalid serialized profile: %+v", timer.Alarm)
	}
}
