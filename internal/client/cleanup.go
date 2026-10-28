package client

import (
	"errors"
	"fmt"
	"net/url"
)

// A CLI-started server timer is stopped on exit by default. This cleanup is
// scoped to the exact ID returned by POST /api/timers: unrelated timers are
// never touched. An already finished/stopped timer requires no cleanup.
func stopOwnedServerTimer(id string) error {
	s, err := state()
	if err == nil {
		for _, t := range s.Timers {
			if t.ID == id {
				if t.Status == "running" || t.Status == "paused" {
					break
				}
				return nil
			}
		}
		// No matching ID means it has already been removed or expired.
		found := false
		for _, t := range s.Timers {
			if t.ID == id {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	// If the status request failed, still attempt to stop the owned timer.
	// This is best effort: no client can guarantee cleanup during a network
	// failure or a forced process kill.
	if stopErr := request("POST", "/api/timers/"+url.PathEscape(id)+"/stop", nil, nil); stopErr != nil {
		return fmt.Errorf("Timer %s beim Verlassen nicht gestoppt: %w", id, stopErr)
	}
	return nil
}

// runAttachedSession performs cleanup even if the TUI exits with an error.
func runAttachedSession(id string, keepRunning bool, run func() error) (err error) {
	if !keepRunning {
		defer func() {
			if stopErr := stopOwnedServerTimer(id); stopErr != nil {
				if err != nil {
					err = errors.Join(err, stopErr)
				} else {
					err = stopErr
				}
			}
		}()
	}
	return run()
}
