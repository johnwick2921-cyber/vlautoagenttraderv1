package kernel

import (
	"testing"
	"time"
)

func ctAt(hh, mm int) time.Time {
	return time.Date(2026, 9, 2, hh, mm, 0, 0, CTLocation())
}

// THE RULING'S TEST: the gate, the grader and the card must return the SAME
// window for the same clock. Before this the grader carried its own copies of
// "12:00"/"13:30" and its own first-5m helper, so it could score against
// windows the gate did not enforce.
func TestNoTradeWindowsOneDefinitionForGateGraderCard(t *testing.T) {
	ls, le := LunchWindowCT()
	n := FirstNoTradeMinutes()

	// The CARD's payload, built from the same definitions.
	sess := SessionDef{Name: SessionNY, WindowStartCT: "08:30", WindowEndCT: "14:45"}
	wins := BuildMachineNoTradeWindows(sess)
	var lunch, firstN *NoTradeWindow
	for i := range wins {
		switch wins[i].Kind {
		case KindLunch:
			lunch = &wins[i]
		case KindFirstN:
			firstN = &wins[i]
		}
	}
	if lunch == nil || firstN == nil {
		t.Fatalf("card payload missing a kind: %+v", wins)
	}
	if HHMM(lunch.StartMin) != ls || HHMM(lunch.EndMin) != le {
		t.Fatalf("card lunch %s–%s ≠ definition %s–%s", HHMM(lunch.StartMin), HHMM(lunch.EndMin), ls, le)
	}
	if firstN.EndMin-firstN.StartMin != n {
		t.Fatalf("card first-N spans %d min, definition says %d", firstN.EndMin-firstN.StartMin, n)
	}

	// The GRADER and the CARD must agree at every minute across both windows.
	for _, tc := range []struct {
		at   time.Time
		want bool
		why  string
	}{
		{ctAt(8, 30), true, "first minute after the open"},
		{ctAt(8, 34), true, "last minute of the first-N window"},
		{ctAt(8, 35), false, "one minute past first-N"},
		{ctAt(12, 0), true, "lunch open"},
		{ctAt(13, 29), true, "lunch last minute"},
		{ctAt(13, 30), false, "lunch close is exclusive"},
		{ctAt(11, 59), false, "one minute before lunch"},
	} {
		graderInNoTrade := InLunchNoTrade(tc.at) || InFirstNoTradeMinutes(sess.WindowStartCT, tc.at)
		cardInNoTrade := false
		cur := tc.at.In(CTLocation()).Hour()*60 + tc.at.In(CTLocation()).Minute()
		for _, w := range wins {
			if cur >= w.StartMin && cur < w.EndMin {
				cardInNoTrade = true
			}
		}
		if graderInNoTrade != tc.want || cardInNoTrade != tc.want {
			t.Errorf("%s at %s: grader=%v card=%v want %v",
				tc.why, tc.at.Format("15:04"), graderInNoTrade, cardInNoTrade, tc.want)
		}
	}
}

// Every machine window is stamped with an honest source. The two fixed windows
// say code-constant so the surface never implies a knob that does not exist.
func TestNoTradeWindowsCarryHonestSource(t *testing.T) {
	wins := BuildMachineNoTradeWindows(SessionDef{Name: SessionAsia, WindowStartCT: "17:00", WindowEndCT: "02:00"})
	for _, w := range wins {
		if w.Source != SourceCodeConstant {
			t.Errorf("%s claims source %q — neither window is owner-configurable yet", w.Kind, w.Source)
		}
		if w.Label == "" {
			t.Errorf("%s has no label", w.Kind)
		}
	}
	// A wrapping session's first-N must not wrap incorrectly.
	asia := wins[0]
	if asia.Kind != KindFirstN || HHMM(asia.StartMin) != "17:00" || HHMM(asia.EndMin) != "17:05" {
		t.Fatalf("ASIA first-N wrong: %s–%s", HHMM(asia.StartMin), HHMM(asia.EndMin))
	}
	// And the first-N test must work across midnight for that session.
	if !InFirstNoTradeMinutes("17:00", ctAt(17, 2)) {
		t.Fatal("17:02 is inside ASIA's first-N")
	}
	if InFirstNoTradeMinutes("17:00", ctAt(16, 59)) {
		t.Fatal("16:59 is before the open")
	}
}

