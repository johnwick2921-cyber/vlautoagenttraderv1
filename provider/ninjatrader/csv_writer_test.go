package ninjatrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVWriter_WriteSignal_LongValid(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	sig := SignalRow{
		DateTime:   "05/22/2026 14:30:15",
		Direction:  "LONG",
		EntryPrice: 21505.00,
		StopLoss:   21485.00,
		TakeProfit: 21545.00,
	}
	if err := w.WriteSignal(sig); err != nil {
		t.Fatalf("WriteSignal: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit\n05/22/2026 14:30:15,LONG,21505.00,21485.00,21545.00\n"
	if string(got) != want {
		t.Errorf("file content:\n  got:  %q\n  want: %q", string(got), want)
	}
}

func TestCSVWriter_WriteSignal_RejectsBadLong(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	bad := SignalRow{
		DateTime:   "05/22/2026 14:30:15",
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21520.00, // wrong: stop ABOVE entry for a long
		TakeProfit: 21550.00,
	}
	err := w.WriteSignal(bad)
	if err == nil {
		t.Fatal("want validation error, got nil")
	}
	if !strings.Contains(err.Error(), "LONG stop_loss") {
		t.Errorf("error message %q does not mention LONG stop_loss", err.Error())
	}
}

func TestCSVWriter_WriteSignal_TruncatesPrevious(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	first := SignalRow{DateTime: "05/22/2026 14:30:15", Direction: "LONG", EntryPrice: 100, StopLoss: 90, TakeProfit: 110}
	second := SignalRow{DateTime: "05/22/2026 14:32:00", Direction: "SHORT", EntryPrice: 200, StopLoss: 210, TakeProfit: 190}

	if err := w.WriteSignal(first); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteSignal(second); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	// Should contain ONLY the second signal (plus header), not both.
	if strings.Contains(string(got), "14:30:15") {
		t.Errorf("expected first signal to be truncated, but file still contains it:\n%s", string(got))
	}
	if !strings.Contains(string(got), "14:32:00") {
		t.Errorf("expected second signal in file:\n%s", string(got))
	}
}
