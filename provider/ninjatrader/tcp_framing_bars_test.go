package ninjatrader

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// Plan 4.4 Stage 2 — round-trip tests for the 4 bar frame types added to
// tcp_framing.go. Patterns mirror tcp_framing_test.go (Plan 1.5).
//
// Each test:
//   1. Marshals the payload via WriteFrame into a buffer.
//   2. Reads the envelope back via ReadFrame.
//   3. Asserts env.Type matches the expected FrameType constant (also covers
//      the wire-string contract with the C# AddOn).
//   4. Unmarshals env.Payload into the same payload struct.
//   5. Compares with reflect.DeepEqual.

func TestBarsSubscribe_RoundTrip(t *testing.T) {
	want := BarsSubscribePayload{
		Symbol:     "MNQ",
		Timeframes: []string{"1m", "5m", "15m", "1h"},
		BarsBack:   500,
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarsSubscribe, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarsSubscribe {
		t.Errorf("type: want %q got %q", FrameBarsSubscribe, env.Type)
	}
	var got BarsSubscribePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bars_subscribe mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestBarsHistorical_RoundTrip(t *testing.T) {
	want := BarsHistoricalPayload{
		Symbol:    "MNQ",
		Timeframe: "1m",
		Bars: []Bar{
			{T: 1748352000000, O: 21500.25, H: 21501.00, L: 21500.00, C: 21500.75, V: 42},
			{T: 1748352060000, O: 21500.75, H: 21501.25, L: 21500.50, C: 21501.00, V: 38},
			{T: 1748352120000, O: 21501.00, H: 21501.75, L: 21500.50, C: 21501.50, V: 51},
		},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarsHistorical, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarsHistorical {
		t.Errorf("type: want %q got %q", FrameBarsHistorical, env.Type)
	}
	var got BarsHistoricalPayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bars_historical mismatch:\n got=%+v\nwant=%+v", got, want)
	}
	// Order preservation: bars must remain strictly ascending by T after
	// round-trip. JSON arrays preserve order per RFC 8259 §5; this guards
	// against accidental future use of a map-based encoder.
	for i := 1; i < len(got.Bars); i++ {
		if got.Bars[i].T <= got.Bars[i-1].T {
			t.Errorf("bars not ascending: idx %d t=%d <= idx %d t=%d",
				i, got.Bars[i].T, i-1, got.Bars[i-1].T)
		}
	}
}

func TestBarUpdate_RoundTrip_MultiBar(t *testing.T) {
	// Multi-bar update: a single NT8 tick can fire OnBarUpdate for several
	// indices when crossing minute boundaries on a high-frequency feed.
	// Protocol §7 mandates the C# AddOn emit every bar in MinIndex..MaxIndex
	// in a single array — this test proves the array form survives encoding.
	want := BarUpdatePayload{
		Symbol:    "MNQ",
		Timeframe: "1m",
		Bars: []Bar{
			{T: 1748352000000, O: 21500.25, H: 21501.00, L: 21500.00, C: 21500.75, V: 42},
			{T: 1748352060000, O: 21500.75, H: 21501.25, L: 21500.50, C: 21501.00, V: 38},
			{T: 1748352120000, O: 21501.00, H: 21501.50, L: 21500.75, C: 21501.25, V: 17},
		},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarUpdate, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarUpdate {
		t.Errorf("type: want %q got %q", FrameBarUpdate, env.Type)
	}
	var got BarUpdatePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bar_update multi-bar mismatch:\n got=%+v\nwant=%+v", got, want)
	}
	if len(got.Bars) != 3 {
		t.Errorf("want 3 bars, got %d", len(got.Bars))
	}
}

func TestBarUpdate_RoundTrip_SingleBar(t *testing.T) {
	// Most common live case: a tick updates the current in-progress bar only.
	want := BarUpdatePayload{
		Symbol:    "MNQ",
		Timeframe: "1m",
		Bars: []Bar{
			{T: 1748352120000, O: 21501.00, H: 21501.50, L: 21500.75, C: 21501.25, V: 17},
		},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarUpdate, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarUpdate {
		t.Errorf("type: want %q got %q", FrameBarUpdate, env.Type)
	}
	var got BarUpdatePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bar_update single-bar mismatch:\n got=%+v\nwant=%+v", got, want)
	}
	if len(got.Bars) != 1 {
		t.Errorf("want 1 bar, got %d", len(got.Bars))
	}
}

func TestBarsUnsubscribe_RoundTrip_Specific(t *testing.T) {
	want := BarsUnsubscribePayload{
		Symbol:     "MNQ",
		Timeframes: []string{"1m", "5m"},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarsUnsubscribe, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarsUnsubscribe {
		t.Errorf("type: want %q got %q", FrameBarsUnsubscribe, env.Type)
	}
	var got BarsUnsubscribePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bars_unsubscribe specific mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestBarsUnsubscribe_RoundTrip_AllForSymbol(t *testing.T) {
	// Protocol §8: empty/nil Timeframes means "all timeframes for this
	// symbol". The `omitempty` tag must elide the field from the JSON so the
	// C# AddOn treats it as absent (and tears down every BarsRequest for the
	// symbol). We also assert the post-round-trip struct re-decodes with
	// Timeframes nil — a non-nil empty slice would still DeepEqual-fail.
	want := BarsUnsubscribePayload{
		Symbol:     "MNQ",
		Timeframes: nil,
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, FrameBarsUnsubscribe, want); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	// Peek at the raw envelope JSON to confirm `timeframes` is omitted on
	// the wire (this is what gives the C# side its "absent = all" semantics).
	// We have to re-marshal to inspect because ReadFrame consumes the buffer.
	rawProbe := &bytes.Buffer{}
	if err := WriteFrame(rawProbe, FrameBarsUnsubscribe, want); err != nil {
		t.Fatalf("WriteFrame probe: %v", err)
	}
	probeEnv, err := ReadFrame(rawProbe)
	if err != nil {
		t.Fatalf("ReadFrame probe: %v", err)
	}
	if bytes.Contains(probeEnv.Payload, []byte(`"timeframes"`)) {
		t.Errorf("expected omitempty to elide timeframes from wire, got payload=%s",
			string(probeEnv.Payload))
	}

	env, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if env.Type != FrameBarsUnsubscribe {
		t.Errorf("type: want %q got %q", FrameBarsUnsubscribe, env.Type)
	}
	var got BarsUnsubscribePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bars_unsubscribe all-for-symbol mismatch:\n got=%+v\nwant=%+v", got, want)
	}
	if got.Timeframes != nil {
		t.Errorf("Timeframes: want nil (absent on wire => decoded as nil), got %#v", got.Timeframes)
	}
}

// TestFrameType_WireStrings asserts the FrameType constants encode to the
// exact strings the C# AddOn switches on (protocol §5-8). A regression here
// silently breaks the AddOn's `switch (envelope.type)` dispatcher.
func TestFrameType_WireStrings(t *testing.T) {
	cases := []struct {
		ft   FrameType
		wire string
	}{
		{FrameBarsSubscribe, "bars_subscribe"},
		{FrameBarsHistorical, "bars_historical"},
		{FrameBarUpdate, "bar_update"},
		{FrameBarsUnsubscribe, "bars_unsubscribe"},
	}
	for _, tc := range cases {
		// Direct constant check (compile-time-friendly).
		if string(tc.ft) != tc.wire {
			t.Errorf("FrameType const %q != wire %q", string(tc.ft), tc.wire)
		}
		// End-to-end check: encode an empty-ish frame and confirm the
		// decoded envelope.Type matches both the constant and the wire
		// string.
		var buf bytes.Buffer
		if err := WriteFrame(&buf, tc.ft, struct{}{}); err != nil {
			t.Fatalf("WriteFrame %q: %v", tc.wire, err)
		}
		env, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("ReadFrame %q: %v", tc.wire, err)
		}
		if env.Type != tc.ft {
			t.Errorf("decoded FrameType: want %q got %q", tc.ft, env.Type)
		}
		if string(env.Type) != tc.wire {
			t.Errorf("decoded wire string: want %q got %q", tc.wire, string(env.Type))
		}
	}
}
