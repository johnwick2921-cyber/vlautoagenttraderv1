package kernel

// B1 (T3, fail-register wave 2026-08-20) — THE one timeframe→duration table.
//
// The anatomy census found THREE disagreeing TF→ms tables (kernel stale_data
// knew 1m/5m/15m, the dodge knew six, the NT8 bridge knew fourteen and
// defaulted unknown → 60 000ms — silently corrupting the bar-close gate and
// the supersession watermark for any unmapped primary TF). This file is now
// the single source; every consumer delegates here:
//   - forming/closed prompt labels (engine_prompt.go tfIntervalMs)
//   - B4 stale-entry ages (stale_data.go)
//   - the dodge's next-close boundary (trader/discard_burn.go)
//   - the NT8 bridge's CloseTime derivation (trader/ninjatrader)
// An UNMAPPED primary timeframe is a BOOT FAIL (auto_trader Run), never a
// silent 60s default.

// TFDurationMs returns the duration of a timeframe label in milliseconds.
// ok=false for anything not in the vocabulary.
func TFDurationMs(tf string) (int64, bool) {
	switch tf {
	case "1m":
		return 60_000, true
	case "2m":
		return 120_000, true
	case "3m":
		return 180_000, true
	case "5m":
		return 300_000, true
	case "15m":
		return 900_000, true
	case "30m":
		return 1_800_000, true
	case "1h":
		return 3_600_000, true
	case "2h":
		return 7_200_000, true
	case "4h":
		return 14_400_000, true
	case "6h":
		return 21_600_000, true
	case "8h":
		return 28_800_000, true
	case "12h":
		return 43_200_000, true
	case "1d":
		return 86_400_000, true
	case "3d":
		return 259_200_000, true
	case "1w":
		return 604_800_000, true
	}
	return 0, false
}
