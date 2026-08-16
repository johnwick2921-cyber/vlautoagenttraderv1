package api

import (
	"os"
	"strings"
	"time"

	"nofx/market"
)

// SANDBOX MODE — an isolated demo instance (its own DB, its own port, no NT8
// wire, no orders, canned planner replies). Everything here is env-gated and
// default-OFF, so the live bot is byte-identical unless SANDBOX_MODE=1.

// SandboxMode reports whether this process is a sandbox.
func SandboxMode() bool {
	v := strings.TrimSpace(os.Getenv("SANDBOX_MODE"))
	return v == "1" || strings.EqualFold(v, "true")
}

// SandboxRealLLM reports whether the sandbox should make REAL (paid) planner
// calls instead of canned replies. Opt-in via SANDBOX_LLM=real.
func SandboxRealLLM() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("SANDBOX_LLM")), "real")
}

// sandboxPlannerRaw returns a canned but realistic planner reply in the REAL wire
// format (it must parse through kernel.ParsePlannerReply like any live reply), so
// the owner can exercise Apply / Keep-as-is at zero cost.
//
// Which reply comes back is chosen from the request so the owner can reach all
// three outcomes deliberately:
//   - a question with no numbers/evidence words → DEFEND (bare disagreement)
//   - an edit-level / delete change             → DEFEND (no plan change needed)
//   - anything else (add-level, bulk-add, ...)  → PROPOSE-MERGE with a patch
func sandboxPlannerRaw(userPrompt string) string {
	lower := strings.ToLower(userPrompt)
	bare := !strings.ContainsAny(lower, "0123456789") ||
		strings.Contains(lower, "are you sure") || strings.Contains(lower, "u sure")
	noChange := strings.Contains(lower, "change_kind: edit-level") ||
		strings.Contains(lower, "change_kind: delete-level")

	switch {
	case bare:
		return `{"evidence":"Bias long rests on: price above EMA200 daily, ONL swept and reclaimed 07:40, and the demand shelf at 30160-52 holding on two tests. The flip line (2x5m < 30148) is where I would be wrong.",
"point_class":"BARE-DISAGREEMENT",
"verdict":"DEFEND",
"summary":"No new information here - this is a confidence check, not evidence. Holding the plan: the flip line is already the disagreement point, and price has not touched it."}`
	case noChange:
		return `{"evidence":"Your change stays inside S1's existing trigger band (30160-52) and does not move the demand shelf, the bias, or the flip line (2x5m < 30148).",
"point_class":"NEW-INFO",
"verdict":"DEFEND",
"summary":"Your level fits S1 as it stands - no plan change needed. I am leaving bias, scenarios and targets untouched."}`
	default:
		return `{"evidence":"S1 triggers on a sweep+reclaim of 30160-52; the level you added sits just under that band and shares its 4h origin. The long bias rests on the same shelf, and the flip line (2x5m < 30148) sits below your zone.",
"point_class":"NEW-INFO",
"verdict":"PROPOSE-MERGE",
"summary":"Your zone does not need a new play - it EXTENDS S1's trigger band and sharpens the invalidation. Widening S1's trigger and re-pointing its first target; bias and flip line stay as they are.",
"patch":[{"op":"replace","path":"/scenarios/0/trigger","value":"sweep 30160-30156 (4h OB) then reclaim"},
         {"op":"replace","path":"/scenarios/0/quality","value":"A+"},
         {"op":"replace","path":"/levels/2/instruction","value":"magnet - expect touch; no entries at it (S1 target 1)"}]}`
	}
}

// InstallSandboxBars gives a sandbox process a deterministic synthetic bar feed.
//
// WHY: level_facts, the mini-chart, the live price and the B2 price-armor are all
// computed from the NT8 BarCache at request time. A sandbox has no NT8 wire, so
// without this the plan card renders an EMPTY levels table, no chart, and the armor
// silently accepts a fat-fingered price (nothing to compare against). The synthetic
// series walks around anchor with realistic 1m structure so every panel has content.
//
// Called ONLY when SANDBOX_MODE=1 — the live bot never reaches this.
func InstallSandboxBars(symbol string, anchor float64) {
	market.FuturesBarsProvider = func(sym, tf string, count int) []market.Kline {
		if count <= 0 || count > 4000 {
			count = 400
		}
		step := int64(60_000) // 1m
		switch tf {
		case "5m":
			step = 300_000
		case "15m":
			step = 900_000
		case "1h":
			step = 3_600_000
		case "4h":
			step = 14_400_000
		case "1d":
			step = 86_400_000
		}
		end := time.Now().UnixMilli()
		out := make([]market.Kline, 0, count)
		px := anchor
		for i := count - 1; i >= 0; i-- {
			t := end - int64(i)*step
			// deterministic pseudo-random walk: no rand, so a reset reproduces it
			n := float64((t/step)%97) - 48
			px = anchor + n*0.75
			hi := px + 3.25
			lo := px - 3.25
			op := px - 1.0
			out = append(out, market.Kline{
				OpenTime: t, Open: op, High: hi, Low: lo, Close: px,
				Volume: 850 + float64((t/step)%140), CloseTime: t + step - 1,
			})
		}
		return out
	}
}
