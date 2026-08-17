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
}

// CTO-VERIFY P-B / P-G — THE ROOT FINDING OF THIS RUN, NOW FIXED (H8).
//
// NINE sites decide "does session X run for this strategy". W15.A introduced
// sessionRunnable() as the single resolver but converted only TWO of them. The
// deciding trading sites that still read the registry's Enabled flag — the pure
// entry gate (sessionGateDecision) and the ActivePlanProvider — are now converted
// too, so an explicitly-enabled session is honored end to end: the read fires,
// the entry gate allows, AND the executor receives the plan.
func TestW16SessionEnableIsHonoredByEveryGatingSite(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	asia, _ := reg.SessionByName(kernel.SessionAsia)

	// The owner switches ASIA on for this strategy (the W15.A control), while the
	// registry default still says ASIA.Enabled=false.
	on := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", Enable: bp(true)}}, nil)

	// The resolver agrees ASIA runs.
	if runnable, why := on.sessionRunnable(asia); !runnable {
		t.Fatalf("scheduler must consider ASIA runnable once switched on: %s", why)
	}

	// Site: the pure entry gate — now takes the resolver and must ALLOW the entry
	// (not block on the registry flag). 18:00 CT is inside ASIA's window and clear
	// of first-5m / lunch / red-news.
	reason, blocked := sessionGateDecision(reg, ctTimeForTest(t, 18, 0), nil, on.sessionRunnable)
	if blocked {
		t.Fatalf("entry gate must honor sessionRunnable(ASIA)=true, got blocked=%q", reason)
	}

	// Registry-disabled + no override must STILL block (the resolver inherits the
	// registry default as the DEFAULT, not a veto when the owner stays silent).
	off := dpWith(nil, nil)
	reason, blocked = sessionGateDecision(reg, ctTimeForTest(t, 18, 0), nil, off.sessionRunnable)
	if !blocked || reason == "" {
		t.Fatalf("ASIA with no override must still block (registry default), got blocked=%v reason=%q", blocked, reason)
	}
}