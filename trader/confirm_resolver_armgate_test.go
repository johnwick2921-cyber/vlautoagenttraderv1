package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W-EXEC-TRUTH W2 — the arm gate, the recorder and the desk read the ONE
// resolver: a breakdown pullback arm whose scenario STORES 2x5m_close is not
// armed after one close + a retest (the defect read BD_MIN_CLOSES=1 and armed
// it), is armed after two closes + a retest that fails, and the arm log names
// the rule it actually gated on. Production call sites: maybeManageArmedOrdersAt,
// recordScenarioStateAt, DeskStripAt.
func TestConfirmResolverBreakdownArmStoredTwoCloses(t *testing.T) {
	t.Setenv("BD_MIN_CLOSES", "1") // the live env [A] — never consulted for a stored rule
	at, st, plan, tape, base := confirmationTrader(t)
	logBuf := captureTraderLog(t)
	plan.Doc.Bias.Direction = "short"
	sc := &plan.Doc.Scenarios[0]
	sc.Condition, sc.Direction, sc.Fvg = "breakdown_continue", "short", nil
	sc.Trigger, sc.Invalid = "2x5m close below 100.00 PDL, sell the failed retest", "a 5m close above 100.00"
	sc.Breakdown = &kernel.PlanBreakdownContinue{Level: 100, LevelLabel: "PDL", EntryMode: "pullback"}
	sc.Confirm = &kernel.PlanConfirm{Rule: "2x5m_close", RefPrice: 100, Side: "below"}
	sc.Arm = &kernel.PlanArmSpec{Enabled: true, Entry: 100.25, Stop: 104, Target: 90, WaitConfirm: true}
	sc.TargetChain = []float64{90}

	// Bucket 10:00–10:05: closes below, no reach back (close #1).
	// Bucket 10:05–10:10: reaches back to 100.5 and closes below (close #2 + a retest).
	// Bucket 10:10–10:15: reaches back again and closes below (the retest that fails AFTER leg 1).
	*tape = nil
	for i := 0; i < 15; i++ {
		o := base.Add(time.Duration(i) * time.Minute).UnixMilli()
		hi := 99.5
		if i >= 5 {
			hi = 100.5
		}
		*tape = append(*tape, market.Kline{OpenTime: o, CloseTime: o + 59999, Open: 99, High: hi, Low: 98, Close: 99})
	}
	all := append([]market.Kline{}, (*tape)...)
	cut := func(n int) { *tape = append([]market.Kline{}, all[:n]...) }

	meta := func(now time.Time) kernel.ConfirmVerdict {
		t.Helper()
		at.recordScenarioStateAt(now)
		raw, err := st.GetSystemConfig(store.ScenarioMetaKey(at.id, plan.PlanID, plan.Version))
		if err != nil {
			t.Fatal(err)
		}
		var m struct {
			Confirm map[string]kernel.ConfirmVerdict `json:"confirm"`
		}
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatal(err)
		}
		return m.Confirm["S1"]
	}
	desk := func(now time.Time) string {
		t.Helper()
		for _, l := range at.DeskStripAt(now).Lines {
			if l.Key == "confirmation" {
				return l.Text
			}
		}
		t.Fatal("desk is missing the recorded confirmation row")
		return ""
	}

	// 10:10 — two closes exist, but the retest sits INSIDE the second close's
	// bucket: leg 1 completes at 10:10, so no retest has failed after it yet.
	cut(10)
	now := base.Add(10 * time.Minute)
	at.maybeManageArmedOrdersAt(nil, now)
	if rows, err := st.ArmedOrders().ListNonTerminal(at.id); err != nil || len(rows) != 0 {
		t.Fatalf("stored 2x5m_close: no arm after one close + the retest (the defect armed here): rows=%d err=%v", len(rows), err)
	}
	v := meta(now)
	if v.Met || v.Outcome != "NOT MET" || v.Rule != "2x5m_close" || v.RuleSource != kernel.ConfirmSourceStored || len(v.Legs) != 2 || !v.Legs[0].Met || v.Legs[1].Met {
		t.Fatalf("recorded meta at 10:10: leg 1 MET on the stored rule, overall NOT MET: %+v", v)
	}
	if d := desk(now); !strings.Contains(d, "S1 NOT MET") {
		t.Fatalf("desk at 10:10 must read the recorded NOT MET: %s", d)
	}

	// 10:15 — the 10:10–10:15 retest fails to reclaim: both legs MET → armed.
	cut(15)
	now = base.Add(15 * time.Minute)
	at.maybeManageArmedOrdersAt(nil, now)
	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil || len(rows) != 1 {
		t.Fatalf("stored 2x5m_close: armed after two closes + a failed retest: rows=%d err=%v\nlog:\n%s", len(rows), err, logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "wait_confirm MET (leg 1 2x5m_close [stored] → leg 2 retest_fail) — arming") {
		t.Fatalf("the arm log must name the rule it gated on:\n%s", logBuf.String())
	}
	if v := meta(now); !v.Met || v.Outcome != "MET" || v.Rule != "2x5m_close" || v.RuleSource != kernel.ConfirmSourceStored {
		t.Fatalf("recorded meta at 10:15: overall MET on the stored rule: %+v", v)
	}
	if d := desk(now); !strings.Contains(d, "S1 MET") {
		t.Fatalf("desk at 10:15 must read the recorded MET: %s", d)
	}
}
