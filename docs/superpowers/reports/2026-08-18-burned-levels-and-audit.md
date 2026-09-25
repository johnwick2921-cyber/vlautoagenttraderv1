# COMBINED TRAIN — FALSE BURNED LEVELS + REMAINING AUDIT FINDINGS (2026-08-18)

**LINE 1:** 17 false burns purged (12 pre-H10-interval, 5 pre-window) · **🔐 BOOT INTEGRITY OK — rev 1c418a1f2338 +dirty · built 2026-08-17T15:22:10Z · expected 1c418a1f2338 · goldens PASS** · bot PID 955130.

**Root cause [A]:** `recordLevelState` fed the FULL ~33h 1m cache to acceptance counting with NO window and NO touch gate — any level price merely sat beyond read "accepted through" and burned within minutes. 08-17 NY: 12/12 levels `done|1` (7 written 13:11 UTC pre-H10 2×1m bug, 5 post-H10 14:36 UTC still windowless). The old binary burned 5 more by 15:21.

**Part 1 — fixes (commits 1a95a444 · 5fe152d7 · a85d7363 · 3b7a1d7c · 6a0cf41d):**
- 1c: consumption = in-window TOUCH + N rule-TF closes beyond, judged only on bars since the row's birth (`kernel.ConsumedSince`/`BarsSince`/`LevelTouchedOn`); wick ≠ consume; 2×1m ≠ 2×5m.
- 1c: consumed levels ROLE-FLIP and stay seated (`freshMult 0→0.5`, label `flipped`, PLAN STATUS says "watch the trap") — no longer deleted.
- 1d: aging — burned → `done` this CME day → `C` next 1–2 days → `B` from day 3 (`store.AgedFreshness`).
- 1b: `cmd/dayplan-level-repair` (dry-run default); executed after deploy: backup `~/nofx-backups/level-repair-2026-08-18/` → 17 rows reset to C|0; the new binary re-evaluates honestly from bars each cycle. Verified post-state: `C|0|17`.
- 1e: 8 new tests (wick/touch/window/interval, aging, reset, W11b role-flip, W7 windowed).

**Part 2 (8559ea0c · 65834726 · 9c10c149 · 1c418a1f):**
- 2a: `MakePlanIDForTrader` + `ResolvePlanID` (chain continuity); plans unique key → (trade_date, session, strategy_id, version); legacy rows keep working.
- 2b: `PruneAckedOlderThan` wired (acked P2 >7d, 1×/CME day); dead HandoverBanner deleted (lifecycle expired|died|superseded never written — no_trade/active only).
- 2c: `ai_latency_ms` mirrored at both write sites (UI reads it); DisciplinePanel = minimal FE consumers for /plan/trades + /plan/stats.
- 2d: H8 residuals verified gone (no `sess.Enabled` in api; scan tests pass).
- 2e sweep at new head — 3 NEW instances fixed: planner-read claim key now trader-scoped (D4), level-state maxLevels from config (D2), risk_check_passed/prompt_version/ai_model now stamped honestly (D3). Reported-not-fixed (benign/latent): first-trader closure in symbol-keyed providers; legacy plan-id comparator (compat); ListVersions/RecordTest unused; FillPrice/FillLatencyMs writers still absent (fill facts live on order rows).

**Exit bar:** `go build/vet/test ./...` green; `-race` green (kernel/trader/store/api); `tsc` clean; vitest 243/244 (2 pre-existing: RegistrationDisabled logo, e2e/gate.spec.ts); goldens untouched + PASS. Deployed by me per standing rule: flat (0 OPEN / 517 CLOSED), other session idle 8h+, RELEASE=1c418a1f2338, kill -9 → systemd relaunch, boot line receipt above, one PID, trader hoang cycling (38 trades, stats logged).

**Still open (stated, not claimed):** B3/B4/C3 fixtures remain absent from the repo — the bypass-proof chain is verified structurally, NOT by the cited reproducible fixtures; nothing here changes that.
