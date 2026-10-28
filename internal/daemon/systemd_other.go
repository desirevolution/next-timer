//go:build !linux

package daemon

import "errors"

func Service(string) error {
	return errors.New("service-Kommandos werden derzeit nur unter Linux/systemd unterstützt")
}
