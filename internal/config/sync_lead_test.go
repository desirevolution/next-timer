package config

import "testing"

func TestAWTRIXSyncLeadDefaultAndValidation(t *testing.T) {
	c, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if c.AWTRIXSyncLead != "100ms" || c.Validate() != nil {
		t.Fatalf("unexpected sync default %+v: %v", c, c.Validate())
	}
	for _, raw := range []string{"0ms", "80ms", "500ms"} {
		c.AWTRIXSyncLead = raw
		if err := c.Validate(); err != nil {
			t.Errorf("valid lead %s rejected: %v", raw, err)
		}
	}
	for _, raw := range []string{"-10ms", "501ms", "bad"} {
		c.AWTRIXSyncLead = raw
		if err := c.Validate(); err == nil {
			t.Errorf("invalid lead %s accepted", raw)
		}
	}
}
func TestAWTRIXSyncLeadEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AWTRIX_SYNC_LEAD", "180ms")
	c, _, err := Read()
	if err != nil || c.AWTRIXSyncLead != "180ms" {
		t.Fatalf("env override: %+v %v", c, err)
	}
}
