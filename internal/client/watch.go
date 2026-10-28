package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

func watchArgs(args []string) (string, bool, error) {
	var selector string
	compact := false
	for _, arg := range args {
		switch arg {
		case "--compact", "-c":
			compact = true
		case "--big":
			compact = false
		default:
			if strings.HasPrefix(arg, "-") || selector != "" {
				return "", false, errors.New("Syntax: timer watch [--compact] [ID oder Name]")
			}
			selector = arg
		}
	}
	return selector, compact, nil
}

func progressBar(fraction float64, width int) string {
	if width < 8 {
		width = 8
	}
	if width > 72 {
		width = 72
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	done := int(float64(width)*fraction + 0.5)
	return "[" + strings.Repeat("█", done) + strings.Repeat("░", width-done) + "]"
}
func terminalWidth() int {
	n, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	if n < 20 || n > 500 {
		return 80
	}
	return n
}
func renderWatch(t protocol.TimerView, now time.Time, compact, tty bool) string {
	rem := remaining(t, now)
	status := map[string]string{"running": "Läuft", "paused": "Pausiert", "finished": "Fertig"}[t.Status]
	if status == "" {
		status = t.Status
	}
	if compact || !tty || terminalWidth() < 62 {
		return fmt.Sprintf("%s  %s  [%s]", formatClock(rem), t.Name, status)
	}
	f := 0.0
	if t.DurationMS > 0 {
		f = 1 - float64(rem.Milliseconds())/float64(t.DurationMS)
	}
	var b strings.Builder
	b.WriteString("NEXTTIMER                                      [Strg+C: Anzeige schließen]\n\n")
	b.WriteString("  " + t.Name + "  ·  " + status + "\n\n")
	for _, line := range strings.Split(largeClockAdaptive(formatClock(rem), terminalWidth()-4, 15), "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n  " + progressBar(f, min(terminalWidth()-6, 55)) + "\n")
	b.WriteString("\n  Timer ID: " + t.ID + "\n")
	return b.String()
}
func watch(selector string, compact bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	tty := false
	if info, err := os.Stdout.Stat(); err == nil {
		tty = info.Mode()&os.ModeCharDevice != 0
	}
	if tty {
		fmt.Print("\x1b[?1049h\x1b[?25l")
		defer fmt.Print("\x1b[?25h\x1b[?1049l")
	}
	ticker := time.NewTicker(125 * time.Millisecond)
	defer ticker.Stop()
	var snap protocol.Snapshot
	var lastFetch time.Time
	var skewMs int64
	var lastFrame string
	var lastLine string
	for {
		now := time.Now()
		if now.Sub(lastFetch) >= 750*time.Millisecond {
			started := time.Now()
			s, err := state()
			if err != nil {
				return err
			}
			received := time.Now()
			snap = s
			// Estimate the server clock at the request midpoint, not its end.
			skewMs = s.ServerNowMS - started.Add(received.Sub(started)/2).UnixMilli()
			lastFetch = received
		}
		t, err := mainTimer(snap, selector, false)
		frame := "Kein laufender Timer. Strg+C schließt nur diese Anzeige."
		if err == nil {
			frame = renderWatch(t, now.Add(time.Duration(skewMs)*time.Millisecond), compact, tty)
		}
		if tty {
			if frame != lastFrame {
				fmt.Printf("\x1b[H\x1b[J%s", frame)
				lastFrame = frame
			}
		} else if frame != lastLine {
			fmt.Println(frame)
			lastLine = frame
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
