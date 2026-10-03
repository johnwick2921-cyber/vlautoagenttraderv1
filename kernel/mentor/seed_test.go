package mentor

import (
	"strings"
	"testing"
	"time"

	"vl/market"
)

// ct returns CT epoch millis for the given day offset and hour:minute.
func ctMs(t *testing.T, dayOffset, hour, minute int) int64 {
	t.Helper()
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, ctime())
	ms := base.AddDate(0, 0, dayOffset).Add(time.Duration(hour)*time.Hour +
		time.Duration(minute)*time.Minute).UnixMilli()
	return ms
}

func mkBar(open, close float64, openMs int64, tfMin int) market.Kline {
	return market.Kline{
		Open:      open,
		Close:     close,
		High:      max(open, close),
		Low:       min(open, close),
		OpenTime:  openMs,
		CloseTime: openMs + int64(tfMin)*60_000 - 1,
	}
}

// TestSeedMissingColdStart: an empty store names every source, fail-closed.
func TestSeedMissingColdStart(t *testing.T) {
	now := ctMs(t, 0, 10, 0)
	got := SeedMissing(nil, nil, now)
	if len(got) != 5 {
		t.Fatalf("want 5 missing sources, got %d: %v", len(got), got)
	}
	for _, s := range got {
		if !strings.HasPrefix(s, "bar history depth: ") {
			t.Fatalf("source %q does not carry the depth prefix", s)
		}
	}

	e := New(DefaultConfig())
	if m := Seed(e, nil, nil, now); len(m) != 5 {
		t.Fatalf("Seed returned %d missing, want 5", len(m))
	}
	if len(e.SourcesMissing()) != 5 {
		t.Fatalf("SourcesMissing() = %v, want 5 entries", e.SourcesMissing())
	}
}

// TestSeedKeyLevelsGolden: SeedLevels == a full key-level walk over the same
// stored 1h candles (the rebuild golden).
func TestSeedKeyLevelsGolden(t *testing.T) {
	var bars1h []market.Kline
	// 8 days × 7 RTH candles (08:00..14:00 CT), alternating colours each candle.
	for d := 0; d < 8; d++ {
		for h := 8; h < 15; h++ {
			open := 1000.0 + float64(d*10+h)
			cl := open + 2
			if (d+h)%2 == 1 {
				cl = open - 2
			}
			bars1h = append(bars1h, mkBar(open, cl, ctMs(t, d, h, 0), 60))
		}
	}
	now := ctMs(t, 8, 9, 0)

	e := New(DefaultConfig())
	Seed(e, nil, bars1h, now)

	want := keyLevelsFromCandles(keyLevel1HBars(bars1h), e.Cfg.KeyLevelPrunePts)
	if len(e.State.SeedLevels) != len(want) {
		t.Fatalf("SeedLevels len %d, rebuild %d", len(e.State.SeedLevels), len(want))
	}
	for i := range want {
		if e.State.SeedLevels[i].Price != want[i].Price ||
			e.State.SeedLevels[i].Kind != want[i].Kind {
			t.Fatalf("level %d: seed %+v, rebuild %+v", i, e.State.SeedLevels[i], want[i])
		}
	}
}

// TestSeedIncrementalEMA: seed over the first half of the tape, then advance —
// the incremental EMA must equal a full rebuild over the whole tape.
func TestSeedIncrementalEMA(t *testing.T) {
	cfg := DefaultConfig()
	var bars []market.Kline
	cl := 100.0
	for i := 0; i < 100; i++ {
		cl += 0.5
		if i%3 == 1 {
			cl -= 1.0
		}
		bars = append(bars, mkBar(cl-0.25, cl, ctMs(t, 0, 8, 30)+int64(i)*60_000, 1))
	}
	nowMid := bars[49].CloseTime
	e := New(cfg)
	Seed(e, bars[:50], nil, nowMid)
	if e.State.EMA34 == 0 || e.State.EMA9 == 0 {
		t.Fatal("seed did not set the 1m EMAs")
	}

	// The full tape closes bar by bar — the incremental path must track.
	for i := 50; i < 100; i++ {
		e.seededLevels(bars[:i+1], bars[i].CloseTime)
	}
	want34 := emaValue(bars, cfg.EMAPeriod34)
	want9 := emaValue(bars, cfg.EMAPeriod9)
	if abs(e.State.EMA34-want34) > 1e-9 {
		t.Fatalf("EMA34 incremental %.6f, rebuild %.6f", e.State.EMA34, want34)
	}
	if abs(e.State.EMA9-want9) > 1e-9 {
		t.Fatalf("EMA9 incremental %.6f, rebuild %.6f", e.State.EMA9, want9)
	}
}

// TestSeedFailClosed: with a missing source the sweep drops entries, keeps
// cancels; without one nothing is dropped.
func TestSeedFailClosed(t *testing.T) {
	ins := []Intent{
		{Action: PlaceStopEntry, Side: SideLong, Price: 100},
		{Action: PlaceStopLimitEntry, Side: SideShort, Price: 101},
		{Action: CancelArm, ArmID: "a"},
		{Action: ExtendArm, ArmID: "a", ExpiryMs: 123},
	}
	got := failClosedFilter(ins)
	if len(got) != 2 || got[0].Action != CancelArm || got[1].Action != ExtendArm {
		t.Fatalf("failClosedFilter kept %v", got)
	}
}

