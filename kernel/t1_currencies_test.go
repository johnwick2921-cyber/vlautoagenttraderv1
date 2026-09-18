package kernel

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"nofx/store"
)

// W-T1-CURRENCIES (2026-09-18) — a T1 blackout that never asked which currency
// the event was in. The 2026-09-17 ASIA slice (calendar_slices, source
// forexfactory) carried "BOJ Policy Rate" at 2026-09-18T02:54:00Z JPY T1 —
// 21:54 CT — and the MNQ bot sat in a HARD window for a Japanese rate
// decision. Under the shipped default only USD red events hard-block; every
// other T1 event is an ADVISORY line that renders and gates nothing.

// legacyT1BlackoutWindows is the pre-wave function body, copied VERBATIM from
// kernel/calendar_blackout.go @ origin/dev 0dd27940, so the ALL pin compares
// against what shipped rather than against the new code's own idea of itself.
func legacyT1BlackoutWindows(events []PlannerCalendarEvent) []CTWindow {
	var out []CTWindow
	for _, e := range events {
		if strings.ToUpper(strings.TrimSpace(e.Impact)) != "T1" {
			continue
		}
		m, ok := hhmmToMinK(e.TimeCT)
		if !ok {
			continue
		}
		out = append(out, CTWindow{
			Start: ((m-T1BlackoutMinutes)%1440 + 1440) % 1440,
			End:   ((m+T1BlackoutMinutes)%1440 + 1440) % 1440,
			Label: fmt.Sprintf("%s %s CT ±%dm", strings.TrimSpace(e.Title), e.TimeCT, T1BlackoutMinutes),
		})
	}
	return out
}

func legacyT1NoTradeLines(events []PlannerCalendarEvent) []string {
	var out []string
	for _, w := range legacyT1BlackoutWindows(events) {
		out = append(out, "🔴 "+w.Label+" — HARD no-trade (red news)")
	}
	return out
}

// threeCurrencyFixture: the 2026-09-17 evidence (JPY BOJ 21:54 CT, GBP BoE
// 06:00 CT) plus a USD FOMC and a T2 that must never gate.
func threeCurrencyFixture() []PlannerCalendarEvent {
	return []PlannerCalendarEvent{
		{TimeCT: "06:00", Currency: "GBP", Title: "Official Bank Rate", Impact: "T1"},
		{TimeCT: "13:00", Currency: "USD", Title: "FOMC Rate Decision", Impact: "T1"},
		{TimeCT: "21:54", Currency: "JPY", Title: "BOJ Policy Rate", Impact: "T1"},
		{TimeCT: "09:30", Currency: "USD", Title: "Fed Chair Speaks", Impact: "T2"},
	}
}

func TestT1DefaultUSDOnlyFOMCHardBOJAdvisory(t *testing.T) {
	sp := SplitT1(threeCurrencyFixture(), store.DefaultT1Currencies())
	if len(sp.Hard) != 1 || !strings.HasPrefix(sp.Hard[0].Label, "FOMC Rate Decision 13:00 CT") {
		t.Fatalf("default USD: only FOMC may hard-block, got %+v", sp.Hard)
	}
	if sp.Hard[0].Start != 765 || sp.Hard[0].End != 795 {
		t.Fatalf("FOMC window = [%d,%d] want [765,795]", sp.Hard[0].Start, sp.Hard[0].End)
	}
	if len(sp.Advisory) != 2 {
		t.Fatalf("GBP + JPY must be advisory, got %v", sp.Advisory)
	}
	wantBOJ := "🟠 BOJ Policy Rate 21:54 CT (JPY) — red news, advisory only (t1_currencies=USD)"
	if sp.Advisory[1] != wantBOJ {
		t.Fatalf("advisory line:\n got %q\nwant %q", sp.Advisory[1], wantBOJ)
	}
	if len(sp.Uncurrencied) != 0 {
		t.Fatalf("every event carried a currency: %v", sp.Uncurrencied)
	}
	// The gate never sees the BOJ event: 21:54 CT is tradeable under the default.
	if label, blocked := InT1Blackout(21*60+54, sp.Hard); blocked {
		t.Fatalf("BOJ (JPY) must not block MNQ under the USD default: %q", label)
	}
	// The plan lines carry the hard line first, then the two advisories.
	lines := T1NoTradeLines(threeCurrencyFixture(), store.DefaultT1Currencies())
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "🔴 FOMC") || !strings.HasPrefix(lines[1], "🟠 Official Bank Rate") || lines[2] != wantBOJ {
		t.Fatalf("no-trade lines wrong: %v", lines)
	}
	// Drift widening touches the hard window only; the advisory text is unchanged.
	wide := T1NoTradeLinesDrift(threeCurrencyFixture(), store.DefaultT1Currencies(), 90_000)
	if len(wide) != 3 || !strings.Contains(wide[0], "+2m (clock drift)") || wide[2] != wantBOJ {
		t.Fatalf("drift lines wrong: %v", wide)
	}
}

