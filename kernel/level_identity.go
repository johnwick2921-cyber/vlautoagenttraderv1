package kernel

import "math"

// W7 — pure level-IDENTITY helper (DB-blind; the trader layer maps this + the
// existing LevelTypeFromLabel onto store.LevelStateDB for cross-session
// persistence). A level's durable identity is (type, price-bin): the same price +
// same type is the SAME level across sessions, so state (times-tested / consumed /
// freshness) accumulates rather than resetting each day. "A level burned in one
// session cannot return fresh in the next."

// LevelBinIndex maps a price to the absolute SVP row grid — floor(price / rowHeight)
// — identical to kernel/svp.go svpBinIndex so a level lands in the same row
// regardless of which session reads it.
func LevelBinIndex(price float64) int {
	return int(math.Floor(price / SVPRowHeight))
}
