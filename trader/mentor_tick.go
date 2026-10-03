package trader

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"vl/calendar"
	"vl/kernel"
	"vl/kernel/mentor"
	"vl/logger"
	"vl/market"
	ntwire "vl/provider/ninjatrader"
	"vl/store"
)

// ── MENTOR P3 — the cycle hook ─────────────────────────────────────────────
//
// One evaluator tick per NEW 1m close. Inert unless the per-strategy
// mentor_mode is ON. Placements stay behind the MENTOR_PLACE env gate until
// P1 (#309) lands — until then the hook sizes and LOGS/COUNTS every intent
// (the size audit the spec demands) and places NOTHING (L4, default OFF).

// mentorPlaceEnv resolves the placement gate (env MENTOR_PLACE, default OFF).
func mentorPlaceEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MENTOR_PLACE"))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// mentorTick runs the evaluator on the latest 1m bars (the 2-minute scan's
// fallback — the PRIMARY path is the event pass on every FINAL 1m bar). It
// never touches the wire unless mentor mode AND MENTOR_PLACE are both on.
func (at *AutoTrader) mentorTick(ctx *kernel.Context) {
	if !at.mentorEnabled() {
		return
	}
	if market.FuturesBarsProvider == nil {
		return
	}
	bars := market.FuturesBarsProvider("MNQ", "1m", 240)
	if len(bars) == 0 {
		return
	}
	at.mentorEvalOnce(bars)
}

// noteLiveBarsForMentorPass is the sink's mentor half: a FINAL 1m/5m frame
// stamps the arrival time and kicks the event loop for a mentor-mode trader
// (event-driven, not scan-driven).
func (at *AutoTrader) noteLiveBarsForMentorPass(symbol, tf string, bars []ntwire.Bar, receivedAt time.Time) {
	if at == nil || len(bars) == 0 || !at.mentorEnabled() {
		return
	}
	if tf != "1m" && tf != "5m" {
		return
	}
	root := instrumentRoot(symbol)
	if root == "" || root != instrumentRoot(at.futuresSymbol()) {
		return
	}
	final := false
	for _, b := range bars {
		if b.Final {
			final = true
			break
		}
	}
	if !final {
		return
	}
	at.mentorFinalArrival.Store(receivedAt.UnixMilli())
	if l := at.armedEvent.Load(); l != nil {
		l.poke()
	}
}

// mentorEventPassAt runs one mentor event pass (the event loop calls it after
// the armed pass; ≤1/s by the loop's own gap). Returns whether it ran.
func (at *AutoTrader) mentorEventPassAt(now time.Time) bool {
	if at == nil || !at.mentorEnabled() {
		return false
	}
	if market.FuturesBarsProvider == nil {
		return false
	}
	bars := market.FuturesBarsProvider("MNQ", "1m", 240)
	if len(bars) == 0 {
		return false
	}
	if bars[len(bars)-1].OpenTime <= at.mentorLastTickOpen {
		return false // already evaluated this bar (the scan or an earlier pass)
	}
	at.mentorEvalOnce(bars)
	return true
}

