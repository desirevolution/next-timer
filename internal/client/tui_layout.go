//go:build !lite

package client

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/desirevolution/next-timer/internal/protocol"
)

// Only presentation lives in this file. Networking, timer selection and
// key handling stay in tui_bubbletea.go.
var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E6EDF5")).Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#8795A7"))
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#75D5AC")).Bold(true)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5766")).Bold(true)
	borderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#394555"))
	panelStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#1B2734"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("#284B50")).Foreground(lipgloss.Color("#FFFFFF"))
)

func viewSize(m tuiModel) (int, int) {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 30
	}
	return width, height
}

func renderTUI(m tuiModel) string {
	width, height := viewSize(m)
	switch m.mode {
	case helpMode:
		return renderDialog(width, height, "TASTENHILFE", []string{
			"n                 Neuer Timer: 10m Kaffee (Server)",
			"n, !5m Kaffee     Lokalen Client-Timer starten",
			"Alt+Enter         Starten und Übersicht behalten",
			"Alt+1–9 / Alt+0   Favorit starten, Übersicht behalten",
			"1–9 / 0           Favorit starten + Fokus",
			"Tab               Bereich wechseln",
			"↑/↓ oder j/k      Eintrag auswählen",
			"Enter             Starten / Ansehen",
			"Space / p         Pause / Fortsetzen",
			"+ / −             Eine Minute ändern",
			"x                 Stoppen (nur in der Übersicht)",
			"Esc               Zurück / Auswahl aufheben",
			"q / Ctrl+C        Client schließen (lokale Timer enden dabei)",
			"",
			"Esc · Zurück",
		})
	case newMode:
		return renderDialog(width, height, "NEUER TIMER", []string{
			mutedStyle.Render("Dauer und optional Name"),
			"",
			accentStyle.Render("› ") + m.input + "▌",
			"",
			mutedStyle.Render("Beispiel: 10m Kaffee oder !5m Kaffee (lokal)"),
			"Enter · Fokus     Alt+Enter · Übersicht     Esc · Abbrechen",
		})
	case confirmStopMode:
		return renderDialog(width, height, "TIMER STOPPEN?", []string{
			warningStyle.Render(shortName(m.confirmName, max(12, width-15))),
			mutedStyle.Render("Timer #" + m.confirmID + " wirklich beenden?"),
			"",
			"Enter · Bestätigen     Esc · Abbrechen",
		})
	case watchMode:
		return renderFocus(m, width, height)
	default:
		return renderOverview(m, width, height)
	}
}

