package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

var client = &http.Client{Timeout: 4 * time.Second}

func SetServerURL(raw string) { serverURL = strings.TrimRight(raw, "/") }

var serverURL = strings.TrimRight(env("TIMER_URL", "http://127.0.0.1:8080"), "/")

func request(method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, serverURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Service nicht erreichbar (%s): %w", serverURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return errors.New(e.Error)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func state() (protocol.Snapshot, error) {
	var s protocol.Snapshot
	err := request("GET", "/api/state", nil, &s)
	return s, err
}
func mainTimer(s protocol.Snapshot, selector string, allowPaused bool) (protocol.TimerView, error) {
	// Optional row shortcut: #1 selects the first row in `nexttimer ls`.
	if strings.HasPrefix(selector, "#") {
		number, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
		if err != nil || number < 1 || number > len(s.Timers) {
			return protocol.TimerView{}, errors.New("ungültige Listenposition")
		}
		return s.Timers[number-1], nil
	}
	if selector == "" || selector == "focus" {
		for _, t := range s.Timers {
			if t.ID == s.FocusID {
				return t, nil
			}
		}
		if allowPaused {
			for _, t := range s.Timers {
				if t.Status == "paused" {
					return t, nil
				}
			}
		}
		return protocol.TimerView{}, errors.New("kein laufender Haupttimer; bitte Timer-ID angeben")
	}
	var match *protocol.TimerView
	for i := range s.Timers {
		t := &s.Timers[i]
		if t.ID == selector {
			return *t, nil
		}
		if strings.EqualFold(t.Name, selector) || (selector != "" && strings.HasPrefix(strings.ToLower(t.ID), strings.ToLower(selector))) {
			if match != nil {
				return protocol.TimerView{}, errors.New("mehrdeutiger Timername; bitte ID angeben")
			}
			match = t
		}
	}
	if match != nil {
		return *match, nil
	}
	return protocol.TimerView{}, errors.New("Timer nicht gefunden")
}
func Run(args []string) error { return RunWithClientOptions(args, Options{}) }

type Options struct {
	Detach      bool // start server timer and exit without opening a TUI
	Local       bool // timer exists only while this client process runs
	KeepRunning bool // when an attached CLI session ends, leave the server timer running
}

