//go:build !lite

package client

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/desirevolution/next-timer/internal/protocol"
)

// Terminal modes are deliberately local to the client. They never change the
// server's automatic primary timer selection.
type tuiMode uint8

const (
	overviewMode tuiMode = iota
	watchMode
	newMode
	confirmStopMode
	helpMode
)

type tuiSection uint8

const (
	timersSection tuiSection = iota
	favoritesSection
)

type pulseMsg time.Time
type snapshotMsg struct {
	snap        protocol.Snapshot
	err         error
	requestedAt time.Time
	receivedAt  time.Time
}
type actionMsg struct {
	stayInOverview bool
	err            error
	info           string
	started        *protocol.TimerView
	stopped        bool
}

type tuiModel struct {
	snap          protocol.Snapshot
	remote        protocol.Snapshot
	locals        *localStore
	localOnly     bool
	cliLaunch     bool // timer was created by a direct CLI command; exit instead of showing overview
	now           time.Time
	skew          time.Duration
	width, height int
	ticks         int
	mode          tuiMode
	previousMode  tuiMode
	section       tuiSection
	selectedID    string // empty = follow the server's primary timer
	favoriteIndex int
	watchID       string // pinned to a specific timer; not the global primary ID
	pendingWatch  bool
	doneName      string
	doneSince     time.Time
	confirmID     string
	confirmName   string
	input         string
	message       string
	exitErr       error
}

func fetchSnapshotFor(localOnly bool) tea.Msg {
	if localOnly {
		return snapshotMsg{snap: protocol.Snapshot{ServerNowMS: time.Now().UnixMilli()}}
	}
	start := time.Now()
	s, err := state()
	return snapshotMsg{snap: s, err: err, requestedAt: start, receivedAt: time.Now()}
}
func pulse() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return pulseMsg(t) })
}
func (m tuiModel) Init() tea.Cmd { return pulse() }

func activeTimers(s protocol.Snapshot) []protocol.TimerView {
	list := make([]protocol.TimerView, 0, len(s.Timers))
	for _, t := range s.Timers {
		if t.Status == "running" || t.Status == "paused" {
			list = append(list, t)
		}
	}
	return list
}
func findTimer(s protocol.Snapshot, id string) *protocol.TimerView {
	for i := range s.Timers {
		if s.Timers[i].ID == id {
			return &s.Timers[i]
		}
	}
	return nil
}
func (m tuiModel) target() *protocol.TimerView {
	if m.mode == watchMode && m.watchID != "" {
		t := findTimer(m.snap, m.watchID)
		if t != nil && (t.Status == "running" || t.Status == "paused") {
			return t
		}
		// A finished/missing watched timer must never cause shortcuts to control
		// a different timer while its end state is being processed.
		return nil
	}
	if m.selectedID != "" && m.mode != watchMode {
		t := findTimer(m.snap, m.selectedID)
		if t != nil && (t.Status == "running" || t.Status == "paused") {
			return t
		}
	}
	return findTimer(m.snap, m.snap.FocusID)
}
func (m tuiModel) serverNow() time.Time { return m.now.Add(m.skew) }

