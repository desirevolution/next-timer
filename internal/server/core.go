package server

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/desirevolution/next-timer/internal/protocol"
)

type Timer struct {
	ID         string
	Name       string
	Duration   time.Duration
	Deadline   time.Time
	Remaining  time.Duration
	Status     string
	CreatedAt  time.Time
	FinishedAt time.Time
	Sound      bool
	Notify     bool
	Alarm      *protocol.BrowserAlarm
}

type TimerOptions struct {
	Sound  bool
	Notify bool
	Alarm  *protocol.BrowserAlarm
}

func enabled(flag *bool) bool { return flag == nil || *flag }

func validateBrowserAlarm(a *protocol.BrowserAlarm) error {
	if a == nil {
		return nil
	}
	if a.Volume != nil && (math.IsNaN(*a.Volume) || math.IsInf(*a.Volume, 0) || *a.Volume < 0 || *a.Volume > 1) {
		return errors.New("Ungültige Lautstärke (0–1)")
	}
	switch a.Tone {
	case "none", "gong", "bell", "digital", "alert":
	default:
		return errors.New("Ungültiger Alarmton")
	}
	switch a.Repeat {
	case "once", "three", "until_ack":
	default:
		return errors.New("Ungültige Alarmwiederholung")
	}
	return nil
}

// The empty group is the implicit default group; other names are case-insensitively unique.
func cleanGroup(group string) (string, error) {
	group = strings.TrimSpace(group)
	if strings.EqualFold(group, "Standard") {
		return "", nil
	}
	if len([]rune(group)) > 48 {
		return "", errors.New("Gruppenname zu lang (max. 48 Zeichen)")
	}
	for _, ch := range group {
		if ch < 32 || ch == 127 {
			return "", errors.New("Ungültiger Gruppenname")
		}
	}
	return group, nil
}

func sameGroup(a, b string) bool { return strings.EqualFold(a, b) }

// Persist creation order as metadata on member templates. Groups have no independent
// lifecycle: removing their last member automatically removes them.
func (s *Service) groupInfoLocked(name string) (string, int, bool) {
	for _, t := range s.templates {
		if name != "" && sameGroup(t.Group, name) {
			return t.Group, t.GroupOrder, true
		}
	}
	return "", 0, false
}
func (s *Service) newGroupOrderLocked() int {
	order := 0
	for _, t := range s.templates {
		if t.GroupOrder > order {
			order = t.GroupOrder
		}
	}
	return order + 1
}
func orderGroups(list []Template) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Group == "" {
			return false
		}
		if b.Group == "" {
			return true
		}
		return a.GroupOrder < b.GroupOrder
	})
}
func groupNames(list []Template) []string {
	names := []string{}
	seen := map[string]bool{}
	for _, t := range list {
		key := strings.ToLower(t.Group)
		if t.Group != "" && !seen[key] {
			seen[key] = true
			names = append(names, t.Group)
		}
	}
	return names
}

type TimerView = protocol.TimerView
type Template = protocol.Template
type Snapshot = protocol.Snapshot
type Event = protocol.Event

type Service struct {
	mu           sync.Mutex
	timers       map[string]*Timer
	templates    []Template
	templatePath string
	finishedTTL  time.Duration
	subscribers  map[chan Event]struct{}
}

var nextID atomic.Uint64

func id(prefix string) string {
	if prefix == "t" {
		return strings.ToLower(fmt.Sprintf("%s", strconv.FormatUint(nextID.Add(1), 36)))
	}
	var bytes [5]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return "f-" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes[:]))
}

