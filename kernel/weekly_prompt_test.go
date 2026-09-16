package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ── fixture helpers ────────────────────────────────────────────────────────

func wkDoc() *WeeklyDoc {
	return &WeeklyDoc{
		WeeklyLevels: []WeeklyLevel{{Name: "PWH", Px: 30500.25}, {Name: "PWL", Px: 29980.00}, {Name: "IPDA-20d-high", Px: 30600.00}},
		Narrative:    "PWH and PWL bracket the accepted range\nIPDA 20d high is the first upper reference",
	}
}

func refsForDoc() []float64 { return []float64{30500.25, 29980.00, 30600.00, 29900.00} }

func weeklyPxBar(openTimeMs int64, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: openTimeMs, Open: o, High: h, Low: l, Close: c, Volume: 1}
}

// ── validator r1-r4 (refs-only, class 50) ──────────────────────────────────

func TestWeeklyValidatorRefsOnly(t *testing.T) {
	d := wkDoc()
	if got := ValidateWeeklyDoc(d, refsForDoc(), false); got != "" {
		t.Fatalf("valid refs-only doc rejected: %s", got)
	}
	d2 := wkDoc()
	d2.WeeklyLevels = nil
	if got := ValidateWeeklyDoc(d2, refsForDoc(), false); !strings.Contains(got, "r1") {
		t.Fatalf("proving line: r1 empty weekly_levels — got %q", got)
	}
	d3 := wkDoc()
	d3.WeeklyLevels[0].Px = 0
	if got := ValidateWeeklyDoc(d3, refsForDoc(), false); !strings.Contains(got, "r1") {
		t.Fatalf("proving line: r1 px>0 — got %q", got)
	}
	// A legacy directional doc (pre-wave shape) is STILL ACCEPTED — the schema
	// tolerates stored rows; the direction is simply never read.
	legacy := &WeeklyDoc{Bias: "bull", Conviction: "high", Draw: WeeklyDraw{Name: "PWH", Px: 30500.25},
		Invalidation: WeeklyInvalidation{Px: 30300, Basis: "1h close beyond 30300.00"},
		WeeklyLevels: []WeeklyLevel{{Name: "PWH", Px: 30500.25}, {Name: "PWL", Px: 29980}},
		Narrative:    "facts only"}
	if got := ValidateWeeklyDoc(legacy, refsForDoc(), false); got != "" {
		t.Fatalf("proving line: legacy-shaped doc parses — got %q", got)
	}
}

func TestWeeklyValidatorR2NarrativeLines(t *testing.T) {
	d := wkDoc()
	d.Narrative = "l1\nl2\nl3\nl4"
	if got := ValidateWeeklyDoc(d, refsForDoc(), false); !strings.Contains(got, "r2") {
		t.Fatalf("proving line: r2 narrative >3 lines — got %q", got)
	}
}

func TestWeeklyValidatorR3DayOfWeekTokens(t *testing.T) {
	d := wkDoc()
	d.Narrative = "Monday seasonality favors longs"
	if got := ValidateWeeklyDoc(d, refsForDoc(), false); !strings.Contains(got, "r3") {
		t.Fatalf("proving line: r3 day-of-week token — got %q", got)
	}
}

func TestWeeklyValidatorR4DirectionalCallForbidden(t *testing.T) {
	d := wkDoc()
	d.Narrative = "price should trend higher — bullish above PWH"
	if got := ValidateWeeklyDoc(d, refsForDoc(), false); !strings.Contains(got, "r4") {
		t.Fatalf("proving line: r4 directional token — got %q", got)
	}
	// thin history no longer interacts with conviction — no conviction exists.
	d2 := wkDoc()
	if got := ValidateWeeklyDoc(d2, refsForDoc(), true); got != "" {
		t.Fatalf("proving line: thin history + refs-only doc accepted — got %q", got)
	}
}

// ── prompt sections + facts hash ───────────────────────────────────────────

