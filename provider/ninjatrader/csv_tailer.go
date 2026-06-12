package ninjatrader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CSVTailer reads new rows appended to trades_taken.csv by vltrader.cs.
type CSVTailer struct {
	dataDir      string
	pollInterval time.Duration
	seen         int // rows already delivered (excluding header)
}

func NewCSVTailer(dataDir string, pollInterval time.Duration) *CSVTailer {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &CSVTailer{dataDir: dataDir, pollInterval: pollInterval}
}

func (t *CSVTailer) FillsPath() string {
	return filepath.Join(t.dataDir, "trades_taken.csv")
}

// TailFills blocks until ctx is cancelled. For each new fill row appended to
// the file, the callback is invoked synchronously. Reset on file-shrink (e.g.
// if NT cycles the file at session boundary).
func (t *CSVTailer) TailFills(ctx context.Context, onFill func(FillRow)) error {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := t.readNew(onFill); err != nil {
				fmt.Fprintf(os.Stderr, "ninjatrader tailer: %v\n", err)
			}
		}
	}
}

func (t *CSVTailer) readNew(onFill func(FillRow)) error {
	f, err := os.Open(t.FillsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file may not exist yet — first fill creates it
		}
		return err
	}
	defer f.Close()

	rows := []string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			rows = append(rows, line)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Strip header if present
	if len(rows) > 0 && strings.HasPrefix(rows[0], "DateTime,") {
		rows = rows[1:]
	}

	// File shrunk → NT cycled; reset our cursor (hazard H4 noted in CLAUDE.md;
	// not fully fixed here — see Plan 1.5 for stable-fill-ID dedup).
	if len(rows) < t.seen {
		t.seen = 0
	}

	for i := t.seen; i < len(rows); i++ {
		fill, err := parseFillRow(rows[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ninjatrader tailer: parse %q: %v\n", rows[i], err)
			continue
		}
		onFill(fill)
	}
	t.seen = len(rows)
	return nil
}

func parseFillRow(line string) (FillRow, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return FillRow{}, fmt.Errorf("expected 3 fields, got %d", len(parts))
	}
	price, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return FillRow{}, fmt.Errorf("parse entry price: %w", err)
	}
	return FillRow{
		DateTime:   strings.TrimSpace(parts[0]),
		Direction:  strings.ToUpper(strings.TrimSpace(parts[1])),
		EntryPrice: price,
	}, nil
}
