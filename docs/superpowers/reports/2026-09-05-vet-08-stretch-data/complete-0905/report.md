# Section 8 — The three-day stretch: complete trader assessment

Owner: hoang. Section 8 only; integration belongs to Section 9. Branch `docs/vet-08-0905-complete`; pinned source `b4376246c2c502ecedd119c6a44a27956ed2f616`. Final transaction extraction: 2026-09-05T22:54:16.973730+00:00 (UTC); analytical cutoff remains the historical CT windows below. All trading access was read-only: SQLite `mode=ro`, `query_only=on`, transaction-scoped extraction; no JWT generator, orders, runtime, configuration or production code changes.

## One-page summary

**My verdict:** I cannot establish an executable edge from this stretch. I can establish a losing recorded book, a planner that failed to authorize a usable long at the relevant time, and an execution path whose stop-entry semantics differ from its intent. The three-day book contains **five eligible trades, −$521.50 / −260.75 MNQ points**, all losses; zero wins out of five, Wilson 95% interval **0–43.45%**. There are **188 identified scenario opportunities**, counting repeated proposals, retries and split legs once per plan/version/scenario. This is a census of recorded ideas, not a census of every trade a human could have found. [T; D/trades.csv:2; D/opportunity_ledger.csv:2; D/analysis_summary.json:1]

**My three biggest problems:**

1. **The thesis and the executable trade are different objects.** At 10:00 September 3, NY v4 was long/trend, but all three scenarios had no enabled arm, and the highest authored target was 29375.25, only 12 points above the last closed minute. A 75.95-point minimum stop required roughly 151.90 points of reward. Calling the day bullish did not supply that reward or authorize an entry. [T; D/plans.json, NY v4; D/q2_asof.json:1; D/q2_atr.json:1]
2. **Prior replay profits were manufactured by assumptions.** The earlier −144.71-point “current rules” result repaired the guard, substituted mutable ledger stops, assumed fills, and skipped alternate minute bars. Its +151.90-point 10:00 trade was hindsight-selected. I withdraw both as trading results. The new source-based checkpoint replay finds four independently price-reachable opportunities, only three on one alternate-minute phase; none is a proven counterfactual fill. [T; D/replay.go:1; D/reach_bounds.csv:2]
3. **An empty ledger did not establish operational control.** On September 3 the process was silent for 114m51s, then returned without the broker link. On September 4, a live TCP connection coexisted with missing bars. Earlier that morning, 21 submissions of one stop-entry idea carried `Stop price=0`. These are distinct failures requiring distinct checks. [T; production log `nofx_2026-09-03.log:5422`; D/log_evidence.csv:2; D/broker_evidence.csv:6]

**My three biggest opportunities:**

1. **Make the morning decision a risk-and-reward decision.** I would retain the long thesis, refuse an unplanned market purchase at 10:00, and record a separately specified breakout alternative in shadow. Judge it on information available before entry, including where the thesis fails and whether meaningful reward remains. [I: my analytical judgment, untested here.]
2. **Measure the full idea lifecycle in one currency.** Carry-in arm 30, duplicate stop submissions and repeated AI refusals belong in the same opportunity ledger. This reveals which ideas were never authorized, never reached, invalidated, or operationally stranded without pretending these are independent losses. [T; D/opportunity_ledger.csv:2]
3. **Use broker acknowledgments and a staffed recovery procedure.** Verify trigger, bracket and current exposure in the broker’s book. Use the existing Studio alert surface and a named person watching it; the owner declined a new channel. A missing feed is a reason to reconcile, not a reason to seek a recovery trade. [I; `trader/auto_trader_alerts.go:15`; `trader/auto_trader_feedwatch.go:68`]

## Evidence conventions and corrections

