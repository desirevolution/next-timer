package config

import "testing"

func TestAlarmWebhookValidation(t *testing.T) {
	c, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://localhost:1880/finished", "https://hooks.example.org/timer"} {
		c.AlarmWebhook = raw
		if err := c.Validate(); err != nil {
			t.Errorf("valid url %q rejected: %v", raw, err)
		}
	}
	for _, raw := range []string{"file:///tmp/timer", "http://", "https://username:pass@example.org/finish", "not-a-url"} {
		c.AlarmWebhook = raw
		if err := c.Validate(); err == nil {
			t.Errorf("invalid url %q accepted", raw)
		}
	}
}
func TestAlarmWebhookEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TIMER_ALARM_WEBHOOK_URL", "http://localhost:1880/timer")
	c, _, err := Read()
	if err != nil || c.AlarmWebhook != "http://localhost:1880/timer" {
		t.Fatalf("env override failed: %+v %v", c, err)
	}
}
