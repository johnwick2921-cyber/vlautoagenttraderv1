package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// W-FLIP-LINE-SIDE-OF-PRICE (2026-09-17) — the trader half. Write site: the
// real attempt loop (runPlannerReadCoreWithFactsGrades) rejects a flip line
// already beyond price on its own side, the retry prompt carries the
// validator's own sentence plus RepairFlipSideOfPriceLaw, and the repaired
// doc lands; an unknown authoring price is a WARN, never a reject, and the
// landed doc carries price_at_write only when the price was known. Read
// path: a stored impossible line is named once per plan version at BOTH
// evaluators (describeActivePlanDeath, describeDormantCleared), from the
// stamped price or the tape's close at created_at — never an invented one.

// sideOfPricePlanJSON is the class-39 fixture universe (price 15550) with a
// parametrized flip line; bias long, so the flip is side "below" → short.
func sideOfPricePlanJSON(flipPrice string) string {
	return `{
  "reasoning": "Balance below PDH; fade the PWL touch.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "2x5m < ` + flipPrice + `"},
  "levels": [
    {"price": 15480, "label": "PWL", "grade": "A", "instruction": "fade"},
    {"price": 15520, "label": "RN 15525", "grade": "B", "instruction": "fade"},
    {"price": 15575, "label": "RN 15575", "grade": "B", "instruction": "fade"},
    {"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade"},
    {"price": 15650, "label": "RN 15650", "grade": "B", "instruction": "fade"},
    {"price": 15700, "label": "RN 15700", "grade": "B", "instruction": "fade"}
  ],
  "scenarios": [{"id": "S1", "trigger": "fade the touch at 15480 PWL", "condition": "reject", "direction": "long",
    "target_chain": [15550, 15620], "invalid": "5m close below 15470", "quality": "A",
    "confirm": {"rule": "touch", "ref_price": 15480, "side": "below"},
    "economics":{"entry_zone":[15480,15480],"first_obstacle":{"price":15550,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":7,"r_to_arm_target":7},
    "arm": {"enabled": true, "entry": 15480, "stop": 15470, "target": 15550, "wait_confirm": true}}],
  "no_trade": ["first 5m"],
  "death_condition": "acceptance above 15620",
  "death": {"price": 15620, "side": "above", "rule": "2x5m"},
  "flip": {"price": ` + flipPrice + `, "side": "below", "rule": "2x5m", "flip_to": "short"},
  "day_type": "balance"
}`
}

