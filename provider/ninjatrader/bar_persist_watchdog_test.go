package ninjatrader

import (
	"os"
	"strings"
	"testing"
	"time"
)

// PRE-REOPEN F2 (2026-08-28) — the persist-stall watchdog env knob: default
// 60s, garbage → default, and a 10s floor so a mistyped value can't make the
// alarm scream every second.
func TestPersistWatchdogSeconds(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int64
	}{
		{"unset defaults to 60", "", 60},
		{"explicit value", "45", 45},
		{"below floor falls back to default", "5", 60},
		{"negative clamps to default", "-20", 60},
		{"garbage falls back to default", "fast", 60},
		{"whitespace tolerated", " 90 ", 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env == "" {
				_ = os.Unsetenv("PERSIST_STALL_WATCHDOG_S")
			} else {
				_ = os.Setenv("PERSIST_STALL_WATCHDOG_S", tc.env)
			}
			if got := persistWatchdogSeconds(); got != tc.want {
				t.Fatalf("persistWatchdogSeconds() = %d, want %d", got, tc.want)
			}
		})
	}
	_ = os.Unsetenv("PERSIST_STALL_WATCHDOG_S")
}

// S1 WIRE-UP (2026-08-29) — the BEHAVIOR fixture the half-ship escaped without:
// the alarm must FIRE (return the exact ERROR text) exactly once on a simulated
// 61s stall, stay silent in the same watchdog window (dedup via persistAlarmAt),
// and NOT re-alarm after the flush stamp advances. Existence tests can't catch
// a declared-but-unwired guard — this one proves firing.
func TestPersistWatchdogAlarmFiresOnceAndRecovers(t *testing.T) {
	_ = os.Setenv("PERSIST_STALL_WATCHDOG_S", "10")
	defer os.Unsetenv("PERSIST_STALL_WATCHDOG_S")

	now := time.Now().Unix()

	// Cold start (no flush ever) must never alarm.
	persistLastFlushAt.Store(0)
	persistLastFrameAt.Store(0)
	persistAlarmAt.Store(0)
	if m := persistWatchdogAlarmAt(now); m != "" {
		t.Fatalf("cold start must not alarm, got %q", m)
	}

	// W1: frames are FLOWING (live market) while the flush is stalled 61s →
	// the alarm FIRES, with the loud ERROR text.
	persistLastFlushAt.Store(now - 61)
	persistLastFrameAt.Store(now) // frames flowing right now
	a1 := persistWatchdogAlarmAt(now)
	if a1 == "" {
		t.Fatal("61s stall must fire the watchdog alarm")
	}
	if !strings.Contains(a1, "🔕 PERSIST WATCHDOG") || !strings.Contains(a1, "61s") {
		t.Fatalf("alarm text must carry the marker + duration, got %q", a1)
	}

	// Dedup: within the same watchdog window the alarm is silent (exactly once).
	if a2 := persistWatchdogAlarmAt(now + 5); a2 != "" {
		t.Fatalf("dedup must silence the same window, got %q", a2)
	}

	// Resumed flush → the stamp advances → the next window is clean.
	persistLastFlushAt.Store(now + 65)
	if a3 := persistWatchdogAlarmAt(now + 70); a3 != "" {
		t.Fatalf("resumed flush must not re-alarm, got %q", a3)
	}

	// A new stall AFTER recovery fires again (the guard is alive, not a one-shot).
	persistLastFlushAt.Store(now + 130)
	persistLastFrameAt.Store(now + 200) // frames still flowing (a GORM stall never stops the wire)
	persistAlarmAt.Store(0)
	if a4 := persistWatchdogAlarmAt(now + 200); a4 == "" {
		t.Fatal("a fresh stall after recovery must re-fire the alarm")
	}
}

// W1 QUIET-WIRE (2026-08-29) — the behavior fixture that kills the 373-line
// weekend storm: a stale flush stamp (e.g. the boot backfill, hours ago) with
// NO live frames flowing must stay SILENT — the wire is idle, not stalled.
// The alarm may only fire once frames are flowing again and flushes still
// aren't. This is the exact 2026-08-29 boot/storm reproduction: last flush at
// boot+2min, then 5.8h of market-closed idle = 1 ERRO/min under the old code.
func TestPersistWatchdogIdleWireStaysSilent(t *testing.T) {
	_ = os.Setenv("PERSIST_STALL_WATCHDOG_S", "10")
	defer os.Unsetenv("PERSIST_STALL_WATCHDOG_S")

	now := time.Now().Unix()
	persistAlarmAt.Store(0)

	// Weekend idle: boot backfill flushed hours ago, no frames since. Silent.
	persistLastFlushAt.Store(now - 61)
	persistLastFrameAt.Store(now - 3600)
	if m := persistWatchdogAlarmAt(now); m != "" {
		t.Fatalf("idle wire (no frames for an hour) must stay silent, got %q", m)
	}

	// Idle for another full watchdog window — still silent (no 1/min storm).
	for i := int64(1); i <= 12; i++ {
		if m := persistWatchdogAlarmAt(now + i*10); m != "" {
			t.Fatalf("idle wire tick %d must stay silent, got %q", i, m)
		}
	}

	// Frames resume AND the flush also recovers → silent (healthy).
	persistLastFlushAt.Store(now + 130)
	persistLastFrameAt.Store(now + 130)
	if m := persistWatchdogAlarmAt(now + 140); m != "" {
		t.Fatalf("flowing wire with healthy flushes must stay silent, got %q", m)
	}

	// Frames flowing while the flush stalls again → fires (the guard is alive
	// against REAL stalls, only the idle cry-wolf is dead).
	persistLastFrameAt.Store(now + 261) // frames flowing at the check (stall ≠ wire death)
	persistAlarmAt.Store(0)
	if m := persistWatchdogAlarmAt(now + 261); m == "" {
		t.Fatal("frames flowing + flush stalled 61s must fire the watchdog")
	}
}
