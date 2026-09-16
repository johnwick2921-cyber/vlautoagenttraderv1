package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// WAVE A / D1 — THE TOUCH RECORDER'S THREE CONTAMINATIONS, PINNED.
//
// The live table is 677 rows that are 423 episodes; RTH-L is 140 rows of ONE
// price (29199.25) that are 14 episodes, and all 14 opened BEFORE the level
// existed. 471 of 677 rows read "ordinal 1". These three pins are that defect
// stated as tests: pre-formation lookahead, duplicate re-recording, and an
// ordinal that resets on every read.
//
// All three are caller defects, not store defects: detector_record.go hands
// LastOpenedAtMs/NextOrdinal the CURRENT session-day, so any episode that
// opened before 17:00 CT today never matches the filter — the watermark reads
// 0 (re-record) and MAX(ordinal) reads 0 (ordinal 1), on every single read.

// recorderFixture builds an AutoTrader over a tape long enough to cross the
// CME session-day boundary, which is where all three defects live.
func recorderFixture(t *testing.T, level float64, bars int, endAt time.Time) (*AutoTrader, *store.Store, time.Time) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "wavea.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	now := time.Now()
	tape := oscillatingTape(level, endAt.Add(-time.Duration(bars)*time.Minute), bars)
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline { return tape }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	at := &AutoTrader{
		id: "hoang", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true},
		}},
	}
	return at, st, now
}

func seatedAt(level, formedAtMs float64, formed int64) []kernel.ScoredLevel {
	lv := kernel.ScoredLevel{Grade: "A", Score: 1.0}
	lv.Price = level
	lv.Kind = "RTH-L"
	lv.FormedAtMs = formed
	return []kernel.ScoredLevel{lv}
}

// E1 — PRE-FORMATION PIN. A level formed at T cannot have been touched before
// T. The recorder scans the whole ~33 h void scope regardless of formation, so
// today it records episodes that opened before the level existed. This is the
// RTH-L 14/14 defect.
func TestRecorderNeverRecordsAnEpisodeBeforeTheLevelFormed(t *testing.T) {
	const level = 29141.25
	at, st, now := recorderFixture(t, level, 2000, time.Now())

	// The level was born ONE HOUR AGO — every episode before that is lookahead.
	formed := now.Add(-60 * time.Minute).UnixMilli()

	at.recordDetectorOutputs("MNQ", "P1", "NY", 1,
		nil, seatedAt(level, 0, formed), level, 10, 2.0, 12, now, nil)

	rows, err := st.TouchOutcomes().AllOutcomes()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatalf("fixture produced no episodes at all — the tape does not exercise the recorder")
	}
	pre := 0
	for _, r := range rows {
		if r.OpenedAtMs < formed {
			pre++
		}
	}
	t.Logf("episodes=%d  formed_at=%d  pre-formation=%d", len(rows), formed, pre)
	if pre != 0 {
		t.Fatalf("E1 RED: %d of %d recorded episodes opened BEFORE the level formed — the scan does not start at FormedAtMs (this is the RTH-L 14/14 defect)", pre, len(rows))
	}
}

// E2 — DUPLICATE PIN. The same touch seen on three consecutive reads must be
// ONE row. Today the day-scoped watermark re-writes every episode that opened
// before the current session-day start, on every read: 677 rows / 423 episodes.
func TestRecorderWritesOneRowPerEpisodeAcrossRepeatedReads(t *testing.T) {
	const level = 29141.25
	// THE LIVE SITUATION: a static level whose every episode opened before
	// 17:00 CT today. The day-scoped watermark then reads 0 on every read,
	// because no recorded episode matches `opened_at_ms >= <today>` — so the
	// whole set is written again. This is how RTH-L became 140 rows of 14
	// episodes. A tape that reaches into the current session day HIDES the
	// defect: one fresh episode raises the watermark and silences the rest.
	boundary := kernel.CMESessionDayStart(time.Now())
	at, st, now := recorderFixture(t, level, 2000, boundary.Add(-10*time.Minute))
	seated := seatedAt(level, 0, 0) // formed long ago: formation is not what this pin tests

	at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, seated, level, 10, 2.0, 12, now, nil)
	first := st.TouchOutcomes().CountOutcomes()
	if first == 0 {
		t.Fatalf("fixture produced no episodes at all — the tape does not exercise the recorder")
	}
	// Two more identical reads. Nothing new happened on the tape.
	at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, seated, level, 10, 2.0, 12, now, nil)
	at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, seated, level, 10, 2.0, 12, now, nil)
	after := st.TouchOutcomes().CountOutcomes()

	t.Logf("read1=%d  after 3 identical reads=%d", first, after)
	if after != first {
		t.Fatalf("E2 RED: %d rows after three identical reads, expected %d — the episode is being re-recorded (this is 677 rows / 423 episodes)", after, first)
	}
}

