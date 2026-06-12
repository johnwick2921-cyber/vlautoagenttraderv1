package databento

// CRITICAL: Fixtures in this file MUST match the actual Databento
// API response shape, NOT a fabricated approximation. A prior
// fabricated fixture allowed a struct-shape bug to ship to Plan 1
// SHIPPED status (cf. smoke-test 2026-05-25). Verify any fixture
// updates against a live curl response before committing.
//
// Reference shape (ohlcv-1m, GLBX.MDP3, NQ.c.0):
//   - hd.ts_event: nanoseconds since epoch as string
//   - hd.rtype: 33 for ohlcv-1m
//   - open/high/low/close: price × 1e9 as string
//   - volume: integer as string

import (
	"testing"
	"time"
)

// TestGetOHLCV_MockEndToEnd exercises the full client → HTTP → parser path
// against a captured-shape NDJSON fixture served by NewMockServer. This is
// the regression guard for the 2026-05-25 hd.ts_event nested-struct bug:
// the fixture mirrors the real Databento wire format, so any drift in the
// rawBar shape will fail here instead of only in live smoke.
func TestGetOHLCV_MockEndToEnd(t *testing.T) {
	srv := NewMockServer(t, "fixtures/nq-ohlcv-1m-real.json", "fixtures/resolve-nqm6.json")
	c := NewClient(srv.URL+"/v0", "test-key")

	start := time.Unix(0, 1779456600000000000).UTC()
	end := start.Add(100 * time.Minute)
	bars, err := c.GetOHLCV("NQ.c.0", "1m", start, end)
	if err != nil {
		t.Fatalf("GetOHLCV: %v", err)
	}
	if len(bars) != 100 {
		t.Fatalf("want 100 bars, got %d", len(bars))
	}

	// First bar shape sanity: real NQ levels around 21500, non-zero OHLCV.
	b0 := bars[0]
	if b0.Timestamp.IsZero() {
		t.Error("bar[0] has zero timestamp — hd.ts_event likely not parsed")
	}
	if b0.Open < 20000 || b0.Open > 25000 {
		t.Errorf("bar[0].Open = %v, out of NQ realistic band", b0.Open)
	}
	if b0.High < b0.Low || b0.Volume <= 0 {
		t.Errorf("bar[0] OHLCV invariants violated: %+v", b0)
	}

	// Monotonic, strictly increasing 60s spacing across all 100 bars.
	for i := 1; i < len(bars); i++ {
		dt := bars[i].Timestamp.Sub(bars[i-1].Timestamp)
		if dt != 60*time.Second {
			t.Fatalf("bars[%d].Timestamp - bars[%d].Timestamp = %v, want 60s", i, i-1, dt)
		}
	}
}

// realDatabentoResponseLine is a real OHLCV-1m record captured from a live
// GLBX.MDP3 NQ.c.0 curl on 2026-05-25 (smoke-test diagnosis window).
const realDatabentoResponseLine = `{"hd":{"ts_event":"1779387780000000000","rtype":33,"publisher_id":1,"instrument_id":42004058},"open":"29456250000000","high":"29458750000000","low":"29445500000000","close":"29454750000000","volume":"581"}`

func TestParseOHLCVResponse_TwoBars(t *testing.T) {
	// Two real-shape OHLCV-1m records (second is the first with a
	// 60-second-later ts_event + slightly different prices).
	body := []byte(realDatabentoResponseLine + "\n" +
		`{"hd":{"ts_event":"1779387840000000000","rtype":33,"publisher_id":1,"instrument_id":42004058},"open":"29454250000000","high":"29473250000000","low":"29453750000000","close":"29473000000000","volume":"713"}` + "\n")

	bars, err := parseOHLCVResponse(body)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(bars) != 2 {
		t.Fatalf("want 2 bars, got %d", len(bars))
	}

	want0 := Bar{
		Timestamp: time.Unix(0, 1779387780000000000).UTC(),
		Open:      29456.25,
		High:      29458.75,
		Low:       29445.50,
		Close:     29454.75,
		Volume:    581,
	}
	if bars[0] != want0 {
		t.Errorf("bar[0] = %+v, want %+v", bars[0], want0)
	}

	if bars[1].Close != 29473.00 {
		t.Errorf("bar[1].Close = %v, want 29473.00", bars[1].Close)
	}
}

func TestParseOHLCVResponse_Empty(t *testing.T) {
	bars, err := parseOHLCVResponse([]byte(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(bars) != 0 {
		t.Errorf("want 0 bars, got %d", len(bars))
	}
}

func TestParseOHLCVResponse_MalformedLine(t *testing.T) {
	// Real-shape record but with a non-numeric ts_event — should fail parsing.
	body := []byte(`{"hd":{"ts_event":"abc","rtype":33,"publisher_id":1,"instrument_id":42004058},"open":"1","high":"1","low":"1","close":"1","volume":"1"}` + "\n")
	_, err := parseOHLCVResponse(body)
	if err == nil {
		t.Fatal("want error on malformed ts_event, got nil")
	}
}

func TestParseOHLCVResponse_UnexpectedRtype(t *testing.T) {
	// Real-shape record but with the wrong rtype — the sanity check in
	// toBar() should reject it so we never silently parse a non-OHLCV-1m
	// record as a bar.
	body := []byte(`{"hd":{"ts_event":"1779387780000000000","rtype":1,"publisher_id":1,"instrument_id":42004058},"open":"29456250000000","high":"29458750000000","low":"29445500000000","close":"29454750000000","volume":"581"}` + "\n")
	_, err := parseOHLCVResponse(body)
	if err == nil {
		t.Fatal("want error on unexpected rtype, got nil")
	}
}
