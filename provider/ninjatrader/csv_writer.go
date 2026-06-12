package ninjatrader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"nofx/internal/retry"
)

// CSVWriter writes trade signals to a Windows-shared CSV file that
// NinjaTrader's vltrader.cs polls every 2 seconds.
type CSVWriter struct {
	dataDir string
	mu      sync.Mutex
}

func NewCSVWriter(dataDir string) *CSVWriter {
	return &CSVWriter{dataDir: dataDir}
}

// SignalsPath returns the absolute path to trade_signals.csv.
func (w *CSVWriter) SignalsPath() string {
	return filepath.Join(w.dataDir, "trade_signals.csv")
}

// WriteSignal validates and writes one signal. The file is truncated and
// rewritten with header + this single row — vltrader.cs clears the file
// after processing, so we always overwrite to avoid stale rows accumulating.
//
// Atomicity: write to temp file then os.Rename. NT polls every 2s; without
// atomic update NT could read a partial file mid-write.
func (w *CSVWriter) WriteSignal(s SignalRow) error {
	if err := s.Validate(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	row := fmt.Sprintf("%s,%s,%.2f,%.2f,%.2f",
		s.DateTime,
		strings.ToUpper(s.Direction),
		s.EntryPrice,
		s.StopLoss,
		s.TakeProfit,
	)
	content := signalsHeader + "\n" + row + "\n"

	tmp, err := os.CreateTemp(w.dataDir, "trade_signals.*.tmp")
	if err != nil {
		return fmt.Errorf("ninjatrader writer: create temp: %w", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: close temp: %w", err)
	}
	// Plan 4 Task 24 — retry rename to handle Windows EBUSY (Defender scanner lock)
	renameErr := retry.RetryWithBackoff(context.Background(), 3, func() error {
		return os.Rename(tmp.Name(), w.SignalsPath())
	})
	if renameErr != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: rename temp: %w", renameErr)
	}
	return nil
}
