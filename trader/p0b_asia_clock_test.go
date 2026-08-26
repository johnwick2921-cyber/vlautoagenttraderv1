package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// P0-B (2026-08-18) — ASIA CLOCK. Two defects, two guarantees:
//  1. the designed 16:55 read must fire even though IsCMEOpen(now) is false
//     during the 16:00–17:00 CME maintenance break (it builds from STORED data);
//  2. one session instance maps to EXACTLY one plan chain across the midnight
//     trade-date roll — no second plan at 00:30 CT.

func asiaClockTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "asia.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	enabled := true
	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true, Sessions: []store.DayPlanSessionOverride{
				{Session: "ASIA", Enable: &enabled},
			}},
		}},
	}
	at.mcpClient = &planClient{} // planner returns validTraderPlanJSON
	return at, st
}

func ctTime(t *testing.T, y int, mo time.Month, d, hh, mm int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	return time.Date(y, mo, d, hh, mm, 0, 0, loc)
}

func TestP0BAsiaReadFiresAt1655WhileMarketClosed(t *testing.T) {
	at, st := asiaClockTrader(t)
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		// Stored-data path: supply a realistic prior session so DailyATRProxy
		// resolves (~100 pts) instead of the thin one-bar fallback that the
		// P0.2 target-reachability rule would trip over.
		now := time.Now().UnixMilli()
		bars := make([]market.Kline, 0, 390)
		base := now - 400*60_000
		for i := 0; i < 390; i++ {
			o := base + int64(i)*60_000
			bars = append(bars, market.Kline{OpenTime: o, High: 15650 + float64(i%10), Low: 15550 + float64(i%10), Close: 15600 + float64(i%10), CloseTime: o + 59_000})
		}
		return bars
	}

	// Tuesday 16:55 CT — the CME maintenance break. IsCMEOpen(now) == false.
	now := ctTime(t, 2026, 8, 18, 16, 55)
	if kernel.IsCMEOpen(now) {
		t.Fatalf("fixture: 16:55 must be inside the maintenance break (IsCMEOpen=false)")
	}

	at.maybeRunSessionReadsAt(now)

	row, err := st.Plan().GetLatestPlanForTraderSession("2026-08-18", "ASIA", "t1")
	if err != nil || row == nil {
		t.Fatalf("the 16:55 ASIA read must fire while the market is closed, row=%v err=%v", row, err)
	}
	if row.Lifecycle != "active" || row.TriggerReason != "ASIA_scheduled_read" {
		t.Fatalf("plan row wrong: %+v", row)
	}
}

func TestP0BAsiaReadDoesNotFireOutsideItsWindow(t *testing.T) {
	at, st := asiaClockTrader(t)
	// 16:30 CT — before ReadCT 16:55. Nothing may fire.
	at.maybeRunSessionReadsAt(ctTime(t, 2026, 8, 18, 16, 30))
	if row, _ := st.Plan().GetLatestPlanForTraderSession("2026-08-18", "ASIA", "t1"); row != nil {
		t.Fatalf("16:30 is outside the read window — no plan may be written, got %+v", row)
	}
	// Sunday 16:55: the session INSTANCE opens Sunday 17:00 (live Globex) —
	// reading is correct.
	at.maybeRunSessionReadsAt(ctTime(t, 2026, 8, 23, 16, 55))
	if row, _ := st.Plan().GetLatestPlanForTraderSession("2026-08-23", "ASIA", "t1"); row == nil {
		t.Fatalf("Sunday 17:00 is a live session open — the 16:55 Sunday read must fire")
	}
}

func TestP0BMidnightRollWritesNoSecondPlan(t *testing.T) {
	at, st := asiaClockTrader(t)
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		now := time.Now().UnixMilli()
		return []market.Kline{{OpenTime: now - 600_000, High: 15610, Low: 15590, Close: 15600, CloseTime: now - 300_000}}
	}

	// The 08-18 ASIA instance (opens 17:00 Tue) already has its chain.
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID("2026-08-18", "ASIA"), StrategyID: "t1",
		TradeDate: "2026-08-18", Session: "ASIA", Lifecycle: "active", Doc: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	// Wednesday 00:30 CT — inside the SAME instance's tail. The chain date must
	// still resolve to 2026-08-18, so the dedupe sees the existing plan and NO
	// second plan is written.
	at.maybeRunSessionReadsAt(ctTime(t, 2026, 8, 19, 0, 30))

	var n int64
	st.GormDB().Model(&store.PlanDB{}).Where("trade_date = ? AND session = ?", "2026-08-18", "ASIA").Count(&n)
	if n != 1 {
		t.Fatalf("the midnight roll must NOT write a second plan for the same instance; count=%d", n)
	}
	if row, _ := st.Plan().GetLatestPlanForTraderSession("2026-08-19", "ASIA", "t1"); row != nil {
		t.Fatalf("no plan may be keyed to the next date mid-instance, got %+v", row)
	}
}
