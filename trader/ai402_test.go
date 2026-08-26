package trader

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// P5 (ledger-close 2026-08-19) — 402 outage alerting. The 08-18 incident: 139
// dead cycles, 4.3 hours, zero alerts.

func ai402Trader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "a402.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "t402", store: st, exchange: "ninjatrader"}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	return at, st
}

func alertCount(t *testing.T, st *store.Store, kind string, ackedToo bool) (n, unacked int) {
	t.Helper()
	alerts, _ := st.Alert().List("t402", 200)
	for _, a := range alerts {
		if a.Kind != kind {
			continue
		}
		n++
		if !a.Acked {
			unacked++
		}
	}
	_ = ackedToo
	return n, unacked
}

func TestClassifyAIError(t *testing.T) {
	deepseek402 := errors.New(`Failed to get AI decision: AI API call failed: API returned error (status 402): {"error":{"message":"Insufficient Balance"}}`)
	if got := classifyAIError(deepseek402); got != "ai_payment_402" {
		t.Fatalf("the live 402 text must classify as ai_payment_402, got %q", got)
	}
	if got := classifyAIError(errors.New("context deadline exceeded")); got != "ai_call_failed" {
		t.Fatalf("a timeout must classify as ai_call_failed, got %q", got)
	}
	if got := classifyAIError(nil); got != "" {
		t.Fatalf("nil error must classify empty, got %q", got)
	}
}

// A 139-burst raises exactly ONE P0 banner, and the first success clears it.
func Test402BurstOneAlertThenAutoClear(t *testing.T) {
	at, st := ai402Trader(t)
	now := time.Now()
	err402 := errors.New("API returned error (status 402): Insufficient Balance")

	for i := 0; i < 139; i++ {
		at.on402Failure(now.Add(time.Duration(i)*3*time.Minute), err402)
	}
	n, unacked := alertCount(t, st, "ai-payment", true)
	if n != 1 || unacked != 1 {
		t.Fatalf("139-burst must raise exactly one unacked P0, got n=%d unacked=%d", n, unacked)
	}

	// First success → outage cleared, banner auto-acked.
	at.onAISuccess(now.Add(5 * time.Hour))
	if at.ai402OutageStartMs != 0 {
		t.Fatal("success must clear the outage latch")
	}
	if _, unacked := alertCount(t, st, "ai-payment", true); unacked != 0 {
		t.Fatal("success must auto-ack the 402 banner")
	}

	// A NEW outage (later) re-alerts with a fresh event id.
	at.on402Failure(now.Add(24*time.Hour), err402)
	if n, unacked := alertCount(t, st, "ai-payment", true); n != 2 || unacked != 1 {
		t.Fatalf("a new outage must raise a NEW banner, got n=%d unacked=%d", n, unacked)
	}
}

// A success with no latched outage is a no-op (no spurious acks/logs).
func TestOnAISuccessNoOutageNoOp(t *testing.T) {
	at, _ := ai402Trader(t)
	at.onAISuccess(time.Now())
	if at.ai402OutageStartMs != 0 {
		t.Fatal("no-outage success must stay clear")
	}
}

// AI_BALANCE_WARN default-off contract.
func TestAIBalanceWarnDefaultOff(t *testing.T) {
	if _, on := aiBalanceWarnThreshold(); on {
		t.Fatal("AI_BALANCE_WARN must default OFF")
	}
	t.Setenv("AI_BALANCE_WARN", "5.0")
	if th, on := aiBalanceWarnThreshold(); !on || th != 5.0 {
		t.Fatalf("AI_BALANCE_WARN=5.0 must arm the poll, got %v %v", th, on)
	}
	t.Setenv("AI_BALANCE_WARN", "junk")
	if _, on := aiBalanceWarnThreshold(); on {
		t.Fatal("malformed AI_BALANCE_WARN must stay OFF")
	}
}