func TestWeeklyPromptSectionsAndHash(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 45, 0, 0, CTLocation())
	f := WeeklyFacts{
		Now: now, Price: 30200.5,
		Weeks: []WeekCandle{
			{WeekStart: "2026-08-03", Open: 29800, High: 30200, Low: 29700, Close: 30150, Volume: 1000, StructTag: "HH"},
			{WeekStart: "2026-08-10", Open: 30150, High: 30400, Low: 29900, Close: 30300, Volume: 1100, StructTag: "HH"},
		},
		Refs: WeeklyRefs{WeeklyOpen: 30120, PWH: 30400, PWL: 29900, PWC: 30300}, RefsOK: true,
		NWOGs:       []NWOG{{Born: "2026-08-30", Hi: 30310, Lo: 30290, CE: 30300, Filled: false}},
		IPDA:        []IPDARange{{Days: 20, High: 30600, Low: 29500, PosPct: 0.63}},
		ThinHistory: false,
	}
	f.SectionsText = RenderWeeklyFactsSections(f)
	f.FactsHash = WeeklyFactsHash(f)

	for _, want := range []string{"## Weekly candles (12 completed weeks, oldest → latest)",
		"## Weekly references", "## NWOG (last 5 weekend gaps, oldest → latest)",
		"## IPDA (trailing dealing ranges)", "## Prior week recap (facts only)",
		"weekly_open 30120.00 · PWH 30400.00 · PWL 29900.00 · PWC 30300.00",
		"2026-08-03  29800.00  30200.00  29700.00  30150.00  1000  HH"} {
		if !strings.Contains(f.SectionsText, want) {
			t.Fatalf("proving line: facts sections missing %q", want)
		}
	}
	if f.FactsHash == "" || f.FactsHash != WeeklyFactsHash(f) {
		t.Fatalf("proving line: facts hash must be deterministic sha256 hex — %q", f.FactsHash)
	}
	prompt := BuildWeeklyPrompt(f)
	for _, want := range []string{"## Instructions (THE RULES — violations are rejected)",
		"weekly_levels", "REFS ONLY", `{"weekly_levels":[{"name":"<ref name>","px":<n>}],"narrative":"≤3 lines, facts only"}`,
		"Day-of-week reasoning is FORBIDDEN"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("proving line: weekly prompt missing %q", want)
		}
	}
}

// ── W4 invalidation twins ──────────────────────────────────────────────────

func TestWeeklyInvalidationTwins(t *testing.T) {
	base := int64(1725000000000)
	// bull doc invalidates on a CLOSED bar below px.
	bars := []market.Kline{
		weeklyPxBar(base, 30200, 30210, 30190, 30205),
		weeklyPxBar(base+60000, 30205, 30215, 30180, 30180), // closes below 30200
	}
	if !WeeklyInvalidationCrossed("bull", 30200.0, bars) {
		t.Fatalf("proving line: bull invalidation high — close 30180 < 30200 must cross")
	}
	// bear doc invalidates on a CLOSED bar above px.
	bars = []market.Kline{
		weeklyPxBar(base, 30100, 30110, 30090, 30105),
		weeklyPxBar(base+60000, 30105, 30140, 30100, 30140), // closes above 30120
	}
	if !WeeklyInvalidationCrossed("bear", 30120.0, bars) {
		t.Fatalf("proving line: bear invalidation low — close 30140 > 30120 must cross")
	}
	// the mirror directions do NOT cross: bull with closes ABOVE px.
	if WeeklyInvalidationCrossed("bull", 30080.0, bars) {
		t.Fatalf("proving line: bull doc must not cross on closes ABOVE px")
	}
	// neutral / px≤0 never crosses.
	if WeeklyInvalidationCrossed("neutral", 30200.0, bars) || WeeklyInvalidationCrossed("bull", 0, bars) {
		t.Fatalf("proving line: neutral/zero-px never cross")
	}
}

// ── W3 injection render — refs-only (class 50) ─────────────────────────────

func TestWeeklyContextLineRefsOnly(t *testing.T) {
	if got := WeeklyContextLine(nil, 0); got != "WEEKLY: none" {
		t.Fatalf("proving line: no-doc state — got %q", got)
	}
	d := wkDoc()
	if got := WeeklyContextLine(d, 0); got != "WEEKLY: refs only — PWH 30500.25 · PWL 29980.00" {
		t.Fatalf("proving line: refs-only state — got %q", got)
	}
	// A legacy directional doc renders refs only too — the direction is ignored.
	legacy := &WeeklyDoc{Bias: "bull", Conviction: "high", Draw: WeeklyDraw{Name: "PWH", Px: 30500.25},
		Invalidation: WeeklyInvalidation{Px: 30300, Basis: "1h close beyond 30300.00"}, InvalidatedAt: "2026-08-28 10:15 CT",
		WeeklyLevels: []WeeklyLevel{{Name: "PWH", Px: 30500.25}, {Name: "PWL", Px: 29980}}}
	if got := WeeklyContextLine(legacy, 0); got != "WEEKLY: refs only — PWH 30500.25 · PWL 29980.00" {
		t.Fatalf("proving line: legacy doc renders refs-only — got %q", got)
	}
	thin := wkDoc()
	thin.ThinHistory = true
	if got := WeeklyContextLine(thin, 3); got != "WEEKLY: refs only — PWH 30500.25 · PWL 29980.00 (thin history 3w)" {
		t.Fatalf("proving line: thin-history state — got %q", got)
	}
	if got := WeeklyExecutorLine(d); got != "WEEKLY: refs only — PWH 30500.25 · PWL 29980.00" {
		t.Fatalf("proving line: executor line — got %q", got)
	}
	if got := WeeklyExecutorLine(nil); got != "" {
		t.Fatalf("nil doc line = %q, want empty", got)
	}
}

