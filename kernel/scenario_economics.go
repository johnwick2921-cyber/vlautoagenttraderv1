package kernel

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"nofx/logger"
	"nofx/market"
)

// New-authoring contract only. Legacy PlanDoc reads never call its refusal seam.
// The version is stamped by the parser, never trusted as a model-supplied bypass.
const ScenarioEconomicsContractVersion = 1
const scenarioObstacleRequired = true
const scenarioEconomicsRequired = true

type ScenarioObstacle struct {
	Price    *float64 `json:"price"`
	Level    string   `json:"level"`
	Family   string   `json:"family"`
	Response string   `json:"response"`
}

// ScenarioGeometry is proposed geometry for a scenario WITHOUT an arm. It never
// authorizes an arm. When arm is present its entry/stop/target are authoritative.
type ScenarioGeometry struct {
	Entry  float64 `json:"entry"`
	Stop   float64 `json:"stop"`
	Target float64 `json:"target"`
}
type ScenarioRoleException struct {
	Level  string `json:"level"`
	Use    string `json:"use"`
	Reason string `json:"reason"`
}
type ScenarioEconomics struct {
	Version             int                     `json:"version"`
	EntryZone           []float64               `json:"entry_zone"`
	Geometry            *ScenarioGeometry       `json:"geometry,omitempty"`
	FirstObstacle       *ScenarioObstacle       `json:"first_obstacle"`
	RToObstacle         *float64                `json:"r_to_obstacle"`
	RToArmTarget        *float64                `json:"r_to_arm_target"`
	TargetPathException string                  `json:"target_path_exception,omitempty"`
	RoleExceptions      []ScenarioRoleException `json:"role_exceptions,omitempty"`
}

type ScenarioEconomicsView struct {
	Geometry  *ScenarioGeometry
	ObstacleR *float64
	ArmR      *float64
	Sub1      bool
}

// EconomicsFor reads explicitly authored economics. Missing legacy economics
// stays UNKNOWN, even where a legacy arm would permit an inferred ratio.
func EconomicsFor(s PlanScenario) ScenarioEconomicsView {
	v := ScenarioEconomicsView{}
	if s.Arm != nil {
		v.Geometry = &ScenarioGeometry{s.Arm.Entry, s.Arm.Stop, s.Arm.Target}
	}
	if s.Economics == nil {
		return v // existing arm prices are known; newly introduced R stays UNKNOWN
	}
	if v.Geometry == nil {
		v.Geometry = s.Economics.Geometry
	}
	g := v.Geometry
	if g == nil || !economicsPrice(g.Entry) || !economicsPrice(g.Stop) || g.Entry == g.Stop {
		return v
	}
	risk := math.Abs(g.Entry - g.Stop)
	if economicsPrice(g.Target) {
		r := math.Abs(g.Target-g.Entry) / risk
		v.ArmR = &r
	}
	if o := s.Economics.FirstObstacle; o != nil && o.Price != nil && economicsPrice(*o.Price) {
		r := math.Abs(*o.Price-g.Entry) / risk
		v.ObstacleR = &r
		v.Sub1 = r < 1
	}
	return v
}
func economicsPrice(n float64) bool { return n > 0 && !math.IsNaN(n) && !math.IsInf(n, 0) }
func economicsNumber(n *float64) string {
	if n == nil {
		return "UNKNOWN"
	}
	return fmt.Sprintf("%.6f", *n)
}

// EconomicsSummary is shared by the scenario desk line and per-attempt log.
func EconomicsSummary(s PlanScenario) string {
	v := EconomicsFor(s)
	obstacle, response, target := "UNKNOWN", "UNKNOWN", "UNKNOWN"
	if v.Geometry != nil && economicsPrice(v.Geometry.Target) {
		target = fmt.Sprintf("%.2f", v.Geometry.Target)
	}
	if e := s.Economics; e != nil && e.FirstObstacle != nil {
		o := e.FirstObstacle
		response = o.Response
		obstacle = fmt.Sprintf("%s %s (%s)", economicsNumber(o.Price), o.Level, o.Family)
	}
	mark := ""
	if v.Sub1 {
		mark = " · sub-1R first obstacle (WARN)"
	}
	return fmt.Sprintf("%s obstacle=%s response=%s arm-target=%s · obstacle R=%s · arm R=%s%s", s.ID, obstacle, response, target, economicsNumber(v.ObstacleR), economicsNumber(v.ArmR), mark)
}

type ScenarioEconomicsCounts struct{ Checked, PathEvaluated, PathCoherent, Sub1, RoleWarnings, Contradictions, Corrected, SchemaRefusals uint64 }

var economicsCounts struct {
	sync.Mutex
	values ScenarioEconomicsCounts
}