func controlCmd(id, action, delta string, locals *localStore) tea.Cmd {
	return func() tea.Msg {
		if isLocalID(id) && locals != nil {
			return actionMsg{err: locals.control(id, action, delta), info: action, stopped: action == "stop"}
		}
		var payload any
		if action == "adjust" {
			payload = map[string]string{"delta": delta}
		}
		err := request("POST", "/api/timers/"+url.PathEscape(id)+"/"+action, payload, nil)
		return actionMsg{err: err, info: action, stopped: action == "stop"}
	}
}
func startFavoriteCmd(tmpl protocol.Template, locals *localStore, forceLocal, stayInOverview bool) tea.Cmd {
	return func() tea.Msg {
		if tmpl.BrowserOnly || forceLocal {
			if locals == nil {
				return actionMsg{err: errors.New("lokale Timer nicht verfügbar")}
			}
			d, err := parseDuration(tmpl.Duration)
			if err != nil {
				return actionMsg{err: err}
			}
			t := locals.create(d, tmpl.Name)
			return actionMsg{info: "Lokaler Favorit gestartet", started: &t, stayInOverview: stayInOverview}
		}
		var t protocol.TimerView
		err := request("POST", "/api/timers", map[string]string{"template_id": tmpl.ID}, &t)
		return actionMsg{err: err, info: "Favorit gestartet", started: &t, stayInOverview: stayInOverview}
	}
}
func createCmd(input string, locals *localStore, localOnly, overview bool) tea.Cmd {
	return func() tea.Msg {
		input = strings.TrimSpace(input)
		forceLocal := localOnly || strings.HasPrefix(input, "!")
		if strings.HasPrefix(input, "!") {
			input = strings.TrimSpace(strings.TrimPrefix(input, "!"))
		}
		fields := strings.Fields(input)
		if len(fields) == 0 {
			return actionMsg{err: errors.New("Dauer eingeben (z. B. 10m Kaffee)")}
		}
		d, err := parseDuration(fields[0])
		if err != nil {
			return actionMsg{err: err}
		}
		name := strings.Join(fields[1:], " ")
		if forceLocal {
			if locals == nil {
				return actionMsg{err: errors.New("lokale Timer nicht verfügbar")}
			}
			t := locals.create(d, name)
			return actionMsg{info: "Lokaler Timer gestartet", started: &t, stayInOverview: overview}
		}
		var t protocol.TimerView
		err = request("POST", "/api/timers", map[string]string{"duration": d.String(), "name": name}, &t)
		return actionMsg{err: err, info: "Timer gestartet", started: &t, stayInOverview: overview}
	}
}

func (m tuiModel) newWatch(id string) tuiModel {
	m.mode = watchMode
	m.watchID = id
	m.doneName = ""
	m.doneSince = time.Time{}
	m.pendingWatch = findTimer(m.snap, id) == nil
	m.selectedID = ""
	return m
}

func (m tuiModel) handleSnapshot(s protocol.Snapshot) tuiModel {
	m.remote = s
	if m.locals != nil {
		s = mergeLocal(s, m.locals.snapshot())
	}
	m.snap = s
	m.skew = time.UnixMilli(s.ServerNowMS).Sub(time.Now())
	if m.pendingWatch && findTimer(s, m.watchID) != nil {
		m.pendingWatch = false
	}
	if m.selectedID != "" {
		if t := findTimer(s, m.selectedID); t == nil || (t.Status != "running" && t.Status != "paused") {
			m.selectedID = ""
		}
	}
	if m.favoriteIndex >= len(s.Templates) {
		m.favoriteIndex = max(0, len(s.Templates)-1)
	}
	if m.mode == watchMode && !m.cliLaunch && m.watchID != "" && !m.pendingWatch && m.doneSince.IsZero() {
		watched := findTimer(s, m.watchID)
		if watched == nil || watched.Status == "finished" {
			if watched != nil {
				m.doneName = watched.Name
			} else if m.doneName == "" {
				m.doneName = "Timer"
			}
			m.doneSince = time.Now()
		}
	}
	return m
}

// Attached CLI launches close as soon as the server confirms completion (or
// the local timer reaches finished). Do not quit based solely on a locally
// calculated zero: the server may not yet have emitted its finish webhook,
// and the owned-timer cleanup would otherwise cancel the timer prematurely.
func (m tuiModel) attachedTimerFinished() bool {
	if !m.cliLaunch || m.mode != watchMode || m.watchID == "" || m.pendingWatch {
		return false
	}
	t := findTimer(m.snap, m.watchID)
	return t == nil || t.Status == "finished" || t.Status == "stopped"
}

