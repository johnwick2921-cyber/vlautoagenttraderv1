package kernel

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// P3.3 — the day-plan document (the schema-strict JSON the planner AI emits).
// The card renders it; the executor cites its scenarios. Reasoning-fields-FIRST
// (reasoning before the answer fields) per the contract.

// PlanBias is the plan's directional bias + explicit flip condition.
type PlanBias struct {
	Direction     string `json:"direction"`      // long | short | neutral
	Conviction    string `json:"conviction"`     // high | medium | low
	FlipCondition string `json:"flip_condition"` // e.g. "flips short on 2x5m < 30148"
}

// PlanLevel is one graded reference level with an instruction verb.
type PlanLevel struct {
	Price       float64 `json:"price"`
	Label       string  `json:"label"`       // provenance chip: PDH, ONH, nPOC·Tue, RN, EQH…
	Grade       string  `json:"grade"`       // A | B | C (MODEL-written)
	Instruction string  `json:"instruction"` // instruction verb, e.g. "fade", "reclaim-long"
	// MachineGrade is the deterministic detector-side grade (type × freshness ×
	// confluence × HTF — levels_score.go) stamped at plan write by matching the
	// plan level back to the Go-ranked candidate table. Empty when no match
	// (owner levels grade A by design; unseated detector levels have none).
	// Master-audit finding 8.4: the card used to show ONLY the model's grade.
	MachineGrade string `json:"machine_grade,omitempty"`
}

// PlanScenario is one if/then play in the formal grammar.
// PlanConfirm (C1/F3, fail-register wave 2026-08-20) — the STRUCTURED
// confirmation the owner asked for ("2x5m vs 15m close — I need to totally
// understand"). The prose trigger/invalid REMAIN AI-judged; this object is
// machine-computed into an advisory prompt line + card chip (MET / NOT MET)
// using the same acceptance machinery as plan death — the AI stays the final
// judge (no new hard gate; the suppression class stays dead), but it reasons
// from machine truth instead of re-deriving closes itself.
type PlanConfirm struct {
	Rule     string  `json:"rule"`      // touch | 1x5m_close | 2x5m_close | 15m_close
	RefPrice float64 `json:"ref_price"` // the price the closes are counted against
	Side     string  `json:"side"`      // above | below
}

type PlanScenario struct {
	ID          string    `json:"id"`           // S1, S2, S3
	Trigger     string    `json:"trigger"`      // the setup description
	Condition   string    `json:"condition"`    // reclaim|hold|sweep_reclaim|reject|acceptance|breakout_retest
	Direction   string    `json:"direction"`    // long | short
	TargetChain []float64 `json:"target_chain"` // ordered targets
	Invalid     string    `json:"invalid"`      // invalidation
	// Confirm (C1) — REQUIRED after the grace window; see PlanConfirm.
	Confirm *PlanConfirm `json:"confirm,omitempty"`
	Quality string       `json:"quality"` // A+ | A | B
	// G5 (regime wave 2026-08-21) — true when the trigger level was CONSUMED at
	// write/re-align time: quality is capped at C and the card badges it
	// "level consumed". Advisory — never a gate.
	Consumed bool `json:"consumed,omitempty"`
}

// PlanDoc is the full plan (stored as the plans.doc JSON).
type PlanDoc struct {
	Reasoning      string         `json:"reasoning"` // reasoning FIRST
	Bias           PlanBias       `json:"bias"`
	Levels         []PlanLevel    `json:"levels"`
	Scenarios      []PlanScenario `json:"scenarios"`
	NoTrade        []string       `json:"no_trade"`
	DeathCondition string         `json:"death_condition"`
	DayType        string         `json:"day_type,omitempty"`

	// P0.3 (2026-08-19) — MACHINE-EVALUABLE death/flip. The prose fields above
	// stay (card display + back-compat); these structured fields are what Go
	// evaluates every cycle. Empty → the old all-levels-consumed fallback
	// remains for legacy stored plans.
	DeathStructured *PlanCondition `json:"death,omitempty"`
	FlipStructured  *PlanCondition `json:"flip,omitempty"`
}

