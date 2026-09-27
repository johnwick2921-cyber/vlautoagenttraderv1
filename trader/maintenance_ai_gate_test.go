package trader

import (
	"fmt"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
	"nofx/telemetry"
)

// W-ONE-BUTTON M2 site 1 — the AI entry path, AT THE PRODUCTION CALL SITE
// (executeDecisionWithRecord), not a rebuilt predicate.
//
// The fixture has no broker (at.trader == nil), so an entry that gets PAST the
// maintenance gate panics further down (reconcile/positions). The helper
// records that as "went past the gate" — exactly what a missing gate does. A
// refusal returns cleanly with the maintenance_hold record.
func runDecision(at *AutoTrader, action string) (rec *store.DecisionAction, err error, pastGate bool) {
	rec = &store.DecisionAction{Action: action, Symbol: "MNQ"}
	defer func() {
		if r := recover(); r != nil {
			pastGate = true
			err = fmt.Errorf("panicked past the gate: %v", r)
		}
	}()
	err = at.executeDecisionWithRecord(&kernel.Decision{Action: action, Symbol: "MNQ"}, rec)
	return rec, err, !strings.HasPrefix(rec.Error, "maintenance_hold:")
}

func gateBlocks(traderID, gate string) int {
	_, table := telemetry.GateBlockSnapshot()
	return table[traderID][gate]
}

func TestAIEntryRefusedAtExecuteDecisionWhileHeld(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-ai")
	at, _ := pauseTrader(t)
	at.id = "maint-ai-1"
	for _, action := range []string{"open_long", "open_short"} {
		before := gateBlocks(at.id, "maintenance_hold")
		rec, err, past := runDecision(at, action)
		if past {
			t.Fatalf("%s went past the maintenance gate (err=%v, rec.Error=%q)", action, err, rec.Error)
		}
		if err != nil || rec.Success || !strings.Contains(rec.Error, "job-ai") {
			t.Fatalf("%s: refusal must return nil, Success=false, Error naming the job; got err=%v rec=%+v", action, err, rec)
		}
		if gateBlocks(at.id, "maintenance_hold") != before+1 {
			t.Fatalf("%s: telemetry maintenance_hold gate block not counted", action)
		}
	}
}

// Closes are position management: the hold never refuses them.
func TestAICloseIsNotRefusedByTheHold(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-ai")
	at, _ := pauseTrader(t)
	at.id = "maint-ai-2"
	for _, action := range []string{"close_long", "close_short"} {
		rec, err, past := runDecision(at, action)
		if !past {
			t.Fatalf("%s was refused by the maintenance hold (rec.Error=%q, err=%v) — closes must stay open", action, rec.Error, err)
		}
	}
}

// Absent file = today's behaviour: the entry is not refused by this gate.
func TestAIEntryNotRefusedWithoutAHold(t *testing.T) {
	withMaintenanceDir(t)
	at, _ := pauseTrader(t)
	at.id = "maint-ai-3"
	if rec, _, past := runDecision(at, "open_long"); !past {
		t.Fatalf("no hold file: the maintenance gate must not refuse (rec.Error=%q)", rec.Error)
	}
}

// Gate order: an owner pause refusal still NAMES the pause (it ranks first),
// and the maintenance block sits between stop_until and contract-roll. The
// source slice the pause test reads must not gain close tokens.
func TestMaintenanceGateSitsAfterTheOwnerPause(t *testing.T) {
	src := readOrdersSource(t)
	pause := strings.Index(src, "⏸ stop_until: %s %s REFUSED")
	maint := strings.Index(src, "🔒 maintenance hold: %s %s REFUSED")
	roll := strings.Index(src, "P3 (ledger-close 2026-08-19) — CONTRACT-ROLL gate")
	if pause < 0 || maint < 0 || roll < 0 || !(pause < maint && maint < roll) {
		t.Fatalf("order must be stop_until (%d) < maintenance (%d) < contract-roll (%d)", pause, maint, roll)
	}
	block := src[maint:roll]
	if strings.Contains(block, "close_long") || strings.Contains(block, "close_short") {
		t.Fatal("the maintenance gate must never match close actions")
	}
}
