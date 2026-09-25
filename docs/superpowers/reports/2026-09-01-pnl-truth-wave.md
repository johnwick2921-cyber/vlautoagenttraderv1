# P&L-TRUTH WAVE — the model must never read a fabricated track record (corrected-column law on every prompt-facing surface)

Date: 2026-09-01 · Owner: hoang · Agent: Fable 5 · Worktree: `../nofx-pnltruth` (branch `fix/pnl-truth-wave`) · Checklist: **CLASS 40**
Evidence tiers: **[A]** directly verified · **[B]** inferred from strong evidence · **[C]** speculation.

## STATUS

| Item | State |
|---|---|
| Code | **MERGED to dev @ `23f56f49`** (fix `a6ea66a0` · checklist 40 `dbf7e45e` · report), fast-forward from `713def53` |
| Suites (rebased tree) | `go test ./...` 27 ok / 0 FAIL · goldens PASS · vitest 38 files / 298 pass · tsc clean |
| Sequencing (B) | class 39 booted 00:01:07 CT (rev aeb11179, its marker after its boot, lock released 00:05:36); this wave rebased onto its cutover docs `713def53`, re-ran the full suite (27 ok · goldens · vitest 298 · tsc), took the lock 00:05:47 |
| Cutover | **DONE 00:10:15 CT 2026-09-02** on owner GO (+ explicit hold override) — boot 2 at 00:11:48 `BOOT INTEGRITY OK — rev 23f56f49 · goldens PASS`, PID 2097561; marker `128a3c53` after the passed boot; **LIVE PROOF: decision record 36121 (00:16:32 CT)** carries `Track record: +304.32 over 105 resolved trades (115 unresolved trades excluded — see note).` (see CUTOVER section) |
| Footprint (B) | zero edits to `kernel/plan_doc.go`, `trader/auto_trader_planner.go`, the E8 logger, `plays.ts`, `store/planner_rejected.go` — `git diff origin/dev --stat` names none of them |

---

## C — THE BUG, measured [A]

Live executor prompt, `decision_records` id **36090** (23:07:13 CT, account Sim101), verbatim:

```
8. MNQ short | Entry 29459.0000 Exit 0.0000 | Profit: +0.00 USDT (+100.00%) | 2026-08-31 13:09 CT→2026-08-31 13:13 CT (3m)
…
## Historical Trading Statistics
Total Trades: 220 | Profit Factor: 0.97 | Sharpe: -0.01 | Win/Loss Ratio: 1.60
Total PnL: -203.68 USDT | Avg Win: +68.57 | Avg Loss: -42.84 | Max Drawdown: 9.6%
Performance: NEEDS IMPROVEMENT - improve win/loss ratio, optimize TP/SL
```

The same predicate `GetFullStats` uses (trader `8d5c8af5_…_1781246265`, `status='CLOSED'`, `account='Sim101'`, `close_reason NOT IN (reconcile_flat, unresolved, e7_farside_test)`), recomputed from the raw store (A21):

| figure | value | rows |
|---|---|---|
| rows in the predicate | 220 | ids 237–586 |
| resolved (`pnl_corrected NOT NULL`) | **105** | |
| unresolved (`pnl_corrected NULL`) | **115** | |
| strict sum over resolved | **+304.32** | |
| coerced sum `COALESCE(pnl_corrected, realized_pnl)` | **−203.68** | = the prompt |
| row 526 | raw −1,458.00 · corrected −69.43 (×21 lot-math artifact, `pnl_correction_note` "recorded −1458.00 disagreed with stored prices × row qty by −1388…") | |

Sign, magnitude and count were all wrong in the model's context. Row 8 above is an unresolved short (exit 0): `PnLPct = (entry − 0)/entry × 100 × leverage = +100.00%`.

Dashboard header: `PNL::{account.total_pnl}` (`TraderDashboardPage.tsx:662`) renders the NT8-native total (equity − initial balance = 0.00; `daily_pnl` is `at.dailyPnL`, reset at `auto_trader_loop.go:384` and never written) beside the position-history footer's +212.00. Today's ledger day total, footer rule: **+212.00 over 6 resolved, 0 unresolved, ids 581–586**; NT8 equity **52,216.00 → 52,428.00 = +212.00** (snapshot ids 35540–36211) — the three agree.

