package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func dpWith(sessions []store.DayPlanSessionOverride, enabledSubset []string) *AutoTrader {
	sc := store.StrategyConfig{
		Indicators: store.IndicatorConfig{Klines: store.KlineConfig{PrimaryTimeframe: "5m"}},
		DayPlan: &store.DayPlanConfig{
			PlanEnabled: true, Sessions: sessions, SessionsEnabled: enabledSubset,
		},
	}
	return &AutoTrader{
		id: "t1", exchange: "ninjatrader",
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &sc},
	}
}

func bp(b bool) *bool { return &b }

// PART A — the session enable resolver. This is what made the toggle real: before,
// the read scheduler ANDed the HARDCODED registry flag in, so switching ASIA on at
// the strategy level could never take effect.
func TestW15SessionRunnable(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	ny, _ := reg.SessionByName(kernel.SessionNY)
	asia, _ := reg.SessionByName(kernel.SessionAsia)
	london, _ := reg.SessionByName(kernel.SessionLondon)

	// ── defaults: NY on, ASIA/LONDON OFF (they earn enablement)
	def := dpWith(nil, nil)
	if ok, _ := def.sessionRunnable(ny); !ok {
		t.Fatal("NY must run by default")
	}
	for _, s := range []*kernel.SessionDef{asia, london} {
		if ok, why := def.sessionRunnable(s); ok {
			t.Fatalf("%s must default OFF, got runnable (%s)", s.Name, why)
		}
	}

	// ── THE BUG: turning ASIA on must actually take effect, even though the
	// hardcoded registry says ASIA.Enabled=false.
	on := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", Enable: bp(true)}}, nil)
	if ok, why := on.sessionRunnable(asia); !ok {
		t.Fatalf("an explicit ASIA toggle must WIN over the registry default; got %q", why)
	}

	// ── and turning a registry-enabled session OFF must also take effect
	off := dpWith([]store.DayPlanSessionOverride{{Session: "NY", Enable: bp(false)}}, nil)
	if ok, _ := off.sessionRunnable(ny); ok {
		t.Fatal("an explicit NY=off toggle must switch NY off for this strategy")
	}
	if _, why := off.sessionRunnable(ny); why == "" {
		t.Fatal("a refusal must carry a reason (it is logged at the gate)")
	}

	// ── no explicit override → inherit registry AND the subset
	inherit := dpWith(nil, []string{"NY", "ASIA"})
	if ok, _ := inherit.sessionRunnable(asia); ok {
		t.Fatal("sessions_enabled alone must NOT override a registry-disabled session")
	}
	if ok, _ := inherit.sessionRunnable(ny); !ok {
		t.Fatal("NY in the subset + registry-enabled must run")
	}
	// subset that excludes NY turns it off (inherit path)
	sub := dpWith(nil, []string{"LONDON"})
	if ok, _ := sub.sessionRunnable(ny); ok {
		t.Fatal("NY excluded from sessions_enabled must not run")
	}

	// ── nil session is never runnable (defensive)
	if ok, _ := def.sessionRunnable(nil); ok {
		t.Fatal("nil session must not be runnable")
	}
}

// PART A — enabling a session must NOT retroactively create plans for reads whose
// window already passed. The scheduler's read window is the guard: sessionRunnable
// only decides IF a session participates, never WHEN.
func TestW15EnableDoesNotBackfillPastReads(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	asia, _ := reg.SessionByName(kernel.SessionAsia)

	// ASIA is switched on mid-afternoon (13:00 CT) — long after its 16:55 read time
	// and outside its 17:00–02:00 window.
	at := dpWith([]store.DayPlanSessionOverride{{Session: "ASIA", Enable: bp(true)}}, nil)
	if ok, _ := at.sessionRunnable(asia); !ok {
		t.Fatal("ASIA should be runnable once switched on")
	}
	// but the read window still says no at 13:00 CT
	ct := ctTimeForTest(t, 13, 0)
	if inSessionReadWindow(ct, asia.ReadCT, asia.WindowEndCT) {
		t.Fatal("13:00 CT is outside ASIA's read window — enabling must not backfill a past read")
	}
	// and it DOES fire once its own window comes around
	inWin := ctTimeForTest(t, 18, 0)
	if !inSessionReadWindow(inWin, asia.ReadCT, asia.WindowEndCT) {
		t.Fatal("18:00 CT is inside ASIA's window — the next read should fire there")
	}
}

// ctTimeForTest builds a CT wall-clock time on a fixed weekday (Mon 2026-08-17).
func ctTimeForTest(t *testing.T, hh, mm int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata")
	}
	return time.Date(2026, 8, 17, hh, mm, 0, 0, loc)
}
