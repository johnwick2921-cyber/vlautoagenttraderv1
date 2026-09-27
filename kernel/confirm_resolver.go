package kernel

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// W-EXEC-TRUTH W2 (2026-09-23) — THE ONE CONFIRM RESOLVER.
//
// The defect: a scenario STORES its confirmation rule (confirm.rule), and the
// continuation branch of the evaluator never read it. breakdown_continue /
// breakup_continue synthesized "%dx5m_close" from the BD_MIN_CLOSES env knob at
// EVALUATION time, so a plan authored with 2x5m_close (plan row 435, 09-21
// LONDON v5 S1) was judged — rendered, recorded, desked and ARMED — on ONE
// close. The same shape held for time_hold: the minutes the prose states
// ("holds above it for 3 minutes of 1m closes", row 452 S2) existed nowhere a
// machine could read, so ACCEPT_HOLD_MIN (10) silently replaced them.
//
// The law (master dispatch §2 W2 + amendment A5):
//   - the STORED rule governs evaluation. BD_MIN_CLOSES / ACCEPT_HOLD_MIN supply
//     only the AUTHORING default, consulted when no valid stored value exists;
//   - every consumer (the evaluator, the write-time validator, the executor
//     prompt render, the recorded card chip + desk, the arm gate's log and the
//     planner's void/displacement facts) reads THIS resolver, so there is one
//     count everywhere;
//   - touch, displacement (BD_MIN_DISP_ATR), plan death (PlanIsDeadSince) and
//     the bias flip (FlipConfirmCloses) stay DISTINCT rules with their own
//     counters — this resolver never answers for them.
//
// Source-scan pin (confirm_resolver_test.go): bdConfirmCloses() and
// AcceptHoldMin() are read only here, in EntryLawBootLedger and in the planner
// prompt's authoring-default text; plan_lifecycle.go never calls the resolver.

// ConfirmSourceStored — the count/duration came from the scenario's own stored
// rule. ConfirmSourceAuthoringDefault — nothing valid was stored, so the env
// authoring default was used; Why says what was missing.
const (
	ConfirmSourceStored           = "stored"
	ConfirmSourceAuthoringDefault = "authoring_default"
)

// Rule kinds — which primitive the evaluator runs.
const (
	ConfirmKindTouch    = "touch"
	ConfirmKindCloses   = "closes"
	ConfirmKindTimeHold = "time_hold"
	ConfirmKindMSS      = "mss"
)

// ResolvedConfirmRule is the machine's reading of one confirmation: the exact
// rule, the count or duration it will be judged on, and where that number came
// from.
type ResolvedConfirmRule struct {
	Rule     string  `json:"rule"`               // the rule evaluated (e.g. "2x5m_close")
	Kind     string  `json:"kind"`               // touch | closes | time_hold | mss
	Closes   int     `json:"closes,omitempty"`   // closes kind: completed closes needed
	TFMin    int     `json:"tf_min,omitempty"`   // closes kind: the bucket length they are counted on
	HoldMin  int     `json:"hold_min,omitempty"` // time_hold: minutes of completed 1m closes
	RefPrice float64 `json:"ref_price"`
	Side     string  `json:"side"`
	Source   string  `json:"source"`        // stored | authoring_default
	Why      string  `json:"why,omitempty"` // authoring_default: what was missing
}

// AsConfirm returns the confirm the evaluator runs for this resolution. A
// time_hold carries its resolved duration as a stored value so the evaluator
// never re-resolves it to a different number.
func (r ResolvedConfirmRule) AsConfirm() PlanConfirm {
	c := PlanConfirm{Rule: r.Rule, RefPrice: r.RefPrice, Side: r.Side}
	if r.Kind == ConfirmKindTimeHold && r.HoldMin > 0 {
		n := r.HoldMin
		c.HoldMin = &n
	}
	return c
}

// Label names the rule as it was gated, with its source — the arm log and the
// validator message read this, never a literal. e.g. "2x5m_close [stored]",
// "time_hold 10min [ACCEPT_HOLD_MIN=10 · authoring default (no hold_min stored)]".
func (r ResolvedConfirmRule) Label() string {
	rule := r.Rule
	if r.Kind == ConfirmKindTimeHold {
		rule = fmt.Sprintf("%s %dmin", r.Rule, r.HoldMin)
	}
	if r.Source != ConfirmSourceAuthoringDefault {
		return rule + " [stored]"
	}
	knob := ""
	switch r.Kind {
	case ConfirmKindCloses:
		knob = fmt.Sprintf("BD_MIN_CLOSES=%d · ", r.Closes)
	case ConfirmKindTimeHold:
		knob = fmt.Sprintf("ACCEPT_HOLD_MIN=%d · ", r.HoldMin)
	}
	return rule + " [" + knob + r.Why + "]"
}

