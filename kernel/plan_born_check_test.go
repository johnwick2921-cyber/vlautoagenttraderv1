package kernel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// W-EXEC-TRUTH W2 A1 + A2 (2026-09-23). Fixture: plans rowid 455
// (2026-09-23:LONDON v1) and rowid 452 (2026-09-22:ASIA v1), and the MNQ 1m
// bars open_time_ms 1790144400000..1790146500000 (01:20..01:55 CDT), exported
// read-only — provenance inside the JSON.

type w2Fixture struct {
	ReadClockMs    int64 `json:"read_clock_ms"`
	PublishClockMs int64 `json:"publish_clock_ms"`
	Rows           map[string]struct {
		PriceAtWrite float64       `json:"price_at_write"`
		Death        PlanCondition `json:"death"`
		Flip         PlanCondition `json:"flip"`
		Scenarios    []struct {
			ID      string `json:"id"`
			Invalid string `json:"invalid"`
		} `json:"scenarios"`
	} `json:"rows"`
	Bars []struct {
		OpenTimeMs int64   `json:"open_time_ms"`
		O          float64 `json:"o"`
		H          float64 `json:"h"`
		L          float64 `json:"l"`
		C          float64 `json:"c"`
	} `json:"bars_mnq_1m"`
}

func loadW2Fixture(t *testing.T) (w2Fixture, []market.Kline, time.Time, time.Time) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "w2_born_check", "row455_row452_tape_20260923.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f w2Fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	publish := time.UnixMilli(f.PublishClockMs)
	var bars []market.Kline
	for _, b := range f.Bars {
		// The provider at publish (01:51:47) holds nothing that opens later.
		if b.OpenTimeMs >= f.PublishClockMs {
			continue
		}
		bars = append(bars, market.Kline{OpenTime: b.OpenTimeMs, CloseTime: b.OpenTimeMs + 59_999, Open: b.O, High: b.H, Low: b.L, Close: b.C})
	}
	return f, bars, time.UnixMilli(f.ReadClockMs), publish
}

func w2Doc(f w2Fixture, row string) *PlanDoc {
	r := f.Rows[row]
	d := &PlanDoc{}
	for _, s := range r.Scenarios {
		d.Scenarios = append(d.Scenarios, PlanScenario{ID: s.ID, Invalid: s.Invalid})
	}
	death, flip := r.Death, r.Flip
	d.DeathStructured, d.FlipStructured = &death, &flip
	return d
}

// A2 group set: read 01:30:27 / publish 01:51:47 → the four groups closing at
// 01:35, 01:40, 01:45, 01:50 (open ids below).
func TestBornGroupsRow455ReadToPublish(t *testing.T) {
	_, _, read, publish := loadW2Fixture(t)
	var got []int64
	for _, g := range AuthoredBornGroups(read, publish) {
		got = append(got, g.OpenMs)
	}
	want := []int64{1790145000000, 1790145300000, 1790145600000, 1790145900000}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	// Legacy (zero read): the latest completed group only — today's window.
	if g := AuthoredBornGroups(time.Time{}, publish); len(g) != 1 || g[0].OpenMs != 1790145900000 {
		t.Fatalf("zero read must judge only the latest completed group, got %+v", g)
	}
	// Read and publish inside one bucket: still the latest completed group,
	// never nothing (the check never judges LESS than before).
	if g := AuthoredBornGroups(publish.Add(-30*time.Second), publish); len(g) != 1 || g[0].OpenMs != 1790145900000 {
		t.Fatalf("same-bucket read must judge the latest completed group, got %+v", g)
	}
}

