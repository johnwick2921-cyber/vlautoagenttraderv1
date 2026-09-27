package ninjatrader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_Smoke(t *testing.T) {
	dir := t.TempDir()
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})
	if tr == nil {
		t.Fatal("New returned nil")
	}
}

func TestOpenLong_WritesSignal(t *testing.T) {
	dir := t.TempDir()
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})

	// Stash SL/TP first — they get bundled into the order on OpenLong.
	_ = tr.SetStopLoss("MNQ", "LONG", 1, 21480.00)
	_ = tr.SetTakeProfit("MNQ", "LONG", 1, 21540.00)

	res, err := tr.OpenLong("MNQ", 1, 1)
	if err != nil {
		t.Fatalf("OpenLong: %v", err)
	}
	if res["status"] != "submitted" {
		t.Errorf("status = %v, want submitted", res["status"])
	}

	body, _ := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	if !strings.Contains(string(body), "LONG") {
		t.Errorf("signal file missing LONG row:\n%s", string(body))
	}
	if !strings.Contains(string(body), "21480.00") || !strings.Contains(string(body), "21540.00") {
		t.Errorf("signal file missing SL/TP:\n%s", string(body))
	}
}

// TestCSVMaintenancePermit (WAVE 1a-plan N5) — the CSV transport takes the
// installation maintenance permit around the signal write: a refused permit
// refuses the entry with ErrMaintenanceHold and writes NOTHING; an allowed
// permit writes the signal.
func TestCSVMaintenancePermit(t *testing.T) {
	dir := t.TempDir()
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})
	_ = tr.SetStopLoss("MNQ", "LONG", 1, 21480.00)
	_ = tr.SetTakeProfit("MNQ", "LONG", 1, 21540.00)

	tr.SetEntryPermit(func() (func(), bool) { return nil, false })
	if _, err := tr.OpenLong("MNQ", 1, 1); err == nil || !errors.Is(err, ErrMaintenanceHold) {
		t.Fatalf("a refused permit must refuse the entry with ErrMaintenanceHold, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "trade_signals.csv")); !os.IsNotExist(err) {
		t.Fatalf("a refused permit must write NOTHING, stat err=%v", err)
	}

	tr.SetEntryPermit(func() (func(), bool) { return func() {}, true })
	if _, err := tr.OpenLong("MNQ", 1, 1); err != nil {
		t.Fatalf("an allowed permit must write the signal, got %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "trade_signals.csv")); err != nil || !strings.Contains(string(body), "LONG") {
		t.Fatalf("signal file must carry the LONG row after the permit allows, err=%v body=%s", err, body)
	}
}
