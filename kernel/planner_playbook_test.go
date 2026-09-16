package kernel

import (
	"strings"
	"testing"
	"time"
)

// A1-A5 (planner-contract wave 2026-08-26) — render tests for the new prompt
// sections + the chain_after WARN validator. Advisory only: WARN, never fail.

func TestBuildPlannerPromptPlaybookSections(t *testing.T) {
	in := samplePlannerInput()
	in.BiasCtxFacts = BiasContext{Price: 15600, PDC: 15550, VAH: 15650, VAL: 15400, NearestLiquidity: "ONH (+40.0)"}
	p := BuildPlannerPrompt(in)
	for _, want := range []string{
		"## BIAS-TREE", "bull-continuation", "premium/discount", "draw-on-liquidity",
		"## Priority setup — THE CHAIN", "chain_after", "raw-FVG", "displacement → FVG retrace",
		// "10:30 ET" → "09:30 CT": one-clock ruling 2026-09-03, every time in
		// the prompt is CT and three lines were printing Eastern.
		"## No-trade gates", "balance-day", "1.2×ATR", "09:30 CT", "Tier-1 news",
		"## Killzone weighting", "premium FVG window", "Thursday/Friday",
		"## STOP-DOING", "0% win evidence",
		`"chain_after"`, "bias-tree",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("planner prompt missing %q", want)
		}
	}
	// No-trade gates must stay ≤8 lines (6 bullets + header + blank = 8).
	block := p[strings.Index(p, "## No-trade gates"):]
	if i := strings.Index(block, "\n## "); i > 0 {
		block = block[:i]
	}
	if got := strings.Count(block, "\n"); got > 8 {
		t.Fatalf("no-trade gates block is %d lines (>8)", got)
	}
}

func TestRenderBiasTreeBranches(t *testing.T) {
	levels := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindPDH, Price: 15620, Label: "PDH"}, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindPDL, Price: 15450, Label: "PDL"}, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindPDC, Price: 15550, Label: "PDC"}, Grade: "A"},
	}
	// Branch 1: close > PDH.
	out := RenderBiasTree(15640, levels, BiasContext{Price: 15640})
	if !strings.Contains(out, "facts match branch 1 (close > PDH)") {
		t.Fatalf("branch 1 not flagged:\n%s", out)
	}
	// Branch 3: inside day, close above PDC → long LOW.
	out = RenderBiasTree(15580, levels, BiasContext{Price: 15580})
	if !strings.Contains(out, "facts match branch 3 (inside day; close long PDC → long LOW)") {
		t.Fatalf("branch 3 not flagged:\n%s", out)
	}
	// Premium: price above the 50% mark of the value range.
	out = RenderBiasTree(15600, levels, BiasContext{Price: 15600, VAH: 15650, VAL: 15400})
	if !strings.Contains(out, "PREMIUM — longs disallowed by branch 5") {
		t.Fatalf("premium not flagged:\n%s", out)
	}
	// Discount: below the 50% mark of the dealing range (15450–15620).
	out = RenderBiasTree(15480, levels, BiasContext{Price: 15480, VAH: 15650, VAL: 15400})
	if !strings.Contains(out, "DISCOUNT — shorts disallowed by branch 5") {
		t.Fatalf("discount not flagged:\n%s", out)
	}
	// S-dispatch (2026-08-27) — price BELOW the dealing range's low is no
	// longer a >100% percentage; it renders the extended clamp.
	out = RenderBiasTree(15420, levels, BiasContext{Price: 15420, VAH: 15650, VAL: 15400})
	if !strings.Contains(out, "BEYOND range low (extended)") {
		t.Fatalf("beyond-range-low must render the extended label:\n%s", out)
	}
	if strings.Contains(out, "% of the range") && strings.Contains(out, "price at 1") {
		t.Fatalf("a beyond-range price must not render a >100%% percentage:\n%s", out)
	}
}

