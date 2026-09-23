package trader

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// pictureHtfTestEnv drives the evaluator through the PRODUCTION call site
// (Evaluate/OnBars) with a seeded FuturesBarsProvider — the same ladder the
// live path reads. The submit seam records calls instead of sending wire.
type pictureHtfTestEnv struct {
	t       *testing.T
	at      *AutoTrader
	st      *store.Store
	eval    *PictureHtfEvaluator
	submits []string
	now     time.Time
}

func newPictureHtfEnv(t *testing.T, cfg store.PictureHtfConfig) *pictureHtfTestEnv {
	t.Helper()
	// W-EXEC-TRUTH W0: Picture is admitted only while its trader RUNS and the
	// Day Plan master is ON (admitEntry — it used to run regardless, D26), so
	// the harness is a running trader with the master on.
	sc := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &cfg, PlanEnabled: true}}
	sc.RiskControl.MinRiskRewardRatio = 2.5
	at, st := resetTrader(t, sc)
	at.isRunningMutex.Lock()
	at.isRunning = true
	at.isRunningMutex.Unlock()
	eval := NewPictureHtfEvaluator(at, store.PictureHtfResolved(&cfg))
	env := &pictureHtfTestEnv{t: t, at: at, st: st, eval: eval}
	orig := pictureHtfSubmitSeam
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env.submits = append(env.submits, row.OppKey)
		return nil
	}
	origCap := pictureHtfCapabilityProven
	pictureHtfCapabilityProven = func(*AutoTrader) bool { return true }
	t.Cleanup(func() {
		pictureHtfSubmitSeam = orig
		pictureHtfCapabilityProven = origCap
		market.FuturesBarsProvider = nil
	})
	return env
}

// t4h0 is the 4H ladder base (2026-09-13 01:00Z). The chronology:
//
//	4H ladder i=0..8 → pivot 101 knowable at 09-13 17:00Z, pivot 110 knowable
//	at 09-14 09:00Z.
//	H1 ladder i=0..41 → last pair closes 17:00Z (prev 101.0) / 18:00Z
//	(cur 101.5) — after BOTH pivots are knowable.
//	5m ladder ends 18:59:59.999Z → the next 5m interval opens 19:00Z;
//	now = 19:00:00.5Z (14:00:00.5 CT) — inside the 10s entry window.
var t4h0 = time.Date(2026, 9, 13, 1, 0, 0, 0, time.UTC).UnixMilli()

func mkBar(open, span int64, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: open, CloseTime: open + span - 1, Open: o, High: h, Low: l, Close: c, Final: true}
}

func tailOf(bars []market.Kline, n int) []market.Kline {
	if n <= 0 || len(bars) <= n {
		return bars
	}
	return bars[len(bars)-n:]
}

// pictureBars4H builds the full ladder: an OLD 110 resistance (i=1), a
// pullback, and the RECENT 101 resistance (i=7) that the H1 pair later
// breaks. No later 4H close exceeds 101, so the 101 level is still ACTIVE
// (not retired) when the H1 crosses it — and the 110 zone above is the
// opposing target. This mirrors reality: old high → consolidation →
// breakout → target the old high.
func pictureBars4H() []market.Kline {
	vals := [][4]float64{
		{108, 110, 106, 109}, {109, 111, 107, 110}, {107, 108, 105, 106}, {105, 106, 103, 104},
		{103, 104, 101, 102}, {100, 101, 99, 100.5}, {98.5, 100, 97.5, 99}, {99, 105, 96, 101},
		{96, 97, 94, 95}, {94, 95, 92, 93}, {93, 94, 91, 92},
	}
	out := make([]market.Kline, 0, len(vals))
	for i, v := range vals {
		out = append(out, mkBar(t4h0+int64(i)*4*3600*1000, 4*3600*1000, v[0], v[1], v[2], v[3]))
	}
	return out
}

