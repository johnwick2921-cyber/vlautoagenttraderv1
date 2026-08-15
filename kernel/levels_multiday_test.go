package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

func barAt(loc *time.Location, y int, mo time.Month, d, h, mi int, o, hi, lo, c float64) market.Kline {
	open := time.Date(y, mo, d, h, mi, 0, 0, loc)
	return market.Kline{
		OpenTime:  open.UnixMilli(),
		Open:      o,
		High:      hi,
		Low:       lo,
		Close:     c,
		CloseTime: open.Add(time.Minute).UnixMilli() - 1,
	}
}

func levelMap(levels []DetectedLevel) map[LevelKind]float64 {
	m := map[LevelKind]float64{}
	for _, l := range levels {
		m[l.Kind] = l.Price
	}
	return m
}

func TestExtractMultiDayLevels(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	// now = Friday 2026-08-14 10:00 CT (mid-RTH). Futures session-day = 08-13.
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)

	bars := []market.Kline{
		// prior month (July) + prior week (Aug 3-9) singletons
		barAt(loc, 2026, 7, 20, 12, 0, 14750, 14800, 14700, 14750),
		barAt(loc, 2026, 8, 5, 12, 0, 15050, 15100, 15000, 15050),
		// two days ago (08-12)
		barAt(loc, 2026, 8, 12, 12, 0, 15250, 15300, 15200, 15250),
		// prior calendar day 08-13 — RTH (NY) then evening Asia
		barAt(loc, 2026, 8, 13, 9, 0, 15540, 15580, 15500, 15540),  // NY
		barAt(loc, 2026, 8, 13, 14, 0, 15500, 15560, 15450, 15550), // NY (RTH low 15450)
		barAt(loc, 2026, 8, 13, 18, 0, 15570, 15620, 15560, 15600), // Asia of futures-day 08-13 (also calendar 08-13)
		// current overnight (futures-day 08-13): more Asia (after midnight) + London
		barAt(loc, 2026, 8, 14, 0, 30, 15600, 15610, 15530, 15570), // Asia (calendar 08-14)
		barAt(loc, 2026, 8, 14, 4, 0, 15550, 15590, 15480, 15500),  // London
		// today's developing RTH (closed 09:00 bar) — not itself a level
		barAt(loc, 2026, 8, 14, 9, 0, 15600, 15650, 15550, 15600), // NY
	}

	m := levelMap(ExtractMultiDayLevels(bars, reg, now))

	want := map[LevelKind]float64{
		KindPDH:  15620, // 08-13 calendar high (incl. evening Asia)
		KindPDL:  15450, // 08-13 calendar low
		KindPDC:  15600, // 08-13 last close (18:00 bar)
		KindRTHH: 15580, // 08-13 NY high only
		KindRTHL: 15450, // 08-13 NY low
		KindASH:  15620, // overnight Asia high
		KindASL:  15530, // overnight Asia low
		KindLDNH: 15590, // overnight London high
		KindLDNL: 15480, // overnight London low
		KindONH:  15620, // composite
		KindONL:  15480,
		KindPWH:  15100,
		KindPWL:  15000,
		KindPMH:  14800,
		KindPML:  14700,
	}
	for k, v := range want {
		got, ok := m[k]
		if !ok {
			t.Fatalf("missing level %s", k)
		}
		if got != v {
			t.Fatalf("%s = %v want %v", k, got, v)
		}
	}
	if len(m) != len(want) {
		t.Fatalf("unexpected extra levels: got %d kinds, want %d (%v)", len(m), len(want), m)
	}
}

func TestExtractMultiDayLevelsColdStart(t *testing.T) {
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, chicago())
	// No bars → no levels (warms forward, honest empty).
	if got := ExtractMultiDayLevels(nil, reg, now); len(got) != 0 {
		t.Fatalf("cold start should emit no levels, got %v", got)
	}
}

func TestExtractMultiDayLevelsOvernightComposite(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 7, 0, 0, 0, loc) // during London, pre-RTH
	bars := []market.Kline{
		barAt(loc, 2026, 8, 13, 20, 0, 15500, 15560, 15490, 15520), // Asia
		barAt(loc, 2026, 8, 14, 3, 0, 15520, 15540, 15470, 15500),  // London
	}
	m := levelMap(ExtractMultiDayLevels(bars, reg, now))
	// ONH = max(Asia high, London high); ONL = min(lows).
	if m[KindONH] != 15560 || m[KindONL] != 15470 {
		t.Fatalf("ON composite = %v/%v want 15560/15470", m[KindONH], m[KindONL])
	}
	if m[KindASH] != 15560 || m[KindLDNL] != 15470 {
		t.Fatalf("AS/LDN extremes = %v/%v", m[KindASH], m[KindLDNL])
	}
}

func TestExtractMultiDayLevelsNoPriorDay(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)
	// Only TODAY's RTH bars — no prior calendar day, no overnight → no PDH, no ON.
	bars := []market.Kline{
		barAt(loc, 2026, 8, 14, 9, 0, 15600, 15650, 15550, 15600),
	}
	m := levelMap(ExtractMultiDayLevels(bars, reg, now))
	if _, ok := m[KindPDH]; ok {
		t.Fatalf("only today's bars → PDH must be absent")
	}
	if _, ok := m[KindONH]; ok {
		t.Fatalf("no overnight bars → ONH must be absent")
	}
}
