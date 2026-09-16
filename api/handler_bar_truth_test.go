package api

import (
	"os"
	"strings"
	"testing"
)

// CLASS 52 (owner ruling 2026-09-02) — the bar-arbiter backfill must MERGE,
// never wipe: a refresh that deletes before it knows what comes back is the
// bug (on 2026-09-02 the capped replay left a wiped 1m table at ~2,000 rows
// out of 14,508). Wiring-lint pin over the handler source, in the same spirit
// as TestDestructiveRoutesAreWiredSafely: the backfill branch must carry no
// ClearSince call and must state the merge contract in its response.
func TestBarTruthBackfillMergesNeverClears(t *testing.T) {
	src, err := os.ReadFile("handler_bar_truth.go")
	if err != nil {
		t.Fatalf("read handler source: %v", err)
	}
	body := string(src)
	backfillStart := strings.Index(body, "case \"backfill\":")
	if backfillStart < 0 {
		t.Fatal("backfill branch missing")
	}
	diffStart := strings.Index(body, "case \"diff\":")
	if diffStart < 0 || diffStart <= backfillStart {
		t.Fatal("diff branch bounds missing")
	}
	backfill := body[backfillStart:diffStart]

	if strings.Contains(backfill, "ClearSince") {
		t.Fatalf("the backfill branch still wipes the replay window (ClearSince) — the merge law forbids deleting before the replay's returned range is known:\n%s", backfill)
	}
	if !strings.Contains(backfill, "\"cleared_rows\": 0") {
		t.Fatalf("the backfill response must state cleared_rows=0 explicitly:\n%s", backfill)
	}
	if !strings.Contains(backfill, "rows the replay cannot replace are never deleted") {
		t.Fatalf("the backfill response must state the merge contract:\n%s", backfill)
	}
}