// pictureBars4HNoTarget: the same recent 101 resistance but NO old high — the
// opposing-zone refusal fixture.
func pictureBars4HNoTarget() []market.Kline {
	vals := [][4]float64{
		{98, 99, 97, 98}, {99, 100, 98, 99.5}, {99, 105, 96, 101},
		{96, 97, 94, 95}, {94, 95, 92, 93}, {93, 94, 91, 92},
	}
	out := make([]market.Kline, 0, len(vals))
	for i, v := range vals {
		out = append(out, mkBar(t4h0+int64(i)*4*3600*1000, 4*3600*1000, v[0], v[1], v[2], v[3]))
	}
	return out
}

func pictureBarsH1() []market.Kline {
	out := make([]market.Kline, 0, 42)
	for i := 0; i < 42; i++ {
		c := 99.0 + float64(i)*0.0625
		out = append(out, mkBar(t4h0+int64(i)*3600*1000, 3600*1000, c-0.2, c+0.4, c-0.4, c))
	}
	out[40] = mkBar(t4h0+40*3600*1000, 3600*1000, 100.8, 101.3, 100.5, 101.0) // prev
	out[41] = mkBar(t4h0+41*3600*1000, 3600*1000, 101.0, 102.0, 100.6, 101.5) // cur
	return out
}

func pictureBars5M(withSwing bool) []market.Kline {
	// 28 bars, last close 18:59:59.999Z, closes rise to ~101.5.
	base := t4h0 + 39*3600*1000 + 40*60*1000 // 16:40Z
	out := make([]market.Kline, 0, 28)
	for i := 0; i < 28; i++ {
		c := 99.3 + float64(i)*0.08
		lo := c - 0.6
		if withSwing && i == 22 {
			lo = 98.5 // the strict swing low (neighbors higher)
		}
		if i >= 23 {
			lo = c - 0.4
		}
		out = append(out, mkBar(base+int64(i)*5*60*1000, 5*60*1000, c, c+0.6, lo, c+0.03))
	}
	return out
}

// seedPictureTape installs the full ladder (4H with both pivots, H1 pair, 5m
// with the swing) and sets now = 0.5s into the next 5m interval.
func (env *pictureHtfTestEnv) seedPictureTape() {
	env.t.Helper()
	env.seed(pictureBars4H(), pictureBarsH1(), pictureBars5M(true))
	env.now = time.UnixMilli(t4h0 + 42*3600*1000 + 500) // 19:00:00.5Z
}

func (env *pictureHtfTestEnv) seed(bars4h, barsH1, bars5m []market.Kline) {
	env.t.Helper()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		switch tf {
		case "4h":
			return tailOf(bars4h, count)
		case "1h":
			return tailOf(barsH1, count)
		case "5m":
			return tailOf(bars5m, count)
		}
		return nil
	}
}

func (env *pictureHtfTestEnv) fresh5m() {
	env.t.Helper()
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now.Add(-500*time.Millisecond))
}

func TestPictureHtfEvaluatorSubmitsOnceAndAdmits(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// The production call site is the 5m bar EVENT: OnBars → evaluate → submit.
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now)
	if len(env.submits) != 1 {
		t.Fatalf("exactly one submission from the 5m event, got %d", len(env.submits))
	}
	// A duplicate evaluation (tick fallback, restart): the durable claim
	// blocks a second submission and reports it.
	res2 := env.eval.Evaluate("MNQ", env.now)
	if res2.Stage != "watching" || !strings.Contains(res2.Reason, "already claimed") {
		t.Fatalf("duplicate evaluation must report the existing claim, got %+v", res2)
	}
	if len(env.submits) != 1 {
		t.Fatalf("duplicate evaluation must not re-submit")
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok || row.Stage != "place_pending" {
		t.Fatalf("the opportunity must sit place_pending awaiting broker evidence: %+v", row)
	}
	if row.StopPx <= 0 || row.TargetPx <= 0 || row.RREstimate < row.RRConfigured {
		t.Fatalf("geometry must be persisted: %+v", row)
	}
	if row.Direction != "long" || row.LevelRole != "resistance" || row.LevelBodyTop != 101 {
		t.Fatalf("the broken level must be the 101 resistance, got %+v", row)
	}
	if row.TargetPx != 110 {
		t.Fatalf("the opposing target must be the old 110 high, got %+v", row)
	}
}

