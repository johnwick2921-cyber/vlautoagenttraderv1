package kernel

import (
	"fmt"
	"time"
)

// ── NO-TRADE WINDOWS — ONE DEFINITION EACH (owner ruling 2026-09-02) ────────
// The first-N-minutes and lunch windows were hardcoded in THREE places that
// could drift apart: the entry gate (trader/auto_trader_session.go), the
// adherence grader (adherence.go:121, its own copy of "12:00"/"13:30"), and
// the plan card, which rendered whatever prose the model had written. The gate
// enforced one thing, the grader scored another, and the card claimed a third.
//
// These are the definitions. The gate, the grader and the card all read them.
// They are NOT owner-configurable — every payload built from them is stamped
// SourceCodeConstant so the surface never implies a knob that does not exist.

// NoTradeSource names where a window's boundaries came from.
const (
	SourceCodeConstant = "code-constant" // fixed in code; no config surface yet
	SourceRegistry     = "session-registry"
	SourceCalendar     = "calendar"
)

// NoTradeKind classifies a window for rendering and filtering.
const (
	KindFirstN   = "first_n"
	KindLunch    = "lunch"
	KindT1       = "t1"
	KindKillzone = "killzone"
)

// FirstNoTradeMinutes is the count of minutes after a session's open during
// which entries are refused. ONE definition — the gate's `cur < start+N` and
// the grader's first-N test both resolve here.
func FirstNoTradeMinutes() int { return 5 }

// LunchWindowCT returns the lunch no-trade window as CT wall-clock "HH:MM"
// bounds. ONE definition — the gate's InBlackoutWindow call and the grader's
// used to carry separate copies of these strings.
func LunchWindowCT() (startCT, endCT string) { return "12:00", "13:30" }

// InFirstNoTradeMinutes reports whether t falls in the first N minutes of a
// session whose window starts at startCT. Wrap-safe via minutes-of-day.
func InFirstNoTradeMinutes(startCT string, t time.Time) bool {
	start, ok := parseHHMM(startCT)
	if !ok {
		return false
	}
	n := FirstNoTradeMinutes()
	cur := t.In(CTLocation()).Hour()*60 + t.In(CTLocation()).Minute()
	d := ((cur-start)%1440 + 1440) % 1440
	return d < n
}

// InLunchNoTrade reports whether t falls in the lunch window.
func InLunchNoTrade(t time.Time) bool {
	s, e := LunchWindowCT()
	return InBlackoutWindow(t, s, e)
}

// NoTradeWindow is one machine-derived constraint, stored on the plan doc and
// re-evaluated at read time. Start/End are CT minutes-of-day so a wrapping
// session (ASIA 17:00→02:00) needs no special case at the boundary.
type NoTradeWindow struct {
	StartMin int    `json:"start_min"`
	EndMin   int    `json:"end_min"`
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Source   string `json:"source"`
}

