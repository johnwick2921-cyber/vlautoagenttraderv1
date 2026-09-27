# SYSTEM VERIFIED — 41 remaining, 0 blocking

Read-only audit of `/home/hoang/nofx` @ `5472f316`, Sunday 2026-08-16 (market closed). 22 agents (13 hardcode/pipeline + 9 matrix/checklist) plus my own verification of every claim I report. **Seven fixes shipped** — six trivial+safe, one a regression this audit found in my own W15.C change. Everything else is listed with a size. **Nothing found is blocking for Monday's SIM open**, but items 1–4 and 9 change what the bot actually does and should be decided before you enable more sessions.

Evidence tiers: **[A]** = I read the exact line or ran it and saw output · **[B]** = strong inference · **[C]** = speculation. Every finding below is [A] unless marked.

> ## ⛔ CORRECTION (2026-08-17, P0 investigation)
>
> **The "empty risk_control" headline below is WRONG, and the error was mine.** `risk_control` is nested under `ai_config` — exactly where the hand-rolled codec writes it (`store/strategy.go:763-769` / `:806-811`). My audit query read the **top level**, found nothing, and defaulted the miss to `{}`.
>
> The live values were correct all along: **`min_risk_reward_ratio: 3` · `min_confidence: 65` · `max_contracts_per_order: 2` · `hold_discipline: true` · `breakeven_enabled: true`**, on BOTH running strategies. Proven by loading the real stored JSON through the production codec: after `ClampLimits()` the effective gate values are **minRR=3.00 · minConf=65 · maxContracts=2 · holdLock=true**.
>
> Therefore, in the table and prose below: **R:R gate = ✅ MATCH (3.0)**, **confidence gate = ✅ MATCH (65)**, **size cap = ✅ MATCH (2)**, and the §5.3 finding "hold-lock is OFF on both live traders" is **withdrawn**. `guardrails_enabled: false` is unchanged and remains the owner's deliberate decision.
>
> Pinned against recurrence by `kernel/risk_config_truth_test.go` (`TestRiskConfigLivesUnderAIConfig`).


---

## STEP 0

| Check | Result |
|---|---|
| HEAD ≥ 5472f316 | ✅ `5472f316` |
| Tree clean-ish | ✅ `M deploy/RELEASE` (your re-arm) + the known `??` files |
| Live bot on 5472f316 | ✅ PID 175755, `vcs.revision=5472f316…`, restarted 09:39:22 |
| Boot integrity armed, expected==actual | ✅ `BOOT INTEGRITY OK — rev 5472f31608af · expected 5472f31608af · goldens PASS` |
| Sandbox :3001 untouched | ✅ own PID/DB/port (8081 / 36985 / sandbox.db) |
| One session | ⚠️ PID 100656 (session `554049f5`) still resident but **idle — zero repo writes in 30 min**; proceeded as before |

---

## PART 1 · THE HARDCODE HUNT

The root disease, confirmed: a value compiled in where config should rule. **9 SHADOWS-CONFIG classes where the literal WINS** (deduped from 42 raw agent claims; each re-verified by me).

| # | Literal | file:line | Shadows | Literal wins? | Verdict |
|---|---|---|---|---|---|
| H1 | `band := 1.5 * dATR` | `kernel/levels_score.go:125` | `proximity_filter_atr` | **YES** | SHADOWS-CONFIG |
| H2 | `band := 1.5 * dATR` | `kernel/levels_intraday.go:24` | `proximity_filter_atr` | **YES** | SHADOWS-CONFIG |
| H3 | `ActivationWindowK` passed as `k` | `kernel/plan_render.go:65,67` | `proximity_filter_atr` | **YES** | SHADOWS-CONFIG |
| H4 | `planMaxLevels = 8` | `kernel/plan_doc.go:60` (enforced :101) | `max_levels` (documented 3–12) | **YES** | SHADOWS-CONFIG |
| H5 | `planMaxScenarios = 3` | `kernel/plan_doc.go:61` (enforced :115) | `scenario_cap` (documented 1–5) | **YES** | SHADOWS-CONFIG |
| H6 | `replansLeft := 2 - (version-1)` | `trader/auto_trader_planner.go:580` | `replan_cap` | **YES** | SHADOWS-CONFIG |
| H7 | `DefaultSessionRegistry()` | `kernel/engine_analysis.go:356,384` | the persisted admin registry | **YES** | SHADOWS-CONFIG |
| H8 | `!sess.Enabled` ×7 sites | `auto_trader_session.go:108`, `planner.go:528,566`, `handler_plan.go:143/475/555/711/1122` | `sessions[].enable` | **YES** | SHADOWS-CONFIG |
| H9 | bars `"1d"/"1h"/"5m"` | `trader/auto_trader_planner.go:421-424` | `planner_timeframes` | **YES** (see below) | SHADOWS-CONFIG |

**Runtime proofs** (an agent built a probe module with `replace nofx =>` and ran the real kernel functions):
- `ProximityFilterATR=3.0`, price 21000, dATR 100 → a PDH at 2.0×dATR was **DROPPED**; only the 0.5× level seated. `ScoreLevels` has no proximity parameter at all, so the configured value is *structurally incapable* of reaching it. H2 is the upstream twin: round-number levels beyond 1.5× are never even generated, so fixing H1 alone only half-honours the setting.
- `MaxLevels=12` + a 9-level doc → `too many levels: 9 (max 8)`. `ScenarioCap=5` + 4 scenarios → `scenarios count 4 invalid (1..3)`. Both **reject the whole plan** → fail-closed NO-TRADE. The upper half of both documented ranges is unreachable, and choosing it degrades to no plan. (`auto_trader_planner.go:315` does read `scenarioCap()`, but post-parse — it can only truncate below 3, never raise above.)

