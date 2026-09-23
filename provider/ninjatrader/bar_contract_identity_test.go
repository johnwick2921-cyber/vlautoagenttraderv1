package ninjatrader

import (
	"encoding/json"
	"testing"
)

// W-EXEC-TRUTH W4 / D21 identity — the wire has named the contract on EVERY
// bar frame since 2026-09-11 (owner ruling; VLBarsSubscriptionManager.cs:488
// for bars_historical and :562 for bar_update, both `["contract"] =
// entry.Contract`). Go's payload structs had no field for it, so it was parsed
// nowhere and every consumer was blind to which instrument its bars belonged
// to. No AddOn change is needed to fix that — only a field.

func TestBarUpdatePayload_ParsesTheContract(t *testing.T) {
	const frame = `{"symbol":"MNQ","timeframe":"4h","contract":"MNQ 12-26",
	 "bars":[{"t":1789000000000,"o":1,"h":2,"l":0.5,"c":1.5,"v":10,"final":true,"emitted_at":1789000000123}]}`
	var p BarUpdatePayload
	if err := json.Unmarshal([]byte(frame), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Contract != "MNQ 12-26" {
		t.Fatalf("bar_update dropped the contract the AddOn named: got %q", p.Contract)
	}
	if len(p.Bars) != 1 || !p.Bars[0].Final || p.Bars[0].EmittedAt != 1789000000123 {
		t.Fatalf("the frame's own evidence must survive the parse: %+v", p.Bars)
	}
}

func TestBarsHistoricalPayload_ParsesTheContract(t *testing.T) {
	const frame = `{"symbol":"MNQ","timeframe":"4h","contract":"MNQ 12-26","bars":[]}`
	var p BarsHistoricalPayload
	if err := json.Unmarshal([]byte(frame), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Contract != "MNQ 12-26" {
		t.Fatalf("bars_historical dropped the contract the AddOn named: got %q", p.Contract)
	}
}
