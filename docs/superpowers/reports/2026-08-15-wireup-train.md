# VL Day-Plan — wire-up train (fixes the audit's 10 dead wires)

**LINE 1 — ALL 10 WIRES DONE + GREEN (W1..W10). Full suite 26 pkgs green, goldens
byte-identical, SIM-only, additive. Two deploys: W1+W2 pre-Sun-17:00 (done), W3–W10
one rebuild before Mon 08:00 (owner). HEAD `0f79fb4f`.**
Read-only audit `7344820d` → this train wires the dead wires end-to-end. Additive,
SIM-only, guardrails untouched, own commit per item.

---

## ✅ W1 — Sunday-read guard + digest window (`57b518d8`) [A]
The read scheduler used `timeReachedCT` (fires for ANY time ≥ ReadCT), so at
**Sunday 17:00 CT** (CME reopens = Monday's trade-date) the NY 08:25 read fired
early, building Monday's plan from stale Sunday-evening data; the real 08:25 read
was then deduped away.
- `inSessionReadWindow(now, ReadCT, WindowEnd)` — fire ONLY inside the session's
  own window `[ReadCT, WindowEnd)`, wrap-aware (ASIA 16:55→02:00). NY `[08:25,15:00)`
  → Sunday 17:00 is past it → NY never fires there. + `IsCMEOpen(now)` (holiday/
  weekend-aware) gate. Monday's plan is now built fresh at 08:25.
- Daily digest moved from `>=16:00` (inside the CME break → fired 17:00+ with the
  NEW day's empty P&L window; Friday never fired) to `[15:00,16:00)` (RTH-close→
  break) where the trade-date + P&L window are still the CLOSING day's. Reachable
  Mon–Fri.
- **Proven:** `TestW1SundayReadGuard` + a full **Sat+Sun sweep** (NY read false at
  every weekend step) + daily-roll window + ASIA wrap. `timeReachedCT` kept for the
  last-entry/eod-flat gates (correct all-day-after use).

## ✅ W2 — pin the EXACT planner model + stats reset (`0c8bae59`) [A]
Plans stamped the provider alias `deepseek`, not an exact string (§125 violation).
- `mcp.AIClient.ResolvedModel()` exposes the exact model; `DefaultModelForAlias`/
  `IsProviderAlias` map/detect aliases (`deepseek`→`deepseek-v4-pro`).
  `resolvePlannerClient` pins the exact id on all 3 return paths (prefer the
  client's model, else map the alias, else warn) → every plan row carries an exact
  model string.
- §128 no cross-model pooling: `maybeResetStatsOnModelChange` records the pinned
  model in system_config; on a change → `MatchedRandomStore.ResetWindow` (clears
  verdicts + weekly), logged.
- **Proven:** alias→exact mapping/detection + live `ResolvedModel` + `ResetWindow`
  clears both tables + the resolver now asserts an exact id. 26 pkgs green.

---

## ⏫ DEPLOY NOW (owner, before Sunday 17:00 CT — Go touched, rebuild+restart):
```bash
cd /home/hoang/nofx && git pull && go build -o nofx-bin ./... && echo BUILD OK
kill -9 $(pgrep -f nofx-bin)   # systemd Restart=on-failure respawns the new binary
```
(`sudo systemctl restart nofx` is classifier-blocked; SIGKILL is the deploy per
CLAUDE.md. Do it in the flat/CME-closed weekend window. No AddOn F5 — no `.cs`
changed.) This makes Monday's 08:25 read fire correctly + exact model pinned.

---

## ✅ W3 — calendar producer + red-news blackout (`89e2809f`) [A]
The FF calendar was consumed by the planner but never PRODUCED, and no executor
blackout existed.
- `maybeFetchCalendar` (gated · ≤1/hr throttle · idempotent `SaveSliceIfAbsent`)
  stores this week's FF slices; the planner read already consumes them.
- `kernel.T1BlackoutWindows`/`InT1Blackout` + `T1NoTradeLines`: a T1 (High-impact)
  event → HARD ±15-min no-entry blackout in the session gate + injected into the
  plan's `no_trade`. T2 makes no hard window.
- **Proven:** FF fixture → T1 window blocks the gate ±15m, T2 makes none; no-trade
  lines dedup-appended. Prompt: `no_trade` gains a red-news line ONLY when a live
  T1 slice exists (dynamic, not goldened).

## ✅ W4 — overlays → executor (`5729e2b5`) [A]
`installActivePlanProvider` served the BASE plan doc; owner overlay edits (RFC-6902)
never reached the executor's brain — only the card resolved them.
- `resolveActivePlanDoc` folds overlays into `plan_final` + armors via
  `ValidatePlanDoc` (bad overlay → base doc), the SAME resolution `GET /plan/today`
  does → card and executor can't diverge.
- **Proven:** an overlay that moves a level/bias is reflected in the provider's doc;
  a malformed overlay falls back to base.

## ✅ W5 — learning loop on REAL exits (`e165f35e`) [A]
Excursion/adherence/matched-random fired only on AI `close` decisions — NT8 OCO
SL/TP, EOD-flat, and manual closes (the majority) were never graded.
- `recordClosedTradeAnalytics(p)` (idempotent via `AdherenceGrade != ""`) +
  `maybeRecordClosedTradeAnalytics()` polls ungraded closes since a
  `dayplan_analytics_since` epoch (excludes the 516 historical closes).
  `GetUngradedClosedPositions` is the store query.
- **Proven:** a store-path close (no AI decision) gets MAE/MFE + grade + a
  matched-random verdict; a second pass is a no-op; the epoch excludes history.

## ✅ W6 — alert Emit call-sites (`da3fb458`) [A]
`store.AlertStore.Emit` had ZERO production callers — the P4.4 in-app alert bus was
dead. `emitAlert(level,kind,eventID,…)` (gated · deduped by event_id) fires at:
P0 fill · P0 close+P&L (carries adherence) · P0 read-fail/fail-closed · P0
plan-died→no-trade · P0 halt (deduped per session-day) · P1 armed (plan written).
Scenario triggered/flip intentionally omitted — that state machine doesn't exist
(would be a fabricated event).
- **Proven:** gated-off no-op · gated-on write · dedupe by event_id · per-trader
  ack (IDOR guard).

## ✅ W7 — level-state writer (`77ffbe8f`) [A]
`store.LevelStateStore` (times-tested/consumed/freshness/re-arm) had ZERO callers;
`levels_assemble.go:64` literally says "freshness=nil until the executor writes
level-state (P3.6 loop)" — that loop never existed.
- `recordLevelState()` each bar-close cycle: `EnsureLevel` (identity =
  `kernel.LevelTypeFromLabel` + `LevelBinIndex`, origin_date="" = cross-session
  anchor) · `MarkConsumed` on accept-through · `RecordPlay` on a fresh sweep/reject,
  debounced by `ReArmEligible` (the READER of persisted cooldown). Burned re-touch →
  telemetry + P1 alert.
- **Proven:** accept-through→consumed · consumed never re-arms · burned stays burned
  across sessions + re-touch alert · gated-off writes nothing.
- **Flagged follow-up (NOT silent):** surfacing persisted freshness INTO
  `RenderPlanStatus`/`ScoreLevels` is a deliberate prompt-regression (golden regen,
  both states) — the writer lands now; the prompt-read is the next deliberate step.

## ✅ W8 — gates read the admin registry (`fb5bd624`) [A]
Every gate called `kernel.DefaultSessionRegistry()` directly; the admin registry in
system_config (`session_registry`) was never read (TODO(P4) markers named it).
- `loadStoredRegistry`/`(*AutoTrader).sessionRegistry(now)`: read stored → fail-safe
  to default; cached per CME session-day so an edit is honored NEXT session-day,
  never mid-session. Rewired all trader gates (session, night, planner reads×3,
  ActivePlanProvider, calendar, EOD-flat, adherence, matched-random, level-state).
- `kernel.ValidateSessionRegistry` (an EDIT is refused, not defaulted) +
  GET/POST `/api/plan/session-registry`.
- **Proven:** validator (default ok · empty/no-name/bad-HH:MM/bad-KZ refused) ·
  loader (empty→default · stored honored · malformed→default) · per-session-day
  cache (mid-session hold + next-day pickup). Empty key → byte-identical default.

## ✅ W9 — the 6 display-only config fields wired (`1d476407`) [A]
Six DayPlanConfig fields the FE persists had 0 readers. `auto_trader_planconfig.go`
resolvers (per-session override wins; defaults = shipped → byte-identical):
- `plan_mode` → entry gate (advisory=no-op · direction blocks vs bias · strict
  blocks uncited) · `approval_required` → holds entries until owner grants the
  session-day (`POST /plan/approve` token + P0 alert) · `sessions_enabled` → gates
  reads AND entries (default [NY]) · `proximity_filter_atr` → level activation
  window · `scenario_cap` → post-parse truncation (def 3 = no-op) · `evening_digest`
  → gates the daily roll-up · per-session `plan_mode`/`replan_cap`/`enable`.
- **Proven:** resolvers (defaults · strategy-level · per-session · bounds) ·
  plan-mode gate (all 3 modes + no-plan) · approval gate (grant + session-day scope).

## ✅ W10 — realized-vol baseline supplied (`0f79fb4f`) [A]
Audit #10: the RV-vs-20d baseline was never fed → RV reported "warming" forever.
- `kernel.RVBaselineFrom5m` buckets stored 5m by CME session-day, computes each
  COMPLETE day's RV (same estimator as the recent value), averages ≤20 days;
  <5 days → warming (honest). `assemblePlannerInput` fetches a separate multi-day
  5m series for the baseline (recent RV keeps its ~1-day window); VIX stays 0 →
  honest n/a (no feed, NOT faked).
- **Proven:** warms below 5 days / computes above / nil→warming · ComputeRegime
  clears warming when fed + keeps VIX n/a. Planner REGIME line ("RV=Y%-of-normal"
  once warm) is a dynamic input, not goldened; tests stay warming → no regression.

---

## ✅ FINAL — all 10 wires landed. Full suite: `go build ./...` ✓ · `go vet ./...` ✓
· `go test ./...` **26 pkgs green** (kernel goldens byte-identical — every wire is
default-preserving; no prompt-content change shipped). SIM-only, guardrails master
untouched, additive throughout. HEAD `0f79fb4f`.

## ⏫ DEPLOY (owner, before Mon 08:00 CT — second rebuild, Go-only, no AddOn F5):
```bash
cd /home/hoang/nofx && git pull && go build -o nofx-bin ./... && echo BUILD OK
kill -9 $(pgrep -f nofx-bin)   # systemd Restart=on-failure respawns the new binary
```
Do it in the flat/CME-closed window. No `.cs` changed → no NT8 restart. This
activates W3–W10 (W1+W2 already deployed pre-Sun-17:00). All new behavior is
dormant at default config — the owner opts in per field (plan_mode, sessions_enabled,
approval_required, etc.) via Studio; the admin registry via `POST /plan/session-registry`.