// mentorEvalOnce runs one evaluator tick over the bars and processes every
// intent (size → latency → no-chase → place-or-hold).
func (at *AutoTrader) mentorEvalOnce(bars []market.Kline) {
	last := bars[len(bars)-1]
	at.mentorLastTickOpen = last.OpenTime

	if at.mentorEval == nil {
		cfg := mentor.DefaultConfig()
		cfg.Enabled = true
		at.mentorEval = mentor.New(cfg)
	}
	emitMs := time.Now().UnixMilli()
	intents := at.mentorEval.Tick(bars, last.OpenTime)
	// S9 (D5.2 p2 @05:21): strong-day detection from the recent CLOSED 5m bars
	// — 50–80 pt candles cut every tier to 1–2.
	extra := mentorTierInputs{}
	if bars5 := market.FuturesBarsProvider("MNQ", "5m", 12); len(bars5) > 0 && mentorStrongDayFrom5m(bars5) {
		extra.StrongDay = true
	}
	for _, in := range intents {
		switch in.Action {
		case mentor.PlaceStopEntry:
			if why := mentorRuleGate(in, extra); why != "" {
				rule := "other"
				if i := strings.Index(why, ":"); i > 0 {
					rule = strings.ToLower(strings.TrimSpace(why[:i]))
				}
				mentorCount("refused_" + rule)
				at.logWarnf("🧑‍🏫 mentor intent REFUSED — %s", why)
				continue
			}
			choice, err := at.mentorSizeFor(in, extra)
			if err != nil {
				continue
			}
			mentorCount("intent_" + in.Setup)
			if !mentorPlaceEnv() {
				at.logInfof("🧑‍🏫 mentor intent SIZED, NOT PLACED (MENTOR_PLACE off — P1 #309 first): %s %s %d contracts @ %.2f (stop %.2f, target %.2f, tier %s)",
					in.Setup, in.Side, choice.Contracts, in.Price, in.Stop, in.Target, choice.Tier)
				mentorCount("placement_held")
				continue
			}
			at.mentorPlaceIntent(in, choice, last.CloseTime, emitMs)
		case mentor.CancelArm, mentor.LevelInvalid:
			// P3 scope: the arm lifecycle (cancel frames, level invalidation
			// bookkeeping) lands with P1; the intent is recorded, never silent.
			mentorCount("intent_" + string(in.Action))
			at.logInfof("🧑‍🏫 mentor %s intent recorded (arm lifecycle lands with P1): %s", in.Action, in.Reason)
		}
	}
}

// mentorPlaceIntent converts a sized intent into a synthetic decision and runs
// the FULL existing pipeline (every admission gate reused, SIM-only untouched).
// The no-chase rule runs FIRST: a stop entry whose price is already through the
// trigger is skipped — he never enters at market (§3).
func (at *AutoTrader) mentorPlaceIntent(in mentor.Intent, choice mentorSizeChoice, barCloseMs, emitMs int64) {
	if hold, why := at.mentorNewsGate(); hold {
		mentorCount("news_hold")
		at.logWarnf("🧑‍🏫 mentor placement REFUSED — %s", why)
		return
	}
	if latest, ok := at.mentorLatestPrice(); ok {
		if skip, why := mentorNoChase(in.Side, latest, in.Price); skip {
			mentorCount("no_chase_skip")
			at.logWarnf("🧑‍🏫 mentor NO-CHASE skip: %s", why)
			return
		}
	}
	if mentorPlaceRecorderForTest != nil {
		mentorPlaceRecorderForTest(in, choice.Contracts)
		return // test seam: the real pipeline is never reached from a test
	}
	d := mentor.BuildDecision(in, "MNQ", choice.Contracts)
	rec := &store.DecisionAction{
		Action:     d.Action,
		Symbol:     d.Symbol,
		Quantity:   0,
		Leverage:   d.Leverage,
		Price:      0,
		StopLoss:   d.StopLoss,
		TakeProfit: d.TakeProfit,
		Confidence: d.Confidence,
		Reasoning:  d.Reasoning,
		Timestamp:  time.Now().UTC(),
		Success:    false,
	}
	if err := at.executeDecisionWithRecord(d, rec); err != nil {
		mentorCount("placement_error")
		at.logErrorf("🧑‍🏫 mentor placement error: %v", err)
		return
	}
	mentorCount("placed_" + choice.Tier)
	ackMs := time.Now().UnixMilli()
	recordMentorLatency(barCloseMs, at.mentorFinalArrival.Load(), emitMs, ackMs)
	at.logInfof("🧑‍🏫 mentor placed: %s %s %d contracts (tier %s) — close→ack %dms", in.Setup, in.Side, choice.Contracts, choice.Tier, ackMs-barCloseMs)
}

// mentorLatestPrice is the latest live price for the no-chase check. The
// default reads the newest cached 1m bar's close (the freshest tape we have
// without a dedicated tick accessor); tests substitute their own source.
var mentorLatestPriceSource func() (float64, bool)