func NewService(path string, ttl time.Duration) (*Service, error) {
	s := &Service{timers: make(map[string]*Timer), templatePath: path, finishedTTL: ttl, subscribers: make(map[chan Event]struct{}), templates: []Template{}}
	data, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &s.templates); err != nil {
			return nil, fmt.Errorf("Vorlagen nicht lesbar: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	groupPositions := map[string]int{}
	maxPosition := 0
	for _, t := range s.templates {
		if t.GroupOrder > maxPosition {
			maxPosition = t.GroupOrder
		}
	}
	for i := range s.templates {
		t := &s.templates[i]
		group, err := cleanGroup(t.Group)
		if err != nil {
			return nil, fmt.Errorf("Vorlage %s: %w", t.Name, err)
		}
		t.Group = group
		if group != "" {
			key := strings.ToLower(group)
			if position, ok := groupPositions[key]; ok {
				t.GroupOrder = position
			} else {
				if t.GroupOrder == 0 {
					maxPosition++
					t.GroupOrder = maxPosition
				}
				groupPositions[key] = t.GroupOrder
			}
		} else {
			t.GroupOrder = 0
		}
	}
	orderGroups(s.templates)
	for _, t := range s.templates {
		if t.ID == "" || strings.TrimSpace(t.Name) == "" {
			return nil, errors.New("ungültige Vorlagen-Datei")
		}
		if err := validateBrowserAlarm(t.Alarm); err != nil {
			return nil, fmt.Errorf("Vorlage %s: %w", t.Name, err)
		}
		if _, err := parseDuration(t.Duration); err != nil {
			return nil, fmt.Errorf("ungültige Vorlage %s: %w", t.Name, err)
		}
	}
	return s, nil
}

func parseDuration(input string) (time.Duration, error) {
	input = strings.TrimSpace(input)
	d, err := time.ParseDuration(input)
	if err != nil || d < time.Second || d > 7*24*time.Hour {
		return 0, errors.New("Dauer muss zwischen 1s und 168h liegen (z.B. 10m, 1h30m)")
	}
	return d, nil
}

func (t *Timer) view(now time.Time) TimerView {
	remain := t.Remaining
	if t.Status == "running" {
		remain = t.Deadline.Sub(now)
	}
	if remain < 0 {
		remain = 0
	}
	v := TimerView{ID: t.ID, Name: t.Name, DurationMS: t.Duration.Milliseconds(), RemainingMS: remain.Milliseconds(), Status: t.Status, CreatedAtMS: t.CreatedAt.UnixMilli()}
	if t.Status == "running" {
		v.DeadlineMS = t.Deadline.UnixMilli()
	}
	v.Sound, v.Notify, v.Alarm = t.Sound, t.Notify, t.Alarm
	if !t.FinishedAt.IsZero() {
		v.FinishedAtMS = t.FinishedAt.UnixMilli()
	}
	return v
}

func (s *Service) focusLocked() *Timer {
	var current *Timer
	for _, t := range s.timers {
		if t.Status != "running" {
			continue
		}
		if current == nil || t.Deadline.Before(current.Deadline) || (t.Deadline.Equal(current.Deadline) && (t.CreatedAt.Before(current.CreatedAt) || (t.CreatedAt.Equal(current.CreatedAt) && t.ID < current.ID))) {
			current = t
		}
	}
	return current
}

func (s *Service) snapshotLocked(now time.Time) Snapshot {
	out := Snapshot{ServerNowMS: now.UnixMilli(), Timers: make([]TimerView, 0, len(s.timers)), Templates: append([]Template{}, s.templates...), Groups: groupNames(s.templates)}
	if f := s.focusLocked(); f != nil {
		out.FocusID = f.ID
	}
	for _, t := range s.timers {
		out.Timers = append(out.Timers, t.view(now))
	}
	rank := func(status string) int {
		switch status {
		case "running":
			return 0
		case "paused":
			return 1
		default:
			return 2
		}
	}
	sort.Slice(out.Timers, func(i, j int) bool {
		a, b := out.Timers[i], out.Timers[j]
		if rank(a.Status) != rank(b.Status) {
			return rank(a.Status) < rank(b.Status)
		}
		if a.Status == "running" && a.DeadlineMS != b.DeadlineMS {
			return a.DeadlineMS < b.DeadlineMS
		}
		return a.CreatedAtMS > b.CreatedAtMS
	})
	// The stored slice is the user-defined order; do not sort by name.
	return out
}

func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(time.Now())
}

func (s *Service) publishLocked(typ string, t *Timer, now time.Time) {
	ev := Event{Type: typ, State: s.snapshotLocked(now)}
	if t != nil {
		v := t.view(now)
		ev.Timer = &v
	}
	for c := range s.subscribers {
		select {
		case c <- ev:
		default:
			select {
			case <-c:
			default:
			}
			select {
			case c <- ev:
			default:
			}
		}
	}
}
func (s *Service) Subscribe() (chan Event, func()) {
	c := make(chan Event, 32)
	s.mu.Lock()
	s.subscribers[c] = struct{}{}
	c <- Event{Type: "snapshot", State: s.snapshotLocked(time.Now())}
	s.mu.Unlock()
	return c, func() { s.mu.Lock(); delete(s.subscribers, c); close(c); s.mu.Unlock() }
}

