package server

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The Go counterpart of ui/js/quick-timer.js. Both parsers are checked
// against scripts/quick-input-cases.json so the REST and browser input agree.
var quickUnit = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:ms|h|m|s)`)

type quickInput struct {
	Duration time.Duration
	Name     string
	Shortcut string
}

type quickInputError struct {
	Code    string
	Message string
}

func (e *quickInputError) Error() string { return e.Message }

func invalidQuick(code, message string) error {
	return &quickInputError{Code: code, Message: message}
}

func parseQuickInput(value string) (quickInput, error) {
	input := strings.TrimSpace(value)
	if strings.HasPrefix(input, "#") {
		if len(input) == 2 && input[1] >= '0' && input[1] <= '9' {
			return quickInput{Shortcut: input[1:]}, nil
		}
		return quickInput{}, invalidQuick("INVALID_SHORTCUT", "Favorit mit #1 bis #0 aufrufen.")
	}
	if input == "" {
		return quickInput{}, invalidQuick("INVALID_DURATION", "Dauer angeben, z. B. 5m Tee.")
	}

	var units strings.Builder
	index := 0
	for index < len(input) {
		beforeSpace := index
		for index < len(input) {
			r, size := utf8.DecodeRuneInString(input[index:])
			if !unicode.IsSpace(r) {
				break
			}
			index += size
		}
		match := quickUnit.FindString(input[index:])
		if match == "" {
			index = beforeSpace // whitespace belongs to the name, not the duration
			break
		}
		units.WriteString(match)
		index += len(match)
		if index < len(input) {
			r, _ := utf8.DecodeRuneInString(input[index:])
			if !unicode.IsSpace(r) && (r < '0' || r > '9') {
				return quickInput{}, invalidQuick("INVALID_INPUT", "Dauer und Name mit einem Leerzeichen trennen, z. B. 5m Tee.")
			}
		}
	}

	duration, err := parseDuration(units.String())
	if err != nil {
		return quickInput{}, invalidQuick("INVALID_DURATION", err.Error())
	}
	name := strings.TrimSpace(input[index:])
	if len([]rune(name)) > 80 {
		return quickInput{}, invalidQuick("NAME_TOO_LONG", "Name zu lang (maximal 80 Zeichen).")
	}
	// The JS duration parser rounds to whole milliseconds. Match its timer
	// duration rather than letting Go preserve sub-millisecond fractions.
	duration = time.Duration(math.Round(float64(duration)/float64(time.Millisecond))) * time.Millisecond
	return quickInput{Duration: duration, Name: name}, nil
}

// StartQuick resolves shortcut and starts the timer under one service lock.
// A shortcut only ever resolves a server-side favorite.
func (s *Service) StartQuick(input string) (*TimerView, error) {
	parsed, err := parseQuickInput(input)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if parsed.Shortcut != "" {
		for _, favorite := range s.templates {
			if favorite.Shortcut != parsed.Shortcut || favorite.BrowserOnly {
				continue
			}
			duration, err := parseDuration(favorite.Duration)
			if err != nil {
				return nil, invalidQuick("INVALID_DURATION", err.Error())
			}
			return s.startConfiguredLocked(favorite.Name, duration,
				TimerOptions{Sound: enabled(favorite.Sound), Notify: enabled(favorite.Notify), Alarm: favorite.Alarm}, time.Now()), nil
		}
		return nil, invalidQuick("FAVORITE_NOT_FOUND", "Kein serverseitiger Favorit für #"+parsed.Shortcut+" zugewiesen.")
	}
	return s.startConfiguredLocked(parsed.Name, parsed.Duration, TimerOptions{Sound: true, Notify: true}, time.Now()), nil
}

// StopNext stops the next *running* timer, never a paused or finished one.
// Expirations are processed at this instant before selecting the target;
// both the selection and stop are atomic with respect to other REST actions.
func (s *Service) StopNext() *TimerView {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, timer := range s.timers {
		if timer.Status == "running" && !timer.Deadline.After(now) {
			s.finishLocked(timer, now)
		}
	}
	target := s.focusLocked()
	if target == nil {
		return nil
	}
	return s.stopLocked(target, now)
}

// stopLocked is shared by ID-based and next/stop endpoints.
// Its returned view follows the existing stop endpoint's response semantics.
func (s *Service) stopLocked(timer *Timer, now time.Time) *TimerView {
	if timer.Status == "running" || timer.Status == "paused" {
		s.recordCompletionLocked(timer, now)
	}
	delete(s.timers, timer.ID)
	s.publishLocked("stopped", timer, now)
	view := timer.view(now)
	return &view
}

func quickErrorCode(err error) string {
	var quickErr *quickInputError
	if errors.As(err, &quickErr) {
		return quickErr.Code
	}
	return "INVALID_INPUT"
}
