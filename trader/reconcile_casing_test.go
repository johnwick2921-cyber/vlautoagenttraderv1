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

	if got, err := at.ntHeldPosition("MNQ"); err != nil || got != "long" {
		t.Fatalf("ntHeldPosition(MNQ) = %q err=%v, want \"long\" (uppercase NT8 side must normalize so reconcile routes to CloseLong)", got, err)
	}
	// No matching symbol → flat.
	if got, err := at.ntHeldPosition("ES"); err != nil || got != "" {
		t.Fatalf("ntHeldPosition(ES) = %q err=%v, want \"\" (flat)", got, err)
	}
}