// TestSeedSwingLineWarm: 9 days of 1h closes → the swing line is seeded as the
// EMA recurrence over all closed 4h candles, bucket start = the current one.
func TestSeedSwingLineWarm(t *testing.T) {
	cfg := DefaultConfig()
	var bars1h []market.Kline
	cl := 20000.0
	day, hour := 0, 0
	for i := 0; i < 19*24; i++ {
		cl += 2
		if i%5 == 0 {
			cl -= 3
		}
		bars1h = append(bars1h, mkBar(cl-0.5, cl, ctMs(t, day, hour, 0), 60))
		hour++
		if hour == 24 {
			hour, day = 0, day+1
		}
	}
	now := ctMs(t, 18, 20, 0) // day 18, 20:00 CT — inside the 17:00–21:00 bucket
	closed := fourHClosedBuckets(bars1h, now)
	if len(closed) < FourHEMA34Min {
		t.Fatalf("fixture too short: %d closed 4h candles", len(closed))
	}

	// today's 1m tape: 110 closed bars at 17:15 CT onward — inside the
	// truncated-epoch session day that contains 20:00 CT (dayStartCT truncates
	// on the UTC boundary, not CT midnight), and past the 102 warm-up.
	var bars1m []market.Kline
	for i := 0; i < 110; i++ {
		c := 20500 + float64(i)
		bars1m = append(bars1m, mkBar(c-0.25, c, ctMs(t, 18, 17, 15)+int64(i)*60_000, 1))
	}

	e := New(cfg)
	if m := Seed(e, bars1m, bars1h, now); len(m) != 0 {
		t.Fatalf("unexpected missing: %v", m)
	}
	var closes []float64
	for _, b := range closed {
		closes = append(closes, b.Close)
	}
	want := emaFromCloses(closes, cfg.Swing.EMAPeriod)
	if abs(e.State.Swing.Line-want) > 1e-9 {
		t.Fatalf("seeded swing line %.6f, want %.6f", e.State.Swing.Line, want)
	}
	if e.State.Swing.BucketStart != fourHBucketStart(now, ctime()) {
		t.Fatalf("bucket start %d, want current bucket %d",
			e.State.Swing.BucketStart, fourHBucketStart(now, ctime()))
	}
	if e.State.Swing.EmaCount != len(closed) {
		t.Fatalf("EmaCount %d, want %d", e.State.Swing.EmaCount, len(closed))
	}
}

// TestSeedSwingFlipIncremental: a 4h flip advances the warm line by the SAME
// EMA recurrence (not a cold recompute) — parity with a full rebuild.
func TestSeedSwingFlipIncremental(t *testing.T) {
	cfg := DefaultConfig()
	var bars1h []market.Kline
	cl := 20000.0
	day, hour := 0, 0
	for i := 0; i < 9*24; i++ {
		cl += 2
		bars1h = append(bars1h, mkBar(cl-0.5, cl, ctMs(t, day, hour, 0), 60))
		hour++
		if hour == 24 {
			hour, day = 0, day+1
		}
	}
	now := ctMs(t, 8, 20, 0)
	e := New(cfg)
	Seed(e, nil, bars1h, now)
	line0 := e.State.Swing.Line
	count0 := e.State.Swing.EmaCount

	// New closed 4h bucket: day 8 21:00–day 9 01:00 (the 17:00 anchor).
	// 5m bars inside the OLD bucket (so bucketClose finds the close) + one
	// inside the new bucket to force the flip.
	oldStart := e.State.Swing.BucketStart
	oldClose := 0.0
	var bars5m []market.Kline
	for m := 0; m < 48; m++ {
		t5 := oldStart + int64(m)*5*60_000
		c := 21000 + float64(m)
		if m == 47 {
			oldClose = c
		}
		bars5m = append(bars5m, mkBar(c-0.25, c, t5, 5))
	}
	newStart := oldStart + 240*60_000
	// fill the NEW bucket too, then close it: the flip fires when the new
	// bucket is CLOSED (swingLine needs ≥2 closed buckets to return a line).
	for m := 0; m < 48; m++ {
		t5 := newStart + int64(m)*5*60_000
		c := 21100 + float64(m)
		bars5m = append(bars5m, mkBar(c-0.25, c, t5, 5))
	}
	now2 := newStart + 240*60_000 // the new bucket has closed

	out := SwingTick(&e.State.Swing, bars5m, cfg.Swing, now2)
	_ = out
	k := 2.0 / float64(cfg.Swing.EMAPeriod+1)
	want := line0 + k*(oldClose-line0)
	if abs(e.State.Swing.Line-want) > 1e-9 {
		t.Fatalf("flip line %.6f, want incremental %.6f", e.State.Swing.Line, want)
	}
	if e.State.Swing.EmaCount != count0+1 {
		t.Fatalf("EmaCount %d, want %d", e.State.Swing.EmaCount, count0+1)
	}
	if e.State.Swing.BucketStart != newStart+240*60_000 {
		t.Fatalf("BucketStart %d, want current bucket %d",
			e.State.Swing.BucketStart, newStart+240*60_000)
	}
}