// Write site: flip{15600 below → short} with price 15550 is a line ABOVE price
// on side "below" — rejected with the validator's sentence; attempt 2's repair
// prompt carries that sentence and the law excerpt; the repaired doc
// (flip{15480 below}) lands with price_at_write stamped.
func TestWriteSiteRejectsFlipLineOnWrongSideOfPrice(t *testing.T) {
	at := plannerTestTrader(t)
	facts := kernel.PlanFacts{Price: 15550, DATR: 300}
	machine := map[float64]string{15480: "PWL", 15700: "RN 15700"}
	prompts := []string{}
	ver, lc, err := at.runPlannerReadCoreWithFactsGradesClock(sideOfPriceClock, "NY", "2026-09-01", "owner_reset",
		"deepseek-v4-pro", "hashFSP1", "", "", "", "PROMPT", facts, nil, machine, nil, true,
		func(userPrompt string) (string, error) {
			prompts = append(prompts, userPrompt)
			if len(prompts) == 1 {
				return sideOfPricePlanJSON("15600"), nil // the ASIA v2 shape, mirrored for a long bias
			}
			return sideOfPricePlanJSON("15480"), nil // repaired: line below price on side below
		})
	if err != nil || lc != "active" {
		t.Fatalf("the repaired doc must land on attempt 2: ver=%d lc=%q err=%v", ver, lc, err)
	}
	if len(prompts) != 2 {
		t.Fatalf("attempt 1 must have been rejected and attempt 2 accepted; got %d call(s)", len(prompts))
	}
	want := "flip{below 15600.00 → short} is already above price 15550.00 at authoring: a flip line must sit on the far side of price (it can never be touched from the near side)"
	if !strings.Contains(prompts[1], want) {
		t.Fatalf("the retry must carry the validator's sentence %q, got:\n%s", want, prompts[1])
	}
	if !strings.Contains(prompts[1], kernel.RepairFlipSideOfPriceLaw) {
		t.Fatalf("the retry must carry RepairFlipSideOfPriceLaw, got:\n%s", prompts[1])
	}
	row, rerr := at.store.Plan().GetLatestPlanForTraderSession("2026-09-01", "NY", at.id)
	if rerr != nil || row == nil {
		t.Fatalf("read back: %v", rerr)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.PriceAtWrite != 15550 {
		t.Fatalf("the landed doc must carry price_at_write=15550 (the price its lines were judged against), got %v", doc.PriceAtWrite)
	}
	if doc.FlipStructured == nil || doc.FlipStructured.Price != 15480 {
		t.Fatalf("the landed doc must be the repaired one, got flip=%+v", doc.FlipStructured)
	}
}

// Unknown authoring price (facts absent): the impossible line is NOT
// rejected — never invent a price — the write site says so once, and the
// landed doc carries no price_at_write.
func TestWriteSiteUnknownPriceWarnsNeverRejects(t *testing.T) {
	at := plannerTestTrader(t)
	buf := captureTraderLog(t)
	calls := 0
	ver, lc, err := at.runPlannerReadCoreWithFactsGradesClock(sideOfPriceClock, "NY", "2026-09-02", "owner_reset",
		"deepseek-v4-pro", "hashFSP2", "", "", "", "PROMPT", kernel.PlanFacts{}, nil, nil, nil, true,
		func(userPrompt string) (string, error) {
			calls++
			return sideOfPricePlanJSON("15600"), nil
		})
	if err != nil || lc != "active" {
		t.Fatalf("with no authoring price the doc must land on attempt 1: ver=%d lc=%q err=%v\n%s", ver, lc, err, buf.String())
	}
	if calls != 1 {
		t.Fatalf("must NOT burn a retry on an unknown price — %d calls", calls)
	}
	if !strings.Contains(buf.String(), "flip/death line side-of-price UNJUDGED: authoring price unknown") {
		t.Fatalf("the write site must WARN that the line went unjudged, got:\n%s", buf.String())
	}
	row, rerr := at.store.Plan().GetLatestPlanForTraderSession("2026-09-02", "NY", at.id)
	if rerr != nil || row == nil {
		t.Fatalf("read back: %v", rerr)
	}
	if strings.Contains(row.Doc, "price_at_write") {
		t.Fatalf("an unknown price must not be stamped: %s", row.Doc)
	}
}

// --- read path -------------------------------------------------------------

// beyondPriceShortPlanDoc is the ASIA v2 shape on the flip-hold universe:
// short bias, flip{100 above → long} and death{102 above}, authored at 104.
func beyondPriceShortPlanDoc(t *testing.T, stamp float64) string {
	t.Helper()
	doc := kernel.PlanDoc{
		Bias:            kernel.PlanBias{Direction: "short", FlipCondition: "flips long on 2x5m above 100"},
		FlipStructured:  &kernel.PlanCondition{Price: 100, Side: "above", Rule: "2x5m", FlipTo: "long"},
		DeathStructured: &kernel.PlanCondition{Price: 102, Side: "above", Rule: "2x5m"},
		PriceAtWrite:    stamp,
	}
	blob, _ := json.Marshal(doc)
	return string(blob)
}

func TestDescribeActivePlanDeath_StoredLineBeyondPriceNamedOnce(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-20"
	row := appendVersion(t, st, at, td, "NY_scheduled_read", beyondPriceShortPlanDoc(t, 104), now.Add(-45*time.Minute))
	barsAt(flipHoldTape(now, 40, 10))
	buf := captureTraderLog(t)

	at.describeActivePlanDeath(row)
	at.describeActivePlanDeath(row) // second evaluation adds no line

	log := buf.String()
	if !strings.Contains(log, "flip_line_beyond_price plan="+row.PlanID+" v1 (active) flip{above 100.00 → long} is already below price 104.00 at authoring") {
		t.Fatalf("a stored flip line beyond price must be named at the active evaluator from the stamped price; got:\n%s", log)
	}
	if !strings.Contains(log, "death_line_beyond_price plan="+row.PlanID+" v1 (active) death{above 102.00} is already below price 104.00 at authoring") {
		t.Fatalf("a stored death line beyond price must be named too; got:\n%s", log)
	}
	if n := strings.Count(log, "flip_line_beyond_price"); n != 1 {
		t.Fatalf("the flip line must print once per plan version, got %d:\n%s", n, log)
	}
	if n := strings.Count(log, "death_line_beyond_price"); n != 1 {
		t.Fatalf("the death line must print once per plan version, got %d:\n%s", n, log)
	}
}

func TestDescribeDormantCleared_StoredLineBeyondPriceNamed(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-21" // distinct plan id (once-per-version memory is process-wide)
	row := appendVersion(t, st, at, td, "dormant:flip:flip-condition", beyondPriceShortPlanDoc(t, 104), now.Add(-45*time.Minute))
	barsAt(flipHoldTape(now, 40, 10))
	buf := captureTraderLog(t)

	at.describeDormantCleared(row)

	if !strings.Contains(buf.String(), "flip_line_beyond_price plan="+row.PlanID+" v1 (dormant)") {
		t.Fatalf("a dormant plan's impossible flip line must be named at its evaluator; got:\n%s", buf.String())
	}
}

// A row written BEFORE the stamp existed: the authoring price is the tape's
// last closed 1m bar at created_at (100 on this tape, 45 min before the
// break), so flip{98 above} is named and flip{100 above} at price 100 is
// "at price" — nothing is invented.
func TestDescribeActivePlanDeath_UnstampedRowJudgedFromTapeAtCreatedAt(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-24"
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "short"}, FlipStructured: &kernel.PlanCondition{Price: 98, Side: "above", Rule: "2x5m", FlipTo: "long"}}
	blob, _ := json.Marshal(doc)
	row := appendVersion(t, st, at, td, "NY_scheduled_read", string(blob), now.Add(-45*time.Minute))
	barsAt(flipHoldTape(now, 60, 10)) // tape covers created_at (now-45m) at close 100
	buf := captureTraderLog(t)

	at.describeActivePlanDeath(row)

	if !strings.Contains(buf.String(), "flip_line_beyond_price plan="+row.PlanID+" v1 (active) flip{above 98.00 → long} is already below price 100.00 at authoring") {
		t.Fatalf("an unstamped row must be judged from the tape's close at created_at; got:\n%s", buf.String())
	}
}

