package ninjatrader

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCSVTailer_DetectsAppendedRows(t *testing.T) {
	dir := t.TempDir()
	fillsPath := filepath.Join(dir, "trades_taken.csv")
	// Seed with header
	if err := os.WriteFile(fillsPath, []byte(fillsHeader+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		mu    sync.Mutex
		fills []FillRow
	)
	cb := func(f FillRow) {
		mu.Lock()
		defer mu.Unlock()
		fills = append(fills, f)
	}

	tailer := NewCSVTailer(dir, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tailer.TailFills(ctx, cb) }()

	// Allow tailer to read initial state (header only, no rows)
	time.Sleep(150 * time.Millisecond)

	// Append two fills (NT-style)
	f, err := os.OpenFile(fillsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("05/22/2026 14:30:15,LONG,21505.25\n")
	f.WriteString("05/22/2026 14:35:42,SHORT,21520.50\n")
	f.Close()

	// Wait for tailer to pick them up
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(fills)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(fills) != 2 {
		t.Fatalf("got %d fills, want 2: %+v", len(fills), fills)
	}
	if fills[0].Direction != "LONG" || fills[0].EntryPrice != 21505.25 {
		t.Errorf("fill[0] = %+v", fills[0])
	}
	if fills[1].Direction != "SHORT" || fills[1].EntryPrice != 21520.50 {
		t.Errorf("fill[1] = %+v", fills[1])
	}
}