// HHMM renders a minutes-of-day value as CT wall clock.
func HHMM(min int) string {
	m := ((min % 1440) + 1440) % 1440
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// BuildMachineNoTradeWindows assembles the machine-derived windows for one
// session: the first-N window from the session registry's own open, and lunch
// from its single definition. T1 windows are NOT baked here — they are
// re-resolved at read time so a later calendar change shows up (owner ruling).
func BuildMachineNoTradeWindows(sess SessionDef) []NoTradeWindow {
	var out []NoTradeWindow
	if start, ok := parseHHMM(sess.WindowStartCT); ok {
		n := FirstNoTradeMinutes()
		out = append(out, NoTradeWindow{
			StartMin: start, EndMin: (start + n) % 1440,
			Kind: KindFirstN, Source: SourceCodeConstant,
			Label: fmt.Sprintf("first %dm after the %s open (%s–%s CT)", n, sess.Name, HHMM(start), HHMM(start+n)),
		})
	}
	ls, le := LunchWindowCT()
	if s, ok1 := parseHHMM(ls); ok1 {
		if e, ok2 := parseHHMM(le); ok2 {
			out = append(out, NoTradeWindow{
				StartMin: s, EndMin: e, Kind: KindLunch, Source: SourceCodeConstant,
				Label: fmt.Sprintf("lunch %s–%s CT", ls, le),
			})
		}
	}
	return out
}

// ── READ-TIME EVALUATION ────────────────────────────────────────────────────

// NoTradeStatus is how one window relates to the moment it is rendered.
const (
	StatusLive         = "live"          // intersects [now, session EOD]
	StatusElapsed      = "elapsed"       // ended before now
	StatusOtherSession = "other_session" // starts after this session's EOD
)

// RenderedNoTradeWindow is one window plus its read-time verdict.
type RenderedNoTradeWindow struct {
	NoTradeWindow
	Status string `json:"status"`
	// CT wall-clock bounds, resolved here so the card renders a window without
	// doing clock arithmetic of its own (A24 — no literals on the surface).
	StartCT string `json:"start_ct"`
	EndCT   string `json:"end_ct"`
}

// EvaluateNoTradeWindows filters windows against [nowMin, eodMin] in CT
// minutes-of-day, wrap-aware so an ASIA session spanning midnight is handled
// without a special case. Nothing is dropped: elapsed and other-session
// windows come back marked, for the collapsed section.
func EvaluateNoTradeWindows(wins []NoTradeWindow, nowMin, sessionStartMin, eodMin int) []RenderedNoTradeWindow {
	out := make([]RenderedNoTradeWindow, 0, len(wins))
	// Geometry is measured from the session start so a window that wraps past
	// midnight stays monotonic; "has it finished" is measured from NOW on a
	// signed ±12h axis, because a pre-session read has nothing elapsed yet and
	// offsets-from-start alone cannot tell 08:00-before-open from 16:00-after.
	off := func(m int) int { return ((m-sessionStartMin)%1440 + 1440) % 1440 }
	rel := func(m int) int {
		d := ((m-nowMin)%1440 + 1440) % 1440
		if d > 720 {
			d -= 1440 // the nearer reading of a wrapped clock is the past one
		}
		return d
	}
	sessionLen := off(eodMin)
	if sessionLen == 0 {
		sessionLen = 1440
	}
	for _, w := range wins {
		sOff, eOff := off(w.StartMin), off(w.EndMin)
		if eOff <= sOff {
			eOff += 1440 // window wraps past the session start
		}
		// Does the window touch this session's tradeable span at all? A window
		// that does not can never constrain this session, whatever the clock
		// says — that is the ASIA-at-23:00 lie (NY lunch on an ASIA card).
		intersects := sOff < sessionLen || eOff > 1440
		status := StatusLive
		switch {
		case !intersects:
			status = StatusOtherSession
		case rel(w.EndMin) <= 0:
			status = StatusElapsed
		}
		out = append(out, RenderedNoTradeWindow{
			NoTradeWindow: w, Status: status,
			StartCT: HHMM(w.StartMin), EndCT: HHMM(w.EndMin),
		})
	}
	return out
}

// LiveNoTradeWindows returns only the windows still ahead in this session.
func LiveNoTradeWindows(r []RenderedNoTradeWindow) []RenderedNoTradeWindow {
	out := make([]RenderedNoTradeWindow, 0, len(r))
	for _, w := range r {
		if w.Status == StatusLive {
			out = append(out, w)
		}
	}
	return out
}

// T1NoTradeWindowsFromCT converts ALREADY-RESOLVED red-news blackout windows
// into structured band entries. It takes CTWindows rather than raw calendar
// events on purpose: the widening decision (clock drift, static fallback) is
// the ENFORCER'S, made once in t1WindowsFor, so the card cannot compute a
// second answer and disagree with the gate that will actually refuse the entry.
func T1NoTradeWindowsFromCT(wins []CTWindow) []NoTradeWindow {
	out := make([]NoTradeWindow, 0, len(wins))
	for _, w := range wins {
		out = append(out, NoTradeWindow{
			StartMin: w.Start, EndMin: w.End, Kind: KindT1,
			Source: SourceCalendar, Label: "🔴 " + w.Label + " — HARD no-trade (red news)",
		})
	}
	return out
}

// NoTradeBandBootLine is the boot line (every field read from code).
func NoTradeBandBootLine() string {
	ls, le := LunchWindowCT()
	return fmt.Sprintf("no-trade band: first_n=%dm lunch=%s–%s (source=%s, shared by gate+grader+card) · T1 taken from the enforcing gate at plan time (widening and fail-closed fallback included) · every window judged against the reader's clock, not the write clock · model prose renders as notes",
		FirstNoTradeMinutes(), ls, le, SourceCodeConstant)
}

// RenderNoTradeBand evaluates a plan's machine-written no-trade windows against
// the clock the reader is holding, for the session the card is showing.
//
// This is the read-time half. The plan's windows are written once; whether any
// of them still constrains anything is a question only the read can answer, and
// answering it at write time is what put three dead constraints on an ASIA card
// at 23:00 CT. A doc with no machine windows (written before this wave, or by a
// fail-closed path) returns nil and the card falls back to the model's prose.
func RenderNoTradeBand(doc *PlanDoc, sess *SessionDef, now time.Time) []RenderedNoTradeWindow {
	if doc == nil || sess == nil || len(doc.NoTradeWindows) == 0 {
		return nil
	}
	start, okS := hhmmToMinK(sess.WindowStartCT)
	eod, okE := hhmmToMinK(sess.WindowEndCT)
	if !okS || !okE {
		return nil
	}
	ct := now.In(CTLocation())
	return EvaluateNoTradeWindows(doc.NoTradeWindows, ct.Hour()*60+ct.Minute(), start, eod)
}

// NoTradeSchemaExample renders the no_trade field's example in the OUTPUT
// schema from the RESOLVED windows, so the prompt cannot teach the model a
// window the gate does not enforce. It used to be a hand-written literal
// ("first 5m (CT)", "12:00-13:30 CT lunch"): a second copy of the definitions,
// in the one place nothing would ever fail if it drifted.
func NoTradeSchemaExample() string {
	return `  "no_trade": ["<your own sit-out conditions, or omit>"],`
}

// ETtoCT converts an "HH:MM" Eastern wall clock to Central. America/New_York
// and America/Chicago change offset on the same instants, so the gap is always
// exactly one hour and this needs no date.
//
// It exists so the prompt can keep advisory windows that the research states in
// ET while printing them in the ONE clock the prompt declares. The clock line
// says every time in the prompt is CT; before this, three lines printed ET, and
// a model reading "10:30 ET" as CT is an hour out.
func ETtoCT(hhmm string) string {
	m, ok := hhmmToMinK(hhmm)
	if !ok {
		return hhmm // unparseable: pass it through rather than invent a time
	}
	return HHMM(((m-60)%1440 + 1440) % 1440)
}

// NoTradeInstruction is the rule sentence governing what the model may put in
// no_trade (owner ruling 2026-09-03, seam closed).
//
// It used to open "no_trade may contain ONLY the fixed session windows (first
// 5m, 12:00-13:30 CT lunch) plus T1 HARD-blackout lines" — naming as permitted
// content exactly what the machine writes. ASIA v14 came back carrying both.
// Neither the example nor the sentence names a machine window now; the windows
// are still STATED in the no-trade gate block, so the author knows they exist
// and is simply not asked to repeat them.
//
// No resolved values appear here on purpose: there is nothing left to resolve.
func NoTradeInstruction() string {
	return "no_trade: the machine enforces the session windows and T1 blackouts regardless; do not list them — no_trade is for your OWN sit-out conditions, or omit it. " +
		"A T2 caution event is NEVER added to no_trade and never stops entries. "
}