func (s *Service) Start(name string, duration time.Duration) *TimerView {
	return s.StartConfigured(name, duration, TimerOptions{Sound: true, Notify: true})
}
func (s *Service) StartConfigured(name string, duration time.Duration, options TimerOptions) *TimerView {
	now := time.Now()
	if strings.TrimSpace(name) == "" {
		name = "Timer"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Timer{ID: id("t"), Name: strings.TrimSpace(name), Duration: duration, Deadline: now.Add(duration), Remaining: duration, Status: "running", CreatedAt: now, Sound: options.Sound, Notify: options.Notify, Alarm: options.Alarm}
	s.timers[t.ID] = t
	s.publishLocked("started", t, now)
	v := t.view(now)
	return &v
}
func (s *Service) StartTemplate(templateID string) (*TimerView, error) {
	s.mu.Lock()
	var selected *Template
	matches := 0
	// IDs always win; names are only accepted when they identify a single favorite.
	for i := range s.templates {
		t := &s.templates[i]
		if t.ID == templateID {
			copy := *t
			selected = &copy
			matches = 1
			break
		}
	}
	if selected == nil {
		for i := range s.templates {
			t := &s.templates[i]
			if strings.EqualFold(t.Name, templateID) {
				copy := *t
				selected = &copy
				matches++
			}
		}
	}
	s.mu.Unlock()
	if matches > 1 {
		return nil, errors.New("Favoritenname ist mehrdeutig; bitte Favoriten-ID verwenden")
	}
	if selected == nil {
		return nil, errors.New("Vorlage nicht gefunden")
	}
	if selected.BrowserOnly {
		return nil, errors.New("Dieser Favorit läuft nur im Browser und kann nicht über den Server oder die CLI gestartet werden")
	}
	d, _ := parseDuration(selected.Duration)
	return s.StartConfigured(selected.Name, d, TimerOptions{Sound: enabled(selected.Sound), Notify: enabled(selected.Notify), Alarm: selected.Alarm}), nil
}
func (s *Service) Control(timerID, action string, delta time.Duration) (*TimerView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	t := s.timers[timerID]
	if t == nil {
		return nil, errors.New("Timer nicht gefunden")
	}
	// The deadline is authoritative: a timer cannot be paused after it expired.
	if t.Status == "running" && !t.Deadline.After(now) {
		s.finishLocked(t, now)
	}
	switch action {
	case "pause":
		if t.Status != "running" {
			return nil, errors.New("Timer läuft nicht")
		}
		t.Remaining = t.Deadline.Sub(now)
		t.Status = "paused"
	case "resume":
		if t.Status != "paused" {
			return nil, errors.New("Timer ist nicht pausiert")
		}
		t.Deadline = now.Add(t.Remaining)
		t.Status = "running"
	case "adjust":
		if t.Status != "running" && t.Status != "paused" {
			return nil, errors.New("Timer ist bereits abgelaufen")
		}
		remain := t.Remaining
		if t.Status == "running" {
			remain = t.Deadline.Sub(now)
		}
		if remain+delta <= 0 {
			s.finishLocked(t, now)
			v := t.view(now)
			return &v, nil
		}
		if remain+delta > 7*24*time.Hour {
			return nil, errors.New("Maximale Restzeit: 168h")
		}
		t.Remaining = remain + delta
		// Preserve elapsed time while adjusting the total duration.
		// The progress fraction must remain correct even if remaining > original duration.
		t.Duration += delta
		if t.Status == "running" {
			t.Deadline = now.Add(t.Remaining)
		}
	case "stop":
		delete(s.timers, timerID)
		s.publishLocked("stopped", t, now)
		v := t.view(now)
		return &v, nil
	default:
		return nil, errors.New("Ungültige Aktion")
	}
	eventName := map[string]string{"pause": "paused", "resume": "resumed", "adjust": "adjusted"}[action]
	s.publishLocked(eventName, t, now)
	v := t.view(now)
	return &v, nil
}
func (s *Service) finishLocked(t *Timer, now time.Time) {
	t.Status = "finished"
	t.Remaining = 0
	t.FinishedAt = now
	s.publishLocked("finished", t, now)
}
func (s *Service) Tick() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.timers {
		if t.Status == "running" && !t.Deadline.After(now) {
			s.finishLocked(t, now)
		}
	}
	removed := false
	for id, t := range s.timers {
		if t.Status == "finished" && now.Sub(t.FinishedAt) >= s.finishedTTL {
			delete(s.timers, id)
			removed = true
		}
	}
	if removed {
		s.publishLocked("cleanup", nil, now)
	}
}
func (s *Service) saveTemplatesLocked(list []Template) error {
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	path := s.templatePath
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".templates-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(append(b, '\n'))
	}
	if err == nil {
		err = tmp.Sync()
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	s.templates = list
	return nil
}
func (s *Service) AddTemplate(name, duration string) (Template, error) {
	return s.AddTemplateConfigured(Template{Name: name, Duration: duration})
}
func (s *Service) AddTemplateConfigured(opts Template) (Template, error) {
	if err := validateBrowserAlarm(opts.Alarm); err != nil {
		return Template{}, err
	}
	name, duration := opts.Name, opts.Duration
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return Template{}, errors.New("Vorlagenname muss 1-80 Zeichen haben")
	}
	d, err := parseDuration(duration)
	if err != nil {
		return Template{}, err
	}
	group, err := cleanGroup(opts.Group)
	if err != nil {
		return Template{}, err
	}
	for _, t := range s.templates {
		if strings.EqualFold(t.Name, name) && sameGroup(t.Group, group) {
			return Template{}, errors.New("Name in dieser Gruppe bereits vergeben")
		}
	}
	order := 0
	if group != "" {
		if existing, position, ok := s.groupInfoLocked(group); ok {
			group, order = existing, position
		} else {
			order = s.newGroupOrderLocked()
		}
	}
	item := Template{ID: id("f"), Name: name, Duration: d.String(), Group: group, GroupOrder: order, BrowserOnly: opts.BrowserOnly, Sound: opts.Sound, Notify: opts.Notify, Alarm: opts.Alarm}
	list := append(append([]Template{}, s.templates...), item)
	orderGroups(list)
	if err = s.saveTemplatesLocked(list); err != nil {
		return Template{}, err
	}
	s.publishLocked("template_added", nil, time.Now())
	return item, nil
}

