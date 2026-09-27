package kernel

import (
	"testing"

	"nofx/market"
)

// W-EXEC-TRUTH W4 / D22 — pivot evidence is fail-closed.
//
// A 4H body pivot is a claim about FIVE candles: the pivot and its four
// neighbours i-2, i-1, i+1, i+2. Before this wave the discovery loop ran
// i := 1 .. len-2 and the neighbour loop `continue`d past any neighbour that
// was out of range or absent, so a "pivot" could be declared on as few as two
// observed neighbours; and when i+2 did not exist, KnowableAt was left at 0,
// which every consumer read as ALREADY KNOWABLE — an absent timestamp doing
// the work of a real one (canon: no fabricated values; absent != 0).
//
// These tests are written at BodyPivots4H and ActiveLevels, the production
// functions the evaluator calls (trader/picture_htf_evaluator.go:137,231),
// with inputs shaped the way the production cache delivers them: CloseTime
// populated, because "completed" is CloseTime < now && Final there.

const fourHMs = int64(4) * 3600 * 1000

// D22.1 — all four neighbours must EXIST. A candidate at index 1 has only one
// candle below it; i-2 is out of range and is silently skipped today, so the
// level forms on three observed neighbours instead of four.
func TestBodyPivots4H_PivotNeedsAllFourCompletedNeighbours(t *testing.T) {
	bars := []market.Kline{
		bar4h(0, 90, 95, 85, 92),
		bar4h(1, 99, 105, 96, 101), // candidate: i-2 does not exist
		bar4h(2, 96, 100, 94, 97),
		bar4h(3, 95, 99, 93, 96),
		bar4h(4, 94, 98, 92, 95),
	}
	for _, l := range BodyPivots4H(bars, 120) {
		if l.SourceIdx == 1 {
			t.Fatalf("pivot formed at idx 1 with no i-2 neighbour: %+v", l)
		}
	}
}

// D22.2 — both TRAILING candles must have closed. A candidate at len-2 has no
// i+2 at all; today it forms and carries KnowableAt=0, which ActiveLevels
// reads as knowable, so the level is usable before the evidence for it exists.
func TestBodyPivots4H_NoPivotWithoutBothTrailingCandles(t *testing.T) {
	bars := []market.Kline{
		bar4h(0, 90, 95, 85, 91),
		bar4h(1, 91, 96, 86, 92),
		bar4h(2, 92, 97, 87, 93),
		bar4h(3, 99, 105, 96, 101), // candidate at len-2: i+2 missing
		bar4h(4, 94, 98, 92, 95),
	}
	for _, l := range BodyPivots4H(bars, 120) {
		if l.SourceIdx == 3 {
			t.Fatalf("pivot formed at len-2 without its second trailing candle: %+v", l)
		}
		if l.KnowableAt == 0 {
			t.Fatalf("level carries KnowableAt=0 (absent standing in for a real time): %+v", l)
		}
	}
}

// D22.3 — KnowableAt is the COMPLETION of i+2, not its open. A level built
// from i+2 is not knowable while i+2 is still forming; anchoring to its OPEN
// makes it knowable a whole 4H period early.
func TestBodyPivots4H_KnowableAtIsCompletionOfSecondTrailingCandle(t *testing.T) {
	bars := pivot4HFixture()
	levels := BodyPivots4H(bars, 120)
	if len(levels) == 0 {
		t.Fatalf("fixture must produce pivots")
	}
	for _, l := range levels {
		nxt := bars[l.SourceIdx+2]
		if l.KnowableAt != nxt.CloseTime {
			t.Fatalf("KnowableAt = %d (i+2 open = %d), want i+2 completion = %d",
				l.KnowableAt, nxt.OpenTime, nxt.CloseTime)
		}
	}
}

// D22.4 — the consumers must not read an ABSENT KnowableAt as knowable.
// A level whose knowable-time was never established is missing evidence, and
// missing evidence refuses (R1/R2).
func TestActiveLevels_AbsentKnowableAtIsNotKnowable(t *testing.T) {
	lv := []PictureHtfLevel{{Role: "resistance", Boundary: 100, KnowableAt: 0}}
	if got := ActiveLevels(lv, 1789999999999); len(got) != 0 {
		t.Fatalf("a level with no KnowableAt must never be active, got %+v", got)
	}
}
