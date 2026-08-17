package kernel

import (
	"strings"
	"testing"
)

func TestScoreLevels(t *testing.T) {
	levels := []DetectedLevel{
		lineLevel(KindPDH, 15600, "PDH", "d", true),
		lineLevel(KindPDL, 15450, "PDL", "d", true),
		lineLevel(KindPDC, 15555, "PDC", "d", true),
		lineLevel(KindRound, 15550, "RN 15550 (50)", "", false),
		lineLevel(KindPWH, 16000, "PWH", "w", true), // 470 pts away → proximity-filtered
		lineLevel(KindONH, 15500, "ONH", "d", false),
		{Kind: KindDemand, Price: 15400, Lo: 15390, Hi: 15410, Label: "Demand", OriginDate: "d"}, // standalone → excluded
		{Kind: KindSupply, Price: 15552, Lo: 15548, Hi: 15556, Label: "Supply", OriginDate: "d"}, // clustered → C
	}
	freshness := func(l DetectedLevel) string {
		if l.Kind == KindONH {
			return "done" // consumed → dropped
		}
		return "" // fresh
	}
	scored := ScoreLevels(levels, 15530, 200, freshness, 8, 1.5)

	grades := map[LevelKind]string{}
	present := map[LevelKind]bool{}
	for _, s := range scored {
		grades[s.Kind] = s.Grade
		present[s.Kind] = true
	}

	if present[KindPWH] {
		t.Fatalf("PWH is beyond ±1.5×dATR → must be proximity-filtered")
	}
	if present[KindONH] {
		t.Fatalf("consumed ONH must be dropped")
	}
	if present[KindDemand] {
		t.Fatalf("standalone demand zone must be excluded (confluence-only)")
	}
	if grades[KindPDH] != "A" || grades[KindPDL] != "A" || grades[KindPDC] != "A" {
		t.Fatalf("HTF structurals should grade A: %v", grades)
	}
	if grades[KindRound] != "B" {
		t.Fatalf("clustered round number should grade B, got %q", grades[KindRound])
	}
	if grades[KindSupply] != "C" {
		t.Fatalf("clustered supply zone must be capped at C, got %q", grades[KindSupply])
	}
	// Output is nearest-first: RN 15550 (dist 20) leads.
	if len(scored) == 0 || scored[0].Price != 15550 {
		t.Fatalf("nearest-first ordering broken: first = %+v", scored[0])
	}
}

func TestScoreLevelsCapAndPriority(t *testing.T) {
	// 4 today-priority + 4 round numbers, cap 3 → only priority survive.
	levels := []DetectedLevel{
		lineLevel(KindPDH, 15600, "PDH", "d", true),
		lineLevel(KindPDL, 15460, "PDL", "d", true),
		lineLevel(KindONH, 15590, "ONH", "d", false),
		lineLevel(KindORH, 15580, "OR-H", "d", false),
		lineLevel(KindRound, 15525, "RN 15525 (25)", "", false),
		lineLevel(KindRound, 15575, "RN 15575 (25)", "", false),
		lineLevel(KindRound, 15625, "RN 15625 (25)", "", false),
		lineLevel(KindRound, 15475, "RN 15475 (25)", "", false),
	}
	scored := ScoreLevels(levels, 15530, 200, nil, 3, 1.5)
	if len(scored) != 3 {
		t.Fatalf("cap 3 not honored: got %d", len(scored))
	}
	for _, s := range scored {
		if s.Kind == KindRound {
			t.Fatalf("round numbers must lose their seats to today-priority levels: %+v", s)
		}
	}
}

func TestRenderKeyLevelsBlock(t *testing.T) {
	scored := ScoreLevels([]DetectedLevel{
		lineLevel(KindPDH, 15600, "PDH", "d", true),
		lineLevel(KindPDC, 15555, "PDC", "d", true),
	}, 15530, 200, nil, 8, 1.5)
	block := RenderKeyLevelsBlock(scored, 15530)
	if !strings.HasPrefix(block, "KEY LEVELS (map, nearest-first; price 15530.00):") {
		t.Fatalf("block header wrong:\n%s", block)
	}
	if !strings.Contains(block, "PDH") || !strings.Contains(block, "PDC") {
		t.Fatalf("block missing a level:\n%s", block)
	}
	if !strings.Contains(block, "Anchor: react AT these levels") {
		t.Fatalf("block missing anchor line:\n%s", block)
	}
	// Empty input → empty block (dormant / B9-style).
	if RenderKeyLevelsBlock(nil, 15530) != "" {
		t.Fatalf("empty scored set must render an empty block")
	}
}