**H9 nuance (mine):** `planner_timeframes` *is* read (`planner.go:362`) — but only to emit placeholder lines `"<tf>: structure read"` into the prompt, while the bars actually fetched are hardcoded 1d/1h/5m. So the prompt **asserts a read-set the planner never received**. That is worse than dead: select "4h" and the planner is told it read 4h structure it never saw.

**1b · Two rulebooks (same default applied in >1 file).** Single source of truth that should exist → `store.DefaultDayPlanConfig` for values, `sessionRunnable()` for session enablement, the persisted registry for clock times.
- Session enablement: 9 deciding sites, 2 converted (see §PART 2 P-E/P-G).
- Re-plan budget: `store/strategy.go:933` `ReplanCapFor` **and** `planner.go:580` literal.
- Re-align cap: `store/strategy.go:870` tag default **and** `auto_trader_planconfig.go:218` `DefaultRealignCap`.
- Clock times: `kernel/session_registry.go` **and** `web/src/components/plan/sessionConfig.ts:41-72` (FE copy of every window/read/flat/killzone).
- FE `DEFAULT_DAY_PLAN` vs Go `DefaultDayPlanConfig`: agree today; nothing enforces that they keep agreeing.

**1c · Frontend constants gating backend-decided behavior.** `SESSION_BANDS[].enabled` still drives `SessionTimelineStrip.tsx:76` shading and `DayPlanEditor.tsx:291-297` `sessionRunnableUI` (the plan card's tabs were fixed in `e943f9c3`, these two were not). Also FE-only rules with no backend counterpart: `levelState.ts:44` `bandPts=3` conflict band, `levelState.ts:11` `NEAR_THRESHOLD_PTS=12`, `vocab.ts:21-29` instruction verbs presented as "canonical tokens the executor understands" against a free-text Go field.

**Fixed this run:** none of the SHADOWS-CONFIG class — every one changes trading behavior and none is trivial+safe on a Sunday before an open. All are in the REMAINING list with sizes.

---

## PART 2 · PIPELINE TRACES — 5 COMPLETE, 3 BREAK

| Pipeline | Verdict | Proof / break |
|---|---|---|
| **P-A** bars → detectors → scorer → KEY LEVELS → prompt → decision | ✅ COMPLETE | golden tests run |
| **P-B** bars → regime → planner → plan JSON → PLAN BLOCK → prompt | ✅ COMPLETE | `TestFuturesPlanReorder` + `TestFuturesPlanGolden` run, PASS |
| **P-C** calendar → slice → planner → T1 blackout → entry refused | ✅ COMPLETE | 6 tests written+run in an isolated copy (repo untouched); the executor **does** refuse inside a T1 window |
| **P-D** owner edit → overlay → plan_final → prompt → realign → Apply | ✅ COMPLETE | `TestW4OverlayReachesExecutor` PASS + live overlay row. The hop I suspected (base doc vs plan_final) is **not** broken — the executor gets plan_final |
| **P-E** config → dual codec → row → hot-read → consumer | ❌ **BREAK** | at the final consumers of `sessions[].enable` / `sessions_enabled` — 7 sites read the registry flag instead |
| **P-F** fill → exit → MAE/MFE + adherence → digest → next read | ❌ **BREAK** | `kernel/digest.go:12,18` — the digest formatter has **no parameter** for grade/MAE/MFE. The learning loop is open |
| **P-G** registry → scheduler → entry gate → flat → night | ❌ **BREAK** | `auto_trader_session.go:108` — a live-reachable bypass that vetoes the owner's explicit `enable=true` |
| **P-H** alert emit → feed → P0 banner → ack | ✅ COMPLETE | `TestAlertDedupeByEventID` run PASS + live DB receipt; the banner cannot be dismissed without acking |

**The break, stated once:** `sessionRunnable()` (W15.A) is the resolver for "does session X run", and **only 2 of 9 sites use it**. With ASIA/LONDON switched on — which your live config does — the read scheduler fires (LLM spend), a plan row is written, and then the executor gets `nil`, entries are blocked, no digest is written and the card stays empty. A toggle that costs money and changes nothing. Pinned by `TestW16SessionEnableIsHonoredByOnlySomeGates`.

**Worse, and separate — ASIA's read can never fire at all.** `IsCMEOpen` returns false for the whole 16:00–17:00 CT maintenance break, and ASIA's `ReadCT` is **16:55**. Probe output:

```
ASIA designed read (Mon) 16:55  IsCMEOpen=false  => read BLOCKED
ASIA designed read (Tue) 16:55  IsCMEOpen=false  => read BLOCKED
ASIA designed read (Sun) 16:55  IsCMEOpen=false  => read BLOCKED
ASIA after midnight (Tue) 00:30 IsCMEOpen=true   => read FIRES   ← the only one that can
LONDON designed read     01:55  IsCMEOpen=true   => read FIRES
NY designed read         08:25  IsCMEOpen=true   => read FIRES
```

The spec explicitly requires this path — *"the 16:55 closed-market read is a first-class tested path — planner builds from stored data"* — and the CME-open gate blocks it. The only ASIA read that can fire is the accidental post-midnight one, under the **wrong trade date** (plan identity uses `plannerTradeDateCT`, which rolls at midnight, while `CMESessionDayKey` rolls at 17:00). Pinned by `TestW16TradeDateNotionsDisagree`. *(I first reported this as a "duplicate read"; the adversarial review overturned that — the designed read never fires.)*

---

## PART 3 · CONTROL + STATE MATRIX

The 11 controls fixed in `bc360a38`/`d396201e`/`e943f9c3` were re-verified end-to-end and **all stayed fixed**. Playwright, own profile, headless, after a **hard reload** (`shots/2026-08-17-cto-controls-verified.png`):

```
NY=true  ASIA=true (🔸 override chip)  LONDON=false
last_entry_ct=12:45   eod_flat_ct=14:45   eod warning correctly absent at the window end
```

Caveat on the toggles: they are live in the *config* sense (written, persisted, read by the scheduler and the outer entry gate) but land in the 7-site gap above — so "LIVE" here means the control works, not that enabling ASIA yields a working session.

Three agents verdicted the rest (day-plan block: 19 LIVE / 2 BROKEN / 5 DISPLAY-ONLY / 1 READ-ONLY / 1 MISSING · plan card + alerts: 13 LIVE / 5 BROKEN / 6 DISPLAY-ONLY / 2 MISSING / 1 READ-ONLY). I re-verified every BROKEN claim; all held. **Three were trivial+safe and are fixed** (`e33532e2`, `2427c850`, `637b137a`); the rest are in REMAINING.

**M1 · The owner door was mutating the wrong session — my regression, now fixed.** Making the tabs real (`e943f9c3`) let the card show a *sibling* session, but `/plan/overlay`, `/plan/ask` and `/plan/realign` all re-derive the session from `ActiveSession(now)` **server-side** and take no session argument. `doorEnabled` was `!!traderId` and never checked which session was displayed — so editing a level while looking at ASIA would have written the overlay onto NY's plan_final, silently. Fixed by gating the door on `plan.is_active !== false` (the flag the same commit already added), with a visible reason line instead of a mute disabled surface. `e33532e2`.

**M2 · Ask-Planner argued about a document the owner had already edited — fixed.** `handlePlanAsk` unmarshalled `row.Doc` directly while the card folds overlays and re-align uses `resolvePlanFinal`. Owner edits were invisible to the one path built for owner dialogue. `2427c850`.

**M3 · A NO-TRADE plan rendered as a gold "ACTIVE" chip — fixed.** `store/plan.go` is append-only with no lifecycle mutator, so `expired`/`died`/`superseded` are **never written**; `no_trade` **is** (fail-closed / re-plans exhausted). It was the one value missing from `LifecycleChip`'s map, so `?? map.active` painted the fail-closed plan as live. `637b137a`. *(Consequence not fixed: `HandoverBanner` requires the three unreachable values, so it is dead in production.)*

**Card states.** reading / active / night / disabled / fail-closed / no-plan / warming / DEGRADED all render and are reachable. `expired` is **not** — nothing ever writes it. One more honesty gap: `vix_level` is structurally dark forever (no VIX feed; `kernel/regime.go:94` requires `VIX > 0`), so a perfectly healthy read starts at **1/7 dark** and the DEGRADED threshold effectively means "three more must also fail". The live `2026-08-15:NY` v2 doc says "regime/VIX absent" in its own reasoning.

**Executor indicators — one regression.** Timeframes, EMA/MACD/RSI/ATR/BOLL (with configured periods driving the math), SVP and the crypto-only hides are all intact. But **Volume is half-inert on futures**: `formatTimeframeSeriesData` (`kernel/engine_prompt.go:710-719`) writes the OHLCV header and volume column unconditionally whenever `len(data.Klines) > 0` — always true on the NT8 path — and `EnableVolume` only gates the mid-prices fallback at `:724`. Turning Volume off does not remove volume from the prompt.

---

## PART 4 · RESEARCH-DEFAULTS AUDIT

Spec = `docs/VL-DAYPLAN-FULL-SPEC.md` (lines 60-74 field table, line 21 risk defaults). `docs/NOFX-MASTER-TRADING-SPEC.md` **does not exist** in the repo. LIVE = strategy `a5b7662e` (trader `hoang`, Sim101) — the day-plan strategy.

| Setting | Researched | Code default | LIVE VALUE | Verdict |
|---|---|---|---|---|
| proximity_filter_atr | 1.5×dATR | 1.5 | 1.5 | MATCH *(but shadowed — H1/H2/H3)* |
| max_levels | 8 | 8 | 8 | MATCH *(9–12 unreachable — H4)* |
| max_scenarios | 3 | 3 | 3 | MATCH *(4–5 unreachable — H5)* |
| replan_cap | 2 | 2 | 2 | MATCH *(prompt value hardcoded — H6)* |
| acceptance_rule | 2×5m | 2x5m | 2x5m | MATCH |
| approval_required | OFF | false | false | MATCH |
| evening_digest | ON | true | true | MATCH |
| plan_mode | advisory | advisory | advisory | MATCH |
| last_entry_ct | 13:00 CT | 13:00 | 13:00 | MATCH |
| eod_flat_ct | 14:45 CT | 14:45 | 14:45 | MATCH |
| sessions enabled | NY on, ASIA+LONDON off | NY only | **NY + ASIA + LONDON all on** | **DRIFTED** (more permissive) |
| ASIA min_grade | **A** | — | **B** | **DRIFTED** (weaker) |
| LONDON min_grade | **A** | — | **B** | **DRIFTED** (weaker) |
| ASIA max_trades | **1** | — | **3** | **DRIFTED** (3× permissive) |
| activation window k | 1.5 | `ActivationWindowK=1.5` | 1.5 | MATCH |
| re-arm cooldown | 20m | `ReArmCooldownMin=20` | — | MATCH |
| freshness floor | C | `FreshnessC`/`FreshnessDone` | — | MATCH |
| **R:R floor** | **≥3.0** | `ClampLimits` floors to **1.0** | **1.0** | **NEVER-APPLIED** |
| **confidence floor** | **≥65** | `ClampLimits` floors to **50** | **50** | **NEVER-APPLIED** |
| size cap | 2 | — | not set | NEVER-APPLIED |
| consecutive-loss halt | on | guardrail | master OFF | *deliberate* |
| reentry_cooldown | 20m | guardrail | master OFF | *deliberate* |
| stats gate n | ~1565 + Bonferroni α≈0.006 | `PreRegisteredN=1565`, `0.05/8` | — | MATCH |
| digest chain | sessions + 3 dailies + days 4–7 | `BuildDigestChain` | — | MATCH (content is thin — P-F) |
| planner model | exact pinned id | `pinExactModel` dealiases | inherit primary | MATCH |

**The headline.** Both running strategies have an **empty `risk_control` block**, so `ClampLimits()` (called on the decision path at `kernel/engine_analysis.go:247`, mutating the same struct the gates read at :296) floors them to **MinConfidence=50, MinRiskRewardRatio=1.0**. Receipt:

```
EMPTY risk_control after ClampLimits(): MinConfidence=50  MinRiskRewardRatio=1.00
SPEC researched values:                 MinConfidence=65   MinRiskRewardRatio=3.00
```

The spec calls these *"hard gates (R:R≥3, conf≥65, guardrails, armor) always outrank [the plan]"*. They are running at a third of the researched risk-reward floor. This is **not** covered by your deliberate guardrails-master-OFF — these are the base decision validators, not guardrails. **Confirmed and unchanged: `risk_control` is `{}`, guardrails master OFF — your learning mode. I did not touch it.**

---

## PART 5 · THE CTO CHECKLIST

| # | Item | Verdict |
|---|---|---|
| 5.1 | **Safety precedence** | ⚠️ **FINDING** (order verified; 2 side doors) |
| 5.2 | **The SIM lock** | ✅ **VERIFIED** |
| 5.3 | **Hold-lock + exit integrity** | ⚠️ **FINDING** (HIGH) |
| 5.4 | **Money math on the live path** | ✅ **VERIFIED** |
| 5.5 | **Time correctness** | ⚠️ **FINDING** (MEDIUM) |
| 5.6 | **Idempotency + restart** | ⚠️ **FINDING** (HIGH — digest/plan races, memory-only state) |
| 5.7 | **Fail-closed coverage** | ⚠️ **FINDING** (MEDIUM-HIGH) |
| 5.8 | **Data hygiene** | ⚠️ **FINDING** (MEDIUM) |
| 5.9 | **Secret safety** | ⚠️ **FINDING — 1 FIXED** |
| 5.10 | **Observability** | ⚠️ **FINDING** (3 blind spots) |
| 5.11 | **Owner's daily loop** | ⚠️ **FINDING** — 3 of 5 questions unanswerable |
| 5.12 | **Rollback** | ⚠️ **FINDING — FIXED** (`6fe92f16`) |

**5.2 SIM LOCK — VERIFIED.** Entries are **double-guarded**: Go `placeEntry` (`tcp_trader.go:234-241`) refuses an unbound trader outright (no fallback to the shared active account) and refuses `!isAccountTradeable` (must be found **and** `IsSim` **and** on `AllowedNTAccounts` if set); C# `VLTraderTCPClient.cs:772` re-checks `IsSimAccount` as the last line before submit and rejects. Closes are **single-guarded in C#** (`HandleClosePosition:969`) — it refuses and leaves the position OPEN rather than flattening a live account, which is the right call. Enumerated every widening path: `AllowedNTAccounts` can only **narrow**; the PlanDoc schema has **no account field**, so neither planner output nor an owner overlay can influence account selection; no bypass/force-live flag exists anywhere. `/api/debug/nt-test-trade` is JWT-protected + futures-only and bypasses the AI+risk gates **by design**, but still passes the SIM lock and the B3 dedupe/rate breaker.

**5.4 MONEY MATH — VERIFIED.** MNQ $2/pt cross-checked on a real closed trade: `30199.5 → 30179.25 = −20.25 pts × $2 × 1 = −$40.50` ✓ matches `realized_pnl`. `close_sync.go:67` already cross-checks NT8's reported PointValue/TickSize against the local table and warns on mismatch. 456 `sync` + 60 `reconcile_flat` closes all carry correct quantities. *(The one row with `quantity=0` is the seeded demo trade — see 5.8, not a production defect. I initially mis-read it as a TP-path bug and corrected.)*

**5.5 TIME — FINDING.** Two date notions coexist: `plannerTradeDateCT` (midnight roll) for plan identity, calendar slices and the read scheduler; `kernel.CMESessionDayKey` (17:00 roll) for the risk daily reset, guardrail counters, alert dedupe keys, level bucketing and the approval key. Each is internally coherent, and the **digest is correct** (it labels with the calendar date while summing from the 17:00 boundary — the comment shows this was reasoned about). The bite is the one midnight-spanning session (ASIA), above. The 14:45 flat **is** single-sourced post-P3 (`WindowEndCT == FlatCT`, pinned by `kernel/session_end_contract_test.go`, which also parses the TypeScript mirror).

**5.7 FAIL-CLOSED — FINDING.** `currentT1Windows` (`auto_trader_calendar.go:104-123`) returns `nil` — i.e. **no blackout, entries allowed** — on four silent paths: nil store, no active session, **`err != nil || slice == nil`**, and malformed EventsJSON. No alert, no log. So "the calendar feed failed" and "there are no red events today" are indistinguishable and both permit trading. Live state: `calendar_slices` holds 2026-08-18…21 and 08-12…14 but **no row for Monday 2026-08-17**. Second fail-open: `kernel/engine_position.go:148` skips the R:R gate entirely when there is no entry reference (documented, argues other gates cover it).

**5.8 DATA HYGIENE — FINDING.** Demo rows remain in the live DB: **2 plans** (`trigger_reason='demo_seed'`, `model_id='demo-seed (no API call)'`), **3 alerts** (`event_id LIKE 'demo:%'`, including a **P0** "DEMO — planner fail-closed"), **1 overlay** + **1 plan_qa** row on `2026-08-16:NY`, and **1 fabricated closed trade** (`source='demo_seed'`, +$224.50, adherence **A**). It is counted in your stats: **$4,271.25 shown → $4,046.75 real (5.3% of displayed P&L is fabricated)**, plus one fake win and an "A" grade feeding the learning loop. The sandbox seeder is correctly fenced (`cmd/sandbox-seed/main.go:33` refuses any path containing `data.db` or lacking `sandbox`); the contamination came from the guarded `trader/demo_seed_test.go` (untracked, `NOFX_DEMO_SEED=1`). Backups are current and restorable (`~/nofx-backups/auto/daily/`, newest 2026-08-16 05:00; timer armed for 17:30; linger on).

> **The documented cleanup command in `2026-08-16-demo-plan-seed.md` is WRONG — do not run it as written.** It does `DELETE FROM plans WHERE trade_date IN ('2026-08-16','2026-08-15')`, which destroys the **two real 2026-08-15 plans** (`planner_fail_closed` v1 + `NY_scheduled_read` v2, authored by deepseek-v4-pro) — the only genuine plan history you have. It also deletes the two real 08-15 alerts and **misses `plan_qa` entirely**. Corrected command in the handoff below.

**5.9 SECRETS — FINDING, ONE FIXED.** The Go API is loopback-bound, but `web/vite.config.ts` bound the dev UI to `0.0.0.0:3000` **and proxies `/api` to localhost:8080** — re-exposing the whole API to the network and defeating the bind it sits behind. Receipts on the live host:

```
GET  http://<lan-ip>:8080/api/traders        → 000   (refused — bind holds)
GET  http://<lan-ip>:3000/api/traders        → 200   (same API, via the proxy)
POST http://<lan-ip>:3000/api/reset-password → 410   (P0 gating still in force)
```

Most routes 401; `/api/traders` returns `[]` unauthenticated, so nothing is known to have leaked — the defect is that the control was bypassable at all. **Fixed** (`a84d6ae2`) to `127.0.0.1`; safe because WSL2 runs `networkingMode=mirrored` and the sandbox UI has bound loopback throughout. Takes effect next dev-server start. Otherwise: `.env` is untracked; no key-shaped strings in tracked source beyond test fixtures; the one key in `web/dist` is `cm_568c67eae410d912c54c`, the **known-dead public NofxOS default** (`provider/nofxos/client.go:19`, returns HTTP 402), not your secret.

**5.3 EXIT INTEGRITY — FINDING.** The hold-lock gate itself is correct and tested (`go test ./trader/ -run TestHoldLock` → 4/4 PASS, including the suppression log). Three gaps:
- **Hold-lock is OFF on both live traders.** `hold_discipline_enabled` is absent from both strategies' `risk_control` (verified in the live DB), so `holdLockSuppressesClose` returns false immediately — the AI *can* close a protected position by opinion today. This is the same class as your deliberate guardrails-master-OFF: a decision awaiting you, not a code defect. *(An agent called this BLOCKING; I downgraded it — the feature works, it is simply not switched on, and switching it on is a one-field config change.)*
- **`holdLockSuppressesClose` only inspects `close_long`/`close_short`.** An AI `open_*` decision can still reach the reconcile-before-open flatten path (`auto_trader_orders.go:398,677` → `:336-370`) and close a live NT8 position. HIGH, size M.
- **`enforceEODFlat` only flattens STORE-known positions** (`auto_trader_clock.go:229`) — an NT8 orphan is never flattened — and it sits below two early returns in `runCycle` (`auto_trader_loop.go:153`, after the isRunning and multi-account gates), so a stopped trader never reaches EOD-flat. MEDIUM.
- Confirmed positive: **EOD-flat does bypass hold-lock** (RECON #10) — it calls `at.trader.CloseLong/CloseShort` directly and never enters `executeDecisionWithRecord`. Proven by code; no test exercises it.

**5.10 OBSERVABILITY — FINDING, 3 blind spots of 5.** Present and correct: **dark regime** (P1 alert, wired end-to-end) and **plan lifecycle/close** alerts — 9 `emitAlert` production sites total. Blind spots, all journald-only with nothing in-app:
- **Clock jump / host sleep.** `kernel/clock_drift.go:78` does block entries on >60s drift (I saw it fire in the test run) but only logs + increments an in-memory counter — no `emitAlert`. Worse, a laptop that suspends past a session's read *window* means **no plan is ever written for that session** and nothing says so.
- **Reconciliation break / A4 FREEZE.** `reconcile.go:135,149` logs `🚨 QTY DIVERGENCE` and `🚨 A4 FREEZE … trader FROZEN` at Error level — the owner-facing surface has neither.
- **B3 rate breaker (order runaway).** `tcp_trader.go:251` logs only, and its counter is filed under the process-wide `""` trader bucket.
- **Silent LLM drift.** `ValidatePlanDoc` bounds levels only from **above** (`plan_doc.go:101`) — a plan with **zero levels** validates so long as it has one scenario with an id, and `Trigger`/`Invalid` are never checked non-empty. A degraded planner response is stored `lifecycle="active"` and the card renders it as a normal plan. *(Agent called this BLOCKING; downgraded to HIGH — the Go-computed KEY LEVELS block is independent and the hard gates are unaffected, so a hollow plan cannot itself cause a bad trade. It is an honesty defect.)*

**5.1 SAFETY PRECEDENCE — VERIFIED ORDER, 2 SIDE DOORS.** The real chain in `executeDecisionWithRecord` (`auto_trader_orders.go:128-289`), in runtime order: feed-down :137 · dead-man :151 · A4 freeze :167 · boot integrity :182 · consecutive-loss :199 · last-entry :216 · session gate :230 · plan-mode :244 · approval :258. Post-gate open path: reconcile-before-open :398 · max-positions :409 · same-side refusal :414 · position-value ratio :446 · max-contracts clamp :458. Per-decision validation (armor, R:R, confidence) runs earlier in `kernel/engine_position.go:34-196`.

**PROVEN POSITIVE, as the checklist demands: no plan content and no owner overlay can influence or skip any hard gate.** The only gate that reads plan content is plan-mode, it sits 8th of 9 — after every hard gate — and can only ADD a restriction. Overlays are RFC-6902 patches against the PlanDoc, which has no risk fields at all.

Deviations from the expected chain, all verified: **there is no killzone gate** — `InKillzone` is used only for post-trade A–F grading (`kernel/adherence.go:104`); armor runs before, not after, the gates. Two authenticated **side doors bypass all nine gates**: `POST /debug/nt-test-trade` (documented harness — still passes the SIM lock and B3) and the Emergency-Flat path. And one config, not plan, CAN disable a family wholesale: `risk_control.guardrails_enabled=false` skips daily loss/profit/max-trades, the blackout window and the consistency rule in one switch — your deliberate learning mode.

Also found here and **fixed** (`f7fa2d3c`): every gate sets `Success=false` + a reason and returns nil, but the caller's else-branch overwrote it with `Success=true` and logged "✓ succeeded" — so **every refused entry was recorded and displayed as an executed one**. That is the root of 5.11 Q3 being unanswerable: the record asserted the opposite outcome while the reason sat unused in the same struct.

**5.6 IDEMPOTENCY + RESTART — FINDING.** `-race` clean on every package; a 25-goroutine concurrent `AppendPlan` produced no duplicate version and no gap; `TestPlanRestartRecovery` passes. Real gaps:
- **Digest identity has no trader scope** but its text is ONE trader's P&L (`planner.go:411-431` computes per-trader, `store/digest.go:24-31` keys on symbol+date+session+kind). With two MNQ day-plan traders live, whichever wins the race writes a digest both then read. HIGH.
- **Planner-read dedupe is read → full AI call → write** with no in-flight guard, so two cycles can both pass the check and write two plan versions. HIGH.
- **Digest and alert dedupe are check-then-insert with no unique index** — duplicates are possible under concurrency. HIGH, size S (add the index).
- **Memory-only state lost on restart**: `peakPnLCache` (disarms the give-back drawdown close), `safeMode`/`consecutiveAIFailures` (re-arms entries after 3 AI failures), `lastBarCloseMs` (re-runs one cycle). MEDIUM.
- `stopUntil` is read but **never written anywhere** — the risk-control pause is inert.
- The live `2026-08-15:NY` rows carry `lifecycle='expired'`, which **no code path writes** — evidence of an out-of-band edit.

**5.7 FAIL-CLOSED — FINDING (expanded).** An agent wrote, ran and deleted a probe proving the compound case: with an empty `MarketDataMap`, price-sanity, stale-data, clock-drift **and** R:R all skip together and `open_long` proceeds. Beyond the T1 calendar fail-open I found: the **T22 stale-data gate is unreachable in production** (guarded by `len(MarketDataMap) > 0`, empty pre-fetch); **`placeEntry` never checks the TCP link**, so an entry issued while NT8 is disconnected queues in `pending`; a **DB write failure right after a real fill** logs at INFO and creates an invisible live position (the P0 fill alert is in the else-branch); a **DB read error silently disables the daily-loss and max-trades caps** (zero values used); and **zero of the eight external-dependency failure modes emits an alert**.

**5.11 OWNER LOOP — FINDING, 3 of 5 unanswerable.** Q1 (today's plan) ✅ and Q4-partial. The gaps:
- **Q2 "which play is armed?" — unanswerable.** The card's scenario status is a passthrough of `system_config` key `scenario_status:<plan_id>`, and the ONLY writer in the repo is `cmd/sandbox-seed/main.go:274`. In production the key never exists, so every scenario paints the same fallback.
- **Q3 "why was an entry refused?" — was actively wrong, now fixed** (`f7fa2d3c`). Still no UI surface: `/api/risk/gate-blocks` has **zero frontend consumers** and the counters are in-memory.
- **Q4 "what did I change?" — adds are attributable (👤 chip), edits and deletes are not**: they go through the overlay path with no version bump, no op list, no owner marker.
- **Q5 "what did the AI propose and what did I do?" — declining renders "Applied — card updated"** and persists nothing (`AskPlannerPanel.tsx:228-238` sets local state only).

**5.11 OWNER LOOP — FINDING.** `telemetry.IncGateBlock` is an **in-memory map** (`telemetry/gate_blocks.go:40-47`) — counters reset on restart and there is no `gate_blocks` table. So "why was an entry refused?" has no durable record; the journal line is the only trace and it is not in the UI. Full screen-by-screen answers in the appendix.

**5.12 ROLLBACK — FIXED.** `deploy/RESTORE.md` documented and tested the **DB** restore but not the **binary** rollback — and after reverting to an older binary, `deploy/RELEASE` still names the newer revision, so `AssertBootIntegrity` (`kernel/boot_integrity.go:135-138`) sets `Refused=true`: **entries blocked, everything else read-only**. The bot comes up looking healthy and silently takes no trades. `6fe92f16` adds the rollback sequence with the re-arm as a numbered step, the journald line to confirm, the blank-value escape hatch for bisecting, and DB-first ordering across a migration.

---

## EXIT BAR

`go build` ✅ · `go vet` ✅ · `go test ./...` ✅ · **`go test -race ./...` ✅ clean** · `tsc --noEmit` ✅ · `npm run build` ✅ · vitest **177/178** (the one failure — `RegistrationDisabled` "NoFx Logo" alt text — and the `e2e/gate.spec.ts` collection error are **pre-existing and untouched**, confirmed against the files I changed) · **goldens byte-identical** (`git diff kernel/testdata/` empty) · Playwright green with screenshot · config-truth run on every field touched.

**Adversarial self-review** (run against my own findings): two challenged, one overturned. (a) *"ASIA fires a duplicate read"* → **WRONG**, `IsCMEOpen` blocks 16:55 entirely; corrected to "the designed read never fires". (b) *"the gates get conf=50/RR=1.0"* → **HOLDS**: `GetConfig()` returns a pointer, `ClampLimits()` mutates in place at :247, `GetRiskControlConfig()` reads the same struct at :296. Also disproved before reporting: the digest's date/window mix (correct by design) and the `quantity=0` row (demo data, not a TP-path bug).

---

## REMAINING — numbered, each with severity, size, recommendation

| # | Finding | Sev | Size | Recommendation |
|---|---|---|---|---|
| 1 | `sessions[].enable` honored by 2 of 9 sites | HIGH | M | Convert the 7 to `sessionRunnable`. **Until then, switch ASIA+LONDON back OFF** — today they cost LLM calls and change nothing |
| 2 | ASIA's 16:55 read blocked by `IsCMEOpen` (spec requires it) | HIGH | S | Allow the pre-open read inside the break window, or move `ReadCT` to 17:05 |
| 3 | `proximity_filter_atr` shadowed at 3 kernel sites | HIGH | M | Thread a `proximityK` param through `ScoreLevels`/`RoundNumberLevels`/`RenderPlanStatus`; fix all three or it stays half-honoured |
| 4 | R:R floor 1.0 and confidence floor 50 vs researched 3.0/65 | HIGH | S | Set `min_risk_reward_ratio` + `min_confidence` on both live strategies (config, no code) |
| 5 | `max_levels` >8 / `scenario_cap` >3 reject the whole plan | MED | M | Parameterize `ValidatePlanDoc` caps; keep 12/5 as hard ceilings |
| 6 | T1 blackout fails open on a missing/failed calendar slice | MED-HIGH | S | Distinguish "no events" from "no data"; alert + fail-closed on the latter |
| 7 | ASIA plan identity rolls at midnight mid-session | MED | M | Key plans by the session instance (window start), not the calendar date |
| 8 | Demo rows in the live DB (+ the documented cleanup is wrong) | MED | S | Run the corrected command below, after a backup |
| 9 | Executor uses `DefaultSessionRegistry`, not the admin one | MED | M | Add a registry provider seam to the kernel, mirroring the bars/nPOC seams |
| 10 | Digest omits adherence/MAE/MFE — learning loop open | MED | M | Extend `FormatSessionDigest`; also `dayType` is always `""` |
| 11 | `planner_timeframes` asserts a read-set never fetched | MED | S | Either fetch the configured TFs or stop claiming them in the prompt |
| 12 | Gate-block reasons in-memory only | MED | M | Persist a `gate_blocks` table; surface in the card |
| 13 | Re-plan budget hardcoded in the executor prompt | MED | S | `ReplanCapFor` at `planner.go:580` |
| 14 | FE `SESSION_BANDS` duplicates the registry clock | MED | M | Serve the registry to the FE (the tabs already do this) |
| 15 | `day_plan` never range-clamped at save | LOW-MED | S | Add a DayPlan branch to `ClampLimits` |
| 16 | Alert dedupe is check-then-insert, no unique index | LOW | S | Add a unique index on `(trader_id, event_id)` |
| 17 | `/api/plan/alerts` has no user→trader ownership check | LOW | S | Low risk while loopback-bound; fix with the next auth pass |
| 18 | `"MNQ"` hardcoded at 6 api sites + 2 FE | LOW | M | Blocks multi-symbol later, harmless today |
| 19 | AI cost rates hardcoded (`handler_plan.go:1276`) | LOW | S | Read from the model price map |
| 20 | FE-only rules with no backend counterpart (conflict band 3pts, near 12pts, instruction verbs) | LOW | S | Either back them with Go or label them as UI heuristics |
| 21 | Deleting a 👤 owner level leaves the `owner_levels` row active — sticky levels return on the next plan | MED | S | `planApi.deleteOwnerLevel` and `OwnerLevelStore.Delete/MarkConsumed` have **no production caller**; wire the delete |
| 22 | `vix_level` is dark forever → DEGRADED baseline is 1/7 on a healthy read | MED | S | Exclude `vix_level` from the count until a VIX feed exists, or wire one |
| 23 | Volume toggle half-inert on the futures prompt path | MED | S | Gate the OHLCV volume column on `EnableVolume` (`engine_prompt.go:710-719`) |
| 24 | No `test` op is ever emitted → the §42 overlay concurrency guard is never armed | MED | S | Emit a `test` op with the level's current value so a stale index 409s |
| 25 | `replan_cap=0` / `realign_cap=0` are unreachable (FE offers 0; Go reads `> 0`) | LOW-MED | S | Use `>= 0` like the per-session twin already does, or raise the FE min to 1 |
| 26 | `expired`/`died`/`superseded` never written → HandoverBanner is dead code | LOW-MED | M | Either write the lifecycle transitions or drop the banner |
| 27 | Post-apply gold flash never renders (`data-flash` handed to a component that spreads nothing) | LOW | S | Spread the attribute in `ZoneRow`, or drop it |
| 28 | `night` (registry) and `runnable_sessions` (strategy) are two authorities in one payload | LOW | S | Derive `night` from the same resolver once item 1 lands |
| 29 | Scenario status has no production writer — every play paints the same fallback (Q2 unanswerable) | HIGH | M | Compute status in the executor; the seeder is the only writer today |
| 30 | `/api/risk/gate-blocks` has **zero** frontend consumers (Q3 has no surface) | HIGH | M | Render refusals on the card; pairs with #12 |
| 31 | Owner edits/deletes leave no visible trace (Q4) | HIGH | M | Stamp overlay ops with an owner marker + show an op list |
| 32 | Declining an Ask-Planner proposal renders "Applied — card updated" (Q5) | HIGH | S | Distinguish declined from applied; persist the decision |
| 33 | Session digest is keyed without trader scope but holds one trader's P&L | HIGH | M | Add trader_id to the digest identity |
| 34 | Planner-read dedupe is read → AI call → write with no in-flight guard | HIGH | M | Hold a claim row / in-flight flag across the call |
| 35 | Digest + alert dedupe are check-then-insert with no unique index | HIGH | S | Add unique indexes on the identity columns |
| 36 | T22 stale-data gate unreachable in production (`len(MarketDataMap) > 0`) | HIGH | S | Move the check after the fetch |
| 37 | `placeEntry` never checks the TCP link — entries queue while NT8 is disconnected | HIGH | S | Call `IsConnected()` before sending |
| 38 | DB write failure after a real fill → invisible live position, INFO log only | HIGH | S | Emit P0 on the error branch |
| 39 | DB read error silently disables the daily-loss and max-trades caps | HIGH | S | Fail closed on the error |
| 40 | Memory-only state lost on restart (peakPnL, safeMode, lastBarCloseMs); `stopUntil` never written | MED | M | Persist or re-derive; delete the inert field |
| 41 | Zero of the 8 external-dependency failure modes emits an alert | MED | M | Wire `emitAlert` into each fail path |

---

## DEPLOY HANDOFF

Six commits. **No change to the decision, sizing, or order-routing path.**

| Commit | What | Risk |
|---|---|---|
| `6fe92f16` | RESTORE.md: binary rollback + the mandatory RELEASE re-arm | docs only |
| `a84d6ae2` | dev UI binds loopback (was proxying the LAN into :8080) | dev server only |
| `9dffec72` | tests pinning three defects | tests only |
| `e33532e2` | owner door scoped to the LIVE session — **regression fix** | FE gating, fails closed |
| `2427c850` | Ask-Planner reads plan_final, not the base doc | one handler, same fallback |
| `637b137a` | NO-TRADE renders as NO-TRADE, not gold ACTIVE | FE label only |
| `f7fa2d3c` | gate-refused entries no longer recorded as successful | record/display only |

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
git rev-parse HEAD > /tmp/rel && { grep '^#' deploy/RELEASE; cat /tmp/rel; } > deploy/RELEASE.new && mv deploy/RELEASE.new deploy/RELEASE   # MANDATORY re-arm
sudo systemctl restart nofx
journalctl -u nofx --since '2 min ago' | grep 'BOOT INTEGRITY'    # must read expected == rev
cd web && npm run build && cd ..
# restart the dev UI so the loopback bind takes effect, then hard-reload the browser
```

**Corrected demo cleanup** (replaces the wrong one in `2026-08-16-demo-plan-seed.md`; back up first):

```bash
cp ~/nofx/data/data.db ~/nofx-backups/pre-demo-cleanup-$(date +%F).db
cd /home/hoang/nofx && sqlite3 data/data.db "
DELETE FROM plans            WHERE trigger_reason = 'demo_seed';
DELETE FROM plan_overlays    WHERE plan_id = '2026-08-16:NY';
DELETE FROM plan_qa          WHERE plan_id = '2026-08-16:NY';
DELETE FROM day_plan_alerts  WHERE event_id LIKE 'demo:%';
DELETE FROM owner_levels     WHERE label = 'DEMO D-zone';
DELETE FROM trader_positions WHERE source = 'demo_seed';
" && sqlite3 "file:data/data.db?mode=ro" "
SELECT 'real plans kept: '||COUNT(*) FROM plans WHERE trigger_reason <> 'demo_seed';
SELECT 'demo rows left:  '||(
  (SELECT COUNT(*) FROM plans WHERE trigger_reason='demo_seed')
 +(SELECT COUNT(*) FROM day_plan_alerts WHERE event_id LIKE 'demo:%')
 +(SELECT COUNT(*) FROM trader_positions WHERE source='demo_seed'));"
```

**Note on `deploy/RELEASE`:** your working tree holds `5472f316` uncommitted. After pulling these commits and rebuilding, that value no longer matches the binary — the re-arm step above is not optional, or the bot boots with `TRADING REFUSED`.

**Before Monday, the two config decisions that are yours:** switch ASIA+LONDON off until item 1 lands, and decide whether R:R 1.0 / confidence 50 is what you want the bot trading at (item 4).
