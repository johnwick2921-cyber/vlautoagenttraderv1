package kernel

import "testing"

// B1 (T3) — one table; every legal primary TF mapped; the previously-invisible
// 3m/30m get forming labels + interval math.
func TestOneTimeframeTable(t *testing.T) {
	for tf, want := range map[string]int64{
		"1m": 60_000, "2m": 120_000, "3m": 180_000, "5m": 300_000,
		"15m": 900_000, "30m": 1_800_000, "1h": 3_600_000, "1d": 86_400_000,
	} {
		got, ok := TFDurationMs(tf)
		if !ok || got != want {
			t.Errorf("TFDurationMs(%s) = %d,%v want %d,true", tf, got, ok, want)
		}
	}
	if _, ok := TFDurationMs("7m"); ok {
		t.Error("unmapped TF must return ok=false (boot-fail upstream, never a silent default)")
	}
	// The B4/label path sees the full table now (was 1m/5m/15m only).
	if tfIntervalMs("3m") != 180_000 || tfIntervalMs("30m") != 1_800_000 {
		t.Error("stale_data tfIntervalMs must delegate to the table (3m/30m coverage)")
	}
}
