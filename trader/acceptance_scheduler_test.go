package trader

// ACCEPTANCE GATE — Part B scheduler receipt. Sweeps a real clock minute-by-
// minute through the LIVE registry's own gate predicates (the exact boolean
// chain maybeRunSessionReads evaluates at trader/auto_trader_planner.go:167)
// and reports the first minute a NY planner read can fire. Pure + offline: no
// store, no network, no paid call — safe in the normal suite.

import (
	"testing"
	"time"

	"nofx/kernel"
)

func TestAcceptanceSchedulerNextFire(t *testing.T) {
	ct, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}
	// The gate session's wall clock: Sat 2026-08-15 19:55 CT.
	from := time.Date(2026, 8, 15, 19, 55, 0, 0, ct)

	// The LIVE registry: system_config has no `session_registry` key, so
	// loadStoredRegistry falls back to kernel.DefaultSessionRegistry (NY only).
	reg := kernel.DefaultSessionRegistry()

	var ny kernel.SessionDef
	for _, s := range reg.Sessions {
		if s.Name == kernel.SessionNY {
			ny = s
		}
		t.Logf("registry: %-6s enabled=%-5v window %s→%s read %s flat %s",
			s.Name, s.Enabled, s.WindowStartCT, s.WindowEndCT, s.ReadCT, s.FlatCT)
	}
	if !ny.Enabled || ny.ReadCT != "08:25" {
		t.Fatalf("NY session unexpected: enabled=%v read=%s", ny.Enabled, ny.ReadCT)
	}

	// Walk minute by minute, applying the SAME two clock predicates the
	// scheduler applies (IsCMEOpen && inSessionReadWindow). sessions_enabled
	// (["NY"]) and s.Enabled are config, already asserted above.
	var firstFire time.Time
	var sundayFires []time.Time
	for i := 0; i < 5*24*60; i++ {
		now := from.Add(time.Duration(i) * time.Minute)
		open := kernel.IsCMEOpen(now)
		inWin := inSessionReadWindow(now, ny.ReadCT, ny.WindowEndCT)
		if open && inWin {
			if firstFire.IsZero() {
				firstFire = now
			}
			if now.Weekday() == time.Sunday {
				sundayFires = append(sundayFires, now)
			}
		}
	}

	if firstFire.IsZero() {
		t.Fatal("no NY read fires in the next 5 days — scheduler would never run")
	}
	t.Logf("FIRST NY READ FIRES: %s", firstFire.Format("Mon 2006-01-02 15:04:05 MST"))

	want := time.Date(2026, 8, 17, 8, 25, 0, 0, ct)
	if !firstFire.Equal(want) {
		t.Errorf("next-fire mismatch: got %s want %s", firstFire, want)
	}
	if len(sundayFires) != 0 {
		t.Errorf("W1 REGRESSION: %d Sunday NY read minutes (first %s) — the Sunday-17:00 read is back",
			len(sundayFires), sundayFires[0])
	} else {
		t.Log("W1 PASS: zero Sunday NY read minutes (17:00 Sunday is past NY's 15:00 window end)")
	}

	// The arithmetic, spelled out for the report.
	t.Logf("arithmetic: now=Sat 19:55 CT → IsCMEOpen(Sat)=false (cme_calendar.go:26-27)")
	t.Logf("            Sun 17:00+ → IsCMEOpen=true (line 28-29) BUT inSessionReadWindow(1020min ≥ 900=15:00 end)=false")
	t.Logf("            Mon 08:25 → IsCMEOpen(Mon, hour!=16)=true AND read=505min ≤ 505 < 900 → FIRE")
	t.Logf("            plan key = (trade_date 2026-08-17, session NY); GetLatestPlanForSession=nil → runPlannerRead")
}

// TestAcceptanceEODFlatClock proves the 13:00 last-entry / 14:45 eod-flat
// boundaries the gate table depends on, from the same registry.
func TestAcceptanceEODFlatClock(t *testing.T) {
	ct, _ := time.LoadLocation("America/Chicago")
	reg := kernel.DefaultSessionRegistry()
	var ny kernel.SessionDef
	for _, s := range reg.Sessions {
		if s.Name == kernel.SessionNY {
			ny = s
		}
	}
	if ny.FlatCT != "14:45" {
		t.Errorf("NY flat is %s, owner-confirmed value is 14:45 CT (=15:45 ET)", ny.FlatCT)
	}
	// no half-days registered → effective flat == configured flat
	monday := time.Date(2026, 8, 17, 12, 0, 0, 0, ct)
	key := kernel.CMESessionDayKey(monday)
	eff := effectiveEODFlatCT(reg, key, ny.FlatCT)
	t.Logf("session-day %s: configured flat %s → effective %s (half-days registered: %d)", key, ny.FlatCT, eff, len(reg.HalfDays))
	if eff != ny.FlatCT {
		t.Errorf("effective flat %s != configured %s with no half-day registered", eff, ny.FlatCT)
	}
	for _, kz := range ny.Killzones {
		t.Logf("killzone %-6s %s→%s", kz.Name, kz.StartCT, kz.EndCT)
	}
}
