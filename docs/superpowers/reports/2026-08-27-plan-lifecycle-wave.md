# PLAN-LIFECYCLE WAVE — HYSTERESIS + REARM + 5 DROUGHT FIXES (2026-08-27)

Branch `fix/plan-lifecycle` · 8/8 fixes SHIPPED in one cutover · base: merged
dev == running (verified pre-cutover: `deploy/RELEASE=717acd34e5`, binary
`717acd34e52b`, dev tip `1a9dd62a` docs-only). Cite F# from
`docs/superpowers/reports/2026-08-27-london-drought.md`.

## 1. The 8 fixes

| # | Fix | Commit | Mechanism |
|---|-----|--------|-----------|
| 1 | FLIP/DEATH HYSTERESIS | `5ab02bce` | `kernel/plan_lifecycle.go`: close-based only (wick-through never counts), line buffered by `FLIP_ATR_BUFFER` (default 0.5)×ATR14(5m) in the protective direction for BOTH sides, `FLIP_CONFIRM_CLOSES` (default 2) CONSECUTIVE decision-TF closes beyond. |
| 2 | DEACTIVATE-AND-REARM | `32a78819` | flip/death-line hit → `UpdatePlanLifecycle(dormant)` on the SAME row (no new version, no budget); auto re-arm when price closes back on the valid side (same hysteresis, mirrored buffer) after `DORMANT_MIN_HOLD_MIN` (default 5, flap guard; dormancy timestamp in system_config). `replan_cap` now only gates legacy-consumption deaths / rereads / AI-quality fail-closed. |
| 3 | LATENCY ROUTING | `6e008a11` | executor in-loop `AI_EXEC_REASONING` (default fast → wire effort `low`) vs planner `AI_PLAN_REASONING` (default max); re-asserted per call (`mcp.ApplyThinking`), mode + latency logged per call. Boot line states both. |
| 4 | [F2] HONEST C6 | `8f4dc7e4` | `GateRefusalError` sentinel from the C6 gate; `callWithSchemaRetry` returns it immediately (NEVER the parse-error retry loop); mapped to `SkipReason=executor_plan_gate:…` → `risk_check_error` logged. Planless cycles get `⚠ NO ACTIVE PLAN — entries will be refused (C6)` appended to the user prompt (the old warning lived only in the PLAN STATUS tail, which is absent when planless). |
| 5 | [F3] DORMANT KEEPS EYES | `32a78819` | dormant AND no_trade plans now run `maybeWakePlannerOnLevelEvents` — level touches wake the PLANNER for a fresh read (new version); the dormant row is never flipped active (re-arm only via the close-back predicate). |
| 6 | [F5] BARS INTEGRITY | `2b8e9756` | root cause: `INSERT OR IGNORE` against a table whose gorm PK only exists on fresh DBs → 17,695 duplicate revisions. Fix: one-shot `DELETE … WHERE rowid NOT IN (SELECT MAX(rowid) GROUP BY natural key)` + `bars_pre_dedupe_<date>` safety copy + REAL unique index; writer → `ON CONFLICT … DO UPDATE` (revision wins); 1m-only storage (5m/15m DERIVE-ON-READ — no reader reads stored aggregates); nightly integrity check in `pruneLoop` (dups=0 + tfs=[1m], WARN on drift). |
| 7 | [F1] TRIGGER FALSE POSITIVES | `2fdd5850` | scenario-trigger evaluation windowed to plan birth (`kernel.BarsSince(bars, plan.BirthMs)`) so pre-plan sweeps/rejects can't read as "triggered now"; the log line is relabeled `🎯 scenario → ≈status (display-only estimate, never execution-wired)`. |
| 8 | [F4+F6] ATTRIBUTION + bias_ctx | `2fdd5850`+`8f4dc7e4`+`8692a3ea` | backfill script now uses session-INSTANCE chain dates (`PlanChainTradeDate` semantics — the old 17:00-roll rule stamped #562 with yesterday's plan) and re-evaluates mis-linked rows; executor `bias_ctx` PDC now stamps from the FULL detector universe via `ApplyUniverseDayAnchors` (same family as the bias-tree fix). |

## 2. Test expectation changes (old → new, per ruling)

| Test | OLD | NEW |
|------|-----|-----|
| `kernel/plan_condition_rule_test.go` `TestFiveMCloseDeathFiresOnOneClose` | 5m_close death fires on ONE 5m close | renamed `TestFiveMCloseDeathNeedsConfirmFloor`: one close does NOT fire; two consecutive do (FLIP_CONFIRM_CLOSES=2 floor) |
| `store/bar_history_test.go` `TestBarInsertAndDedup` | re-insert dedupes via OR IGNORE (first write kept) | revision UPDATE wins on the natural key; non-1m rows not stored; `BarsIntegrity()` asserts dups=0 tfs=[1m] |

No golden changes; flip-eval is not exercised by `golden_selfcheck.go`.

## 3. New tests

`kernel/plan_hysteresis_test.go` (7): wick-through no-fire · two-closes-beyond fires · buffer blocks near-close · long/short symmetry · confirm floor · rearm close-back (both sides). `trader/plan_lifecycle_wave_test.go` (5): reasoning wire defaults/off/fallback · dormant↔rearm round-trip (same version) · flip death → DORMANT, no new version, no terminal no_trade · dormant blocks entries (gate verdict, management passes) · reasoning routing per call type.

## 4. Cutover (flat-gated)

- Flat check: 0 OPEN positions (00:03 CT).
- Marker: `Deploy marker: 99b96b15493e` · boot 00:04:59 CT PID 2701087:
  `🔐 BOOT INTEGRITY OK — rev 99b96b15493e +dirty · built 2026-08-27T05:03:20Z · expected 99b96b15493e · goldens PASS`
  `🧬 plan lifecycle: hysteresis=buffer0.5×ATR14 confirm=2close(s) · flip/death→dormant+auto-rearm (version unchanged, budget untouched) · exec_reasoning=fast→low plan_reasoning=max`
- Bars migration ran at boot: `✅ bars integrity OK: dups=0 tfs=1m total=6574` (pre-fix: 17,695 dup revisions + mixed TFs; backup table `bars_pre_dedupe_2026-08-27` created).

## 5. E-proofs

1. **deaths ≤2 next chop** — PENDING the next chop session (journal watch; expect ≤2 vs yesterday's 10).
2. **zero terminal `replans_exhausted` markers** — structural: the flip-death branch cannot reach `writeNoTradePlan` (test-pinned: `TestFlipDeathMarksDormantAndSkipsBudget` asserts no `no_trade` row).
3. **zero parse-error refusal loops** — structural: `GateRefusalError` returns before the retry branch; live watch of `⚠️ AI response parse failed` lines pending.
4. **bars dup count 0** — CAPTURED: boot integrity line above.
5. **dormant→rearm quoted pair** — PENDING first live flip (log lines `😴 … DORMANT` → `⚡ … REARMED`).
6. **dormant-plan touch firing a planner wake** — PENDING first live event (log `🗓️ level wake … waking the planner` on a dormant/no_trade row).
7. **attribution** — CAPTURED: #562 → `2026-08-26:LONDON v1` (matches entry record), #563 → `2026-08-26:LONDON v9` (was wrong-day / UNRESOLVABLE). Backfill run: resolved=39, unresolvable=4 (2 no-version + 2 off-plan — honest).

## 6. Deferred

None — 8/8 shipped.
