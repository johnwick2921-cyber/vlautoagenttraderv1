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
//   session override > strategy override > LIVE_CONDITIONS > SHADOW_CONDITIONS > defaults
// (W1: a name in BOTH env lists resolves LIVE — the order is decided here, not
// by the order the env string happens to list its tokens in.)
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
// chain. Precedence: session override > strategy override (base) >
// LIVE_CONDITIONS > SHADOW_CONDITIONS > defaults. Within env, a live entry for
// the name outranks a shadow entry wherever each appears in the string. Empty
// values in the maps are ignored. Condition names are case-trimmed.
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
		sawLive, sawShadow := false, false
		for _, tok := range strings.Split(env, ",") {
			parts := strings.SplitN(strings.TrimSpace(tok), "=", 2)
			name := strings.ToLower(strings.TrimSpace(parts[0]))
			if name != c {
				continue
			}
			switch {
			case len(parts) == 1:
				sawShadow = true // bare name in SHADOW_CONDITIONS
			case parts[1] == ConditionLive:
				sawLive = true
			case parts[1] == ConditionShadow:
				sawShadow = true
			}
		}
		if sawLive {
			return ConditionLive
		}
		if sawShadow {
			return ConditionShadow
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

// ShadowConditionsEnv composes the env chain the resolver reads, in precedence
// order: LIVE_CONDITIONS=csv forces live (the highest env priority, below
// config), then SHADOW_CONDITIONS=csv adds shadow entries. ConditionStatus
// ranks live over shadow itself, so the order here is for the reader.
func ShadowConditionsEnv() string {
	var parts []string
	if v := strings.TrimSpace(os.Getenv("LIVE_CONDITIONS")); v != "" {
		for _, tok := range strings.Split(v, ",") {
			t := strings.TrimSpace(tok)
			if t != "" {
				parts = append(parts, t+"="+ConditionLive)
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("SHADOW_CONDITIONS")); v != "" {
		for _, tok := range strings.Split(v, ",") {
			t := strings.TrimSpace(tok)
			if t != "" {
				parts = append(parts, t+"="+ConditionShadow)
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
