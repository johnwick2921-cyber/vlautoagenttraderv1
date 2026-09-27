# CHECKLIST RUN 1: 39 ✅ · 17 owner-to-verify · 25 pending · 13 NOT DONE (of which 0 blocking)

Run against `docs/VL-VERIFICATION-CHECKLIST.md` v1 (committed `02ebb9e3` **before** this run, so the standard could not be edited to fit the result). READ-ONLY: nothing was fixed, nothing restarted, `data/data.db` opened `mode=ro` only. Prior evidence cited rather than re-derived where it already proves a row.

**STEP 0** — HEAD `02ebb9e3` (≥ 909d3a48 ✓) · live bot PID 322463 on `909d3a48` · tree clean-ish · no other session writing the repo.

> ## ⛔ CORRECTION (2026-08-17, P0 investigation)
>
> **Rows B5, B6 and the Section E R:R / confidence / size-cap entries below are WRONG — my audit query read `risk_control` at the top level, but it is nested under `ai_config`.** The live values are `min_risk_reward_ratio: 3`, `min_confidence: 65`, `max_contracts_per_order: 2`, `hold_discipline: true` on both running strategies, and the effective gate values after `ClampLimits()` are 3.00 / 65 / 2 / true.
>
> Corrected states: **B5 ✅** (cap 2, and master-independent — the agent I "corrected" was right) · **B6 ✅** (hold-lock ON) · Section E **R:R ✅ MATCH · confidence ✅ MATCH · size cap ✅ MATCH**. B11's finding ③ (39 non-re-derivable rows) and the W12 inverted-note finding ① stand. Guardrails master remains OFF by the owner's decision.
>
> Revised header count: **43 ✅ · 17 owner-to-verify · 25 pending · 9 NOT DONE (0 blocking)**.
>
> Pinned by `kernel/risk_config_truth_test.go`.


---

## SECTION A — THE MACHINE IS WHAT YOU THINK IT IS

| # | State | Receipt |
|---|---|---|
| A1 | ✅ | `🔐 BOOT INTEGRITY OK — rev 909d3a48f288 +dirty · built 2026-08-16T16:48:48Z · expected 909d3a48f288 · goldens PASS` (13:04:21) |
| A2 | ✅ *(with a caveat)* | `deploy/RELEASE` = `909d3a48…` = the running binary's `vcs.revision`. **Caveat:** HEAD is now 1 commit ahead (`02ebb9e3`, this run's checklist commit). Rebuilding without re-arming would refuse trading. |
| A3 | ✅ | `pgrep -fa nofx-bin` → exactly one: `322463 /home/hoang/nofx/nofx-bin` (the sandbox runs as `nofx-sandbox`, a separate binary) |
| A4 | 🔵 | Bundle IS current: `web/dist/assets/index-DLSi5dVY.js` built 13:04, and both W16 strings are in it (`Refused this session`, `kept the plan as it was`). The **hard-reload in your browser** is OWNER-TO-VERIFY. |
| A5 | ✅ | `tcp_server: hello handshake OK protocol_version=3 source=vltrader-addon` (13:07:15) |
| A6 | ✅ | `System clock synchronized: yes` · `NTP service: active` · `Time zone: America/Chicago (CDT, -0500)` |
| A7 | ✅ | `/dev/sdf 1007G 90G 866G 10% /` · journals 1.9G (≤ 2G cap) |
| A8 | ✅ | `nofx-2026-08-16_050006.db.gz` (today, 37M) + `deploy/RESTORE.md` present |
| A9 | OWNER-TO-VERIFY | `RESTORE.md:57-89` documents the binary rollback **with the re-arm as step 3 "← never skip"**; the DB half is marked `TESTED read-back — 2026-08-13`. The binary half has never been rehearsed — the row asks you to read it once. |

---

## SECTION B — THE SAFETY SPINE