// TestSeedLineNamesDepths: the one-liner reports per-source depths.
func TestSeedLineNamesDepths(t *testing.T) {
	e := New(DefaultConfig())
	Seed(e, nil, nil, ctMs(t, 0, 9, 0))
	l := e.seedLine
	for _, want := range []string{"4h EMA34 0/102", "1m EMA34 0/102", "1H RTH levels 0 candles", "levels 0"} {
		if !strings.Contains(l, want) {
			t.Fatalf("seed line %q missing %q", l, want)
		}
	}
}

// TestSeedWarmupGate: between the absolute minimum (34) and the stated warm-up
// (102) the source is still REFUSED and named with the warm-up denominator —
// a barely-formed EMA must never trade (CTO 07:00 ruling).
func TestSeedWarmupGate(t *testing.T) {
	short := hourTape(t, 9) // 55 closed 4h candles: above 34, below 102
	long := hourTape(t, 19) // 114 closed 4h candles: warm

	now := ctMs(t, 18, 20, 0)
	var bars1m []market.Kline
	for i := 0; i < 110; i++ {
		c := 20500 + float64(i)
		bars1m = append(bars1m, mkBar(c-0.25, c, ctMs(t, 18, 17, 15)+int64(i)*60_000, 1))
	}

	miss := SeedMissing(bars1m, short, now)
	want := "bar history depth: 4h EMA34 warm-up (55/102 4h candles)"
	found := false
	for _, s := range miss {
		if s == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("55 4h candles must be refused as %q, got %v", want, miss)
	}

	if m := SeedMissing(bars1m, long, now); len(m) != 0 {
		t.Fatalf("114 4h candles must be warm, got %v", m)
	}
}

func hourTape(t *testing.T, days int) []market.Kline {
	t.Helper()
	var bars []market.Kline
	cl := 20000.0
	day, hour := 0, 0
	for i := 0; i < days*24; i++ {
		cl += 2
		if i%5 == 0 {
			cl -= 3
		}
		bars = append(bars, mkBar(cl-0.5, cl, ctMs(t, day, hour, 0), 60))
		hour++
		if hour == 24 {
			hour, day = 0, day+1
		}
	}
	return bars
}

// TestSeedFailClosedTick: the sweep runs INSIDE Tick — the same ISB tape that
// emits an entry when unseeded must emit NO entry when seeded with a missing
// source (cancels still flow). The mutant that disables the sweep turns this
// RED: the placement comes back.
func TestSeedFailClosedTick(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	head := market.Kline{Open: 121, High: 121.5, Low: 120.5, Close: 120, CloseTime: 1}
	mother := market.Kline{Open: 98, High: 106, Low: 97, Close: 105, CloseTime: 60_000 - 1}
	c1 := market.Kline{Open: 103, High: 104, Low: 99, Close: 100, CloseTime: 119_999}
	c2 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 179_999}
	c3 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 239_999}
	c4 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 299_999}
	bars := []market.Kline{head, mother, c1, c2, c3, c4}

	count := func(e *Evaluator) (placements, cancels int) {
		for i := 2; i <= len(bars); i++ {
			now := bars[i-1].CloseTime + 1
			for _, in := range e.Tick(bars[:i], now) {
				switch in.Action {
				case PlaceStopLimitEntry, PlaceStopEntry:
					placements++
				case CancelArm:
					cancels++
				}
			}
		}
		return
	}

	setup := func() *Evaluator {
		e := New(cfg)
		e.State.Trigger = TriggerLine{Dir: SideLong, Price: 90}
		e.State.HTF = HTF{FourH: TriggerLine{Dir: SideLong, Price: 90}}
		e.State.ORB = ORB{Day: 0, High: 90, Low: 85, Drawn: true, Escaped: SideLong}
		return e
	}

	// Control: unseeded, the tape emits the arm (legacy behaviour).
	ctrl := setup()
	p, c := count(ctrl)
	if p != 1 {
		t.Fatalf("control placements = %d, want 1 (tape must emit when unseeded)", p)
	}

	// Seeded with a missing source: the same tape must refuse entries, but
	// the ISB stacking cancel still flows.
	seeded := setup()
	Seed(seeded, nil, nil, bars[len(bars)-1].CloseTime)
	if len(seeded.SourcesMissing()) == 0 {
		t.Fatal("nil bars must leave every source missing")
	}
	p, c = count(seeded)
	if p != 0 {
		t.Fatalf("seeded-missing placements = %d, want 0 (fail-closed)", p)
	}
	if c == 0 {
		t.Fatalf("seeded-missing cancels = 0, want the stacking cancel to flow")
	}
}
