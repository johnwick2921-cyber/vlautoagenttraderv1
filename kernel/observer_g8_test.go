package kernel

import (
	"strings"
	"testing"
)

func TestG8ObserverPromptGainsStructureQuestion(t *testing.T) {
	e := &StrategyEngine{}
	in := ObserverInput{
		Symbol: "MNQ", Side: "SHORT", EntryPrice: 29400, StopLoss: 29500, TakeProfit: 29200,
		Thesis:        "short on rejection at 29470.25",
		StructureLine: "STRUCTURE 15m: TRENDING_DOWN (LL 29220.25 @07:00) · 1h: RANGING",
	}
	p := e.BuildObserverSystemPrompt(in, 0)
	if !strings.Contains(p, "MACHINE STRUCTURE") || !strings.Contains(p, "TRENDING_DOWN") {
		t.Fatalf("structure line missing from observer prompt:\n%s", p)
	}
	if !strings.Contains(p, "structure_conflict") || !strings.Contains(p, "none | warning | confirmed") {
		t.Fatalf("conflict question missing:\n%s", p)
	}
}

func TestG8ParseObserverAssessmentStructureConflict(t *testing.T) {
	a, err := ParseObserverAssessment(`{"thesis_status":"intact","structure_conflict":"warning","note":"x","confidence":60}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if a.StructureConflict != "warning" {
		t.Fatalf("want warning, got %q", a.StructureConflict)
	}
	// absent field → none (additive schema, older models fine).
	a, err = ParseObserverAssessment(`{"thesis_status":"intact","note":"x","confidence":60}`)
	if err != nil || a.StructureConflict != "none" {
		t.Fatalf("absent field must default to none, got %q err=%v", a.StructureConflict, err)
	}
	// bad enum → unparseable.
	if _, err = ParseObserverAssessment(`{"thesis_status":"intact","structure_conflict":"banana","confidence":60}`); err == nil {
		t.Fatalf("bad enum must fail")
	}
	// action-like field still ignored (zero wire frames).
	a, err = ParseObserverAssessment(`{"thesis_status":"intact","structure_conflict":"none","action":"close_long","confidence":60}`)
	if err != nil || !a.ActionIgnored {
		t.Fatalf("action field must be flagged-ignored, got %+v err=%v", a, err)
	}
}

// TestG8Replay_StructureLineForHeldLosers proves the observer WOULD have seen
// the machine structure on the shift day: a held SHORT's structure line names
// the 10:45 flush swing (the event that would color the dot amber/red once a
// close-through ever happened). Honest: on 08-21 no 15m close-through
// occurred, so the line reads TRENDING_DOWN with the flush as the newest HH.
func TestG8Replay_StructureLineForHeldLosers(t *testing.T) {
	bars := real15m0821()
	st := ComputeStructureState(bars, 15, 0, hourMs(11, 30))
	line := StructurePromptLine(map[string]StructureState{"15m": st})
	if !strings.Contains(line, "TRENDING_DOWN") && !strings.Contains(line, "RANGING") {
		t.Fatalf("structure line missing trend: %q", line)
	}
	if st.Swing == nil || st.Swing.Price != 29488.50 {
		t.Fatalf("the 10:45 flush must be the newest swing, got %+v", st.Swing)
	}
}
