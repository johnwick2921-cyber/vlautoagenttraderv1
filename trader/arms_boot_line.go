// D6 (arms-follow-bias, 2026-09-04) — one line stating what this binary does
// about arms. READ from the same resolvers the code uses, never a literal: a
// boot line that restates its own source cannot report a change (A24).

package trader

import (
	"fmt"

	"nofx/kernel"
)

// ArmsBootLine renders the arms posture for the boot log.
func ArmsBootLine() string {
	// Quoted from the table, so a condition losing its stop-entry kind changes
	// the line rather than leaving it lying.
	stopEntry := "off"
	if kernel.ArmKindFor("reclaim") == kernel.ArmKindStopEntry {
		stopEntry = "on(reclaim)"
	}
	return fmt.Sprintf("🎯 arms: bias-coherent=warn · stop-entry=%s · far-arm counter=on(%.1f×ATR5m) · ledger append-only=on",
		stopEntry, farArmThreshold())
}
