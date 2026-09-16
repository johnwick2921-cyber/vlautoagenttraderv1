package trader

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// E2 — the breaker's whole contract in one table.
func TestSessionRiskAdjudication(t *testing.T) {
	const N, M = 8, 5
	cases := []struct {
		name       string
		losses     int
		band       string
		wantRefuse bool
		wantWarn   bool
		wantClass  string
	}{
		{"clean session", 0, "", false, false, ""},
		{"below the warn", 4, "", false, false, ""},
		{"at the warn — counted, never refused", 5, "", false, true, ""},
		{"one below the halt — still only a warn", 7, "", false, true, ""},
		{"at the halt", 8, "", true, false, "consecutive_loss"},
		{"past the halt", 12, "", true, false, "consecutive_loss"},
		// The band is a fact about the clock and wins over the record.
		{"inside the no-trade band", 0, "lunch 12:00-13:00 CT", true, false, "no_trade_band"},
		{"band wins over a halt", 9, "lunch 12:00-13:00 CT", true, false, "no_trade_band"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := adjudicateSessionRisk(c.losses, N, M, c.band)
			if v.Refuse != c.wantRefuse {
				t.Fatalf("refuse=%v want %v (why: %s)", v.Refuse, c.wantRefuse, v.Reason)
			}
			if v.Warn != c.wantWarn {
				t.Fatalf("warn=%v want %v (why: %s)", v.Warn, c.wantWarn, v.Reason)
			}
			if c.wantClass != "" && v.Class != c.wantClass {
				t.Fatalf("class=%q want %q", v.Class, c.wantClass)
			}
			if (v.Refuse || v.Warn) && v.Reason == "" {
				t.Fatal("a refusal or a warning with no reason — the owner cannot act on a silent gate")
			}
		})
	}
}

// N=0 disables the breaker entirely; the band is unaffected.
func TestBreakerDisabledByZero(t *testing.T) {
	if v := adjudicateSessionRisk(99, 0, 5, ""); v.Refuse {
		t.Fatal("N=0 must disable the halt outright")
	}
	if v := adjudicateSessionRisk(99, 0, 5, "lunch"); !v.Refuse || v.Class != "no_trade_band" {
		t.Fatal("N=0 must not disable the no-trade band")
	}
}

// E2 — N resolves from the knob when the owner set one, else the [I] default.
func TestBreakerThresholdResolves(t *testing.T) {
	if got := breakerHaltN(nil); got != breakerHaltDefault {
		t.Fatalf("unset → N=%d, want the [I] default %d", got, breakerHaltDefault)
	}
	cfg := &store.StrategyConfig{}
	cfg.RiskControl.ConsecutiveLossHalt = 3
	if got := breakerHaltN(cfg); got != 3 {
		t.Fatalf("owner knob 3 → N=%d, want 3 (the owner's value always wins)", got)
	}
	t.Setenv("BREAKER_HALT_N", "0")
	if got := breakerHaltN(nil); got != 0 {
		t.Fatalf("BREAKER_HALT_N=0 → N=%d, want 0 (an explicit off switch must exist)", got)
	}
	_ = os.Unsetenv("BREAKER_HALT_N")
}

// E2 — the run is scoped to the CME session-day, so it clears at the roll and
// not at some wall-clock midnight.
func TestBreakerClearsAtTheCMERoll(t *testing.T) {
	before := time.Date(2026, 9, 8, 16, 30, 0, 0, chicagoLoc()) // before the 17:00 roll
	after := time.Date(2026, 9, 8, 17, 30, 0, 0, chicagoLoc())  // after it
	if kernel.CMESessionDayStart(before).Equal(kernel.CMESessionDayStart(after)) {
		t.Fatal("the 17:00 CT roll did not start a new session-day — the breaker would carry a run across it")
	}
}

// E3 — the cool-down COUNTS and never refuses.
func TestPostLossCoolDownIsACounterNotAGate(t *testing.T) {
	if !sameMergedLevel(29546.25, 29546.25, 0.25) {
		t.Fatal("an exact level match must count as the same level")
	}
	if sameMergedLevel(29546.25, 29546.75, 0.25) {
		t.Fatal("two ticks apart is a different level")
	}
	// The adjudicator has no post-loss branch at all: the counter cannot refuse
	// because there is nowhere for it to say so.
	if v := adjudicateSessionRisk(0, 8, 5, ""); v.Refuse || v.Warn {
		t.Fatal("a clean session must pass — the post-loss counter must never reach the verdict")
	}
	if !store.PostLossWindow(time.Now().Add(-10*time.Minute).UnixMilli(), time.Now(), 30) {
		t.Fatal("10 minutes must fall inside a 30-minute window")
	}
	if store.PostLossWindow(time.Now().Add(-45*time.Minute).UnixMilli(), time.Now(), 30) {
		t.Fatal("45 minutes must fall outside a 30-minute window")
	}
}

// E4 — a daily_force_flat refusal is counted under a class that EXISTS.
// vet-06's fourth hole: leg D refuses with this text and armRefusalClass had no
// case, so every such refusal was tallied as "other".
func TestDailyForceFlatHasItsOwnRefusalClass(t *testing.T) {
	legD := "entry_gate: refused: daily_force_flat — daily loss $470.00 >= limit $450.00 " +
		"(new entries blocked on the arm path until the daily window resets; open positions are not closed by this leg)"
	if got := armRefusalClass(legD); got != "daily_force_flat" {
		t.Fatalf("armRefusalClass(leg D) = %q, want \"daily_force_flat\" — a refusal tallied as %q is a "+
			"refusal nobody can count", got, got)
	}
	for verdict, want := range map[string]string{
		"consecutive_loss_halt: 8 consecutive losing trades this session-day (limit 8)": "consecutive_loss",
		"no_trade_band: lunch 12:00-13:00 CT":                                           "no_trade_band",
		"R:R 1.59 below arm min 2.00":                                                   "rr",
	} {
		if got := armRefusalClass(verdict); got != want {
			t.Errorf("armRefusalClass(%q) = %q, want %q", verdict, got, want)
		}
	}
}

// THE BAND IS CONSULTED BY THE PATH THAT TRADES.
//
// Source pin, because the defect is an ABSENCE: armed_executor.go held zero
// references to the band, and no runtime assertion can observe a call that was
// never written. Under plan_mode=strict the arm path is the only entry, so a
// band enforced only on the decision path is enforced nowhere that matters.
func TestArmPathConsultsTheSessionRiskGate(t *testing.T) {
	b, err := os.ReadFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	fn := strings.Index(src, "func (at *AutoTrader) maybeManageArmedOrdersAt(")
	if fn < 0 {
		t.Fatal("maybeManageArmedOrdersAt not found — this pin has lost its subject")
	}
	loop := strings.Index(src[fn:], "for _, sc := range kernel.OneSetupOrder(doc.Scenarios, osCycle.allowed()) {")
	gate := strings.Index(src[fn:], "at.sessionRiskGateAt(now)")
	if gate < 0 {
		t.Fatal("the ARM path does not consult the session-risk gate — the no-trade band and the " +
			"consecutive-loss breaker guard only the decision path, which plan_mode=strict forbids " +
			"from entering at all")
	}
	if loop >= 0 && gate > loop {
		t.Fatalf("the session-risk gate is consulted AFTER the scenario loop begins (gate@%d > loop@%d) — "+
			"an arm would be authored before the session was asked whether it may trade", gate, loop)
	}
}