// ResolveConfirm resolves one stored confirm object.
func ResolveConfirm(c PlanConfirm) ResolvedConfirmRule {
	r := ResolvedConfirmRule{Rule: c.Rule, RefPrice: c.RefPrice, Side: c.Side, Source: ConfirmSourceStored}
	switch c.Rule {
	case "touch":
		r.Kind = ConfirmKindTouch
	case "1m_mss":
		r.Kind = ConfirmKindMSS
	case "time_hold":
		r.Kind = ConfirmKindTimeHold
		switch {
		case c.HoldMin != nil && *c.HoldMin > 0:
			r.HoldMin = *c.HoldMin
		case c.HoldMin != nil:
			r.HoldMin, r.Source = AcceptHoldMin(), ConfirmSourceAuthoringDefault
			r.Why = fmt.Sprintf("authoring default (stored hold_min %d is not positive)", *c.HoldMin)
		default:
			r.HoldMin, r.Source = AcceptHoldMin(), ConfirmSourceAuthoringDefault
			r.Why = "authoring default (no hold_min stored)"
		}
	default:
		// Nx5m_close (and the legacy 15m_close tolerance): the stored rule
		// names both the count and the timeframe.
		r.Kind = ConfirmKindCloses
		r.Closes, r.TFMin = acceptanceRuleShape(confirmAcceptanceRule(c.Rule))
	}
	return r
}

// ResolveScenarioConfirm resolves a scenario's PRIMARY confirmation (leg 1).
// Waterfall-class scenarios (breakdown_continue / breakup_continue with a
// breakdown{} object) are judged against Breakdown.Level on the condition's
// side; their count is the STORED confirm.rule when the entry law allows it for
// the condition (1x5m_close | 2x5m_close), else the BD_MIN_CLOSES authoring
// default. ok=false: the scenario carries no confirmation at all.
func ResolveScenarioConfirm(sc PlanScenario) (ResolvedConfirmRule, bool) {
	if IsBreakdownCondition(sc.Condition) && sc.Breakdown != nil {
		side := "above"
		if breakdownShort(sc.Condition) {
			side = "below"
		}
		why := "no confirm{} stored"
		if sc.Confirm != nil {
			rule := NormalizeConfirmRule(strings.TrimSpace(sc.Confirm.Rule))
			if law, ok := EntryLawFor(sc.Condition); ok && law.Allowed[rule] {
				need, tf := acceptanceRuleShape(confirmAcceptanceRule(rule))
				return ResolvedConfirmRule{Rule: rule, Kind: ConfirmKindCloses, Closes: need, TFMin: tf,
					RefPrice: sc.Breakdown.Level, Side: side, Source: ConfirmSourceStored}, true
			}
			why = fmt.Sprintf("stored confirm.rule %q is not a continuation close rule", sc.Confirm.Rule)
		}
		n := bdConfirmCloses()
		return ResolvedConfirmRule{Rule: fmt.Sprintf("%dx5m_close", n), Kind: ConfirmKindCloses, Closes: n, TFMin: 5,
			RefPrice: sc.Breakdown.Level, Side: side, Source: ConfirmSourceAuthoringDefault,
			Why: "authoring default (" + why + ")"}, true
	}
	if sc.Confirm == nil {
		return ResolvedConfirmRule{}, false
	}
	return ResolveConfirm(*sc.Confirm), true
}

// ConfirmGatedRuleLabel names every leg a wait_confirm arm is actually gated
// on, each with its source — the arm log line (armed_executor wait_confirm MET)
// used to print the single-arm leg literal "touch" for EVERY rule.
func ConfirmGatedRuleLabel(sc PlanScenario) string {
	r, ok := ResolveScenarioConfirm(sc)
	if !ok {
		return "no confirm{}"
	}
	if IsBreakdownCondition(sc.Condition) && sc.Breakdown != nil {
		return "leg 1 " + r.Label() + " → leg 2 retest_fail"
	}
	if sc.Confirm2 != nil {
		return "leg 1 " + r.Label() + " → leg 2 " + ResolveConfirm(*sc.Confirm2).Label()
	}
	return r.Label()
}

// ---- A5: the stored duration of a time_hold, checked at write ----

// validateConfirmHoldMin is the STRUCTURAL check on a present hold_min
// (ValidatePlanDocWithCaps — absent on every stored row that predates it, so
// history never trips it): positive, and only on the rule that reads it.
func validateConfirmHoldMin(i int, label string, c *PlanConfirm) error {
	if c == nil || c.HoldMin == nil {
		return nil
	}
	if c.Rule != "time_hold" {
		return fmt.Errorf("scenario[%d].%s.hold_min is only valid with rule time_hold (got rule %q) — hold_min is the minutes a time_hold holds; drop it or change the rule", i, label, c.Rule)
	}
	if *c.HoldMin <= 0 {
		return fmt.Errorf("scenario[%d].%s.hold_min %d invalid (must be > 0 minutes)", i, label, *c.HoldMin)
	}
	return nil
}

