package trader

import (
	"strings"
	"testing"
)

// E1 — POSITION 589, the chase this leg exists for. Entry 29192.50 after the
// 140-pt rally from the 29048-29050 dip; MFE was 10.25 and it was stopped for
// −155.00. ATR5m ~26 and the stop floor is 1.5×ATR5m, so the run ceiling is
// ~39 pts and the actual run was ~142.
func TestNoChasePin589(t *testing.T) {
	v := EvaluateNoChase(NoChaseInputs{
		Entry: 29192.50, CitedLevel: 29050.00, LevelKind: "RTH-L",
		LastTouchPx: 29050.00, HasTouch: true, ATR5m: 26.0, MinSLMult: 1.5,
	})
	if !v.Applicable {
		t.Fatal("a cited level was supplied — the leg must apply")
	}
	if !v.WouldRefuse {
		t.Fatalf("589 must be WOULD_REFUSE: dist=%.1f/%.2f×ATR run=%.1f max_run=%.1f", v.DistPts, v.DistATR, v.RunPts, v.MaxRunPts)
	}
	if !strings.Contains(v.Why, "run") {
		t.Fatalf("589's chase is the RUN, not just the distance: %q", v.Why)
	}
	line := NoChaseLine("decision", "S3", v)
	for _, want := range []string{"🚫 no-chase WOULD_REFUSE decision S3", "dist=142.5pts", "run=142.5pts", "WARN-first, entry PROCEEDING"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line missing %q:\n%s", want, line)
		}
	}
}

// E2 — 587 entered near its level on a small run: ok. And an ARM entering AT
// the level is dist 0 / run 0, which must always be ok.
func TestNoChase587AndArmAtLevelAreOK(t *testing.T) {
	// 587: entry 29079.25, level ~29068 (VWAP−1σ per the plan), touched there.
	v := EvaluateNoChase(NoChaseInputs{
		Entry: 29079.25, CitedLevel: 29068.05, LastTouchPx: 29068.05, HasTouch: true,
		ATR5m: 26.0, MinSLMult: 1.5,
	})
	if v.WouldRefuse {
		t.Fatalf("587 entered near its level and must be ok: dist=%.1f run=%.1f why=%s", v.DistPts, v.RunPts, v.Why)
	}
	// The arm path rests AT the level by construction.
	atLevel := EvaluateNoChase(NoChaseInputs{
		Entry: 29068.05, CitedLevel: 29068.05, LastTouchPx: 29068.05, HasTouch: true,
		ATR5m: 26.0, MinSLMult: 1.5,
	})
	if atLevel.WouldRefuse || atLevel.DistPts != 0 || atLevel.RunPts != 0 {
		t.Fatalf("an arm resting AT the level must be dist 0 / run 0 / ok: %+v", atLevel)
	}
}

// E3 — no cited level: the leg ABSTAINS. It never invents a distance, and the
// caller counts the case separately.
func TestNoChaseNoCitedLevelAbstains(t *testing.T) {
	v := EvaluateNoChase(NoChaseInputs{Entry: 29192.50, ATR5m: 26.0, MinSLMult: 1.5})
	if v.Applicable || v.WouldRefuse {
		t.Fatalf("no cited level → not applicable, never a refusal: %+v", v)
	}
	if v.DistPts != 0 || v.DistATR != 0 {
		t.Fatalf("no level → no fabricated distance: %+v", v)
	}
}

// A level cited but never touched: dist still measures, run abstains.
func TestNoChaseUntouchedLevelMeasuresDistOnly(t *testing.T) {
	v := EvaluateNoChase(NoChaseInputs{
		Entry: 29192.50, CitedLevel: 29050.00, HasTouch: false,
		ATR5m: 26.0, MinSLMult: 1.5,
	})
	if v.RunPts != 0 {
		t.Fatalf("an untouched level has no run: %.1f", v.RunPts)
	}
	if !v.WouldRefuse || !strings.Contains(v.Why, "dist") {
		t.Fatalf("142 pts is 5.5×ATR — the DIST test must fire on its own: %+v", v)
	}
}

// E4 — knobs resolve, clamp, and the boot line reads them.
func TestNoChaseKnobsAndBootLine(t *testing.T) {
	t.Setenv("NOCHASE_MAX_DIST_ATR", "")
	if NoChaseMaxDistATR() != 1.0 {
		t.Fatalf("default = %v want 1.0", NoChaseMaxDistATR())
	}
	t.Setenv("NOCHASE_MAX_DIST_ATR", "2.5")
	if NoChaseMaxDistATR() != 2.5 {
		t.Fatalf("env = %v", NoChaseMaxDistATR())
	}
	for _, bad := range []string{"0", "-1", "99", "junk"} {
		t.Setenv("NOCHASE_MAX_DIST_ATR", bad)
		if NoChaseMaxDistATR() != 1.0 {
			t.Fatalf("%q must clamp to the default, got %v", bad, NoChaseMaxDistATR())
		}
	}
	t.Setenv("NOCHASE_MAX_DIST_ATR", "1.75")
	line := NoChaseBootLine()
	if !strings.Contains(line, "max_dist=1.75×ATR") {
		t.Fatalf("boot line must READ the resolver, not a literal:\n%s", line)
	}
	for _, want := range []string{"mode=warn", "counters=on", "[I] PROVISIONAL"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line missing %q:\n%s", want, line)
		}
	}
	// The run ceiling tracks the stop floor, not a constant.
	a := noChaseMaxRunPts(NoChaseInputs{ATR5m: 20, MinSLMult: 1.5})
	b := noChaseMaxRunPts(NoChaseInputs{ATR5m: 40, MinSLMult: 1.5})
	if a != 30 || b != 60 {
		t.Fatalf("run ceiling must scale with ATR5m: %.1f / %.1f", a, b)
	}
	if noChaseMaxRunPts(NoChaseInputs{ATR5m: 0, MinSLMult: 1.5}) != 0 {
		t.Fatal("no ATR → the run test must abstain, never use a literal")
	}
}

