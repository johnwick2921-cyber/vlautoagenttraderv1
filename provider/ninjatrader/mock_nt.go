// Package ninjatrader — mock harness for the NT8 CSV bridge.
//
// StartMockNT simulates the NinjaTrader-side polling loop in a goroutine. It
// is intended for tests and the cmd/nq_smoke roundtrip — never for production.
// The real NinjaTrader polls trade_signals.csv every 2 seconds; the mock polls
// every 100ms so tests stay fast.
//
// The mock dedups by (DateTime + Direction + EntryPrice) so an unchanged
// signal written twice produces exactly one fill, matching NT's behaviour
// (NT dedups by DateTime+Direction at 1-second resolution).
package ninjatrader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// StartMockNT spawns a background goroutine that simulates NinjaTrader's
// vltrader.cs polling loop. For each newly observed signal in
// trade_signals.csv, the mock writes a synthetic fill to trades_taken.csv
// after fillDelay.
//
// Fill semantics:
//   - DateTime is "now" formatted as MM/dd/yyyy HH:mm:ss (the real NT writes
//     the wall-clock time of the fill, not the signal time).
//   - Direction is copied verbatim from the signal.
//   - EntryPrice is copied verbatim — no slippage modelled (tests assert
//     exact match against the rounded entry the trader sent).
//
// The returned stop function cancels the polling loop and waits briefly so
// callers can teardown the data directory without racing an in-flight read.
func StartMockNT(dataDir string, fillDelay time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	seen := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processSignals(ctx, dataDir, seen, &mu, &wg, fillDelay)
			}
		}
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}

// processSignals reads the current trade_signals.csv and schedules a fill
// for any row not yet seen. The CSV writer truncates the file on every
// WriteSignal, so the file contains at most one data row (plus header) at
// any given time — but we iterate defensively in case that ever changes.
func processSignals(ctx context.Context, dataDir string, seen map[string]struct{}, mu *sync.Mutex, wg *sync.WaitGroup, fillDelay time.Duration) {
	signalsPath := filepath.Join(dataDir, "trade_signals.csv")
	fillsPath := filepath.Join(dataDir, "trades_taken.csv")

	f, err := os.Open(signalsPath)
	if err != nil {
		return // file may not exist yet — first WriteSignal creates it
	}
	defer f.Close()

	var rows [][]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// Skip header row produced by csv_writer.go (signalsHeader).
		if strings.HasPrefix(line, "DateTime,") {
			continue
		}
		rows = append(rows, strings.Split(line, ","))
	}
	if err := sc.Err(); err != nil {
		return
	}

	for _, row := range rows {
		// SignalRow has 5 fields: DateTime, Direction, EntryPrice, StopLoss, TakeProfit
		if len(row) < 5 {
			continue
		}
		key := row[0] + "|" + row[1] + "|" + row[2]
		mu.Lock()
		if _, ok := seen[key]; ok {
			mu.Unlock()
			continue
		}
		seen[key] = struct{}{}
		mu.Unlock()

		// Schedule the fill asynchronously after fillDelay. Track via wg so
		// stop() can wait for any in-flight writes to complete.
		wg.Add(1)
		go func(signalRow []string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case <-time.After(fillDelay):
			}
			writeFill(fillsPath, signalRow)
		}(row)
	}
}

// writeFill appends one synthetic fill row to trades_taken.csv. The file is
// created with the standard fillsHeader if it does not exist. This matches
// the contract csv_tailer.go expects.
func writeFill(fillsPath string, signalRow []string) {
	direction := strings.TrimSpace(signalRow[1])
	entryPrice, err := strconv.ParseFloat(strings.TrimSpace(signalRow[2]), 64)
	if err != nil {
		return
	}

	// Ensure the file exists with the canonical header on first write.
	if _, statErr := os.Stat(fillsPath); os.IsNotExist(statErr) {
		hdr, err := os.OpenFile(fillsPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		_, _ = hdr.WriteString(fillsHeader + "\n")
		_ = hdr.Close()
	}

	fillTime := time.Now().Format("01/02/2006 15:04:05")
	fillRow := fmt.Sprintf("%s,%s,%.2f\n", fillTime, strings.ToUpper(direction), entryPrice)

	f, err := os.OpenFile(fillsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(fillRow)
}