// PlanCondition is a checkable predicate: price closes beyond `Price` on the
// rule timeframe (`Rule`: "2x5m" | "15m_close" | "5m_close"), on `Side`
// ("below" | "above"). `FlipTo` names the direction the bias flips to when the
// flip condition fires ("" for death).
type PlanCondition struct {
	Price  float64 `json:"price"`
	Side   string  `json:"side"` // below | above
	Rule   string  `json:"rule"` // 2x5m | 15m_close | 5m_close
	FlipTo string  `json:"flip_to,omitempty"`
}

// conditionRules / conditionSides are the enums PlanCondition validates against.
var (
	conditionRules = map[string]bool{"2x5m": true, "15m_close": true, "5m_close": true}
	conditionSides = map[string]bool{"below": true, "above": true}
)

var (
	biasDirections    = map[string]bool{"long": true, "short": true, "neutral": true}
	biasConvictions   = map[string]bool{"high": true, "medium": true, "low": true}
	levelGrades       = map[string]bool{"A": true, "B": true, "C": true}
	scenarioConds     = map[string]bool{"reclaim": true, "hold": true, "sweep_reclaim": true, "reject": true, "acceptance": true, "breakout_retest": true}
	scenarioDirs      = map[string]bool{"long": true, "short": true}
	// C is ACCEPTED: it is the G5 machine-demoted state (trigger level consumed
	// at write/re-align time), never a model-written grade. The write path runs
	// demoteConsumedScenarios BEFORE validation, so rejecting C made every
	// consumed-trigger plan fail-closed (London/ASIA 2026-08-23/24).
	scenarioQualities = map[string]bool{"A+": true, "A": true, "B": true, "C": true}
)

const (
	planMaxLevels    = 8 // shipped default; the owner's max_levels (3–12) may raise it
	planMaxScenarios = 3 // shipped default; the owner's scenario_cap (1–5) may raise it

	// PlanHardMaxLevels / PlanHardMaxScenarios are the HARD CEILINGS no plan may
	// ever exceed — the UI range's top (max_levels 3–12, scenario_cap 1–5). The
	// resolved config can raise the shipped default up to these, never past.
	PlanHardMaxLevels    = 12
	PlanHardMaxScenarios = 5
)

// resolvePlanCaps turns resolved config values into effective caps: ≤0 → shipped
// defaults; above the hard ceilings → clamped DOWN to them (a bad config value
// can never widen the schema past what the UI offers).
func resolvePlanCaps(maxLevels, maxScenarios int) (maxL, maxS int) {
	maxL = planMaxLevels
	if maxLevels > 0 {
		maxL = maxLevels
	}
	if maxL > PlanHardMaxLevels {
		maxL = PlanHardMaxLevels
	}
	maxS = planMaxScenarios
	if maxScenarios > 0 {
		maxS = maxScenarios
	}
	if maxS > PlanHardMaxScenarios {
		maxS = PlanHardMaxScenarios
	}
	return maxL, maxS
}

// CollapsePlanLevels merges plan levels closer than tol points into ONE entry
// (the P0.4 cluster rule, applied to the model's own output — 2026-08-24 ASIA
// v2 fail-closed because the model wrote two levels 2.13 pts apart and the
// duplicate-seat validation burned all retries). The survivor is the higher
// grade; ties keep the first. Consumed ORs across the merge. Pure.
func CollapsePlanLevels(levels []PlanLevel, tol float64) ([]PlanLevel, int) {
	if len(levels) < 2 || tol <= 0 {
		return levels, 0
	}
	out := make([]PlanLevel, 0, len(levels))
	merged := 0
	for _, l := range levels {
		hit := -1
		for i, o := range out {
			if math.Abs(o.Price-l.Price) <= tol {
				hit = i
				break
			}
		}
		if hit < 0 {
			out = append(out, l)
			continue
		}
		merged++
		old := out[hit]
		if planGradeRank(l.Grade) > planGradeRank(old.Grade) {
			out[hit] = l
		}
	}
	return out, merged
}

