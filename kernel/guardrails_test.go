package kernel

import (
	"testing"
	"time"
)

// --- Strategy Studio Phase 1 — Chunk 2 daily guardrails ---

func TestDailyGuardrails_DailyLoss(t *testing.T) {
	base := DailyGuardrails{MasterEnabled: true, DailyLossEnabled: true, DailyLossLimitUSD: 500}

	// loss exactly AT the limit trips (stop once loss >= $X)
	g := base
	g.DailyRealizedPnL = -500
	if d, err := g.Check(); err == nil || d != RiskForceFlat {
		t.Fatalf("loss=-500 limit=500 should ForceFlat, got d=%v err=%v", d, err)
	}
	// loss past the limit trips
	g = base
	g.DailyRealizedPnL = -600.01
	if _, err := g.Check(); err == nil {
		t.Fatal("loss=-600 should trip the daily-loss limit")
	}
	// loss under the limit passes
	g = base
	g.DailyRealizedPnL = -499
	if _, err := g.Check(); err != nil {
		t.Fatalf("loss=-499 under limit should pass, got %v", err)
	}
	// toggle OFF → not enforced even at a huge loss
	g = base
	g.DailyLossEnabled = false
	g.DailyRealizedPnL = -10000
	if _, err := g.Check(); err != nil {
		t.Fatalf("daily-loss toggle OFF must not block, got %v", err)
	}
}

func TestDailyGuardrails_ProfitTarget(t *testing.T) {
	g := DailyGuardrails{MasterEnabled: true, DailyProfitEnabled: true, DailyProfitTargetUSD: 1000, DailyRealizedPnL: 1000}
	if d, err := g.Check(); err == nil || d != RiskBlockEntry {
		t.Fatalf("profit=1000 target=1000 should BlockEntry, got d=%v err=%v", d, err)
	}
	g.DailyRealizedPnL = 999
	if _, err := g.Check(); err != nil {
		t.Fatalf("profit=999 under target should pass, got %v", err)
	}
	g.DailyProfitEnabled = false
	g.DailyRealizedPnL = 999999
	if _, err := g.Check(); err != nil {
		t.Fatalf("profit toggle OFF must not block, got %v", err)
	}
}

func TestDailyGuardrails_MaxTrades(t *testing.T) {
	g := DailyGuardrails{MasterEnabled: true, MaxDailyTradesEnabled: true, MaxDailyTrades: 3, TradesToday: 3}
	if d, err := g.Check(); err == nil || d != RiskBlockEntry {
		t.Fatalf("trades=3 max=3 should BlockEntry (4th blocked), got d=%v err=%v", d, err)
	}
	g.TradesToday = 2
	if _, err := g.Check(); err != nil {
		t.Fatalf("trades=2 max=3 should pass, got %v", err)
	}
	g.MaxDailyTradesEnabled = false
	g.TradesToday = 99
	if _, err := g.Check(); err != nil {
		t.Fatalf("max-trades toggle OFF must not block, got %v", err)
	}
}

func TestDailyGuardrails_MasterSwitchBypassesAll(t *testing.T) {
	// Master OFF must bypass EVERY guardrail, even ones that would otherwise trip.
	g := DailyGuardrails{
		MasterEnabled:         false,
		DailyLossEnabled:      true, DailyLossLimitUSD: 100, DailyRealizedPnL: -9999,
		MaxDailyTradesEnabled: true, MaxDailyTrades: 1, TradesToday: 99,
	}
	if d, err := g.Check(); err != nil || d != RiskAllow {
		t.Fatalf("master OFF must bypass ALL guardrails (RiskAllow,nil), got d=%v err=%v", d, err)
	}
}

func TestDailyGuardrails_UnsetValueNotEnforced(t *testing.T) {
	// toggle ON but value 0 (unset) → not enforced (no false block)
	g := DailyGuardrails{MasterEnabled: true, DailyLossEnabled: true, DailyLossLimitUSD: 0, DailyRealizedPnL: -100000}
	if _, err := g.Check(); err != nil {
		t.Fatalf("enabled but value=0 must not block, got %v", err)
	}
}

func TestBoolOrDefault(t *testing.T) {
	tr, fa := true, false
	if boolOrDefault(nil, true) != true || boolOrDefault(nil, false) != false {
		t.Fatal("nil should return the default")
	}
	if boolOrDefault(&tr, false) != true || boolOrDefault(&fa, true) != false {
		t.Fatal("non-nil should return the pointed value")
	}
}

func TestFirstPositive(t *testing.T) {
	if firstPositive(0, 500) != 500 {
		t.Fatal("per-strategy 0 → env fallback 500")
	}
	if firstPositive(250, 500) != 250 {
		t.Fatal("per-strategy 250 → overrides env")
	}
	if firstPositive(0, 0) != 0 {
		t.Fatal("both unset → 0")
	}
}

func TestCMESessionDayStart(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	// 18:00 CT → session started 17:00 CT the SAME calendar day
	now := time.Date(2026, 6, 1, 18, 0, 0, 0, chicago)
	s := CMESessionDayStart(now)
	if s.Hour() != 17 || s.Month() != time.June || s.Day() != 1 {
		t.Fatalf("18:00 CT → session start 17:00 same day, got %v", s)
	}
	// 09:00 CT → session started 17:00 CT the PREVIOUS calendar day
	now = time.Date(2026, 6, 1, 9, 0, 0, 0, chicago)
	s = CMESessionDayStart(now)
	if s.Hour() != 17 || s.Month() != time.May || s.Day() != 31 {
		t.Fatalf("09:00 CT → session start 17:00 prev day (May 31), got %v", s)
	}
}

// --- Chunk 3 — max contracts + visible equity×20 cap resolvers ---

