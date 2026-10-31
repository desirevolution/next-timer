package protocol

// TimerView is the public, transport-safe view of a countdown timer.
// BrowserAlarm is optional. Nil means the browser should apply its own defaults.
// It changes browser signalling only, never AWTRIX or server webhooks.
type BrowserAlarm struct {
	Tone   string `json:"tone"`
	Repeat string `json:"repeat"`
	Visual bool   `json:"visual"`
	System bool   `json:"system"`
	// Optional: nil inherits the browser-wide volume. Zero means muted.
	Volume *float64 `json:"volume,omitempty"`
}

type TimerView struct {
	ID           string        `json:"id"`
	Local        bool          `json:"local,omitempty"`
	Name         string        `json:"name"`
	DurationMS   int64         `json:"duration_ms"`
	RemainingMS  int64         `json:"remaining_ms"`
	DeadlineMS   int64         `json:"deadline_ms,omitempty"`
	Status       string        `json:"status"`
	CreatedAtMS  int64         `json:"created_at_ms"`
	FinishedAtMS int64         `json:"finished_at_ms,omitempty"`
	Alarm        *BrowserAlarm `json:"alarm,omitempty"`
	Sound        bool          `json:"sound"`
	Notify       bool          `json:"notify"`
}

type Template struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Duration    string        `json:"duration"`
	Group       string        `json:"group,omitempty"`
	GroupOrder  int           `json:"group_order,omitempty"`
	Shortcut    string        `json:"shortcut,omitempty"`
	BrowserOnly bool          `json:"browser_only,omitempty"`
	Alarm       *BrowserAlarm `json:"alarm,omitempty"`
	Sound       *bool         `json:"sound,omitempty"`  // nil on legacy templates means enabled
	Notify      *bool         `json:"notify,omitempty"` // nil on legacy templates means enabled
}

// RecentTimer captures immutable settings at start time for one-click replay.
type RecentTimer struct {
	Name        string        `json:"name"`
	Duration    string        `json:"duration"`
	BrowserOnly bool          `json:"browser_only"`
	Sound       bool          `json:"sound"`
	Notify      bool          `json:"notify"`
	Alarm       *BrowserAlarm `json:"alarm,omitempty"`
	StartedAtMS int64         `json:"started_at_ms"`
}

type Snapshot struct {
	ServerNowMS int64         `json:"server_now_ms"`
	FocusID     string        `json:"focus_id,omitempty"`
	Timers      []TimerView   `json:"timers"`
	Templates   []Template    `json:"templates"`
	Groups      []string      `json:"groups"`
	Recent      []RecentTimer `json:"recent"`
}

type Event struct {
	Type  string     `json:"type"`
	Timer *TimerView `json:"timer,omitempty"`
	State Snapshot   `json:"state"`
}
