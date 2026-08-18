package kernel

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// P0.3 — SESSION REGISTRY (global-admin, CT-anchored).
//
// The day-plan system runs PER-SESSION reads (ASIA / LONDON / NY). This registry
// is the single source of truth for each session's clock: window, planner read
// time, session-flat time, killzones, and whether it is enabled. It follows the
// NT8 Trading-Hours pattern — every time is America/Chicago wall-clock "HH:MM",
// NEVER local machine time — so DST is handled by the tz database and the ~3
// weeks US/UK diverge only surface as a London DST-drift warning (UI concern).
//
// This is FOUNDATION only: the struct, defaults, storage codec, and pure
// evaluators. Wiring it into the live decision gate is P2 (THE CLOCK). Persisted
// as JSON in system_config under SessionRegistryConfigKey; empty/absent → the
// default (NY-only) registry, so nothing changes until an admin edits it.

// SessionRegistryConfigKey is the system_config key holding the registry JSON.
const SessionRegistryConfigKey = "session_registry"

// Session names (CT-anchored windows; see DefaultSessionRegistry).
const (
	SessionAsia   = "ASIA"
	SessionLondon = "LONDON"
	SessionNY     = "NY"
)

// KillzoneCT is a high-probability CT [start,end) window inside a session.
type KillzoneCT struct {
	Name    string `json:"name"`
	StartCT string `json:"start_ct"` // "HH:MM" America/Chicago
	EndCT   string `json:"end_ct"`   // "HH:MM" America/Chicago
}

// SessionDef is one CT-anchored trading session. Windows may wrap midnight
// (ASIA 17:00→02:00); all times are America/Chicago "HH:MM".
type SessionDef struct {
	Name          string       `json:"name"`
	WindowStartCT string       `json:"window_start_ct"`
	WindowEndCT   string       `json:"window_end_ct"`
	ReadCT        string       `json:"read_ct"` // planner read time (≈5 min before open)
	FlatCT        string       `json:"flat_ct"` // session-flat time
	Killzones     []KillzoneCT `json:"killzones,omitempty"`
	Enabled       bool         `json:"enabled"`
}

// SessionRegistry is the global-admin session config.
type SessionRegistry struct {
	Sessions []SessionDef `json:"sessions"`
	// HalfDays maps a CME session-day key (YYYY-MM-DD, per CMESessionDayKey) to
	// an early-close CT "HH:MM" that overrides the affected session's flat time.
	// EMPTY BY DEFAULT (dormant) — the half-day truth hook (RECON #6). Nothing
	// changes live while empty; the calendar feed populates it later.
	HalfDays map[string]string `json:"half_days,omitempty"`
}

// DefaultSessionRegistry returns the shipped registry: three CT-anchored
// sessions with only NY enabled (ASIA/LONDON earn enablement via replay +
// NY match-rate evidence). Read/window/flat per the spec.
//
// SINGLE SOURCE OF TRUTH for session clocks. Everything else — the trader gates,
// the EOD flat, the FE strip/tabs (web/src/components/plan/sessionConfig.ts) and
// the spec — MIRRORS this table and is asserted against it by
// TestSessionEndSingleSourceOfTruth. Every value is America/Chicago ("CT"),
// never ET, never local.
//
// NY CONTRACT (owner, 2026-08-16): the NY session ENDS at 15:45 ET = 14:45 CT and
// the EOD flat is at that same instant. Window end and FlatCT are therefore the
// SAME value: before this they drifted (window ran to 15:00 CT while the flat
// fired at 14:45), leaving a 15-minute band where the gate still called the
// session open even though positions had already been flattened.
func DefaultSessionRegistry() SessionRegistry {
	return SessionRegistry{
		Sessions: []SessionDef{
			{
				Name:          SessionAsia,
				WindowStartCT: "17:00",
				WindowEndCT:   "02:00", // wraps midnight
				ReadCT:        "16:55",
				FlatCT:        "02:00",
				Killzones:     []KillzoneCT{{Name: "asia_kz", StartCT: "19:00", EndCT: "23:00"}},
				Enabled:       false,
			},
			{
				Name:          SessionLondon,
				WindowStartCT: "02:00",
				WindowEndCT:   "08:30",
				ReadCT:        "01:55",
				FlatCT:        "08:30",
				Killzones:     []KillzoneCT{{Name: "london_kz", StartCT: "02:00", EndCT: "05:00"}},
				Enabled:       false,
			},
			{
				Name:          SessionNY,
				WindowStartCT: "08:30",
				WindowEndCT:   "14:45", // = 15:45 ET — session end == EOD flat (owner contract)
				ReadCT:        "08:25",
				FlatCT:        "14:45", // = 15:45 ET — same instant as WindowEndCT by contract
				Killzones: []KillzoneCT{
					{Name: "ny_am", StartCT: "08:30", EndCT: "11:00"},
					{Name: "ny_pm", StartCT: "13:00", EndCT: "14:45"},
				},
				Enabled: true,
			},
		},
	}
}

// Marshal serializes the registry to a JSON string for system_config.
func (r SessionRegistry) Marshal() (string, error) {
	b, err := json.Marshal(r)
	return string(b), err
}