// ReorderTemplates persists the visual order for every client. An exact ID set is
// required so that a stale browser cannot accidentally erase newly added favorites.
func (s *Service) ReorderTemplates(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) != len(s.templates) {
		return errors.New("Favoriten wurden verändert; bitte Ansicht aktualisieren")
	}
	byID := make(map[string]Template, len(s.templates))
	for _, t := range s.templates {
		byID[t.ID] = t
	}
	ordered := make([]Template, 0, len(ids))
	for _, id := range ids {
		t, found := byID[id]
		if !found {
			return errors.New("Unbekannter oder doppelter Favorit")
		}
		ordered = append(ordered, t)
		delete(byID, id)
	}
	if len(byID) != 0 {
		return errors.New("Nicht alle Favoriten enthalten")
	}
	// Reject attempts to move a favorite outside its group or reorder group blocks.
	check := append([]Template{}, ordered...)
	orderGroups(check)
	for i := range ordered {
		if ordered[i].ID != check[i].ID {
			return errors.New("Favoriten können nur innerhalb ihrer Gruppe sortiert werden")
		}
	}
	if err := s.saveTemplatesLocked(ordered); err != nil {
		return err
	}
	s.publishLocked("templates_reordered", nil, time.Now())
	return nil
}

// ReorderGroups stores the order of non-default groups by updating the
// group_order metadata of every member. The implicit default group stays last.
// IDs of favorites and their order within each group are left unchanged.
func (s *Service) ReorderGroups(names []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := groupNames(s.templates)
	if len(names) != len(current) {
		return errors.New("Gruppen wurden verändert; bitte Ansicht aktualisieren")
	}
	canonical := make(map[string]string, len(current))
	for _, name := range current {
		canonical[strings.ToLower(name)] = name
	}
	positions := make(map[string]int, len(names))
	for i, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if _, ok := canonical[key]; !ok {
			return errors.New("Unbekannte Gruppe")
		}
		if _, exists := positions[key]; exists {
			return errors.New("Doppelte Gruppe")
		}
		positions[key] = i + 1
	}
	list := append([]Template{}, s.templates...)
	for i := range list {
		if list[i].Group != "" {
			list[i].GroupOrder = positions[strings.ToLower(list[i].Group)]
		}
	}
	orderGroups(list)
	if err := s.saveTemplatesLocked(list); err != nil {
		return err
	}
	s.publishLocked("groups_reordered", nil, time.Now())
	return nil
}