// E3 — ORDINAL PIN. The ordinal must count a level's touches within the
// episode's OWN session-day and must never be reset by a read. Today every
// episode that opened before the current session-day start is written with
// ordinal 1 — which is why 471 of 677 live rows read 1.
func TestRecorderOrdinalIsNotResetByAReRead(t *testing.T) {
	const level = 29141.25
	boundary := kernel.CMESessionDayStart(time.Now())
	at, st, now := recorderFixture(t, level, 2000, boundary.Add(-10*time.Minute))
	seated := seatedAt(level, 0, 0)

	at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, seated, level, 10, 2.0, 12, now, nil)

	rows, err := st.TouchOutcomes().AllOutcomes()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Skipf("need >=2 episodes to test ordinals, got %d", len(rows))
	}
	// Group by the episode's own CME session-day; within a day the ordinals
	// must be distinct. A reset shows up as many rows sharing ordinal 1.
	perDay := map[int64]map[int]int{}
	for _, r := range rows {
		day := kernel.CMESessionDayStart(time.UnixMilli(r.OpenedAtMs)).UnixMilli()
		if perDay[day] == nil {
			perDay[day] = map[int]int{}
		}
		perDay[day][r.Ordinal]++
	}
	for day, ords := range perDay {
		for ord, n := range ords {
			if n > 1 {
				t.Fatalf("E3 RED: session-day %d has %d episodes all carrying ordinal %d — the ordinal is reset per read, not derived from the episode's own day (this is 471 of 677 rows reading ordinal 1)", day, n, ord)
			}
		}
	}
}

// E1b — THE BLINDSPOT, PINNED AS A FACT RATHER THAN A FOOTNOTE.
//
// FormedAtMs is set ONLY by kernel/levels_zones.go (DEMAND/SUPPLY/OB/FVG).
// Every LINE level is built by lineLevel (kernel/levels.go:93-95), which never
// sets it — 503 of the 677 live rows (74.3%), including all 140 RTH-L rows
// that are the premise's own evidence. So the formation floor CANNOT gate the
// kinds the premise complains about.
//
// This pin exists so that fact is enforced, not remembered: such episodes are
// recorded, marked unverified:no_formation, and excluded from every rate. If
// someone later makes line levels carry a birth time, this test tells them to
// come back and delete it.
func TestLineLevelsCannotBeCertifiedAndAreExcludedFromRates(t *testing.T) {
	const level = 29141.25
	boundary := kernel.CMESessionDayStart(time.Now())
	at, st, now := recorderFixture(t, level, 2000, boundary.Add(-10*time.Minute))

	// A LINE level: exactly what lineLevel produces — no formation time.
	seated := seatedAt(level, 0, 0)
	at.recordDetectorOutputs("MNQ", "P1", "NY", 1, nil, seated, level, 10, 2.0, 12, now, nil)

	rows, err := st.TouchOutcomes().AllOutcomes()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("fixture produced no episodes")
	}
	for _, r := range rows {
		if r.Validity != store.ValidityNoFormation {
			t.Fatalf("a level with no formation time must be recorded as %q, got %q — an uncertifiable episode must never be dressed as certified (A24)",
				store.ValidityNoFormation, r.Validity)
		}
	}
	// And they must not reach a rate.
	rates, err := st.TouchOutcomes().RatesBy("")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range rates {
		n += r.N()
	}
	if n != 0 {
		t.Fatalf("uncertified episodes reached a rate: n=%d — the validity chokepoint in RatesBy is not holding", n)
	}
	t.Logf("%d episodes recorded, all %q, and 0 of them reached a rate — the blindspot is explicit",
		len(rows), store.ValidityNoFormation)
}
