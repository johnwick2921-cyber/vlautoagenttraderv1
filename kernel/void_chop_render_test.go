package kernel

import (
	"strings"
	"testing"
)

// ── VOID CHOP RENDER (owner ruling 2026-09-03) ───────────────────────────────
//
// Measured on the first live read-facts row (00:00:56 CT, ASIA): void=18 across
// NINE distinct levels, and ALL NINE were void on BOTH sides. Eighteen lines to
// say "these nine levels are chop". The session-day window (class 53) bounded
// the lookback but not the crossing frequency, so it could not fix this — the
// doubling comes from intra-session oscillation.
//
// Both-sided levels now collapse into ONE line; one-sided voids keep their own
// line with side and reclaim time, because those carry real directional news.
func TestVoidChopCollapsesBothSidedLevels(t *testing.T) {
	v := []VoidBreakdownLevel{
		{Price: 29141.25, Short: true, ReclaimedAtCT: "2026-09-02 17:17 CT"},
		{Price: 29141.25, Short: false, ReclaimedAtCT: "2026-09-02 17:18 CT"},
		{Price: 29159.02, Short: true, ReclaimedAtCT: "2026-09-02 17:21 CT"},
		{Price: 29159.02, Short: false, ReclaimedAtCT: "2026-09-02 17:30 CT"},
		{Price: 29193.38, Short: true, ReclaimedAtCT: "2026-09-02 18:47 CT"}, // one-sided
	}
	out := RenderVoidBreakdownLevels(v, 12)

	if !strings.Contains(out, "CHOP (broken and reclaimed both ways this session)") {
		t.Fatalf("both-sided levels must collapse into a CHOP line:\n%s", out)
	}
	// The two chop levels ride ONE line; the one-sided level does not.
	var chopLine string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "CHOP (") {
			chopLine = l
		}
	}
	for _, want := range []string{"29141.25", "29159.02", "(2 of 12 seated)", "waterfall plays at these will be refused", "prefer touch/fade"} {
		if !strings.Contains(chopLine, want) {
			t.Errorf("chop line missing %q:\n%s", want, chopLine)
		}
	}
	if strings.Contains(chopLine, "29193.38") {
		t.Errorf("a ONE-SIDED void must not be folded into CHOP:\n%s", chopLine)
	}
	// The one-sided void keeps its side and its reclaim time.
	if !strings.Contains(out, "29193.38 breakdown (reclaimed 2026-09-02 18:47 CT)") {
		t.Errorf("one-sided void lost its side/time:\n%s", out)
	}
	// Line budget: header + chop + one-sided + instruction, not 5 price lines.
	if n := strings.Count(strings.TrimSpace(out), "\n"); n > 4 {
		t.Errorf("expected a compact block, got %d newlines:\n%s", n, out)
	}
	t.Logf("rendered:\n%s", out)
}

// All one-sided → no CHOP line at all (never an empty aggregate header).
func TestVoidChopAbsentWhenNoBothSidedLevel(t *testing.T) {
	v := []VoidBreakdownLevel{{Price: 29200.50, Short: true, ReclaimedAtCT: "17:23 CT"}}
	out := RenderVoidBreakdownLevels(v, 12)
	if strings.Contains(out, "CHOP (") {
		t.Errorf("no both-sided level → no CHOP line:\n%s", out)
	}
	if !strings.Contains(out, "29200.50 breakdown") {
		t.Errorf("the one-sided void must still render:\n%s", out)
	}
}

// Empty list stays silent — silence means "nothing is void", never a header.
func TestVoidChopEmptyStaysSilent(t *testing.T) {
	if got := RenderVoidBreakdownLevels(nil, 12); got != "" {
		t.Errorf("empty must render nothing, got %q", got)
	}
}