// F1 — THE PIN. The ASIA card at 23:00 CT showed three constraints when one
// applied: first-5m (elapsed six hours earlier), the NY lunch window (cannot
// apply to an ASIA session), and a 09:00 T1 blackout that had passed fourteen
// hours before. Session-scoped evaluation collapses all three correctly.
func TestNoTradeBandAsiaAt2300(t *testing.T) {
	asia := SessionDef{Name: SessionAsia, WindowStartCT: "17:00", WindowEndCT: "02:00"}
	wins := BuildMachineNoTradeWindows(asia)
	// A 09:00 CT T1 event, as the calendar would resolve it at read time.
	wins = append(wins, NoTradeWindow{StartMin: 8*60 + 45, EndMin: 9*60 + 15,
		Kind: KindT1, Source: SourceCalendar, Label: "🔴 09:00 ISM PMI — HARD no-trade (red news)"})

	now := 23 * 60   // 23:00 CT
	start := 17 * 60 // ASIA opens 17:00
	eod := 2 * 60    // ASIA flat 02:00 (wraps midnight)
	got := EvaluateNoTradeWindows(wins, now, start, eod)

	byKind := map[string]string{}
	for _, w := range got {
		byKind[w.Kind] = w.Status
	}
	if byKind[KindFirstN] != StatusElapsed {
		t.Errorf("first-5m opened at 17:00 and is six hours gone at 23:00 — want elapsed, got %q", byKind[KindFirstN])
	}
	if byKind[KindLunch] != StatusOtherSession {
		t.Errorf("lunch 12:00–13:30 is an NY window and cannot apply to ASIA — want other_session, got %q", byKind[KindLunch])
	}
	if byKind[KindT1] != StatusOtherSession {
		t.Errorf("a 09:00 T1 is not in ASIA's remaining window — want other_session, got %q", byKind[KindT1])
	}
	if n := len(LiveNoTradeWindows(got)); n != 0 {
		t.Fatalf("ASIA at 23:00 has NO live constraint, got %d: %+v", n, LiveNoTradeWindows(got))
	}
}

// F2 — NY at 08:00: first-5m and lunch are both ahead; an 07:30 T1 is gone.
func TestNoTradeBandNYAt0800(t *testing.T) {
	ny := SessionDef{Name: SessionNY, WindowStartCT: "08:30", WindowEndCT: "14:45"}
	wins := BuildMachineNoTradeWindows(ny)
	wins = append(wins,
		NoTradeWindow{StartMin: 7*60 + 15, EndMin: 7*60 + 45, Kind: KindT1, Source: SourceCalendar, Label: "🔴 07:30 event"},
		NoTradeWindow{StartMin: 9*60 + 45, EndMin: 10*60 + 15, Kind: KindT1, Source: SourceCalendar, Label: "🔴 10:00 event"})
	got := EvaluateNoTradeWindows(wins, 8*60, 8*60+30, 14*60+45)

	live := map[string]bool{}
	for _, w := range LiveNoTradeWindows(got) {
		live[w.Label] = true
	}
	if len(live) != 3 {
		t.Fatalf("want first-5m + lunch + the 10:00 T1 live, got %d: %+v", len(live), live)
	}
	// The 07:15–07:45 blackout never overlaps NY's 08:30–14:45 span, so it is
	// scoped out rather than merely spent. Either way it must not render live.
	for _, w := range got {
		if w.Label == "🔴 07:30 event" && w.Status == StatusLive {
			t.Errorf("an 07:30 T1 cannot constrain an 08:30 session, got %q", w.Status)
		}
	}
}

// F3 — a window straddling "now" is LIVE, not elapsed.
func TestNoTradeBandStraddlingNowIsLive(t *testing.T) {
	got := EvaluateNoTradeWindows(
		[]NoTradeWindow{{StartMin: 12 * 60, EndMin: 13*60 + 30, Kind: KindLunch}},
		12*60+45, 8*60+30, 14*60+45)
	if got[0].Status != StatusLive {
		t.Fatalf("12:45 is inside 12:00–13:30 — want live, got %q", got[0].Status)
	}
}