func (m tuiModel) advanceAfterDone() tuiModel {
	if m.mode != watchMode || m.doneSince.IsZero() || time.Since(m.doneSince) < 2500*time.Millisecond {
		return m
	}
	if m.cliLaunch {
		return m // CLI sessions quit immediately on the finished snapshot.
	}
	if m.snap.FocusID != "" && m.snap.FocusID != m.watchID {
		return m.newWatch(m.snap.FocusID)
	}
	m.mode = overviewMode
	m.watchID = ""
	m.doneSince = time.Time{}
	m.doneName = ""
	return m
}

func (m tuiModel) moveSelection(delta int) tuiModel {
	if m.section == favoritesSection {
		if len(m.snap.Templates) > 0 {
			m.favoriteIndex = (m.favoriteIndex + delta + len(m.snap.Templates)) % len(m.snap.Templates)
		}
		return m
	}
	active := activeTimers(m.snap)
	if len(active) == 0 {
		m.selectedID = ""
		return m
	}
	index := -1
	if m.selectedID != "" {
		for i, t := range active {
			if t.ID == m.selectedID {
				index = i
				break
			}
		}
	}
	if index < 0 {
		for i, t := range active {
			if t.ID == m.snap.FocusID {
				index = i
				break
			}
		}
	}
	if index < 0 {
		index = 0
	}
	m.selectedID = active[(index+delta+len(active))%len(active)].ID
	return m
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = x.Width, x.Height
	case snapshotMsg:
		if x.err != nil {
			m.exitErr = fmt.Errorf("Verbindung zum NextTimer-Server verloren: %w", x.err)
			return m, tea.Quit
		}
		m = m.handleSnapshot(x.snap)
		if !x.requestedAt.IsZero() && !x.receivedAt.IsZero() {
			midpoint := x.requestedAt.Add(x.receivedAt.Sub(x.requestedAt) / 2)
			m.skew = time.UnixMilli(x.snap.ServerNowMS).Sub(midpoint)
		}
		if m.attachedTimerFinished() {
			return m, tea.Quit
		}
	case pulseMsg:
		m.now = time.Time(x)
		if m.locals != nil {
			// Local timers reach finished without any server round-trip.
			m.snap = mergeLocal(m.remote, m.locals.snapshot())
			if m.mode == watchMode && !m.cliLaunch && m.watchID != "" && m.doneSince.IsZero() && !m.pendingWatch {
				if t := findTimer(m.snap, m.watchID); t == nil || t.Status == "finished" {
					m.doneName = "Timer"
					if t != nil {
						m.doneName = t.Name
					}
					m.doneSince = time.Now()
				}
			}
		}
		if m.attachedTimerFinished() {
			return m, tea.Quit
		}
		m = m.advanceAfterDone()
		m.ticks++
		fetchEvery := 5 // once/second normally
		if m.cliLaunch && m.mode == watchMode {
			if t := findTimer(m.snap, m.watchID); t != nil && remaining(*t, m.serverNow()) < 2*time.Second {
				fetchEvery = 2 // check finish confirmation faster near zero
			}
		}
		if m.ticks%fetchEvery == 0 {
			return m, tea.Batch(pulse(), func() tea.Msg { return fetchSnapshotFor(m.localOnly) })
		}
		return m, pulse()
	case actionMsg:
		if x.err != nil {
			m.message = x.err.Error()
		} else {
			m.message = x.info
			if x.started != nil {
				if x.stayInOverview {
					m.mode = overviewMode
					m.watchID = ""
					m.pendingWatch = false
				} else {
					m = m.newWatch(x.started.ID)
				}
			}
			if x.stopped && m.cliLaunch {
				return m, tea.Quit
			}
			if x.stopped {
				m.mode = overviewMode
				m.watchID = ""
				m.selectedID = ""
				m.doneSince = time.Time{}
			}
		}
		return m, func() tea.Msg { return fetchSnapshotFor(m.localOnly) }
	case tea.KeyPressMsg:
		key := x.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.mode {
		case confirmStopMode:
			switch key {
			case "esc", "n":
				m.mode = m.previousMode
				m.confirmID = ""
				m.confirmName = ""
			case "enter", "y":
				id := m.confirmID
				m.mode = m.previousMode
				m.confirmID, m.confirmName = "", ""
				if id != "" {
					return m, controlCmd(id, "stop", "", m.locals)
				}
			}
			return m, nil
		case newMode:
			switch key {
			case "esc":
				m.mode = overviewMode
				m.input = ""
			case "enter", "alt+enter":
				entry := m.input
				m.mode, m.input = overviewMode, ""
				return m, createCmd(entry, m.locals, m.localOnly, key == "alt+enter")
			case "backspace":
				if len(m.input) > 0 {
					_, n := utf8.DecodeLastRuneInString(m.input)
					m.input = m.input[:len(m.input)-n]
				}
			default:
				m.input += x.Key().Text
			}
			return m, nil
		case helpMode:
			if key == "esc" || key == "?" || key == "q" {
				m.mode = m.previousMode
			}
			return m, nil
		}
		if key == "q" {
			return m, tea.Quit
		}
		if key == "?" {
			m.previousMode = m.mode
			m.mode = helpMode
			return m, nil
		}
		if key == "esc" {
			if m.cliLaunch {
				return m, tea.Quit
			}
			if m.mode == watchMode {
				m.mode = overviewMode
				m.watchID = ""
				m.doneSince = time.Time{}
			} else {
				m.selectedID = ""
			}
			return m, nil
		}
		// In fullscreen, Esc/q/Ctrl+C already exit the client. For an
		// attached timer the normal exit policy stops it by default.
		// The stop-confirmation shortcut remains available in the dashboard.
		if key == "x" && m.mode != watchMode {
			if t := m.target(); t != nil && (t.Status == "running" || t.Status == "paused") {
				m.previousMode = m.mode
				m.mode = confirmStopMode
				m.confirmID, m.confirmName = t.ID, t.Name
			}
			return m, nil
		}
		if key == "space" || key == "p" || key == "+" || key == "=" || key == "-" {
			if t := m.target(); t != nil {
				switch key {
				case "space", "p":
					if t.Status == "running" {
						return m, controlCmd(t.ID, "pause", "", m.locals)
					}
					if t.Status == "paused" {
						return m, controlCmd(t.ID, "resume", "", m.locals)
					}
				case "+", "=":
					return m, controlCmd(t.ID, "adjust", "+1m", m.locals)
				case "-":
					return m, controlCmd(t.ID, "adjust", "-1m", m.locals)
				}
			}
			return m, nil
		}
		if key == "n" && !m.cliLaunch {
			m.mode = newMode
			m.input = ""
			return m, nil
		}
		if key == "r" {
			return m, func() tea.Msg { return fetchSnapshotFor(m.localOnly) }
		}
		if m.mode == watchMode {
			return m, nil
		}
		shortcut := key
		stay := false
		if strings.HasPrefix(shortcut, "alt+") {
			shortcut = strings.TrimPrefix(shortcut, "alt+")
			stay = true
		}
		if len(shortcut) == 1 && shortcut[0] >= '0' && shortcut[0] <= '9' {
			index := int(shortcut[0] - '1')
			if shortcut == "0" {
				index = 9
			}
			if index >= 0 && index < len(m.snap.Templates) {
				return m, startFavoriteCmd(m.snap.Templates[index], m.locals, m.localOnly, stay)
			}
			return m, nil
		}
		switch key {
		case "tab":
			if m.section == timersSection {
				m.section = favoritesSection
				m.selectedID = ""
			} else {
				m.section = timersSection
			}
		case "up", "k":
			m = m.moveSelection(-1)
		case "down", "j":
			m = m.moveSelection(1)
		case "w":
			if t := m.target(); t != nil {
				m = m.newWatch(t.ID)
			}
		case "enter", "alt+enter":
			if m.section == favoritesSection {
				if m.favoriteIndex < len(m.snap.Templates) {
					return m, startFavoriteCmd(m.snap.Templates[m.favoriteIndex], m.locals, m.localOnly, key == "alt+enter")
				}
			} else if t := m.target(); t != nil {
				m = m.newWatch(t.ID)
			}
		}
	}
	return m, nil
}