func TestPictureHtfEvaluatorLateFrameExpires(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// The freshest 5m frame is 5s old — outside the 2s freshness limit.
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now.Add(-5*time.Second))
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "expired" {
		t.Fatalf("a late frame must expire, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a late frame must never submit")
	}
}

func TestPictureHtfEvaluatorPastWindowExpires(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// 30s into the interval — outside the 10s entry window.
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now)
	late := env.now.Add(30 * time.Second)
	res := env.eval.Evaluate("MNQ", late)
	if res.Stage != "expired" {
		t.Fatalf("past the entry window must expire, got %+v", res)
	}
}

func TestPictureHtfEvaluatorBeforeBoundaryWatches(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	early := env.now.Add(-1 * time.Second)
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), early)
	if res := env.eval.Evaluate("MNQ", early); res.Stage != "watching" {
		t.Fatalf("before the boundary the mode watches, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("no submission before the window")
	}
}

func TestPictureHtfEvaluatorNoSwingRefuses(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seed(pictureBars4H(), pictureBarsH1(), pictureBars5M(false))
	env.now = time.UnixMilli(t4h0 + 42*3600*1000 + 500)
	env.fresh5m()
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "refused" || res.Reason == "" {
		t.Fatalf("no swing must refuse with a reason, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a refused opportunity must never submit")
	}
}

func TestPictureHtfEvaluatorNoOpposingZoneRefuses(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	// Only the 101 pivot — no old high above the entry.
	env.seed(pictureBars4HNoTarget(), pictureBarsH1(), pictureBars5M(true))
	env.now = time.UnixMilli(t4h0 + 42*3600*1000 + 500)
	env.fresh5m()
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "refused" {
		t.Fatalf("no opposing zone must refuse, got %+v", res)
	}
}

func TestPictureHtfEvaluatorLowRRRefuses(t *testing.T) {
	// The qualified tape offers ~2.6R; demanding 5R must refuse with the named
	// reason and NEVER skip the nearer zone to fix the number.
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 5.0})
	env.seedPictureTape()
	env.fresh5m()
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "refused" {
		t.Fatalf("R:R below the configured minimum must refuse, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("a refused opportunity must never submit")
	}
}

func TestPictureHtfEvaluatorDisabledDoesNothing(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: false})
	env.seedPictureTape()
	if res := env.eval.Evaluate("MNQ", env.now); res.Stage != "watching" {
		t.Fatalf("disabled mode must watch, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("disabled mode must never submit")
	}
}

func TestPictureHtfEvaluatorCapabilityGateBlocks(t *testing.T) {
	// The DEFAULT capability seam reads the concrete trader — resetTrader has
	// none, so capability is NOT proven and the mode must stay unavailable.
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	ev := NewPictureHtfEvaluator(at, store.PictureHtfResolved(&store.PictureHtfConfig{Enabled: true}))
	res := ev.Evaluate("MNQ", time.Now())
	if res.Stage != "watching" || !strings.Contains(res.Reason, "mode unavailable") {
		t.Fatalf("an unproven AddOn must gate the mode off, got %+v", res)
	}
}