// F5 — a healthy clock produces NO "(clock drift)" claim, and the minute of
// boundary protection survives the change. The suffix used to be appended for
// any nonzero measurement, so this machine's 108 ms NTP offset put "the clock
// is drifting" on the card for the whole trading day.
func TestWidenCTWindowsDriftClaim(t *testing.T) {
	base := []CTWindow{{Start: 8*60 + 45, End: 9*60 + 15, Label: "ISM PMI 09:00 CT ±15m"}}

	healthy := WidenCTWindows(base, 108) // +108 ms, the audited NTP offset
	if contains(healthy[0].Label, "clock drift") {
		t.Fatalf("a healthy clock must not claim the machine is drifting: %q", healthy[0].Label)
	}
	if healthy[0].Start != 8*60+44 || healthy[0].End != 9*60+16 {
		t.Fatalf("boundary widening must survive the label change: %d–%d", healthy[0].Start, healthy[0].End)
	}

	skewed := WidenCTWindows(base, 90_000) // 90 s of real skew
	if !contains(skewed[0].Label, "clock drift") {
		t.Fatalf("a real skew must be stated on the card: %q", skewed[0].Label)
	}
	if skewed[0].Start != 8*60+43 || skewed[0].End != 9*60+17 {
		t.Fatalf("90s skew widens by 2m: %d–%d", skewed[0].Start, skewed[0].End)
	}
}

// F6 — T1 band entries are MAPPED from the enforcer's resolved windows, never
// recomputed, so the card cannot disagree with the gate about red news.
func TestT1NoTradeWindowsFromCT(t *testing.T) {
	got := T1NoTradeWindowsFromCT([]CTWindow{{Start: 525, End: 555, Label: "ISM PMI 09:00 CT ±15m"}})
	if len(got) != 1 {
		t.Fatalf("want one window, got %d", len(got))
	}
	if got[0].StartMin != 525 || got[0].EndMin != 555 {
		t.Errorf("bounds must pass through untouched: %d–%d", got[0].StartMin, got[0].EndMin)
	}
	if got[0].Kind != KindT1 || got[0].Source != SourceCalendar {
		t.Errorf("red news is calendar-sourced T1, got kind=%q source=%q", got[0].Kind, got[0].Source)
	}
}

// F1 (THE PIN) — the ASIA card at 23:00 CT. The doc the planner actually wrote
// carries three no_trade prose lines; all three were rendered as live rules
// while the session was mid-flight and none of them could constrain anything.
// The machine band, evaluated against the reader's clock, says so.
func TestRenderNoTradeBandAsiaCardAt2300(t *testing.T) {
	asia := &SessionDef{Name: SessionAsia, WindowStartCT: "17:00", WindowEndCT: "02:00"}
	doc := &PlanDoc{
		// verbatim shape of the prose the card used to render as three rules
		NoTrade: []string{
			"no entries in the first 5m after the open",
			"no entries 12:00–13:30 CT (lunch)",
			"🔴 ISM PMI 09:00 CT ±15m — HARD no-trade (red news)",
		},
		NoTradeWindows: append(BuildMachineNoTradeWindows(*asia),
			T1NoTradeWindowsFromCT([]CTWindow{{Start: 525, End: 555, Label: "ISM PMI 09:00 CT ±15m"}})...),
	}

	at2300 := time.Date(2026, 9, 2, 23, 0, 0, 0, CTLocation())
	rendered := RenderNoTradeBand(doc, asia, at2300)
	if len(rendered) != len(doc.NoTradeWindows) {
		t.Fatalf("every window must be rendered with a status, got %d of %d", len(rendered), len(doc.NoTradeWindows))
	}
	if n := len(LiveNoTradeWindows(rendered)); n != 0 {
		t.Fatalf("ASIA at 23:00 has NO live no-trade constraint; card would show %d: %+v", n, LiveNoTradeWindows(rendered))
	}
	if len(doc.NoTrade) != 3 {
		t.Fatalf("the model's prose is untouched by this wave, got %d lines", len(doc.NoTrade))
	}

	// Same doc, same session, read at 17:02 — the first-5m band IS live.
	at1702 := time.Date(2026, 9, 2, 17, 2, 0, 0, CTLocation())
	live := LiveNoTradeWindows(RenderNoTradeBand(doc, asia, at1702))
	if len(live) != 1 || live[0].Kind != KindFirstN {
		t.Fatalf("at 17:02 exactly the first-5m band is live, got %+v", live)
	}

	// A doc from before this wave has no machine windows and must not render an
	// empty band as "no constraints" — it returns nil so the card keeps prose.
	if RenderNoTradeBand(&PlanDoc{NoTrade: doc.NoTrade}, asia, at2300) != nil {
		t.Fatal("a doc with no machine windows must return nil, not an empty band")
	}
}
