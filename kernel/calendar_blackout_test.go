package kernel

import "testing"

// W3 — T1 red-news HARD blackout windows.
func TestT1BlackoutWindows(t *testing.T) {
	evs := []PlannerCalendarEvent{
		{TimeCT: "13:00", Title: "FOMC Rate Decision", Impact: "T1"},
		{TimeCT: "09:30", Title: "Fed Chair Speaks", Impact: "T2"}, // T2 → not a hard block
		{TimeCT: "bad", Title: "malformed", Impact: "T1"},          // skipped
	}
	w := T1BlackoutWindows(evs)
	if len(w) != 1 {
		t.Fatalf("only the timed T1 event should make a window, got %d", len(w))
	}
	// 13:00 = 780; ±15 → [765,795]
	if w[0].Start != 765 || w[0].End != 795 {
		t.Fatalf("window = [%d,%d] want [765,795]", w[0].Start, w[0].End)
	}

	// inside → blocked; edges inclusive; outside → clear.
	if _, blocked := InT1Blackout(780, w); !blocked {
		t.Fatal("13:00 must be inside the blackout")
	}
	if _, blocked := InT1Blackout(765, w); !blocked {
		t.Fatal("12:45 (edge) must be inside")
	}
	if _, blocked := InT1Blackout(760, w); blocked {
		t.Fatal("12:40 must be OUTSIDE the blackout")
	}

	lines := T1NoTradeLines(evs)
	if len(lines) != 1 || !contains(lines[0], "FOMC") || !contains(lines[0], "HARD no-trade") {
		t.Fatalf("no-trade line wrong: %v", lines)
	}
}

func TestT1BlackoutEmpty(t *testing.T) {
	if w := T1BlackoutWindows(nil); len(w) != 0 {
		t.Fatal("no events → no windows")
	}
	if _, blocked := InT1Blackout(600, nil); blocked {
		t.Fatal("no windows → never blocked")
	}
}
