# S-LIST CLOSER WAVE — 3 fixes built + proven, NOTHING deployed (2026-08-27)

**Branch `fix/slist-closer` off `origin/dev` (cf4b182c) · worktree `~/nofx-slist` · deployed rev UNTOUCHED at c21ad24a**
HARD RULE honored: zero deploys, zero restarts, zero config/env writes to the live bot. The running process was never signaled; `data/data.db` was only READ (the nightly proof ran against a sqlite `.backup` copy inside the worktree, since deleted).

---

## PER-FIX LEDGER

### FIX 1 [S] — EOD armed-order race window (deep-verify hole 11)
**Root cause:** `enforceEODFlatAt` flattened POSITIONS only and returned true → the cycle skipped `maybeManageArmedOrders`, so the armed cancel ran on the NEXT cycle — a working limit could fill up to one 2m cycle after the flat.

**Fix:** new `cancelArmedOrdersSync` — per working row: send the NT8 cancel frame, then drain the SHARED order_update stream (never a second subscribe — that would close the consumer's channel) through the same `onArmedOrderUpdate` ledger machine until the ack/terminal state lands or the deadline passes; one retry; ledger flips cancelled regardless; unacked → loud WARN and the flatten proceeds (never held hostage). Positions are re-read after the cancels so a fill that won the race mid-cancel is also flattened. Timeout env knob `ARMED_CANCEL_ACK_TIMEOUT_MS` (default 2000ms).
**Callers ordered cancel-first:** (a) `enforceEODFlatAt` in-session branch, (b) its no-active-session branch (session end), (c) `maybeManageArmedOrders` session-end/dormancy branch, and — folded in, same race class — (d) `enforceT1ForceFlatAt` (red-news). Folded hygiene: `armedUpdateStream` replaces the `LoadOrStore(nt.OrderUpdates())` eager-arg subscribe pattern in the consumer.

**Fixtures (all green, `trader/slist_eod_race_test.go`):**
- `TestSListEODFlatCancelsArmsBeforeFlatten` — 14:30 CT (in-session branch, 15-min offset override; log proves the branch: `🕒 EOD-FLAT (14:30 CT (NY)`), open position + working arm, **twin long/short**: every `cancel:sig-eod` wire frame strictly precedes `close_long:MNQ`/`close_short:MNQ`; ledger terminal; a late `filled` frame does NOT resurrect the cancelled row.
- `TestSListEODFlatCancelAckTimeoutStillFlattens` — silent ack stream → exactly 2 cancel attempts (retry), flatten proceeds, row cancelled with reason `… (ack timeout — flatten proceeds)`, and the live WARN path fired: `⚠️ armed sync cancel UNACKED S1 signal=sig-eod after retry — ledger cancelled, flatten proceeds` + `⚠️ EOD-FLAT: 1 armed cancel(s) unacked after retry — flattening anyway (wire reconciles next cycle)`.
- `TestSListDormancySyncCancelsWorkingArm` — dormant lifecycle + working arm → a WIRE cancel via the sync path (pre-fix code issued no wire frame here), ledger terminal.
- `TestSListSessionEndSyncCancelsWorkingArm` — 15:30 CT between-session gap + position + working arm → cancel strictly before flatten.
- `TestSListSyncCancelSkipsOtherTraders` — another trader's working row untouched.
- Register cell #14 (session-ended × working order) is now DEFINED: cancel-before-flatten.

### FIX 2 [S] — nightly level_stats evaluates 0 (the T1 saga ends here)
**Root cause (binding-class bug, not the lookup):** `ListVersionsForTrader` matches `(trade_date, session, strategy_id)` and the hoang trader's rows exist — the lookup was fine. The WIRING was wrong: a process-global `sync.Once` let whichever trader constructed first own the nightly job. At boot the non-running **"15m"** trader constructs first (journal 21:27:36 `Loading trader 15m` → `Using NinjaTrader`), its strategy_id has zero plan rows, so every night printed `2026-08-26/NY|ASIA|LONDON no plan versions — skipped` for the whole process. The backfill CLI worked because it passes the hoang trader id explicitly.

**Fix:** per-trader idempotency key (`levelStatsWired sync.Map`, pure `wireLevelStatsForTrader`) — each trader wires its own nightly loop and evaluates ITS OWN plans. `runLevelStatsDayAt` now returns the evaluated count and takes an injectable clock.

**Fixtures (green, `trader/ninjatrader/level_stats_nightly_test.go`):**
- `TestLevelStatsNightlyPerTraderWiring` — trader-A once, trader-B gets its own job (the pre-fix global once denied exactly this).
- `TestLevelStatsNightlyEvaluatesSeatedRows` — hermetic: seeded bars + plan row → `runLevelStatsDayOnce` evaluates 1 seated level into `level_stats`.
- `TestLevelStatsNightlyProofDB` — **DB-copy proof** (env-gated; live DB untouched): copy via `sqlite3 ".backup"`, the day's rows cleared from the COPY, real `runLevelStatsDayAt` replayed with pinned clock:
  - `PROOF plan lookup: 2026-08-26/NY → 8 version(s) (pre-fix nightly saw 0)`
  - `PROOF plan lookup: 2026-08-26/ASIA → 10 version(s) (pre-fix nightly saw 0)`
  - `PROOF plan lookup: 2026-08-26/LONDON → 15 version(s) (pre-fix nightly saw 0)`
  - `PROOF nightly replay: pinned now=2026-08-27T22:00:00 day=2026-08-26 trader=8d5c… evaluated=18 rows before=56 after=74` → **18 rows written by the nightly path itself** (56→74; 74 == the live table's backfill count — same evaluation, now reachable via the nightly).

### FIX 3 [XS] — INGEST_QUEUE_CAP 1024 → 4096
**Trigger:** 2026-08-27 21:42 flood touched 1024/1024 (1 intrabar drop). Default cap raised to **4096** (env override intact). The `peak_depth` 1-line/min summary now also fires on the CLEAN path (rate-limited heartbeat), so a zero-drop reopen still proves its peak in the journal.

**Fixtures (green, `provider/ninjatrader/tcp_server_ingest_burst_test.go`):**
- `TestIngestQueueCapDefault4096` — default 4096, env override honored, junk → 4096.
- `TestIngestBurst600FpsUnderCapNoDrops` — synthetic 600-frames/s × 5s = 3000 frames with a fully STALLED drainer: 0 intrabar/current/historical drops, `closes_dropped = 0`, `peak_depth ≥ 3000` sampled (the capacity alone absorbs the burst).
- `TestIngestBurstOverCapCountsIntrabarOnly` — 5000 frames: exactly 904 drop-OLDEST events (sent − cap), 0 drop-current, **0 closes dropped**, peak = 4096 (the honest-counter contract).

### SWEEP — foldable items beyond the 3
- **Folded:** T1 force-flat cancel-first (same race class as FIX1); consumer eager-subscribe hygiene (`armedUpdateStream`); register cell #14 now DEFINED.
- **Stays EVENT-WAIT:** live armed-fill chain (#8), modify_bracket BE+40 (#6/#9), STRICT-era open proposal.
- **Out of scope tonight:** partner `.cs` drift (needs the partner repo + owner push). The 15m trader's nightly now logs per-trader (honest, zero-row, harmless).

---

## GATES — ALL GREEN AT BRANCH HEAD
- `go test ./...` — **PASS, zero failures** (full suite).
- Goldens — **PASS, no diffs**: `TestFuturesKeyLevelsGolden`, `TestFuturesPlanGolden`, `TestFormatKlineTimeframes_Golden` (7 subtests). No prompt-render surfaces touched.
- FE `tsc --noEmit` — **0 errors** (local toolchain; the worktree symlinked the main tree's `node_modules`, removed after).
- FE `vitest run` — **33 files / 277 tests passed**.
- FE `npm run build` — **✓ built** (only the pre-existing chunk-size advisory).
- `gofmt` clean · `go vet` clean on all touched packages.

## CUTOVER STATUS
**Ready for a single cutover** (one deploy covers all three) — awaiting owner review and an explicit "go cutover". Nothing deployed, the live bot stays on c21ad24a untouched. Suggested E-proofs post-cutover: boot line + `evaluated N>0` at the next nightly run; `peak_depth < cap` line after the 17:00 reopen; first EOD flat with an armed order.