func (s *Service) UpdateTemplate(id, name, duration string) (Template, error) {
	return s.UpdateTemplateConfigured(id, Template{Name: name, Duration: duration})
}
func (s *Service) UpdateTemplateConfigured(id string, opts Template) (Template, error) {
	if err := validateBrowserAlarm(opts.Alarm); err != nil {
		return Template{}, err
	}
	name, duration := opts.Name, opts.Duration
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return Template{}, errors.New("Vorlagenname muss 1-80 Zeichen haben")
	}
	d, err := parseDuration(duration)
	if err != nil {
		return Template{}, err
	}
	group, err := cleanGroup(opts.Group)
	if err != nil {
		return Template{}, err
	}
	list := append([]Template{}, s.templates...)
	index := -1
	for i, t := range list {
		if t.ID == id {
			index = i
		} else if strings.EqualFold(t.Name, name) && sameGroup(t.Group, group) {
			return Template{}, errors.New("Name in dieser Gruppe bereits vergeben")
		}
	}
	if index < 0 {
		return Template{}, errors.New("Vorlage nicht gefunden")
	}
	list[index].Name = name
	list[index].Duration = d.String()
	oldGroup := list[index].Group
	if !sameGroup(group, oldGroup) {
		moved := list[index]
		list = append(list[:index], list[index+1:]...)
		order := 0
		if group != "" {
			if existing, position, ok := s.groupInfoLocked(group); ok {
				group, order = existing, position
			} else {
				order = s.newGroupOrderLocked()
			}
		}
		moved.Group, moved.GroupOrder = group, order
		list = append(list, moved)
		index = len(list) - 1
	}
	list[index].BrowserOnly = opts.BrowserOnly
	list[index].Sound = opts.Sound
	list[index].Notify = opts.Notify
	list[index].Alarm = opts.Alarm
	orderGroups(list)
	if err := s.saveTemplatesLocked(list); err != nil {
		return Template{}, err
	}
	s.publishLocked("template_updated", nil, time.Now())
	for _, updated := range list {
		if updated.ID == id {
			return updated, nil
		}
	}
	return Template{}, errors.New("Vorlage nach Aktualisierung nicht gefunden")
}

func (s *Service) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]Template, 0, len(s.templates))
	found := false
	for _, t := range s.templates {
		if t.ID == id {
			found = true
		} else {
			list = append(list, t)
		}
	}
	if !found {
		return errors.New("Vorlage nicht gefunden")
	}
	if err := s.saveTemplatesLocked(list); err != nil {
		return err
	}
	s.publishLocked("template_deleted", nil, time.Now())
	return nil
}

// RenameGroup updates all affected templates while retaining their creation order.
// Empty/default groups cannot be renamed; groups automatically vanish when empty.
func (s *Service) RenameGroup(oldName, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldName = strings.TrimSpace(oldName)
	newName, err := cleanGroup(newName)
	if err != nil {
		return err
	}
	if oldName == "" || strings.EqualFold(oldName, "Standard") || newName == "" {
		return errors.New("Die Standardgruppe kann nicht umbenannt werden")
	}
	found := false
	for _, t := range s.templates {
		if sameGroup(t.Group, oldName) {
			found = true
		} else if sameGroup(t.Group, newName) {
			return errors.New("Gruppenname bereits vergeben")
		}
	}
	if !found {
		return errors.New("Gruppe nicht gefunden")
	}
	list := append([]Template{}, s.templates...)
	for i := range list {
		if sameGroup(list[i].Group, oldName) {
			list[i].Group = newName
		}
	}
	if err := s.saveTemplatesLocked(list); err != nil {
		return err
	}
	s.publishLocked("group_renamed", nil, time.Now())
	return nil
}
