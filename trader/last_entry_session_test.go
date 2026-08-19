package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// PHASE 4 (T1–T4) — the last-entry cutoff is SESSION-scoped, America/Chicago.
//
// The bug these pin against: a single day-scoped 13:00 CT cutoff
// (timeReachedCT(now, "13:00")) stayed true from 13:00 CT to midnight, refusing
// every Asia-evening entry — the live 21:00 CT refusal in incident B.

func ctTimeAt(t *testing.T, date, hhmm string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, loc)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func lastEntryTrader(t *testing.T, overrides []store.DayPlanSessionOverride) *AutoTrader {
	t.Helper()
	at := &AutoTrader{exchange: "ninjatrader"}
	at.config.StrategyConfig = &store.StrategyConfig{
		DayPlan: &store.DayPlanConfig{PlanEnabled: true, Sessions: overrides},
	}
	return at
}

func TestLastEntryCutoffIsSessionScoped(t *testing.T) {
	ny105 := 105 // NY end 14:45 − 105 = 13:00, reproducing the legacy NY cutoff
	at := lastEntryTrader(t, []store.DayPlanSessionOverride{
		{Session: kernel.SessionNY, LastEntryOffsetMin: &ny105},
	})

	for _, tc := range []struct {
		name    string
		now     time.Time
		blocked bool
		want    string // substring of the refusal; "" when not blocked
	}{
		// T1 — 21:00 CT, ASIA session: the exact instant the old gate refused.
		{"T1 asia 21:00 passes", ctTimeAt(t, "2026-08-18", "21:00"), false, ""},
		// T2 — 13:05 CT, NY session, offset 120 → cutoff 13:00 → refused (NY).
		{"T2 ny 13:05 refused", ctTimeAt(t, "2026-08-18", "13:05"), true, "past last-entry 13:00 CT (NY)"},
		// T2b — 12:55, same config: one side of the boundary must pass.
		{"T2b ny 12:55 passes", ctTimeAt(t, "2026-08-18", "12:55"), false, ""},
		// T3 — London mid-session passes; post-cutoff (08:30−15=08:15) refuses, scoped.
		{"T3 london 05:00 passes", ctTimeAt(t, "2026-08-18", "05:00"), false, ""},
		{"T3b london 08:20 refused", ctTimeAt(t, "2026-08-18", "08:20"), true, "(LONDON)"},
		// T4 — 15:30 CT: between NY end (14:45) and ASIA start (17:00). NOT this
		// gate's refusal — the session gate owns it.
		{"T4 between sessions not last-entry", ctTimeAt(t, "2026-08-18", "15:30"), false, ""},
		// ASIA's own cutoff wraps midnight: 02:00 − 15 = 01:45.
		{"asia 01:50 refused wrapped", ctTimeAt(t, "2026-08-19", "01:50"), true, "past last-entry 01:45 CT (ASIA)"},
		{"asia 01:40 passes wrapped", ctTimeAt(t, "2026-08-19", "01:40"), false, ""},
		// DST spring-forward day (2026-03-08): CT wall-clock stays authoritative.
		{"DST asia 21:00 passes", ctTimeAt(t, "2026-03-08", "21:00"), false, ""},
		{"DST asia 01:50 refused", ctTimeAt(t, "2026-03-09", "01:50"), true, "(ASIA)"},
	} {
		reason, blocked := at.entryBlockedByLastEntryAt(tc.now)
		if blocked != tc.blocked {
			t.Errorf("%s: blocked=%v want %v (reason %q)", tc.name, blocked, tc.blocked, reason)
			continue
		}
		if tc.want != "" && !strings.Contains(reason, tc.want) {
			t.Errorf("%s: reason %q must contain %q — the refusal names session + resolved time", tc.name, reason, tc.want)
		}
	}
}

// The default offset is config, not a literal: an unset session resolves 15.
func TestLastEntryOffsetDefaults(t *testing.T) {
	var dp *store.DayPlanConfig
	if got := dp.LastEntryOffsetFor("ASIA"); got != store.DefaultLastEntryOffsetMin {
		t.Fatalf("nil config must resolve the default, got %d", got)
	}
	zero := 0
	cfg := &store.DayPlanConfig{Sessions: []store.DayPlanSessionOverride{
		{Session: "ASIA", LastEntryOffsetMin: &zero},
	}}
	if got := cfg.LastEntryOffsetFor("ASIA"); got != 0 {
		t.Fatalf("an explicit 0 offset is a real choice (cutoff = session end), got %d", got)
	}
}

// CONFIG-TRUTH: the new per-session fields survive the hand-rolled codec.
func TestLastEntryOffsetSurvivesTheCodec(t *testing.T) {
	forty := 40
	cfg := store.StrategyConfig{
		StrategyType: "ai_trading",
		DayPlan: &store.DayPlanConfig{PlanEnabled: true, Sessions: []store.DayPlanSessionOverride{
			{Session: "ASIA", LastEntryOffsetMin: &forty, EODFlatOffsetMin: &forty},
		}},
	}
	blob, err := cfg.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var back store.StrategyConfig
	if err := back.UnmarshalJSON(blob); err != nil {
		t.Fatal(err)
	}
	if got := back.DayPlan.LastEntryOffsetFor("ASIA"); got != 40 {
		t.Fatalf("last_entry_offset_min did not survive the codec round-trip: %d", got)
	}
	if got := back.DayPlan.EODFlatOffsetFor("ASIA"); got != 40 {
		t.Fatalf("eod_flat_offset_min did not survive the codec round-trip: %d", got)
	}
}
