package trader

import (
	"path/filepath"
	"testing"

	"nofx/store"
)

// TestBootLineReportsTheBOUNDStrategy — the defect this wave shipped and the
// boot of 2026-09-09 18:14 exposed.
//
// bootRiskFacts scanned EVERY strategy row and returned the first carrying
// risk-shaped fields. The live DB holds nine; it read 70695b25 ("New Strategy",
// daily_loss_enabled=true, consecutive_loss_halt=2) — a row bound to no trader
// at all — and printed its knobs as though they governed. The trader's actual
// strategy, a5b7662e, has daily_loss_enabled=false and no halt knob, so the
// line said "daily_loss_enabled on · breaker=2" about a desk where neither was
// true.
//
// The runtime was never wrong: breakerHaltN reads at.config.StrategyConfig, the
// trader's own bound row. Only the boot line lied — on the one line whose whole
// job is to say what is enforced. That is class 82 exactly, shipped in the line
// beside the deskGuardrail fix that was written for it.
//
// ONE SOURCE FOR THE BOUND ROW, BOTH READERS: the boot line must resolve through
// traders.strategy_id, the same row deskGuardrail reads at runtime.
func TestBootLineReportsTheBOUNDStrategy(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "boot.db"))
	if err != nil {
		t.Fatal(err)
	}
	g := st.GormDB()

	// Nine rows with different risk fields — the live shape. The DECOY sorts
	// before the bound row by id, which is exactly how the real one won.
	mk := func(id, cfg string) {
		if err := g.Exec(`INSERT INTO strategies (id, user_id, name, config) VALUES (?,?,?,?)`,
			id, "u1", "s-"+id, cfg).Error; err != nil {
			t.Fatal(err)
		}
	}
	decoy := `{"ai_config":{"risk_control":{"guardrails_enabled":false,"daily_loss_enabled":true,"daily_loss_limit_usd":450,"consecutive_loss_halt":2}}}`
	bound := `{"ai_config":{"risk_control":{"guardrails_enabled":false,"daily_loss_enabled":false,"daily_loss_limit_usd":450}}}`
	mk("00-decoy", decoy)
	for i, blank := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		_ = i
		mk("11-blank-"+blank, `{"ai_config":{}}`)
	}
	mk("99-bound", bound)

	if err := g.Exec(`INSERT INTO traders (id, user_id, name, strategy_id, ai_model_id, exchange_id, initial_balance) VALUES (?,?,?,?,?,?,?)`,
		"trader-1", "u1", "hoang", "99-bound", "m1", "e1", 50000.0).Error; err != nil {
		t.Fatalf("cannot seed the bound trader: %v", err)
	}

	cfg, limit, master, leg, ok := bootRiskFacts(st)
	if !ok {
		t.Fatal("bootRiskFacts could not resolve the bound strategy at all")
	}
	if leg {
		t.Errorf("the boot line reports daily_loss_enabled ON — that is the DECOY row's value. "+
			"The bound strategy has it OFF, and the desk it describes is the bound one. (limit=%.0f master=%v)", limit, master)
	}
	if got := breakerHaltN(cfg); got != breakerHaltDefault {
		t.Errorf("breaker resolves to %d — that is the DECOY's consecutive_loss_halt. The bound row has "+
			"no halt knob, so it must fall to the [I] default %d", got, breakerHaltDefault)
	}
}

// A threshold the owner set that can never fire is REPORTED, never silently
// clamped — he should see what he set.
func TestUnreachableWarnIsReportedNotClamped(t *testing.T) {
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.ConsecutiveLossHalt = 2 // N below the WARN default M=5
	line := SessionRiskBootLine(cfg, 450, false, false, "14:45 CT")
	if !containsAll(line, "UNREACHABLE", "halt=2") {
		t.Fatalf("a WARN above the halt is a dead threshold and the line does not say so:\n  %s", line)
	}
	// And a sane pair says nothing of the sort.
	cfg.RiskControl.ConsecutiveLossHalt = 8
	if line := SessionRiskBootLine(cfg, 450, true, true, "14:45 CT"); containsAll(line, "UNREACHABLE") {
		t.Fatalf("warn 5 under halt 8 is reachable and must not be flagged:\n  %s", line)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, x := range subs {
		found := false
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