func renderDialog(width, height int, heading string, lines []string) string {
	content := titleStyle.Render(heading) + "\n\n" + strings.Join(lines, "\n")
	innerWidth := min(max(24, width-6), 58)
	if width < 36 {
		innerWidth = max(12, width-6)
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#49576B")).
		Padding(1, 2).
		Width(innerWidth).
		Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func styledProgress(fraction float64, width int, urgent bool) string {
	width = max(6, width)
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	n := int(float64(width)*fraction + .5)
	c := accentStyle
	if urgent {
		c = warningStyle
	}
	return c.Render(strings.Repeat("━", n)) + mutedStyle.Render(strings.Repeat("─", width-n))
}

// shortDurationLabel is a compact hint about the timer's configured duration.
// The hint is presentation-only: it does not change timer execution.
func shortDurationLabel(ms int64) string {
	sec := ms / 1000
	if sec < 60 {
		return fmt.Sprintf("%d Sek.", sec)
	}
	if sec < 3600 {
		if sec%60 == 0 {
			return fmt.Sprintf("%d Min.", sec/60)
		}
		return fmt.Sprintf("%d:%02d Min.", sec/60, sec%60)
	}
	if sec%3600 == 0 {
		return fmt.Sprintf("%d Std.", sec/3600)
	}
	if sec%60 == 0 {
		return fmt.Sprintf("%d Std. %d Min.", sec/3600, sec%3600/60)
	}
	return formatClock(time.Duration(ms) * time.Millisecond)
}

func renderFocus(m tuiModel, width, height int) string {
	footerText := "Space Pause · +/− 1 Min. · Esc Übersicht · q Ende"
	if m.cliLaunch {
		footerText = "Space Pause · +/− 1 Min. · Esc/q Ende"
	}
	if width < 59 {
		footerText = "Space Pause · Esc Zurück · q Ende"
		if m.cliLaunch {
			footerText = "Space Pause · Esc/q Ende"
		}
	}
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center, mutedStyle.Render(footerText))

	var body string
	if !m.doneSince.IsZero() && !m.cliLaunch {
		body = lipgloss.JoinVertical(lipgloss.Center,
			mutedStyle.Render(shortName(m.doneName, width-6)),
			"",
			warningStyle.Render("F E R T I G"),
			"",
			mutedStyle.Render("Nächster Timer wird übernommen …"),
		)
	} else if t := findTimer(m.snap, m.watchID); t != nil && (t.Status == "running" || t.Status == "paused") {
		rem := remaining(*t, m.serverNow())
		urgent := t.Status == "running" && rem <= 10*time.Second
		clockStyle := titleStyle
		if urgent {
			clockStyle = warningStyle
		}
		name := t.Name
		hint := shortDurationLabel(t.DurationMS)
		if t.Local {
			hint += " · lokal"
		}
		suffix := " (" + hint + ")"
		// Keep the duration muted, without sacrificing space for the clock.
		maxName := max(6, width-6-lipgloss.Width(suffix))
		label := titleStyle.Render(shortName(name, maxName)) + mutedStyle.Render(suffix)
		status := accentStyle.Render("● LÄUFT")
		if t.Status == "paused" {
			status = mutedStyle.Render("Ⅱ PAUSIERT")
		}
		digits := largeClockAdaptive(formatClock(rem), width-4, max(5, height-10))
		if lipgloss.Width(digits) > width-2 {
			digits = formatClock(rem)
		}
		fraction := 0.0
		if t.DurationMS > 0 {
			fraction = 1 - float64(rem.Milliseconds())/float64(t.DurationMS)
		}
		progress := styledProgress(fraction, min(72, max(6, width-8)), urgent)
		body = lipgloss.JoinVertical(lipgloss.Center,
			label,
			status,
			"",
			clockStyle.Render(digits),
			"",
			progress,
		)
	} else {
		body = mutedStyle.Render("Timer wird geladen …")
	}
	return lipgloss.Place(width, max(1, height-1), lipgloss.Center, lipgloss.Center, body) + "\n" + footer
}

func renderOverviewCompact(m tuiModel, width, height int) string {
	usable := max(18, width-2)
	var sections []string
	head := titleStyle.Render("nexttimer") + mutedStyle.Render("  ·  ") + accentStyle.Render(func() string {
		if m.localOnly {
			return "● Nur Client"
		}
		return "● Verbunden"
	}())
	sections = append(sections, head, borderStyle.Render(strings.Repeat("─", min(usable, 90))))
	now := m.serverNow()
	focus := findTimer(m.snap, m.snap.FocusID)
	if focus == nil {
		sections = append(sections, mutedStyle.Render("KEIN LAUFENDER TIMER"),
			titleStyle.Render("Bereit für deinen nächsten Countdown"))
	} else {
		rem := remaining(*focus, now)
		urgent := focus.Status == "running" && rem <= 10*time.Second
		s := titleStyle
		if urgent {
			s = warningStyle
		}
		fraction := 0.0
		if focus.DurationMS > 0 {
			fraction = 1 - float64(rem.Milliseconds())/float64(focus.DurationMS)
		}
		clock := s.Render(formatClock(rem))
		label := mutedStyle.Render("AKTUELL  ·  ") + titleStyle.Render(shortName(focus.Name, max(10, width-20)))
		sections = append(sections, label,
			lipgloss.JoinHorizontal(lipgloss.Center, clock, "  "+accentStyle.Render("● Läuft")),
			styledProgress(fraction, min(60, usable), urgent))
	}
	sections = append(sections, "", titleStyle.Render("FAVORITEN")+tabIndicator(m.section == favoritesSection))
	cols := 1
	if width >= 74 {
		cols = 2
	}
	favRows := max(1, min(5, (height-15)/3))
	pageSize := max(cols, cols*favRows)
	start := 0
	if m.section == favoritesSection && len(m.snap.Templates) > pageSize {
		start = (m.favoriteIndex / pageSize) * pageSize
	}
	if len(m.snap.Templates) == 0 {
		sections = append(sections, mutedStyle.Render("Keine Favoriten · im Web bearbeiten"))
	} else {
		cellWidth := max(12, (usable-cols+1)/cols-2)
		for i := start; i < len(m.snap.Templates) && i < start+pageSize; i += cols {
			var cells []string
			for j := 0; j < cols; j++ {
				if i+j >= len(m.snap.Templates) {
					break
				}
				t := m.snap.Templates[i+j]
				hotkey := "·"
				if i+j < 9 {
					hotkey = fmt.Sprint(i + j + 1)
				} else if i+j == 9 {
					hotkey = "0"
				}
				label := shortName(t.Name, max(4, cellWidth-14))
				if t.BrowserOnly {
					label += " ⌁"
				}
				line := fmt.Sprintf("%s  %s  ·  %s", hotkey, label, t.Duration)
				st := panelStyle.Padding(0, 1).Width(cellWidth)
				if m.section == favoritesSection && i+j == m.favoriteIndex {
					st = selectedStyle.Padding(0, 1).Width(cellWidth)
				}
				cells = append(cells, st.Render(line))
			}
			sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
		}
		if len(m.snap.Templates) > pageSize {
			sections = append(sections, mutedStyle.Render(fmt.Sprintf("  %d–%d von %d Favoriten", start+1, min(start+pageSize, len(m.snap.Templates)), len(m.snap.Templates))))
		}
	}
	sections = append(sections, "", titleStyle.Render("WEITERE TIMER")+tabIndicator(m.section == timersSection))
	other := make([]protocol.TimerView, 0)
	for _, t := range activeTimers(m.snap) {
		if t.ID != m.snap.FocusID {
			other = append(other, t)
		}
	}
	if len(other) == 0 {
		sections = append(sections, mutedStyle.Render("Keine weiteren Timer"))
	}
	maxRows := max(1, min(5, height-len(sections)-3))
	offset := 0
	if m.section == timersSection && m.selectedID != "" {
		for i, t := range other {
			if t.ID == m.selectedID && i >= maxRows {
				offset = i - maxRows + 1
			}
		}
	}
	for i := offset; i < len(other) && i < offset+maxRows; i++ {
		t := other[i]
		selected := m.section == timersSection && m.selectedID == t.ID
		prefix := "  "
		if selected {
			prefix = "› "
		}
		stateLabel := formatClock(remaining(t, now))
		if t.Status == "paused" {
			stateLabel += " · pausiert"
		}
		name := t.Name
		if t.Local {
			name += " (lokal)"
		}
		row := fmt.Sprintf("%s#%s  %s  %s", prefix, t.ID, shortName(name, max(8, width-30)), stateLabel)
		if selected {
			row = selectedStyle.Render(row)
		} else {
			row = mutedStyle.Render(row)
		}
		sections = append(sections, row)
	}
	footer := mutedStyle.Render("1–9 Start · n Neu · Tab Bereich · ↑↓ Auswahl · Enter · ? Hilfe · q Ende")
	if width < 72 {
		footer = mutedStyle.Render("1–9 Start · n Neu · Tab Wechsel · ? Hilfe · q Ende")
	}
	// Keep footer visible on short terminals. Prioritize the timer and favorites.
	if len(sections)+1 > height {
		sections = sections[:max(0, height-1)]
	}
	sections = append(sections, footer)
	if m.message != "" && len(sections)+1 <= height {
		sections = append(sections, warningStyle.Render(shortName(m.message, width-3)))
	}
	return strings.Join(sections, "\n")
}

func tabIndicator(active bool) string {
	if active {
		return accentStyle.Render("  • ausgewählt")
	}
	return ""
}

// renderOverviewPanels uses stable Lazygit-inspired panes: favorites and timers
// on the left, an always-visible live detail pane on the right.
func renderOverviewPanels(m tuiModel, width, height int) string {
	const gap = 1
	leftW := min(43, max(28, width*35/100))
	rightW := width - leftW - gap
	if rightW < 36 {
		return renderOverviewCompact(m, width, height)
	}
	bodyH := max(12, height-3)
	favH := max(8, bodyH*54/100)
	timerH := bodyH - favH
	if timerH < 6 {
		timerH = 6
		favH = bodyH - timerH
	}
	left := lipgloss.JoinVertical(lipgloss.Left,
		panelBox(favoritesPanelLines(m, leftW-4, favH-2), leftW, favH, m.section == favoritesSection),
		panelBox(timersPanelLines(m, leftW-4, timerH-2), leftW, timerH, m.section == timersSection),
	)
	right := panelBox(detailPanelLines(m, rightW-4, bodyH-2), rightW, bodyH, false)
	host := shortName(serverURL, max(12, width-42))
	if m.localOnly {
		host = "ohne Server / AWTRIX"
	}
	header := titleStyle.Render(" nexttimer ") + mutedStyle.Render("  │  ") + accentStyle.Render(func() string {
		if m.localOnly {
			return "● Nur Client"
		}
		return "● Verbunden"
	}()) + mutedStyle.Render("    "+host)
	footer := mutedStyle.Render(" Tab Panel  │  j/k Auswahl  │  1–9 Start  │  Enter Öffnen  │  n Neu  │  Space Pause  │  x Stop  │  ? Hilfe  │  q Beenden")
	if width < 112 {
		footer = mutedStyle.Render(" Tab Wechsel  ·  j/k Auswahl  ·  1–9 Start  ·  Enter  ·  n Neu  ·  ? Hilfe  ·  q Ende")
	}
	if m.message != "" {
		footer = warningStyle.Render(shortName(m.message, width-4))
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Left, header) + "\n" +
		lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Left, footer)
}