func TestResolveMaxContracts(t *testing.T) {
	// Hardening D3 (audit F2): the futures contract clamp is ALWAYS ON — there is
	// no master/toggle argument that can disable it. Per-strategy value overrides;
	// unset/invalid → venue default; NEVER 0 (a futures order is never unclamped).
	if got := ResolveMaxContracts(4, 10); got != 4 { // per-strategy overrides
		t.Fatalf("per-strategy 4 → 4, got %d", got)
	}
	if got := ResolveMaxContracts(0, 10); got != 10 { // unset → venue default
		t.Fatalf("unset → default 10, got %d", got)
	}
	if got := ResolveMaxContracts(-1, 10); got != 10 { // invalid → venue default, never 0
		t.Fatalf("negative → default 10 (never unclamped), got %d", got)
	}
}

func TestResolveNotionalLeverage(t *testing.T) {
	// Hardening D3 (audit F2): the futures notional ceiling is ALWAYS ON —
	// master-independent venue safety. Per-strategy overrides; unset/invalid →
	// venue default; NEVER 0 (a futures order always has a notional ceiling).
	if got := ResolveNotionalLeverage(50, 20); got != 50 { // per-strategy overrides
		t.Fatalf("per-strategy 50 → 50, got %v", got)
	}
	if got := ResolveNotionalLeverage(0, 20); got != 20 { // unset → venue default
		t.Fatalf("unset → default 20, got %v", got)
	}
	if got := ResolveNotionalLeverage(-5, 20); got != 20 { // invalid → venue default, never 0
		t.Fatalf("negative → default 20 (never uncapped), got %v", got)
	}
}

func TestResolveConcurrentCap(t *testing.T) {
	// Hardening D4 (audit F3): per-strategy max_positions is authoritative; the env
	// RISK_MAX_CONCURRENT_TRADES value is only the fallback.
	if n, src := ResolveConcurrentCap(3, 2); n != 3 || src != "strategy max_positions" {
		t.Fatalf("per-strategy 3 must govern over env 2, got %d/%q", n, src)
	}
	if n, src := ResolveConcurrentCap(0, 2); n != 2 || src != "env RISK_MAX_CONCURRENT_TRADES" {
		t.Fatalf("unset per-strategy → env fallback 2, got %d/%q", n, src)
	}
	if n, _ := ResolveConcurrentCap(5, 2); n != 5 {
		t.Fatalf("per-strategy 5 must reach the gate (was silently capped at env 2), got %d", n)
	}
}

// --- Chunk 4 — time/news blackout window ---

func TestInBlackoutWindow(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	mk := func(h, m int) time.Time { return time.Date(2026, 6, 1, h, m, 0, 0, chicago) }

	// window 13:25–13:35 CT (start inclusive, end exclusive)
	if !InBlackoutWindow(mk(13, 30), "13:25", "13:35") {
		t.Fatal("13:30 in [13:25,13:35) should be blackout")
	}
	if InBlackoutWindow(mk(13, 35), "13:25", "13:35") {
		t.Fatal("13:35 == end (exclusive) should NOT be blackout")
	}
	if InBlackoutWindow(mk(13, 24), "13:25", "13:35") {
		t.Fatal("13:24 before start should NOT be blackout")
	}
	if InBlackoutWindow(mk(9, 0), "13:25", "13:35") {
		t.Fatal("09:00 outside should NOT be blackout")
	}
	// wraps midnight 23:00–01:00
	if !InBlackoutWindow(mk(23, 30), "23:00", "01:00") {
		t.Fatal("23:30 in wrap window should be blackout")
	}
	if !InBlackoutWindow(mk(0, 30), "23:00", "01:00") {
		t.Fatal("00:30 in wrap window should be blackout")
	}
	if InBlackoutWindow(mk(2, 0), "23:00", "01:00") {
		t.Fatal("02:00 outside wrap should NOT be blackout")
	}
	// malformed / empty / zero-width → false (a misconfig never silently halts)
	if InBlackoutWindow(mk(13, 30), "", "13:35") {
		t.Fatal("empty start should NOT block")
	}
	if InBlackoutWindow(mk(13, 30), "bad", "13:35") {
		t.Fatal("malformed should NOT block")
	}
	if InBlackoutWindow(mk(13, 30), "13:30", "13:30") {
		t.Fatal("zero-width window should NOT block")
	}
}

// --- Chunk 5 — consistency rule ---

func TestConsistencyBreached(t *testing.T) {
	// today=600 of total=1000 (prior=400), pct=50 → 600 ≥ 500 → breach
	if !ConsistencyBreached(600, 1000, 50) {
		t.Fatal("today 600 ≥ 50% of 1000 should breach")
	}
	// boundary: today=500 of total=1000 (prior=500) → 500 ≥ 500 → breach
	if !ConsistencyBreached(500, 1000, 50) {
		t.Fatal("boundary 500 == 50% of 1000 should breach")
	}
	// today=400 of total=1000 (prior=600) → 400 < 500 → ok
	if ConsistencyBreached(400, 1000, 50) {
		t.Fatal("today 400 < 50% of 1000 should NOT breach")
	}
	// first profitable day: today==total (prior=0) → never self-lock
	if ConsistencyBreached(1000, 1000, 50) {
		t.Fatal("first profitable day (today==total) should NOT self-lock")
	}
	// losing day, zero total, or pct≤0 → never breach
	if ConsistencyBreached(-100, 1000, 50) {
		t.Fatal("losing day should not breach")
	}
	if ConsistencyBreached(600, 0, 50) {
		t.Fatal("total 0 should not breach")
	}
	if ConsistencyBreached(600, 1000, 0) {
		t.Fatal("pct 0 (disabled) should not breach")
	}
}