// A correctly-placed stored line is silent.
func TestDescribeActivePlanDeath_StoredLineOnFarSideSilent(t *testing.T) {
	at, st, now := flipHoldTrader(t)
	td := "2026-08-25"
	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "short"}, FlipStructured: &kernel.PlanCondition{Price: 100, Side: "above", Rule: "2x5m", FlipTo: "long"}, PriceAtWrite: 99}
	blob, _ := json.Marshal(doc)
	row := appendVersion(t, st, at, td, "NY_scheduled_read", string(blob), now.Add(-45*time.Minute))
	barsAt(flipHoldTape(now, 40, 10))
	buf := captureTraderLog(t)

	at.describeActivePlanDeath(row)

	if strings.Contains(buf.String(), "_line_beyond_price") {
		t.Fatalf("a line on the far side of price must not be named:\n%s", buf.String())
	}
}

// authoringPriceFor: the stamp wins; else the tape's last close before
// created_at within the window; else UNKNOWN (0) — never a later bar, never
// a stale one.
func TestAuthoringPriceFor(t *testing.T) {
	created := time.Date(2026, 9, 17, 22, 52, 4, 0, time.UTC)
	ms := created.UnixMilli()
	bar := func(closeMs int64, close float64) market.Kline {
		return market.Kline{OpenTime: closeMs - 59_999, CloseTime: closeMs, Close: close}
	}
	tape := []market.Kline{bar(ms-180_000, 29760), bar(ms-120_000, 29764), bar(ms+60_000, 29770)}
	if got := authoringPriceFor(&kernel.PlanDoc{PriceAtWrite: 29764.25}, created, tape); got != 29764.25 {
		t.Fatalf("stamp must win, got %v", got)
	}
	if got := authoringPriceFor(&kernel.PlanDoc{}, created, tape); got != 29764 {
		t.Fatalf("unstamped: last close BEFORE created_at, got %v", got)
	}
	if got := authoringPriceFor(&kernel.PlanDoc{}, created, []market.Kline{bar(ms-authoringPriceWindowMs-1, 29700)}); got != 0 {
		t.Fatalf("a close older than the window is UNKNOWN, got %v", got)
	}
	if got := authoringPriceFor(&kernel.PlanDoc{}, created, []market.Kline{bar(ms+60_000, 29770)}); got != 0 {
		t.Fatalf("a bar that closed after created_at is not the authoring close, got %v", got)
	}
	if got := authoringPriceFor(&kernel.PlanDoc{}, time.Time{}, tape); got != 0 {
		t.Fatalf("no created_at → UNKNOWN, got %v", got)
	}
}

// sideOfPriceClock is the write loop's authoring clock for these tests (class
// 60/113: no wall clock in a test); the fixture universe has no clock rules.
func sideOfPriceClock() time.Time { return time.Date(2026, 9, 1, 14, 35, 30, 0, time.UTC) }
