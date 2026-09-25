# Veteran deep review — Section 05: execution and trading method

Owner: hoang · 2026-09-05 · branch `docs/vet-05-0905-complete` · source base **b4376246** · Section 5 only.

## One-page summary

**Verdict: historical execution economics are negative; current strict execution profitability is UNMEASURABLE; live readiness is UNVERIFIED with two PROVEN stop-entry defects.** I would keep the present one-contract SIM baseline while measuring a coherent entry-to-exit method. Fixing order correctness is necessary, but it does not establish an edge. My first-person judgments below are [I], untested here; I claim no professional biography.

**My three biggest problems:**

1. **[T] The payoff does not cover the loss frequency.** The corrected 58-position population has **18 wins, 38 losses, two flats**, **−$466.428572**, **−$8.041872/trade**, across **12 CME session-days**. Winners average $125.472222 and losers −$71.708647: payoff **1.749750**, versus **2.111111 needed at the observed nonflat win frequency**. Win frequency is **18/56 = 32.1%, Wilson 95% 21.4–45.2%**. These are historical mixed-rule results; there are zero realized trades after the supplied strict boot. Source: E1/E4 below.
2. **[T→I] Entry, invalidation and payout describe different trades.** Arm 35 entered at its exact submitted limit and stopped at its exact accepted broker stop. Its loss cannot be explained by slippage. Yet its fade was invalidated on a closed bar well before the protective stop, while the first authored target paid less than 1R and the worked target barely paid 2R. I would test a mechanical thesis exit before widening entry discretion or choosing targets from hindsight. Q1/Q5, position 591.
3. **[A, S/BROKEN] The continuation method has no valid execution test.** C# places the intended stop trigger in the limit-price slot and Go applies a limit-side guard to stop entries. **21 sent rows, one repeated scenario, zero fills** is an operational failure, not 21 losing ideas or proof against continuation. The excursion table is empty and daily-loss switches remain off. Q2/Q4/Q6.

**My three biggest opportunities:**

1. **Make entry permission depend on the actual trade geometry.** Preserve cancellation for crossed fades; a provisional continuation cap of four ticks is a shadow candidate only. Arm 35 could not tolerate even one adverse tick while retaining 2:1 against its accepted bracket. Q3, E6.
2. **Separate thesis failure from emergency protection.** Shadow the authored closed-bar invalidation with the accepted protective stop retained; compare against the unchanged stop/target/session-flat baseline on identical opportunities. Winner MAE proxies do not identify the optimal stop. Q5.
3. **Measure opportunities and executable costs, not message volume.** The aligned funnel is **22 authored enabled identities → 7 armed → 4 sent → 2 range-reached proxies → 1 filled → 0 won**. Find why fifteen identities never armed before relaxing gates, and record quotes, initial broker prices and excursion timing before estimating live fills. Q2/Q4.

## Evidence contract, exclusions and supersession

[A] means source/store/log observation; [T] is a tape measurement; [I] is my analytical judgment, untested here. [R] is primary-source research/documentation, scoped to what it establishes. I invoke no named market study to justify an MNQ trading edge, ATR multiplier, cap or exit. Vendor/exchange order documentation does not establish strategy profitability.

All current scripts connect with **`mode=ro` and `PRAGMA query_only=ON`**. I did not invoke `cmd/gate-jwt`, application initialization, broker commands, configuration changes or runtime changes. A read-only unauthenticated `/api/health` returned **HTTP 200**, reporting **36648655cfe0** at 17:35 CT; this report analyzes requested dev **b4376246**, not an assertion that dev is deployed. Source differences matter particularly for the daily-loss latch. The user's explicit dev worktree and retention instructions supersede the checklist's running-revision/removal convention. Parent owns integration; the main lock and other worktrees are untouched.

Performance window is **2026-08-15 00:00 CT inclusive, epoch 1786770000000, to 2026-09-05 00:00 CT exclusive, epoch 1788584400000**. I use `pnl_corrected` only; exclude test source, NULL corrected P&L, unresolved close/correction flags, and **`upper(trim(coalesce(plan_id,'')))='UNRESOLVABLE'`**. Of 71 era rows, exclude **530,539,545,546,566,571,580** (plan sentinel), **572–574** (test), **576,577,579** (NULL). The exact eligible manifest P is E1. Position 566's missing extrema are irrelevant to primary eligibility: its sentinel excludes its dollars and outcome too.

**This report and its `execution-data/complete/` outputs supersede every conflicting q01–q34 historical conclusion, including the previous 65-row report.** Historical files outside `complete/` remain preserved as provenance, not primary data. I withdraw the old 65-row P&L/payoff, 62-bar fill tables, 20/42 excursion cohorts, 44/65 stop and 13/65 target tables, eleven-winner floor comparison, and fourteen-calendar-date statement. Current calculations use **58 trades, 55 containing bars per leg, 18/38 excursion cohorts, 40/58 stop labels, 11/58 target labels, ten fresh floor proxies, twelve CT dates and twelve CME days**. No realized current-strict claim can be drawn from either historical population.

Evidence paths below are relative to `docs/superpowers/reports/2026-09-05-vet-05-execution-data/`; each contains row ids and reproducible query logic. `complete/` always means the new run. Source citations refer to files at b4376246. All trading times are CT unless explicitly stated otherwise.

| Evidence key | Committed path:line / contents |
|---|---|
| E1 | `complete/q31_verified.out:1`: P, winner/loser/flat ids, performance; query in `complete/q31_verified.py:19`; excluded ids in `complete/q35_complete.out:2` |
| E2 | `complete/q31_verified.out:2`: MAE/MFE proxy ids and percentiles; `:3`: exit reasons; `:4`: floor comparison; contributing bars in `complete/q31_verified.json` |
| E3 | `complete/q31_verified.out:5`: fill rates; **all 116 entry/exit rows** in `complete/eligible_fill_audit.csv:2`; provenance groups `complete/q35_complete.out:3` |
| E4 | `complete/q34_integration.out:1`: day count/strict absence; exact day manifests and bar 440248 in `complete/q34_integration.json` |
| E5 | `complete/q35_complete.out:5`: consistent scenario-identity funnel; `complete/q31_verified.json`: all arms, signal ids and reached-proxy bar ids |
| E6 | `complete/q35_complete.out:7`: arm35 geometry; `complete/q34_integration.out:2`: zero broker slippage vs 3.3715271-point ledger drift |
| E7 | `complete/q32_sources.out:1`: fresh code, selected NT8 logs with original filenames/line numbers, selected strategy flags; `complete/q37_sources.out:1`: supplementary source excerpts |
| E8 | `complete/raw_arm35_sources.out:7`: exact placement-time ATR; `complete/raw_guard_sources.out:5`: arm33 actual decision price; `complete/raw_close_sources.out:1`: fresh exit-label raw lines |
| E9 | `complete/q37_sources.out:1`: NY registry 14:45, offset resolver, selected stored session fields; `complete/q33_metrics.json`: all bar-bucket ids |

