package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W5 foundation — the evidence adapter reads the evaluator's hand-off, and
// fails closed on anything it cannot vouch for.

func evidenceFixture() (*PictureHtfEvaluator, *store.PictureHtfOpportunityDB) {
	at := &AutoTrader{id: "t1", config: AutoTraderConfig{StrategyConfig: &store.StrategyConfig{}}}
	e := &PictureHtfEvaluator{at: at, pendingAdmission: &pictureAdmission{
		EntryRef: 21530, LatestClose: 21528.5, Stop: 21510, Target: 21590, ATR5m: 6.5, KnobMinRR: 2}}
	row := &store.PictureHtfOpportunityDB{
		OppKey: "t1|acct|MNQ 12-26|long|resistance|1790150000000|1790190000000", SignalID: "picture-htf-1790190001000",
		TraderID: "t1", StrategyID: "s1", Contract: "MNQ 12-26", Symbol: "MNQ", Direction: "LONG", RuleVer: 1,
		LevelRole: "resistance", LevelBodyTop: 21520, LevelBodyBot: 21500, LevelWickHi: 21526, LevelWickLo: 21494,
		LevelBarOpen: 1790150000000, LevelKnowable: 1790164400000,
		H1PrevClose: 21515, H1NewClose: 21531, H1Boundary: 21520, H1OpenTime: 1790186400000, H1CloseTime: 1790189999999,
		WindowOpen: 1790190000000, WindowClose: 1790190010000,
		StopSource: "5m swing low", TargetZone: "4H body 21585-21600", RREstimate: 3.0,
	}
	return e, row
}

func TestPictureEvidenceFromMapsTheHandOff(t *testing.T) {
	e, row := evidenceFixture()
	now := time.UnixMilli(1790190001234)
	ev, err := pictureEvidenceFrom(e, row, 21510, 21590, now)
	if err != nil {
		t.Fatal(err)
	}
	if ev.OppKey != row.OppKey || ev.ClaimID != row.SignalID || ev.Direction != "long" || ev.Rule != kernel.MachineRulePictureH1CloseBreak ||
		ev.BodyTop != 21520 || ev.H1NewClose != 21531 || ev.EntryRef != 21530 || ev.LatestClose != 21528.5 ||
		ev.Stop != 21510 || ev.Target != 21590 || ev.WindowCloseMs != 1790190010000 || ev.EvalAtMs != now.UnixMilli() {
		t.Fatalf("evidence = %+v", ev)
	}
	if ev.RRFloor < 2 {
		t.Fatalf("the R:R floor is max(knob, strategy floor), got %v", ev.RRFloor)
	}
	if ev.SourceEmittedAtMs != nil || ev.ReceivedAtMs != nil || ev.DepthCompleted != nil || ev.DepthNeeded != nil || ev.Generation != nil {
		t.Fatalf("W4-owned facts are absent until W4 lands them, never 0: %+v", ev)
	}
	b, _ := json.Marshal(ev)
	for _, k := range []string{"source_emitted_at_ms", "received_at_ms", "depth_completed", "generation"} {
		if strings.Contains(string(b), k) {
			t.Fatalf("an absent W4 fact must not serialize (%s): %s", k, b)
		}
	}
}

func TestPictureEvidenceFromFailsClosed(t *testing.T) {
	now := time.UnixMilli(1790190001234)
	for name, f := range map[string]func(*PictureHtfEvaluator, *store.PictureHtfOpportunityDB) (float64, float64){
		"no admission record": func(e *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) {
			e.pendingAdmission = nil
			return 21510, 21590
		},
		"no opportunity key": func(_ *PictureHtfEvaluator, r *store.PictureHtfOpportunityDB) (float64, float64) {
			r.OppKey = ""
			return 21510, 21590
		},
		"sideways": func(_ *PictureHtfEvaluator, r *store.PictureHtfOpportunityDB) (float64, float64) {
			r.Direction = "flat"
			return 21510, 21590
		},
		"no stop":   func(_ *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) { return 0, 21590 },
		"no target": func(_ *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) { return 21510, 0 },
		"no ATR": func(e *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) {
			e.pendingAdmission.ATR5m = 0
			return 21510, 21590
		},
		"no window": func(_ *PictureHtfEvaluator, r *store.PictureHtfOpportunityDB) (float64, float64) {
			r.WindowClose = 0
			return 21510, 21590
		},
		"no R:R floor": func(e *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) {
			e.at.config.StrategyConfig = nil
			return 21510, 21590
		},
	} {
		e, row := evidenceFixture()
		stop, target := f(e, row)
		if _, err := pictureEvidenceFrom(e, row, stop, target, now); err == nil {
			t.Errorf("%s: must fail closed", name)
		}
	}
}
