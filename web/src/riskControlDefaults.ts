// UI fallbacks for Risk Control knobs — the values the runtime RESOLVES when
// the strategy leaves a knob unset. The component and its tests read THESE
// constants, never a second literal: changing the runtime default is one edit
// here plus the Go source cited below (and its tests), not a search-and-hope.
//
// Go sources (kept in sync by the RiskControlEditor tests + go tests):
// - max_positions: store/strategy.go ClampLimits floors an unset (<1) value
//   at 1; the MaxPositions=3 const is the CEILING, not the default.
// - trailing_arm_points: store.RiskControlConfig.TrailingArmPoints is 0 when
//   unset (json omitempty); trader/auto_trader_trailing.go:58 reads it raw,
//   and it is used iff trailing_arm === 'after_trigger_points'.
// - max_contracts_per_order: kernel.ResolveMaxContractsWithSource(0, venue 2)
//   is capped by kernel.StageAContractCapDefault = 1
//   (kernel/risk_limits.go:446) → the effective unset value is 1.
export const RISK_DEFAULT_MAX_POSITIONS = 1
export const RISK_DEFAULT_TRAILING_ARM_POINTS = 0
export const RISK_DEFAULT_MAX_CONTRACTS_PER_ORDER = 1