// The leg is wired into the ONE gate and runs on BOTH paths, WARN-only: the
// gate's verdict must stay "allow" no matter what the leg measures.
func TestNoChaseIsWiredButNeverRefuses(t *testing.T) {
	var got NoChaseVerdict
	var called int
	// THE CASE THIS LEG EXISTS FOR (premise C2): a chase whose R:R at the fill
	// is still plausible, so every existing leg allows it. Entry 29192.50 sits
	// 142 pts above the level it cites, but with a 42.5-pt stop and a 97.5-pt
	// target its R:R is 2.29 — above the floor — and its stop clears
	// 1.5×ATR5m. Only the no-chase leg has anything to say about it.
	//
	// (589 AS ACTUALLY FILLED had R:R 1.61 and is refused by the R:R leg
	// before this one runs — verified while writing this test.)
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29192.50, Stop: 29150.00, Target: 29290.00,
		ATR5m: 26.0, MinRR: 2.0, MinSLMult: 1.5,
		CitedScenario: "S3", ScenarioDir: "long", PlanMode: "advisory",
		CitedLevelPx: 29050.00, CitedLevelKind: "RTH-L",
		LastTouchPx: 29050.00, HasTouch: true,
		OnNoChase: func(v NoChaseVerdict) { got = v; called++ },
	})
	if called != 1 {
		t.Fatalf("the leg must run exactly once per intent, ran %d", called)
	}
	if !got.WouldRefuse {
		t.Fatalf("589's shape must measure as a chase: %+v", got)
	}
	if refused {
		t.Fatalf("WARN-FIRST: this wave refuses NOTHING, got refusal %q", reason)
	}
	// And an intent with no callback wired must not panic (A10).
	// stop 29068.05-29020 = 48.05 > 1.5×26 = 39, R:R = (29200-29068.05)/48.05 = 2.75
	if reason2, r := EntryGate(EntryIntent{
		Path: "arm", Action: "open_long", Symbol: "MNQ",
		Entry: 29068.05, Stop: 29020, Target: 29200, ATR5m: 26, MinRR: 2, MinSLMult: 1.5,
		PlanMode: "advisory",
	}); r {
		t.Fatalf("a nil OnNoChase must be inert, never a refusal: %s", reason2)
	}
}

// The counter key is per path and per outcome, so a week of counts can be read
// without parsing logs.
func TestNoChaseCounterKeys(t *testing.T) {
	if k := NoChaseCounterKey("decision", "would_refuse"); k != "nochase_decision_would_refuse" {
		t.Fatalf("key = %q", k)
	}
	if NoChaseCounterKey("arm", "ok") == NoChaseCounterKey("decision", "ok") {
		t.Fatal("paths must count separately — the arm path rests AT the level and should never chase")
	}
	if NoChaseCounterKey("decision", "no_level") == NoChaseCounterKey("decision", "ok") {
		t.Fatal("an uncited level is NOT an ok — it must count separately (A24: no plausible-zero)")
	}
}

// OWNER RULING 2026-09-02 — BOTH measures, OR'd; an unmeasurable run is NULL,
// counted, and judged on dist alone. A zero run and an unknown run must never
// look the same.
func TestNoChaseOrAndNullRun(t *testing.T) {
	// dist fires alone, run unknown → would_refuse on dist, RunKnown false.
	a := EvaluateNoChase(NoChaseInputs{Entry: 29192.50, CitedLevel: 29050, HasTouch: false, ATR5m: 26, MinSLMult: 1.5})
	if !a.WouldRefuse || !a.DistFired || a.RunFired || a.RunKnown {
		t.Fatalf("dist alone must carry it with run NULL: %+v", a)
	}
	if !strings.Contains(NoChaseLine("decision", "S3", a), "run=NULL(no recorded touch)") {
		t.Fatalf("an unknown run must print NULL, never 0.0pts:\n%s", NoChaseLine("decision", "S3", a))
	}
	// run fires alone: near the level by distance, but far from its touch.
	b := EvaluateNoChase(NoChaseInputs{Entry: 29100, CitedLevel: 29095, LastTouchPx: 29020, HasTouch: true, ATR5m: 26, MinSLMult: 1.5})
	if !b.WouldRefuse || b.DistFired || !b.RunFired {
		t.Fatalf("run alone must be able to carry it: %+v", b)
	}
	if !strings.Contains(b.Why, " OR ") == false && strings.Count(b.Why, "OR") > 0 {
		t.Fatalf("a single-half verdict must not read as an OR of two: %q", b.Why)
	}
	// both fire → the reason names both, joined by OR (not AND).
	c := EvaluateNoChase(NoChaseInputs{Entry: 29192.50, CitedLevel: 29050, LastTouchPx: 29050, HasTouch: true, ATR5m: 26, MinSLMult: 1.5})
	if !c.DistFired || !c.RunFired || !strings.Contains(c.Why, " OR ") {
		t.Fatalf("both halves fired — the reason must join them with OR: %q", c.Why)
	}
	// entering AT the touch is a KNOWN zero run, not an unknown one.
	d := EvaluateNoChase(NoChaseInputs{Entry: 29068.05, CitedLevel: 29068.05, LastTouchPx: 29068.05, HasTouch: true, ATR5m: 26, MinSLMult: 1.5})
	if !d.RunKnown || d.RunPts != 0 || d.WouldRefuse {
		t.Fatalf("entering at the touch is a known zero run: %+v", d)
	}
}
