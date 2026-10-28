//go:build linux

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const unit = "nexttimer.service"

func UnitPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "systemd", "user", unit), nil
}
func Service(action string) error {
	path, err := UnitPath()
	if err != nil {
		return err
	}
	systemctl := func(args ...string) error {
		c := exec.Command("systemctl", append([]string{"--user"}, args...)...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	switch action {
	case "install":
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, err = filepath.Abs(exe)
		if err != nil {
			return err
		}
		if strings.ContainsAny(exe, "\n\r\"\\") {
			return fmt.Errorf("nicht unterstützter Binary-Pfad: %q", exe)
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("Unit bereits vorhanden: %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		exe = strings.ReplaceAll(exe, "%", "%%")
		content := fmt.Sprintf("[Unit]\nDescription=NextTimer Countdown Service\nAfter=network.target\n\n[Service]\nType=simple\nExecStart=\"%s\"\nRestart=on-failure\nRestartSec=3\n\n[Install]\nWantedBy=default.target\n", exe)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			return err
		}
		if err := systemctl("daemon-reload"); err != nil {
			return err
		}
		if err := systemctl("enable", "--now", unit); err != nil {
			return err
		}
		fmt.Println("Installiert:", path)
		return nil
	case "uninstall":
		if _, err := os.Stat(path); err != nil {
			return err
		}
		if err := systemctl("disable", "--now", unit); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		return systemctl("daemon-reload")
	case "status":
		return systemctl("status", unit)
	case "start", "stop", "restart":
		return systemctl(action, unit)
	default:
		return fmt.Errorf("service: install|uninstall|status|start|stop|restart")
	}
}
