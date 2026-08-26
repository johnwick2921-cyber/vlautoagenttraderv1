package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

func htfsnap(trend string, swing *SwingRef, events []StructureEvent) map[string]StructureState {
	return map[string]StructureState{"1h": {Trend: trend, Swing: swing, LastEvents: events}}
}

func TestHTFVeto_OpposedRefusedWithMessage(t *testing.T) {
	snap := htfsnap("TRENDING_UP", &SwingRef{Kind: "HH", Price: 29470.25, TimeMs: hourMs(4, 45)},
		[]StructureEvent{{Type: "BOS", Dir: "up", Price: 29470.25, TimeMs: hourMs(4, 45)}})
	blocked, msg := HTFVetoVerdict(snap, "open_short", "1h")
	if !blocked {
		t.Fatalf("short vs 1h TRENDING_UP must be refused")
	}
	if !strings.HasPrefix(msg, "htf_veto: short vs 1h TRENDING_UP") || !strings.Contains(msg, "BOS-up 29470.25") {
		t.Fatalf("bad refusal message: %q", msg)
	}
}

func TestHTFVeto_AlignedAndRangingPass(t *testing.T) {
	up := htfsnap("TRENDING_UP", &SwingRef{Kind: "HH", Price: 29470.25, TimeMs: hourMs(4, 45)}, nil)
	if blocked, _ := HTFVetoVerdict(up, "open_long", "1h"); blocked {
		t.Fatalf("long aligned with TRENDING_UP must pass")
	}
	down := htfsnap("TRENDING_DOWN", &SwingRef{Kind: "LL", Price: 29220, TimeMs: hourMs(7, 0)}, nil)
	if blocked, _ := HTFVetoVerdict(down, "open_short", "1h"); blocked {
		t.Fatalf("short aligned with TRENDING_DOWN must pass")
	}
	rng := htfsnap("RANGING", nil, nil)
	if blocked, _ := HTFVetoVerdict(rng, "open_short", "1h"); blocked {
		t.Fatalf("RANGING never vetoes")
	}
	if blocked, _ := HTFVetoVerdict(rng, "open_long", "1h"); blocked {
		t.Fatalf("RANGING never vetoes (long)")
	}
}

func TestHTFVeto_MissingSnapshotFailsOpen(t *testing.T) {
	if blocked, _ := HTFVetoVerdict(nil, "open_short", "1h"); blocked {
		t.Fatalf("detector-unavailable must fail open")
	}
	if blocked, _ := HTFVetoVerdict(map[string]StructureState{}, "open_long", "1h"); blocked {
		t.Fatalf("empty snapshot must fail open")
	}
}

// hourBars0821 is the real 1h table from the 2026-08-21 stored prompt
// (rec 31587) — the series G1's veto replays against.
func hourBars0821() []market.KlineBar {
	rows := []struct {
		d, h         int
		o, hi, lo, c float64
	}{
		{20, 19, 29293.75, 29365.00, 29290.50, 29321.50},
		{20, 20, 29320.75, 29373.00, 29291.75, 29353.25},
		{20, 21, 29352.75, 29399.75, 29342.75, 29384.50},
		{20, 22, 29384.50, 29388.00, 29341.75, 29352.25},
		{20, 23, 29352.75, 29373.50, 29321.25, 29358.25},
		{21, 0, 29358.50, 29381.00, 29340.00, 29366.25},
		{21, 1, 29366.00, 29402.25, 29354.25, 29387.25},
		{21, 2, 29386.25, 29456.00, 29380.25, 29446.50},
		{21, 3, 29446.50, 29468.00, 29390.50, 29453.25},
		{21, 4, 29453.50, 29492.00, 29410.75, 29485.25},
		{21, 5, 29485.50, 29515.50, 29470.00, 29479.00},
		{21, 6, 29479.75, 29539.75, 29452.75, 29499.25},
		{21, 7, 29499.50, 29533.75, 29472.00, 29511.50},
		{21, 8, 29511.75, 29516.25, 29257.25, 29303.75},
		{21, 9, 29303.50, 29335.75, 29220.25, 29331.75},
		{21, 10, 29332.00, 29488.50, 29326.25, 29443.75},
		{21, 11, 29443.25, 29454.50, 29380.50, 29412.25},
		{21, 12, 29412.00, 29423.75, 29375.00, 29383.50},
		{21, 13, 29383.25, 29405.50, 29353.75, 29400.25},
		{21, 14, 29399.75, 29433.25, 29380.00, 29411.50},
	}
	kb := make([]market.KlineBar, len(rows))
	for i, r := range rows {
		kb[i] = market.KlineBar{Time: time.Date(2026, 8, r.d, r.h, 0, 0, 0, CTLocation()).UnixMilli(),
			Open: r.o, High: r.hi, Low: r.lo, Close: r.c}
	}
	return kb
}