func ScenarioEconomicsCounters() ScenarioEconomicsCounts {
	economicsCounts.Lock()
	defer economicsCounts.Unlock()
	return economicsCounts.values
}
func ScenarioEconomicsBootLine() string {
	c := ScenarioEconomicsCounters()
	contract, required := "off", "off"
	if scenarioEconomicsRequired {
		contract = "on"
	}
	if scenarioObstacleRequired {
		required = "on"
	}
	return fmt.Sprintf("📐 scenario economics: contract=%s · obstacle-required=%s · target-path-coherent=%d/%d · sub-1R-first-obstacle=%d · role-use-disagreements=%d · contradictions refused=%d · corrected=%d · schema refusals=%d · checked=%d (new-authoring checks since boot; legacy UNKNOWN by design)", contract, required, c.PathCoherent, c.PathEvaluated, c.Sub1, c.RoleWarnings, c.Contradictions, c.Corrected, c.SchemaRefusals, c.Checked)
}

// validateNewScenarioEconomics is called at the existing model-output parser,
// including retries/shadow authoring. It is never called by stored-plan readers.
// minRR is the resolved R:R floor (R4, owner ruling 2026-09-15): a misstated
// r_to_arm_target whose computed value is at or above the floor is auto-corrected
// and accepted; minRR <= 0 keeps the strict contradiction refusal.
func validateNewScenarioEconomics(d *PlanDoc, minRR float64) error {
	var errors []string
	for i := range d.Scenarios {
		s := &d.Scenarios[i]
		issues, contradiction, pathKnown, pathCoherent, corrected := scenarioEconomicsIssues(*s, minRR)
		v := EconomicsFor(*s)
		warnings := scenarioRoleWarnings(*s, d.Levels)
		// Record one complete observation atomically, so a concurrent boot/log
		// reader cannot combine a new numerator with an old denominator.
		economicsCounts.Lock()
		economicsCounts.values.Checked++
		if pathKnown {
			economicsCounts.values.PathEvaluated++
			if pathCoherent {
				economicsCounts.values.PathCoherent++
			}
		}
		if v.Sub1 {
			economicsCounts.values.Sub1++
		}
		economicsCounts.values.RoleWarnings += uint64(len(warnings))
		if len(issues) > 0 {
			if contradiction {
				economicsCounts.values.Contradictions++
			} else {
				economicsCounts.values.SchemaRefusals++
			}
		}
		if corrected {
			economicsCounts.values.Corrected++
		}
		economicsCounts.Unlock()
		if v.Sub1 {
			warnings = append(warnings, "sub-1R first obstacle: fact only; target policy unchanged")
		}
		verdict := "PASS"
		if len(issues) > 0 {
			verdict = "REFUSED"
			errors = append(errors, fmt.Sprintf("%s: %s", s.ID, strings.Join(issues, "; ")))
		} else {
			s.Economics.Version = ScenarioEconomicsContractVersion
		}
		logScenarioEconomics(verdict, *s, issues, warnings)
	}
	if len(errors) > 0 {
		return fmt.Errorf("scenario economics: %s", strings.Join(errors, " | "))
	}
	return nil
}

