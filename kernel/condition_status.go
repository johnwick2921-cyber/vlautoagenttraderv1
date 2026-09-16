package kernel

import (
	"os"
	"sort"
	"strings"
)

// SHADOW DEMOTION (0C, owner ruling 2026-08-31) — per-condition live|shadow
// status. A shadowed condition MAY still be authored, validated, and scored by
// the E8 counterfactual logger; it MUST NOT place any order at the arm seam.
//
// Resolution chain (class-8: quote the RESOLVED value, never the file default):
//   per-session override > strategy base > env > defaults
// The defaults shadow exactly the two owner-ruled conditions:
//   fvg_entry       = shadow  (external null ×2 + own null)
//   breakout_retest = shadow  (no evidence anywhere + 80.7% stop-out falsification)
// Everything else = live.

const (
	ConditionLive   = "live"
	ConditionShadow = "shadow"
)

// defaultConditionStatus is the owner-ruled baseline (0C, 2026-08-31).
var defaultConditionStatus = map[string]string{
	"fvg_entry":       ConditionShadow,
	"breakout_retest": ConditionShadow,
}

// ConditionStatus resolves ONE condition's live|shadow status from the full
// chain. Precedence: session override > base > env > defaults. Empty values in
// the maps are ignored. Condition names are case-trimmed.
func ConditionStatus(condition string, base, session map[string]string, env string) string {
	c := strings.ToLower(strings.TrimSpace(condition))
	if c == "" {
		return ConditionLive
	}
	if s, ok := session[c]; ok && (s == ConditionShadow || s == ConditionLive) {
		return s
	}
	if s, ok := base[c]; ok && (s == ConditionShadow || s == ConditionLive) {
		return s
	}
	if env != "" {
		for _, tok := range strings.Split(env, ",") {
			parts := strings.SplitN(strings.TrimSpace(tok), "=", 2)
			name := strings.ToLower(strings.TrimSpace(parts[0]))
			if name != c {
				continue
			}
			if len(parts) == 1 {
				return ConditionShadow // bare name in SHADOW_CONDITIONS
			}
			if parts[1] == ConditionShadow || parts[1] == ConditionLive {
				return parts[1]
			}
		}
	}
	if s, ok := defaultConditionStatus[c]; ok {
		return s
	}
	return ConditionLive
}

// IsConditionShadowed reports whether the resolved status for a condition is
// shadow (the arm seam's single choke point).
func IsConditionShadowed(condition string, base, session map[string]string, env string) bool {
	return ConditionStatus(condition, base, session, env) == ConditionShadow
}

// ShadowConditionsEnv composes the env chain the resolver reads: the SHADOW
// and LIVE condition lists. SHADOW_CONDITIONS=csv adds shadow entries;
// LIVE_CONDITIONS=csv forces live (highest env priority, below config).
func ShadowConditionsEnv() string {
	var parts []string
	if v := strings.TrimSpace(os.Getenv("SHADOW_CONDITIONS")); v != "" {
		for _, tok := range strings.Split(v, ",") {
			t := strings.TrimSpace(tok)
			if t != "" {
				parts = append(parts, t+"="+ConditionShadow)
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("LIVE_CONDITIONS")); v != "" {
		for _, tok := range strings.Split(v, ",") {
			t := strings.TrimSpace(tok)
			if t != "" {
				parts = append(parts, t+"="+ConditionLive)
			}
		}
	}
	return strings.Join(parts, ",")
}

// KnownConditions is the full scenario-condition vocabulary (the enum, plan_doc
// schema comment: reclaim|hold|sweep_reclaim|reject|acceptance|
// breakout_retest|fvg_entry|breakdown_continue|breakup_continue).
func KnownConditions() []string {
	return []string{
		"reclaim", "hold", "sweep_reclaim", "reject", "acceptance",
		"breakout_retest", "fvg_entry", "breakdown_continue", "breakup_continue",
	}
}

// ResolvedConditionStatuses resolves the status of EVERY known condition for
// the given base/session maps + env (nil-safe). Used by the boot line and the
// per-trader boot log so the ledger renders the RESOLVED map, never literals.
func ResolvedConditionStatuses(base, session map[string]string, env string) map[string]string {
	out := make(map[string]string, len(KnownConditions()))
	for _, c := range KnownConditions() {
		out[c] = ConditionStatus(c, base, session, env)
	}
	return out
}

// ConditionStatusLedger renders the resolved status map for the boot line:
//
//	🔬 conditions: live [..] · shadow [..]
//
// Renders from the RESOLVED map (class-8), sorted for stable output.
func ConditionStatusLedger(base, session map[string]string, env string) string {
	statuses := ResolvedConditionStatuses(base, session, env)
	var live, shadow []string
	for _, c := range KnownConditions() {
		if statuses[c] == ConditionShadow {
			shadow = append(shadow, c)
		} else {
			live = append(live, c)
		}
	}
	sort.Strings(live)
	sort.Strings(shadow)
	return "🔬 conditions: live [" + strings.Join(live, ", ") + "] · shadow [" + strings.Join(shadow, ", ") + "]"
}