func TestPictureHtfEvaluatorIgnoresUnfinalizedBars(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	// Override the 5m ladder: the NEWEST bar is time-complete (CloseTime < now)
	// but the AddOn never finalized it — it must not count as a completed bar,
	// so the previous interval's window is past and the evaluation expires.
	bars5m := pictureBars5M(true)
	bars5m[len(bars5m)-1].Final = false
	env.seed(pictureBars4H(), pictureBarsH1(), bars5m)
	env.eval.OnBars("MNQ", "5m", tailOf(bars5m, 1), env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "expired" {
		t.Fatalf("an unfinalized bar must not establish the current interval, got %+v", res)
	}
	if len(env.submits) != 0 {
		t.Fatalf("an unfinalized newest bar must never submit")
	}
}

// W-PICTURE-HTF (2026-09-20) — the AGREED sequence, proven at the production
// call site with native-grid candles: H1 close → the 5m bar completing AT that
// close → intervalStart = that bar's open + 5m (the production formula) →
// evaluation at the interval's first instant, using ONLY prices knowable then.
// The fixture adds a FORMING 19:00 5m bar with a decoy close/high/low — a
// price knowable only AFTER the entry instant — and asserts it never leaks
// into the interval computation, the entry reference, or the stop.
func TestPictureHtfH1CloseToNext5mSequenceNativeAlignment(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape() // qualifies at now = 19:00:00.5Z, H1 cur closes 18:59:59.999Z

	bars5m := pictureBars5M(true) // last completed bar: open 18:55, close 18:59:59.999 @ 101.49
	h1 := pictureBarsH1()
	if bars5m[len(bars5m)-1].CloseTime != h1[41].CloseTime {
		t.Fatalf("fixture grid misaligned: 5m close %d != H1 close %d", bars5m[len(bars5m)-1].CloseTime, h1[41].CloseTime)
	}
	forming := mkBar(t4h0+42*3600*1000, 5*60*1000, 101.5, 150.0, 101.0, 150.0) // decoy, Final=false
	env.seed(pictureBars4H(), h1, append(bars5m, forming))

	// The entry trigger is the first 5m frame of the next interval.
	env.eval.OnBars("MNQ", "5m", tailOf(append(bars5m, forming), 1), env.now)
	if len(env.submits) != 1 {
		t.Fatalf("the qualifying sequence must submit once, got %d", len(env.submits))
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok {
		t.Fatalf("admitted row missing")
	}
	// The production interval formula: previous completed 5m bar's open + 5m.
	wantInterval := bars5m[len(bars5m)-1].OpenTime + 5*60_000
	if row.WindowOpen != wantInterval || row.WindowOpen != t4h0+42*3600*1000 {
		t.Fatalf("interval must be the completed bar's open + 5m (19:00:00.000), got %d", row.WindowOpen)
	}
	// EntryRef is the close of the bar COMPLETED at the H1 close — knowable at
	// the entry instant. The forming bar's decoy close (150) must not leak in.
	if row.EntryRef != 101.49 {
		t.Fatalf("entry ref must be the completed bar's close 101.49 (knowable at entry), got %.2f", row.EntryRef)
	}
	if row.H1CloseTime != t4h0+42*3600*1000-1 {
		t.Fatalf("H1 close must be 18:59:59.999, got %d", row.H1CloseTime)
	}
	// The stop comes from the confirmed swing low (98.5 → 98.25 after the
	// one-tick buffer); the forming bar's decoy low (101.0) must not distort it.
	if row.StopPx != 98.25 {
		t.Fatalf("stop must be the swing low − tick (98.25), got %.2f", row.StopPx)
	}
}

// ── Short-direction mirrors ──────────────────────────────────────────────

// pictureBars4HShort: an OLD support at 90 (i=0), a rally, and the RECENT
// support at 94.5 (i=3) the H1 pair later breaks down through. No later 4H
// close trades below 94.5, so the level is still active at the break; the 90
// support is the opposing target below.
func pictureBars4HShort() []market.Kline {
	vals := [][4]float64{
		{92, 92.5, 90.5, 91}, {91.5, 91.8, 90.2, 90}, {95, 96, 94, 95.5},
		{95.5, 96.8, 95, 96.5}, {96, 96.5, 94, 94.5}, {95.25, 96.2, 95, 95.5},
		{95.5, 96.3, 95.2, 95.9}, {95.75, 96.4, 95.4, 96},
	}
	out := make([]market.Kline, 0, len(vals))
	for i, v := range vals {
		out = append(out, mkBar(t4h0+int64(i)*4*3600*1000, 4*3600*1000, v[0], v[1], v[2], v[3]))
	}
	return out
}

func pictureBarsH1Short() []market.Kline {
	out := make([]market.Kline, 0, 42)
	for i := 0; i < 42; i++ {
		c := 96.3 - float64(i)*0.05
		out = append(out, mkBar(t4h0+int64(i)*3600*1000, 3600*1000, c+0.2, c+0.5, c-0.5, c))
	}
	out[40] = mkBar(t4h0+40*3600*1000, 3600*1000, 95.2, 95.6, 94.6, 95.0)  // prev (>= boundary)
	out[41] = mkBar(t4h0+41*3600*1000, 3600*1000, 94.9, 95.2, 94.0, 94.25) // cur (<= boundary − tick)
	return out
}

func pictureBars5MShort(withSwing bool) []market.Kline {
	base := t4h0 + 39*3600*1000 + 40*60*1000 // 16:40Z
	out := make([]market.Kline, 0, 28)
	for i := 0; i < 28; i++ {
		c := 96.0 - float64(i)*0.057
		hi := c + 0.5
		if withSwing && i == 22 {
			hi = 96.0 // the strict swing HIGH (neighbors' highs strictly lower)
		}
		out = append(out, mkBar(base+int64(i)*5*60*1000, 5*60*1000, c, hi, c-0.5, c-0.01))
	}
	return out
}

// Mirrored short end-to-end at the production call site: break down through
// the 94.5 support → swing HIGH stop + ONE tick (symmetric) → nearest support
// below (90) as the target. The broken level can never become its own target.
func TestPictureHtfEvaluatorMirroredShortEndToEnd(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.0})
	// W-EXEC-TRUTH W0 (Q7): the floor is max(knob, strategy floor) — this
	// geometry test sets both to 2.0 (the harness default floor is 2.5).
	env.at.config.StrategyConfig.RiskControl.MinRiskRewardRatio = 2.0
	bars4h := pictureBars4HShort()
	barsH1 := pictureBarsH1Short()
	bars5m := pictureBars5MShort(true)
	env.seed(bars4h, barsH1, bars5m)
	env.now = time.UnixMilli(t4h0 + 42*3600*1000 + 500) // 19:00:00.5Z
	env.eval.OnBars("MNQ", "5m", tailOf(bars5m, 1), env.now)
	if len(env.submits) != 1 {
		t.Fatalf("the mirrored short must submit once, got %d", len(env.submits))
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok {
		t.Fatalf("admitted row missing")
	}
	if row.Direction != "short" || row.LevelRole != "support" || row.LevelBodyBot != 94.5 {
		t.Fatalf("must be a short over the 94.5 support: %+v", row)
	}
	// ONE tick beyond the swing high, symmetrically: 96.0 + 0.25.
	if row.StopPx != 96.25 {
		t.Fatalf("short stop must be swing high + one tick (96.25), got %.2f", row.StopPx)
	}
	// Nearest support below entry — and never the broken 94.5 itself.
	if row.TargetPx != 90 {
		t.Fatalf("target must be the 90 support, got %.2f", row.TargetPx)
	}
	if row.TargetPx >= row.EntryRef {
		t.Fatalf("short target must be below entry: target %.2f entry %.2f", row.TargetPx, row.EntryRef)
	}
}

// ── Simultaneous H1/4H completion + arrival-order independence ───────────

// The last 4H candle completes AT the same boundary as the confirming H1 and
// closes ABOVE the broken resistance (but below the higher target): its
// retirement must NOT retro-kill the breakout (the pre-close reference is
// preserved), while it IS applied to target selection. Frame arrival order
// must not change anything.
func TestPictureHtfSimultaneousH1And4HCompletionPreservesBreakoutReference(t *testing.T) {
	for _, order := range []string{"4h-first", "5m-first"} {
		t.Run(order, func(t *testing.T) {
			env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
			bars4h := pictureBars4H()
			// Replace the last 4H candle: open 38h, close == the H1 confirm
			// close (42h−1ms), close 103 — above the 101 resistance (would
			// retire it) but below the 110 target (stays active).
			bars4h = bars4h[:len(bars4h)-1]
			bars4h = append(bars4h, mkBar(t4h0+38*3600*1000, 4*3600*1000, 101.5, 104, 101, 103))
			barsH1 := pictureBarsH1()
			bars5m := pictureBars5M(true)
			env.seed(bars4h, barsH1, bars5m)
			env.now = time.UnixMilli(t4h0 + 42*3600*1000 + 500)
			if order == "4h-first" {
				env.eval.OnBars("MNQ", "4h", tailOf(bars4h, 1), env.now.Add(-400*time.Millisecond))
				env.eval.OnBars("MNQ", "5m", tailOf(bars5m, 1), env.now)
			} else {
				env.eval.OnBars("MNQ", "5m", tailOf(bars5m, 1), env.now)
				env.eval.OnBars("MNQ", "4h", tailOf(bars4h, 1), env.now.Add(100*time.Millisecond))
			}
			if len(env.submits) != 1 {
				t.Fatalf("the breakout must fire on the pre-close reference: %d submits", len(env.submits))
			}
			row, ok, _ := env.st.PictureHtfGet(env.submits[0])
			if !ok {
				t.Fatalf("admitted row missing")
			}
			if row.LevelBodyTop != 101 {
				t.Fatalf("the breakout reference must be the 101 resistance as it stood before the H1 opened, got %.2f", row.LevelBodyTop)
			}
			if row.TargetPx != 110 {
				t.Fatalf("target must still be the 110 zone (the simultaneous close did not cross it): %.2f", row.TargetPx)
			}
		})
	}
}

// A new evaluator instance over the same trader/store (a restart) must not
// produce a second submission for an already-claimed opportunity.
func TestPictureHtfRestartAfterClaimSingleSubmission(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	env.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now)
	if len(env.submits) != 1 {
		t.Fatalf("first evaluation must submit once, got %d", len(env.submits))
	}
	// Restart: a brand-new evaluator with fresh in-memory state.
	env2 := &pictureHtfTestEnv{t: t, at: env.at, st: env.st, eval: NewPictureHtfEvaluator(env.at, store.PictureHtfResolved(&store.PictureHtfConfig{Enabled: true, MinRR: 2.5}))}
	orig := pictureHtfSubmitSeam
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env2.submits = append(env2.submits, row.OppKey)
		return nil
	}
	defer func() { pictureHtfSubmitSeam = orig }()
	env2.eval.OnBars("MNQ", "5m", tailOf(market.FuturesBarsProvider("MNQ", "5m", 28), 1), env.now)
	res := env2.eval.Evaluate("MNQ", env.now)
	if len(env2.submits) != 0 {
		t.Fatalf("a restarted evaluator must not re-submit: %d", len(env2.submits))
	}
	if res.Stage != "watching" || !strings.Contains(res.Reason, "already claimed") {
		t.Fatalf("the restart must report the durable claim, got %+v", res)
	}
}