// Geometry coherence, not target selection. The MNQ tick is read from the
// instrument registry; comparison is in price units, never dimensionless R.
func scenarioEconomicsIssues(s PlanScenario, minRR float64) (issues []string, contradiction, pathKnown, pathCoherent, corrected bool) {
	e := s.Economics
	if e == nil {
		return []string{"economics required for new authoring (including first_obstacle)"}, false, false, false, false
	}
	if len(e.EntryZone) != 2 || !economicsPrice(e.EntryZone[0]) || !economicsPrice(e.EntryZone[1]) || e.EntryZone[0] > e.EntryZone[1] {
		issues = append(issues, "entry_zone requires positive [low, high]")
	}
	if strings.TrimSpace(s.Trigger) == "" || strings.TrimSpace(s.Invalid) == "" || s.Confirm == nil {
		issues = append(issues, "trigger, confirmation and structural invalidation required for new authoring")
	}
	o := e.FirstObstacle
	if scenarioObstacleRequired && (o == nil || o.Price == nil || !economicsPrice(*o.Price) || strings.TrimSpace(o.Level) == "" || strings.TrimSpace(o.Family) == "") {
		issues = append(issues, "first_obstacle price, level and family required for new authoring")
	}
	if o != nil {
		switch o.Response {
		case "pass_through", "reduce", "exit", "decline_setup":
		default:
			issues = append(issues, "first_obstacle response required: pass_through|reduce|exit|decline_setup")
		}
	}
	v := EconomicsFor(s)
	g := v.Geometry
	if g == nil || !economicsPrice(g.Entry) || !economicsPrice(g.Stop) || !economicsPrice(g.Target) || g.Entry == g.Stop {
		issues = append(issues, "entry, protective stop and arm target required with nonzero risk")
		return
	}
	tick := market.FuturesTickSize("MNQ")
	risk := math.Abs(g.Entry - g.Stop)
	for _, r := range []struct {
		name             string
		stated, computed *float64
	}{{"r_to_obstacle", e.RToObstacle, v.ObstacleR}, {"r_to_arm_target", e.RToArmTarget, v.ArmR}} {
		if r.stated == nil || math.IsNaN(*r.stated) || math.IsInf(*r.stated, 0) {
			issues = append(issues, r.name+" required for new authoring")
			continue
		}
		if r.computed != nil && math.Abs(*r.stated-*r.computed)*risk > tick+1e-8 {
			// R4 (owner ruling 2026-09-15): an arm-target R at or above the
			// minimum floor is AUTO-CORRECTED to the machine's computed value
			// and accepted. The machine trusts its own math over the model's
			// rounding; the downstream minimum gates (arm seam, entry gate,
			// structural geometry) still refuse anything below the floor, so
			// no protection is weakened. minRR <= 0 keeps the strict legacy
			// refusal (stored readers / offline validator pass no floor).
			if r.name == "r_to_arm_target" && minRR > 0 && *r.computed >= minRR {
				*r.stated = *r.computed
				corrected = true
				continue
			}
			issues = append(issues, fmt.Sprintf("implied_r %s stated %.6f disagrees with geometry %.6f (entry %.2f stop %.2f; tolerance %.2f price points)", r.name, *r.stated, *r.computed, g.Entry, g.Stop, tick))
			contradiction = true
		}
	}
	pathKnown = true
	for _, p := range s.TargetChain {
		if math.Abs(p-g.Target) <= tick+1e-8 {
			pathCoherent = true
			break
		}
	}
	if !pathCoherent && strings.TrimSpace(e.TargetPathException) == "" {
		issues = append(issues, fmt.Sprintf("target_path arm target %.2f absent from path %v within %.2f; name target_path_exception", g.Target, s.TargetChain, tick))
		contradiction = true
	}
	// The contract explicitly permits a named exception. The coherence counter
	// counts membership OR that declared exception, not inferred path membership.
	if strings.TrimSpace(e.TargetPathException) != "" {
		pathCoherent = true
	}
	if o != nil && o.Price != nil && economicsPrice(*o.Price) {
		beyond := (s.Direction == "long" && *o.Price > g.Target+1e-8) || (s.Direction == "short" && *o.Price < g.Target-1e-8)
		if beyond {
			issues = append(issues, fmt.Sprintf("obstacle_beyond_target obstacle %.2f beyond arm target %.2f", *o.Price, g.Target))
			contradiction = true
		}
	}
	return
}

// Instructions are free prose. Diagnose only explicit, known role categories;
// unknown prose stays unknown. A role difference is NEVER a refusal.
func scenarioRoleWarnings(s PlanScenario, levels []PlanLevel) []string {
	v := EconomicsFor(s)
	if s.Economics == nil || v.Geometry == nil {
		return nil
	}
	var warnings []string
	tick := market.FuturesTickSize("MNQ")
	invalidPrices := triggerNumbers(s.Invalid)
	for _, l := range levels {
		role := strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(l.Instruction), "_", " ")), " ")
		uses := []string{}
		if math.Abs(v.Geometry.Entry-l.Price) <= tick {
			uses = append(uses, "entry")
		}
		target := math.Abs(v.Geometry.Target-l.Price) <= tick
		for _, p := range s.TargetChain {
			target = target || math.Abs(p-l.Price) <= tick
		}
		if target {
			uses = append(uses, "target")
		}
		for _, p := range invalidPrices {
			if math.Abs(p-l.Price) <= tick {
				uses = append(uses, "invalidation")
				break
			}
		}
		for _, use := range uses {
			differs := false
			switch role {
			case "confluence", "confluence only", "confluence reference", "confluence ref", "htf confluence", "htf confluence only", "htf confluence reference only":
				differs = use == "target" || use == "entry"
			case "target", "target only":
				differs = use == "invalidation" || use == "entry"
			case "invalid", "invalidate", "invalidation", "death line":
				differs = use == "entry" || use == "target"
			}
			if !differs {
				continue
			}
			reason := "no explicit exception"
			for _, x := range s.Economics.RoleExceptions {
				if x.Level == l.Label && x.Use == use && strings.TrimSpace(x.Reason) != "" {
					reason = "declared exception: " + x.Reason
					break
				}
			}
			warnings = append(warnings, fmt.Sprintf("%s level %s %.2f role=%s use=%s (%s)", s.ID, l.Label, l.Price, l.Instruction, use, reason))
		}
	}
	return warnings
}

// Telemetry failure must not replace the validation result or panic the loop.
func logScenarioEconomics(verdict string, s PlanScenario, issues, warnings []string) {
	defer func() {
		if recover() != nil {
			fmt.Fprintln(os.Stderr, "WARN scenario economics telemetry unavailable; validation verdict preserved")
		}
	}()
	logger.Infof("📐 scenario economics %s: %s · issues=%v · WARN=%v · %s", verdict, EconomicsSummary(s), issues, warnings, ScenarioEconomicsBootLine())
}
