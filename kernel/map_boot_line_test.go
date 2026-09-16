package kernel

import (
	"strings"
	"testing"
)

// D7 — THE MAP BOOT LINE.
//
// Canon: "Boot lines are READ, never literal, and a field the process cannot
// know yet prints n/a" (CLAUDE.md; checklist 45, 49).
//
// detected / merged / entry-candidates / no-target refused / projections are
// all PER-READ. At boot no planner read has happened, so printing 0 would
// assert a measurement that was never taken — the same fault
// kernel/detector_d1prime.go:270-271 exists to prevent. They print n/a at boot
// and real counts on the per-read line.

func TestD7_BootLinePrintsNAForPerReadFieldsNeverZero(t *testing.T) {
	got := MapBootLine(DefaultMaxLevels, PlanHardMaxLevels, true)

	for _, field := range []string{"detected=n/a", "merged=n/a", "entry-candidates=n/a", "refused=n/a", "projections=n/a"} {
		if !strings.Contains(got, field) {
			t.Errorf("boot line must print %q — a field the process cannot know yet is n/a, never 0.\n got: %s", field, got)
		}
	}
	if strings.Contains(got, "=0") {
		t.Errorf("boot line contains a plausible zero (A24):\n %s", got)
	}
	// The RESOLVED fields are real values, read from the caller's resolver.
	// `cap` is PER-TRADER. The boot process serves several traders and must not
	// print one trader's number as if it were global — it labels it, the same
	// shape VolumeWaveBootLine adopted when 0e016635 fixed that exact class.
	if !strings.Contains(got, "cap=per-trader (default 8, hard cap 12)") {
		t.Errorf("cap must be LABELLED per-trader with its default and hard cap, got: %s", got)
	}
	if !strings.Contains(got, "pwh/pwl seatable=yes") {
		t.Errorf("pwh/pwl seatable must reflect whether a daily source is installed, got: %s", got)
	}
	// Every rule this wave adds is [I] until E4 measures it.
	if !strings.Contains(got, "order=reachability[I]") {
		t.Errorf("the ordering rule must be labelled [I] on the boot line, got: %s", got)
	}
}

func TestD7_BootLineSaysSeatableNoWithoutADailySource(t *testing.T) {
	got := MapBootLine(DefaultMaxLevels, PlanHardMaxLevels, false)
	if !strings.Contains(got, "pwh/pwl seatable=no") {
		t.Errorf("with no daily source the boot line must say seatable=no, got: %s", got)
	}
}

// The PER-READ line carries the real counts, and they are COUNTED from the map
// that was actually built — never inferred (canon 35: counters record).
func TestD7_PerReadLineCarriesCountedValues(t *testing.T) {
	price := 29650.00
	in := []ScoredLevel{
		mapFixtureLevel(KindPDC, 29655.00, "PDC", "A"),
		mapFixtureLevel(KindVWAP, 29656.00, "VWAP+1σ", "A"), // merges with PDC
		mapFixtureLevel(KindONH, 29760.00, "ONH", "A"),
	}
	projections := []MapCandidate{{
		Price: 29900.00, Names: []string{"PWH"}, Kinds: []LevelKind{KindPWH},
		MergedCount: 1, Projection: true, ProjectionMethod: "prior-week high (daily bars)",
	}}
	cs := BuildMapWithProjections(in, projections, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})

	counts := CountMap(len(in), cs)
	if counts.Detected != 3 {
		t.Errorf("Detected = %d, want 3 (the references handed in)", counts.Detected)
	}
	if counts.Merged != 3 {
		t.Errorf("Merged = %d, want 3 (PDC+VWAP merged, ONH, PWH projection)", counts.Merged)
	}
	if counts.Projections != 1 {
		t.Errorf("Projections = %d, want 1", counts.Projections)
	}
	if counts.EntryCandidates+counts.NoTargetRefused+counts.Projections != counts.Merged {
		t.Errorf("every merged candidate must be exactly one of entry/refused/projection: %+v", counts)
	}

	line := MapReadLine(counts)
	if !strings.Contains(line, "detected=3") || !strings.Contains(line, "merged=3") || !strings.Contains(line, "projections=1") {
		t.Errorf("per-read line lost a counted value: %s", line)
	}
	if strings.Contains(line, "n/a") {
		t.Errorf("the per-read line has real counts and must not print n/a: %s", line)
	}
}
