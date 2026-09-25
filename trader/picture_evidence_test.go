package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W5 foundation — the evidence adapter reads the evaluator's hand-off, and
// fails closed on anything it cannot vouch for.

// withFourHourDepth installs a bar provider holding n completed 4H candles
// that closed before the fixture's evaluation clock (W4's depth4H reads it).
func withFourHourDepth(t *testing.T, n int) {
	t.Helper()
	withFourHourDepthEnding(t, n, 1790190000000)
}

// withFourHourDepthEnding is withFourHourDepth with the last candle closing
// just before endMs (the caller's evaluation clock).
func withFourHourDepthEnding(t *testing.T, n int, endMs int64) {
	t.Helper()
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(_ string, tf string, limit int) []market.Kline {
		if tf != "4h" {
			return nil
		}
		var out []market.Kline
		end := endMs
		for i := n; i >= 1; i-- {
			open := end - int64(i)*4*3600*1000
			out = append(out, market.Kline{OpenTime: open, CloseTime: open + 4*3600*1000 - 1, Final: true, Close: 21500})
		}
		if len(out) > limit {
			out = out[len(out)-limit:]
		}
		return out
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
}

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
	withFourHourDepth(t, 12)
	e, row := evidenceFixture()
	e.freshest5mAt = time.UnixMilli(1790190000900)
	e.freshest5mEmitted = 1790190000400
	e.freshest5mClose = 1790189999999
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
	// W4's evidence, READ at the seam: the three clocks of the freshest
	// completed 5m frame and the 4H depth (fetch = required + margin).
	if ev.SourceEmittedAtMs == nil || *ev.SourceEmittedAtMs != 1790190000400 || ev.ReceivedAtMs == nil || *ev.ReceivedAtMs != 1790190000900 ||
		ev.FrameCloseMs == nil || *ev.FrameCloseMs != 1790189999999 {
		t.Fatalf("the frame clocks: source=%v received=%v close=%v", ev.SourceEmittedAtMs, ev.ReceivedAtMs, ev.FrameCloseMs)
	}
	if ev.DepthNeeded == nil || *ev.DepthNeeded != 4 || ev.DepthFetched == nil || *ev.DepthFetched != 4+pictureHtfDepthMargin ||
		ev.DepthCompleted == nil || *ev.DepthCompleted < *ev.DepthNeeded {
		t.Fatalf("the depth evidence: fetched=%v completed=%v needed=%v", ev.DepthFetched, ev.DepthCompleted, ev.DepthNeeded)
	}
	if ev.Generation != nil {
		t.Fatalf("the generation is not exposed at the seam — absent, never 0: %v", *ev.Generation)
	}
	b, _ := json.Marshal(ev)
	if strings.Contains(string(b), "generation") {
		t.Fatalf("an absent fact must not serialize: %s", b)
	}
}

// A clock the evaluator never saw stays absent — never a fabricated 0.
func TestPictureEvidenceClocksAbsentWhenNeverSeen(t *testing.T) {
	withFourHourDepth(t, 12)
	e, row := evidenceFixture()
	ev, err := pictureEvidenceFrom(e, row, 21510, 21590, time.UnixMilli(1790190001234))
	if err != nil {
		t.Fatal(err)
	}
	if ev.SourceEmittedAtMs != nil || ev.ReceivedAtMs != nil || ev.FrameCloseMs != nil {
		t.Fatalf("unseen clocks must be nil: %+v", ev)
	}
	b, _ := json.Marshal(ev)
	for _, k := range []string{"source_emitted_at_ms", "received_at_ms", "frame_close_ms"} {
		if strings.Contains(string(b), k) {
			t.Fatalf("an absent clock must not serialize (%s): %s", k, b)
		}
	}
}

func TestPictureEvidenceFromFailsClosed(t *testing.T) {
	now := time.UnixMilli(1790190001234)
	withFourHourDepth(t, 12)
	prevProvider := market.FuturesBarsProvider
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
		// W4 contract: KnowableAt is a completion instant; 0 means refuse.
		"unknown knowable instant": func(_ *PictureHtfEvaluator, r *store.PictureHtfOpportunityDB) (float64, float64) {
			r.LevelKnowable = 0
			return 21510, 21590
		},
		// W4 contract: below PivotWindow+4 completed 4H candles, no scenario.
		"insufficient 4H depth": func(e *PictureHtfEvaluator, _ *store.PictureHtfOpportunityDB) (float64, float64) {
			market.FuturesBarsProvider = func(string, string, int) []market.Kline { return nil }
			return 21510, 21590
		},
	} {
		market.FuturesBarsProvider = prevProvider
		e, row := evidenceFixture()
		stop, target := f(e, row)
		if _, err := pictureEvidenceFrom(e, row, stop, target, now); err == nil {
			t.Errorf("%s: must fail closed", name)
		}
	}
}