**D/** means `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/`. CSV evidence includes database row IDs; bars include both SQLite rowid and the durable `(MNQ,1m,open_time_ms,convention)` key. Source line references refer to the pinned revision. Log references retain absolute original paths and line numbers in D/log_evidence.csv and D/broker_evidence.csv. Short opportunity names below mean `trade_date / session / version / scenario`; D/opportunity_ledger.csv carries the complete plan IDs. Times are CT; session windows are half-open. I use [T] for measured tape and [I] for analytical judgment untested here. I claim no personal trading biography. No named market study supplies evidence for the proposed trade.

I supersede the previous Section 8 report in full. Its historical data remain for provenance and are explicitly deprecated; none of its replay, payoff, risk-multiple or level-retirement conclusions should be reused without remeasurement.

- Correct era filter: `entry_time >= 1786770000000`, nonempty resolved plan ID, `plan_id != 'UNRESOLVABLE'`, source other than `e7_farside_test`, and non-NULL `pnl_corrected`. It gives **58**, sum **−466.428572**, **18W/38L/2 flats**, across **12** CME 17:00-CT entry session-days. Exclusions: **530,539,545,546,566,571,580**, test **572–574**, NULL **576,577,579**. The prior 65-row primary population, 21W/42L payoff and dependent 2R claims are withdrawn. Win share including flats is 18/58, **31.03% [20.62,43.80]**; excluding flats it is 18/56, **32.14% [21.40,45.18]**. These intervals describe rows; clustering by day reduces inferential information. [D/era.csv:2; D/extract_summary.json:1; D/analysis_summary.json:1]
- `trade_excursions` has **zero rows**. Position MAE/MFE fields are proxies, not immutable excursions at a verified initial risk. I report no MFE≥2R statistic and do not infer initial R from an eventual exit distance or a changed arm stop. [D/extract_summary.json:1]
- Position **591**: accepted protective stop **29355**, fill **29355**, broker stop slippage **0 points**. The difference from ledger stop **29351.63** is **3.37 points of ledger drift**, not fill slippage. Broker rows timestamp entry protection at 09:03:53; position materialization is 09:05:14.627. I retain both clocks. [D/broker_evidence.csv:2–5, original NT8 September 3 lines 6079,6085,6259,6260; D/trades.csv:6; D/arms.csv, id35]
- I withdraw the previous RTH-L flip/retirement inference. Its 140 raw rows represent only 14 distinct price-time keys; the 677-touch population has 423 keys and formation-time leakage. Section 2 owns its remeasurement. Those repeated observations are not independent trades and contribute nothing to this section’s opportunity or edge estimate.
- There are **no eligible realized trades after the September 3 11:10 CT strict-enforcement boot** in this extraction. Historical losing trades do not measure post-strict expectancy. [D/trades.csv:2; D/era.csv:2]

## 1. Re-derive September 2–4: every trade, refusal, arm, plan version and session

### Trades and trader interpretation

| Position | Idea / decision record | Entry → exit CT | Side; entry → exit | Corrected dollars / points |
|---|---|---|---|---:|
| 587 | Sep1 ASIA v7 S3; 36122 | Sep2 00:17:44 → 01:03:41 | long 29079.25 → 29048 | −62.50 / −31.25 |
| 588 | Sep2 LONDON v3 S2; 36336 | Sep2 07:41:05 → 07:51:38 | long 29082.50 → 29050 | −65 / −32.50 |
| 589 | Sep2 NY v3 S3; 36394 | Sep2 09:41:04 → 09:59:27 | long 29192.50 → 29115 | −155 / −77.50 |
| 590 | Sep2 NY v5 S4; 36422 | Sep2 10:37:17 → 10:49:30 | long 29193.25 → 29143.75 | −99 / −49.50 |
| 591 | Sep3 NY v2 S1; arm35 | Sep3 09:05:14 → 09:20:45 | short 29285 → 29355 | −140 / −70 |

[D/trades.csv:2–6; D/intents.csv:2; quantities all one, MNQ point conversion also reconciles entry/exit arithmetic. The `sync` close reason alone does not identify an exchange stop; 591 has direct broker confirmation.]

I see repeated buying near 29193 on September 2, with entries **0.75 points apart** but different cited versions, both losing. Re-authoring did not create independent market evidence. This is a hypothesis about duplicate exposure, not proof that a “two attempts” cap would improve expectancy. September 3 then lost on an early short while the later v4 long scenarios offered no actionable arm. The response should be to compare setup freshness and remaining reward, not to infer that shorts are bad or that a stopped fade is automatically a buy signal. [T n=2 entries, ids589/590; I judgment.]

### Plans, arms, refusals: one denominator

There are **62 plan versions whose session/authorization windows overlap the wall-clock audit**: four September 1 ASIA carry-in versions, 33 September 2 versions, 16 September 3 versions and nine September 4 versions. They contain **187 scenario-version opportunities**. A superseded September 1 ASIA v6 S1 arm, **30**, remained part of the observed opening inventory and adds one **carry-in-arm-only** opportunity: **188 total**. D/retained_plan_versions.json additionally preserves older versions used to resolve lineage; they are not all fresh opportunities.

The 188-row ledger joins all **39 non-test arm rows**, **24 open proposals**, and all five positions. The 24 proposals are just **nine** scenario opportunities. The **20 recorded open refusals** are just **five** opportunities. There were **1,652 decision records**; calls and opportunities are different units. All refusal text and timestamps are preserved in D/refusals.csv, with broader arm/prefilter/reconciliation refusals in D/log_evidence.csv. Unlogged or deduplicated-away refusals are **UNMEASURABLE**, not counted as zero. [D/plans.csv:2; D/opportunity_ledger.csv:2; D/decisions.csv:2; D/intents.csv:2; D/refusals.csv:2]

| Stage, always unique scenario opportunities | Count / 188 | Descriptive share; Wilson 95% |
|---|---:|---:|
| Stored enabled-arm ideas, including carry-in | 61 / 188 | 32.45%; 26.16–39.43% |
| Has observed non-test arm lineage | 13 / 188 | 6.91%; 4.09–11.47% |
| Has an arm with broker signal ID | 7 / 188 | 3.72%; 1.82–7.49% |
| Has observed arm fill | 1 / 188 | 0.53%; 0.09–2.95% |
| Has decision-path open proposal | 9 / 188 | 4.79%; 2.54–8.85% |
| Has recorded decision-path open refusal | 5 / 188 | 2.66%; 1.14–6.07% |
| Has eligible recorded trade, either path | 5 / 188 | 2.66%; 1.14–6.07% |

These are overlapping attributes, not an additive waterfall. Revisions can repeat the same economic level; Wilson intervals are provided as requested, not evidence of independent trials. A signal ID establishes attempted broker lineage; it does not prove a fill. D/opportunity_ledger.csv lists every contributing ID.

All arm IDs: **29–39,62,65,67,70,73,75,77,79,81,83,85,87,89,91–105**. Arm15 is a stale test seam and excluded. The duplicate stop idea is **Sep4 NY v3 S2**, rows **38,62,65,67,70,73,75,77,79,81,83,85,87,89,91,93,95,97,99,101,102**: 21 submissions, **one opportunity**. Other repeated/split groups are v3 S1 **39/92/94/96/98**, v5 S1 **100/103**, v6 S2 **104/105**. Their full terminal reasons and timestamps are in D/arms.csv:2.

Decision refusals are exhaustive for stored open proposals: **36164** cutoff; **36640/36641/36642** wrong-ATR min-stop, **36645** R:R, all one Sep2 ASIA v6 S2 idea; **36703** R:R on v9 S2; **36864** R:R on Sep3 LONDON v1 S1; **37304,37305,37306,37307,37308,37310,37311,37313,37315,37318,37319,37320,37322** strict on one Sep3 ASIA v5 S1 idea. Repeated min-stop arm warnings and deduplicated arm-gate messages are preserved with log path:line; I do not add log-line counts to opportunity counts. The two explicit canonical invalidation refusals are Sep4 LONDON S2 at 02:00:46 and NY S1 at 09:02:09. They do not prove losses saved. [D/refusals.csv:2; D/log_evidence.csv:2]

### Tape per session

The table below uses disjoint CT fragments so overnight bars are not double counted. Full bars, row IDs and summed volume appear in D/sessions.csv:2 and D/bars.csv:2. Open/close are first/last retained bars, not broker execution prices.

| CT date / session fragment | Minute bars | Open | High | Low | Last close | Net points |
|---|---:|---:|---:|---:|---:|---:|
| Sep2 ASIA 00:00–02:00 | 120 | 29059 | 29105 | 29016.50 | 29076.75 | +17.75 |
| Sep2 LONDON 02:00–08:30 | 390 | 29076.75 | 29149.25 | 28927.25 | 29098.25 | +21.50 |
| Sep2 NY 08:30–14:45 | 375 | 29098.25 | 29211.75 | 29017.25 | 29175 | +76.75 |
| Sep2 ASIA 17:00–24:00 | 420 | 29175.75 | 29255 | 29132.50 | 29207.50 | +31.75 |
| Sep3 ASIA 00:00–02:00 | 120 | 29207.75 | 29208.75 | 29075 | 29173.50 | −34.25 |
| Sep3 LONDON | 390 | 29173.25 | 29293 | 29101.75 | 29248.75 | +75.50 |
| Sep3 NY | 375 | 29249.50 | 29585 | 29199.25 | 29524 | +274.50 |
| Sep3 ASIA 17:00–24:00 | 420 | 29500 | 29601 | 29481.50 | 29597.25 | +97.25 |
| Sep4 ASIA 00:00–02:00 | 120 | 29597.75 | 29644.25 | 29573.75 | 29576 | −21.75 |
| Sep4 LONDON | 390 | 29577.25 | 29720 | 29502 | 29588.25 | +11 |
| Sep4 NY, retained through 12:19 | 230 | 29588.50 | 29692 | 29477.75 | 29531.50 | −57 |
| Sep4 ASIA evening | 0 | — | — | — | — | UNMEASURABLE |

September 2 offered rotation and reversal despite a positive NY close; buying high twice was costly. September 3 expanded upward after the early short, but that path was not visible at 10:00. September 4’s large London and morning NY ranges describe two-way movement; a rally to a high is not by itself proof of acceptance there. Missing September 4 afternoon bars prevent a full-session direction, range, volume or exit conclusion. Retained September 3 afternoon bars may be historical backfill; their presence does not prove they reached the bot during the outage. [T; table; I interpretation.]

**My ruling on the two-day audit:** I agree with the five recorded losses and the separation of decision entries from arm35. I agree that operating gaps and unexecutable long plans matter. I reject its percentage attribution split because it mixes units. I withdraw “gates saved 84 points”: the 13 strict refusals concern one idea and lack a demonstrated alternative fill and exit. I also reject “cadence caused everything” and “cadence suppressed nothing”: logs distinguish warn-only cooldown before the September 3 10:28 boot from enforcing suppression afterward. The latter cannot explain a decision at 10:00. I do not infer that a stop near a later turn was premature; that requires an ex-ante invalidation rule and matched trade paths. [D/log_evidence.csv:2; D/refusals.csv:2]

## 2. What I would do at 10:00 CT on September 3

**I would stay flat under the existing authorization, retain a bullish watch, and write down a conditional continuation alternative before seeing the next minute.** This is my analytical judgment [I], untested here. I would not turn the previous short loss into a compulsory reversal or move a target merely to pass R:R.

My information cutoff is **10:00:00**, using only fully elapsed one-minute bars through **09:59**, assuming their timely delivery. This is the strongest reconstructible market-information set; the exact received feed at that instant is not retained. The 09:59:35 decision **37098** actually reasoned from price **29340**, not the subsequently completed minute’s **29363.25**. It was therefore wrong for the old report to assume the bot saw the full minute at that decision. [D/q2_decisions_before_1000.csv, id37098; D/q2_asof.json:1]

What was knowable:

- NY open **29249.50**; last completed close **29363.25**: **+113.75** points. Initial balance 08:30–09:29 high **29375.25**, low **29199.25**. The high had **not** broken in the information set. A one-tick-above trigger would be **29375.50**. [D/q2_asof.json:1]
- The **09:56** bar, rowid **440388**, ranged **29246–29325.50**, closed **29296.75**, volume **16,450**. That volume was **2.37×** the median of the **86** preceding RTH minutes and ranked third among those 87 bars. The **09:59** bar, rowid **440395**, closed **29363.25** after a high **29367.75**. This supports “sharp selloff recovered,” not “liquidation” or “institutional absorption”: aggressor side, order-book depletion and participant identity are missing. The typical-price weighted RTH VWAP **proxy** is **29302.38**, not an exchange tick-VWAP observation. [D/q2_asof.json:1]
- NY **v4**, published **09:44:33**, was the available plan. S1/S2 were `breakout_retest` and S3 `reclaim`; **none had an arm**. S1’s targets were **29303.68/29346.88/29375.25**, S2’s **29346.88/29375.25**, S3’s **29260/29303.68/29346.88**. At the cutoff, all lay below or close to price. Versions **5–7** were still in the future and cannot justify this trade. [D/plans.json, Sep3 NY v4; D/plans.csv:2]
- Source-derived ATR5m on the last 2,000 closed minutes was **50.632739**, floor **75.949108**. Buying **29375.50** with that floor would risk about **$151.90** per MNQ before friction; rounded outward to a valid quarter-point stop **29299.50**, risk becomes **76 points/$152**, and a purely mechanical 2R target becomes **29527.50**. That target was **not** in v4. A structural stop one tick below the 09:56 low, **29245.75**, risks **129.75 points/$259.50**, with mechanical 2R **29635**. Neither risk choice is “correct” because of the later high. [D/q2_atr.json:1; D/q2_asof.json:1; arithmetic]

The conditional alternative I would record is: one MNQ only if an independently preauthorized breakout method permits the IB high trigger, the broker confirms the correct stop-entry and protective bracket, and the risk budget accepts the stop. If it already trades through the trigger before acknowledgment, record a missed entry; do not assume a fill there. Use the structural stop if the thesis is failure of the recovered 09:56 low; use the ATR stop only if that is the separately tested method. If available structure cannot justify sufficient reward, remain flat. This is a prospective method proposal, not a current-rule replay and not an instruction applied to the bot. [I]

**News context:** the primary [ISM release calendar](https://www.ismworld.org/supply-management-news-and-reports/reports/rob-report-calendar/) lists Services PMI on September 3 at its stated 10:00 Eastern release slot, ordinarily 09:00 CT; its generic page labels the zone “EST,” so I do not infer an exact historical receipt timestamp from that label. The [BLS calendar](https://www.bls.gov/schedule/2026/) scheduled the August Employment Situation for September 4 at 08:30 Eastern/07:30 CT. These schedules could inform a morning briefing. The retained 09:57/09:59 prompt search contains no corresponding news lines, and I have no timestamped consensus, first-release content or receipt latency establishing what the bot knew. I do not attribute the rally to a news surprise. These calendars are context, not market-edge studies. [D/q2_news_at_time.json:1]

**Correct the hindsight premises:** **483.25** is the later London-plus-NY high-low difference **29585−29101.75**, not an available-at-10:00 gain. The “+199 while empty” depends on endpoints: the 09:20 and 11:58 minute closes are **29348** and **29532**, a **184-point** difference (bars rowids **440271/440791**); exact subminute endpoint prices are missing. Position591 closed at 09:20:45.677; arm37 was created at 11:58:33 but did not fill. Thus 11:58 ends an **unarmed interval**, not the flat-position interval. A large later range does not turn an unfilled idea into lost profit. [D/bars.csv:2; D/trades.csv:6; D/arms.csv, id37]

The system would need a preauthorized continuation definition, an enabled correctly typed arm, an immutable initial bracket, a valid structural or explicitly mechanical target rule, and a timestamped receipt/acknowledgment trail. `breakup_continue` remains a pullback limit by design (`kernel/arm_kind.go:41`); reclaim stop-entry support (`:60`) does not auto-enable old plans (`trader/armed_executor.go:290`). The existing strict regime should not be loosened to recover this hindsight-selected day. [I]

## 3. Replay under CURRENT rules: measurable components, bounded result, unknown broker path

**Exact full-book armed/filled/refused counts and point P&L are UNMEASURABLE from the retained inputs.** I deliver an implemented offline **necessary-condition replay**, not a substitute profit figure. The new replay actually executes pinned kernel functions and preserves every checkpoint/result. It is materially stronger than the old Python approximation while explicitly stopping before unsupported broker-state reconstruction.

### Rules actually executed

D/replay.go invokes `ArmSpecValid` (`kernel/plan_doc.go:130`), `ArmKindFor`/`ArmKindMismatch` (`kernel/arm_kind.go:36,71`), condition shadowing (`kernel/condition_status.go:57`), `EvaluateConfirm` (`kernel/plan_confirm.go:49`), `BarsSince`, `EvaluateScenario` (`kernel/scenario_state.go:169`) and the actual ATR calculation (`trader/entry_gate.go:332`). The pure stop composer and limit wrong-side predicate are copied verbatim from `trader/arm_stop_anchor.go:71` and `trader/armed_executor.go:985`, respectively; provenance is verified by D/verify.py. No `store.New`, production application, broker or JWT tool runs.

At each minute endpoint, the harness supplies at most the latest **2,000 closed 1m bars**, using plan birth (not retrospectively selected touch times), authored stop/target and stored enabled arms. It uses the actual source’s partial-five-minute bucket behavior: `AcceptanceRunEver` itself does not filter its last aggregate by `nowMs` (`kernel/scenario_facts.go:440`). It does not “fix” that semantics silently. Recorded DORMANT events stop eligibility for that version; a REARMED lifecycle transition is not assumed to authorize fresh orders. The implemented parameters are RR2, ATR multiple1.5, two-tick clearance/offset, anchor bound3ATR, limit band25 points, strict routing, shadow defaults and split capacity2. D/rule_config.json records Studio’s per-order capacity **2**, despite observed orders being one lot. No setting was changed.

This replay tests **local necessary conditions** independently per opportunity. Open-position competition, HTF state, all session overrides, order retries, cancellations, process boot history and actual event callbacks are not silently assumed to pass in a portfolio simulation. Their absence is why “passes known local checks” is not labeled “armed.” Min-quality/HTF and dynamic lifecycle differences remain unresolved inputs. Recorded dormant windows are observational constraints, not a claim that counterfactual lifecycle would be identical.

### Exact and bounded answers, in opportunities

| Requested result | Defensible measured result |
|---|---|
| Strict decision routing | All **24 retained proposals**, representing **nine opportunities**, are refused by strict; decision-path new fills **0** conditional on replaying these proposals. D/strict_replay.csv lists all IDs. Future LLM outputs under changed positions/plans are unknown. |
| Stored arm authorization | **60** overlapping opportunities have enabled arms, representing **66 legs**. **127** opportunities have no enabled arm. Carry-in v6 S1 is separate opening inventory. Old disabled reclaims stay disabled; no 24-arm invention. |
| Local arming eligibility | **13** of the 60 have at least one endpoint passing implemented local checks; **47** have none in the sampled prefixes. This is not 47 proven live refusals. Exact local statuses and checkpoints are in D/replay_opportunities.csv:2 and D/replay_checkpoints.csv:2. |
| Placement predicate, implemented bugs | **Seven** opportunities have a locally admissible submission endpoint: six limits and one stop-entry whose payload has zero stop price. |
| Placement predicate, corrected stop semantics | **Six** have a submission endpoint: the same limits; the stop-entry is rejected as already through. This sensitivity fixes only guard/slot semantics, not the strategy or broker. |
| Subsequent entry-price reach | **Four independent opportunities** reach their reference price after a qualifying endpoint within the retained version window, under either variant. One alternate-minute phase reduces this to **three**. Each has a conditional fill-indicator envelope **0–1**; the sum **0–4** (or **0–3**) is an independent-opportunity envelope, **not a full-book trade-count bound**. |
| Exact broker fills / total P&L points | **UNMEASURABLE.** No fills are assigned and no counterfactual P&L is summed. The only full observed result remains **−260.75 points** on positions587–591. |

These are deterministic census counts, not estimated fill/win rates. D/bounds_summary.json:1 and D/reach_bounds.csv:2 preserve the sensitivity. Absence of a qualifying endpoint cannot exclude a transient intrabar opportunity.

The **13 local-pass IDs** are: Sep1 ASIA **v7 S1**; Sep2 NY **v5 S3, v12 S3**; Sep2 ASIA **v13 S1**; Sep3 NY **v2 S1, v3 S2, v7 S3**; Sep4 NY **v2 S2, v3 S1, v3 S2, v4 S2, v5 S1, v6 S2**. The **seven submission IDs** are that list minus Sep2 v5S3/ASIA v13S1 and Sep4 v2S2/v4S2/v5S1/v6S2. The **four price-reach IDs** are Sep1 ASIA **v7 S1** (observed arm29), Sep2 NY **v12 S3** (arm33), Sep3 NY **v2 S1** (arm35/position591), and **v3 S2** (arm36). The odd-minute phase loses the carry-in v7S1 reach. Each exact bar rowid is in D/reach_bounds.csv.

### Implemented buggy stops versus corrected mechanics

The implemented stop branch applies `limitMarketableWrongSide` at `trader/armed_executor.go:940`. For a short, it cancels when price is above the trigger—the valid sell-stop side—and admits a trigger already above the market. The C# creation call then uses `qty, orderPx, 0` at `ninjascript/VLTraderTCPClient.cs:976`, placing the trigger in the limit-price parameter. The official [NinjaTrader CreateOrder signature](https://ninjatrader-staging.ninjatrader.com/support/helpguides/nt8/createorder.htm) identifies distinct limitPrice and stopPrice parameters. This documents API semantics, not market behavior.

Sep4 NY v3 S2 is a short reclaim. Its **21 distinct signal submissions** all show `Stop price=0` in original NT8 log lines **6496 onward**, deduplicated by signal name, not state-update count. Zero valid nonzero stop triggers among 21 submissions: Wilson **0–15.46%**. With retained prices positive, that observed sell-stop shape never reaches its zero trigger. In the corrected branch the short trigger is already through at all 40 locally eligible endpoints, so it is not submitted. Neither fact tells us how a hypothetical broker would execute a valid stop on a different path. [D/broker_evidence.csv:6; D/replay_checkpoints.csv, Sep4 NY v3 S2]

The reaper now asks a **fresh received broker snapshot** (`trader/armed_executor.go:1072`) rather than declaring silence dead; checklist class79 (`AUDIT-CHECKLIST.md:1820`) is already fixed. D/snapshots.csv preserves **emitted and received** timestamps and order membership. A stored observed snapshot can establish what happened to observed signals; it cannot contain signals/orders created only by a counterfactual. Therefore I do not “replay the fixed reaper” by deleting all historical cancellations or assuming perfect broker continuity. A fixed reaper could prevent duplicate submissions and preserve an order longer, changing later exposure, gates and plans.

I also replayed the current three-valued reaper predicate at the **11 observed stale-cancel timestamps**: **8 ALIVE** (arms39,73,75,77,79,81,83,85; preceding snapshot IDs1398,1603,1613,1623,1633,1643,1653,1663), **3 UNKNOWN** (arms29/30/31, no retained preceding snapshot), **0 GONE**. Both alive and unknown mean no reaper cancellation. This uses a 30-second snapshot interval and assumes the retained cache survived to that observation; a boot can instead make the cache unknown. It measures a local change to the observed path, not how many replacement orders or later fills would exist. [D/reaper.py:1; D/reaper_observed.csv:2; `trader/reaper_snapshot.go:51`; `trader/f12_leg4.go:197`]

The daily-limit leg exists and precedes strict (`trader/entry_gate.go:157`); the read-only strategy snapshot has **guardrails_enabled=false**, **daily_loss_enabled=false**, limit **$450**. Its current loss-trigger contribution is inactive. A hypothetical enabled $450 limit is a **different scenario**, requiring the defined daily equity/fee window and path-dependent realized/unrealized P&L; it is not inferred from the three-day aggregate loss. Bias-arm checking is warning-only (`kernel/arms_bias_coherent.go:74`) and creates no authorization.

### Exact missing inputs required to close Q3

1. **Decision-time market snapshots:** ordered ticks/bid/ask, every partial 1m/5m update actually delivered, receipt times, complete historical depth and revisions, plus September 4 post-12:19 bars. Final OHLC and bar timestamps do not establish original availability or within-bar ordering.
2. **Event scheduling:** actual arm/placement/reconcile/LLM callback times, monotonic ordering, delays, queue drops, boot identities and source revision per boot. A regular one- or two-minute grid is not that sequence.
3. **Immutable authorization state:** complete active-plan/lifecycle history, birth semantics across boots, rearm authorization, scenario invalidation/consumption facts, per-session resolved config and environment at every callback, HTF snapshots and quality floors. Current plan rows and sparse lifecycle records cannot recover every intermediate state.
4. **Original orders and risk:** unmodified initial entry/SL/TP specifications, broker-rounded request and accepted bracket, every modification/cancel and acknowledgment, and the association to plan/version/scenario/leg. Mutable `armed_orders.stop_px` cannot substitute.
5. **Broker execution model:** snapshot membership at each counterfactual decision, order acceptance/rejection, queues, bid/ask, tick path, fill priority, slippage, fees, partial fills, gap handling and cancel/fill races. Observed snapshots do not specify hypothetical outcomes or whether SL versus TP came first inside a bar.
6. **Full state propagation:** starting broker inventory including carry-in arms29/30/31, one-open-position competition, sibling order effects, complete daily equity and guardrail state, EOD flatten request/ack/fill, changed planner inputs and resulting new outputs after counterfactual fills. Holding observed plans fixed is an explicit conditional experiment, not an exact rerun of the autonomous system.
7. **Operational availability:** distinguish recovered/backfilled bars from live receipt during September 3 silence and September 4 feed failure; capture whether local stops are broker-held and whether connection/account frames are current. No finite trustworthy counterfactual point-P&L interval follows from the present data.

## 4. The 12:30 outage and blind boot: my runbook

**Observed:** September 3 last recorded decision **37169 at 12:22:44**; last log line **12:23:33** (`nofx_2026-09-03.log:5422`); startup banner **14:18:24** on the next line after a NUL block; silence **114m51s**. The **14:18:27** dead-man message says TCP down (`:5549`); later no-balance skips precede the first recovered decision around **15:08:51**, after the 14:45 flat. Thus process return and usable trading return are different clocks, roughly **50m27s** apart. At 12:30 the last observed position had closed and arm37 had been cancelled at **12:15:01**; that is the last known state, not a fresh broker confirmation. [D/decisions.csv, id37169; D/arms.csv, id37; D/log_evidence.csv:2]

September 4 last retained minute is **12:19**; the **12:30:01** feed alert reports 10m1s without a bar (`nofx_2026-09-04.log:17339`). The **13:25:47** startup (`:19749`) did not demonstrate a recovered feed. Ledger104/105 remained armed without signal IDs while the observed broker orders had been cancelled. TCP health alone cannot certify market-data health. [D/arms.csv; D/snapshots.csv; D/log_evidence.csv]

My procedure is an **owner-run proposal [I], not an action performed in this audit**:

1. **At 12:30, stop approving new exposure and establish the broker’s facts.** Read NT8 Positions and Orders independently of Studio. Note instrument, side, quantity, entry orders, protective stops/targets and acknowledgment times. If the broker cannot be reached, record exposure **unknown**; do not infer flatness from a stale ledger.
2. **Remove entry risk only after identifying orders.** The authorized operator cancels unfilled entries and confirms their terminal acknowledgments. Preserve protective exits while reconciling any position. If protection is absent or cannot be verified, use the owner-approved broker-side flatten/escalation procedure and verify the result. “Cancel everything” can remove the only working protection and is not this runbook.
3. **Assign a person and distinguish the failure.** Check newest bar receipt, broker-link heartbeat, account frame, process heartbeat and last acknowledgment separately. In this incident the owner would read the existing Studio P0 feed and NT8; require an acknowledgment and a named handoff. No new Telegram/push channel is proposed. An in-app alert cannot summon an absent operator; unattended host-failure detection remains a stated limitation under that channel choice.
4. **Recover by evidence, not by a banner.** Require fresh consecutive bars, fresh account/order/position snapshots, reconciliation of every ledger/broker discrepancy, and no unacknowledged cancellation before resuming. A feed-age check can strengthen the existing cutover checklist; do not reimplement the already fixed boot sweep or snapshot reaper.
5. **Do not try to earn back the outage.** With the September 3 recovery after 14:45, entries remain off until the next authorized session/read. If recovery occurs earlier, the owner decides whether enough session remains for a fresh plan and full risk review; “60 minutes” would be an untested operating threshold, not a researched market edge.
6. **Record one incident timeline.** Detection, operator acknowledgment, last-known exposure, broker-verified exposure, entry cancellation acknowledgments, feed recovery, reconciliation and entry reauthorization. Review lost opportunity separately from actual losses. Metrics: time to acknowledgment, time exposed without verified protection, stale-data submissions, and reconciliation discrepancies.

## Recommendations, ordered for this section

| What I recommend | Why | Implementation category | Number I would watch |
|---|---|---|---|
| Require an ex-ante target/risk feasibility statement for each morning idea; no reordering distant targets just to clear the gate | [T] v4 at 10:00 had no enabled arms and at most 12 points to its highest target versus 75.95 floor risk | **Prompt + data first**; preserve structural target meaning | Unique opportunities with sufficient reward before authorization; realized point expectancy after friction, grouped by day |
| Specify one continuation alternative prospectively and score it in shadow, including skipped entries and news timing | [I] recovered selloff and unbroken IB high merit a hypothesis, not a hindsight buy | **Owner ruling + data first**, later code/prompt only if justified | At least 30 independent session-days as an initial review sample, not automatic approval; day-block lower expectancy CI above zero, sensitivity to friction, missed fills and stop choice |
| Correct stop guard/payload semantics and demand broker echo evidence before trusting that order type | [T] one idea generated 21 zero-stop submissions; no additional price-reachable entry in the corrected checkpoint sensitivity | **Code + broker acceptance proof**, separate authorized implementation; not applied | Zero malformed stop echoes; accepted nonzero trigger equals requested rounded trigger; idempotent order membership |
| Retain immutable received market snapshots, initial brackets, amendments and callback sequence | [T] exact replay and risk multiples are unmeasurable today | **Code/data first** | Fraction of unique opportunities with a complete reproducible decision→ack→fill→exit path, with n; unresolved counterfactual paths |
| Review duplicate economic levels across plan versions in one daily opportunity ledger | [T] ids589/590 bought within 0.75 points; 21 stop submissions and 13 strict refusals are each one idea | **Docs/data first**; a new trading cap would require an owner ruling | Repeated-risk dollars at the same level before genuinely new evidence; do not count retries as independent setups |
| Staff the existing alert surface and adopt the broker-first outage runbook | [T] silence and blind recovery were distinct; [I] procedure | **Owner ruling + operating docs**, targeted feed-age code only if authorized | Detection→acknowledgment minutes; minutes with unverified protection; stale-feed entry submissions |

I do **not** recommend retiring/flipping level kinds, promoting continuation after one profitable-looking day, extending the flat, relaxing strict, altering the daily limit, opening a new alert channel, or substituting farthest targets into the gate. The prior recommendations to do those things on this evidence are superseded. Checklist class79 and reclaim-kind wiring are already present; the outstanding recommendations are verification and scoped additions, not requests to rebuild them. [AUDIT-CHECKLIST.md:1806,1820; `trader/entry_gate.go:157`; `kernel/arm_kind.go:36`]

## Requirement coverage: no hidden completion claims

| Original requirement/subpart | Result and evidence | Status / exact remaining inputs |
|---|---|---|
| Q1 every trade, corrected population | Five positions, table above; D/trades.csv; 58-row D/era.csv | Measured; no immutable initial R for all trades |
| Q1 every refusal | Twenty stored open refusals/five ideas, full IDs; broader retained warnings in D/log_evidence.csv | Measured retained records; unlogged/deduplicated attempts UNMEASURABLE without immutable per-evaluation log |
| Q1 every arm | 39 non-test rows, 13 ideas plus full states and lineage in D/arms.csv and D/opportunity_ledger.csv | Measured current retained rows; unrecorded intermediate specifications require order-event history |
| Q1 every plan version | 62 overlapping versions + carry-in v6 arm; D/plans.csv, D/retained_plan_versions.json | Measured retained versions; overwritten runtime lifecycle is not inferred |
| Q1 tape per session and volume | Disjoint session table; D/sessions.csv and keyed bars | Sep4 afternoon/evening UNMEASURABLE: missing bars; original delivery timing missing throughout |
| Q1 audit agreement/disagreement; one currency | Explicit rulings and 188-row opportunity ledger | Measured recorded idea universe; not all possible market setups |
| Q2 10:00 evidence available then | v4 only; bars≤09:59, source ATR, decision37098; D/q2_* | Reconstructed elapsed-bar information; exact received tick snapshot missing |
| Q2 what I would do | Remain flat under authorization; prospectively specified shadow continuation alternative | [I] judgment, not tested edge or fill |
| Q2 what system needs | Authorization, method/targets, correct trigger/bracket and receipt/ack evidence | Design proposal only; no trading changes |
| Q2 news / +483 / +199 / empty-book premises | Primary calendars; corrected endpoints and position/arm distinction | Surprise/causality and subminute endpoint marks UNMEASURABLE without timestamped release/consensus/tick inputs |
| Q3 strict | Nine opportunities/24 proposals refused; D/strict_replay.csv | Exact conditional routing; changed future AI outputs unknown |
| Q3 reclaim and arms-follow-bias | Enabled arms only; kind functions; warning creates no arm | Source-executed local result; not new plan generation |
| Q3 daily limit | Leg exists; retained master and loss switches off; D/rule_config.json | Current switch state measured; enabled counterfactual needs path-dependent equity history |
| Q3 snapshot reaper | Source and observed received/emitted snapshots; no assumed counterfactual membership | Exact replay UNMEASURABLE without hypothetical broker event sequence |
| Q3 armed/filled/refused/point P&L with IDs | 60 authorized ideas; 13 locally eligible; 7/6 submission ideas; 4/3 conditional reaches; all keyed | Full-book counts and P&L UNMEASURABLE; seven missing-input groups listed above; no −144.71 claim |
| Q3 implemented bugs vs corrected vs broker unknowns | Two guard/slot variants, direct NT8 evidence, D/replay.go and D/reach_bounds.csv | Explicitly separated; no inferred profitable fills |
| Q4 12:30 runbook and blind boot | Broker-first procedure with existing alerts; timestamped last-known exposure | Proposed operation; outage cause and unattended notification remain unverified/unavailable |
| Required summary, recommendations, labels, evidence and reproducibility | Summary; what/why/category/metric table; scripts/outputs and manifest under D | Delivered for Section8 only; parent integrates dev |

## Reproduction and limits

D/README.md documents extraction, the offline Go invocation and analysis commands. D/verify.py checks the population, one-currency joins, source-copy provenance, time cutoff and docs-only scope; D/verification.txt records the outcome. D/manifest.sha256 pins scripts and retained inputs/outputs. SQL is embedded in D/extract.py with outputs retained, not merely described. The source-based replay has no order or P&L simulator, deliberately: the retained evidence cannot identify those outcomes. This completes the bounded audit deliverable; it does not claim the missing Q3 broker path was reconstructed.
