// ── W2 — THE PRODUCTION CALL PATH FOR THE STAMP ──────────────────────────────
//
// One wrapper, one call site (detector_record.go), registered in the A29 gate
// by ITS OWN NAME — W1 proved that registering only the inner function lets
// an unwired wrapper satisfy the gate for everything inside it.
package trader

import (
	"time"

	"nofx/kernel"
	"nofx/store"
)

// stampFadePermissionAtOpen evaluates the label with the clock set to the
// episode's OPEN and writes it once. A fault here must never stop the loop
// (A10/class 23): it is a label, and the recover() below is pinned.
func (at *AutoTrader) stampFadePermissionAtOpen(id uint, symbol string, openedAt time.Time, price float64, seated []kernel.ScoredLevel, direction string) {
	defer func() {
		if r := recover(); r != nil {
			at.logWarnf("🚦 fade permission: stamp recovered from panic on episode %d: %v (label skipped, loop continues)", id, r)
		}
	}()
	if at == nil || at.store == nil || id == 0 {
		return
	}
	facts := at.fadeFactsAt(openedAt, symbol, price, seated, direction)
	v := kernel.FadePermissionAt(openedAt, facts)

	stamp := store.FadeStamp{Evaluated: v.Evaluated, Permitted: v.Permitted, AtMs: openedAt.UnixMilli()}
	if len(v.Exclusions) > 0 {
		stamp.Measured = map[string]map[string]float64{}
		for _, ex := range v.Exclusions {
			stamp.Exclusions = append(stamp.Exclusions, ex.Name)
			stamp.Measured[ex.Name] = map[string]float64{
				"measured": ex.Measured, "threshold": ex.Threshold, "n": float64(ex.SampleN),
			}
		}
	}
	if err := at.store.TouchOutcomes().StampFadePermission(id, stamp); err != nil {
		at.logWarnf("🚦 fade permission: stamp failed on episode %d: %v", id, err)
		return
	}
	// A9 LOUD: every verdict names what fired, the resolved threshold and the
	// measured value. Permitted verdicts log at debug volume; exclusions at
	// info, because an exclusion is the thing E3 will be counting.
	if !v.Evaluated {
		return
	}
	if v.Permitted {
		return
	}
	for _, ex := range v.Exclusions {
		at.logInfof("🚦 fade EXCLUDED (label only, arm unaffected): episode %d %s %.2f — %s measured=%.2f threshold=%.2f %s%s",
			id, direction, price, ex.Name, ex.Measured, ex.Threshold, ex.Basis, fadeNoteSuffix(ex.Name))
	}
}

func fadeNoteSuffix(name string) string {
	if n, ok := kernel.FadeExclusionCoverageNote[name]; ok {
		return " · " + n
	}
	return ""
}