func TestT1AllIsByteIdenticalToTheShippedFunction(t *testing.T) {
	evs := threeCurrencyFixture()
	for _, set := range [][]string{{"ALL"}, {"all"}, {"*"}, {"USD", "ALL"}} {
		got := T1BlackoutWindows(evs, set)
		if !reflect.DeepEqual(got, legacyT1BlackoutWindows(evs)) {
			t.Fatalf("set %v: windows differ from the shipped function:\n got %+v\nwant %+v", set, got, legacyT1BlackoutWindows(evs))
		}
		if lines := T1NoTradeLines(evs, set); !reflect.DeepEqual(lines, legacyT1NoTradeLines(evs)) {
			t.Fatalf("set %v: lines differ from the shipped function:\n got %v\nwant %v", set, lines, legacyT1NoTradeLines(evs))
		}
		if adv := T1AdvisoryLines(evs, set); len(adv) != 0 {
			t.Fatalf("set %v: ALL has no advisories, got %v", set, adv)
		}
	}
}

func TestT1CurrencyMatchIsCaseInsensitiveAndMultiValued(t *testing.T) {
	evs := threeCurrencyFixture()
	got := T1BlackoutWindows(evs, []string{"usd", " Gbp "})
	if len(got) != 2 || !strings.HasPrefix(got[0].Label, "Official Bank Rate") || !strings.HasPrefix(got[1].Label, "FOMC") {
		t.Fatalf("usd+gbp (any case/space) must hard-block GBP and USD only: %+v", got)
	}
	evs[2].Currency = "jpy"
	if adv := T1AdvisoryLines(evs, []string{"USD"}); len(adv) != 2 || !strings.Contains(adv[1], "(JPY)") {
		t.Fatalf("lower-case event currency must still be placed and printed upper: %v", adv)
	}
	if T1CurrencySetLabel([]string{"USD", "EUR"}) != "USD,EUR" || T1CurrencySetLabel(nil) != "USD" || T1CurrencySetLabel([]string{"*"}) != "ALL" {
		t.Fatalf("set label: %q %q %q", T1CurrencySetLabel([]string{"USD", "EUR"}), T1CurrencySetLabel(nil), T1CurrencySetLabel([]string{"*"}))
	}
}

func TestT1EventWithoutCurrencyIsHardAndNamed(t *testing.T) {
	evs := []PlannerCalendarEvent{
		{TimeCT: "13:00", Currency: "", Title: "Unlabelled Red Event", Impact: "T1"},
		{TimeCT: "21:54", Currency: "JPY", Title: "BOJ Policy Rate", Impact: "T1"},
	}
	sp := SplitT1(evs, store.DefaultT1Currencies())
	if len(sp.Hard) != 1 || !strings.HasPrefix(sp.Hard[0].Label, "Unlabelled Red Event") {
		t.Fatalf("an event without a currency must FAIL CLOSED to hard: %+v", sp.Hard)
	}
	if !reflect.DeepEqual(sp.Uncurrencied, []string{"Unlabelled Red Event"}) {
		t.Fatalf("the caller must be told which event was uncurrencied: %v", sp.Uncurrencied)
	}
	if len(sp.Advisory) != 1 {
		t.Fatalf("the JPY event is still advisory: %v", sp.Advisory)
	}
}

// A nil/empty set can never gate nothing: it resolves to the shipped default.
func TestT1EmptySetResolvesToDefault(t *testing.T) {
	evs := threeCurrencyFixture()
	if !reflect.DeepEqual(SplitT1(evs, nil), SplitT1(evs, store.DefaultT1Currencies())) {
		t.Fatal("nil set must behave as the shipped default")
	}
	if !reflect.DeepEqual(SplitT1(evs, []string{}), SplitT1(evs, store.DefaultT1Currencies())) {
		t.Fatal("empty set must behave as the shipped default")
	}
}

// The prompt tags an out-of-set T1 as advisory, never as a machine blackout.
func TestPlannerPromptTagsOutOfSetT1AsAdvisory(t *testing.T) {
	in := samplePlannerInput()
	in.Calendar = threeCurrencyFixture()
	in.T1Currencies = store.DefaultT1Currencies()
	p := BuildPlannerPrompt(in)
	for _, want := range []string{
		"13:00 USD T1 — FOMC Rate Decision (HARD no-trade blackout — the machine writes and enforces it",
		"21:54 JPY T1 — BOJ Policy Rate (red news, ADVISORY only — NOT a machine blackout (t1_currencies=USD)",
		"06:00 GBP T1 — Official Bank Rate (red news, ADVISORY only",
		"09:30 USD T2 — Fed Chair Speaks (caution — NOT a no-trade blackout",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q\n---\n%s", want, p)
		}
	}
	in.T1Currencies = []string{store.T1CurrencyAll}
	if p := BuildPlannerPrompt(in); strings.Contains(p, "ADVISORY only") || !strings.Contains(p, "21:54 JPY T1 — BOJ Policy Rate (HARD no-trade blackout") {
		t.Fatalf("ALL must tag every T1 as HARD:\n%s", p)
	}
}