// TestG1Replay_ShiftDayVetoes replays the shift-day's 12 positions through the
// veto with the REAL 1h bars. HONEST TRUTH (pinned, not wished for): on
// 2026-08-21 the 1h series never holds four confirmed fractal swings at any
// entry instant — evening entries see 2 swings (H 29399.75 → L 29321.25), the
// post-crash morning 3, and the full day ends mixed (up-pair highs, down-pair
// lows) → the 3-swing standard reads RANGING everywhere → the veto blocks
// NOTHING on this day (Σ blocked = 0). That is fail-open working as designed:
// the day's damage came from LTF structure, which is G4's transition stand-down
// territory — the veto must never veto on an unconfirmed HTF. The mechanics
// (opposed vs confirmed blocks, aligned/RANGING passes) are pinned by the unit
// tests above; this test pins the day's truth and guards the winners.
func TestG1Replay_ShiftDayVetoes(t *testing.T) {
	bars := hourBars0821()
	type pos struct {
		id   int
		side string // LONG|SHORT
		at   int64  // entry instant
		pnl  float64
	}
	day := []pos{
		{533, "SHORT", time.Date(2026, 8, 20, 17, 59, 18, 0, CTLocation()).UnixMilli(), -54.5},
		{534, "SHORT", time.Date(2026, 8, 20, 19, 22, 28, 0, CTLocation()).UnixMilli(), -66.0},
		{535, "SHORT", time.Date(2026, 8, 20, 21, 8, 43, 0, CTLocation()).UnixMilli(), -31.0},
		{536, "SHORT", time.Date(2026, 8, 20, 21, 36, 19, 0, CTLocation()).UnixMilli(), 124.0},
		{537, "SHORT", time.Date(2026, 8, 21, 0, 25, 10, 0, CTLocation()).UnixMilli(), -79.5},
		{538, "SHORT", time.Date(2026, 8, 21, 1, 40, 26, 0, CTLocation()).UnixMilli(), 30.5},
		{539, "SHORT", time.Date(2026, 8, 21, 3, 54, 9, 0, CTLocation()).UnixMilli(), -84.5},
		{540, "LONG", time.Date(2026, 8, 21, 5, 4, 40, 0, CTLocation()).UnixMilli(), -98.0},
		{541, "LONG", time.Date(2026, 8, 21, 6, 26, 47, 0, CTLocation()).UnixMilli(), -83.0},
		{542, "SHORT", time.Date(2026, 8, 21, 8, 49, 3, 0, CTLocation()).UnixMilli(), 0.0},
		{543, "SHORT", time.Date(2026, 8, 21, 10, 12, 43, 0, CTLocation()).UnixMilli(), -88.5},
		{544, "SHORT", time.Date(2026, 8, 21, 10, 47, 45, 0, CTLocation()).UnixMilli(), -61.5},
	}
	veto := func(side string, at int64) (bool, string) {
		action := "open_long"
		if side == "SHORT" {
			action = "open_short"
		}
		st := ComputeStructureState(bars, 60, 0, at)
		return HTFVetoVerdict(map[string]StructureState{"1h": st}, action, "1h")
	}
	var blockedIDs []int
	var blockedPnL float64
	for _, p := range day {
		b, msg := veto(p.side, p.at)
		if b {
			blockedIDs = append(blockedIDs, p.id)
			blockedPnL += p.pnl
			t.Logf("BLOCKED %d %s pnl=%+.2f — %s", p.id, p.side, p.pnl, msg)
		} else {
			t.Logf("pass     %d %s pnl=%+.2f", p.id, p.side, p.pnl)
		}
	}
	t.Logf("blocked count=%d Σpnl(blocked)=%+.2f", len(blockedIDs), blockedPnL)
	// Pin the honest day: the 1h detector never confirms a trend at any entry
	// instant → zero vetoes (fail-open, never a guess).
	if len(blockedIDs) != 0 {
		t.Fatalf("replay: 08-21 1h structure must fail open everywhere (no confirmed trend at any entry), got blocked=%v", blockedIDs)
	}
	// The 1h state at the late-morning entries is RANGING (3 swings, no pair).
	for _, probe := range []int64{
		time.Date(2026, 8, 21, 8, 49, 3, 0, CTLocation()).UnixMilli(),
		time.Date(2026, 8, 21, 10, 47, 45, 0, CTLocation()).UnixMilli(),
	} {
		if st := ComputeStructureState(bars, 60, 0, probe); st.Trend != "RANGING" {
			t.Fatalf("late-morning 1h state must be RANGING, got %s", st.Trend)
		}
	}
}
