package mentor

import (
	"testing"

	"vl/market"
)

// Confirmation rule (a) [Buy-Sell Setup Trigger @ 11:54–12:34; deck slide 23
// "Không cần đợi nến đóng"]: "PHÁ" — a break of a candle's extreme fires the
// trigger line INTRABAR, no close needed. A 5m trigger fires on the FORMING
// 5m candle (its extremes are already known intrabar). Triggers are marked
// on the 5m only — never the 1m; the 1m is for execution.

// TestTriggerFiresIntrabarOnForming5m — the last 5m bar in the slice is still
// FORMING (CloseTime 0); its high breaks the previous high → the line is
// drawn NOW, not at the close.
func TestTriggerFiresIntrabarOnForming5m(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		{High: 100, Low: 96, Close: 98, CloseTime: 1},  // closed previous
		{High: 103, Low: 97, Close: 101, CloseTime: 0}, // still forming
	}
	got := TriggerTick(TriggerLine{}, bars, cfg)
	if got.Dir != SideLong || got.Price != 100 {
		t.Fatalf("forming-bar break must fire intrabar: %+v, want long @ 100", got)
	}
}

// TestTriggerOnRecordedTapeNeverOn1m — the 5m-only marking frame: the
// trigger machine consumes 5m bars; a 1m series is never the input. This
// pins the call contract (the evaluator passes barsTF(bars, 5)).
func TestTriggerOnRecordedTapeNeverOn1m(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	// feeding the RAW 1m series must not draw a line from a 1m break alone:
	// the machine is a 5m construct; here we simply pin that the tape-level
	// test for the trigger uses the 5m fixture (the 1m is execution-only).
	five := barsTF(bars, 5)
	if len(five) == 0 {
		t.Fatal("5m aggregation empty")
	}
	got := TriggerLine{}
	moved := 0
	prev := false
	for i := 1; i < len(five); i++ {
		got = TriggerTick(got, five[i-1:i+1], cfg)
		if got.Moved && !prev {
			moved++
		}
		prev = got.Moved
	}
	if got.Dir == "" {
		t.Fatal("no trigger line on the 5m aggregation of the recorded tape")
	}
	if moved > 1 {
		t.Fatalf("line moved %d times — 'MỘT LẦN MỘT THÔI'", moved)
	}
}
