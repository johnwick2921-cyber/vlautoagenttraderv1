package telemetry

import "sync/atomic"

// shadowDemotion (0C, owner ruling 2026-08-31) — counts arm placements REFUSED
// because the scenario's condition is shadowed. Class-1: refusals are loud and
// countable; the Sep-9 court reads this alongside the E8 counterfactuals.
var armsRefusedShadowed atomic.Int64

// IncShadowedArmRefusal bumps the counter at the arm-seam refusal site.
func IncShadowedArmRefusal() {
	armsRefusedShadowed.Add(1)
}

// ShadowedArmRefusalCount returns the running count (boot line / UI visibility).
func ShadowedArmRefusalCount() int64 {
	return armsRefusedShadowed.Load()
}
