package trader

import "testing"

// reconcileBeforeOpenNT flattens an NT8 orphan before opening, routing on
// `held == "long"`. NT8 GetPositions emits an UPPERCASE side, so before the fix a
// held LONG returned "LONG", missed the == "long" branch, and CloseShort was sent
// — the flatten never confirmed flat and every open was refused. ntHeldPosition
// must normalize to lowercase so the routing is correct.
func TestNtHeldPosition_NormalizesUppercaseNT8SideToLower(t *testing.T) {
	m := &MockTrader{positions: []map[string]interface{}{
		{"symbol": "MNQ", "side": "LONG", "positionAmt": 2.0},
	}}
	at := &AutoTrader{trader: m}

	if got := at.ntHeldPosition("MNQ"); got != "long" {
		t.Fatalf("ntHeldPosition(MNQ) = %q, want \"long\" (uppercase NT8 side must normalize so reconcile routes to CloseLong)", got)
	}
	// No matching symbol → flat.
	if got := at.ntHeldPosition("ES"); got != "" {
		t.Fatalf("ntHeldPosition(ES) = %q, want \"\" (flat)", got)
	}
}