func panelBox(lines []string, width, height int, active bool) string {
	innerW := max(8, width-4)
	innerH := max(1, height-2)
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	for i, line := range lines {
		if lipgloss.Width(line) > innerW {
			lines[i] = shortName(line, innerW)
		}
	}
	border := lipgloss.Color("#435267")
	if active {
		border = lipgloss.Color("#72DEB3")
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(0, 1).Width(innerW).Render(strings.Join(lines, "\n"))
}

func favoritesPanelLines(m tuiModel, innerW, innerH int) []string {
	count := len(m.snap.Templates)
	heading := titleStyle.Render("FAVORITEN") + mutedStyle.Render(fmt.Sprintf("  (%d)", count))
	if m.section == favoritesSection {
		heading += accentStyle.Render("  ●")
	}
	lines := []string{heading, mutedStyle.Render(strings.Repeat("─", max(1, innerW-1)))}
	if count == 0 {
		return append(lines, "", mutedStyle.Render("Keine Favoriten"), mutedStyle.Render("Im Browser hinzufügen"))
	}
	maxRows := max(1, innerH-4)
	start := 0
	if m.section == favoritesSection && m.favoriteIndex >= maxRows {
		start = m.favoriteIndex - maxRows + 1
	}
	for i := start; i < count && i < start+maxRows; i++ {
		t := m.snap.Templates[i]
		num := " "
		if i < 9 {
			num = fmt.Sprint(i + 1)
		} else if i == 9 {
			num = "0"
		}
		name := t.Name
		maxName := max(5, innerW-len(t.Duration)-9)
		name = shortName(name, maxName)
		line := fmt.Sprintf(" %s  %-*s %s", num, maxName, name, t.Duration)
		if t.BrowserOnly {
			line = fmt.Sprintf(" %s  %s  %s", num, shortName(name, max(5, innerW-len(t.Duration)-12)), t.Duration) + "  ⌁"
		}
		if m.section == favoritesSection && i == m.favoriteIndex {
			line = selectedStyle.Width(innerW).Render("›" + strings.TrimPrefix(line, " "))
		}
		lines = append(lines, line)
	}
	if count > maxRows {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  %d–%d von %d", start+1, min(count, start+maxRows), count)))
	}
	return lines
}

func timersPanelLines(m tuiModel, innerW, innerH int) []string {
	active := activeTimers(m.snap)
	heading := titleStyle.Render("WEITERE TIMER") + mutedStyle.Render(fmt.Sprintf("  (%d)", len(active)))
	if m.section == timersSection {
		heading += accentStyle.Render("  ●")
	}
	lines := []string{heading, mutedStyle.Render(strings.Repeat("─", max(1, innerW-1)))}
	if len(active) == 0 {
		return append(lines, "", mutedStyle.Render("Keine laufenden Timer"))
	}
	maxRows := max(1, innerH-4)
	start := 0
	if m.section == timersSection && m.selectedID != "" {
		for i, t := range active {
			if t.ID == m.selectedID && i >= maxRows {
				start = i - maxRows + 1
			}
		}
	}
	now := m.serverNow()
	for i := start; i < len(active) && i < start+maxRows; i++ {
		t := active[i]
		clock := formatClock(remaining(t, now))
		if t.Status == "paused" {
			clock = "Ⅱ " + clock
		}
		marker := " "
		if t.ID == m.snap.FocusID {
			marker = "◆"
		}
		n := shortName(t.Name, max(5, innerW-len(clock)-12))
		line := fmt.Sprintf(" %s %-4s %-*s %s", marker, "#"+t.ID, max(5, innerW-len(clock)-12), n, clock)
		if m.section == timersSection && t.ID == m.selectedID {
			line = selectedStyle.Width(innerW).Render("›" + strings.TrimPrefix(line, " "))
		}
		lines = append(lines, line)
	}
	if len(active) > maxRows {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  %d–%d von %d", start+1, min(len(active), start+maxRows), len(active))))
	}
	return lines
}

func detailPanelLines(m tuiModel, innerW, innerH int) []string {
	head := []string{titleStyle.Render("COUNTDOWN") + mutedStyle.Render("  /  DETAILS"), mutedStyle.Render(strings.Repeat("─", max(1, innerW-1)))}
	var t *protocol.TimerView
	if m.section == timersSection && m.selectedID != "" {
		t = findTimer(m.snap, m.selectedID)
	}
	if t == nil {
		t = findTimer(m.snap, m.snap.FocusID)
	}
	if t == nil {
		head = append(head, "", mutedStyle.Render("Kein laufender Timer"), "", accentStyle.Render("Neuen Timer starten: n"), mutedStyle.Render("Favorit starten: 1–9 / Enter"))
		if len(m.snap.Templates) > 0 && m.section == favoritesSection {
			fav := m.snap.Templates[min(m.favoriteIndex, len(m.snap.Templates)-1)]
			head = append(head, "", mutedStyle.Render("AUSGEWÄHLTER FAVORIT"), titleStyle.Render(shortName(fav.Name, innerW-2)), accentStyle.Render("Dauer: "+fav.Duration))
		}
		return head
	}
	rem := remaining(*t, m.serverNow())
	urgent := t.Status == "running" && rem <= 10*time.Second
	st := accentStyle
	if urgent {
		st = warningStyle
	}
	status := "● LÄUFT"
	if t.Status == "paused" {
		status = "Ⅱ PAUSIERT"
		st = mutedStyle
	}
	if urgent {
		status = "● LETZTE 10 SEKUNDEN"
	}
	label := t.Name
	if t.Local {
		label += " · LOKAL"
	}
	head = append(head, "", titleStyle.Render(shortName(label, innerW-2)), st.Render(status), "")
	clock := formatClock(rem)
	big := largeClockAdaptive(clock, innerW-3, min(9, max(5, innerH-14)))
	if lipgloss.Width(big) > innerW {
		big = clock
	}
	for _, line := range strings.Split(st.Render(big), "\n") {
		head = append(head, lipgloss.PlaceHorizontal(innerW, lipgloss.Center, line))
	}
	frac := 0.0
	if t.DurationMS > 0 {
		frac = 1 - float64(rem.Milliseconds())/float64(t.DurationMS)
	}
	head = append(head, "", styledProgress(frac, max(6, min(innerW-2, 60)), urgent), "")
	head = append(head, mutedStyle.Render("ID #"+t.ID+"   │   Gesamt "+formatClock(time.Duration(t.DurationMS)*time.Millisecond)))
	head = append(head, "", mutedStyle.Render("Space Pause/Weiter   +/− 1 Min"), mutedStyle.Render("Enter / w Vollbild     x Stoppen"))
	return head
}

func renderOverview(m tuiModel, width, height int) string {
	if width < 72 || height < 19 {
		return renderOverviewCompact(m, width, height)
	}
	return renderOverviewPanels(m, width, height)
}