// ConfirmHoldMinHint is the remediation phrase of the write-time duration
// refusal (registered in ValidatorHints; its tokens are confirm-field tokens).
const ConfirmHoldMinHint = "write confirm.hold_min equal to the minutes your trigger/invalid prose states for the time_hold, or take the minutes out of the prose"

var holdWordMinutes = map[string]int{
	"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
	"eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "twenty": 20, "thirty": 30,
}

const holdCountWord = `(\d{1,3}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|twenty|thirty)`

// "3 minutes", "three-minute", "10 consecutive minutes" …
var holdMinutesRe = regexp.MustCompile(`(?i)\b` + holdCountWord + `(?:\s+|-)(?:full\s+|consecutive\s+|straight\s+)?(?:minutes?|mins?)\b`)

// "3 1m closes", "three consecutive one-minute closes" …
var holdMinuteClosesRe = regexp.MustCompile(`(?i)\b` + holdCountWord + `\s+(?:full\s+|consecutive\s+|straight\s+)?(?:1m|1-min|1-minute|one-minute)\s+(?:closes|bars|candles)\b`)

// A "N-minute close/bar/candle" is a TIMEFRAME, not a hold duration.
var holdTimeframeTailRe = regexp.MustCompile(`(?i)^\s*(?:closes?|bars?|candles?|chart|timeframe|tf)\b`)

func holdCount(tok string) (int, bool) {
	if n, err := strconv.Atoi(tok); err == nil && n > 0 {
		return n, true
	}
	n, ok := holdWordMinutes[strings.ToLower(tok)]
	return n, ok
}

// ProseHoldMinutes returns the distinct hold durations (minutes) the prose
// states, ascending. Empty = the prose states none.
func ProseHoldMinutes(prose string) []int {
	seen := map[int]bool{}
	for _, m := range holdMinutesRe.FindAllStringSubmatchIndex(prose, -1) {
		if holdTimeframeTailRe.MatchString(prose[m[1]:]) {
			continue
		}
		if n, ok := holdCount(prose[m[2]:m[3]]); ok {
			seen[n] = true
		}
	}
	for _, m := range holdMinuteClosesRe.FindAllStringSubmatch(prose, -1) {
		if n, ok := holdCount(m[1]); ok {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// ValidateConfirmHoldProse is the A5 WRITE-TIME check (new authoring only —
// parsePlanDocument's newAuthoring branch; stored rows are never re-judged):
// a time_hold whose trigger/invalid prose states a duration must STORE it.
// Refused: the prose states minutes and hold_min is absent (the machine would
// count the authoring default instead), or hold_min is none of the minutes the
// prose states. A prose with no minutes keeps the authoring default, named.
func ValidateConfirmHoldProse(d *PlanDoc) error {
	if d == nil {
		return nil
	}
	for i, s := range d.Scenarios {
		for _, c := range []struct {
			label string
			c     *PlanConfirm
		}{{"confirm", s.Confirm}, {"confirm2", s.Confirm2}} {
			if c.c == nil || c.c.Rule != "time_hold" {
				continue
			}
			stated := ProseHoldMinutes(s.Trigger + " " + s.Invalid)
			if len(stated) == 0 {
				continue
			}
			r := ResolveConfirm(*c.c)
			if r.Source != ConfirmSourceStored {
				return fmt.Errorf("scenario[%d].%s: the prose states a %s-minute hold but %s.hold_min is absent — the machine would count %s; %s (hold_min: %d)",
					i, c.label, joinInts(stated), c.label, r.Label(), ConfirmHoldMinHint, stated[0])
			}
			match := false
			for _, n := range stated {
				match = match || n == r.HoldMin
			}
			if !match {
				return fmt.Errorf("scenario[%d].%s.hold_min %d disagrees with the prose, which states a %s-minute hold — %s",
					i, c.label, r.HoldMin, joinInts(stated), ConfirmHoldMinHint)
			}
		}
	}
	return nil
}

func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, "/")
}

// RepairHoldMinLaw is the repair-prompt excerpt for a hold_min refusal.
const RepairHoldMinLaw = "HOLD-MINUTES LAW: a time_hold is judged on the minutes it STORES — confirm.hold_min is the number of completed 1m closes the hold needs and is legal on time_hold only. When the trigger/invalid prose states minutes (\"holds above for 3 minutes\"), confirm.hold_min MUST equal that number; absent, the machine counts the ACCEPT_HOLD_MIN authoring default instead of your prose. Fix the object (or the prose), never the rule."