// TestWeeklyChipsNeverCarryDirection (class 50) — THE LAW pinned: no rendered
// weekly line may contain a directional token, whatever the stored doc says.
func TestWeeklyChipsNeverCarryDirection(t *testing.T) {
	legacy := &WeeklyDoc{Bias: "bear", Conviction: "high", Draw: WeeklyDraw{Name: "PWL", Px: 28947.75},
		Invalidation: WeeklyInvalidation{Px: 29535, Basis: "1h close beyond 29535.00"}, InvalidatedAt: "2026-08-30 17:07 CT",
		WeeklyLevels: []WeeklyLevel{{Name: "PWH", Px: 29900}, {Name: "PWL", Px: 28947.75}}}
	for _, line := range []string{WeeklyContextLine(legacy, 0), WeeklyExecutorLine(legacy)} {
		for _, tok := range []string{"bull", "bear", "long", "short", "invalidated"} {
			if strings.Contains(strings.ToLower(line), tok) {
				t.Fatalf("proving line: weekly line %q carries directional token %q", line, tok)
			}
		}
	}
}

// TestWeeklyRuleBiasShadow — the deterministic rule survives as SHADOW (class
// 50): it computes a direction but nothing renders it.
func TestWeeklyRuleBiasShadow(t *testing.T) {
	f := WeeklyFacts{Refs: WeeklyRefs{WeeklyOpen: 30000, PWH: 30400, PWL: 29600},
		Weeks: []WeekCandle{{WeekStart: "2026-08-24", Open: 30050, High: 30500, Low: 29900, Close: 30450}}}
	if bias, why := WeeklyRuleBias(f); bias != "bull" || why == "" {
		t.Fatalf("bull case: %q %q", bias, why)
	}
	f.Weeks[0].Close = 29500
	if bias, _ := WeeklyRuleBias(f); bias != "bear" {
		t.Fatalf("bear case: %q", bias)
	}
	f.Weeks[0].Close = 30100
	if bias, _ := WeeklyRuleBias(f); bias != "neutral" {
		t.Fatalf("neutral case: %q", bias)
	}
	// The shadow result is NOT part of any rendered line.
	d := wkDoc()
	d.ShadowBias, d.ShadowWhy = "bull", "would have been bull"
	if got := WeeklyContextLine(d, 0); strings.Contains(got, "bull") {
		t.Fatalf("shadow bias leaked into the chip: %q", got)
	}
}

// TestApplyWeeklyDOAStampsNeutralAtWrite — F5 (2026-08-30): a bias whose own
// invalidation basis is ALREADY crossed at write is stamped neutral + stamp
// time, never written stillborn (the 17:07:15 bear lived 3 seconds).
func TestApplyWeeklyDOAStampsNeutralAtWrite(t *testing.T) {
	now := time.Date(2026, 8, 30, 22, 7, 15, 0, time.UTC)
	doc := &WeeklyDoc{Bias: "bear", Conviction: "low", Invalidation: WeeklyInvalidation{Px: 29535, Basis: "1h close beyond 29535.00"}}
	breached := []market.Kline{{OpenTime: 1, Close: 29541.75}} // 1h close beyond 29535
	if !ApplyWeeklyDOA(doc, breached, now) {
		t.Fatal("DOA should stamp neutral on a breach-at-write")
	}
	if doc.Bias != "neutral" || doc.InvalidatedAt == "" {
		t.Fatalf("doc = bias %q invalidated_at %q, want neutral + stamp", doc.Bias, doc.InvalidatedAt)
	}
	// Not crossed → untouched; already invalidated → untouched; neutral bias → untouched.
	doc2 := &WeeklyDoc{Bias: "bull", Invalidation: WeeklyInvalidation{Px: 100, Basis: "1h close below 100.00"}}
	if ApplyWeeklyDOA(doc2, []market.Kline{{OpenTime: 1, Close: 101}}, now) {
		t.Fatal("no breach → no DOA stamp")
	}
	if ApplyWeeklyDOA(doc, breached, now) { // already invalidated
		t.Fatal("already-invalidated doc must not re-stamp")
	}
	doc3 := &WeeklyDoc{Bias: "neutral", Invalidation: WeeklyInvalidation{Px: 100}}
	if ApplyWeeklyDOA(doc3, []market.Kline{{OpenTime: 1, Close: 50}}, now) {
		t.Fatal("neutral bias → no DOA stamp")
	}
}
