package trader

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// oscillatingTape crosses `level` repeatedly so D1′ produces real episodes.
func oscillatingTape(level float64, start time.Time, n int) []market.Kline {
	bars := make([]market.Kline, 0, n)
	p := level
	for i := 0; i < n; i++ {
		// deterministic zig-zag through the level, wide enough to breach ±k·Δ
		phase := float64(i % 40)
		p = level + (phase-20)*1.4
		o := p
		c := level + (float64((i+1)%40)-20)*1.4
		h := math.Max(o, c) + 1.5
		l := math.Min(o, c) - 1.5
		ts := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: ts.UnixMilli(), Open: o, High: h, Low: l, Close: c,
			CloseTime: ts.Add(time.Minute).UnixMilli()})
	}
	return bars
}

// THE PROOF THE OWNER ASKED FOR: a touch_outcomes row written on a fixture
// THROUGH THE PRODUCTION CALL PATH — not by calling the store directly.
func TestDetectorWritesThroughTheProductionPath(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "det.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Date(2026, 9, 3, 20, 0, 0, 0, kernel.CTLocation())
	const level = 29141.25
	bars := oscillatingTape(level, kernel.CMESessionDayStart(now).Add(time.Hour), 600)

	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline { return bars }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	at := &AutoTrader{id: "hoang", store: st}
	// WAVE A / D1e — a rate is drawn from CERTIFIED rows only, and a level is
	// certifiable only if it carries a formation time. This fixture gives its
	// level one (born with the tape) so the rate assertion below still has a
	// population; an uncertified line level would correctly yield episodes and
	// no rate, which TestLineLevelsCannotBeCertifiedAndAreExcludedFromRates pins.
	seated := []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{
		Price: level, Label: "ONL", FormedAtMs: bars[0].OpenTime,
	}, Score: 91, Grade: "A"}}
	all := []kernel.DetectedLevel{
		{Price: level, Label: "ONL"},
		{Price: level + 500, Label: "FAR"}, // cut on proximity
	}

	at.recordDetectorOutputs("MNQ", "2026-09-03:ASIA:hoang", "ASIA", 3, all, seated,
		level, 300, 1.5, 12, now, nil)

	ts := st.TouchOutcomes()
	if n := ts.CountOutcomes(); n == 0 {
		t.Fatal("the production path wrote NO touch_outcomes row — the detector is still not recording")
	}
	rows, _ := ts.RatesBy("")
	r := rows[0]
	t.Logf("FIRST ROWS THROUGH THE PRODUCTION PATH: %d episode(s) · hold=%d break=%d ambiguous=%d",
		ts.CountOutcomes(), r.Hold, r.Break, r.Ambiguous)

	// The row must carry the scope it was judged under — a verdict without its
	// instrument settings cannot be re-checked later.
	var got store.TouchOutcomeRow
	if err := st.GormDB().Order("id ASC").First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.K != kernel.DetectorK() || got.Horizon != kernel.DetectorHorizonBars() || got.ExitOn != kernel.DetectorExitOn() {
		t.Errorf("row lost its scope: k=%.1f H=%d exit=%s", got.K, got.Horizon, got.ExitOn)
	}
	if got.Delta <= 0 || got.BandPts <= 0 {
		t.Errorf("Δ and band must be recorded, got Δ=%.4f band=%.4f", got.Delta, got.BandPts)
	}
	if got.Ordinal < 1 {
		t.Errorf("ordinal must come from the store, got %d", got.Ordinal)
	}
	if !got.CandidateSeated {
		t.Error("a seated level's episode must be marked seated")
	}

	// IDEMPOTENT: a second read writes nothing new — the watermark is the store.
	before := ts.CountOutcomes()
	at.recordDetectorOutputs("MNQ", "2026-09-03:ASIA:hoang", "ASIA", 3, all, seated,
		level, 300, 1.5, 12, now, nil)
	if after := ts.CountOutcomes(); after != before {
		t.Errorf("a repeated read must write no new episodes: %d → %d", before, after)
	}

	// D3: the pool carries BOTH candidates, and the cut one carries its reason.
	// NOTE: the pool is recorded PER READ by design (D3: "at every planner
	// read"), so after the idempotence re-run above there are two pools of two.
	// Episodes de-duplicate on a watermark; pools deliberately do not, because
	// the question they answer is "what did the constructor produce THAT read".
	pool, _ := st.CandidatePool().LatestPool(50)
	if len(pool) != 4 {
		t.Fatalf("two reads × two candidates = 4 pool rows, got %d", len(pool))
	}
	var seatedN, cutN int
	for _, p := range pool {
		if p.Seated {
			seatedN++
			continue
		}
		cutN++
		if p.CutReason == "" {
			t.Error("a cut candidate with no reason defeats the point of the table")
		}
	}
	if seatedN != 2 || cutN != 2 {
		t.Errorf("want 2 seated / 2 cut across the two reads, got %d/%d", seatedN, cutN)
	}
	var aCutReason string
	for _, p := range pool {
		if !p.Seated {
			aCutReason = p.CutReason
			break
		}
	}
	t.Logf("pool: %d seated, %d cut · a cut reason: %q", seatedN, cutN, aCutReason)
}

// E7 — a telemetry failure WARNs and the loop continues (class 23).
func TestDetectorRecordingNeverStopsTheLoop(t *testing.T) {
	at := &AutoTrader{id: "hoang", store: nil} // nil store: the worst case
	done := make(chan struct{})
	go func() {
		defer close(done)
		at.recordDetectorOutputs("MNQ", "p", "ASIA", 1, nil, nil, 29000, 300, 1.5, 12, time.Now(), nil)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("recording blocked the caller — telemetry must never stall the loop")
	}
}
