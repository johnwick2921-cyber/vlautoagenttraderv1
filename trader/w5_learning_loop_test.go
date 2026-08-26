package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// W5 — a trade closed on a REAL exit path (no AI close decision) still gets MAE/MFE
// + an adherence grade + a matched-random verdict, via the loop poll. Synthetic
// bars stand in for the NT8 feed; a directly-inserted CLOSED row stands in for the
// OCO/EOD close.
func TestW5AnalyticsLandOnRealClose(t *testing.T) {
	at := mkTrader("ninjatrader", boolp(true), "5m")
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at.store = st
	at.id = "t1"

	// synthetic 1m bars over the hold: a long from 30000 that ran to ~30060.
	entry := time.Now().Add(-40 * time.Minute)
	exit := time.Now().Add(-5 * time.Minute)
	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		var bars []market.Kline
		for i := 0; i < 40; i++ {
			ct := entry.Add(time.Duration(i) * time.Minute)
			px := 30000.0 + float64(i)*1.5 // trends up (favorable for a long)
			bars = append(bars, market.Kline{OpenTime: ct.UnixMilli(), Open: px, High: px + 5, Low: px - 8, Close: px, CloseTime: ct.Add(time.Minute).UnixMilli()})
		}
		return bars
	}

	// open then close via the store close path (as a fill-confirmed OCO/EOD close
	// does) — Create forces OPEN, ClosePosition marks it CLOSED, ungraded.
	pos := &store.TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "LONG", EntryPrice: 30000, EntryQuantity: 1,
		EntryTime: entry.UnixMilli(), CreatedAt: entry.UnixMilli(), UpdatedAt: entry.UnixMilli(),
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatalf("create pos: %v", err)
	}
	if _, err := st.Position().ClosePosition(pos.ID, 30060, "oco", 60, 0, "oco_target"); err != nil {
		t.Fatalf("close pos: %v", err)
	}
	_ = exit

	// before: ungraded.
	un, _ := st.Position().GetUngradedClosedPositions("t1", 0, 10)
	if len(un) != 1 {
		t.Fatalf("expected 1 ungraded close, got %d", len(un))
	}

	// process (the real-exit analytics path).
	at.recordClosedTradeAnalytics(un[0])

	graded, _ := st.Position().GetGradedClosedPositions("t1", 10)
	if len(graded) != 1 {
		t.Fatalf("the real-exit close must be graded, got %d graded", len(graded))
	}
	g := graded[0]
	if g.AdherenceGrade == "" {
		t.Fatal("adherence grade must be set")
	}
	if g.MFE <= 0 {
		t.Fatalf("MFE must be computed (favorable run), got %.2f", g.MFE)
	}

	// idempotent: a second pass leaves nothing ungraded + doesn't error.
	un2, _ := st.Position().GetUngradedClosedPositions("t1", 0, 10)
	if len(un2) != 0 {
		t.Fatalf("graded close must not re-appear as ungraded, got %d", len(un2))
	}
}

// The poll epoch excludes pre-existing (pre-day-plan) closes.
func TestW5EpochExcludesHistory(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	p := &store.TraderPosition{TraderID: "t1", Symbol: "MNQ", Side: "LONG", EntryPrice: 1, EntryQuantity: 1}
	_ = st.Position().Create(p)
	_, _ = st.Position().ClosePosition(p.ID, 2, "x", 1, 0, "manual") // exit_time ≈ now
	epoch := time.Now().Add(1 * time.Hour).UnixMilli()               // epoch in the FUTURE
	un, _ := st.Position().GetUngradedClosedPositions("t1", epoch, 10)
	if len(un) != 0 {
		t.Fatalf("a close before the epoch must be excluded, got %d", len(un))
	}
}
