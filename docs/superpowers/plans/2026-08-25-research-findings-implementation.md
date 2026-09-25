# Full Implementation Plan — Research Findings C1–C14 (2026-08-25)

Every item from the 25-agent mega research, in recommended execution order.
Each item: severity · files · exact steps · tests · deploy · tools used.

Legend: tools = [terminal] (build/test/git) · [edit] (file edits) · [db] (read-only SQLite) · [api] (curl + minted JWT) · [deploy] (ritual: build → deploy/RELEASE → kill -9 → verify BOOT + one PID + /api/health).

---

## PHASE 1 — P0 SECURITY (fix first, highest risk)

### C1. Cross-user IDOR on /api/plan/*
**Files:** `api/handler_plan.go` (every handler: today/versions/reset/reread/status)
**Steps:**
1. [edit] Add a helper `requireTraderOwnership(c, traderID) bool` — loads the JWT `user_id` from context, queries `traders` by id, compares `user_id`; returns 404 (not 403 — don't leak existence) on mismatch.
2. [edit] Call it at the top of every `/api/plan/*` handler before any read/write.
3. [terminal] `go test ./api/ -count=1` + add tests: user A's JWT cannot read/mutate user B's trader plan (404).
4. [deploy] Rebuild + restart (flat window), verify BOOT line + health.
**Tool:** edit → terminal → deploy.

### C2. Owner-level cross-user leak
**Files:** `api/handler_plan.go` (owner lookup), `store/owner_level.go`, `store/plan.go`
**Steps:**
1. [db] `sqlite3 "file:data/data.db?mode=ro" ".schema owner_levels"` — verify the table has no user_id column.
2. [edit] Add `user_id` column migration (idempotent `ALTER TABLE` guarded by PRAGMA check) in the store migration set.
3. [edit] `ListActive(symbol)` → `ListActiveForUser(userID, symbol)`; filter all reads/writes by user_id. Backfill: existing rows keep current owner's user_id.
4. [terminal] `go test ./api/ ./store/ -count=1`.
5. [deploy] Rebuild + restart.
**Tool:** db → edit → terminal → deploy.

### C3. getTraderFromQuery global-trader fallback
**Files:** `api/handler_plan.go` / wherever `getTraderFromQuery` lives (grep first)
**Steps:**
1. [terminal] `grep -rn "getTraderFromQuery" api/` to locate.
2. [edit] On "no trader found for this user", return 404 instead of falling back to a global trader.
3. [terminal] Tests.
4. [deploy] Rebuild + restart.
**Tool:** terminal → edit → terminal → deploy.

### C4. Auth on /api/agent/klines
**Files:** `api/server.go` (route registration)
**Steps:**
1. [terminal] `grep -rn "agent/klines" api/` to locate the route.
2. [edit] Move it into the JWT-protected group (same middleware as other authed routes).
3. [terminal] Test that unauthenticated request → 401.
4. [deploy] Rebuild + restart.
**Tool:** terminal → edit → terminal → deploy.

---

## PHASE 2 — TRADING INTEGRITY (HIGH)

### C5. MSS wake cap + same-cycle double-append (S1-1/2)
**Files:** `trader/auto_trader_transition.go` (maybeWakePlannerOnMSS), `trader/auto_trader_planner.go` (maybeRunSessionReadsAt)
**Steps:**
1. [edit] Add `mssWakeCount` per (tradeDate, session) — cap MSS-triggered replans to `replanCap` budget via `MayReplanFrom` before `runPlannerReadWithTrigger("structure_mss")`.
2. [edit] Reorder in maybeRunSessionReadsAt: run the death check FIRST, MSS wake after — so a dead vN never gets double-appended; or skip the death check for versions appended this cycle.
3. [edit] Add `carryOwnerEditsInto` + `warnIfReplanOrphansOverlays` to the MSS wake path (owner overlays must survive).
4. [terminal] `go test ./trader/ -count=1` + new tests (cap exhaustion → NO-TRADE instead of silent append).
5. [deploy].
**Tool:** edit → terminal → deploy.

### C6. Executor-side dead-plan gate (S2-1)
**Files:** `trader/auto_trader_loop.go` (decision cycle), `kernel/plan_lifecycle.go` (reuse PlanDeathOrFlipSinceFresh)
**Steps:**
1. [edit] In the executor cycle, before building the decision prompt: if `day_plan` on and the active plan is dead (same machine evaluation the planner uses) → the executor refuses scenario entries from that plan; log `executor_plan_dead`, telemetry `IncGateBlock`.
2. [edit] After cap exhaustion (plan lifecycle `no_trade`), the executor must NOT trade planless — require an active plan to enter.
3. [terminal] Tests.
4. [deploy].
**Tool:** edit → terminal → deploy.

### C7. Reset baseline trader-scoped (S4-1)
**Files:** `store/strategy.go` (ResetBaselineKey/Set/Get)
**Steps:**
1. [edit] Change key to `dayplan_reset:<traderID>:<tradeDate>:<session>` (mirror ScenarioStatusKey).
2. [edit] All callers pass `at.id`.
3. [terminal] Tests (two traders, one reset, other unaffected).
4. [deploy].
**Tool:** edit → terminal → deploy.

### C8. Entry-rejection reconciliation visibility (S2-2)
**Files:** `trader/auto_trader_orders.go`, `trader/ninjatrader/tcp_trader.go` (placeEntry result), `trader/auto_trader_loop.go` (reconcile)
**Steps:**
1. [edit] When NT8 placeEntry returns a rejection (not ACK), log ERROR + emit P1 alert + mark the pending entry as rejected in the position map (no phantom position).
2. [edit] Reconcile loop: if a "position" exists in ctx but NT8 never confirmed it within N seconds → drop it and alert, instead of waiting for close events.
3. [terminal] Tests.
4. [deploy].
**Tool:** edit → terminal → deploy.

---

## PHASE 3 — MEDIUM (correctness + robustness)

### C9. Re-read races + carryOwnerEditsInto (S1-5, S4-2)
**Files:** `trader/auto_trader_reread.go`
**Steps:**
1. [edit] `ForceReread`: capture the claim result; if lost → honest Note (like ForceReset).
2. [edit] After a successful re-read append, call `carryOwnerEditsInto`.
3. [terminal] Tests.
4. [deploy].

### C10. Calendar past-date freeze (S4-4)
**Files:** `store/calendar.go` (UpdateLiveSliceIfChanged)
**Steps:**
1. [edit] Add `trade_date == today` (CT) guard: never overwrite past-date forexfactory rows.
2. [terminal] Tests.
3. [deploy].

### C11. BarCache timeframeMs table (S3-1)
**Files:** `market/bar_cache.go` (or provider/ninjatrader equivalent)
**Steps:**
1. [edit] Add missing entries: 6h/8h/12h/3d/1w (and 1w) timeframeMs mappings so open-stamp shift + CloseTime stay correct.
2. [terminal] Tests (per-TF close-time math).
3. [deploy].

### C12. AddOn cursor persistence (S3-2)
**Files:** `ninjascript/VLBarsSubscriptionManager.cs`
**Steps:**
1. [edit] Persist the dedup cursor across Go reconnects (static field or file) so a reconnect does not re-emit 2000×14 bars and overwrite fresher live bars.
2. [deploy] Copy to `C:\Users\hoang\Documents\NinjaTrader 8\bin\Custom\AddOns\` → F5 compile in NT8 → full NT8 restart (HARD RULE — editing the repo file alone does nothing).
3. [terminal] Verify Go side: no re-emit storm in journal after reconnect.
**Tool:** edit → deploy (NT8 dance) → terminal.

### C13. Zone size ÷ ATR grading axis (A5)
**Files:** `kernel/levels_score.go` (zoneEvidence), `kernel/levels.go` (DetectedLevel already has Lo/Hi)
**Steps:**
1. [edit] Add a size multiplier to the zone score: `sizeMult = clamp((Hi-Lo)/atr, 0.5, 1.25)` banded (small tight base = stronger per S/D literature; oversized zone = weaker).
2. [edit] Thread `atr` into `zoneEvidence`/ScoreLevels (already have dATR; use 14-period ATR of the detecting TF when available, else dATR/20).
3. [terminal] Tests + goldens (goldens capture the executor block — regenerate if the rendered output changes; boot integrity MUST stay green).
4. [deploy].
**CAUTION:** boot-integrity goldens — any change to rendered plan blocks requires golden regeneration in the SAME commit, else trading bricks.

### C14. Confluence cap at 3 (prior audit)
**Files:** `kernel/levels_score.go` (ScoreLevels)
**Steps:**
1. [edit] `confEff := min(conf, 3)` in both score formulas (diminishing returns per research).
2. [edit] Make the cap a config knob (`CONFLUENCE_CAP`, default 3) — no hardcode rule.
3. [terminal] Tests + goldens.
4. [deploy].

---

## PHASE 4 — DEFERRED (owner gate, replay-data validation)

- VWAP advisory level (R20) — only after ≥100 VWAP-conditional trades.
- RV buckets / CHOP regime gates — only after ≥200 trades, in/out-of-sample.
- Grade calibration by outcome expectancy (A9) — needs per-grade trade attribution.
- EOD-flat offset verification (S1-2 MED-HIGH): confirm the live strategy's `eod_flat_offset_min` intent (14:30 vs contracted 14:45 CT flat).
- SQLite VACUUM INTO backups already in place; add integrity_check to the daily timer.

---

## Execution order & deploy cadence
1. Phase 1 (C1–C4) — one deploy, all four together (security batch).
2. Phase 2 (C5–C8) — one deploy each (behavioral, verify journal after each).
3. Phase 3 (C9–C14) — C9–C12 one deploy; C13–C14 graded carefully with golden regeneration.
4. Every deploy: full `go test ./...` green → build with `-ldflags "-X main.buildRevision=$RELEASE"` → `deploy/RELEASE` → `kill -9` → verify `🔐 BOOT INTEGRITY OK` + one PID + `/api/health` → commit.
