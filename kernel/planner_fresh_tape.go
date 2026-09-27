package kernel

import (
	"fmt"
	"strings"
	"time"

	"nofx/market"
)

// A6 bounds — the fresh tape carried by attempt N+1 after a born-dead /
// flip-met refusal is the COMPLETED bars between the read clock and the
// refusal: at most the last 30 completed 1m closes and the last 6 completed
// 5m closes. The forming bar is never included — its close does not exist yet,
// and printing it would be a fabricated value (checklist law 49).
const (
	PlannerFreshTape1mWindowMs = 30 * 60_000
	PlannerFreshTape5mCount    = 6
)

// PlannerFreshTape renders the A6 repair block for attempt N+1 after a
// born-dead / flip-met refusal (PLANNER A6, 2026-09-25): the validator judged
// the previous attempt against bars that arrived while the model read, so the
// re-author gets the tape that exists NOW — the completed closes between the
// read clock and the refusal, plus the breached condition verbatim. The
// born-dead check itself is NOT relaxed; this block only re-sights the model.
//
// bars1m is the provider tape (1m). The 5m rows are aggregated from it with
// the same bucket math the born check's tape reads, so the block can never
// disagree with what the validator judged.
func PlannerFreshTape(bars1m []market.Kline, read, publish time.Time, reason string) string {
	var b strings.Builder
	b.WriteString("## FRESH TAPE SINCE YOUR READ (previous attempt refused born-dead / flip-met)\n\n")
	b.WriteString("The validator refused your previous attempt because the market moved during the read:\n")
	b.WriteString(reason)
	b.WriteString("\n\n")
	if publish.IsZero() {
		publish = time.Now()
	}
	pubMs := publish.UnixMilli()
	windowFrom := pubMs - PlannerFreshTape1mWindowMs
	completed := make([]market.Kline, 0, 32)
	for _, k := range bars1m {
		if k.CloseTime <= 0 {
			continue
		}
		// Only closes strictly between the read clock and the refusal, and
		// only within the bounded window. The forming bar (close time in the
		// future) is excluded by construction.
		if k.CloseTime <= read.UnixMilli() || k.CloseTime > pubMs {
			continue
		}
		if k.OpenTime < windowFrom {
			continue
		}
		completed = append(completed, k)
	}
	b.WriteString(fmt.Sprintf(
		"Completed 1m closes between the read clock (%s) and the refusal (%s), newest last:\n",
		FormatCT(read), FormatCT(publish)))
	if len(completed) == 0 {
		b.WriteString("(none — no completed 1m closes in the window)\n")
	} else {
		for _, k := range completed {
			b.WriteString(fmt.Sprintf("%s 1m close %.2f\n",
				FormatCT(time.UnixMilli(k.CloseTime)), k.Close))
		}
	}

	five := AggregateBars(bars1m, 5*60_000)
	fiveRows := make([]market.Kline, 0, PlannerFreshTape5mCount)
	for _, k := range five {
		closeMs := k.OpenTime + 5*60_000
		// The bucket must be closed inside (read, publish] — a bucket still
		// forming at the refusal is not a close yet.
		if closeMs <= read.UnixMilli() || closeMs > pubMs {
			continue
		}
		fiveRows = append(fiveRows, market.Kline{OpenTime: k.OpenTime, Close: k.Close})
	}
	if len(fiveRows) > PlannerFreshTape5mCount {
		fiveRows = fiveRows[len(fiveRows)-PlannerFreshTape5mCount:]
	}
	b.WriteString(fmt.Sprintf("Completed 5m closes (last %d, newest last):\n", PlannerFreshTape5mCount))
	if len(fiveRows) == 0 {
		b.WriteString("(none — no 5m bucket closed in the window)\n")
	} else {
		for _, k := range fiveRows {
			b.WriteString(fmt.Sprintf("%s 5m close %.2f\n",
				FormatCT(time.UnixMilli(k.OpenTime+5*60_000)), k.Close))
		}
	}
	b.WriteString("\nRe-author against THIS tape and the current price facts. Do not reuse the read-time levels or conditions — the validator judged them against bars that no longer exist. The born-dead check itself is unchanged: a plan whose lines are already crossed at publication is still refused.\n")
	return b.String()
}
