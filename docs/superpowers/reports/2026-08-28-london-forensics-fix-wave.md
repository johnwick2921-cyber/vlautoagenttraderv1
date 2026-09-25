# LONDON-FORENSICS FIX WAVE — F1–F6 built & proven, NOTHING deployed (2026-08-28)

**Branch `fix/london-forensics` off `origin/dev` (3023281c) · worktree `~/nofx-lfx` · deployed rev UNTOUCHED (67d2d10e)**
Zero deploys, zero restarts, zero config/env writes. Live DB only READ (the F2 integrity check queried bars read-only).

---

## PER-FIX LEDGER

### F1a [S] — planner completion cap + loud truncation
- `AI_PLAN_MAX_TOKENS` (default **65536** = 2× the observed 32768-token ceiling; provider max 393216) applied per planner call via new `mcp.ApplyMaxTokens` (set + defer-restore — the shared executor client's cap is untouched; same precedent as `ApplyThinking`).
- Truncation is now planner-aware: `mcp.LastFinishReason` added; the planner logs `📐 planner output TRUNCATED by the provider (finish_reason=length, cap=N) — retrying at the same cap will not fix truncation` instead of a bare parse-reject; the existing per-response `🚨 finish_reason=length … TRUNCATED at N completion tokens` WARN (mcp/client.go) already counts every occurrence.
- Fixture: `TestApplyMaxTokensSetAndRestore` — set→65536, restore→32768, non-Client no-op, empty finish-reason. **PASS.**

### F1b [S] — rule alias completion at parse
- Audit (journal 2026-08-23→28): `flip.rule "2x5m_close" invalid` ×15, `confirm.rule "5m_close" invalid` ×2 — exactly the 02:03/02:08/02:23 rejects.
- `kernel.NormalizePlanDocRules` now runs at the top of `ValidatePlanDocWithCaps` (the single chokepoint every parse path funnels through): confirm `5m_close/5m-close/5mclose/1x5m→1x5m_close`, `15m…→15m_close`, `2x5m/2x_5m→2x5m_close`; flip/death `2x5m_close/2x_5m/2x5→2x5m`, `1x5m…→5m_close`. Unknown spellings pass through and are still rejected honestly.
- Fixtures: `TestPlanDocRuleAliasesNormalizeAndPass` (the exact 02:03/02:08 failing shapes re-validated → pass, normalized values asserted), `TestPlanDocRuleAliasUnknownStillRejected`, `TestPlanDocRuleAliasFullVocabulary` (26 spellings). **All PASS.**

### F2 [S] — CLOSES ARE SACRED (persist queue)
- Root cause of the 06:09 event: the bounded persist queue dropped close-carrying batches when the GORM single-connection writer stalled (`closes_dropped=8`).
- `fanOutBarPersist`: queue-full now **bounded-blocks (2s × 3) for close-carrying and historical batches** — backpressure to the ingest drainer (the socket read loop never sees it), never a silent drop; only after 6s does a close batch drop, and then it shouts **ERROR** with the count. Intrabar-only batches keep the old instant-drop+counter path. Queue cap raised 1024 → 4096.
- **Integrity re-check of the 8 lost minutes:** 1m MNQ bars 05:55–06:25 CT = **30 contiguous bars, zero gaps** — the cache-tail self-heal already covered them; no backfill needed.
- Fixtures: `TestFanOutClosesSurviveWriterStall` (cap+20 close-bearing frames vs a stalled writer → `closes_dropped=0`, `queue_drops=0`), `TestFanOutClosesLastResortIsHonest` (stall beyond the retry window → exactly 1 close dropped, counted), updated `TestFanOutBarPersistQueueFullDropsCounted` (forming bars only). **All PASS.**

### F3 [S] — lineage on materialization (+ #567 repair)
- `StampArmedLineageIfMatched` called at the reconcile materialization site: matches the armed ledger's FILLED row (trader + side + |entry−fill| ≤ 1 tick) and stamps `plan_id/version/band=armed_fill/scenario`.
- `RepairArmedLineage` (idempotent) runs once at `StartPositionReconcile` and back-fills every `plan_version=0` position — this repairs #567's class at cutover boot (the live row itself is a write reserved for the cutover; proven on the fixture DB).
- Fixtures: `TestRepairArmedLineageStampsMaterializedPosition` (row-5's shape → version=12, scenario=S1, band=armed_fill, plan id stamped; second pass stamps 0), `TestStampArmedLineageSkipsMismatchedEntry` (121-pt mismatch → no stamp). **All PASS.**

### F4 [A] — arm authoring quality
- **Prompt:** the ARMED ORDERS mandate gains the FEASIBILITY CONTRACT: `arm{}` MUST meet R:R ≥ ARM_MIN_RR **and** stop ≥ 1×ATR5m, cite the live session ATR, omit `arm{}` if both can't hold. (Goldens unaffected — all 5 golden tests PASS, no diffs.)
- **Write-time WARN:** `kernel.ArmFeasibilityWarnings` (pure, mirrors the gate math) called at the write site with the live ATR5m → `⚔️ arm feasibility: … (WARN — write proceeds; the gate-at-arm chain enforces)`. Never fails a write.
- **Executor dedup:** `armRefusalChanged` — the `⚔️ arm REFUSED` line prints once per (plan, version, scenario, verdict) and stays silent until the spec or reason changes (kills the ~120 lines/session).
- Fixtures: `TestArmFeasibilityWarningsMatchTheLiveRefusals` (the three live-refused LONDON arms all warn; the ASIA v12 arm that filled does not), `TestArmRefusalChangedDedup` (first-log / identical-silent / reason-change-logs / version-change-logs). **All PASS.**

### F5 [XS] — claw402 retry noise
- `x402WarnThrottled` (per-key, 1 line/hour) applied to all 6 `Payment expired (402)` / re-sign-failed / no-header WARN sites (both payment flows). Retries still run; only the log is throttled.

### F6 [note] — planner no longer stalls the executor loop
- **Verified:** the W6/MSS wake re-reads were ALREADY async (`go func` in `auto_trader_wake_levels.go` / `auto_trader_transition.go`). The two synchronous in-cycle blockers — the FIRST read and the DEATH re-plan (the 02:03/02:08 483s/311s calls that produced the 19m33s overrun) — are now async with the same pattern; the plan store's single-writer queue serializes the writes.
- Test updates: `TestP0BAsiaReadFiresAt1655WhileMarketClosed` / `TestP0BAsiaReadDoesNotFireOutsideItsWindow` now poll the store for the async write (contract preserved). **PASS.**

---

## GATES — ALL GREEN AT BRANCH HEAD
- `go test ./...` — **zero failures** (full suite; the only touch-ups were the two P0B tests re-synced to the async read).
- Goldens — **PASS, no diffs**: `TestFuturesPlanGolden`, `TestFuturesKeyLevelsGolden`, `TestFormatKlineTimeframes_Golden`, `TestFvgValidateGolden`, `TestVerifyPromptGoldensPasses`.
- FE — `tsc --noEmit` **0 errors** · `vitest` **33 files / 277 tests** · `npm run build` ✓.
- `gofmt` clean on every changed file · `go vet` clean on `trader/ kernel/ provider/ninjatrader/ store/ mcp/`.

## CUTOVER STATUS
**Ready for a single cutover** — one deploy covers all six. Awaiting the owner's explicit "go cutover". Post-cutover E-proofs: planner calls log `cap=65536` and no new `finish_reason=length`; next wake re-read lands a new version instead of freezing; `closes_dropped` stays 0 through the next stall; #567's row carries version 12 / band `armed_fill`; the REFUSED spam is one line per arm-spec.

---

# CUTOVER RECORD (2026-08-28, owner-acked ×2)

## Swap 1 — main wave (07:28:10 CT)
- Flat-gate ALL-ORIGIN PASS: DB `OPEN=0` · NT8 `positions snapshot account=Sim101 count=0` + `account=SimAccount1 count=0` @ 07:27:27 · API `/api/positions` `[]` ×2 @ 07:27:57 · armed_orders non-terminal=0.
- `nofx-bin` 67d2d10e → `nofx-bin.prev.londonfix`; new binary `b4dc8345` (clean-clone build, vcs.revision == dev tip); `deploy/RELEASE=b4dc8345`; marker commit pushed; `kill -9 3299799`.
- Boot block (07:28:15, PID 3433125):
  - `🔐 BOOT INTEGRITY OK — rev b4dc8345905d · built 2026-08-28T12:25:25Z · expected b4dc8345 · goldens PASS`
  - `🛑 min-sl guard: atr_mult=1.0 level_clearance=2tick(s)`
  - `🧬 plan lifecycle: hysteresis=buffer0.5×ATR14 confirm=2close(s) · flip/death→dormant+auto-rearm … exec_reasoning=fast→low plan_reasoning=max`
  - `⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off arm_rr=2.0 (gate-at-arm only; market-entry floor 3.0 unchanged)`
  - `📐 planner cap: plan_max_tokens=65536 (AI_PLAN_MAX_TOKENS; default 65536) · truncation → 🚨 WARN, never silent`
  - `✅ bars integrity OK: dups=0 tfs=1m total=14622`
- **Defect found post-boot:** the `🩹 RepairArmedLineage` line was absent — the startup repair CALL never landed in commit 0085a2b7 (materialization-time stamp was live; the #567 back-fill call was missing). Fix committed as `2738d158` ("hotfix: wire RepairArmedLineage into StartPositionReconcile"), full `go test ./...` green at that sha.

## Swap 2 — hotfix (07:39:16 CT, owner "GO HOTFIX")
- Flat-gate: DB `OPEN=0` · NT8 `count=0` ×2 @ 07:38:27 · API `[]` @ 07:38:44 · one non-terminal arm: row 7 LONDON S4 `state=armed` (authorization only, no signal → nothing resting on the wire; survives the restart in the ledger — the protocol WAIT clause targets `working` arms).
- `nofx-bin` b4dc8345 → `nofx-bin.prev.londonfix2`; new binary `2738d158` (vcs.revision == dev tip, clean); `deploy/RELEASE=2738d158`; marker commit pushed; `kill -9 3433125`.
- Boot block (07:39:21, PID 3441452):
  - `🔐 BOOT INTEGRITY OK — rev 2738d158ee58 · built 2026-08-28T12:32:09Z · expected 2738d158 · goldens PASS`
  - **`🩹 RepairArmedLineage: stamped 1 position(s) with their armed-fill plan linkage (the #567 class)`**
  - `🧬 plan lifecycle …` · `⚔️ armed_orders=on … arm_rr=2.0 …` · `📐 planner cap: plan_max_tokens=65536 …` — all intact.

## #567 LINEAGE PROOF (fresh DB query @ 07:40)
`id=567 · plan_version=12 · cited_scenario_id=S1 · plan_band=armed_fill · adherence_grade=B · plan_id=2026-08-27:ASIA:8d5c… · plan_session=ASIA`
The F grade was cleared by the repair and W5 regraded the close with the armed-fill plan in hand → **B**. Grade ≠ F ✓.

## POST-BOOT SANITY (first minutes)
- `/api/status` 200 · `/api/positions` `[]` · cycles resuming (cycle #1 at boot, cadence healthy) · `✅ bars integrity OK: dups=0 tfs=1m total=14644` · ingest `intrabar_dropped=0 … peak_depth=0/4096`.
- `🧠 planner model: empty binding → using primary, pinned "deepseek-v4-pro"` @ 07:30:15 (CONFIRM-1 live evidence — no legacy alias on the wire; `deepseek-reasoner` exists only in the deprecated claw402 payment route table).
- CONFIRM-2: the 32768 ceiling was OUR `.env AI_MAX_TOKENS=32768` → `mcp/config.go:60` `getEnvInt("AI_MAX_TOKENS", 32768)`; provider probed ceiling 393216 accepted.
- One notice (not a regression): `🚨 CLOCK EARLY-WARNING [session-roll:LONDON]: |drift| 37755ms` — pre-existing WSL2 time-sync machinery, log-only, under the 60s tolerance.
- The LONDON S4 `armed` row survived the restart in the ledger (id 7, still `armed`).

## PENDING E-PROOFS (tonight)
1. **17:05 CT** — nightly level_stats solo night #2 (evaluates 08-27) → `evaluated N>0`.
2. **17:00 CT Globex reopen** — `peak_depth < 4096` AND `closes_dropped=0` through the flood.
3. Next planner read — `cap=65536` in the log, zero `finish_reason=length`, wake re-reads succeeding (the London-freeze class dead).
4. First session-end with a WORKING arm — cancel-first wire proof (still the FIX-1 pending E-proof).
