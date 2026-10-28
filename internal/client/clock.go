package client

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func parseDuration(input string) (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(input))
	if err != nil || d < time.Second || d > 7*24*time.Hour {
		return 0, errors.New("Dauer muss zwischen 1s und 168h liegen (z.B. 10m, 1h30m)")
	}
	return d, nil
}
func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int64(math.Ceil(d.Seconds()))
	if sec >= 3600 {
		return fmt.Sprintf("%02d:%02d:%02d", sec/3600, (sec/60)%60, sec%60)
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}
