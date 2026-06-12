package ninjatrader

import (
	"testing"
	"time"

	ntpkg "nofx/provider/ninjatrader"
)

// TestTrader_RoundTrip_OpenLong exercises the full Go → CSV → mock NT →
// CSV → tailer → Trader.GetPositions loop for a LONG entry.
func TestTrader_RoundTrip_OpenLong(t *testing.T) {
	dir := t.TempDir()
	stop := ntpkg.StartMockNT(dir, 100*time.Millisecond)
	defer stop()

	tr := New(Config{DataDir: dir, Symbol: "MNQ"})

	if err := tr.SetStopLoss("MNQ", "LONG", 1, 21450); err != nil {
		t.Fatalf("SetStopLoss: %v", err)
	}
	if err := tr.SetTakeProfit("MNQ", "LONG", 1, 21550); err != nil {
		t.Fatalf("SetTakeProfit: %v", err)
	}
	if _, err := tr.OpenLong("MNQ", 1, 1); err != nil {
		t.Fatalf("OpenLong: %v", err)
	}

	// Tailer polls every 1s (NewCSVTailer default in trader.New), and the
	// mock fill arrives ~100ms after the signal. Allow generous time.
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		positions, err := tr.GetPositions()
		if err != nil {
			t.Fatalf("GetPositions: %v", err)
		}
		if len(positions) > 0 {
			pos := positions[0]
			if side, _ := pos["side"].(string); side != "LONG" {
				t.Fatalf("expected side LONG, got %v", pos["side"])
			}
			// Entry price should equal the midpoint of (SL, TP) rounded to
			// the NQ/MNQ tick (0.25). Midpoint = (21450 + 21550) / 2 = 21500.
			if entry, _ := pos["entryPrice"].(float64); entry != 21500.0 {
				t.Errorf("expected entry 21500, got %v", entry)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no fill observed within 4s")
}

// TestTrader_RoundTrip_OpenShort mirrors OpenLong for the SHORT side.
func TestTrader_RoundTrip_OpenShort(t *testing.T) {
	dir := t.TempDir()
	stop := ntpkg.StartMockNT(dir, 100*time.Millisecond)
	defer stop()

	tr := New(Config{DataDir: dir, Symbol: "MNQ"})

	if err := tr.SetStopLoss("MNQ", "SHORT", 1, 21550); err != nil {
		t.Fatalf("SetStopLoss: %v", err)
	}
	if err := tr.SetTakeProfit("MNQ", "SHORT", 1, 21450); err != nil {
		t.Fatalf("SetTakeProfit: %v", err)
	}
	if _, err := tr.OpenShort("MNQ", 1, 1); err != nil {
		t.Fatalf("OpenShort: %v", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		positions, err := tr.GetPositions()
		if err != nil {
			t.Fatalf("GetPositions: %v", err)
		}
		if len(positions) > 0 {
			pos := positions[0]
			if side, _ := pos["side"].(string); side != "SHORT" {
				t.Fatalf("expected side SHORT, got %v", pos["side"])
			}
			if entry, _ := pos["entryPrice"].(float64); entry != 21500.0 {
				t.Errorf("expected entry 21500, got %v", entry)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no fill observed within 4s")
}

// TestTrader_RoundTrip_NoSLTP_Rejected confirms the trader refuses to write
// a signal when SetStopLoss/SetTakeProfit have not been called.
func TestTrader_RoundTrip_NoSLTP_Rejected(t *testing.T) {
	dir := t.TempDir()
	// No mock needed — we never expect a signal to reach disk.
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})

	if _, err := tr.OpenLong("MNQ", 1, 1); err == nil {
		t.Fatal("expected OpenLong to error without SL/TP, got nil")
	}
	if _, err := tr.OpenShort("MNQ", 1, 1); err == nil {
		t.Fatal("expected OpenShort to error without SL/TP, got nil")
	}
}
