package ninjatrader

import "testing"

// T8 (2026-08-27) — the bar_update frame-log sampler knob: default 500, env
// override, non-positive values fall back. The frame log is now DEBUG-sampled
// so the journald cap survives multi-day retention.
func TestT8BarUpdateLogSampleKnob(t *testing.T) {
	t.Setenv("BAR_UPDATE_LOG_SAMPLE", "")
	if got := barUpdateLogSample(); got != 500 {
		t.Fatalf("default = %d, want 500", got)
	}
	t.Setenv("BAR_UPDATE_LOG_SAMPLE", "1000")
	if got := barUpdateLogSample(); got != 1000 {
		t.Fatalf("override = %d, want 1000", got)
	}
	t.Setenv("BAR_UPDATE_LOG_SAMPLE", "0") // invalid → default
	if got := barUpdateLogSample(); got != 500 {
		t.Fatalf("zero override = %d, want 500 fallback", got)
	}
	t.Setenv("BAR_UPDATE_LOG_SAMPLE", "x") // unparsable → default
	if got := barUpdateLogSample(); got != 500 {
		t.Fatalf("junk override = %d, want 500 fallback", got)
	}
}
