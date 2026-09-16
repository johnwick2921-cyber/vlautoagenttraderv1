package kernel

import (
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"nofx/market"
)

// THE 2026-08-16 ASIA POST-MORTEM — a replay of all six versions.
//
// Six consecutive plans died in 25 minutes. This file replays each death check
// against the REAL inputs and asserts that under the fixed logic every version
// survives. If a future change resurrects the loop, these six fail.
//
// INPUTS, all real:
//   - Levels + born/died timestamps: `plans` table + journald (unit nofx).
//   - Post-reopen bars: captured verbatim from the live NT8 cache via
//     GET /api/klines (testdata/asia_2026-08-16_reopen_1m.json, 45 bars,
//     22:01–22:45Z, zero flat bars — the session was genuinely trading).
//   - The Friday block is RECONSTRUCTED rather than captured, because the padded
//     bars have since been purged from the cache. Its shape is not a guess: NT8's
//     .ncd file was byte-decoded as 1438 identical empty-minute placeholders with
//     a header base price of exactly 30147.50, which the bar builder materialised
//     as O=H=L=C=30147.50, volume 0. buildFridayPadding reproduces exactly that.

//go:embed testdata/asia_2026-08-16_reopen_1m.json
var asiaReopenBarsJSON []byte

// The REAL bars the 2000-bar window reached back into: Thu 22:56Z → Fri 04:59Z,
// range 30124.25–30232.50. This half of the cache is what "touched" the upper
// levels (ONH 30200.50 / 30203) — two calendar days before the plan was written.
//
//go:embed testdata/asia_2026-08-16_prefriday_1m.json
var asiaPreFridayBarsJSON []byte

type replayBar struct {
	T int64   `json:"t"`
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"`
}

func ms(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.UnixMilli()
}

// asiaVersion is one row of the post-mortem table.
type asiaVersion struct {
	version int
	born    int64
	died    int64
	rule    string // the acceptance rule in force for ASIA
	levels  []PlanLevel
}

func lvl(label string, price float64) PlanLevel {
	return PlanLevel{Price: price, Label: label, Grade: "A", Instruction: "watch"}
}

// The six versions exactly as stored. Note the prior-day trio is IDENTICAL in
// every one: PDL 30146.75 / PDC 30147.50 / PDH 30148.25 — a 1.5-POINT CLUSTER
// sitting ~45 points below spot. v1 even labels it "PDC/RTH-H/RTH-L", i.e. the
// planner was told Friday's close, RTH high and RTH low were all the same price.
// They were: Friday was 959 padded bars at 30147.50, and PDH/PDL are precisely
// the high and low of the single real Friday bar (h30148.25 / l30146.75).
func asiaVersions6() []asiaVersion {
	prior := []PlanLevel{
		lvl("PDL", 30146.75), lvl("PDC", 30147.5), lvl("PDH", 30148.25),
		lvl("ONL", 30166.25),
	}
	with := func(eql, onh float64) []PlanLevel {
		out := append([]PlanLevel{}, prior...)
		return append(out, lvl("EQL", eql), lvl("ONH", onh))
	}
	return []asiaVersion{
		{1, ms("2026-08-16T22:12:46Z"), ms("2026-08-16T22:17:53Z"), "2x5m", with(30192.25, 30200.5)},
		{2, ms("2026-08-16T22:19:02Z"), ms("2026-08-16T22:20:58Z"), "2x5m", with(30192.25, 30203)},
		{3, ms("2026-08-16T22:23:34Z"), ms("2026-08-16T22:25:11Z"), "2x5m", with(30191.25, 30203)},
		{4, ms("2026-08-16T22:26:19Z"), ms("2026-08-16T22:31:20Z"), "2x5m", with(30191.25, 30203)},
		{5, ms("2026-08-16T22:32:28Z"), ms("2026-08-16T22:37:29Z"), "2x5m", with(30199.5, 30203)},
		// v6 is the levels:null NO-TRADE plan the exhausted budget produced. It is
		// the CONSEQUENCE, not a death: a plan with no levels can never die this way.
		{6, ms("2026-08-16T22:37:29Z"), ms("2026-08-16T22:45:00Z"), "2x5m", nil},
	}
}

// buildFridayPadding reproduces the poisoned Friday block the cache held at the
// time: contiguous 1-minute bars, O=H=L=C=30147.50, volume 0, from Fri 00:02 CT
// to 16:00 CT. This is what made PDC "touched" — the level sat EXACTLY on the
// padding price — and what the prior-day levels were derived from in the first
// place.
func buildFridayPadding() []market.Kline {
	start := ms("2026-08-14T05:02:00Z") // Fri 00:02 CT
	end := ms("2026-08-14T21:00:00Z")   // Fri 16:00 CT
	const px = 30147.5
	var out []market.Kline
	for t := start; t < end; t += 60_000 {
		out = append(out, market.Kline{
			OpenTime: t, CloseTime: t + 59_999,
			Open: px, High: px, Low: px, Close: px, Volume: 0,
		})
	}
	return out
}

// fridayRealBoundaryBar is the ONE genuine Friday bar (00:01 is absent, 00:02
// onward is the flat block). It matters enormously: its high and low ARE
// PDH 30148.25 and PDL 30146.75 — a whole day's prior-day levels came from a
// single minute of trade — and it is the only thing in the entire Friday window
// that can "touch" either, which is why the depth sweep found the plans flipping
// DEAD the moment the window reached back to 05:00Z.
func fridayRealBoundaryBar() market.Kline {
	t := ms("2026-08-14T05:00:00Z")
	return market.Kline{
		OpenTime: t, CloseTime: t + 59_999,
		Open: 30147.75, High: 30148.25, Low: 30146.75, Close: 30147.25, Volume: 53,
	}
}

