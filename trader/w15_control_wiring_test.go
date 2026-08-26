package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

// W15.B — the per-session acceptance_rule was persisted by the FE, rendered in
// the accordion, and read by NOTHING: every consumer (level-state writer, plan-
// death check, executor prompt) went straight to the strategy-level field. One
// resolver now serves all of them.
func TestW15AcceptanceRuleResolution(t *testing.T) {
	cases := []struct {
		name    string
		dp      *store.DayPlanConfig
		session string
		want    string
	}{
		{"nil config → shipped default", nil, "NY", "2x5m"},
		{"empty strategy field → default", &store.DayPlanConfig{}, "NY", "2x5m"},
		{"strategy level wins over default",
			&store.DayPlanConfig{AcceptanceRule: "15m-close"}, "NY", "15m-close"},
		{"per-session override wins over strategy level",
			&store.DayPlanConfig{
				AcceptanceRule: "2x5m",
				Sessions:       []store.DayPlanSessionOverride{{Session: "NY", AcceptanceRule: sp("15m-close")}},
			}, "NY", "15m-close"},
		{"another session is unaffected by NY's override",
			&store.DayPlanConfig{
				AcceptanceRule: "2x5m",
				Sessions:       []store.DayPlanSessionOverride{{Session: "NY", AcceptanceRule: sp("15m-close")}},
			}, "ASIA", "2x5m"},
		{"empty override string does NOT blank the rule",
			&store.DayPlanConfig{
				AcceptanceRule: "15m-close",
				Sessions:       []store.DayPlanSessionOverride{{Session: "NY", AcceptanceRule: sp("  ")}},
			}, "NY", "15m-close"},
		{"session name match is case-insensitive",
			&store.DayPlanConfig{
				Sessions: []store.DayPlanSessionOverride{{Session: "ny", AcceptanceRule: sp("15m-close")}},
			}, "NY", "15m-close"},
		{"no active session (night) → strategy level",
			&store.DayPlanConfig{AcceptanceRule: "15m-close"}, "", "15m-close"},
	}
	for _, tc := range cases {
		if got := tc.dp.AcceptanceRuleFor(tc.session); got != tc.want {
			t.Errorf("%s: AcceptanceRuleFor(%q) = %q, want %q", tc.name, tc.session, got, tc.want)
		}
	}
}

// The trader-side reader must agree with the store resolver — it is the same
// function, and this pins that it stays that way.
func TestW15TraderAcceptanceMatchesStore(t *testing.T) {
	at := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", AcceptanceRule: sp("15m-close")}}, nil)
	at.config.StrategyConfig.DayPlan.AcceptanceRule = "2x5m"
	if got := at.acceptanceRuleFor("ASIA"); got != "15m-close" {
		t.Fatalf("trader resolver ignored the ASIA override: %q", got)
	}
	if got := at.acceptanceRuleFor("NY"); got != "2x5m" {
		t.Fatalf("NY should inherit the strategy rule, got %q", got)
	}
}

// W15.B — replan_cap + plan_mode resolvers moved into the store so the API card
// resolves them the SAME way the gates do. Behavior must be unchanged.
func TestW15ReplanAndModeResolution(t *testing.T) {
	dp := &store.DayPlanConfig{
		ReplanCap: 4, PlanMode: "direction",
		Sessions: []store.DayPlanSessionOverride{
			{Session: "ASIA", ReplanCap: ip(0), PlanMode: sp("strict")},
		},
	}
	if got := dp.ReplanCapFor("NY"); got != 4 {
		t.Errorf("NY replan cap = %d, want 4 (strategy level)", got)
	}
	if got := dp.ReplanCapFor("ASIA"); got != 0 {
		t.Errorf("ASIA replan cap = %d, want 0 — an explicit ZERO must be honored, not treated as unset", got)
	}
	if got := dp.PlanModeFor("ASIA"); got != "strict" {
		t.Errorf("ASIA mode = %q, want strict", got)
	}
	if got := dp.PlanModeFor("NY"); got != "direction" {
		t.Errorf("NY mode = %q, want direction", got)
	}
	var nilDP *store.DayPlanConfig
	if got := nilDP.ReplanCapFor("NY"); got != 2 {
		t.Errorf("nil config replan cap = %d, want the shipped 2", got)
	}
	if got := nilDP.PlanModeFor("NY"); got != "advisory" {
		t.Errorf("nil config mode = %q, want advisory", got)
	}
}

// W15.B — per-session max_trades. It was a rendered control enforced by NOTHING.
func TestW15MaxTradesResolution(t *testing.T) {
	dp := &store.DayPlanConfig{
		Sessions: []store.DayPlanSessionOverride{
			{Session: "NY", MaxTrades: ip(3)},
			{Session: "ASIA", MaxTrades: ip(0)},
		},
	}
	if n, ok := dp.MaxTradesFor("NY"); !ok || n != 3 {
		t.Errorf("NY cap = (%d,%v), want (3,true)", n, ok)
	}
	if n, ok := dp.MaxTradesFor("ASIA"); !ok || n != 0 {
		t.Errorf("ASIA cap = (%d,%v), want (0,true) — zero means NO entries, not 'unset'", n, ok)
	}
	if _, ok := dp.MaxTradesFor("LONDON"); ok {
		t.Error("LONDON has no override — must report NO cap so shipped behavior is unchanged")
	}
	var nilDP *store.DayPlanConfig
	if _, ok := nilDP.MaxTradesFor("NY"); ok {
		t.Error("nil config must report no cap")
	}
}

// The session window start must be the CURRENTLY-RUNNING instance. For a
// midnight-spanning session, "today at 17:00" is in the FUTURE at 01:00 CT, so a
// naive computation would count zero trades and the cap would never bite.
func TestW15SessionWindowStartWrapsMidnight(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	asia, _ := reg.SessionByName(kernel.SessionAsia)
	ny, _ := reg.SessionByName(kernel.SessionNY)

	// 01:00 CT Tuesday — inside ASIA, whose window opened 17:00 MONDAY.
	at1 := ctTimeForTest(t, 1, 0)
	start, ok := sessionWindowStart(asia, at1)
	if !ok {
		t.Fatal("expected a resolvable ASIA window start")
	}
	if !start.Before(at1) {
		t.Fatalf("ASIA start %v must precede now %v", start, at1)
	}
	if d := at1.Sub(start); d < 7*time.Hour || d > 9*time.Hour {
		t.Fatalf("ASIA started %v ago; expected ~8h (17:00 → 01:00)", d)
	}

	// NY at 10:00 CT started 08:30 the SAME day.
	at2 := ctTimeForTest(t, 10, 0)
	nyStart, ok := sessionWindowStart(ny, at2)
	if !ok {
		t.Fatal("expected a resolvable NY window start")
	}
	if d := at2.Sub(nyStart); d < time.Hour || d > 2*time.Hour {
		t.Fatalf("NY started %v ago; expected ~1.5h (08:30 → 10:00)", d)
	}
}

// A trader with no cap configured must not even reach the counting query — the
// shipped path stays byte-identical (and works with a nil store, as here).
func TestW15TradeCapNoConfigNeverBlocks(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	ny, _ := reg.SessionByName(kernel.SessionNY)
	at := dpWith(nil, nil)
	if why, blocked := at.sessionTradeCapBlocked(ny, ctTimeForTest(t, 10, 0)); blocked {
		t.Fatalf("no cap configured must never block, got %q", why)
	}
	// nil session is defensive-safe too
	if _, blocked := at.sessionTradeCapBlocked(nil, time.Now()); blocked {
		t.Fatal("nil session must not block")
	}
}