// A1: row 455's four sentences → four grammar refusals in ONE error that
// quotes each sentence; row 452 → S2 refused, S1 (conformant) judged.
func TestBornCheckRow455FourGrammarRefusals(t *testing.T) {
	f, bars, read, publish := loadW2Fixture(t)
	r := EvaluateBornCheck(w2Doc(f, "455"), bars, read, publish)
	if len(r.Grammar) != 4 {
		t.Fatalf("row 455: want 4 grammar refusals, got %d (%+v)", len(r.Grammar), r.Check.Verdicts)
	}
	err := r.Err()
	if err == nil || !strings.HasPrefix(err.Error(), AuthoredGrammarRefusalMarker) {
		t.Fatalf("combined refusal must open with the marker: %v", err)
	}
	for _, s := range f.Rows["455"].Scenarios {
		if !strings.Contains(err.Error(), fmt.Sprintf("%s invalid %q", s.ID, s.Invalid)) {
			t.Errorf("refusal does not quote %s %q: %v", s.ID, s.Invalid, err)
		}
	}
	// D5 on the same tape: death{2x5m above 31075.75} met by 01:35 31081.00 +
	// 01:40 31090.75; flip{5m_close above 31066.32} met by 01:35 31081.00.
	if r.Death == nil || !strings.Contains(r.Death.Reason, "31081.00 (01:35 CT), 31090.75 (01:40 CT)") {
		t.Fatalf("row 455 death must be met by the 01:35+01:40 pair: %+v", r.Death)
	}
	if r.Flip == nil || !strings.Contains(r.Flip.Reason, "31081.00 (01:35 CT)") {
		t.Fatalf("row 455 flip must be met at 01:35: %+v", r.Flip)
	}
	if !strings.Contains(err.Error(), "born dead") || !strings.Contains(err.Error(), "re-author on the flipped side") {
		t.Fatalf("one error names every defect (grammar, death, flip): %v", err)
	}

	r452 := EvaluateBornCheck(w2Doc(f, "452"), bars, read, publish)
	if len(r452.Grammar) != 1 || r452.Grammar[0].ID != "S2" {
		t.Fatalf("row 452: only S2 (%q) is outside the grammar, got %+v", f.Rows["452"].Scenarios[1].Invalid, r452.Grammar)
	}
	if !strings.Contains(r452.Err().Error(), fmt.Sprintf("S2 invalid %q", f.Rows["452"].Scenarios[1].Invalid)) {
		t.Fatalf("row 452 refusal must quote S2: %v", r452.Err())
	}
}

// A2 discriminating cases (D2): a breach only BETWEEN read and publish refuses
// with the read clock and is accepted by the latest-window check (zero read),
// which proves A2 did the refusing. Row 455's own S1/S2 thresholds, written
// conformantly, are breached by both.
func TestBornCheckA2DiscriminatesReadToPublish(t *testing.T) {
	_, bars, read, publish := loadW2Fixture(t)
	cases := []struct {
		invalid          string
		withRead, legacy bool // invalidated?
	}{
		{"5m close above 31085.00", true, false},   // only the 01:40 group (31090.75)
		{"2x5m close above 31080.00", true, false}, // 01:35+01:40 (and 01:40+01:45); latest pair 31083.00/31079.75 is not
		{"5m close above 31075.75", true, true},    // row 455 S1 threshold, conformant
		{"5m close above 31066.32", true, true},    // row 455 S2 threshold, conformant
		{"5m close above 31095.00", false, false},  // above every close in the window
	}
	for _, tc := range cases {
		sc := PlanScenario{ID: "S1", Invalid: tc.invalid}
		v := EvaluateAuthoredInvalidationBetween(sc, bars, read, publish)
		if !v.Known || v.Invalidated != tc.withRead {
			t.Errorf("%q with read clock: %+v", tc.invalid, v)
		}
		if len(v.Groups) != 4 {
			t.Errorf("%q judged %d groups, want 4", tc.invalid, len(v.Groups))
		}
		l := EvaluateAuthoredInvalidationBetween(sc, bars, time.Time{}, publish)
		if !l.Known || l.Invalidated != tc.legacy {
			t.Errorf("%q latest-window (zero read): %+v", tc.invalid, l)
		}
	}
}