## D — LOCATE (quoted before editing) [A]

| aggregator | file:line (pre-fix `91c97dc6`) | column read | NULL handling |
|---|---|---|---|
| `EffectivePnL()` (accessor) | `store/position.go:826-832` | `pnl_corrected` **else raw `realized_pnl`** | **coerces** |
| `GetPositionStats` | `store/position_query.go:25-72` | SQL `COALESCE(pnl_corrected, realized_pnl)` (dead: WHERE excludes NULL) | excluded; `excluded_null_pnl` returned |
| `CountConsecutiveLossesSince` | `:80-101` | `EffectivePnL` (WHERE excludes NULL) | excluded; no count |
| `GetSessionDayActivity` | `:109-140` | SQL `COALESCE(pnl_corrected, realized_pnl)` (dead) | excluded; no count |
| `GetFullStats` | `:146-217` | `EffectivePnL`, **no NULL exclusion** | **coerced** — the live line |
| `GetRecentTrades` | `:236-278` | `EffectivePnL`; no close_reason filter; `PnLPct` from exit 0 | **coerced + fabricated %** |
| `GetSymbolStats` | `:344-404` | `EffectivePnL`, no NULL exclusion | **coerced** |
| `GetHoldingTimeStats` | `:415+` | `EffectivePnL`, no exclusion of any kind | **coerced** |
| `GetDirectionStats` | `:~520-570` | `EffectivePnL` | **coerced** |
| `GetHistorySummary` (+ `calculateStreaks`) | `store/position_history.go:41-135` | `GetFullStats` + `EffectivePnL` | **coerced** |
| AgentBeta `toolGetTradeHistory` | `agent/tools.go:~3350-3450` | **raw `pos.RealizedPnL`** via `GetClosedPositions` | **raw** |
| executor prompt plumbing | `trader/auto_trader_loop.go:1181-1245` (inline in `buildTradingContext`) → `kernel.Context.TradingStats` / `RecentOrders` | | |
| executor prompt render | `kernel/engine_prompt.go:310-322` (recent trades, `Profit: %+.2f USDT (%+.2f%%)`), `:327-371` (`Total PnL: %+.2f USDT`, EN+ZH) · `kernel/formatter.go:135-175, 196-226, 402-445, 470-500` (the `FormatContextForAI` variant) | | |
| dashboard header | `api/handler_order.go:129` `handleAccount` → `trader.GetAccountInfo()` → `trader/auto_trader_decision.go:221-224` (`total_pnl`, `daily_pnl`) · `web/src/pages/TraderDashboardPage.tsx:662` | NT8-native | permanently 0.00 |
| footer rule | `web/src/components/trader/PositionHistory.tsx:135` `computeDayTotal` (kind `normal`, today CT, `pnl_corrected` present) | strict | excluded |
| tests pinning the numbers | `store/test_seam_exclusion_test.go`, `store/pnl_correction_test.go`, `store/consecutive_loss_test.go` (all still pass — their fixtures set `pnl_corrected`) | | |

