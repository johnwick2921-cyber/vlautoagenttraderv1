package api

import (
	"encoding/json"
	"testing"

	storepkg "nofx/store"
)

// Trading-P2 RED→GREEN (audit/0926-trading-pipeline): pnl_corrected NULL for
// the 618-class unresolved rows rendered without P&L. The API boundary must
// stamp every row with an explicit pnl_status ("unresolved" for NULL) and show
// the COUNT — a NULL must read "unresolved", never 0 or blank (canon class 40).
// The production call site is annotatePositionHistory, called by
// handlePositionHistory.
func TestAnnotatePositionHistoryMarksNullPnlUnresolved(t *testing.T) {
	resolvedVal := 12.5
	rows, unresolved := annotatePositionHistory([]*storepkg.TraderPosition{
		{ID: 618, PnlCorrected: nil, CloseReason: "unresolved"},
		{ID: 619, PnlCorrected: &resolvedVal, CloseReason: "sync"},
		{ID: 620, PnlCorrected: nil, CloseReason: ""}, // NULL corrected, no special reason — still unresolved
	})
	if unresolved != 2 {
		t.Fatalf("unresolved count = %d, want 2 (rows 618 and 620)", unresolved)
	}
	if rows[0].PnlStatus != "unresolved" || rows[1].PnlStatus != "resolved" || rows[2].PnlStatus != "unresolved" {
		t.Fatalf("pnl_status stamps wrong: %q %q %q", rows[0].PnlStatus, rows[1].PnlStatus, rows[2].PnlStatus)
	}

	// The wire shape: the unresolved row must carry pnl_status:"unresolved" and
	// NO numeric pnl_corrected — a NULL must never serialize as a number.
	blob, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["pnl_status"] != "unresolved" {
		t.Fatalf("row 618 wire: pnl_status=%v, want unresolved", decoded[0]["pnl_status"])
	}
	if v, has := decoded[0]["pnl_corrected"]; has {
		t.Fatalf("row 618 wire: pnl_corrected present as %v — a NULL must not read as a number", v)
	}
	if decoded[1]["pnl_status"] != "resolved" {
		t.Fatalf("row 619 wire: pnl_status=%v, want resolved", decoded[1]["pnl_status"])
	}
}

// Canon 49/53 — an empty computed list serializes as [] (computed), and the
// count is 0, never a fabricated value.
func TestAnnotatePositionHistoryEmptyIsEmptyArray(t *testing.T) {
	rows, unresolved := annotatePositionHistory(nil)
	if len(rows) != 0 || unresolved != 0 {
		t.Fatalf("empty input: rows=%d unresolved=%d, want 0/0", len(rows), unresolved)
	}
	blob, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != "[]" {
		t.Fatalf("empty computed list must marshal as [], got %s", blob)
	}
}
