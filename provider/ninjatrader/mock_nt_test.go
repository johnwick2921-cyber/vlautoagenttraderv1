package ninjatrader

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// readFills returns the data rows (header stripped) in trades_taken.csv.
// Returns an empty slice if the file does not exist yet.
func readFills(t *testing.T, dir string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, "trades_taken.csv"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("open fills: %v", err)
	}
	defer f.Close()

	var rows []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "DateTime,") {
			continue
		}
		rows = append(rows, line)
	}
	return rows
}

// waitForFillCount polls trades_taken.csv until at least want rows are
// present or the deadline elapses. Returns the final rows observed.
func waitForFillCount(t *testing.T, dir string, want int, timeout time.Duration) []string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var rows []string
	for time.Now().Before(deadline) {
		rows = readFills(t, dir)
		if len(rows) >= want {
			return rows
		}
		time.Sleep(20 * time.Millisecond)
	}
	return rows
}

func TestMockNT_SingleSignal_ProducesOneFill(t *testing.T) {
	dir := t.TempDir()
	stop := StartMockNT(dir, 100*time.Millisecond)
	defer stop()

	w := NewCSVWriter(dir)
	if err := w.WriteSignal(SignalRow{
		DateTime:   "01/02/2026 09:30:00",
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21450.00,
		TakeProfit: 21550.00,
	}); err != nil {
		t.Fatalf("WriteSignal: %v", err)
	}

	rows := waitForFillCount(t, dir, 1, 1*time.Second)
	if len(rows) != 1 {
		t.Fatalf("expected 1 fill, got %d: %v", len(rows), rows)
	}
	if !strings.Contains(rows[0], "LONG") {
		t.Errorf("fill missing LONG direction: %q", rows[0])
	}
	if !strings.Contains(rows[0], "21500.00") {
		t.Errorf("fill missing entry price 21500.00: %q", rows[0])
	}
}

func TestMockNT_MultipleSignals_PreserveDirections(t *testing.T) {
	dir := t.TempDir()
	stop := StartMockNT(dir, 50*time.Millisecond)
	defer stop()

	w := NewCSVWriter(dir)
	signals := []SignalRow{
		{DateTime: "01/02/2026 09:30:00", Direction: "LONG", EntryPrice: 21500.00, StopLoss: 21450.00, TakeProfit: 21550.00},
		{DateTime: "01/02/2026 09:30:05", Direction: "SHORT", EntryPrice: 21490.00, StopLoss: 21540.00, TakeProfit: 21440.00},
		{DateTime: "01/02/2026 09:30:10", Direction: "LONG", EntryPrice: 21495.00, StopLoss: 21445.00, TakeProfit: 21545.00},
	}
	for i, sig := range signals {
		if err := w.WriteSignal(sig); err != nil {
			t.Fatalf("WriteSignal[%d]: %v", i, err)
		}
		// Sleep > mock poll interval (100ms) so the mock observes each
		// signal before the writer truncates the file with the next one.
		time.Sleep(150 * time.Millisecond)
	}

	rows := waitForFillCount(t, dir, 3, 2*time.Second)
	if len(rows) != 3 {
		t.Fatalf("expected 3 fills, got %d: %v", len(rows), rows)
	}

	wantDirs := []string{"LONG", "SHORT", "LONG"}
	for i, want := range wantDirs {
		if !strings.Contains(rows[i], want) {
			t.Errorf("fill[%d]: want direction %q, got %q", i, want, rows[i])
		}
	}
}

func TestMockNT_Stop_HaltsLoop(t *testing.T) {
	dir := t.TempDir()
	stop := StartMockNT(dir, 50*time.Millisecond)

	w := NewCSVWriter(dir)
	// First signal — observed before stop.
	if err := w.WriteSignal(SignalRow{
		DateTime:   "01/02/2026 09:30:00",
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21450.00,
		TakeProfit: 21550.00,
	}); err != nil {
		t.Fatalf("WriteSignal: %v", err)
	}
	rows := waitForFillCount(t, dir, 1, 1*time.Second)
	if len(rows) != 1 {
		t.Fatalf("expected 1 fill before stop, got %d", len(rows))
	}

	stop()

	// Second signal — should NOT be picked up after stop.
	if err := w.WriteSignal(SignalRow{
		DateTime:   "01/02/2026 09:31:00",
		Direction:  "SHORT",
		EntryPrice: 21490.00,
		StopLoss:   21540.00,
		TakeProfit: 21440.00,
	}); err != nil {
		t.Fatalf("WriteSignal: %v", err)
	}
	time.Sleep(400 * time.Millisecond)

	rows = readFills(t, dir)
	if len(rows) != 1 {
		t.Fatalf("expected fill count to stay at 1 after stop, got %d: %v", len(rows), rows)
	}
}

func TestMockNT_Dedup_SameSignalNoDoubleFill(t *testing.T) {
	dir := t.TempDir()
	stop := StartMockNT(dir, 50*time.Millisecond)
	defer stop()

	w := NewCSVWriter(dir)
	sig := SignalRow{
		DateTime:   "01/02/2026 09:30:00",
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21450.00,
		TakeProfit: 21550.00,
	}
	if err := w.WriteSignal(sig); err != nil {
		t.Fatalf("WriteSignal 1: %v", err)
	}
	// Give the mock time to consume the first signal.
	time.Sleep(200 * time.Millisecond)
	// Write the IDENTICAL signal a second time — dedup should suppress it.
	if err := w.WriteSignal(sig); err != nil {
		t.Fatalf("WriteSignal 2: %v", err)
	}
	time.Sleep(400 * time.Millisecond)

	rows := readFills(t, dir)
	if len(rows) != 1 {
		t.Fatalf("expected 1 fill (dedup), got %d: %v", len(rows), rows)
	}
}
