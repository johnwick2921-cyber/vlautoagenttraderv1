package trader

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// ── CLASS 45 F4 / E4 (owner addition) — CUMULATIVE CORRECTIONS, TOP AND TAIL ──
//
// LONDON 2026-09-02: attempt 3's block carried attempt 2's fade defect, not
// attempt 1's void, so the model was corrected about the fade and walked back
// into the void. A block naming only the LAST defect can only teach the model to
// avoid its most recent mistake.
func TestClass45CumulativeRejectsTopAndTail(t *testing.T) {
	live := []string{"reject", "breakdown_continue", "hold"}
	voidErr := errors.New("S1 breakdown_continue: a close came back across 29021.25 — the breakdown is void")
	fadeErr := errors.New(`scenario[0].confirm.rule "1x5m_close" — fade_requires_touch`)

	// The chain accumulates DISTINCT defects in order.
	var h []string
	h = addDistinctReject(h, voidErr)
	h = addDistinctReject(h, fadeErr)
	h = addDistinctReject(h, errors.New(voidErr.Error())) // the same defect again
	if len(h) != 2 {
		t.Fatalf("the history must hold 2 DISTINCT defects, got %d: %v", len(h), h)
	}
	if h[0] != voidErr.Error() || h[1] != fadeErr.Error() {
		t.Fatalf("order must be first-seen-first: %v", h)
	}
	if got := addDistinctReject(nil, nil); got != nil {
		t.Fatalf("a nil error must add nothing, got %v", got)
	}

	header := plannerRejectHeader(h, live)
	tail := plannerRejectTail(h, live)

	// THE HEADER: it leads, it says the standing rules are overridden, and it
	// lists BOTH defects — the fade AND the void attempt 3 walked back into.
	for _, want := range []string{
		"CORRECTIONS FROM THIS READ — read these FIRST",
		"The standing rules below still apply EXCEPT where this correction overrides them.",
		"2 DISTINCT defects",
		"came back across 29021.25",
		"fade_requires_touch",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q:\n%s", want, header)
		}
	}
	// THE TAIL: the same list, repeated at the end of a 6.6k-token prompt.
	for _, want := range []string{"CORRECTIONS FROM THIS READ (repeated", "came back across 29021.25", "fade_requires_touch", "Fix ALL of the above"} {
		if !strings.Contains(tail, want) {
			t.Errorf("tail missing %q:\n%s", want, tail)
		}
	}
	// A single defect reads naturally, not as "1 DISTINCT defects".
	if one := plannerRejectHeader(h[:1], live); strings.Contains(one, "DISTINCT defects") {
		t.Errorf("a single reject must not use the plural form:\n%s", one)
	}
	// No history → no blocks at all (a first attempt is untouched).
	if plannerRejectHeader(nil, live) != "" || plannerRejectTail(nil, live) != "" {
		t.Error("no rejects → no correction blocks")
	}

	// The token cost of leading AND repeating, quoted rather than assumed.
	t.Logf("F4 token delta: header ≈%d tok, tail ≈%d tok, total ≈%d tok added to a ~6,600-tok prompt",
		estimatePromptTokens(header), estimatePromptTokens(tail), estimatePromptTokens(header)+estimatePromptTokens(tail))
}

// The assembly must actually USE both blocks — a source pin, because the retry
// loop needs a live AI client to exercise end to end.
func TestClass45AssemblyUsesHeaderAndTail(t *testing.T) {
	b, err := os.ReadFile("auto_trader_planner.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "userPrompt = header + prompt + plannerRejectTail(rejectHistory, liveConditions)") {
		t.Error("the re-author must assemble header + prompt + tail (E4)")
	}
	if !strings.Contains(src, "rejectHistory = addDistinctReject(rejectHistory, lastErr)") {
		t.Error("every reject site must record into the cumulative history")
	}
	if n := strings.Count(src, "rejectHistory = addDistinctReject"); n < 6 {
		t.Errorf("only %d reject sites record history — every site must, or a defect class goes unremembered", n)
	}
}