// planGradeRank ranks a level grade A > B > C (unknown → 0).
func planGradeRank(g string) int {
	switch strings.ToUpper(strings.TrimSpace(g)) {
	case "A":
		return 3
	case "B":
		return 2
	case "C":
		return 1
	}
	return 0
}

// ParsePlanDoc extracts the JSON object from raw model output (tolerating
// surrounding prose / code fences), unmarshals it, and validates it against the
// schema at the SHIPPED caps (8 levels / 3 scenarios). Any failure → error, which
// the planner treats as a retryable/fail-closed event.
func ParsePlanDoc(raw string) (*PlanDoc, error) {
	return ParsePlanDocCapped(raw, 0, 0)
}

// ParsePlanDocCapped is ParsePlanDoc with the RESOLVED config caps (max_levels,
// scenario_cap). H4/H5: the owner's raised caps (9–12 levels, 4–5 scenarios) must
// pass validation instead of making every read fail-closed against the hardcoded
// 8/3.
func ParsePlanDocCapped(raw string, maxLevels, maxScenarios int) (*PlanDoc, error) {
	js := extractJSONObject(raw)
	if js == "" {
		return nil, fmt.Errorf("no JSON object found in planner output")
	}
	var doc PlanDoc
	if err := json.Unmarshal([]byte(js), &doc); err != nil {
		return nil, fmt.Errorf("plan JSON unmarshal: %w", err)
	}
	if err := ValidatePlanDocWithCaps(&doc, maxLevels, maxScenarios); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ValidatePlanDoc enforces the schema-strict rules at the SHIPPED caps (levels
// ≤8, scenarios 1–3).
func ValidatePlanDoc(d *PlanDoc) error {
	return ValidatePlanDocWithCaps(d, 0, 0)
}

// ValidatePlanDocWithCaps enforces the schema-strict rules: required fields, enum
// values, and counts at the RESOLVED caps (clamped to the 12/5 hard ceilings).
// ≤0 → shipped defaults, so default callers are byte-identical to before.
func ValidatePlanDocWithCaps(d *PlanDoc, maxLevels, maxScenarios int) error {
	maxL, maxS := resolvePlanCaps(maxLevels, maxScenarios)
	if d == nil {
		return fmt.Errorf("nil plan")
	}
	if strings.TrimSpace(d.Reasoning) == "" {
		return fmt.Errorf("reasoning is required (reasoning-first)")
	}
	if !biasDirections[d.Bias.Direction] {
		return fmt.Errorf("bias.direction %q invalid (long|short|neutral)", d.Bias.Direction)
	}
	if d.Bias.Conviction != "" && !biasConvictions[d.Bias.Conviction] {
		return fmt.Errorf("bias.conviction %q invalid (high|medium|low)", d.Bias.Conviction)
	}
	if strings.TrimSpace(d.DeathCondition) == "" {
		return fmt.Errorf("death_condition is required")
	}
	if len(d.Levels) > maxL {
		return fmt.Errorf("too many levels: %d (max %d)", len(d.Levels), maxL)
	}
	for i, l := range d.Levels {
		if !levelGrades[l.Grade] {
			return fmt.Errorf("level[%d].grade %q invalid (A|B|C)", i, l.Grade)
		}
		// P5.1 hardening — a non-positive price is never a real level (armors both
		// the write path and read-time plan_final re-validation, all overlay
		// origins). Positive-but-implausible prices are caught by LevelPriceViolation.
		if l.Price <= 0 {
			return fmt.Errorf("level[%d].price %v invalid (must be > 0)", i, l.Price)
		}
	}
	if len(d.Scenarios) < 1 || len(d.Scenarios) > maxS {
		return fmt.Errorf("scenarios count %d invalid (1..%d)", len(d.Scenarios), maxS)
	}
	for i, s := range d.Scenarios {
		if strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("scenario[%d].id is required", i)
		}
		// A5 (F11, fail-register wave): the id format is a contract now — the
		// cite rule, the status map, the chips and adherence all key on it.
		if !scenarioIDRe.MatchString(strings.TrimSpace(s.ID)) {
			return fmt.Errorf("scenario[%d].id %q invalid (format: S1..S99)", i, s.ID)
		}
		if !scenarioConds[s.Condition] {
			return fmt.Errorf("scenario[%d].condition %q invalid", i, s.Condition)
		}
		if !scenarioDirs[s.Direction] {
			return fmt.Errorf("scenario[%d].direction %q invalid (long|short)", i, s.Direction)
		}
		if !scenarioQualities[s.Quality] {
			return fmt.Errorf("scenario[%d].quality %q invalid (A+|A|B|C — C is the G5 machine-demoted 'level consumed' state)", i, s.Quality)
		}
		for j, t := range s.TargetChain {
			if t <= 0 {
				return fmt.Errorf("scenario[%d].target_chain[%d] %v invalid (must be > 0)", i, j, t)
			}
		}
		// C1 (F3): when authored, the structured confirmation must be coherent
		// AND its number must appear in the prose trigger/invalid (the A3
		// object↔prose contract). Absence is judged at the WRITE SITE (grace
		// window), not here.
		if s.Confirm != nil {
			if !confirmRules[s.Confirm.Rule] {
				return fmt.Errorf("scenario[%d].confirm.rule %q invalid (touch|1x5m_close|2x5m_close|15m_close)", i, s.Confirm.Rule)
			}
			if s.Confirm.Side != "above" && s.Confirm.Side != "below" {
				return fmt.Errorf("scenario[%d].confirm.side %q invalid (above|below)", i, s.Confirm.Side)
			}
			if s.Confirm.RefPrice <= 0 {
				return fmt.Errorf("scenario[%d].confirm.ref_price %v invalid", i, s.Confirm.RefPrice)
			}
			if !numberNearInText(s.Trigger+" "+s.Invalid, s.Confirm.RefPrice, 2.0) {
				return fmt.Errorf("scenario[%d].confirm.ref_price %.2f does not match any number in the trigger/invalid prose (object and prose must agree)", i, s.Confirm.RefPrice)
			}
		}
	}
	// P0.3 (2026-08-19) — structured conditions validate when PRESENT (legacy
	// stored plans without them still pass — the all-levels-consumed fallback
	// governs those).
	for _, cond := range []struct {
		name string
		c    *PlanCondition
	}{{"death", d.DeathStructured}, {"flip", d.FlipStructured}} {
		if cond.c == nil {
			continue
		}
		if cond.c.Price <= 0 {
			return fmt.Errorf("%s.price %v invalid (must be > 0)", cond.name, cond.c.Price)
		}
		if !conditionSides[cond.c.Side] {
			return fmt.Errorf("%s.side %q invalid (below|above)", cond.name, cond.c.Side)
		}
		if !conditionRules[cond.c.Rule] {
			return fmt.Errorf("%s.rule %q invalid (2x5m|15m_close|5m_close)", cond.name, cond.c.Rule)
		}
		if cond.name == "flip" && cond.c.FlipTo != "" && !biasDirections[cond.c.FlipTo] {
			return fmt.Errorf("flip.flip_to %q invalid (long|short)", cond.c.FlipTo)
		}
	}
	// A3 (F5, fail-register wave): the prompt has always CLAIMED "death/flip
	// objects must match the prose lines" — nothing checked it. Now the
	// validator does: a structured price must appear (±2pts) among the numbers
	// in its prose twin, else the planner retries with a named error.
	if d.DeathStructured != nil && strings.TrimSpace(d.DeathCondition) != "" {
		if !numberNearInText(d.DeathCondition, d.DeathStructured.Price, 2.0) {
			return fmt.Errorf("death{price %.2f} does not match any number in death_condition prose %q (object and prose must agree)", d.DeathStructured.Price, d.DeathCondition)
		}
	}
	if d.FlipStructured != nil && strings.TrimSpace(d.Bias.FlipCondition) != "" {
		if !numberNearInText(d.Bias.FlipCondition, d.FlipStructured.Price, 2.0) {
			return fmt.Errorf("flip{price %.2f} does not match any number in bias.flip_condition prose %q", d.FlipStructured.Price, d.Bias.FlipCondition)
		}
	}
	return nil
}

// FlipToDirection parses the flip direction out of a killer line
// ("flip-condition: ... → bias long") — "long"/"short", "" otherwise. Used by
// the write site to enforce that a flip-triggered re-plan honors the flip.
func FlipToDirection(killer string) string {
	k := strings.ToLower(killer)
	if i := strings.Index(k, "bias long"); i >= 0 {
		return "long"
	}
	if i := strings.Index(k, "bias short"); i >= 0 {
		return "short"
	}
	return ""
}

// structuralLabels are the detector-anchored label prefixes the model may NOT
// re-invent: a plan level whose price matches a machine-table row must carry
// the table's label, or the plan ships a phantom anchor (LONDON v1: "PDH
// 29297.75" when the true prior-day high was 29290.5).
var structuralLabels = map[string]bool{
	"PDH": true, "PDL": true, "PDC": true,
	"RTH-H": true, "RTH-L": true,
	"ONH": true, "ONL": true,
	"AS-H": true, "AS-L": true,
	"LDN-H": true, "LDN-L": true,
	"PWH": true, "PWL": true, "PMH": true, "PML": true,
	"EQH": true, "EQL": true,
}

// structuralPrefix returns the leading structural token of a label (e.g.
// "PDL", "EQH·4h" → "EQH"), "" when the label isn't structural. Only the "·"
// separator is split — RTH-H/AS-H/LDN-H are exact structural labels.
func structuralPrefix(label string) string {
	l := strings.TrimSpace(label)
	if i := strings.Index(l, "·"); i > 0 {
		l = l[:i]
	}
	if structuralLabels[l] {
		return l
	}
	return ""
}

// MislabeledStructuralLevels (P0.4-H, 2026-08-25) reports plan levels whose
// rounded price matches a machine-table row but whose label is a DIFFERENT
// structural anchor than the table's. Rounded-price keying mirrors the
// machine-grade stamp. Empty = clean. Pure.
func MislabeledStructuralLevels(d *PlanDoc, machineLabels map[float64]string) []string {
	if d == nil || len(machineLabels) == 0 {
		return nil
	}
	var out []string
	for _, l := range d.Levels {
		k := math.Round(l.Price*100) / 100
		ml, ok := machineLabels[k]
		if !ok {
			continue // not a machine-table price — free label
		}
		mp := structuralPrefix(ml)
		lp := structuralPrefix(l.Label)
		// Flag when EITHER side is a structural anchor and they disagree:
		// the LONDON v1 phantom was the model writing "PDH" (structural) over
		// a machine row whose label is a ZONE (non-structural). A structural
		// label may never be re-invented for a table price.
		if (mp != "" || lp != "") && mp != lp {
			out = append(out, fmt.Sprintf("%.2f labeled %q but the machine table says %q", l.Price, l.Label, ml))
		}
	}
	return out
}
type PlanFacts struct {
	Price float64 // reference price at read time
	DATR  float64 // daily ATR proxy
	PDH   float64 // prior day high (0 = unknown → gap rules skipped)
	PDL   float64 // prior day low (0 = unknown → gap rules skipped)
}

// ValidatePlanDocWithFacts = schema rules + P0.1/P0.2 facts rules:
//   - levels must carry ≥ MinSideLevels on EACH side of price (one-sided maps
//     are the 2026-08-18 pathology — a 110-point breakdown with zero downside
//     levels);
//   - price BELOW PDL (gap-down) → a continuation SHORT scenario is mandatory;
//     price ABOVE PDH (gap-up) → a continuation LONG scenario is mandatory;
//   - no two levels within the cluster tolerance (duplicate seats);
//   - every scenario target must sit within the proximity band of price.
func ValidatePlanDocWithFacts(d *PlanDoc, facts PlanFacts, maxLevels, maxScenarios int) error {
	if err := ValidatePlanDocWithCaps(d, maxLevels, maxScenarios); err != nil {
		return err
	}
	if facts.Price <= 0 {
		return nil // no facts → schema-only (legacy callers/tests)
	}
	// P0.4 — duplicate-level rejection (the planner copied an EQ family 4×).
	for i := 0; i < len(d.Levels); i++ {
		for j := i + 1; j < len(d.Levels); j++ {
			if math.Abs(d.Levels[i].Price-d.Levels[j].Price) <= LevelClusterTicks*0.25 {
				return fmt.Errorf("levels[%d] and [%d] are %.2f apart — duplicates within the cluster tolerance; collapse them into one entry",
					i, j, math.Abs(d.Levels[i].Price-d.Levels[j].Price))
			}
		}
	}
	// P0.1 — both-side minimum.
	below, above := 0, 0
	for _, l := range d.Levels {
		switch {
		case l.Price < facts.Price:
			below++
		case l.Price > facts.Price:
			above++
		}
	}
	if below < MinSideLevels {
		return fmt.Errorf("only %d levels below price %.2f — the plan must carry ≥%d on EACH side (add prior week/month lows, swing lows, round numbers or value-area edges below)", below, facts.Price, MinSideLevels)
	}
	if above < MinSideLevels {
		return fmt.Errorf("only %d levels above price %.2f — the plan must carry ≥%d on EACH side", above, facts.Price, MinSideLevels)
	}
	// P0.2 — continuation scenario on a gap out of the prior range.
	if facts.PDL > 0 && facts.Price < facts.PDL && !hasDirection(d.Scenarios, "short") {
		return fmt.Errorf("price %.2f is BELOW PDL %.2f (gap-down) — the plan MUST include a continuation/breakdown short scenario", facts.Price, facts.PDL)
	}
	if facts.PDH > 0 && facts.Price > facts.PDH && !hasDirection(d.Scenarios, "long") {
		return fmt.Errorf("price %.2f is ABOVE PDH %.2f (gap-up) — the plan MUST include a continuation/breakout long scenario", facts.Price, facts.PDH)
	}
	// P0.2-c — the continuation scenario must be REACHABLE from here: a gap-down
	// short whose trigger needs a rally back above price (the 2026-08-18 S3
	// pathology: "rally into 29853/29919" while price sat at 29687) is not a
	// continuation play, it is a re-entry into the old range. The trigger's
	// nearest numeric level must sit AT or beyond price in the gap direction.
	if facts.PDL > 0 && facts.Price < facts.PDL {
		if !continuationReachable(d.Scenarios, "short", facts.Price) {
			return fmt.Errorf("gap-down at %.2f (< PDL %.2f): the short scenario's trigger must reference a level ≤ current price (breakdown/retest), not a rally back above", facts.Price, facts.PDL)
		}
	}
	if facts.PDH > 0 && facts.Price > facts.PDH {
		if !continuationReachable(d.Scenarios, "long", facts.Price) {
			return fmt.Errorf("gap-up at %.2f (> PDH %.2f): the long scenario's trigger must reference a level ≥ current price (breakout/retest), not a dip back below", facts.Price, facts.PDH)
		}
	}
	// P0.2b — targets must be reachable: inside the proximity band.
	band := 1.5 * facts.DATR
	if band <= 0 {
		band = 0.012 * facts.Price // warm-up fallback
	}
	for i, s := range d.Scenarios {
		for _, t := range s.TargetChain {
			if math.Abs(t-facts.Price) > band {
				return fmt.Errorf("scenario[%d] target %.2f is %.0f pts from price %.2f — outside the %.0f-pt proximity band (unreachable target)", i, t, math.Abs(t-facts.Price), facts.Price, band)
			}
		}
	}
	return nil
}

func hasDirection(scenarios []PlanScenario, dir string) bool {
	for _, s := range scenarios {
		if s.Direction == dir {
			return true
		}
	}
	return false
}

// triggerNumbers extracts the numeric levels (≥3 digits, >100 — filters clock
// times like "08:35") mentioned in a trigger's prose.
func triggerNumbers(trigger string) []float64 {
	var out []float64
	for _, m := range reTriggerNumber.FindAllString(trigger, -1) {
		// A4-consistent (fail-register wave): any positive decimal counts — the
		// old v>100 floor made sub-1000-priced instruments unminable. Callers
		// that need price-magnitude filtering do it themselves
		// (continuationReachable keeps its beyond-price comparison).
		if v, err := strconv.ParseFloat(m, 64); err == nil && v > 0 {
			out = append(out, v)
		}
	}
	return out
}

var reTriggerNumber = regexp.MustCompile(`\d+(?:\.\d+)?`)

// continuationReachable reports whether at least one scenario in `dir` has a
// trigger whose nearest numeric level sits AT or beyond price in that direction
// (a play reachable without crossing the whole map).
func continuationReachable(scenarios []PlanScenario, dir string, price float64) bool {
	for _, s := range scenarios {
		if s.Direction != dir {
			continue
		}
		for _, n := range triggerNumbers(s.Trigger) {
			// Price-magnitude band (A4-consistent widening moved the >100
			// filter out of the miner): only numbers in the same magnitude as
			// price count as price references — "2x5m"-style vocabulary
			// digits ("2", "5", "15") can never satisfy reachability.
			if n < price*0.5 || n > price*1.5 {
				continue
			}
			if dir == "short" && n <= price {
				return true
			}
			if dir == "long" && n >= price {
				return true
			}
		}
	}
	return false
}

// extractJSONObject returns the substring from the first '{' to the matching
// last '}' (brace-balanced), or "" if none.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// NoTradePlanDoc builds the FAIL-CLOSED no-trade plan: a valid plan with a
// neutral bias and a single "no-trade" scenario, so a read failure still writes a
// concrete NO-TRADE plan row (never a stale plan, never nothing).
func NoTradePlanDoc(reason string) *PlanDoc {
	return &PlanDoc{
		Reasoning:      "FAIL-CLOSED: " + reason + " — no valid plan produced; sitting out this session.",
		Bias:           PlanBias{Direction: "neutral", Conviction: "low", FlipCondition: "n/a"},
		Levels:         nil,
		Scenarios:      []PlanScenario{{ID: "S0", Trigger: "none", Condition: "hold", Direction: "long", Invalid: "n/a", Quality: "B"}},
		NoTrade:        []string{"ENTIRE SESSION — planner fail-closed"},
		DeathCondition: "already dead (fail-closed)",
		DayType:        "no-trade",
	}
}

// NoTradePlanDocWithLevels is the P7 form: a NO-TRADE plan that still CARRIES THE
// MAP. Levels are market FACTS; the plan is an opinion about them — a no-trade
// decision must never erase the map. The caller passes the current
// detector/scorer output; when genuinely unavailable, the doc says so explicitly
// (an empty levels section must never render without a reason).
func NoTradePlanDocWithLevels(reason string, levels []PlanLevel) *PlanDoc {
	doc := NoTradePlanDoc(reason)
	if len(levels) == 0 {
		doc.NoTrade = append(doc.NoTrade, "detector data unavailable — no level map could be assembled")
		return doc
	}
	doc.Levels = levels
	return doc
}

// scenarioIDRe — A5 (F11): "S1".."S99", the convention everything keys on.
// S0 is reserved for the Go-authored fail-closed NO-TRADE stub plan.
var scenarioIDRe = regexp.MustCompile(`^S\d{1,2}$`)

// numberNearInText — A3 (F5): does any number token in the prose sit within
// tol points of want? Reuses the trigger-mining tokenizer.
func numberNearInText(text string, want, tol float64) bool {
	for _, v := range triggerNumbers(text) {
		if v >= want-tol && v <= want+tol {
			return true
		}
	}
	return false
}

// confirmRules — C1 vocabulary. 1x5m_close maps to the A2-fixed "5m-close"
// acceptance rule; 2x5m_close → "2x5m"; 15m_close → "15m-close".
var confirmRules = map[string]bool{"touch": true, "1x5m_close": true, "2x5m_close": true, "15m_close": true}