func (at *AutoTrader) mentorLatestPrice() (float64, bool) {
	if mentorLatestPriceSource != nil {
		return mentorLatestPriceSource()
	}
	if market.FuturesBarsProvider == nil {
		return 0, false
	}
	bars := market.FuturesBarsProvider("MNQ", "1m", 1)
	if len(bars) == 0 {
		return 0, false
	}
	return bars[len(bars)-1].Close, true
}

// mentorNoChase is the pure no-chase rule: a stop entry whose trigger is
// already AT OR BEYOND the live price must be skipped — a buy stop below the
// market (or a sell stop above it) fills like a market order, and he never
// enters at market (§3).
func mentorNoChase(side mentor.Side, latest float64, trigger float64) (skip bool, why string) {
	switch side {
	case mentor.SideLong:
		if latest >= trigger {
			return true, fmt.Sprintf("price %.2f already at/beyond the buy stop entry %.2f — no chase [§3]", latest, trigger)
		}
	case mentor.SideShort:
		if latest <= trigger {
			return true, fmt.Sprintf("price %.2f already at/beyond the sell stop entry %.2f — no chase [§3]", latest, trigger)
		}
	}
	return false, ""
}

// mentorPlaceRecorderForTest is a test seam called right before the real
// placement: the no-chase mutant (dropping the check) makes it fire on a
// through-price intent and the test goes RED.
var mentorPlaceRecorderForTest func(in mentor.Intent, contracts int)

// ── STRONG DAY (S9) ────────────────────────────────────────────────────────

const mentorStrongDayPtsMin = 50.0 // a 5m candle range ≥50 pts = the 50–80 band

// mentorStrongDayFrom5m reports a strong day: a closed 5m candle in the window
// ran 50–80 pts [D5.2 p2 @05:21] (anything wider is certainly strong — the 80
// is the band's top as quoted, not a cutoff).
func mentorStrongDayFrom5m(bars []market.Kline) bool {
	for _, b := range bars {
		if b.High-b.Low >= mentorStrongDayPtsMin {
			return true
		}
	}
	return false
}

// ── NEWS 07:30 CT (F11/R12) ────────────────────────────────────────────────

// The 07:30 CT CPI/PPI/Unemployment print: NO resting order through it. The
// hold window is the print minute −10m (no order may still be resting INTO the
// print) to +5m (let the print shake out) — both named rule parameters.
const (
	mentorNewsPreWindow  = 10 * time.Minute
	mentorNewsPostWindow = 5 * time.Minute
)

// mentorNewsPrintTitleTokens matches the BLS 07:30 CT prints the rule names.
var mentorNewsPrintTitleTokens = []string{"cpi", "ppi", "unemployment"}

// mentorNewsHold is the pure gate: with the day's calendar events, reports
// whether a placement at `now` would rest through a 07:30 CT T1 CPI/PPI/
// Unemployment print.
func mentorNewsHold(events []calendar.Event, now time.Time) (hold bool, why string) {
	loc := kernel.CTLocation()
	ct := now.In(loc)
	printAt := time.Date(ct.Year(), ct.Month(), ct.Day(), 7, 30, 0, 0, loc)
	if now.Before(printAt.Add(-mentorNewsPreWindow)) || !now.Before(printAt.Add(mentorNewsPostWindow)) {
		return false, ""
	}
	for _, e := range events {
		if e.Impact != calendar.T1 || e.Time.In(loc).Format("15:04") != "07:30" {
			continue
		}
		title := strings.ToLower(e.Title)
		for _, tok := range mentorNewsPrintTitleTokens {
			if strings.Contains(title, tok) {
				return true, fmt.Sprintf("news: %s prints 07:30 CT — no resting order through the print [F11]", e.Title)
			}
		}
	}
	return false, ""
}