func loadBars(t *testing.T, blob []byte) []market.Kline {
	t.Helper()
	var raw []replayBar
	if err := json.Unmarshal(blob, &raw); err != nil {
		t.Fatal(err)
	}
	out := make([]market.Kline, len(raw))
	for i, b := range raw {
		out[i] = market.Kline{
			OpenTime: b.T, CloseTime: b.T + 59_999,
			Open: b.O, High: b.H, Low: b.L, Close: b.C, Volume: b.V,
		}
	}
	return out
}

// cacheAsItWasAtDeath rebuilds the exact 2000-bar window the death check saw:
// two days of real history, the single real Friday bar, Friday's 16-hour padded
// block, and the Sunday reopen.
func cacheAsItWasAtDeath(t *testing.T) []market.Kline {
	t.Helper()
	out := loadBars(t, asiaPreFridayBarsJSON)
	out = append(out, fridayRealBoundaryBar())
	out = append(out, buildFridayPadding()...)
	return append(out, loadBars(t, asiaReopenBarsJSON)...)
}

func loadReopenBars(t *testing.T) []market.Kline {
	t.Helper()
	var raw []replayBar
	if err := json.Unmarshal(asiaReopenBarsJSON, &raw); err != nil {
		t.Fatal(err)
	}
	out := make([]market.Kline, len(raw))
	for i, b := range raw {
		out[i] = market.Kline{
			OpenTime: b.T, CloseTime: b.T + 59_999,
			Open: b.O, High: b.H, Low: b.L, Close: b.C, Volume: b.V,
		}
	}
	return out
}

// planIsDeadPreFix reproduces the death check EXACTLY as it stood before
// 06f3343e/20a64c51: the whole ~33h cache, acceptance counted on raw 1-minute
// bars. Calling today's PlanIsDead here would not replay the outage — it already
// carries the timeframe fix — so the "before" column would understate the bug.
func planIsDeadPreFix(doc PlanDoc, bars []market.Kline, rule string, now int64) bool {
	if len(doc.Levels) == 0 {
		return false
	}
	for _, l := range doc.Levels {
		touched := levelTouched(bars, l.Price, now)
		consumed := !LevelStillValid(bars, l.Price, rule, now)
		if !(touched && consumed) {
			return false
		}
	}
	return true
}

// THE REGRESSION TEST. Replays all six versions and asserts they survive.
func TestASIA6Deaths_AllSurviveUnderFixedLogic(t *testing.T) {
	cacheAsItWas := cacheAsItWasAtDeath(t)

	deaths := 0
	for _, v := range asiaVersions6() {
		doc := PlanDoc{Levels: v.levels}

		// BEFORE: the whole ~33h cache, acceptance counted on 1-minute bars.
		before := planIsDeadPreFix(doc, cacheAsItWas, v.rule, v.died)
		// AFTER: only bars at/after the plan was written, acceptance on the rule's
		// own timeframe.
		after := PlanIsDeadSince(doc, cacheAsItWas, v.rule, v.born, v.died)

		t.Logf("v%d born %s died %s | levels %d | BEFORE dead=%v | AFTER dead=%v",
			v.version,
			time.UnixMilli(v.born).UTC().Format("15:04:05"),
			time.UnixMilli(v.died).UTC().Format("15:04:05"),
			len(v.levels), before, after)

		if after {
			t.Errorf("v%d STILL DIES under the fixed logic — that is a third cause, find it", v.version)
		}
		if before {
			deaths++
		}
	}

	// The fixture must actually reproduce the outage, or it guards nothing. v6
	// carries no levels and can never die this way, so five is the maximum.
	if deaths < 5 {
		t.Errorf("the replay reproduced only %d/5 deaths under the OLD logic — the fixture no longer represents the outage", deaths)
	}
}

// The padding refusal alone also breaks the loop: with the synthetic Friday bars
// removed at ingest (as they now are), even the OLD whole-cache check has nothing
// to declare PDC "touched" with.
func TestASIA6Deaths_PaddingRemovalAloneBreaksTheLoop(t *testing.T) {
	reopen := loadReopenBars(t) // the cache as it looks now: real bars only
	for _, v := range asiaVersions6() {
		if len(v.levels) == 0 {
			continue
		}
		if planIsDeadPreFix(PlanDoc{Levels: v.levels}, reopen, v.rule, v.died) {
			t.Errorf("v%d dies even against real-bars-only history — padding was not the only cause", v.version)
		}
	}
}

// WHY the prior-day levels were nonsense in the first place: a 1.5-point cluster
// of PDH/PDC/PDL is not a plan, it is a flat day rendered as three levels. This
// pins the detector-facing lesson so a future reader sees it.
func TestASIA6Deaths_PriorDayTrioWasDerivedFromAFlatDay(t *testing.T) {
	v := asiaVersions6()[0]
	var pdh, pdl, pdc float64
	for _, l := range v.levels {
		switch l.Label {
		case "PDH":
			pdh = l.Price
		case "PDL":
			pdl = l.Price
		case "PDC":
			pdc = l.Price
		}
	}
	if spread := pdh - pdl; spread > 2 {
		t.Fatalf("fixture drift: the prior-day range was %.2f points, but the outage's was 1.50", spread)
	}
	// Every padded Friday bar sat exactly on PDC, so PDC was "touched" by 959 bars
	// that never traded.
	padding := buildFridayPadding()
	if !levelTouched(padding, pdc, ms("2026-08-16T22:17:53Z")) {
		t.Error("PDC must be touched by the padding — that is the mechanism")
	}
	if levelTouched(padding, pdl, ms("2026-08-16T22:17:53Z")) {
		t.Error("PDL sits 0.75 below the padding price and must NOT be touched by it")
	}
}