func (m tuiModel) View() tea.View {
	v := tea.NewView(renderTUI(m))
	v.AltScreen = true
	return v
}

func drawTimer(b *strings.Builder, t protocol.TimerView, now time.Time, width int, large bool) {
	remainingDuration := remaining(t, now)
	clock := formatClock(remainingDuration)
	urgent := t.Status == "running" && remainingDuration <= 10*time.Second
	colorStart, colorEnd := "", ""
	if urgent {
		colorStart, colorEnd = "\x1b[1;31m", "\x1b[0m"
	}
	label := "HAUPTTIMER"
	if large {
		label = "COUNTDOWN"
	}
	b.WriteString(fmt.Sprintf("  %s · %s%s  [#%s]\n\n", label, "", t.Name, t.ID))
	if width >= len([]rune(clock))*6+8 && (large || width > 65) {
		b.WriteString(colorStart)
		for _, line := range strings.Split(largeClockAdaptive(clock, width-4, 11), "\n") {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString(colorEnd)
	} else {
		b.WriteString("  " + colorStart + clock + colorEnd + "\n")
	}
	frac := 0.0
	if t.DurationMS > 0 {
		frac = 1 - float64(remainingDuration.Milliseconds())/float64(t.DurationMS)
	}
	b.WriteString("\n  " + colorStart + progressBar(frac, min(max(8, width-8), 50)) + colorEnd + "\n")
	if t.Status == "paused" {
		b.WriteString("  PAUSIERT\n")
	}
}
func shortName(name string, limit int) string {
	r := []rune(strings.TrimSpace(name))
	if len(r) > limit {
		return string(r[:limit-1]) + "…"
	}
	return string(r)
}

// runTUIConfigured never stops timers in a normal dashboard or watch session.
// Only direct CLI launches opt in to the owned-timer lifecycle.
func runTUIConfigured(id string, locals *localStore, localOnly, cliLaunch bool) error {
	var s protocol.Snapshot
	var err error
	requestedAt := time.Now()
	if !localOnly {
		s, err = state()
		if err != nil {
			return err
		}
	}
	receivedAt := time.Now()
	if locals == nil {
		locals = &localStore{}
	}
	m := tuiModel{remote: s, snap: mergeLocal(s, locals.snapshot()), locals: locals,
		localOnly: localOnly, cliLaunch: cliLaunch, now: time.Now(),
		width: 80, height: 30, section: favoritesSection}
	if !localOnly {
		m.skew = time.UnixMilli(m.snap.ServerNowMS).Sub(requestedAt.Add(receivedAt.Sub(requestedAt) / 2))
	}
	if id != "" {
		m = m.newWatch(id)
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if m2, ok := final.(tuiModel); ok && m2.exitErr != nil {
		return m2.exitErr
	}
	return nil
}

func runTUIAttached(id string, keepRunning bool) error {
	return runAttachedSession(id, keepRunning, func() error {
		return runTUIConfigured(id, nil, false, true)
	})
}

func runTUIWithID(id string) error { return runTUIConfigured(id, nil, false, false) }
func runTUI() error                { return runTUIWithID("") }
func runTUILocal(duration time.Duration, name string, focus bool) error {
	locals := &localStore{}
	id := ""
	if duration > 0 {
		t := locals.create(duration, name)
		if focus {
			id = t.ID
		}
	}
	return runTUIConfigured(id, locals, true, false)
}
func runTUILocalAttached(duration time.Duration, name string) error {
	locals := &localStore{}
	t := locals.create(duration, name)
	return runTUIConfigured(t.ID, locals, true, true)
}
func runTUIOverview() error { return runTUIConfigured("", nil, false, false) }