| # | State | Receipt |
|---|---|---|
| B1 | ✅ | CTO §5.2. Double guard: Go `tcp_trader.go:234-241` (unbound trader refused, no fallback; `isAccountTradeable` = found ∧ IsSim ∧ allow-list) + C# `VLTraderTCPClient.cs:772` `IsSimAccount` as the last line before submit. `AllowedNTAccounts` can only NARROW; PlanDoc has no account field; no force-live flag exists. |
| B2 | ✅ | Fixture run in a /tmp rsync copy (repo untouched). The REAL order is **two-phase**, not the single chain the checklist lists: **Phase 0** kernel pre-prompt (CME-closed, expiry, concurrent cap, daily guardrails) → **Phase 1** `validateDecision` (armor, R:R, confidence) → **Phase 2** the nine executor gates (feed-down :137 · dead-man :151 · freeze :167 · boot-integrity :182 · consecutive-loss :199 · last-entry :216 · session :230 · plan-mode :244 · approval :258). ⚠️ **The checklist's list needs correcting**: there is no killzone gate, and 'size cap' is a post-gate clamp, not a gate. |
| B3 | ✅ | `TestB3PlanCannotBypassAnyGate` — 7 subtests, ALL PASS. A maximally demanding PlanDoc (long/high-conviction, matching A+ scenario, empty no_trade) installed via `ActivePlanProvider`, each gate armed in turn: feed_down · dead_man · frozen · boot_integrity · last_entry · session_gate · approval all still REFUSE. Plan-mode sits 8th of 9 and can only ADD a restriction. |
| B4 | ✅ | `TestB4PlanDocHasNoRiskFields` + 10 subtests PASS. `PlanDoc` has exactly 7 top-level fields (reasoning, bias, levels, scenarios, no_trade, death_condition, day_type); no nested type carries a threshold. An owner overlay is an RFC-6902 patch against that schema, so it *cannot* express a risk value. |
| B5 | ⬜ **NOT DONE** | Two halves, one passes. **Master-independence ✅**: `resolveMaxContracts` (`auto_trader_orders.go:47-53`) is documented *"Hardening D3: ALWAYS ON — the guardrails master switch no longer disables it"* and contains no master check. **The VALUE fails ❌**: `max_contracts_per_order` is **ABSENT** on both running strategies, so `ResolveMaxContracts(0, maxFuturesContracts)` returns the venue default **10** — not the researched **2**. *(An agent reported "cap=2"; I re-checked and that is wrong.)* **Sev MEDIUM · Size S (one config field).** |
| B6 | ⬜ **NOT DONE** | The gate is correct and tested (CTO §5.3, `TestHoldLock` 4/4 PASS) but **OFF in the live config**: `hold_discipline_enabled` is ABSENT on both running strategies and `auto_trader_orders.go:79` defaults it false. So today the AI *can* close by opinion. Breakeven is ABSENT too. **Sev MEDIUM · Size S (config, not code).** |
| B7 | ✅ *(with 3 robustness findings)* | Proven three ways: `enforceEODFlat` (`auto_trader_clock.go:217-249`) calls `at.trader.CloseLong/CloseShort` directly — its body contains **no** call to `executeDecisionWithRecord` or `holdLockSuppressesClose`. Findings, all MEDIUM: **(a)** it flattens only STORE-known positions, so an NT8 **orphan rides overnight** (size S); **(b)** it sits below **three** early returns in `runCycle` (:59 stopped, :77 CME gate, :92 account gate) so a stopped trader never flattens (size M); **(c)** NEW — `tickOnce` bar-close cadence gates it too (`clock.go:354-364`), and cadence is active exactly when EOD-flat is armed (size S). |
| B8 | ⬜ **NOT DONE** | CTO §5.7 enumerated multiple FAIL-OPENs that remain: T1 blackout returns nil on a missing/failed calendar slice (`auto_trader_calendar.go:104-123`, 4 silent paths); R:R gate skipped with no entry reference (`engine_position.go:148`); `placeEntry` never checks the TCP link; a DB read error silently zeroes the daily-loss and trade-count caps; **zero of the 8 dependency failure modes emits an alert**. **Sev MEDIUM-HIGH · Size M.** |
| B9 | ✅ | W16/R7. `applyStaleDataBlock` runs post-fetch; 3 tests pinned — `TestW16StaleEntriesAreStillBlockedAfterT22Removal` (stale opens → `wait`, exits untouched), `TestW16FreshFeedIsNotBlocked`, `TestW16NoIntradayDataFailsOpen`. |
| B10 | ✅ | W16/R5 `4381c801`. P0 `untracked-position` alert + `discipline.FreezeTrader`; `TestW16UntrackedPositionFreezesEntriesOnly` + `TestW16FreezeBlocksOpensNotExits` PASS. |
| B11 | ⬜ **NOT DONE** *(the math is right; the row's bar and the audit report are not)* | **What IS verified ✅:** single source of truth `market/futures_symbol.go:88` `MNQ: 2.0` and `:114` `0.25` → $0.50/tick, and every production site that multiplies points by a point value agrees. One REAL trade to the cent — id=**515** (`source='system'`, not the demo row): matches `realized_pnl` exactly. **Three findings:** ① the W12 report's own note at `2026-08-16-math-audit.md:37-39` says the R:R floor is **3.0 (stricter)** — it is actually **1.0 (looser)**; the note is inverted **in the unsafe direction** (**Sev HIGH · Size S — correct the report**). ② `w12_money_math_test.go` uses a LOCAL `pnlRef` reimplementation and never executes the production P&L code in `close_sync.go:139-146` (**MED · S**). ③ **39 REAL closed rows cannot be re-derived** from entry/exit/quantity (multi-fill aggregation) and 60 more are booked at $0 — so any stat that recomputes P&L from those columns is wrong; `realized_pnl` is the only trustworthy field (**MED · M**). Also: the checklist's "3 oracles each" phrasing appears nowhere in the W12 report. |
| B12 | ✅ | `127.0.0.1:8080` (nofx-bin) · `127.0.0.1:3000` (vite). Both loopback — the 0.0.0.0 dev-server bind was closed in `a84d6ae2`. |
| B13 | ✅ | Live, this run: `POST /api/reset-account → 401` · `/api/reset-password → 410` · `/api/crypto/decrypt → 401`. Matches `2026-08-16-security-p0-fix.md`. |
| B14 | ✅ | No `sk-` under `data/`; none in journald over 24h. The one key-shaped string in `web/dist` is `cm_568c67eae410d912c54c` — the **known-dead public NofxOS default** (`provider/nofxos/client.go:19`, returns HTTP 402), not your secret. |
| B15 | **OWNER-ONLY** | DeepSeek console — old keys revoked, new key in Studio. Cannot be verified from here. |

---

## SECTION C — EVERY PIPELINE CONNECTS

*(re-run at HEAD; the CTO traces were taken 10 commits ago and W16 moved 7 things)*

| # | State | Receipt |
|---|---|---|
| C1 | ⬜ **NOT PROVEN** *(not broken)* | Code+golden half PASSES: `TestFuturesKeyLevelsInjection`, `TestFuturesKeyLevelsGolden`, `TestBuildKeyLevelsBlock`, `TestRenderKeyLevelsBlock`, `TestW11bScoreLevelsSurfacesPersistedState` all PASS. **Live half is EMPTY**: 28,914 `decision_records` contain ZERO `KEY LEVELS (map` — fully explained, `day_plan.plan_enabled` was written 2026-08-15 04:06 UTC, **after** the last decision (08-14 21:00 UTC), and CME has been shut since. One weak hop has no test (`engine_analysis.go:314-341`; the rehearsal test re-implements it rather than calling it). **Ticks at Monday's open.** |
| C2 | ✅ COMPLETE | `TestFuturesPlanGolden` PASS — the plan-active futures prompt is byte-identical to the golden. Confirmed the new `kernel/scenario_state.go` (R1) does **not** touch prompt bytes. |
| C3 | ✅ COMPLETE | `TestC3FeedToStoreToGateRefusal` PASS — fixture feed → `maybeFetchCalendar` → stored slice → T1 windows → the executor **refuses** an entry inside the blackout. |
| C4 | ✅ COMPLETE *(2 conditional breaks off-path)* | Overlay → plan_final → executor proven at HEAD. **NEW FINDING, verified by me:** `ListOverlays(planID, planVersion)` (`store/plan.go:337-340`) filters on **both** keys, so a re-plan to v2 **silently orphans every v1 overlay** — the owner's edits vanish from plan_final. Reachable in normal operation (replan_cap=2). **Sev HIGH · Size M.** |
| C5 | ⬜ **BREAK** at `auto_trader_session.go:108` | The **codec half is fully PROVEN**: 6/6 tests, all 23 day_plan fields survive both hand-rolled halves and every one has a production reader — the footgun this row hunts does **not** exist. The break is the same session-enable root cause as C7. |
| C6 | ⬜ **BREAK** at `kernel/digest.go:12` | `TestC6ProbeLearningSignalDiesAtDigest` (probe, /tmp) confirms it at HEAD: `FormatSessionDigest` takes only (entries, realizedPnL) — no parameter for adherence grade or MAE/MFE, so the learning signal dies before the digest that seeds the next read. R4's trader-scope fix landed correctly but did not change this. |
| C7 | ⬜ **BREAK** at `auto_trader_session.go:108` | `TestC7HopByHop` PASS. Still **9 sites**, still only 2 converted — unchanged since the CTO trace. `sessionGateDecision` runs *after* the converted check at :38 and overrides it. |
| C8 | ✅ COMPLETE | Agent ran 4 test sets, all PASS, incl. `TestAlertDedupeByEventID`. Producer count grew 9 → 11 `emitAlert` sites since the CTO trace (R5 added `untracked-position`). Live DB: 5 alerts, 0 unacked, **3 of them demo rows**. |

---

## SECTION E — SETTINGS AT RESEARCHED VALUES

Live query of both running strategies (`a5b7662e` → trader *hoang*/Sim101 · `70695b25` → trader *15m*/SimAccount1). **Nothing changed since the CTO run.**

| Setting | Should be | LIVE VALUE | Verdict |
|---|---|---|---|
| Proximity | 1.5×dATR | `1.5` | ✅ MATCH *(but shadowed at 3 kernel sites — CTO H1/H2/H3)* |
| Max levels | 8 | `8` | ✅ MATCH *(9–12 unreachable — H4)* |
| Max scenarios | 3 | `3` | ✅ MATCH *(4–5 unreachable — H5)* |
| Max re-plans | 2/session | `2` | ✅ MATCH *(prompt value hardcoded — H6)* |
| Acceptance | 2×5m | `2x5m` | ✅ MATCH |
| Approval required | OFF | `false` | ✅ MATCH |
| Evening digest | ON | `true` | ✅ MATCH |
| Plan mode | advisory | `advisory` | ✅ MATCH |
| NY session | ON | `sessions_enabled=['NY']` + NY override | ✅ MATCH |
| ASIA / LONDON | your decision · min_grade **A** · Asia max_trades **1** | **ASIA+LONDON `enable:true`, min_grade `B`, max_trades `3`** | 🔴 **DRIFTED** (3 ways, all more permissive) |
| Last entry | 13:00 CT | `13:00` | ✅ MATCH |
| EOD flat | 14:45 CT | `14:45` | ✅ MATCH |
| Activation window | 1.5×ATR | `ActivationWindowK=1.5` | ✅ MATCH |
| Re-arm cooldown | 20 min | `ReArmCooldownMin=20` | ✅ MATCH |
| Freshness floor | C | `FreshnessC`/`FreshnessDone` | ✅ MATCH |
| **R:R gate** | **≥3.0** | **1.0** (`risk_control={}` → `ClampLimits` floors to 1.0) | 🔴 **NEVER-APPLIED** |
| **Confidence gate** | **≥65** | **50** (same cause) | 🔴 **NEVER-APPLIED** |
| **Size cap** | **2 contracts** | `max_contracts`/`max_positions`/`max_position_size` all **NULL** | 🔴 **NEVER-APPLIED** |
| Re-entry cooldown | 20 min | guardrail — master OFF | *deliberate* |
| Stats gate | n≈1565, α≈0.006 | `PreRegisteredN=1565`, `0.05/8` | ✅ MATCH |
| Digest chain | sessions + 3 dailies + 4–7 | `BuildDigestChain` | ✅ MATCH *(content thin — C6)* |
| Planner model | exact pinned id, never an alias | `dayplan_pinned_model=deepseek-v4-pro`; `IsProviderAlias("deepseek-v4-pro")==false` (asserted by `mcp/model_alias_test.go:26`); real plan rows carry exactly that | ✅ MATCH |
| Guardrails master | OFF (your dated decision) | OFF — `risk_control={}` on both | ✅ **CONFIRMED, NOT TOUCHED** |

---

## SECTION D — EVERY CONTROL WORKS

The sandbox is unauthenticated and `httpClient` redirects to `/login` on 401, so **no row here can be proven in a real logged-in browser by me**. Component-level renders are labelled as such.

| # | State | Note |
|---|---|---|
| D1 | OWNER-TO-VERIFY | Toggle + persist proven at component level (W15 Playwright, `shots/2026-08-17-w15-controls-after-reload.png`). ⚠️ The row's "done" says chips show **A🔸 and max_trades 1🔸** — the live config has **B and 3**, so as written this row FAILS on today's settings (see Section E drift). |
| D2 | OWNER-TO-VERIFY | Config-truth round-trip proven in W15; the browser save/reload is yours. |
| D3–D8, D12, D13, D16–D18 | OWNER-TO-VERIFY | Need a logged-in browser and, for D13/D16, a live plan. |
| D9, D10 | OWNER-TO-VERIFY | Ask-Planner verdicts need a real planner call (costs an LLM round trip). |
| D11 | 🔵 | W16/R2 — declined copy renders and does **not** contain "applied": `shots/2026-08-17-w16-decline-copy.png` + 3 vitest. Browser confirmation is yours. |
| D14 | ⏳ PENDING | R1 shipped the writer, but there is **no active real plan** and **no `scenario_status:*` key** in system_config yet — it has not run against a live plan. Ticks at the next session read. |
| D15 | ⏳ PENDING | Panel shipped (5 vitest); needs a refusal to have occurred this session. |
| D19 | ⬜ **NOT DONE** | **Regression confirmed at HEAD**: `kernel/engine_prompt.go:710-719` writes the OHLCV header and volume column unconditionally when `len(data.Klines)>0` (always true on the NT8 path); `EnableVolume` only gates the mid-prices fallback at `:724`. Turning Volume off does not remove volume from the prompt. **Sev LOW-MED · Size S.** |
| D20 | 🔵 | `DayPlanEditor.tsx:400-402` renders AUTO as a label + tooltip (`autoTooltip`), non-interactive. |

---

## SECTION F — LIVE PROOF

| # | State | What ticks it |
|---|---|---|
| F1 | ⏳ | A plan row with Monday's date at the 08:25 CT read. Last real rows: `2026-08-15 NY v1/v2`, both expired. |
| F2 | ✅ | **Already proven live**: `plans` holds `2026-08-15 NY v1 planner_fail_closed` and `day_plan_alerts` holds `failclosed:2026-08-15:NY` (P0). The fail-closed path fired for real. |
| F3 | ⏳ | Query a decision's prompt after the open. |
| F4 | ⏳ | The only `cited_scenario_id` row is **id=520, `source='demo_seed'`** — fabricated. No real citation exists yet. |
| F5, F6, F7, F8, F9 | ⏳ | Need a live entry / winner / 13:00 / 14:45 / a session boundary. |
| F10 | ⏳ | `day_plan_digests` has **0 rows** — no digest has ever been written. |
| F11 | ⏳ | The only graded row is **id=520 (demo)**. No real MAE/MFE + grade yet. |
| F12–F15 | ⏳ | Daily reconciliation check / a refusal / a re-armed guardrail / a mid-session restart. |

---

## SECTION G — TRUST EARNED

All ⬜ NOT STARTED — these are weeks of live data, not tasks. G1 blind-mark calibration · G2 naked-POC 10/10 warm-up · G3 ≥100 cited decisions · G4 n→1565/type · G5 regime coverage · G6 100+ closed trades · G7–G9 mode promotions and real capital. **No green verdict on G4 before n is reached** (the honesty gate is code-enforced: `PreRegisteredN=1565`, `BonferroniAlpha=0.05/8`).

---

## SECTION H — KNOWN OPEN ITEMS (re-verified at HEAD)

| Item | Size | Still open? |
|---|---|---|
| AddOn watchdog livelock (4/7 regime fields dark) | M | ⬜ **OPEN** — needs C# + F5 + NT8 restart; not verifiable headlessly (`ninjascript/VLBarsSubscriptionManager.cs` present, unchanged) |
| Scenario anchor missing from schema | S | ⬜ **OPEN** — `PlanScenario` still has no anchor/price field. R1 handles it by emitting nothing, so dots are honest-but-absent. |
| Stale-data drift half needs TF-aware thresholds | M | ⬜ **OPEN** — `IsDriftSuspicious`/`HealthSuspiciousDrift` have **zero live callers** |
| Force re-read button | M | ⬜ **OPEN** — no `plan/reread` endpoint exists |
| Plan history browser | S–M | ⬜ **OPEN** — endpoint + api client exist; **no component consumes `getPlanHistory`** |
| Killzone gate (grading-only) | S–M | ⬜ **OPEN** — `InKillzone` used only by `kernel/adherence.go` |
| R3 panel unverified in a real browser | — | ⬜ **OPEN** — 5 component tests only; auth still blocks Playwright |
| `approval_required` has no grant UI | M | ⬜ **OPEN (deferred)** — `POST /plan/approve` exists, no FE consumer |
| 7 low-severity security items | S | ⬜ **OPEN** — behind loopback (B12 ✅) |
| MAE/MFE visualization, per-scenario expectancy | M | ⬜ v1.1 shelf |
| Telegram/external notifications | M | ⬜ v1.1 shelf (your decision: in-app only) |
| **NEW — overlays orphaned by a re-plan** (`store/plan.go:337-340`) | M | ⬜ **OPEN** — `ListOverlays` filters on plan_id **and** plan_version, so a v2 re-plan silently drops every owner edit made against v1. **Sev HIGH** |
| **NEW — 39 real closed rows not re-derivable** (multi-fill) | M | ⬜ **OPEN** — never publish a stat recomputed from entry/exit/quantity |
| **NEW — W12 math-audit R:R note is inverted** | S | ⬜ **OPEN** — says floor 3.0, real floor is 1.0 |
| **NEW — EOD-flat robustness** (orphans, 3 early returns, bar-close cadence) | M | ⬜ **OPEN** — B7 (a)(b)(c) |
| **NEW — effective per-order cap is 10, not 2** | S | ⬜ **OPEN** — `max_contracts_per_order` unset → venue default |
| **NEW — demo rows still in the live DB** | S | ⬜ **OPEN** — 2 demo plans, 3 demo alerts, 1 fabricated trade. They are now **masking F4 and F11**: the only scenario-citing row and the only graded row are both the demo trade. The corrected cleanup command is in the CTO report. |

---

## THE OWNER'S TO-DO LIST (rows only you can tick)

1. **B15** — rotate the DeepSeek key; confirm the old one is revoked.
2. **A9** — read `deploy/RESTORE.md:57-89` once (the binary rollback + re-arm).
3. **A4** — hard-reload the UI and confirm the "Refused this session" panel exists.
4. **Section D** (18 rows) — the 10-minute browser pass.
5. **Two config decisions** (Section E): whether ASIA/LONDON should be on at `min_grade B / max_trades 3` against the researched `A / 1`; and whether R:R 1.0 + confidence 50 + no size cap is the bar you want (all three are `risk_control={}`, not the guardrails master).
6. **Run the corrected demo-row cleanup** — it is currently faking your only F4/F11 evidence.
