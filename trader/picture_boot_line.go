package trader

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W5 — the Picture boot lines (READ, never literal; L7) ──────
//
//	🖼  every in-flight Picture ledger row (store.PictureHtfRecoverableAll —
//	    the set the entry latch and the installation gate count), each with
//	    its id, stage and submitted_at, "none" when there is none; and the
//	    CURRENT run epoch (pictureRunEpoch), n/a when no run is alive (D15,
//	    CTO 1790192366762)
//	📷  D8 RULING: while Picture is on and one_setup resolves ON, a WARN that
//	    one_setup governs planner plays only — a Picture scenario is admitted
//	    by its own switch
//
// Printed once per trader run (logPictureBootLines, called where the other
// day-plan boot lines print).

var pictureBootLogged sync.Map // trader id|run epoch → true

// PictureRowsBootLine renders the 🖼 line from the store's own reader and the
// live run epoch. Every field is READ: an unreadable ledger says so, an absent
// submission stamp prints n/a, never 0.
func PictureRowsBootLine(st *store.Store, epoch int64, epochOK bool) string {
	ep := "n/a"
	if epochOK {
		ep = strconv.FormatInt(epoch, 10)
	}
	if st == nil {
		return "🖼 picture rows in flight: n/a (no store) · run_epoch=" + ep
	}
	rows, err := st.PictureHtfRecoverableAll()
	if err != nil {
		return fmt.Sprintf("🖼 picture rows in flight: UNREADABLE (%v) · run_epoch=%s", err, ep)
	}
	if len(rows) == 0 {
		return "🖼 picture rows in flight (the latch's + installation gate's set, every trader): none · run_epoch=" + ep
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		sub := "n/a"
		if r.SubmittedAt > 0 {
			sub = kernel.ClockCTSeconds(time.UnixMilli(r.SubmittedAt))
		}
		parts = append(parts, fmt.Sprintf("#%d %s submitted_at=%s", r.ID, r.Stage, sub))
	}
	return fmt.Sprintf("🖼 picture rows in flight (the latch's + installation gate's set, every trader): %d — %s · run_epoch=%s",
		len(rows), strings.Join(parts, "; "), ep)
}

// pictureOneSetupWarn is the D8 WARN: "" unless Picture is on and one_setup
// resolves ON (nil = ON). The origin letter is the resolver's.
func pictureOneSetupWarn(cfg *store.StrategyConfig, pictureOn bool) string {
	on, _, src, _ := store.ResolveOneSetup(cfg)
	if !pictureOn || !on {
		return ""
	}
	return fmt.Sprintf("📷 one_setup=ON%s governs planner plays only — Picture scenarios are admitted by their own switch (day_plan.picture_htf.enabled): one_setup never declines, retires, ranks or re-targets a P-scenario (W5 D8 ruling)",
		store.OriginLetter(src))
}

// logPictureBootLines prints the two lines once per trader run.
func (at *AutoTrader) logPictureBootLines() {
	if at == nil || at.exchange != "ninjatrader" {
		return
	}
	// Once per trader RUN: a reload is a new run with a new epoch, and the line
	// must print the epoch that run places under.
	epoch, ok := at.pictureRunEpoch()
	if _, done := pictureBootLogged.LoadOrStore(at.id+"|"+pictureEpochText(epoch, ok), true); done {
		return
	}
	at.logInfof("%s", PictureRowsBootLine(at.store, epoch, ok))
	if w := pictureOneSetupWarn(at.GetStrategyConfig(), at.pictureHtfResolvedConfig().Enabled); w != "" {
		at.logWarnf("%s", w)
	}
}