func RunWithClientOptions(args []string, opt Options) error {
	if opt.Local && opt.Detach {
		return errors.New("--local und --detach sind unvereinbar: lokale Timer benötigen eine laufende TUI")
	}
	if opt.Local && opt.KeepRunning {
		return errors.New("--keep-running ist nur für Server-Timer möglich")
	}
	if opt.Detach && opt.KeepRunning {
		return errors.New("--detach lässt Timer bereits weiterlaufen; --keep-running ist hier unnötig")
	}
	if opt.Local && !isTerminal() {
		return errors.New("--local benötigt ein interaktives Terminal; ein lokaler Timer würde beim Beenden sofort verschwinden")
	}

	if len(args) == 0 {
		if opt.Detach || opt.KeepRunning {
			return errors.New("--detach / --keep-running benötigen einen direkt gestarteten Timer (Dauer oder 'start Favorit')")
		}
		if !isTerminal() {
			usage()
			return nil
		}
		if opt.Local {
			return runTUILocal(0, "", false)
		}
		return runTUI()
	}
	command := args[0]
	// These flags apply exclusively to timers created by this command, not to
	// the interactive dashboard, watch or unrelated server commands.
	isStart := command == "run" || command == "start"
	if !isStart {
		_, err := parseDuration(command)
		isStart = err == nil
	}
	if !isStart && (opt.Detach || opt.KeepRunning) {
		return errors.New("--detach und --keep-running gelten nur für den direkten Timerstart")
	}
	switch command {
	case "help", "-h", "--help":
		usage()
		return nil
	case "ls", "list":
		s, err := state()
		if err != nil {
			return err
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "#\tFOKUS\tID\tNAME\tREST\tSTATUS")
		for idx, t := range s.Timers {
			focus := ""
			if t.ID == s.FocusID {
				focus = "▶"
			}
			rem := remaining(t, time.UnixMilli(s.ServerNowMS))
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n", idx+1, focus, t.ID, t.Name, formatClock(rem), t.Status)
		}
		return tw.Flush()
	case "watch":
		selector, compact, err := watchArgs(args[1:])
		if err != nil {
			return err
		}
		if !compact && isTerminal() {
			snap, err := state()
			if err != nil {
				return err
			}
			chosen, err := mainTimer(snap, selector, false)
			if err != nil {
				return err
			}
			return runTUIWithID(chosen.ID)
		}
		return watch(selector, compact)
	case "run":
		if len(args) < 2 {
			return errors.New("Syntax: nexttimer run <Dauer> [Name]")
		}
		duration, err := parseDuration(args[1])
		if err != nil {
			return err
		}
		name := strings.Join(args[2:], " ")
		if opt.Local {
			return runTUILocalAttached(duration, name)
		}
		if !opt.Detach && !isTerminal() {
			return errors.New("ohne interaktives Terminal bitte --detach verwenden")
		}
		var t protocol.TimerView
		if err := request("POST", "/api/timers", map[string]string{"name": name, "duration": duration.String()}, &t); err != nil {
			return err
		}
		fmt.Printf("Gestartet: %s (%s), ID %s\n", t.Name, formatClock(duration), t.ID)
		if opt.Detach {
			return nil
		}
		return runTUIAttached(t.ID, opt.KeepRunning)
	case "start":
		if len(args) < 2 {
			return errors.New("Syntax: timer start <Favoritenname oder ID>")
		}
		s, err := state()
		if err != nil {
			return err
		}
		name := strings.Join(args[1:], " ")
		chosen, err := favoriteBySelector(s.Templates, name)
		if err != nil {
			return err
		}
		if opt.Local || chosen.BrowserOnly {
			if opt.KeepRunning {
				return errors.New("lokaler Favorit kann nicht mit --keep-running gestartet werden: er endet mit dem Client")
			}
			if opt.Detach {
				return errors.New("lokaler Favorit kann nicht mit --detach gestartet werden: die TUI muss geöffnet bleiben")
			}
			if !isTerminal() {
				return errors.New("lokaler Favorit benötigt ein interaktives Terminal")
			}
			duration, err := parseDuration(chosen.Duration)
			if err != nil {
				return err
			}
			return runTUILocalAttached(duration, chosen.Name)
		}
		if !opt.Detach && !isTerminal() {
			return errors.New("ohne interaktives Terminal bitte --detach verwenden")
		}
		var t protocol.TimerView
		if err = request("POST", "/api/timers", map[string]string{"template_id": chosen.ID}, &t); err != nil {
			return err
		}
		fmt.Printf("Gestartet: %s (%s), ID %s\n", t.Name, formatClock(time.Duration(t.DurationMS)*time.Millisecond), t.ID)
		if opt.Detach {
			return nil
		}
		return runTUIAttached(t.ID, opt.KeepRunning)
	case "pause", "resume", "stop":
		if len(args) > 2 {
			return errors.New("Zu viele Argumente")
		}
		selector := ""
		if len(args) == 2 {
			selector = args[1]
		}
		s, err := state()
		if err != nil {
			return err
		}
		t, err := mainTimer(s, selector, command == "resume")
		if err != nil {
			return err
		}
		if err = request("POST", "/api/timers/"+url.PathEscape(t.ID)+"/"+command, nil, nil); err != nil {
			return err
		}
		fmt.Printf("%s: %s\n", command, t.Name)
		return nil
	case "add", "adjust", "+":
		if len(args) < 2 || len(args) > 3 {
			return errors.New("Syntax: timer add +5m [ID] (oder -2m)")
		}
		selector := ""
		if len(args) == 3 {
			selector = args[2]
		}
		return adjust(args[1], selector)
	case "favorite", "favorites":
		return favorite(args[1:])
	}
	if strings.HasPrefix(command, "+") || strings.HasPrefix(command, "-") {
		selector := ""
		if len(args) > 2 {
			return errors.New("Zu viele Argumente")
		}
		if len(args) > 1 {
			selector = args[1]
		}
		return adjust(command, selector)
	}
	duration, err := parseDuration(command)
	if err != nil {
		usage()
		return fmt.Errorf("unbekannter Befehl: %s", command)
	}
	name := strings.Join(args[1:], " ")
	if opt.Local {
		return runTUILocalAttached(duration, name)
	}
	if !opt.Detach && !isTerminal() {
		return errors.New("ohne interaktives Terminal bitte --detach verwenden")
	}
	var t protocol.TimerView
	if err = request("POST", "/api/timers", map[string]string{"name": name, "duration": duration.String()}, &t); err != nil {
		return err
	}
	fmt.Printf("Gestartet: %s (%s), ID %s\n", t.Name, formatClock(duration), t.ID)
	if opt.Detach {
		return nil
	}
	return runTUIAttached(t.ID, opt.KeepRunning)
}
func remaining(t protocol.TimerView, now time.Time) time.Duration {
	if t.Status == "running" {
		d := time.UnixMilli(t.DeadlineMS).Sub(now)
		if d < 0 {
			return 0
		}
		return d
	}
	return time.Duration(t.RemainingMS) * time.Millisecond
}
func adjust(value, selector string) error {
	s, err := state()
	if err != nil {
		return err
	}
	t, err := mainTimer(s, selector, false)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(value, "-") && !strings.HasPrefix(value, "+") {
		value = "+" + value
	}
	if err = request("POST", "/api/timers/"+url.PathEscape(t.ID)+"/adjust", map[string]string{"delta": value}, nil); err != nil {
		return err
	}
	fmt.Printf("Zeitänderung %s: %s\n", t.Name, value)
	return nil
}
func favorite(args []string) error {
	if len(args) == 0 || args[0] == "ls" || args[0] == "list" {
		s, err := state()
		if err != nil {
			return err
		}
		for i, t := range s.Templates {
			group := t.Group
			if group == "" {
				group = "Standard"
			}
			fmt.Printf("#%-3d %-22s %-10s %-16s %s\n", i+1, t.Name, t.Duration, group, t.ID)
		}
		return nil
	}
	switch args[0] {
	case "add":
		if len(args) < 3 {
			return errors.New("Syntax: timer favorite add <Dauer> <Name>")
		}
		name := strings.Join(args[2:], " ")
		var t protocol.Template
		if err := request("POST", "/api/templates", map[string]string{"duration": args[1], "name": name}, &t); err != nil {
			return err
		}
		fmt.Printf("Favorit gespeichert: %s (%s)\n", t.Name, t.Duration)
		return nil
	case "rm", "remove", "delete":
		if len(args) < 2 {
			return errors.New("Syntax: timer favorite rm <Name oder ID>")
		}
		s, err := state()
		if err != nil {
			return err
		}
		name := strings.Join(args[1:], " ")
		t, err := favoriteBySelector(s.Templates, name)
		if err != nil {
			return err
		}
		if err := request("DELETE", "/api/templates/"+url.PathEscape(t.ID), nil, nil); err != nil {
			return err
		}
		fmt.Printf("Favorit entfernt: %s\n", t.Name)
		return nil
	}
	return errors.New("Unbekannter Favoriten-Befehl")
}
func usage() {
	fmt.Print(`NextTimer – zentraler Countdown-Service

Service:
  nexttimer-server [--listen 127.0.0.1:8080] [--awtrix http://IP]
  nexttimer-server service install|uninstall|status
  nexttimer-server config path|show|check|init

CLI (TIMER_URL=http://HOST:8080 oder nexttimer --url http://HOST:8080):
  nexttimer config set-url http://HOST:8080   Server-Adresse speichern (XDG)
  nexttimer config url                       Aktive Server-Adresse anzeigen
  nexttimer --detach 10m Kaffee              Auf Server starten und sofort beenden
  nexttimer --detach start Kaffee            Favorit auf Server starten, ohne TUI
  nexttimer --keep-running 10m Kaffee        Fokusmodus; bei Verlassen weiterlaufen lassen
  nexttimer --local 10m Kaffee               Lokalen Timer ohne Server starten (TUI)
  nexttimer --local start Kaffee             Favorit lokal im Client starten
  nexttimer --local --detach ...             Nicht möglich: Client müsste weiterlaufen
  nexttimer 10m [Name]                 Fokusmodus; bei Verlassen Timer stoppen (Standard)
  nexttimer                            Interaktives Terminal (Bubble Tea)
  nexttimer ls                         Alle Timer und Haupttimer
  nexttimer watch [ID]                 Termdown-ähnliche Live-Anzeige
  nexttimer watch --compact            Kompakter Einzeiler
  nexttimer run 10m [Name]             Wie direkter Timerstart (mit Fokusmodus)
  nexttimer pause [ID|#1]              Pausieren
  nexttimer resume [ID]                Fortsetzen
  nexttimer stop [ID]                  Entfernen
  nexttimer +5m [ID]                   Zeit verlängern
  nexttimer -2m [ID]                   Zeit verkürzen
  nexttimer favorite add 25m Pomodoro  Vorlage speichern
  nexttimer favorites                  Vorlagen anzeigen
  nexttimer start Pomodoro             Vorlage starten + Fokusmodus (mit Auto-Stop)
  TUI: n, !5m Kaffee                  Lokalen Timer ohne Server erstellen
  TUI: n, 5m Kaffee, Alt+Enter         Timer starten und in Übersicht bleiben
  nexttimer favorite rm Pomodoro       Vorlage löschen

Ein direkt per CLI gestarteter Timer wird bei Esc, q oder Ctrl+C standardmäßig
beendet. Mit --keep-running läuft er beim Verlassen weiter; --detach öffnet
überhaupt keine TUI. Ein natürlich abgelaufener Timer wird nicht erneut gestoppt.
`)
}