// LoadSessionRegistry parses a stored registry JSON. Empty input → the default
// registry. On a parse error it returns the default AND the error so the caller
// can log and never run with an empty registry (fail-safe, never silently off).
func LoadSessionRegistry(raw string) (SessionRegistry, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultSessionRegistry(), nil
	}
	var r SessionRegistry
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return DefaultSessionRegistry(), err
	}
	if len(r.Sessions) == 0 {
		return DefaultSessionRegistry(), nil
	}
	return r, nil
}

// ValidateSessionRegistry rejects a registry that would break the gates — unlike
// LoadSessionRegistry (which fail-safes to the default on bad input), an EDIT must
// be refused, not silently defaulted. Requires ≥1 session, each with a name and
// four parseable HH:MM CT anchors (window start/end, read, flat); killzone bounds
// must parse too. Returns nil when safe to persist.
func ValidateSessionRegistry(r SessionRegistry) error {
	if len(r.Sessions) == 0 {
		return fmt.Errorf("registry must have at least one session")
	}
	for i := range r.Sessions {
		s := r.Sessions[i]
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("session %d: name required", i)
		}
		for field, v := range map[string]string{
			"window_start_ct": s.WindowStartCT, "window_end_ct": s.WindowEndCT,
			"read_ct": s.ReadCT, "flat_ct": s.FlatCT,
		} {
			if _, ok := parseHHMM(v); !ok {
				return fmt.Errorf("session %q: %s %q is not HH:MM", s.Name, field, v)
			}
		}
		for _, kz := range s.Killzones {
			if _, ok := parseHHMM(kz.StartCT); !ok {
				return fmt.Errorf("session %q killzone %q: start %q is not HH:MM", s.Name, kz.Name, kz.StartCT)
			}
			if _, ok := parseHHMM(kz.EndCT); !ok {
				return fmt.Errorf("session %q killzone %q: end %q is not HH:MM", s.Name, kz.Name, kz.EndCT)
			}
		}
	}
	return nil
}

// SessionByName returns the named session (case-insensitive), or (nil, false).
func (r SessionRegistry) SessionByName(name string) (*SessionDef, bool) {
	for i := range r.Sessions {
		if strings.EqualFold(r.Sessions[i].Name, name) {
			return &r.Sessions[i], true
		}
	}
	return nil, false
}

// ActiveSession returns the session whose CT window contains now, or (nil,false)
// if none (the 15:00–17:00 CT post-close gap, the daily break, etc.). First
// match wins; the default windows tile without overlap.
func (r SessionRegistry) ActiveSession(now time.Time) (*SessionDef, bool) {
	for i := range r.Sessions {
		if r.Sessions[i].InWindow(now) {
			return &r.Sessions[i], true
		}
	}
	return nil, false
}

// IsNightMode (P3.6-D) reports whether now is in NIGHT state: there is no ACTIVE
// ENABLED session window (a disabled session window, an interim gap, or overnight
// all count as night). During night the day-plan does no reads and takes no
// entries. Derived purely from the clock → a restart re-derives it identically.
func (r SessionRegistry) IsNightMode(now time.Time) bool {
	sess, ok := r.ActiveSession(now)
	return !ok || !sess.Enabled
}

// EnabledSessions returns the names of enabled sessions, in registry order.
func (r SessionRegistry) EnabledSessions() []string {
	var out []string
	for i := range r.Sessions {
		if r.Sessions[i].Enabled {
			out = append(out, r.Sessions[i].Name)
		}
	}
	return out
}

// EffectiveFlatCT returns the flat time for a session on a given CME session-day,
// honoring a half-day early-close override when one is registered for that day.
func (r SessionRegistry) EffectiveFlatCT(sessionName, sessionDayKey string) (string, bool) {
	s, ok := r.SessionByName(sessionName)
	if !ok {
		return "", false
	}
	if r.HalfDays != nil {
		if early, ok := r.HalfDays[sessionDayKey]; ok && strings.TrimSpace(early) != "" {
			return early, true
		}
	}
	return s.FlatCT, true
}

// InWindow reports whether now (America/Chicago) falls inside [start, end) —
// reusing the midnight-wrap logic that InBlackoutWindow already proves.
func (d SessionDef) InWindow(now time.Time) bool {
	return InBlackoutWindow(now, d.WindowStartCT, d.WindowEndCT)
}

// InKillzone reports whether now falls inside any of the session's killzones.
func (d SessionDef) InKillzone(now time.Time) bool {
	for _, kz := range d.Killzones {
		if InBlackoutWindow(now, kz.StartCT, kz.EndCT) {
			return true
		}
	}
	return false
}

// IsReadTime reports whether now's CT wall-clock minute equals the session's
// read minute (the once-a-day planner-read trigger; P3 adds the "already read"
// dedupe on top of this primitive).
func (d SessionDef) IsReadTime(now time.Time) bool {
	read, ok := parseHHMM(d.ReadCT)
	if !ok {
		return false
	}
	return minutesSinceMidnightCT(now) == read
}

// minutesSinceMidnightCT returns now's minutes-since-midnight in America/Chicago.
func minutesSinceMidnightCT(now time.Time) int {
	chicago := CTLocation()
	ct := now.In(chicago)
	return ct.Hour()*60 + ct.Minute()
}
