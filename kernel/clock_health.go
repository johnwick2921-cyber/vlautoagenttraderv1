package kernel

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"nofx/logger"
	"nofx/market"
)

// PHASE 3.5 (timegate audit 2026-08-18) — CLOCK HEALTH, log-only.
//
// The C2 clock-drift guard already protects signals (feed-stamped since the
// WSL↔NT8 skew finding; thresholds untouched here by owner decision). What was
// missing is OBSERVABILITY: at boot and at each session roll, one line stating
// what the Go clock, the NT8 feed clock, and the OS time-sync layer each
// believe — so the next skew is a grep, not a forensic hunt. On tolerance
// breach it logs CRITICAL and nothing else: no new trade-blocking gate
// (guardrails-master decision), the existing C2 behavior is unchanged.

// C2ToleranceMs exposes the EXISTING C2 tolerance for log-only consumers.
// Additive accessor; the guard's own threshold is untouched.
func C2ToleranceMs() int64 { return clockDriftToleranceMs }

// timesyncStatus reports systemd-timesyncd's view, best-effort with a hard
// 2s ceiling ("unknown" on any failure — WSL2 setups vary).
func timesyncStatus() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "timedatectl", "show",
		"-p", "NTPSynchronized", "-p", "NTP").Output()
	if err != nil {
		return "unknown"
	}
	return strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", " ")
}

// LogClockHealth emits the clock-health line for one symbol's feed. tag names
// the trigger ("boot" | "session-roll:<name>").
func LogClockHealth(tag, symbol string) {
	now := time.Now()
	// One labelled dual read, via the sanctioned tz helpers (the TZ guard test
	// rejects bare layouts — deservedly).
	line := "🕰 clock-health [" + tag + "] go=" + ClockCTAndUTC(now)

	drift := int64(0)
	haveBar := false
	if market.FuturesBarsProvider != nil {
		if bars := market.FuturesBarsProvider(symbol, "1m", 3); len(bars) > 0 {
			last := bars[len(bars)-1]
			nt8Ms := last.OpenTime + 60_000 // close of the freshest 1m bar
			drift = now.UnixMilli() - nt8Ms
			haveBar = true
			line += " nt8_last_bar=" + ClockCTAndUTC(time.UnixMilli(nt8Ms)) +
				" drift_ms=" + i64str(drift)
		}
	}
	if !haveBar {
		line += " nt8_last_bar=none drift_ms=n/a"
	}
	line += " timesync{" + timesyncStatus() + "} tolerance_ms=" + i64str(C2ToleranceMs())
	logger.Infof("%s", line)

	// A stale-but-live feed inflates "drift" honestly (the bar IS old); the
	// CRITICAL only means "Go clock vs newest feed evidence disagree beyond C2
	// tolerance" — exactly what the operator must look at either way.
	if haveBar && absI64(drift) > C2ToleranceMs() {
		logger.Errorf("🚨 CLOCK CRITICAL [%s]: |drift| %dms exceeds C2 tolerance %dms — check WSL2 time-sync (systemd-timesyncd) and the NT8 feed. Log-only: no trading gate added.",
			tag, absI64(drift), C2ToleranceMs())
	}
}

func i64str(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