// Tape-UNKNOWN is never a refusal: a missing minute makes THAT group unknown;
// a breach on complete groups still refuses.
func TestBornCheckTapeUnknownIsPerGroup(t *testing.T) {
	_, bars, read, publish := loadW2Fixture(t)
	drop := func(openMs int64) []market.Kline {
		var out []market.Kline
		for _, b := range bars {
			if b.OpenTime != openMs {
				out = append(out, b)
			}
		}
		return out
	}
	// 01:39 minute removed → the 01:35–01:40 group (the only 31085 breach) is unknown.
	v := EvaluateAuthoredInvalidationBetween(PlanScenario{ID: "S1", Invalid: "5m close above 31085.00"}, drop(1790145540000), read, publish)
	if v.Known || v.Unknown != AuthoredUnknownTape {
		t.Fatalf("breach group without its minutes must be tape-UNKNOWN, got %+v", v)
	}
	r := EvaluateBornCheck(&PlanDoc{Scenarios: []PlanScenario{{ID: "S1", Invalid: "5m close above 31085.00"}}}, drop(1790145540000), read, publish)
	if r.Err() != nil || len(r.TapeUnknown) != 1 {
		t.Fatalf("tape-UNKNOWN must be accepted and counted, never refused: err=%v %+v", r.Err(), r.Check.Verdicts)
	}
	// Removing a minute of the 01:45–01:50 group leaves 01:35–01:40 intact.
	v = EvaluateAuthoredInvalidationBetween(PlanScenario{ID: "S1", Invalid: "5m close above 31085.00"}, drop(1790146140000), read, publish)
	if !v.Known || !v.Invalidated {
		t.Fatalf("a breach on complete groups refuses despite a gap elsewhere, got %+v", v)
	}
	// Duplicate minute in the breach group → that group unknown.
	dup := append([]market.Kline(nil), bars...)
	for _, b := range bars {
		if b.OpenTime == 1790145540000 {
			dup = append(dup, b)
		}
	}
	if len(dup) != len(bars)+1 {
		t.Fatal("fixture lacks the 01:39 minute")
	}
	if v = EvaluateAuthoredInvalidationBetween(PlanScenario{ID: "S1", Invalid: "5m close above 31085.00"}, dup, read, publish); v.Known {
		t.Fatalf("duplicate minute must make its group tape-UNKNOWN, got %+v", v)
	}
}

// A conformant candidate with nothing breached passes and records the clocks,
// the four groups and one verdict per subject.
func TestBornCheckConformantPassesAndRecords(t *testing.T) {
	_, bars, read, publish := loadW2Fixture(t)
	doc := &PlanDoc{Scenarios: []PlanScenario{{ID: "S1", Invalid: "2x5m close above 31100.00"}, {ID: "S2", Invalid: "5m close below 31000.00"}},
		DeathStructured: &PlanCondition{Price: 31110, Side: "above", Rule: "2x5m"},
		FlipStructured:  &PlanCondition{Price: 30990, Side: "below", Rule: "5m_close", FlipTo: "short"}}
	r := EvaluateBornCheck(doc, bars, read, publish)
	if err := r.Err(); err != nil {
		t.Fatalf("conformant, unbreached candidate refused: %v", err)
	}
	c := r.Check
	if c.Policy != AuthoredInvalidationPolicy() || c.ReadClockMs == nil || *c.ReadClockMs != 1790145027000 || c.PublishClockMs != 1790146307000 || len(c.Groups) != 4 || len(c.Verdicts) != 4 {
		t.Fatalf("record: %+v", c)
	}
	var back BornCheck
	if json.Unmarshal([]byte(c.JSON()), &back) != nil || back.Groups[0] != 1790145000000 {
		t.Fatalf("record JSON does not round-trip: %s", c.JSON())
	}
	// Zero read: read clock absent (n/a), death/flip not judged, never invented.
	z := EvaluateBornCheck(doc, bars, time.Time{}, publish).Check
	if z.ReadClockMs != nil || !strings.Contains(z.JSON(), `"read_clock_ms":null`) || z.Verdicts[2].Outcome != "not_judged" {
		t.Fatalf("legacy record must say read n/a and not judge the lines: %s", z.JSON())
	}
}