// An ambiguous send (the wire call failed AFTER the claim) keeps the row
// place_pending and is NEVER blindly retried.
func TestPictureHtfAmbiguousSendStaysPending(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	orig := pictureHtfSubmitSeam
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env.submits = append(env.submits, row.OppKey)
		// The send STARTED (the stamp is written in beforeSend) and then
		// failed — its fate is unknown, which is what makes it ambiguous
		// (W-EXEC-TRUTH W0: a failure BEFORE the stamp is provably unsent and
		// settles refused instead).
		if err := e.at.store.PictureHtfStampSignal(row.OppKey, row.SignalID, "broker-ambiguous"); err != nil {
			return err
		}
		return fmt.Errorf("send ambiguous — wire refused after the claim")
	}
	defer func() { pictureHtfSubmitSeam = orig }()
	env.seedPictureTape()
	// Drive the FIRST evaluation through Evaluate (freshest receipt seeded the
	// same way OnBars would) so its verdict is observable.
	env.eval.freshest5mAt = env.now
	res := env.eval.Evaluate("MNQ", env.now)
	if res.Stage != "submitted" || !strings.Contains(res.Reason, "ambiguous") {
		t.Fatalf("an ambiguous send must be reported as such, got %+v", res)
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok || row.Stage != "place_pending" {
		t.Fatalf("the row must stay place_pending for reconciliation, got %+v", row)
	}
	// The blocked re-entry: the atomic claim requires stage='confirmed'.
	if won, err := env.st.PictureHtfClaimSubmission(row.OppKey, "retry-sig"); err != nil || won {
		t.Fatalf("an ambiguous place_pending row must block re-entry until reconciled: won=%v err=%v", won, err)
	}
}
