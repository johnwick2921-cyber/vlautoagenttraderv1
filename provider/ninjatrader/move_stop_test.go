package ninjatrader

import (
	"bytes"
	"encoding/json"
	"testing"
)

// The move_stop frame must round-trip byte-clean over WriteFrame/ReadFrame, and
// the type constant must be exactly "move_stop" (lockstep with the C# handler).
func TestMoveStopFraming_RoundTrip(t *testing.T) {
	if FrameMoveStop != "move_stop" {
		t.Fatalf("FrameMoveStop=%q, want move_stop (C# dispatches on this literal)", FrameMoveStop)
	}
	in := MoveStopPayload{
		Symbol:      "MNQ",
		SignalID:    "abc-123",
		NewStopLoss: 30352.25,
		Timestamp:   "2026-08-01T00:00:00Z",
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameMoveStop, in); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameMoveStop {
		t.Fatalf("frame type = %q, want %q", env.Type, FrameMoveStop)
	}
	var out MoveStopPayload
	if err := json.Unmarshal(env.Payload, &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}
