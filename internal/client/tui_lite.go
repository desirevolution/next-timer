//go:build lite

package client

import (
	"errors"
	"time"
)

func runTUIWithID(id string) error { return watch(id, false) }
func runTUI() error                { return runTUIWithID("") }
func runTUIOverview() error        { return runTUI() }
func runTUILocal(duration time.Duration, name string, focus bool) error {
	return errors.New("lokale TUI ist im Lite-Build nicht verfügbar")
}

func runTUIAttached(id string, keepRunning bool) error {
	return errors.New("interaktiver CLI-Fokusmodus ist im Lite-Build nicht verfügbar")
}
func runTUILocalAttached(duration time.Duration, name string) error {
	return errors.New("lokaler CLI-Fokusmodus ist im Lite-Build nicht verfügbar")
}
