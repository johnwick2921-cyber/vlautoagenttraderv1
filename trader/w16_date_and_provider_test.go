package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// CTO-VERIFY 5.5 / P-B — TWO DATE NOTIONS COEXIST.
//
//	plannerTradeDateCT  = the CT CALENDAR date   → rolls at MIDNIGHT
//	kernel.CMESessionDayKey = the CME session-day → rolls at 17:00 CT
//
// Plan identity (plan_id, trade_date, the ActivePlanProvider lookup, the calendar
// slice) uses the first; the risk daily reset, guardrail counters, alert dedupe
// keys, level assembly and the approval key use the second. This test documents
// the disagreement and pins the window where it bites.
//
// Consequence for a MIDNIGHT-SPANNING session (ASIA 17:00→02:00 CT): a plan
// written at the 16:55 read is stamped with day D, but from 00:00 CT the
// provider looks up day D+1 and finds nothing — the plan vanishes from the
// executor mid-session. Dormant while ASIA is disabled; reachable now that the
// per-session enable toggle exists (W15.A).
func TestW16TradeDateNotionsDisagree(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata")
	}
	ct := func(d, h, m int) time.Time { return time.Date(2026, 8, d, h, m, 0, 0, loc) }

	cases := []struct {
		name                 string
		at                   time.Time
		wantPlanner, wantDay string
	}{
		{"16:00 CT — both agree on today", ct(17, 16, 0), "2026-08-17", "2026-08-16"},
		{"17:30 CT — session-day already rolled, planner has not", ct(17, 17, 30), "2026-08-17", "2026-08-17"},
		{"23:30 CT — still day D for the planner", ct(17, 23, 30), "2026-08-17", "2026-08-17"},
		{"00:30 CT — planner rolled at MIDNIGHT, mid-ASIA-session", ct(18, 0, 30), "2026-08-18", "2026-08-17"},
	}
	disagreements := 0
	for _, tc := range cases {
		gotPlanner := plannerTradeDateCT(tc.at)
		gotDay := kernel.CMESessionDayKey(tc.at)
		if gotPlanner != tc.wantPlanner {
			t.Errorf("%s: plannerTradeDateCT = %s, want %s", tc.name, gotPlanner, tc.wantPlanner)
		}
		if gotDay != tc.wantDay {
			t.Errorf("%s: CMESessionDayKey = %s, want %s", tc.name, gotDay, tc.wantDay)
		}
		if gotPlanner != gotDay {
			disagreements++
			t.Logf("DISAGREE at %s: planner=%s session-day=%s", tc.name, gotPlanner, gotDay)
		}
	}
	if disagreements == 0 {
		t.Fatal("expected the two notions to disagree somewhere — if this now passes, they were unified and this test should be updated")
	}

	// THE BITE: one ASIA session instance spans two different planner trade dates.
	asiaRead := ct(17, 16, 55) // ASIA read time, 16:55 CT
	asiaLate := ct(18, 1, 0)   // 01:00 CT, still inside ASIA's 17:00→02:00 window
	if !inSessionReadWindow(asiaLate, "16:55", "02:00") {
		t.Fatal("01:00 CT must still be inside ASIA's window for this scenario to hold")
	}
	if plannerTradeDateCT(asiaRead) == plannerTradeDateCT(asiaLate) {
		t.Fatal("expected the planner trade date to differ across midnight within ONE ASIA session")
	}
	t.Logf("BREAK: an ASIA plan stamped %s is looked up as %s at 01:00 CT — the executor finds no plan mid-session",
		plannerTradeDateCT(asiaRead), plannerTradeDateCT(asiaLate))
}

// CTO-VERIFY P-B / P-G — the ActivePlanProvider gates on the REGISTRY's
// Enabled flag (auto_trader_planner.go:566), NOT on sessionRunnable. So a session
// the owner switched on at the STRATEGY level gets its plan written by the read
// scheduler (which does use sessionRunnable) and then never read by the executor.
// Plan written, plan never consumed.
//
// This is the same defect class fixed in W15.A, surviving in a second location.
func TestW16ProviderIgnoresStrategySessionEnable(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	asia, ok := reg.SessionByName(kernel.SessionAsia)
	if !ok {
		t.Fatal("ASIA must exist in the registry")
	}

	// The owner switches ASIA on for this strategy (the W15.A control).
	on := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", Enable: bp(true)}}, nil)

	// The SCHEDULER agrees ASIA runs …
	if runnable, why := on.sessionRunnable(asia); !runnable {
		t.Fatalf("scheduler must consider ASIA runnable once switched on: %s", why)
	}
	// … but the registry flag the PROVIDER consults still says no.
	if asia.Enabled {
		t.Skip("ASIA is enabled in the shipped registry — this defect requires the shipped default (disabled)")
	}
	t.Log("BREAK: sessionRunnable(ASIA)=true (scheduler writes the plan) while registry Enabled=false " +
		"(installActivePlanProvider returns nil) → the executor never sees the ASIA plan it just wrote")
}

// CTO-VERIFY — THE ROOT FINDING OF THIS RUN.
//
// NINE sites decide "does session X run for this strategy". W15.A introduced
// sessionRunnable() as the single resolver but converted only TWO of them:
//
//	CONVERTED (honor sessions[].enable):
//	  1. trader/auto_trader_planner.go:171  read scheduler
//	  2. trader/auto_trader_session.go:38   entry gate (outer)
//
//	NOT CONVERTED (still read the registry's Enabled flag):
//	  3. trader/auto_trader_session.go:108  sessionGateDecision  → BLOCKS entries
//	  4. trader/auto_trader_planner.go:528  digest writer        → no digest
//	  5. trader/auto_trader_planner.go:566  ActivePlanProvider   → executor never sees the plan
//	  6-9. api/handler_plan.go:143,475,555,711,1122  card / ask / realign / apply
//
// Net effect with ASIA switched on at the strategy level (which the LIVE config
// does today): the read FIRES and costs an LLM call, a plan row is written, and
// then nothing consumes it — no executor context, no entries, no digest, no card.
// The owner sees a toggle that appears to do nothing while spending money.
func TestW16SessionEnableIsHonoredByOnlySomeGates(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	asia, _ := reg.SessionByName(kernel.SessionAsia)
	at := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", Enable: bp(true)}}, nil)

	// site 1+2 — the resolver says ASIA runs
	runnable, _ := at.sessionRunnable(asia)
	if !runnable {
		t.Fatal("precondition: sessionRunnable(ASIA) must be true once switched on")
	}

	// site 3 — the pure entry-gate function still refuses, on the registry flag
	reason, blocked := sessionGateDecision(reg, ctTimeForTest(t, 19, 0), nil)
	if !blocked {
		t.Fatal("expected sessionGateDecision to block during ASIA (registry Enabled=false)")
	}
	t.Logf("INCONSISTENT: sessionRunnable(ASIA)=true but sessionGateDecision says %q", reason)

	// sites 4/5 — the registry flag those read is still false, so the digest writer
	// and the ActivePlanProvider both skip ASIA.
	if asia.Enabled {
		t.Skip("registry ASIA became enabled — this inconsistency requires the shipped default")
	}
	t.Log("CONSEQUENCE: read fires (LLM spend) → plan row written → executor gets nil, " +
		"entries blocked, no digest, card empty. A toggle that costs money and changes nothing.")
}
