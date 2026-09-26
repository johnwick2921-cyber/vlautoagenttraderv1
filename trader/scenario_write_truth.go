package trader

import (
	"fmt"
	"strings"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W2 A3 + A4 — the write-site seam. The kernel judges
// (kernel.CheckScenarioWriteTruth); this file records and reports. The planner
// write loop calls scenarioWriteTruth ONCE per attempt, right after the
// born-dead check; shadowVerdictFor calls the same kernel check with no
// recording (the shadow writes nothing).

// writeTruthCounterKey is the recorded counter key: trader + class, never
// price text (canon: counters record, never infer).
func writeTruthCounterKey(traderID, what string) string {
	return "scenario_write_truth:" + traderID + ":" + what
}

// scenarioWriteTruth runs A3 + A4 on a freshly authored document, records the
// verdict, and returns the refusal the retry loop reads (nil = admitted).
// facts.IdentityMap nil → UNKNOWN: nothing is judged or counted.
func (at *AutoTrader) scenarioWriteTruth(d *kernel.PlanDoc, facts kernel.PlanFacts) error {
	v := kernel.CheckScenarioWriteTruth(d, facts.IdentityMap, facts.CapacityCut, market.FuturesTickSize(at.futuresSymbol()))
	if !v.IdentityChecked {
		at.logInfof("🪪📐 write truth: identity map UNKNOWN (facts carry none) — identity≠price and obstacle chain NOT judged for this attempt")
		return nil
	}
	for _, n := range v.Normalized {
		at.logWarnf("📐 tick-normalized at write: %s (recorded; never a refusal)", n)
	}
	if at.store != nil {
		at.incWriteTruth("checked")
		for range v.Normalized {
			at.incWriteTruth("tick_normalized")
		}
		for _, is := range v.Issues {
			at.incWriteTruth("refused:" + is.Class)
		}
	}
	return v.Err()
}

func (at *AutoTrader) incWriteTruth(what string) {
	if _, err := store.IncSystemCounter(at.store, writeTruthCounterKey(at.id, what)); err != nil {
		at.logWarnf("🪪📐 write-truth counter %s write failed: %v", what, err)
	}
}

// writeTruthBootLine renders the recorded write-time verdict counts. Before
// the first recorded check every value prints n/a (a count that was never
// taken is not a zero).
func writeTruthBootLine(st *store.Store, traderID string) string {
	head := "🪪📐 scenario write truth (A3 identity=price · A4 obstacle chain; refusals, not warnings)"
	na := func() string {
		parts := make([]string, 0, len(kernel.WriteTruthClasses()))
		for _, c := range kernel.WriteTruthClasses() {
			parts = append(parts, c+"=n/a")
		}
		return head + ": checked=n/a · refused " + strings.Join(parts, " ") + " · tick-normalized=n/a (no write-time check recorded yet)"
	}
	if st == nil {
		return na()
	}
	read := func(what string) (int, error) { return store.SystemCounter(st, writeTruthCounterKey(traderID, what)) }
	checked, err := read("checked")
	if err != nil || checked == 0 {
		return na()
	}
	parts := make([]string, 0, len(kernel.WriteTruthClasses()))
	for _, c := range kernel.WriteTruthClasses() {
		n, err := read("refused:" + c)
		if err != nil {
			parts = append(parts, c+"=n/a")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%d", c, n))
	}
	norm := "n/a"
	if n, err := read("tick_normalized"); err == nil {
		norm = fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s: checked=%d · refused %s · tick-normalized=%s (recorded per authoring attempt, retries included — not plans)", head, checked, strings.Join(parts, " "), norm)
}
