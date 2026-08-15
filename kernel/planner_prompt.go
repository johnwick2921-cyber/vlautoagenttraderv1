package kernel

import (
	"fmt"
	"strings"
)

// P3.3 — the planner input package (assembled into ONE prompt for the reasoner).
// Pure + deterministic. The 16:55 closed-market read builds entirely from stored
// data (no live feed), which is what makes it a first-class tested path.

// PlannerCalendarEvent is a session-sliced calendar entry (kernel-local to avoid
// importing the calendar package; the trader maps calendar.Event → this).
type PlannerCalendarEvent struct {
	TimeCT   string // "HH:MM" CT
	Currency string
	Title    string
	Impact   string // T1 | T2
}

// PlannerInput is everything the reasoner reads to write a plan.
type PlannerInput struct {
	TradeDate        string
	Session          string // NY | ASIA | LONDON
	ReadKind         string // e.g. "closed-market 16:55 CT read (from stored data)"
	Price            float64
	DATR             float64
	Regime           RegimeBlock
	Levels           []ScoredLevel // Go-ranked, graded (P1.5) — the decision-critical block
	StructureSummary []string      // one line per timeframe
	OvernightStory   string
	PriorDayStory    string
	Calendar         []PlannerCalendarEvent // session-sliced (P1.8)
	DigestChain      []string               // session digests + dailies + one-liners
	OwnerNote        string
	Warming          string // non-empty → cold-start / WARMING annotation
	// W11 — the executor's indicator mirror (per-TF EMA/RSI/ATR/BOLL/MACD, driven by
	// ai_config toggles), rendered once by RenderPlannerIndicatorBlock. Empty → the
	// block is omitted (disabled state = byte-identical prompt). AIConfigHash is the
	// fingerprint of the indicator config that produced it (frozen on the plan row).
	IndicatorsBlock string
	AIConfigHash    string
}

// BuildPlannerPrompt assembles the planner prompt: reasoning-first instruction,
// the input tables, and the schema-strict JSON contract. High-salience blocks
// (ranked levels, T1 blackouts) are positioned prominently per the contract.
func BuildPlannerPrompt(in PlannerInput) string {
	var b strings.Builder

	b.WriteString("# DAY-PLAN READER — CME MNQ futures\n")
	b.WriteString("You are a disciplined pro reading the market ONCE, carefully. Write ONE structured day plan for this session. Facts below are Go-computed; your job is JUDGMENT, not re-deriving the map.\n\n")

	fmt.Fprintf(&b, "## Session\ntrade_date %s · session %s · %s · price %.2f · dATR %.1f\n",
		in.TradeDate, in.Session, in.ReadKind, in.Price, in.DATR)
	if in.Warming != "" {
		fmt.Fprintf(&b, "WARMING: %s (first-week honesty — narrate the machinery, not an edge).\n", in.Warming)
	}
	b.WriteString("\n")

	b.WriteString("## Regime\n" + in.Regime.Render() + "\n\n")

	// W11 — INDICATORS mirror: the SAME per-timeframe indicator state the executor
	// sees, so the planner reasons on the exact indicators the trader trades on.
	// Omitted when no indicator is enabled (disabled state = pre-W11 prompt).
	if in.IndicatorsBlock != "" {
		b.WriteString("## Indicators (executor mirror — your ai_config toggles, computed values)\n")
		b.WriteString(in.IndicatorsBlock + "\n\n")
	}

	// Ranked level table — the decision-critical block, high-salience.
	b.WriteString("## Ranked levels (Go-graded; you never re-sort)\n")
	if len(in.Levels) == 0 {
		b.WriteString("(none in range — warming forward)\n")
	} else {
		for _, l := range in.Levels {
			sign := "+"
			if l.Distance < 0 {
				sign = "-"
			}
			fmt.Fprintf(&b, "  %-9.2f %-14s grade %s  %-8s %s%.1f\n", l.Price, l.Label, l.Grade, l.Fresh, sign, absF(l.Distance))
		}
	}
	b.WriteString("\n")

	if len(in.StructureSummary) > 0 {
		b.WriteString("## Structure (1 line/TF)\n")
		for _, s := range in.StructureSummary {
			b.WriteString("  " + s + "\n")
		}
		b.WriteString("\n")
	}

	if in.OvernightStory != "" || in.PriorDayStory != "" {
		b.WriteString("## Auction story\n")
		if in.PriorDayStory != "" {
			b.WriteString("  prior day: " + in.PriorDayStory + "\n")
		}
		if in.OvernightStory != "" {
			b.WriteString("  overnight: " + in.OvernightStory + "\n")
		}
		b.WriteString("\n")
	}

	// Session-sliced calendar: T1 = HARD blackout, T2 = caution.
	b.WriteString("## Calendar (this session's window)\n")
	if len(in.Calendar) == 0 {
		b.WriteString("  (no filtered events)\n")
	} else {
		for _, e := range in.Calendar {
			tag := "caution"
			if e.Impact == "T1" {
				tag = "HARD no-trade blackout"
			}
			fmt.Fprintf(&b, "  %s %s %s — %s (%s)\n", e.TimeCT, e.Currency, e.Impact, e.Title, tag)
		}
	}
	b.WriteString("\n")

	if len(in.DigestChain) > 0 {
		b.WriteString("## Recent context (digest chain)\n")
		for _, d := range in.DigestChain {
			b.WriteString("  " + d + "\n")
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(in.OwnerNote) != "" {
		b.WriteString("## Owner note\n  " + in.OwnerNote + "\n\n")
	}

	b.WriteString(plannerOutputContract)
	return b.String()
}

// plannerOutputContract is the schema-strict, reasoning-first output spec.
const plannerOutputContract = "## OUTPUT — one JSON object, reasoning FIRST, no prose outside it\n" +
	"{\n" +
	`  "reasoning": "<your read: what the auction is doing and why this plan>",` + "\n" +
	`  "bias": {"direction": "long|short|neutral", "conviction": "high|medium|low", "flip_condition": "<explicit>"},` + "\n" +
	`  "levels": [{"price": <n>, "label": "<PDH|ONH|nPOC…>", "grade": "A|B|C", "instruction": "<verb>"}],  // max 8` + "\n" +
	`  "scenarios": [{"id": "S1", "trigger": "<setup>", "condition": "reclaim|hold|sweep_reclaim|reject|acceptance|breakout_retest", "direction": "long|short", "target_chain": [<n>,…], "invalid": "<line>", "quality": "A+|A|B"}],  // 1..3` + "\n" +
	`  "no_trade": ["first 5m", "12:00-13:30 lunch", "<calendar blackouts>"],` + "\n" +
	`  "death_condition": "<the single line that invalidates this whole plan>",` + "\n" +
	`  "day_type": "trend|balance|<optional>"` + "\n" +
	"}\n" +
	"Rules: levels chosen ONLY from the ranked table above; S/D & FVG are confluence, never standalone. Respect the no-trade windows. If you cannot form a credible plan, say so in reasoning and output a neutral/no-trade plan.\n"

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
