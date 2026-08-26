package trader

import (
	"os"
	"strings"
	"testing"

	"nofx/kernel"
)

func readFileForTest(name string) (string, error) {
	b, err := os.ReadFile(name)
	return string(b), err
}

func containsStr(hay, needle string) bool { return strings.Contains(hay, needle) }

// ---- 2.1 DODGE timing ----

func TestStaleDodgeDefer(t *testing.T) {
	const s = int64(1000)
	cases := []struct {
		name              string
		nowToClose, avg   int64
		wantDodge         bool
		wantDefer         int64
	}{
		// The spec's worked example: cycle due 40s before close, avg 70s →
		// window 84s > 40s → defer to close+1s = 41s.
		{"spec_example_40s_before_close_avg70", 40 * s, 70 * s, true, 41 * s},
		{"far_from_close_no_dodge", 200 * s, 70 * s, false, 0},
		{"short_calls_no_dodge", 40 * s, 20 * s, false, 0}, // window 24s < 40s
		{"no_history_never_dodges", 5 * s, 0, false, 0},
		{"exactly_at_window_no_dodge", 84 * s, 70 * s, false, 0}, // >= window boundary
		{"past_close_no_dodge", -1 * s, 70 * s, false, 0},
	}
	for _, c := range cases {
		now := int64(1_700_000_000_000)
		gotDefer, gotDodge := staleDodgeDefer(now, now+c.nowToClose, c.avg)
		if gotDodge != c.wantDodge || gotDefer != c.wantDefer {
			t.Errorf("%s: got (defer=%d dodge=%v), want (defer=%d dodge=%v)",
				c.name, gotDefer, gotDodge, c.wantDefer, c.wantDodge)
		}
	}
}

func TestAICallRingAverage(t *testing.T) {
	at := &AutoTrader{}
	if at.avgAICallMs() != 0 {
		t.Fatal("empty ring must average 0 (dodge disabled until history exists)")
	}
	// 25 samples: the ring keeps the last 20. First 5 (value 1000) are evicted;
	// the remaining 20 are all 2000.
	for i := 0; i < 5; i++ {
		at.recordAICallMs(1000)
	}
	for i := 0; i < 20; i++ {
		at.recordAICallMs(2000)
	}
	if got := at.avgAICallMs(); got != 2000 {
		t.Errorf("ring average after eviction = %d, want 2000", got)
	}
	at.recordAICallMs(0) // ignored
	if got := at.avgAICallMs(); got != 2000 {
		t.Errorf("zero-duration sample must be ignored, got %d", got)
	}
}

// ---- 2.2 RE-EVAL verdicts ----

func TestReevalSupersededEntry(t *testing.T) {
	const (
		sl    = 29600.0
		snap  = 29650.0
		atr   = 40.0
		drift = 0.25
	)
	// PASS: long, fresh bar stayed above the stop, tiny drift.
	if v := reevalSupersededEntry("open_long", sl, snap, 29660, 29620, 29655, atr, drift); !v.pass {
		t.Errorf("pass path refused: %s", v.reason)
	}
	// REFUSE: long, fresh bar LOW touched the stop.
	if v := reevalSupersededEntry("open_long", sl, snap, 29660, 29600, 29655, atr, drift); v.pass {
		t.Error("sl-breached long must refuse")
	}
	// REFUSE: short, fresh bar HIGH touched the stop.
	if v := reevalSupersededEntry("open_short", 29700, snap, 29700, 29620, 29655, atr, drift); v.pass {
		t.Error("sl-breached short must refuse")
	}
	// REFUSE: drift ≥ 0.25×ATR (limit 10.0; drift exactly 10 refuses).
	if v := reevalSupersededEntry("open_long", sl, snap, 29670, 29630, snap+10, atr, drift); v.pass {
		t.Error("drift at the limit must refuse")
	}
	// PASS: drift just inside the limit.
	if v := reevalSupersededEntry("open_long", sl, snap, 29670, 29630, snap+9.9, atr, drift); !v.pass {
		t.Errorf("drift inside the limit refused: %s", v.reason)
	}
	// Fail-safe refusals: no stop, no ATR, not an entry.
	if v := reevalSupersededEntry("open_long", 0, snap, 29670, 29630, snap, atr, drift); v.pass {
		t.Error("entry without a stated stop must refuse")
	}
	if v := reevalSupersededEntry("open_long", sl, snap, 29670, 29630, snap, 0, drift); v.pass {
		t.Error("atr unavailable must refuse (cannot verify drift)")
	}
	if v := reevalSupersededEntry("wait", sl, snap, 29670, 29630, snap, atr, drift); v.pass {
		t.Error("non-entry must refuse")
	}
}

// ---- classification: waits stay free, closes stay conservative ----

func TestClassifyDecisions(t *testing.T) {
	ds := []kernel.Decision{
		{Action: "wait"}, {Action: "hold"},
	}
	if e, c := classifyDecisions(ds); e != 0 || c != 0 {
		t.Errorf("wait/hold-only classified as e=%d c=%d, want 0/0 (free quiet discard path)", e, c)
	}
	ds = append(ds, kernel.Decision{Action: "open_short"})
	if e, c := classifyDecisions(ds); e != 1 || c != 0 {
		t.Errorf("entries-only path: e=%d c=%d, want 1/0", e, c)
	}
	ds = append(ds, kernel.Decision{Action: "close_long"})
	if e, c := classifyDecisions(ds); e != 1 || c != 1 {
		t.Errorf("close-containing path: e=%d c=%d, want 1/1 (legacy conservative discard)", e, c)
	}
}

// ---- 2.3 status honesty: no "Failed" from the discard path ----
// The discard paths stamp guardrail_skip (deliberate ghost-record semantics,
// Success=false in the DB); the DISPLAY honesty lives in DecisionCard's
// tri-state badge. This pins the reasons the FE keys on, so a rename breaks
// loudly here instead of silently regressing the badge.
func TestDiscardReasonsAreStable(t *testing.T) {
	for _, want := range []string{"superseded_wait", "stale_reeval_refused", "stale_bar_discarded"} {
		found := false
		for _, src := range []string{"auto_trader_loop.go"} {
			b, err := readFileForTest(src)
			if err != nil {
				t.Fatalf("read %s: %v", src, err)
			}
			if containsStr(b, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("discard reason %q no longer emitted — DecisionCard's ℹ️ badge and the gate-block counters key on it", want)
		}
	}
}