Wilson 95% intervals accompany empirical rates; n=0 has no defined interval. Revisions, attempts and trades share sessions, so Wilson intervals are descriptive and cannot manufacture independent samples. Percentiles are linearly interpolated descriptive quantiles, not confidence bounds or risk prescriptions. Payoff ratios and their algebraic break-even probabilities are arithmetic summaries, not observed rates. No initial-risk distribution or all-in live-cost estimate is invented.

## Q1 — One arm end to end, with row ids

Arm **armed_orders.id 35** → position **trader_positions.id 591**, 2026-09-03, NY session.

**Authored [A].** `plans` row `plan_id=2026-09-03:NY:8d5c8af5…`, `version=2`,
`lifecycle=active`, `created_at=2026-09-03 13:45:05 UTC` (08:45:05 CT; `plans.created_at` is
UTC). Bias short, conviction medium. Scenario **S1**: condition `reject`, direction short,
trigger "A retest of 29285.00 OR-H stalls after the 08:30 failure; short the touch", confirm
`{rule: touch, ref_price: 29285, side: below}`, invalid "A 5m close above 29293.00 ONH voids
the fade", targets [29233.08, 29222.75, 29144.5], quality B, `arm {enabled: true, entry: 29285,
stop: 29340, target: 29144.5, wait_confirm: true}` (`complete/q35_complete.json`, `plan35`).

**Composed (0B stop) [A].** Go log 09-03 09:02:54 CT (in `data/nofx_2026-09-02.log` — the
file rotates on process start, not on date):
`🛑 arm stop NY S1 leg 1 short: stop 29354.91 (authored 29340.00 WIDENED) · anchor ONH
29293.00 → beyond 29293.50`. The complete placement log explicitly records **1.5×ATR5m 46.61**, `bound=atr_floor` (`data/nofx_2026-09-02.log:84978`, preserved in `complete/raw_arm35_sources.out:7`). A later-cycle floor observation (not the placement-time ATR): `📓 read facts: void=20 · floor=67.6 pts
(1.5×ATR5m 45.10)` (09:06:54). The placement stop implies 69.91 pts of risk; the later 67.6-pt floor is a different observation. Composition drifted between
29349.90 (09:00:54) and 29354.91 (09:02:54). Code: `kernel/min_sl.go:34` `MinSLATRMultDefault =
1.5`; `:40` `MinSLTickClearance = 2` (the "beyond 29293.50").

**Gated [A].** Same second: `⚔️ arm S1 leg 1 wait_confirm MET (touch) — arming`; `🚫 no-chase:
arm S1 has no recorded touch — run stored NULL, judged on dist alone (0.00×ATR)`;
`armGateVerdictFor` (`trader/armed_executor.go:1349`) returned "" (no `⚔️ arm REFUSED` line);
`⚔️ armed NY S1 leg 1 short limit 29285.00 SL 29354.91 TP 29144.50`. R:R at composition =
(29285 − 29144.5) / (29354.91 − 29285) = 140.5 / 69.91 = **2.01** — close to
the 2.0 floor (this matters in Q3).

**Placed [A].** `📌 armed S1 → WORKING limit 29285.00 signal=f2b1eb20-… (band ±100t)` —
`armedPlaceTicks()` returns 100 (`trader/armed_executor.go:51`) → a 25-pt band around the level;
`runArmedPlacement` `:963-972` → `PlaceLimitEntry` (`trader/ninjatrader/tcp_trader.go:433`).
Ledger row 35: `state=working`, `signal_id=f2b1eb20`, `kind=limit`, `created_at 09:02:54 CT`.

**NT8 [A]** (`log.20260903.00000.txt`): `09:02:54:664 signal f2b1eb20 routed to account
'Sim101' (resolved, sim, connected)`; `:672 Submitted … Action='Sell short' Limit price=29285
Stop price=0 Quantity=1 Type='Limit' Time in force=DAY`; `:780 Accepted` then `Working`;
**`09:03:53:694 Filled`** (59 s after placement); `:702` bracket `-sl` `Type='Stop Market'
Limit price=0 Stop price=29355` and `:707` `-tp` `Type='Limit' Limit price=29144.5`; both
Accepted `:811`. Note the SL at **29355** = round-to-tick of the composed 29354.91.

**order_update → Go [A].** `📡 armed order_update summary (1-line/min): frames=1
initialized=1` and `⚡ armed fill S1 @ 29285.00 (entry_class=armed_fill — stale_reeval NOT
applied)` at **09:05:35** — 102 s after NT8's Filled — because the cycle that placed the order
went straight into `🤖 Requesting AI analysis…` at 09:02:54 and came back at 09:05:35 (`⏱️ AI
call (reasoning=fast→low) duration: 157.03 seconds`; `⏱ cycle overran the scan interval
(2m41.166s > 2m0s)`). `consumeArmedOrderUpdates` is called only from `trader/armed_executor.go:978`.
Ledger row 35 → `filled`, `fill_price 29285` (`:1206-1207`).

**Materialized [A].** `trader/ninjatrader/reconcile.go:395` 09:03:54 `NT8 holds UNTRACKED position
MNQ SHORT @ avg 29285.00 (no open row) — materializing after 60s if it persists` →
`trader/ninjatrader/reconcile.go:436` **09:05:14** `MATERIALIZED untracked NT8 position MNQ SHORT qty=1 @
29285.00 (acct=Sim101)` → `trader_positions.id 591`, `source=reconcile`, `entry_time
09:05:14 CT` (81 s after the fill — the reconcile time-stamp trap), `plan_band=armed_fill`,
`cited_scenario_id=S1`, `plan_matched=1`, `adherence_grade=A`. The executor's own
`materializeArmedEntry` (`:1233`) ran at 09:05:35 and reported `position row not materialized
yet — stamp pending (reconcile completes it)` — 21 s after the reconcile row already existed
[B: the stamp's lookup did not find it; the final row is stamped anyway].

**Ledger vs broker stop [A].** `armed_orders.35.stop_px = 29351.6284728996` while NT8's stop
is 29355: the row's prices were rewritten in place by the 09:05:35 cycle's `UpsertArm` (stop
29351.63 in that cycle's composition line) after the broker already held 29355. This is the
in-place-overwrite class the two-day audit called D12/D40; fixed the next morning by
`3b8d6cd6` (`store/armed_orders.go:201-208`, "refusing to rewrite — the row is working").

**Invalidation while open [A].** 09:15:01 `🎯 scenario S1 → ≈invalidated @ 29285.00 (price
accepted through the level against the trade — … display-only estimate, never exec`. The
09:10 5m bar (complete/q34_integration.json, bar 440248) closed 29301.50 above ONH 29293. Position open at ≈ −16 pts. Nothing acted.

**Excursions.** `trade_excursions`: 0 rows. `trader_positions.591`: `mae=75.0`, `mfe=43.5`.
Close-time grader line 09:20:47: `📐🎓 MNQ close: MAE 75.00 / MFE 43.50 pts · adherence D
(off-plan) cited="" matched=false` — the stored row says A / S1 / matched=1 (see Surprises).

**Exit [A].** NT8 `09:20:45:677 … -sl New state='Filled' … Stop price=29355 Type='Stop
Market'`; `-tp Cancelled :782`. Go, same second: `trader/ninjatrader/close_sync.go:180 📊 exit fill recorded: MNQ
SHORT qty=1.00 @ 29355.00 (tick-exact, pnl -140.00)`; `store/position_builder.go:162 ✅ Full close
… PnL: -140.00`; `trader/ninjatrader/close_sync.go:196 📕 NT position closed … reason=sl`. `trader_fills.433`
(`nt8-exit-591`, BUY 29355, realized −140). `trader_positions.591`: `exit_price 29355`,
`pnl_corrected −140.0`, `close_reason 'sync'`, `pnl_correction_note "T7 close-path:
pnl_corrected = recompute (Δ+0.00 vs realized)"`. The 09:20 1m bar: o 29343.5 **h 29360.0**
l 29342.25 c 29348.0 (complete/q37_sources.out, exit-bar query) — SIM filled the stop-market at exactly 29355 while the bar traded
5 pts (20 ticks) beyond it. The logs show same-second close synchronization in this case; the generic fill consumer also
updates its cache asynchronously (`trader/ninjatrader/tcp_trader.go:232-245`). I distinguish that
cache from the arm ledger, whose `order_update` drain is cycle-gated.

**What the trace teaches.** Nine hops, four clocks (plan UTC, log CT, NT8 local, epoch-ms), one
real defect surfaced by the timing (arm ledger updated 102 s late), one by the prices (ledger stop ≠
broker stop, since fixed), one by the plan itself (invalidation with no exit).


**Integration correction — actual stop slippage [A].** For position **591**, the accepted far-side protective order `f2b1eb20-f89e-4a88-bcff-5ce971d17861-sl` had **Stop price 29355** at **09:03:53.811**, and filled at **29355** at **09:20:45.677**. Raw NT8 `log.20260903.00000.en.txt:6085,6260` is retained with line numbers in complete/q32_sources.out. Thus **29355 − 29355 = 0 pts / 0 ticks** against the broker-authorized stop. **29355 − armed_orders.35.stop_px 29351.62847289964 = 3.3715271 pts** is ledger-to-broker geometry drift, not execution slippage. The earlier composition log was 29354.91, rounded to the accepted 29355. I disagree with any section-04/08 label of this 3.37-point difference as slippage; the far-side stop is the required reference. complete/q34_integration.json independently records the subtraction and ids. The malformed *entry* stops on September 4 are a separate defect from this correctly formed September 3 protective bracket.

**Trader implication [T→I].** I see a rejection thesis carrying a volatility-sized catastrophe stop. The plan says a 5m close above 29293 voids the fade, while its accepted protective stop is 29355. The 09:10 bar **440248**, known only at its 09:15 close, closes 29301.50. That is a possible mechanical thesis-exit event, not an executable fill price. I would test whether continuing to hold after that event pays for the extra risk, using every comparable signal and subsequent tape. I cannot claim a saving from this one losing example.

The accepted bracket has **70 points initial risk and 140.5 points reward = 2.0071:1**. The first authored target, 29233.08, offers only **51.92 points = 0.7417R**. Thus the plan's nearby target narrative and its far broker target imply different exit methods. With one contract I cannot scale out at the first target and retain a runner. I would require one explicit exit contract before entry: where the thesis ends, where the protective stop sits, and which single target is actually worked. This is a specification recommendation, not evidence that taking the first target is profitable. The only initial-R arithmetic I treat as verified here is this broker-linked trade (`complete/q35_complete.out:7`).

## Q2 — Every fill against its bar; slippage; SIM versus live

**[T] I rebuilt both legs for each of the 58 eligible positions, not a convenience sample of fill rows.** E3 supplies 116 rows with position id, price, timestamp source, nearest fill candidate, containing bar id/high/low and extreme flags. This is a per-position leg audit: a position may summarize executions, so it is not proof of complete exchange execution coverage. Raw execution-id coverage across the cohort is **UNMEASURABLE** without an immutable position/order/execution join and complete broker executions.

Entry timestamps: **47 price/side/nearest-time fill matches, nine NT8 signal-log matches, two late materialization proxies (567,569)**. Exit timestamps: **35 price/side/nearest-time matches and 23 position-close timestamps**. Approximate price/time joins are not execution-id proofs; candidate fill ids must not be treated as established foreign keys. The corrected script retains actual selected timestamps, not the containing minute's opening time. Both legs lack bars for **521,522,523**, leaving 55 per leg.

| Bar observation; tolerance for an extreme <0.126 points | Entry | Exit |
|---|---|---|
| Inside containing range | **54/55 = 98.2%, Wilson 90.4–99.7%** | **54/55 = 98.2%, Wilson 90.4–99.7%** |
| Adverse extreme | **1/55 = 1.8%, 0.3–9.6%**, 537 at low | **1/55 = 1.8%, 0.3–9.6%**, 585 at high |
| Favorable extreme | **0/55 = 0%, 0–6.5%** | **0/55 = 0%, 0–6.5%** |

Every numerator/denominator id is in E3. Outside-range entry **569** uses late materialization; outside-range exit **578** is a reconstructed netting close. These are provenance problems, not established impossible broker fills. The excluded sentinels **566,571,580** no longer inflate outside-range entries. A containing bar includes prices from before and after a fill, so it cannot measure available price, first-touch priority, adverse selection or queue position. I explicitly withdraw the earlier “14/14 first-touch limit fills” claim. Arm35 alone has a directly inspected submitted-limit-to-entry equality in this trace.

**[A] AddOn `slippageTicks`: computed and emitted, not consumed as a production Go measure.** `ninjascript/VLTraderTCPClient.cs:1376` computes `(AverageFillPrice−origEntry)/tick` at `:1383` and emits `slippage_ticks` at `:1430`. Go declares `SlippageTicks` at `provider/ninjatrader/tcp_framing.go:86`; E7's production search finds only the declaration and mock initialization, not persistence/decision use. A short needs its sign reversed for adverse-price cost. Reading a JSON field into a struct is not using it as a metric.

**What the NT8 log says:** the fresh E7 scan finds no “slippage” text in the ten September 3–4 log files (including localized duplicates). This does not mean missing slippage equals zero. For signal **f2b1eb20…**, entry was **29285 versus submitted 29285**, and protective stop fill was **29355 versus accepted 29355**, so both price differences are zero for those verified comparisons. The latter is **zero points/zero ticks**, not 3.37 points: that number is drift to mutable `armed_orders.35.stop_px`. A cohort slippage distribution is **UNMEASURABLE** without side-normalized broker-reference and fill pairs for each execution. Cycle-start-bar drift is not a substitute.

**[R→I] SIM limits, with transferability stated.** NinjaTrader's [NT8 Simulator documentation](https://ninjatrader-devel.ninjatrader.com/support/helpguides/nt8/simulation.htm) says its engine models bid/ask volume, trading volume, queue time and state delays. I therefore do not claim SIM ignores queues or fills every touch. It simulates them; this sample does not measure actual live queue priority, executable depth or market impact. I would record decision quote → submission quote → accepted order → fill → short-horizon markouts, treating nonfills as outcomes, before attributing a fade's loss to entry execution. That measurement recommendation is [I].

At one contract there is no smaller contract fraction to test the multi-contract partial pathway. The AddOn emits `partial` at `VLTraderTCPClient.cs:1343`, but submits its bracket on **full Filled** at `:1350`. Go caches partial/fill updates (`trader/ninjatrader/tcp_trader.go:232`). Larger size could leave an executed partial without that bracket until completion; recovery behavior must be tested separately. Zero one-contract partials would not validate two contracts.

A stop trigger does not promise a fill at the trigger. [CME iLink order semantics](https://cmegroupclientsite.atlassian.net/wiki/spaces/EPICSANDBOX/pages/457227032/) include stop-with-protection and stop-limit orders, with possible resting residual quantity at the protection/limit price. Those are exchange semantics; NT8's displayed `Stop Market` label does not prove the exact live routing or protection settings. I would verify the selected broker route and failure behavior, not assume unlimited slippage or guaranteed liquidation.

**14:45 flat:** NY's session-flat baseline uses `trader/auto_trader_clock.go:454`; the old standalone `eodFlatCT` default is explicitly legacy, so a literal string alone does not prove runtime timing. E7 includes the session-registry/offset source. E9 verifies the default NY registry end/flat 14:45 and the stored strategy has no offset override; this is source/store evidence, not proof of every runtime flat execution. The independent bar-context sample in `complete/q33_metrics.json` has **120 14:40–14:49 bars**, mean volume **2,394**, mean range **9.91 points**, versus **120 08:30–08:39 bars**, **13,381 / 38.24**. These are twelve dates' bar buckets, not a 58-trade population or depth estimates. Lower volume than the opening burst does not establish unacceptable flat liquidity. Actual scheduled-flat slippage and executable depth are **UNMEASURABLE** without reason-tagged flat executions, contemporaneous quotes/depth and account-route data. I would keep the scheduled flat while measuring its cost, not move it on a “thin tape” assumption.

## Q3 — Marketable guard: verification, trader ruling and N ticks

**[A/T] The 5/15 and three-fill claims belong to the September 1–3 arm window, ids 23–37.** Guard-cancel ids **25,27,33,34,36** are **5/15 = 33.3%, Wilson 15.2–58.3%**; filled ids **24,28,35** are **3/15 = 20.0%, 7.0–45.2%**, mapping to positions **584,586,591**, all eligible. Since September 2, calendar-created arms contain **37 rows** and only guard ids **33,34,36**: **3/37 = 8.1%, 2.8–21.3%**. The raw prior-window id query is `complete/q33_metrics.json`, the guard rows are in q31, and E8 gives decision logs. These cohorts must not be mixed.

**I correct the “over 1.7 points” premise, and the prior report's claim that the exact price was unavailable.** Arm **33**, short entry **29166.80**, cancelled at **14:10:29 CT**. The raw Go log explicitly says **price 29167.50** (`data/nofx_2026-09-02.log:40385`; `complete/raw_guard_sources.out:5`). The actual logged guard distance is **0.70 points**. Containing bar **435638** later reaches high **29168.50**, giving **1.70**, while its final close happens to equal the logged 29167.50. The bar alone could not establish decision price; the raw log can. Full precision beyond the log's two decimals and executable bid/ask remain unavailable. The same extraction gives logged through-distances **28.00, 2.96, 0.70, 4.75, 7.70 points** for ids **25,27,33,34,36**, respectively (row/side/time matches; rounded logging precision).

**[I] My ruling is cancel crossed fades now; no bounded market entry.** A fade is a rejection trade. Entering after the market has already crossed its limit tests a different timing rule even if a marketable limit improves the fill price. `trader/armed_executor.go:955` implements cancellation; `:985` is the limit-side predicate. The comment “fills at a worse price” is not a limit-order guarantee: a limit constrains price. The concern is whether the thesis still holds. I would not remove confirmation or place orders early solely to raise the fill rate.

For a future, separately authorized continuation experiment, my central candidate is **N=4 MNQ ticks = 1.00 point**, compared with N=0/2/8 and cancellation on identical opportunities. This is **[I] a bounded experimental choice, not a measured optimum**. CME's [MNQ specification](https://www.cmegroup.com/markets/equities/nasdaq/micro-e-mini-nasdaq-100.contractSpecs.html) sets $2/point and 0.25-point ticks, hence four ticks cost $2 per contract in adverse entry movement before fees. Low nominal dollars are not permission: invalidation and worst-fill R:R dominate. A true market order cannot enforce my N; a stop-limit can limit entry price while accepting nonfill risk. Never confuse an entry cap with an emergency protective exit.

**A concrete geometry check [T]:** arm35's accepted bracket is 70 points risk / 140.5 reward. If adverse entry movement is d, R:R becomes `(140.5−d)/(70+d)`. Keeping at least 2 requires **d≤0.166667 points**, less than one 0.25-point tick. One adverse tick yields **1.9964**, four yield **1.9648** (E6). Thus even the proposed four-tick experimental cap must reduce to **zero adverse ticks for this bracket**. This is a geometry illustration for a verified fade; it is not a replay of an unobserved continuation order. I would not quietly lower 2:1 to make a cap pass.

**Guard opportunity cost is UNMEASURABLE here.** I withdraw previous q27/q28 “guard saved $186/$123,” −92.79/−61.43-point policy totals, and “all would fill” conclusions. The six ids **17,25,27,33,34,36** were simulated with a purported bounded policy that never enforced its bound, cancel-containing-bar final closes, absent queues, an invalid stop relationship for 17, and a horizon that ignored some subsequent policy events. There is no valid paired executable P&L comparison. Required inputs: exact decision quotes, order eligibility/lifetimes, tick path, immutable initial bracket, cancellation/replan events, fees and realistic nonfill modeling. I would select the policy on net incremental P&L per opportunity and tail loss across held-out session-days, not gross fill count.

## Q4 — Funnel since September 2, in one opportunity currency

**[T] I use `(plan_id,scenario)` identities throughout the primary table.** The plan-trade-date cohort September 2–4 has **eight plan ids, 58 versions, 177 scenario-versions and 57 enabled-arm versions**; deduplicating enabled identities gives **22**. Version contents can change under one identity, so this is a descriptive family-level funnel, not 22 independent stationary signals. Exact plan/version/scenario/leg and arm-id manifests are E5. Enabled-arm version share is **57/177 = 32.2%, Wilson 25.8–39.4%**, descriptive because versions repeat.

| Stage | Distinct enabled scenario identities | Conversion from previous stage, Wilson 95% | Evidence/result |
|---|---:|---|---|
| Authored with arm enabled in at least one version | 22 | Denominator | E5 `enabled_keys` |
| Armed in ledger | 7 | **7/22 = 31.8%, 16.4–52.7%** | 10 leg keys; 36 ledger rows |
| Sent/placed | 4 | **4/7 = 57.1%, 25.0–84.2%** | 24 signal-bearing rows, four leg keys |
| Reached, **whole-minute range proxy only** | 2 | **2/4 = 50.0%, 15.0–85.0%** | Arm35 and arm102 identities |
| Filled | 1 | **1/2 = 50.0%, 9.5–90.5%**, proxy denominator | Arm35 → position591; sent-identity fill rate **1/4 = 25.0%, 4.6–69.9%** |
| Won | 0 | **0/1 = 0%, 0–79.3%** | Position591, −$140 |

The primary table never substitutes placement rows for scenario identities. The exact broker-eligible reached count is **UNMEASURABLE**: raw ledger creation/update times do not establish active broker lifetime; full-minute bars omit boundary touches and can miss gap crossings; the stop order is malformed. Whole bars count arms **35,102**; a boundary-inclusive sensitivity also counts **37**, yielding **3/4 identities = 75.0%, Wilson 30.1–95.4%**. Neither sensitivity is a live election/fill probability. Actual event/tick reconstruction is required before “reached → filled” can become an executable conversion.

For operational reconciliation only, the calendar-created cohort includes **31**, created September 2 but belonging to September 1 ASIA: **37 rows, 11 legs, 25 sent rows, five sent legs**. Aligned sent rows comprise limits **35,37,39** and **21 stop-entry attempts** on September 4 NY S2; the latter ids are E5 `stop_entry_sent_ids`, ending **102 / 931a761a…**. Zero stop fills is **0/21 = 0%, Wilson 0–15.5%**, descriptive attempt count. This is one malformed continuation identity, not 21 independent setups. Signal existence proves sending, not acceptance; E7's broker records independently prove selected acceptance.

**Where it leaks most [T→I].** The largest absolute loss of identities is **22 → 7**, fifteen unarmed identities (E5 `unarmed_keys`). This does not prove fifteen missed profitable trades: some revisions are superseded, some never become eligible/current, and condition readiness differs. I would first assign each identity a time-stamped eligibility interval and one terminal reason—expired/superseded, never triggered, refused, sent, filled—so opportunity cost can be tested on the same tape. Repeated rr/invalidation refusal log counts cannot allocate those fifteen disjointly.

Downstream, the stop construction/guard must be corrected before measuring continuation timing. I would judge the system on profitable executable opportunities after costs, not on making 22 become 22 fills. My proposed diagnostic sequence is [I]: explain unarmed identities, repair malformed orders, then compare entry methods with all nonfills retained. I do not remove gates or change condition eligibility in this review.

## Q5 — Exits, stop floor, target/stop shares and the exit I would test

**[A] Design at the reviewed source:** stop composition uses the farther anchor clearance and 1.5×ATR5m floor (`kernel/min_sl.go:34`, `:40`; `trader/armed_executor.go` composition excerpt E7). BE/trailing wire changes are suspended by `trader/exit_mechs_suspend.go:35` despite strategy toggles; scheduled session-flat remains. Arm35 gives observed placement-time ATR **46.61**, widened stop **29354.91**, tick-rounded broker stop **29355**. I do not infer a universal floor from later-cycle ATR or mutable arm geometry.

**Requested `trade_excursions` MAE distribution: UNMEASURABLE, table has zero rows.** The separate stored `trader_positions.mae/mfe` values are bar-path proxies. `trader/auto_trader_clock.go:755` includes the containing entry bar; boundary bars can include pre-fill/post-exit movement, and late position timestamps can omit early movement. A stored MAE of 75 against position591's 70-point accepted stop is not proof of five-point broker slippage: the broker fill disproves that. Extrema lack ordering and executable timestamps.

| Eligible proxy cohort, exact ids E2 | MAE p50 / p80 / p95, points | MFE p50 / p80 / p95, points |
|---|---|---|
| Winners, **n=18** | **11.375 / 20.400 / 46.412** | **70.250 / 88.800 / 140.387** |
| Losers, **n=38** | **40.375 / 50.750 / 75.825** | **16.875 / 28.300 / 49.500** |

**Integration qualification:** stored zero MAE in winner IDs **569 and 584** is not proof of no adverse movement. Both are reconcile-origin rows, and the historical writer could encode uncomputed excursions as zero; the current source already leaves uncomputed values NULL (`trader/auto_trader_clock.go:752–765`, fixed in `d4aee04a`). Their exact historical validity is not recovered here. Excluding these two uncertain zero rows for sensitivity leaves **16 winner proxies: MAE p50/p80/p95 = 13.00/22.50/48.1875 points**. Excluding them from the fresh floor comparison leaves **8 winners**, ratio p50/p80/p95 **0.2931/0.5275/0.6995**, **0/8 exceedances, Wilson95 [0,32.44%]**. These remain survivor-selected proxies, not an instruction to tighten stops. Script, contributing IDs and output: `2026-09-05-vet-09-complete-data/q03_proxy_sensitivity.py:1` and `q03_proxy_sensitivity.json:1`. Raw tables below/above are retained as stored-value descriptions; the primary P&L population remains58.

All 18 eligible winners and 38 losers have these proxy fields; flat ids **542,551** are outside these outcome cohorts. Excluded 566 is not a missing eligible winner. Winner and loser ids are E1/E2, and every percentile is recomputed with linear interpolation. These results describe realized winners after existing exits; they cannot establish that a tighter stop preserves those winners, or that winners “move early.”

| Realized performance, E1 | Correct 58-row value |
|---|---:|
| Corrected P&L / mean per trade | **−$466.428572 / −$8.041872** |
| Win/loss/flat counts | **18 / 38 / 2** |
| Wins over all eligible trades | **18/58 = 31.0%, Wilson 20.6–43.8%** |
| Wins excluding flats | **18/56 = 32.1%, Wilson 21.4–45.2%** |
| Average win / average loss | **$125.472222 / −$71.708647** |
| Average-win / average-loss payoff | **1.749750** |
| Algebraic nonflat break-even win probability at that payoff | **36.366939%**, not an observed rate |
| Payoff required at the observed 18:38 win/loss counts | **2.111111**, not a fitted exit target |

This is corrected ledger P&L, not an independently reconciled all-in commission model. The prior report's −$563.93/1.6205 result was wrong for the primary population. The corrected loss remains; I have not changed eligibility to make it profitable. **Trader implication [I]:** raising target distance to the required payoff is not a solution by arithmetic—farther targets can change win frequency. The method needs a paired entry/exit test, not a prettier planned R:R.

**Stop/target shares:** I recomputed q14's side/exit-price/nearest-time log labels from raw source logs in this run, preserving original file:line references in E8. These are reconstructed reasons, not immutable exit-order foreign keys. Generic `close_reason=sync` cannot distinguish them.

| Reconstructed reason, exact ids E2 / `complete/q35_complete.out:4` | Count / 58; Wilson 95% | Corrected P&L subtotal |
|---|---|---:|
| Stop (`sl`) | **40/58 = 69.0%, 56.2–79.4%** | **−$2,569.50** |
| Target (`tp`) | **11/58 = 19.0%, 10.9–30.9%** | **$1,896.00** |
| Manual | **3/58 = 5.2%, 1.8–14.1%** | **−$6.428572** |
| Limit close | **3/58 = 5.2%, 1.8–14.1%** | **$121.50** |
| Unknown (578) | **1/58 = 1.7%, 0.3–9.1%** | **$92.00** |

Stop ids include profitable **557,570** and flat **542,551**. They are not forty losing stop-outs, and their exit distances are not automatically original risk. Reconstructed manual/limit labels do not identify scheduled-flat executions without explicit initiating reason. P&L subtotals sum to the eligible total.

**Is the floor inside or outside what winners need?** In the small available reconstruction it lies **outside observed winner MAE**, but the actual necessary initial stop is **UNMEASURABLE**. q31 uses fifteen fully closed 5m bars to average fourteen true ranges, then 1.5×; it omits stale latest bars (>300 seconds old) from the winner comparison. This is a simple-TR proxy, not the engine's exact smoothed ATR or original broker stop. Winner ids **555,557,560,569,570,575,578,581,582,584**, n=10, have MAE/floor **p50 0.161 / p80 0.481 / p95 0.680**; **0/10** exceed the proxy floor (**Wilson 0–27.8%**). Every contributing bar and freshness value is E2. Eight other eligible winners lack a fresh reconstruction. Missing initial ATR snapshots, broker stops and within-trade paths prevent causal stop optimization.

**My proposed exit method [I], shadow only:** retain one contract, the accepted emergency stop, one declared broker target and session-flat as the unchanged control. Test an additional mechanical thesis exit: for a rejection, the first fully closed specified bar that satisfies the authored invalidation; for continuation, its separately specified loss-of-acceptance condition. Evaluate only after the bar closes, use the first subsequent executable quote, retain the protective stop until flat is acknowledged, and cancel the sibling order through a race-safe path. Do not use a display card as an order signal, presume a fill at the bar close, or apply the rejection exit to every condition. Keep discretionary BE/trailing suspended; time-stop research waits for event-timed excursions. At one contract choose a single exit contract; do not invent half-contract scaling.

**Why this candidate, and what would change my mind:** position591's bar **440248** closed **29301.50 at 09:15 CT**, beyond its authored 29293 invalidation, before the 09:20:45 broker stop. That suggests a coherent alternative loss boundary; it does not prove a $108 saving. A thesis exit can also cut a winner before recovery, so I demand subsequent tape through the baseline horizon on every candidate, including winners, nonfills and EOD. If early invalidation lowers win payoff enough to offset saved losses, I keep the control.

**Evidence gate [I]:** first 30 consecutive eligible SIM closes with complete broker execution timestamps, immutable accepted initial stop/target, entry ATR/anchor snapshot, condition/version lineage, MAE/MFE timestamps, quote references, fees and boundary-ambiguity flags. Thirty is an instrumentation check, not proof of edge. Predeclare one candidate and a held-out multi-session-day comparison, preserve nonfills and all control paths, and report paired incremental net P&L with a session-day resampling confidence interval and same-bar bounds. I would require a positive lower 95% bound, no deterioration beyond an owner-agreed tail-loss tolerance, and zero unresolved order/position mismatches before considering a policy switch. If coverage or uncertainty fails, no switch. Parent/owner must define tail tolerance and authorize implementation; this report changes nothing.

**Regime boundary:** all **58** eligible trades fall in **12 CME days rolling at 17:00 CT** and **12 CT calendar dates** (E4). The supplied enforced-strict boundary is **September 3 11:10 CT**; there are **zero subsequent stored positions**, and therefore zero subsequent eligible realized trades. I verify the empty sample, not the boot assertion. n=0 strict expectancy/win rate/intervals are undefined. Historical pre-strict fills, the post-0B three-loss group **589,590,591 = −$394**, and September4 unfilled arms cannot establish strict profitability or justify optimizing its exits.

**Owner-policy distinction:** disabled guardrails were documented as a deliberate learning-mode choice (`2026-08-17-cto-final-verification.md:13`). The off switches are an operating fact, not by themselves a software defect or authorization to turn them on. The separate dev-versus-running wiring gap and entry-block-versus-flatten semantics must still be described accurately.

## Q6 — Technical live-money breakpoints (analysis only; SIM remains)

1. **[A, S/BROKEN] Stop-entry type/price contract.** `ninjascript/VLTraderTCPClient.cs:972-978` selects `StopMarket` but calls `CreateOrder(... qty, orderPx, 0, ...)`. The official [Account.CreateOrder signature](https://docs.ninjatrader.com/ninjascript/createorder) orders those arguments as limit price then stop price; the bracket code correctly uses `0, b.Sl` at **1822–1824**. complete/q32_sources.out's numbered NT8 records for signals **e38f1774** and **931a761a** show `Limit price=29590.5 Stop price=0 Type='Stop Market'`. `complete/stop_snapshot_states.json` records first distinct observed stop-order states with snapshot ids; complete/q31_verified.json maps all 21 stop ledger rows. Order 102 remained Accepted while whole-minute bars **445398,445402,445403,445406** spanned its intended trigger. **PROVEN malformed construction; observed 0/21 fills (Wilson 0.0–15.5%).** I do not assert how every live broker would reject or handle it.
2. **[A, S/BROKEN] Stop-side guard.** `trader/armed_executor.go:940` calls `limitMarketableWrongSide`; **985–996** uses `long: price<entry`, `short: price>entry`. A resting buy stop needs market below trigger; a resting sell stop needs market above trigger. Thus the predicate refuses the proper resting side and passes the already-crossed side. Equality is also unguarded. This is a two-sided source proof independent of the C# bug. `tcp_framing.go:221-228` uses a build-id comparison for far-side capability; it does not validate these price slots or stop geometry.
3. **[A, S/UNVERIFIED protection] Daily loss.** Fresh complete/q32_sources.out reads the bound strategy's master=false, daily=false, limit=450. `kernel/risk_limits.go:305-306` returns allow when the master is off. At `36648655`, analysis discarded the returned decision and could hold the AI cycle; at reviewed dev `b4376246`, `kernel/engine_analysis.go:183-199` records `RiskForceFlat` in the entry-block latch. `kernel/risk_limits.go:151-175` explicitly describes blocking entries, not closing positions. No revision activates false strategy switches. Owner policy must distinguish refusing new risk from flattening existing risk; a variable name is not protection.
4. **[A/B, A] Fill, size and AddOn bracket lifecycle.** Q1 proves lag to the arm ledger, not total broker blindness: Go has an asynchronous fill cache, and NT8 places the full-fill bracket locally. Arm placement hard-codes quantity one (`trader/armed_executor.go:945,965`). For larger sizes the entry partial is emitted before bracket submission (Q2). Code is not evidence of tested partial/cancel/fill ordering or bracket rejection recovery. Preserve one contract; demand deterministic event traces before any sizing discussion.
5. **[A/B, A] Reconnects and snapshots.** `trader/f12_leg4.go:209` defines a 30-second periodic snapshot; state-change snapshots supplement it. `complete/stop_snapshot_states.json` has freshly read snapshot ids and order prices; snapshots1604–1606 are also retained in q34/q35. Reaper class 79 already replaced silence-as-cancellation with broker-book evidence; D5 already prevents rewriting a working ledger row (`store/armed_orders.go:201-208`). These safeguards are present at reviewed dev; the stop-entry execution files are unchanged against reported running 36648655 (E7) and should not be recommended as missing. They do not prove that in-memory AddOn pending bracket state survives an NT8 restart or that a bracket is exchange-hosted across every connection type. Those are **UNVERIFIED recovery cases**, requiring controlled SIM proof in a separately authorized implementation task.
6. **[I, A/UNVERIFIED] Execution clock and stale policy inputs.** The observed cycle-coupled ledger delay motivates an independent lifecycle consumer and fresh market inputs for mechanical checks. That does not justify arbitrary faster polling, relaxing gates, or an assumed latency target. Measure broker-fill→cache→ledger→bracket acknowledgement separately, retain tail timestamps, and test duplicates/out-of-order updates. “Two-minute cycle plus AI call” is not a universal timing formula; Q1's measured 102-second ledger delay is the evidence.

**My trading priority [I].** At the first live submission, malformed order geometry is the most immediate identified breakpoint. On any filled position, the next concern is whether acknowledged protective quantity and the broker book agree during disconnect/restart; a fresh snapshot is evidence, not a guarantee between snapshots. At an eventual size increase, partial protection becomes a new risk class. Over a loss sequence, disabled or block-only daily protection fails to enforce an assumed liquidation boundary. I cannot assign probabilities or a chronological failure forecast from this SIM tape.

I would demand broker-route/account/instrument read-back, full long/short stop and bracket traces, a documented protected-order recovery action, and a reconciled loss-policy event before live consideration. I would treat loss of broker connectivity or unexplained filled quantity as an owner-action event within the existing monitoring channels, with new entries withheld under the future agreed runbook. A daily entry block can prevent the next loss while allowing the current one to deepen; the owner must choose whether and how a realized/unrealized threshold requests flattening. No runtime test, flatten, cancel or channel change is authorized by this audit. The broader live/size decision belongs to Section6 and the parent's synthesis.

## Ranked recommendations — what, why, implementation category and metric

These are proposals, not applied changes. I cross-checked `AUDIT-CHECKLIST.md:630` (exit suspension), `:1005` (excursion missingness), `:1820` (snapshot reaper), and `store/armed_orders.go:201` (working-order immutability). Existing hooks and fixed reaper/ledger safeguards should be diagnosed/verified, not recommended as absent.

| Priority / what I recommend | Why / evidence label | What it takes: implementation category | Number/event I would watch |
|---|---|---|---|
| 1. Preserve one-contract SIM; correct stop-entry price slots and mirrored guard semantics before scoring continuation | [A] Q6 malformed entry; [T] one repeatedly sent identity, no valid fill; [I] sequencing | **Code**, then owner-coordinated SIM verification; no deployment here | Submitted vs accepted trigger/side/quantity; zero mismatches; protective acknowledgement; both sides and cancellation races |
| 2. Specify one entry-to-exit contract and shadow mechanical thesis failure | [T] position591's invalidation, first-target 0.7417R vs worked 2.0071R; design [I] | **Owner ruling + data first**, then scoped **code/prompt** specification | Paired net P&L per opportunity, winner truncation, loss-tail change and Q5 confidence/coverage gate |
| 3. Keep cancellation for crossed fades; assess an N=4-tick continuation cap only when worst-fill geometry passes | [T] arm33 actual 0.70, arm35 permits zero adverse ticks at 2R; cap [I] | **Data first + owner ruling**, later **code**; no knob change now | Side-normalized entry cost, nonfills, paired net expectancy and worst-fill R:R; N=0/2/4/8 logged separately |
| 4. Diagnose empty excursion rows and persist immutable broker execution geometry, timestamps and slippage | [A] empty table despite existing hooks; [T] 116 audited legs with imperfect joins | **Data first/code**, instrumentation gate before exit optimization | Missing/unmatched execution ids; 30 consecutive complete closes; zero false-zero extrema; quote-to-fill cost tails |
| 5. Explain the fifteen unarmed scenario identities before relaxing gates | [T] E5 22→7; policy [I] | **Data first**, version-aware eligibility/event attribution | Unique opportunities by terminal reason; net executable opportunity cost and session-days, not refusal count |
| 6. Separate lifecycle accounting from model latency and exercise restart/partial/bracket failure recovery | [T] arm35 81-second materialization / 102-second ledger lag; [A] partial bracket timing | **Code + future SIM harness**, owner runbook | Fill→cache→ledger→protective-ack timestamps; unprotected filled quantity/time; duplicates/orphans/mismatches |
| 7. Resolve and demonstrate daily loss behavior with the owner | [A] selected flags false; dev latch blocks entries; flatten policy [I] | **Owner ruling**, then scoped **knob/code** work | Actual threshold event, all entry paths blocked, existing position behavior matching ruling, residual exposure |
| 8. Retain the scheduled flat and measure its execution cost | [T] volume/range buckets do not establish depth/slippage; [I] control preservation | **Data first**; no session knob change | Reason-linked flat quotes/fills, latency, unfilled residual and cost by session/day |

I expressly supersede unsupported recommendations to promote live trading from the mixed-era results, infer initial risk from exit distance or mutable arm rows, lower the floor because observed winners had small MAE, re-enable BE/trail from extrema alone, chase with market orders under a fictional cap, or relocate the flat based only on volume buckets. I make no level retirement/flip recommendation; raw touches and repeated RTH-L keys cannot validate one and belong to another section.

## Requirement coverage — every original Section 5 question and subpart

“Answered” includes an explicit trading judgment and its evidence gate; “UNMEASURABLE” means the requested empirical claim cannot be established, with the missing inputs identified here. It does not mean the missing measurement was completed.

| Original requirement | Status / result | Evidence or exact missing inputs |
|---|---|---|
| 1 authored plan → composed 0B stop | Answered: plan version2/S1 → arm35, ATR46.61, 29354.91 → broker29355 | Q1; E6/E8; `complete/q35_complete.json` plan35 |
| 1 gates → placed order type | Answered: touch met, no-chase warning, short limit sent | Q1; `data/nofx_2026-09-02.log:84979`; E7/E8 |
| 1 order_update → fill → materialized | Answered: signal f2b1eb20, NT8 fill09:03:53.694, position59109:05:14, arm update09:05:35 | Q1; E7/E8 |
| 1 excursions → exit, quote each hop | Exit answered, fill433, stop29355; true excursions **UNMEASURABLE** | Q1; E2/E6; empty trade_excursions, missing event-timed path |
| 2 every fill vs containing range | Answered for all 58 position entry/exit legs; actual every execution **UNMEASURABLE** | Q2; E3 116 rows; missing immutable execution FK; missing bars521–523 and proxy timestamps |
| 2 fills at bar extremes | Answered: adverse entry537, exit585; 1/55 each; favorable0/55 | Q2/E3 with Wilson; bars are retrospective |
| 2 AddOn slippageTicks; Go use | Answered: emitted, production use/persistence not found | Q2; E7; C#:1383/1430, Go framing:86 |
| 2 NT8 log slippage | 591 verified zero broker-stop difference; distribution **UNMEASURABLE** | E6/E7; no logged statistic; missing reference/fill/side pairs and executable quotes |
| 2 SIM queue and live fill probability | Documentation scoped; empirical live transfer **UNMEASURABLE** | Q2 primary NT8 source; actual queue/depth/route/fills absent |
| 2 partial fills | Code path answered; multi-contract behavior **UNMEASURABLE** | Q2/Q6; filled partial exposure timing/recovery fixtures missing |
| 2 stop-market gaps | Exchange semantics answered; actual live cost **UNMEASURABLE** | Q2 CME primary source; broker mapping, depth and gap-fill records missing |
| 2 14:45 flat/thin tape | Baseline and volume context answered; flat slippage **UNMEASURABLE** | Q2/E9; initiating-reason-linked executions, quotes/depth missing |
| 3 verify5/15, three fills, 1.7points | Answered: correct old window; 33 actual0.70, 1.70 is bar high | Q3; E8; ids25/27/33/34/36 and24/28/35 |
| 3 cancel vs bounded market vs stop-limit; N/reason | Answered: cancel fades, no market price cap, shadow stop-limit4ticks subject to tighter RR allowance | Q3/E6; [I], no valid comparative return yet |
| 4 authored→armed→placed→reached→filled→won since09-02 | Answered consistent identity table22→7→4→2proxy→1→0 | Q4/E5; broker-valid reach **UNMEASURABLE**, exact order lifetimes/ticks and valid triggers missing |
| 4 largest leak | Answered: fifteen unarmed identities; causal loss allocation **UNMEASURABLE** | Q4/E5 unarmed_keys; active-version/eligibility/terminal-event attribution missing |
| 5 anchor+1.5ATR, BE/trail suspension, EOD | Answered and checked against existing fixes | Q5/E7/E9; min_sl:34/40, exit_mechs_suspend:35, auto_trader_clock:454 |
| 5 trade_excursions MAE winners/losers p50/p80/p95 | **UNMEASURABLE from requested table**; 18/38 position-proxy quantiles supplied | Q5/E2; trade_excursions0, event timing/initial geometry missing |
| 5 stop-hit and target-hit shares | Reconstructed answered40/58 and11/58 with Wilson; authoritative OCO classification limited | Q5/E2; E8 fresh log joins; execution-id reason lineage missing |
| 5 floor inside/outside winner need | Outside10 observed fresh proxies; causal “needed” distance **UNMEASURABLE** | Q5/E2; immutable initial ATR/stops/path, non-selected comparison missing |
| 5 exit design and switching gate | Answered: mechanical thesis exit shadow vs unchanged bracket/flat; complete30-close instrumentation then held-out day comparison | Q5 and recommendation2; [I], not implemented |
| 6 order type breakpoint | Answered: malformed entry price contract and wrong-side predicate | Q6/E7 C#:976, armed_executor:940 |
| 6 size breakpoint | Answered: one-contract record cannot test multi-contract partial protection | Q2/Q6; controlled size/recovery event data absent |
| 6 AddOn breakpoint | Answered: acknowledgement/protective bracket/reject sequencing | Q6/E7; event tests missing; no guessed probability |
| 6 reconnect breakpoint | **UNMEASURABLE recovery guarantee**, concrete tests/ruling specified | Q6; missing restart/pending-bracket recovery and broker-route custody proof |
| 6 snapshot cadence breakpoint | Answered30-second periodic plus state changes; between-snapshot exposure recognized | Q6/E7; snapshots1604–1606; no cadence-derived safety guarantee |
| 6 daily-limit blocks entries, does not flatten | Answered: false switches, dev block latch, owner must rule on existing risk | Q6/E7 selected configuration and risk_limits:151/305 |

## Reproduction, verification and handoff

The only writable deliverables are this report and its own data directory. Fresh scratch is `/home/hoang/nofx-analysis/vet-05-complete-0905`; retained worktree is `/home/hoang/nofx-vet-05-complete`. The branch is for the parent to merge into dev; I do not merge or remove the worktree.

`complete/README.md` records the exact ordered script invocation, source base and data semantics. q36 extracts raw log evidence first; q11 and q14 rebuild fill-time/reason inputs; q31–q35 rebuild all eligible results; q37 captures supplemental source. Every SQLite reader uses both read-only mechanisms. Current outputs include all row/bar/signal keys; historical q01–q34 outside complete are explicitly superseded. Primary web documentation and its limits are recorded in `complete/PRIMARY-SOURCES.md`.

Validation checks the exact 58-id whitelist, 18/38/2 counts, −466.428572 total, 12 CME days, zero post-cut positions, 116 fill audit rows, 55 bars per leg, 40 stop/11 target labels, ten fresh winner floors, aligned opportunity counts and zero broker-referenced stop slippage. Script parsing and docs-only path/whitespace checks are recorded in `complete/validation.out`. No application test, production API authentication or broker experiment was performed. Read-only health was HTTP200; it does not prove execution readiness.

Remaining measurement limits are explicit above: no realized strict sample, no trade_excursions rows, incomplete broker execution/initial risk lineage, bar-boundary timing uncertainty, no live queue/depth/slippage distribution, no valid capped-entry or thesis-exit counterfactual, and no demonstrated restart/multi-contract recovery. These limits constrain my recommendations; they are not filled with desired profitability.
