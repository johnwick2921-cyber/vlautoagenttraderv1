package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// P0 timezone (owner rule 2026-08-19) — CT is canonical everywhere. These
// tests pin the RENDERED clock: the prompt clock line and every window
// bound share one labelled zone, DST-correct across both 2026 transitions,
// and a non-CT host cannot change the output.

func ctDate(y int, mo time.Month, d, hh, mm int) time.Time {
	return time.Date(y, mo, d, hh, mm, 0, 0, CTLocation())
}

// TestTZClockHelpersDSTCorrect — the SAME CT wall-clock renders the same
// "HH:MM CT" on CST and CDT days, and the UTC mirror shifts by exactly 1h.
func TestTZClockHelpersDSTCorrect(t *testing.T) {
	// CST day (before spring-forward) and CDT day (after) — both 07:06 CT.
	cst := ctDate(2026, 1, 15, 7, 6)
	cdt := ctDate(2026, 7, 15, 7, 6)
	if got := ClockCT(cst); got != "07:06 CT" {
		t.Fatalf("ClockCT(CST 07:06) = %q", got)
	}
	if got := ClockCT(cdt); got != "07:06 CT" {
		t.Fatalf("ClockCT(CDT 07:06) = %q", got)
	}
	if got, want := ClockCTAndUTC(cst), "07:06 CT (13:06 UTC)"; got != want {
		t.Fatalf("ClockCTAndUTC(CST) = %q want %q", got, want)
	}
	if got, want := ClockCTAndUTC(cdt), "07:06 CT (12:06 UTC)"; got != want {
		t.Fatalf("ClockCTAndUTC(CDT) = %q want %q", got, want)
	}
	// Fall-back: 12:30 CT on both sides of 2026-11-01.
	fallCDT := ctDate(2026, 10, 31, 12, 30)
	fallCST := ctDate(2026, 11, 2, 12, 30)
	if ClockCT(fallCDT) != "12:30 CT" || ClockCT(fallCST) != "12:30 CT" {
		t.Fatalf("fall-back ClockCT: %q / %q", ClockCT(fallCDT), ClockCT(fallCST))
	}
	if got, want := ClockCTAndUTC(fallCDT), "12:30 CT (17:30 UTC)"; got != want {
		t.Fatalf("fall-back CDT UTC mirror = %q want %q", got, want)
	}
	if got, want := ClockCTAndUTC(fallCST), "12:30 CT (18:30 UTC)"; got != want {
		t.Fatalf("fall-back CST UTC mirror = %q want %q", got, want)
	}
}

// TestTZPromptAt0706CTDoesNotClaimLunch — the P0 live bug: at 07:06 CT the
// prompt's clock and the lunch window must live in ONE labelled zone, so
// the model cannot read "12:06" off a UTC clock and apply the CT lunch
// numbers to it. 07:06 CT is NOT inside 12:00–13:30 CT.
func TestTZPromptAt0706CTDoesNotClaimLunch(t *testing.T) {
	now := ctDate(2026, 7, 15, 7, 6) // CDT — UTC mirror 12:06
	p := BuildPlannerPrompt(PlannerInput{
		TradeDate: "2026-07-15",
		Session:   "ASIA",
		Now:       now,
		ReadKind:  "closed-market 16:55 CT read (from stored data)",
		Price:     30100, DATR: 55,
	})
	if !strings.Contains(p, "clock 07:06 CT (12:06 UTC)") {
		t.Fatalf("planner prompt must carry the labelled dual clock, got:\n%s", p[:400])
	}
	if !strings.Contains(p, "12:00–13:30 CT") {
		t.Fatalf("lunch window must be CT-labelled in the planner prompt")
	}
	// The prompt must never present the unlabelled lunch numbers that let
	// the model pair CT bounds with a UTC clock.
	if strings.Contains(p, "12:00-13:30 lunch") && !strings.Contains(p, "12:00-13:30 CT lunch") {
		t.Fatalf("unlabelled lunch window found in planner prompt")
	}
}

// TestTZClockLineInjectedIntoBothPrompts — the executor clock context line
// renders in BOTH the crypto and futures system prompts, labelled CT.
func TestTZClockLineInjectedIntoBothPrompts(t *testing.T) {
	e := NewStrategyEngine(&store.StrategyConfig{})
	line := "## Clock\n" + ClockCTAndUTC(ctDate(2026, 7, 15, 7, 6)) + " — ALL times in this prompt are CT (America/Chicago), including every session/window bound. Never apply CT window numbers to a UTC clock."
	e.SetClockContext(line)
	if p := e.BuildSystemPrompt(50000, "default", "BTCUSDT"); !strings.Contains(p, "07:06 CT (12:06 UTC)") {
		t.Fatalf("crypto system prompt missing labelled clock")
	}
	if p := e.BuildSystemPrompt(50000, "futures", "MNQ"); !strings.Contains(p, "07:06 CT (12:06 UTC)") {
		t.Fatalf("futures system prompt missing labelled clock")
	}
	// Without the context line the prompts are byte-identical to before
	// (goldens) — nothing injected.
	e.SetClockContext("")
	if p := e.BuildSystemPrompt(50000, "futures", "MNQ"); strings.Contains(p, "## Clock") {
		t.Fatalf("empty clock context must inject nothing")
	}
}

// TestTZNonCTHostStillRendersCT — the partner case: render on a host whose
// local zone is Tokyo/UTC; the output is unchanged because every helper
// pins America/Chicago explicitly.
func TestTZNonCTHostStillRendersCT(t *testing.T) {
	fixed := ctDate(2026, 7, 15, 7, 6)
	want := "07:06 CT (12:06 UTC)"
	for _, zone := range []string{"Asia/Tokyo", "UTC", "Europe/Berlin"} {
		t.Setenv("TZ", zone)
		if got := ClockCTAndUTC(fixed); got != want {
			t.Fatalf("host TZ=%s rendered %q, want %q", zone, got, want)
		}
	}
}

// TestTZTableTimeCT — table rows render CT under the labelled header.
func TestTZTableTimeCT(t *testing.T) {
	// A UTC bar instant 2026-07-15 12:06 UTC = 07:06 CDT.
	bar := time.Unix(1752581160, 0).UTC()
	if got := TableTimeCT(bar); got != "07-15 07:06" {
		t.Fatalf("TableTimeCT = %q want %q", got, "07-15 07:06")
	}
}