// Split pin: every UNKNOWN names its kind.
func TestAuthoredInvalidationUnknownSplitsGrammarFromTape(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-09-08T08:03:00-05:00")
	end := now.UnixMilli() / 300000 * 300000
	var bars []market.Kline
	for ms := end - 600000; ms < end; ms += 60000 {
		bars = append(bars, market.Kline{OpenTime: ms, CloseTime: ms + 60000, Close: 100})
	}
	for _, tc := range []struct {
		rule string
		bars []market.Kline
		kind string
	}{
		{"auction changes character", bars, AuthoredUnknownGrammar},
		{"5m close back below 101", bars, AuthoredUnknownGrammar},
		{"5m close below 0", bars, AuthoredUnknownGrammar},
		{"5m close below 101", bars[:len(bars)-1], AuthoredUnknownTape},
		{"5m close below 101", append(append([]market.Kline(nil), bars...), bars[9]), AuthoredUnknownTape},
		{"5m close below 101", bars, ""},
	} {
		if v := EvaluateAuthoredInvalidationAt(PlanScenario{ID: "S1", Invalid: tc.rule}, tc.bars, now); v.Unknown != tc.kind {
			t.Errorf("%q: unknown kind %q, want %q (%+v)", tc.rule, v.Unknown, tc.kind, v)
		}
	}
}

// Parity (A1): the ONE example the rendered prompt carries parses under the
// grammar once a number replaces the placeholder, and so does every form.
func TestAuthoredGrammarPromptExampleParses(t *testing.T) {
	prompt := plannerOutputContract(8, 3, true, true, true)
	m := regexp.MustCompile(`"invalid" GRAMMAR[^\n]*Example: "invalid": "([^"]+)"`).FindStringSubmatch(prompt)
	if m == nil {
		t.Fatal("the rendered output contract carries no invalid-grammar example")
	}
	if !strings.Contains(m[1], "<price>") {
		t.Fatalf("the example must use a placeholder price, got %q", m[1])
	}
	samples := append([]string{m[1]}, AuthoredInvalidationGrammarForms()...)
	for _, s := range samples {
		if !strings.Contains(prompt, s) {
			t.Errorf("form %q is not in the rendered prompt", s)
		}
		sc := PlanScenario{ID: "S1", Invalid: strings.ReplaceAll(s, "<price>", "31075.75")}
		if v := EvaluateAuthoredInvalidationAt(sc, nil, time.Now()); v.Unknown == AuthoredUnknownGrammar {
			t.Errorf("prompt form %q does not parse under the grammar: %+v", sc.Invalid, v)
		}
	}
}

// The combined refusal routes to the grammar law in the repair prompt.
func TestAuthoredGrammarRefusalRoutesRepairLaw(t *testing.T) {
	err := BornCheckResult{Grammar: []PlanScenario{{ID: "S4", Invalid: "A 5m close back above 31050.00 invalidates the hold; abandon the continuation."}}}.Err()
	if got := lawExcerptsFor(err.Error()); !strings.Contains(got, RepairInvalidationGrammarLaw) {
		t.Fatalf("grammar refusal must route to RepairInvalidationGrammarLaw, got %q", got)
	}
	if p := BuildPlannerRepairPrompt("{}", err.Error(), nil); !strings.Contains(p, RepairInvalidationGrammarLaw) || !strings.Contains(p, `"A 5m close back above 31050.00`) {
		t.Fatalf("repair prompt must carry the law and the quoted sentence: %s", p)
	}
	if got := lawExcerptsFor("born-dead authored scenario: S1: x"); strings.Contains(got, RepairInvalidationGrammarLaw) {
		t.Fatal("the grammar law must not ride a non-grammar refusal")
	}
}