Consumers whose shapes change: `api/handler_order.go:219/222/269` (stats, symbol stats, trades endpoints — struct JSON gains fields), `store/position_history.go`, `trader/auto_trader_session.go:86` / `trader/auto_trader_planner.go:2138` (call `GetSessionDayActivity` — **signature unchanged**, only its SQL; planner.go is class 39's and was not edited).

## E — THE FIX (file:line on `c738c2ec`) [A]

| file | line | change |
|---|---|---|
| `store/position.go` | 844 | `CorrectedPnL() (float64, bool)` — the strict accessor; 852 `IsUnresolved()`; 833 `EffectivePnL` kept for per-row display, documented as banned from aggregators |
| `store/position_query.go` | 51 | `GetPositionStats`: `SUM(CASE WHEN pnl_corrected > 0 …)`, `COALESCE(SUM(pnl_corrected),0)` — raw column gone; `resolved_trades` added |
| | 92-112 | `CountConsecutiveLossesSince`: `CorrectedPnL`; an unresolved row neither extends nor breaks a streak |
| | 133 | `GetSessionDayActivity`: `COALESCE(SUM(pnl_corrected), 0)` — signature unchanged |
| | 162-236 | `GetFullStats`: WHERE `pnl_corrected IS NOT NULL`; `UnresolvedExcluded` counted under the same predicate (187); `ResolvedTrades`; every figure over resolved rows |
| | 274-330 | `GetRecentTrades`: `ID`, `Resolved`; test-seam rows quarantined; unresolved rows carry P&L 0 / pct 0 / `Resolved=false`; pct only when resolved AND exit > 0 (312) |
| | 391-450 | `GetSymbolStats`: strict + `UnresolvedExcluded` per symbol |
| | 468-540 | `GetHoldingTimeStats`: ledger exclusions added + strict + `UnresolvedExcluded` per range |
| | 548+ | `GetDirectionStats`: strict + `UnresolvedExcluded` |
| `store/position_history.go` | 14, 49, 104-140 | `HistorySummary.UnresolvedExcluded`; recent-window P&L and streaks via `CorrectedPnL` |
| `store/pnl_surface_guard.go` | 24 / 44 / 63 | `PnLSurfaces()` registry (12) · `PnLSurfacesBootLine()` · `GetLedgerDayTotal` (the footer rule, server-side) |
| `kernel/engine.go` | 73-74, 86-87 | `TradingStats{ResolvedTrades, UnresolvedExcluded}`, `RecentOrder{ID, Resolved}` |
| `kernel/engine_prompt.go` | 317 | unresolved row → `N. #id side | Entry x→? UNRESOLVED (exit unknown) | …` |
| | 355 / 381 | EN+ZH stats block: `Resolved Trades: N …` + **`TrackRecordLine`** + note; block renders when resolved>0 OR unresolved>0 (never silent) |
| | 1075 / 1092 | `TrackRecordLine`: `Track record: %+.2f over %d resolved trades (%d unresolved trades excluded — see note).` (0 resolved → `Track record: UNRESOLVED — 0 resolved trades (K unresolved …)`) · `TrackRecordNote` |
| `kernel/formatter.go` | 158, 203, 432, 477 | the `FormatContextForAI` variant: same line, same note, same UNRESOLVED row (EN+ZH) |
| `trader/auto_trader_loop.go` | 1181 → 1326 | plumbing extracted into `attachTradeContext` (strict; logs `🧾 Track record: +X over N resolved (K unresolved excluded)`) |
| `agent/tools.go` | 3395 / 3404 / 3416 | `buildTradeHistory` pure builder: test-seam quarantined, unresolved rows `pnl: null, resolved: false`, summary `resolved_trades` + `unresolved_excluded` + `pnl_column` |
| `agent/planner_runtime.go` | 798-801 | render `Recent trades: +X over N resolved trades (K unresolved excluded), win rate …` |
| `api/handler_order.go` | 159-170 | `/api/account` gains `ledger_day_pnl`, `ledger_day_resolved`, `ledger_day_unresolved`, `ledger_day_date`, `ledger_day_source`; `ledger_day_status: "UNRESOLVED: …"` when uncomputable (never a fabricated 0). NT8 fields kept and labelled |
| `main.go` | 266 | boot line `🧾 P&L surfaces: 12 aggregators strict-corrected, 0 raw (corrected-column guard; unresolved rows counted + excluded, never coerced)` |
| `web/src/types/trading.ts` | 31 | `AccountInfo.ledger_day_*` |
| `web/src/components/trader/LedgerDayPnl.tsx` | new | `LEDGER_DAY::+212.00 (6 resolved, 0 unresolved excluded)` chip; `LEDGER_DAY::UNRESOLVED …` when absent |
| `web/src/pages/TraderDashboardPage.tsx` | 1, 666 | chip beside `PNL::` (NT8 total, now titled) |
| guide | `status.ts:16`, `glossary.ts:95/99`, `faq.ts:47` | boot ledger line · "Track record (pnl_corrected)" · "UNRESOLVED (trade)" · header-vs-footer FAQ |
| `docs/superpowers/AUDIT-CHECKLIST.md` | 445 | **CLASS 40** (highest at merge time was 39, appended by class 39) |

**Definition used for `UnresolvedExcluded`:** NULL-`pnl_corrected` rows INSIDE the ledger predicate (the 0A-2 close_reason filter). Unresolved-REASON rows (`close_reason='unresolved'`) are a separate class already excluded from every aggregate; they still LIST in recent trades as UNRESOLVED. This is exactly the definition behind the live numbers (220 = 105 + 115).

**The lint (E5)** — `store/pnl_surface_guard_test.go:87` `TestPnlSurfaceGuardNoRawAggregation` walks `store/`, `api/`, `trader/`, `agent/`, `kernel/` (non-test `.go`, comments stripped) for `SUM(realized_pnl`, `COALESCE(pnl_corrected, realized_pnl`, `IFNULL(…)`, `+= x.RealizedPnL`, `EffectivePnL()`. Allow-list (file → reason):
- `store/position.go` — defines `EffectivePnL` (per-row display accessor) and WRITES `realized_pnl` at close
- `store/pnl_correction.go` — the correction tooling: reads `realized_pnl` to WRITE `pnl_corrected`
- `trader/auto_trader_clock.go` — per-row close analytics on the row this process just closed (recompute check + watch backfill); not an aggregate
It also asserts every registry entry exists in its file. `TestPnlSurfaceGuardCatchesRawAggregation` (F6) proves the scanner flags a deliberate `s += r.RealizedPnL` and a SQL `COALESCE` fallback, passes an allow-listed path, and ignores comments.

## F — TESTS [A]

**F1 · `TestPnlTruthPinExecutorPrompt`** (`kernel/pnl_truth_pin_test.go:37`) — 3 resolved (+50 −20 +10) + 2 NULL rows (raw −100, −300); `GetFullStats` → the engine's `BuildUserPrompt`. Pre-fix surface only. **RED on `91c97dc6`** (throwaway worktree):
```
--- FAIL: TestPnlTruthPinExecutorPrompt (0.15s)
    pnl_truth_pin_test.go:73: P&L TRUTH: the executor prompt carries a COERCED / bare track record (2 unresolved rows folded in as raw realized_pnl):
        ## Historical Trading Statistics
        Total Trades: 5 | Profit Factor: 0.14 | Sharpe: -0.52 | Win/Loss Ratio: 0.21
        Total PnL: -360.00 USDT | Avg Win: +30.00 | Avg Loss: -140.00 | Max Drawdown: 4.1%
```
**GREEN on `c738c2ec`:** `--- PASS: TestPnlTruthPinExecutorPrompt (0.14s)` — the block reads `Track record: +40.00 over 3 resolved trades (…)`.

**F2 · every aggregator** — `TestPnlTruthGetFullStatsStrict` (+40 over 3, 2 excluded; all-unresolved trader → 0 resolved / 1 excluded / no total), `TestPnlTruthEveryAggregatorExcludesNull` (`GetPositionStats` +40/3/`excluded_null_pnl` 2 · `GetSessionDayActivity` +40 · `CountConsecutiveLossesSince` 0 with the two newest rows unresolved · `GetSymbolStats` · `GetHoldingTimeStats` · `GetHistorySummary`). PASS.
**F3 · `TestPnlTruthRecentTradesUnresolvedRowCarriesNoPnlNoPct`** — the live exit-0 short: `Resolved=false`, P&L 0, pct 0; NULL row with raw −100 never coerced; test-seam row absent; resolved row unchanged. PASS.
**F4 · `TestPnlTruthTradeHistoryToolStrictShape`** (`agent/`) — JSON round-trip: `resolved_trades 1`, `unresolved_excluded 2`, `total_pnl 100`, rows `pnl: null` / `resolved: false` for the unresolved and row-526-shaped rows, test-seam quarantined. PASS.
**F5 · header = footer** — Go `TestPnlTruthLedgerDayTotalMatchesTheFooterRule` (duplicate reconcile_flat, test-seam and unresolved-reason rows excluded; NULL rows counted: +132 / 4 / 2; window + account scoping) and FE `LedgerDayPnl.test.tsx` (chip text `LEDGER_DAY::+212.00 (6 resolved, 0 unresolved excluded)`; `computeDayTotal` on the same fixture = +40 and the chip renders it; no fabricated 0 when the backend has no figure). PASS.
**F6 · lint** — `TestPnlSurfaceGuardNoRawAggregation` PASS on the tree; `TestPnlSurfaceGuardCatchesRawAggregation` PASS (flags 2 deliberate raw sites, passes allow-listed, ignores comments). During development the guard caught `EffectivePnL` in `GetDirectionStats` and the accessor's own definition — the allow-list carries the reason for each exception.
**F7 · recomputed today (A21)** — see section C: +304.32 over 105 resolved, 115 unresolved (ids 237–586, Sim101, ledger predicate); day total +212.00 over 6 (ids 581–586); NT8 equity delta +212.00 (52,216 → 52,428, snapshot ids 35540–36211). Footer, header (post-cutover) and NT8 agree.
**F8 ·** rebased tree: `go test ./...` **27 ok / 0 FAIL** · `go test ./kernel -run Golden` PASS · vitest **38 files / 298 tests** · `tsc --noEmit` clean · `go vet` clean on touched packages. Plus `TestPnlTruthAttachTradeContextRendersTheTruth` (trader): the production seam renders `Track record: +60.00 over 2 resolved trades (1 unresolved trades excluded — see note).`, `#3 MNQ short | Entry 29459.0000→? UNRESOLVED (exit unknown)`, `#4 … UNRESOLVED` for the row-526 shape, and no `Total PnL:` / `-1458` / `100.00%` anywhere.

## G — CUTOVER (pending)

Sequence per B/G: wait for class 39's boot → rebase onto dev → re-run the full suite → clean-clone build (`go version -m`, `vcs.modified=false`) → acquire the lock → stage `nofx-bin.next` → flat gate ×4 (A5) · in-flight (A6) · window + arms (A7) → **owner GO** → swap + `kill -9` → boot checklist incl. **`🧾 P&L surfaces: 12 aggregators strict-corrected, 0 raw`** and the surviving 36/37/38/39 lines → marker AFTER the passed boot (A19) → **proof: the next rendered executor prompt's track-record line, verbatim from `decision_records`** (expected `Track record: +304.32 over 105 resolved trades (115 unresolved trades excluded — see note).` if no trade closes first).

**Rollback (exact):** `cd /home/hoang/nofx && mv nofx-bin nofx-bin.bad.<rev> && cp nofx-bin.prev.boot nofx-bin && kill -9 $(pgrep -f '^/home/hoang/nofx/nofx-bin$')` — `deploy/RELEASE` is untouched until the marker, so a pre-marker rollback needs no RELEASE edit.

## A15 — what the owner will still see wrong

- Until this wave boots, every executor decision still reads `Total PnL: -203.68 USDT` over 220, the `+0.00 (+100.00%)` row, and the header shows `PNL::0.00`.
- After the boot: `PNL::` (NT8 total, 0.00) stays beside the new `LEDGER_DAY::` chip — labelled, not removed (other readers use `/api/account`'s fields).
- The 115 unresolved rows stay unresolved: this wave stops READING the wrong column; it writes nothing (stop-line I) — the backfill is 0A-2's, done.
- `api/handler_plan.go:1685` still echoes raw `realized_pnl` per row in the graded-trades list (per-row, not an aggregate; not in scope). Noted, not changed.
- The vite dev server serves the main tree: the FE chip appears only after the main tree is fast-forwarded (under the lock, at cutover); until then the header is unchanged.

---

## CUTOVER — DONE (owner GO, 2026-09-02) [A]

- **GO 00:09 CT** with conditions: proceed when both holds clear (no read in flight, no working arm), re-quote at swap, hold again if a new arm places. At 00:09:20 both holds were still live (wake re-read in flight since 00:03:07; ASIA v7 S1 leg-0 limit at 29044 `working`). **Owner then ruled "JUST DO IT NO WAIT"** — an explicit override of the A6/A7 holds. Stated consequence, accepted: the in-flight wake re-read dies with the old process (wake reads are non-fatal, `failClosed=false`; the active v7 plan is kept), and the resting S1 limit must be re-adopted across the restart (class 39's cutover had already shown arms surviving a restart).
- **Lock re-acquired 00:10:15 CT** (pid 1860416; no holder present). **A5 at swap:** DB OPEN 0 · API positions `[]` · API open-orders `[]` · NT8 `positions snapshot account=Sim101 count=0` (00:10:04). **A6/A7 at swap (overridden):** `replan_in_flight: true`; armed `S1 … legs[0] working (limit 29044)`. The swap would have aborted on an OPEN position; there was none.
- **Swap 00:10:15 CT:** `cp nofx-bin nofx-bin.prev.boot` (= aeb11179) · `mv nofx-bin nofx-bin.old.aeb11179` · `mv nofx-bin.next nofx-bin` (rev check `23f56f49` first) · `kill -9 2089356`.
- **Boot 1, 00:10:21 CT — REFUSED.** `🔐 BOOT INTEGRITY REFUSED — rev 23f56f49a536 · built 2026-09-02T05:06:06Z · expected aeb11179df5a · goldens PASS` → `🔐 TRADING REFUSED — binary is revision "23f56f49a536" but the intended release is "aeb11179df5a"`. Cause: I had left `deploy/RELEASE` at the running rev, reading A19 ("marker AFTER the boot") as "do not touch RELEASE before the boot". The boot-integrity assertion reads the RELEASE **file** at startup, so a swap without the file edit always refuses. **Correct A19 protocol (as class 39 did it): edit the RELEASE file (uncommitted) BEFORE the swap; COMMIT the marker only after the boot passes; revert the file if it does not.** Every other boot line was already correct on boot 1, including `🧾 P&L surfaces: 12 aggregators strict-corrected, 0 raw` and `🧾 [hoang] Track record: +304.32 over 105 resolved trades (115 unresolved excluded)`. A peer session (`nofx-06`) independently flagged the REFUSED boot at the same minute; the fix below was already in flight and it was told so.
- **Fix + restart 00:11:42 CT:** `deploy/RELEASE` = `23f56f49…`, `GUIDE_BUILT_REV` = same (files only); re-quoted: DB OPEN 0, positions `[]`, open-orders `[]`, `replan_in_flight: false`, S1 leg-0 still `working` (survived restart 1); `kill -9 2096745`.
- **Boot 2, 00:11:48 CT — PASSED:**
  `🔐 BOOT INTEGRITY OK — rev 23f56f49a536 · built 2026-09-02T05:06:06Z · expected 23f56f49a536 · goldens PASS`
  `🚀 planner speed wave … stream_total=1200s (class 37 …)` · `🧪 validator hints: 15 sites … (class 34 + 38 guard)` · `📜 prompt/validator contract: 17 restrictions, all stated in prompt (class 38 guard)` · `⚖ arm normalizer … (class 39)` · `🧮 replan budget: recorded-counter (class 35) …` · `🗓 preflight: scheduled reads bypass freshness in halt/weekend (class 36) …`
  **`🧾 P&L surfaces: 12 aggregators strict-corrected, 0 raw (corrected-column guard; unresolved rows counted + excluded, never coerced)`** ← NEW
  `🧾 [hoang] Track record: +304.32 over 105 resolved trades (115 unresolved excluded), 38.1% win rate, PF=1.08, Sharpe=0.03, DD=9.1%`
  Exactly ONE PID: `2097561`; `go version -m nofx-bin` → `vcs.revision=23f56f49…`; `[ERRO]`/panic since boot: **0**; positions `[]`; S1 leg-0 `working` after boot 2 (survived both restarts).
- **Marker `128a3c53`** (RELEASE + GUIDE_BUILT_REV = `23f56f49…`) committed AFTER the passed boot and pushed (rebased onto a concurrent dev push first).
- **Live surfaces on the new binary (00:12:21 CT):** `/api/account` → `ledger_day_pnl 0.0, ledger_day_resolved 0, ledger_day_unresolved 0, ledger_day_date 2026-09-02` (no trade closed today CT yet — an honest zero WITH its n, not a fabricated one); `/api/trades` rows carry `"resolved": true` (ids 586, 585 …).

### THE PROOF (G) — the next rendered executor prompt, verbatim [A]

`decision_records` id **36121**, created **2026-09-02 00:16:32 CT**, account Sim101 (the first executor cycle on the new binary; the last pre-swap record was 36120 at 00:09:20):

```
8. #579 MNQ short | Entry 29459.0000→? UNRESOLVED (exit unknown) | 2026-08-31 13:09 CT→2026-08-31 13:13 CT (3m)
…
10. #577 MNQ long | Entry 29413.0000→? UNRESOLVED (exit unknown) | 2026-08-31 12:25 CT→2026-08-31 12:29 CT (3m)

## Historical Trading Statistics
Resolved Trades: 105 | Profit Factor: 1.08 | Sharpe: 0.03 | Win/Loss Ratio: 1.70
Track record: +304.32 over 105 resolved trades (115 unresolved trades excluded — see note).
Avg Win: +100.55 | Avg Loss: -59.01 | Max Drawdown: 9.1%
Note: an unresolved trade has no verified exit fill; its P&L is UNKNOWN and is never counted, never coerced to a raw value.
Performance: NORMAL - room for optimization
```

Before (record 36090): `Total Trades: 220 … Total PnL: -203.68 USDT` and row 8 `Exit 0.0000 | Profit: +0.00 USDT (+100.00%)`. After (record 36121): the strict figure, its resolved n and unresolved count, and the two unresolved rows named by id with no P&L and no percentage. The figure agrees with the raw-store recompute in section C to the cent.

### A15 — what the owner will still see wrong (post-cutover)
- The header now shows `PNL::0.00` (NT8-native, unchanged) beside `LEDGER_DAY::+0.00 (0 resolved, 0 unresolved excluded)` — correct at 00:12 CT (nothing closed today); it will move with the first close of the session-day.
- `api/handler_plan.go:1685` still echoes raw `realized_pnl` per row in the graded-trades list (per-row echo, not an aggregate; out of scope, noted).
- The wake re-read in flight at the swap was lost by the override; v7 stays active and the next level event re-wakes it (30-minute wake throttle applies).
- Boot 1's ~80 seconds of TradingRefused (00:10:21–00:11:42) are in the log as two `[ERRO]` lines; no order was attempted in that window (positions `[]` throughout).
- **Correction to "the S1 arm survived both restarts" (00:32 CT):** it survived only as an API state. The arms placed by the OLD process (S1 limit 29044, S3 limit 29068.05) had no `order_update` stream after the restart and were **cancelled at 00:31:48 CT by the stale-window reconcile** (`✕ armed S1 cancelled — no order_update for 15m (reconnect/reconcile)`, same for S3). That is the real cost of overriding the A7 "no cutover with live arms" hold: resting arms authored before a restart are orphaned and cleaned up 15 minutes later, not carried. Meanwhile the new binary is trading normally: position **587** (LONG 1 @ 29079.25, Sim101, cited S3, plan v7) opened at **00:17:44 CT**, one minute after the proof cycle; ASIA is on plan v8 at 00:32.

## Closeout
Commits on dev: `a6ea66a0` fix · `dbf7e45e` checklist 40 · `23f56f49` report · `128a3c53` marker · this addendum. Lock released, worktree `../nofx-pnltruth` removed, repo memory updated (`project_pnl_truth_wave.md`, with the A19 file-before-swap lesson).
