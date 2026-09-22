package ninjatrader

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// LIVE BAR SINK (W-PICTURE-HTF, 2026-09-20) — pins for the deterministic
// evaluator fan-out: live frames are delivered, historical replays are NOT
// (a replay receipt can never mint a real opportunity).

func TestFanOutLiveBarsDelivers(t *testing.T) {
	got := make(chan liveSinkMsg, 2)
	SetLiveBarSink(func(symbol, tf string, bars []Bar, receivedAt time.Time) {
		got <- liveSinkMsg{symbol: symbol, tf: tf, bars: bars, receivedAt: receivedAt}
	})
	t.Cleanup(func() { SetLiveBarSink(nil) })

	fanOutLiveBars("MNQ", "5m", []Bar{{T: 1, C: 101.5}})
	select {
	case m := <-got:
		if m.symbol != "MNQ" || m.tf != "5m" || len(m.bars) != 1 || m.bars[0].C != 101.5 {
			t.Fatalf("delivered frame mismatch: %+v", m)
		}
		if m.receivedAt.IsZero() {
			t.Fatalf("receivedAt must be stamped at the drain instant")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("live frame never delivered")
	}
}

func TestFanOutLiveBarsEmptyNoOp(t *testing.T) {
	SetLiveBarSink(func(symbol, tf string, bars []Bar, receivedAt time.Time) {
		t.Fatalf("empty batch must not fan out")
	})
	t.Cleanup(func() { SetLiveBarSink(nil) })
	fanOutLiveBars("MNQ", "5m", nil)
	time.Sleep(50 * time.Millisecond)
}

func TestDrainBarIngestHistoricalNeverFansOut(t *testing.T) {
	live := make(chan string, 1)
	SetLiveBarSink(func(symbol, tf string, bars []Bar, receivedAt time.Time) {
		live <- tf
	})
	t.Cleanup(func() { SetLiveBarSink(nil) })

	s := NewTCPServer(nil)
	s.barIngestCh = make(chan barIngestMsg, 4)
	s.wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); s.wg.Wait() }()
	go s.drainBarIngest(ctx)

	s.barIngestCh <- barIngestMsg{historical: true, symbol: "MNQ", timeframe: "5m", bars: []Bar{{T: 1}}}
	select {
	case tf := <-live:
		t.Fatalf("historical replay fanned out (%s) — replay receipts must never mint opportunities", tf)
	case <-time.After(200 * time.Millisecond):
	}
	s.barIngestCh <- barIngestMsg{historical: false, symbol: "MNQ", timeframe: "5m", bars: []Bar{{T: 2}}}
	select {
	case tf := <-live:
		if tf != "5m" {
			t.Fatalf("live frame delivered with tf %q", tf)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("live frame never delivered")
	}
}

func TestBarFinalEvidenceRoundTrip(t *testing.T) {
	b := Bar{T: 1, O: 2, H: 3, L: 1.5, C: 2.5, V: 10, Final: true, EmittedAt: 1750000000123}
	body, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"final":true`) || !strings.Contains(string(body), `"emitted_at":1750000000123`) {
		t.Fatalf("final/emitted_at must ride the wire: %s", body)
	}
	var back Bar
	if err := json.Unmarshal(body, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Final || back.EmittedAt != 1750000000123 {
		t.Fatalf("evidence lost in roundtrip: %+v", back)
	}
	// A forming bar without the markers stays unproven (zero value), never
	// "closed".
	fb := Bar{T: 2, C: 3}
	body, _ = json.Marshal(fb)
	if strings.Contains(string(body), "final") {
		t.Fatalf("an unproven bar must not claim finality: %s", body)
	}
}

// W-PICTURE-HTF (2026-09-20) — the LABELED Sept-17 replay: a historical tape
// that would be a perfect two-picture setup (4H resistance 101, H1 pair
// crossing it, 5m swing) must be RECEIVED (the cache holds the bars) but its
// receipts are NEVER claimed — the deterministic evaluator is live-frames
// only, and a replay receipt cannot mint a real opportunity.
func TestSept17ReplayNeverMintsOpportunities(t *testing.T) {
	sinkCalls := make(chan string, 16)
	SetLiveBarSink(func(symbol, tf string, bars []Bar, receivedAt time.Time) {
		sinkCalls <- tf
	})
	t.Cleanup(func() { SetLiveBarSink(nil) })

	s := NewTCPServer(nil)
	s.barIngestCh = make(chan barIngestMsg, 16)
	s.wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); s.wg.Wait() }()
	go s.drainBarIngest(ctx)

	sep17 := time.Date(2026, 9, 17, 5, 0, 0, 0, time.UTC).UnixMilli()
	bars4h := []Bar{
		{T: sep17, O: 98, H: 99, L: 97, C: 98.5, Final: true},
		{T: sep17 + 4*3600*1000, O: 99, H: 105, L: 96, C: 101, Final: true}, // the resistance pivot
		{T: sep17 + 8*3600*1000, O: 96, H: 97, L: 94, C: 95, Final: true},
		{T: sep17 + 12*3600*1000, O: 94, H: 95, L: 92, C: 93, Final: true},
	}
	barsH1 := []Bar{
		{T: sep17 + 16*3600*1000, O: 100.8, H: 101.3, L: 100.5, C: 101.0, Final: true},
		{T: sep17 + 17*3600*1000, O: 101.0, H: 102.0, L: 100.6, C: 101.5, Final: true}, // the break
	}
	bars5m := []Bar{
		{T: sep17 + 17*3600*1000 + 25*5*60*1000, O: 98.6, H: 99.2, L: 98.5, C: 98.9, Final: true}, // the swing
		{T: sep17 + 17*3600*1000 + 26*5*60*1000, O: 99.0, H: 99.6, L: 98.8, C: 99.4, Final: true},
		{T: sep17 + 17*3600*1000 + 27*5*60*1000, O: 99.5, H: 100.0, L: 99.2, C: 99.8, Final: true},
	}
	s.barIngestCh <- barIngestMsg{historical: true, symbol: "MNQ", timeframe: "4h", bars: bars4h}
	s.barIngestCh <- barIngestMsg{historical: true, symbol: "MNQ", timeframe: "1h", bars: barsH1}
	s.barIngestCh <- barIngestMsg{historical: true, symbol: "MNQ", timeframe: "5m", bars: bars5m}

	// The replay is RECEIVED: the cache holds the bars.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(s.BarCache().Get("MNQ", "4h")) == 4 && len(s.BarCache().Get("MNQ", "1h")) == 2 && len(s.BarCache().Get("MNQ", "5m")) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(s.BarCache().Get("MNQ", "5m")) != 3 {
		t.Fatalf("the replay must be received (cache), got 5m=%d", len(s.BarCache().Get("MNQ", "5m")))
	}
	// …but its receipts are NEVER claimed: no live fan-out, no opportunity.
	select {
	case tf := <-sinkCalls:
		t.Fatalf("Sept-17 replay fanned out (%s) — a replay receipt must never mint an opportunity", tf)
	case <-time.After(200 * time.Millisecond):
	}
}
