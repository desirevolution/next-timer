//go:build !linux

package client

import "os"

func isTerminal() bool { s, e := os.Stdin.Stat(); return e == nil && s.Mode()&os.ModeCharDevice != 0 }
