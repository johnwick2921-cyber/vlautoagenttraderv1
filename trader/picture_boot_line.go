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
//	    its id, stage and submitted_at, "none" when there is none; PLUS every
//	    non-terminal armed_orders row with source=picture (since W5 a live
//	    Picture order lives there — U4, #193 N1): id, state, signal short-id,
//	    "none" when none, never 0; and the CURRENT run epoch (pictureRunEpoch),
//	    n/a when no run is alive (D15, CTO 1790192366762)
//	📷  D8 RULING: while Picture is on and one_setup resolves ON, a WARN that
//	    one_setup governs planner plays only — a Picture scenario is admitted
//	    by its own switch
//
// Printed once per trader run (logPictureBootLines, called where the other
// day-plan boot lines print).

var pictureBootLogged sync.Map // trader id|run epoch → true

// PictureRowsBootLine renders the 🖼 line from the store's own readers and the
// live run epoch. Every field is READ: an unreadable ledger or armed set says
// so, an absent submission stamp or signal prints n/a, never 0.
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
	ledger := "none"
	if len(rows) > 0 {
		parts := make([]string, 0, len(rows))
		for _, r := range rows {
			sub := "n/a"
			if r.SubmittedAt > 0 {
				sub = kernel.ClockCTSeconds(time.UnixMilli(r.SubmittedAt))
			}
			parts = append(parts, fmt.Sprintf("#%d %s submitted_at=%s", r.ID, r.Stage, sub))
		}
		ledger = fmt.Sprintf("%d — %s", len(rows), strings.Join(parts, "; "))
	}
	return fmt.Sprintf("🖼 picture rows in flight (the latch's + installation gate's set, every trader): %s · armed picture orders: %s · run_epoch=%s",
		ledger, pictureArmedBootSegment(st), ep)
}

// pictureArmedBootSegment renders the armed_orders source=picture part of the
// 🖼 line: id, state and signal short-id for every non-terminal armed row with
// Source=ArmSourcePicture, "none" when there is none, UNREADABLE when the
// store refuses, never 0 (U4, #193 N1).
func pictureArmedBootSegment(st *store.Store) string {
	armed, err := st.ArmedOrders().ListNonTerminalAllTraders()
	if err != nil {
		return fmt.Sprintf("UNREADABLE (%v)", err)
	}
	pics := make([]string, 0)
	for _, a := range armed {
		if a.Source != store.ArmSourcePicture {
			continue
		}
		sig := "n/a"
		if a.SignalID != "" {
			sig = shortID(a.SignalID)
		}
		pics = append(pics, fmt.Sprintf("#%d %s signal=%s", a.ID, a.State, sig))
	}
	if len(pics) == 0 {
		return "none"
	}
	return fmt.Sprintf("%d — %s", len(pics), strings.Join(pics, "; "))
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