// S-dispatch (2026-08-27) — post-roll fixture: the prior-day anchors sit
// OUTSIDE the proximity band (the 17:46/19:02 ASIA reads rendered "no PDH/PDL
// anchor"). The universe stamping must resolve them anyway and the tree must
// use them — at any distance.
func TestBiasTreeDayAnchorsResolvePostRoll(t *testing.T) {
	levels := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindPDH, Price: 30254, Label: "PDH"}, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindPDL, Price: 29700, Label: "PDL"}, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindPDC, Price: 30000, Label: "PDC"}, Grade: "A"},
	}
	// Seated table (what the planner prompt carries): NO PDH/PDL rows (they
	// lost the seat race to in-band levels).
	seated := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindVWAP, Price: 29586, Label: "VWAP"}, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindONH, Price: 29629, Label: "ONH"}, Grade: "A"},
	}
	bc := ComputeBiasContext(nil, seated, time.Now())
	if bc.PDH != 0 || bc.PDL != 0 {
		t.Fatalf("seated-only facts must not invent anchors: PDH=%v PDL=%v", bc.PDH, bc.PDL)
	}
	// The universe re-derivation stamps them (price 29614, below prior low).
	all := make([]DetectedLevel, 0, len(levels))
	for _, l := range levels {
		all = append(all, l.DetectedLevel)
	}
	ApplyUniverseDayAnchors(&bc, all)
	if bc.PDH != 30254 || bc.PDL != 29700 || bc.PDC != 30000 {
		t.Fatalf("universe anchors not stamped: PDH=%v PDL=%v PDC=%v", bc.PDH, bc.PDL, bc.PDC)
	}
	out := RenderBiasTree(29614, seated, bc)
	for _, want := range []string{
		"computed: PDH 30254.00 · PDL 29700.00 · PDC 30000.00",
		"dealing range 29700.00–30254.00",
		"BEYOND range low (extended)",
		"facts match branch 1 mirror (close < PDL)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("post-roll tree missing %q:\n%s", want, out)
		}
	}
}

// S-dispatch (2026-08-27) — extended-price render: a price ABOVE the dealing
// range high must clamp to "beyond range (extended)", never a >100% figure.
func TestBiasTreeBeyondRangeHighClamped(t *testing.T) {
	bc := BiasContext{Price: 30500, PDH: 30254, PDL: 29700, PDC: 30000}
	out := RenderBiasTree(30500, nil, bc)
	if !strings.Contains(out, "BEYOND range high (extended)") {
		t.Fatalf("beyond-range-high must render the extended label:\n%s", out)
	}
	if strings.Contains(out, "price at 1") || strings.Contains(out, "price at 2") {
		t.Fatalf("a beyond-range price must not render a >100%% percentage:\n%s", out)
	}
}

func TestChainWarnings(t *testing.T) {
	doc := PlanDoc{
		Levels: []PlanLevel{{Price: 15620, Label: "PDH", Grade: "A"}, {Price: 15500, Label: "RN", Grade: "C"}},
		Scenarios: []PlanScenario{
			{ID: "S1", Condition: "sweep_reclaim", Direction: "long"},
			{ID: "S2", Condition: "fvg_entry", Direction: "long",
				Fvg: &PlanFvgEntry{Lo: 15510, Hi: 15530, OriginLevel: "RN", Direction: "long"}},
		},
	}
	// No chain_after + origin C → WARN.
	w := ChainWarnings(doc)
	if len(w) != 1 || !strings.Contains(w[0], "no sweep_reclaim precursor") {
		t.Fatalf("expected 1 chain warning, got %v", w)
	}
	// chain_after → sweep_reclaim → clean.
	doc.Scenarios[1].ChainAfter = "S1"
	if w := ChainWarnings(doc); len(w) != 0 {
		t.Fatalf("chained fvg must not warn, got %v", w)
	}
	// No chain but A-grade origin → clean.
	doc.Scenarios[1].ChainAfter = ""
	doc.Scenarios[1].Fvg.OriginLevel = "PDH"
	if w := ChainWarnings(doc); len(w) != 0 {
		t.Fatalf("A-origin fvg must not warn, got %v", w)
	}
}