// mentorNewsNowSource / mentorDayEventsForTest are the test seams for the news
// gate (nil in production: real clock + the stored calendar slice).
var (
	mentorNewsNowSource    func() time.Time
	mentorDayEventsForTest func() []calendar.Event
)

// mentorDayEvents returns today's stored calendar events. No slice → nil (no
// evidence of a print — the gate allows; the producer is the same one the
// day-plan uses).
func (at *AutoTrader) mentorDayEvents() []calendar.Event {
	if mentorDayEventsForTest != nil {
		return mentorDayEventsForTest()
	}
	if at.store == nil {
		return nil
	}
	slice, err := at.store.Calendar().GetSlice(plannerTradeDateCT(time.Now()))
	if err != nil || slice == nil {
		return nil
	}
	var evs []calendar.Event
	if json.Unmarshal([]byte(slice.EventsJSON), &evs) != nil {
		return nil
	}
	return evs
}

// mentorNewsGate is the call-site half of F11: refuse a placement inside the
// 07:30 print window on a print day.
func (at *AutoTrader) mentorNewsGate() (bool, string) {
	now := time.Now()
	if mentorNewsNowSource != nil {
		now = mentorNewsNowSource()
	}
	return mentorNewsHold(at.mentorDayEvents(), now)
}

// ── LATENCY (§2: measure it) ───────────────────────────────────────────────

const mentorLatencyCap = 256

var (
	mentorLatencyMu   sync.Mutex
	mentorLatencySamp []int64 // close→ack ms, capped ring
)

// recordMentorLatency logs the four stamps and feeds the p50/p95 reservoir.
// A close→ack above 3 s logs a WARN (the target is under 1 s).
func recordMentorLatency(closeMs, arrivalMs, emitMs, ackMs int64) {
	d := ackMs - closeMs
	if d > 3000 {
		logger.Warnf("🧑‍🏫 mentor eval LATENCY WARN: close→ack %dms (close=%d arrival=%d emit=%d ack=%d) — target < 1000ms",
			d, closeMs, arrivalMs, emitMs, ackMs)
	} else {
		logger.Infof("🧑‍🏫 mentor eval latency: close→ack %dms (close=%d arrival=%d emit=%d ack=%d)",
			d, closeMs, arrivalMs, emitMs, ackMs)
	}
	mentorLatencyMu.Lock()
	mentorLatencySamp = append(mentorLatencySamp, d)
	if len(mentorLatencySamp) > mentorLatencyCap {
		mentorLatencySamp = mentorLatencySamp[len(mentorLatencySamp)-mentorLatencyCap:]
	}
	mentorLatencyMu.Unlock()
}

// MentorLatencySnapshot returns n samples and the p50/p95 of close→ack (ms).
func MentorLatencySnapshot() (n int, p50, p95 int64) {
	mentorLatencyMu.Lock()
	samp := append([]int64(nil), mentorLatencySamp...)
	mentorLatencyMu.Unlock()
	n = len(samp)
	if n == 0 {
		return 0, 0, 0
	}
	sort.Slice(samp, func(i, j int) bool { return samp[i] < samp[j] })
	p50 = samp[n*50/100]
	p95 = samp[n*95/100]
	return n, p50, p95
}

// ResetMentorLatencyForTest clears the reservoir.
func ResetMentorLatencyForTest() {
	mentorLatencyMu.Lock()
	mentorLatencySamp = nil
	mentorLatencyMu.Unlock()
}

// MentorLatencyBootLine is the boot line for the close→ack counter: n/a until
// the first sample exists (canon L7 — a value the process cannot know yet
// prints n/a, never a fabricated 0).
func MentorLatencyBootLine() string {
	n, p50, p95 := MentorLatencySnapshot()
	if n == 0 {
		return "🧑‍🏫 mentor eval latency (close→ack): n=0 p50=n/a p95=n/a (target <1s, WARN >3s)"
	}
	return fmt.Sprintf("🧑‍🏫 mentor eval latency (close→ack): n=%d p50=%dms p95=%dms (target <1s, WARN >3s)", n, p50, p95)
}
