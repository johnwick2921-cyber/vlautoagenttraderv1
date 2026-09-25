# The strategy, written down — 8 September 2026

**Owner summary · descriptive audit · no claim about edge**

Your system trades **intraday reactions at named MNQ levels, with a stronger taste for fades than breakouts**. It buys a failed break below support or sells a failed break above resistance, while also writing reclaim, acceptance and breakout-follow setups. Across the retained history there are **770 actionable scenario descriptions**, including **472 explicit reject/sweep-reclaim setups**. The broader, annotated setup-family count is **496 fades, 266 follows and 8 ambiguous**, not a count of executed trades. [A: C1, scenario IDs in the evidence tables; B: style interpretation.]

Every session it builds and revises a map, chooses levels to write scenarios on, and—under the **currently running strict mode**—can enter through authorized scenario arms. A plan sentence alone does not place an order. Current entry checks include scenario/side consistency, permitted conditions, stop geometry, the Studio **2R minimum**, HTF veto and a fresh broker book showing no existing commitment. Breakout-retest and FVG-entry are currently shadowed. Inactive plans and refused gates stand aside; a “trend” label alone does not. [A: C8 source lines and runtime evidence.]

**It is the same style, not the same daily strategy mix.** ASIA contains **209 fades / 106 follows / 5 ambiguous (n=320)**; LONDON **125 / 74 / 1 (n=200)**; NY **162 / 86 / 2 (n=250)**. Balance-labelled scenarios are **337 fades / 163 follows / 6 ambiguous (n=506)**; trend-labelled scenarios **155 / 101 / 2 (n=258)**. Both groups remain fade-heavy. The strongest same-session contrast is LONDON on **25 August versus 1 September**: **6 long / 2 short (n=8)** becomes **4 long / 9 short (n=13)**, with different level kinds. [A: C1, C3, C5–C6; exact plan rows below.]

It differs from a discretionary trader exercising those choices in three concrete ways. **It can write several competing setups rather than select one trade for the day; conviction does not increase current order size; and it does not currently turn a winner into a partial-plus-runner trade.** The running exit policy says **BE=off, trail=off, size=1**. Its retained eligible fills are all one contract (**n=65**, IDs in C7), with a **25.31-minute median hold**, although some last much longer. That looks like a mixture of short intraday trades and intraday swings, not a demonstrated runner programme. [A: C7–C8; B: trade-style description.]

The important difference from the saved prompt: it does trade balance-labelled plans; it is not restricted to one trade a day; and “50 points to breakeven” is not the current exit behavior. Historical arm events, exact authoring-time price and stop-composition winners are incomplete, so missing evidence is labelled unknown. **This report measures style, not profitability, and recommends no changes.** [A: C4, C7, D4–D5.]

---

## Evidence boundary and reproducibility

The frozen SQLite read was **2026-09-08T18:20:45.991334-05:00 (CT)**, in one read transaction with `mode=ro` and `PRAGMA query_only=ON`. [extract.py](2026-09-08-the-strategy-data/extract.py) contains the exact extraction; [queries.json](2026-09-08-the-strategy-data/queries.json) gives every SQL statement, parameters with private identifiers removed, result count and output file. `analyze.py`, `supplement.py` and this renderer reproduce the descriptive tables offline. There were no writes to trading data, configuration, prompts, code, environment or AddOn, and no restart.

Evidence labels: **[A]** a stored value, code line or direct observation; **[B]** an interpretation supported by those observations. No unsupported [C] conclusion is used. Every table below identifies a source table/key whose cells enumerate IDs. `p178:S1:0` means SQLite `plans.rowid=178`, scenario `S1`, zero-based document slot `0`; level `p178:L0` means the same plan and seated-level slot. This prevents different versions of S1 being collapsed into one setup. Natural bar key is `(MNQ, tf, open_time_ms)`. Raw timestamps retain their original offsets; report event times and converted CSV fields use America/Chicago (CT).

**Runtime premise NOT REPRODUCED:** the dispatch named `f8bc7044`; health instead returned **`6f677b55daa1`**, with PID **3726840**, boot **8 September 2026 18:11:54 CT**. `/proc/3726840/exe` resolved to `/home/hoang/nofx/nofx-bin`; Go build metadata gave `6f677b55daa1c7da33b8c35f8bcc67883f36b470`, `vcs.modified=false`. The loaded executable and disk binary both hashed to `418b08a44e81fb85f5f79524f0cd18d57d4e89b7d91c9100dc076bc7e82c375f`. At **18:27:01 CT**, the API positions list was empty; cutover evidence independently showed no DB/API/NT8 position and no working broker order. No live unprotected position was observed. [runtime.json](2026-09-08-the-strategy-data/runtime.json) and [exit-runtime.json](2026-09-08-the-strategy-data/exit-runtime.json) preserve the sanitized observations. [A]

The worktree began at that dev tip and the branch was claimed as `docs/the-strategy-0908`; the verified remote SHA at acceptance was `f0c172e1559aaf2718fa9c139c126bed3b8ebb2a`. Main checkout remained read-only. No lock was touched under this dispatch’s A2 override. Current-code descriptions apply to the measured revision; **the whole historical sample did not run this one version**. In particular, current strict/one-contract/exit rules must not be projected backward onto every August fill.

**Final freshness check:** at **8 September 2026 18:43:07 CT**, health and the loaded/disk SHA256 still matched the same running revision, and positions remained empty with cutover ready. [runtime-final.json](2026-09-08-the-strategy-data/runtime-final.json) preserves the GET-only observation. The report branch was rebased onto dev **954f11b1**. Changes since 6f677b55 affected Stage A archive path handling and its documentation; the cited trading implementations were unchanged. The changed SYSTEM-MAP and AUDIT-CHECKLIST sections were reread and the provenance below refreshed. No historical table was silently refreshed beyond the original SQLite cutoff. [A]

Era boundary is the named `store.DayPlanEraStart`, built by `store/attribution.go:146` and `:153` from **15 August 2026 00:00 CT**, milliseconds **1786770000000**. Retained plans: **279 rows**, comprising **277 intraday versions and 2 weekly reference documents** (weekly rows **223, 257**). These yield **806 scenario rows**: **36 S0/trigger=none no-trade sentinels**, plus **770 real scenarios** in **49 session-day groups**. The sentinel constructor is `kernel/plan_doc.go:1032`; counting it as a hold trade would be false. These are retained accepted versions, not every failed planning attempt and not independent observations. [plan-inventory.csv](2026-09-08-the-strategy-data/plan-inventory.csv); [scenarios.csv](2026-09-08-the-strategy-data/scenarios.csv). [A]

Position cohort: **78 era rows → 65 eligible closed trades**, all linked to a retained scenario. Exclusions are disjoint: test rows **572, 573, 574** (`e7_farside_test`); `UNRESOLVABLE` rows **530, 539, 545, 546, 566, 571, 580**; remaining NULL `pnl_corrected` rows **576, 577, 579**. There are **4 raw NULL rows** including already-excluded test row **572**. NULL is not zero. No raw P&L, win rate, expectancy or profitability conclusion is used. The dispatch’s historical “58 trades, 12 days” is **not this snapshot’s cohort**: current eligible n is **65**, spread over **15 CT entry dates**. [position-cohort.json](2026-09-08-the-strategy-data/position-cohort.json) enumerates all included/excluded IDs. [A]

Basis: `docs/superpowers/SYSTEM-MAP.md` and `docs/superpowers/AUDIT-CHECKLIST.md` were read as maps/checklists, not substitutes for live evidence. The planner-preparation README was **not present on this dev tree**; it was read with `git show 6095ca58:docs/superpowers/reports/2026-09-07-planner-preparation-audit/README.md`. Its pinned historical claims are not silently promoted to current findings. Trading-policy research §09 at `982091d4` governs the descriptive-only boundary; the two-day audit at `f890ea60` is historical context. Every cited source file’s exact `git log -1` result appears in the provenance appendix.

## C1 · The playbook

| Condition | All retained n=806 | Real n=770 | ASIA n=320 | LONDON n=200 | NY n=250 |
| --- | --- | --- | --- | --- | --- |
| reject | 271 | 271 | 114 | 60 | 97 |
| sweep_reclaim | 201 | 201 | 87 | 55 | 59 |
| reclaim | 72 | 72 | 28 | 18 | 26 |
| breakdown_continue | 3 | 3 | 1 | 1 | 1 |
| breakup_continue | 1 | 1 | 0 | 1 | 0 |
| fvg_entry | 0 | 0 | 0 | 0 | 0 |
| acceptance | 56 | 56 | 25 | 15 | 16 |
| hold | 75 | 39 | 16 | 13 | 10 |
| breakout_retest | 127 | 127 | 49 | 37 | 41 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `all_retained_condition, play_session, stand_aside`; every cell carries its contributing row/scenario IDs. [A]

The **36 extra holds are no-trade placeholders**, not entries. The observed **0 fvg_entry scenarios out of 770** is a complete retained-document count, not a claim that FVG was unavailable or that all opportunities were declined. The current condition resolver shadows `breakout_retest` and `fvg_entry`; precedence is session override → strategy → environment → default (`kernel/condition_status.go:33`, `trader/armed_executor.go:26`). The resolved boot list in [runtime-log.json](2026-09-08-the-strategy-data/runtime-log.json) confirms those shadows. Historical authorship of **127 breakout_retests** therefore does not mean 127 currently executable setups. [A]

Fade/follow is an **annotated intended setup family**, not observed tick-by-tick approach direction. `reject` and `sweep_reclaim` fade the failed approach; acceptance, reclaim and break/continue families follow the established/reclaimed break. Breakout-retest follows the breakout impulse even though its immediate pullback approaches the entry from the other side. `hold` is overloaded: its explicit continuation/pullback wording is classified by the visible rules in `analyze.py:43`; **8/39 real holds remain ambiguous**. This convention is necessary because a reclaim or hold can describe more than one path. An exact executed “against the approach” ratio is **not recoverable** from condition labels alone. Excluding all holds leaves the less interpretive **472 explicit fades versus 259 follow-family descriptions (n=731; 1.82:1)**. The broader counts below are [A] reproducible annotations, [B] directional style—not measured entry microstructure.

| session | n | Fade | Follow | Ambiguous | Fade:follow | Long | Short |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ASIA | 320 | 209 | 106 | 5 | 1.97:1 | 133 | 187 |
| LONDON | 200 | 125 | 74 | 1 | 1.69:1 | 86 | 114 |
| NY | 250 | 162 | 86 | 2 | 1.88:1 | 122 | 128 |

| day_type_group | n | Fade | Follow | Ambiguous | Fade:follow | Long | Short |
| --- | --- | --- | --- | --- | --- | --- | --- |
| balance | 506 | 337 | 163 | 6 | 2.07:1 | 244 | 262 |
| premium-fade | 3 | 2 | 1 | 0 | 2.00:1 | 0 | 3 |
| premium_rejection | 3 | 2 | 1 | 0 | 2.00:1 | 0 | 3 |
| trend | 258 | 155 | 101 | 2 | 1.53:1 | 97 | 161 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `family_session, family_day_type, sides_session, sides_day_type`; every cell carries its contributing row/scenario IDs. [A]

| Condition | Balance n=506 | Trend n=258 | premium-fade n=3 | premium_rejection n=3 |
| --- | --- | --- | --- | --- |
| reject | 175 | 94 | 1 | 1 |
| sweep_reclaim | 147 | 52 | 1 | 1 |
| reclaim | 47 | 25 | 0 | 0 |
| breakdown_continue | 0 | 3 | 0 | 0 |
| breakup_continue | 1 | 0 | 0 | 0 |
| fvg_entry | 0 | 0 | 0 | 0 |
| acceptance | 29 | 26 | 0 | 1 |
| hold | 26 | 13 | 0 | 0 |
| breakout_retest | 81 | 45 | 1 | 0 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `play_day_type; play_day_type_raw preserves every unnormalized label`; every cell carries its contributing row/scenario IDs. [A]

For display, `trend`, `trend-down`, `trend_down` share the trend group; labels beginning `balance` share balance. The raw cross-tab is retained, not overwritten. No-trade and weekly documents stay in the inventory, outside the real-scenario denominator.

Current placement adds another distinction between a proposal and a trade: `trader/armed_executor.go:41` resolves `ARM_PLACE_TICKS` to **100 ticks** when unset; with MNQ **0.25-point ticks**, the limit placement band is **25 points** (`:969`, `:1056`). Both initial process environment and dotenv were unset at the final read. Already-marketable wrong-side limits are cancelled (`:1047`); the **stop-entry seam is explicitly off** in dotenv and the measured boot. Follow-family authorship therefore does not imply a stop-entry was sent. [A: runtime-final.json; runtime-log.json.]

## C2 · Levels chosen for a scenario, versus levels merely seated

Each scenario gets **one documented primary reference**: positive `confirm.ref_price`, otherwise the first trigger price matching a seated level within **0.125 points**. That tolerance is an audit join convention (half a MNQ tick), not a trading threshold. It resolves **762/770** to seated levels; **8/770** remain unresolved. Secondary trigger alternatives and target levels are not counted as additional primary anchors. Unknown labels stay `other`; composite labels stay composite. VWAP variants are grouped; raw labels, all matching level IDs and grades remain in [scenarios.csv](2026-09-08-the-strategy-data/scenarios.csv). This is a count of **levels selected for authored scenarios**, not proof those scenarios filled. [A/B]

| Primary level kind | Scenario n | ASIA | LONDON | NY | Seated level rows n |
| --- | --- | --- | --- | --- | --- |
| ONH | 120 | 73 | 30 | 17 | 194 |
| VWAP | 97 | 36 | 22 | 39 | 291 |
| ONL | 88 | 51 | 18 | 19 | 190 |
| PDC | 65 | 21 | 28 | 16 | 173 |
| RTH-H | 42 | 7 | 16 | 19 | 121 |
| PDL | 35 | 7 | 12 | 16 | 132 |
| EQL | 34 | 14 | 4 | 16 | 129 |
| OB | 34 | 15 | 14 | 5 | 289 |
| SWG-H | 32 | 18 | 4 | 10 | 103 |
| OR-L | 31 | 14 | 0 | 17 | 94 |
| EQH | 30 | 10 | 9 | 11 | 116 |
| PDH | 29 | 14 | 9 | 6 | 157 |
| RTH-L | 25 | 4 | 15 | 6 | 99 |
| SWG-L | 25 | 8 | 8 | 9 | 99 |
| SUPPLY | 18 | 5 | 3 | 10 | 120 |
| OR-H | 13 | 0 | 0 | 13 | 101 |
| DEMAND | 11 | 2 | 5 | 4 | 94 |
| unresolved | 8 | 6 | 2 | 0 | 0 |
| PWL | 6 | 0 | 0 | 6 | 8 |
| NPOC | 6 | 5 | 0 | 1 | 24 |
| RN | 4 | 1 | 0 | 3 | 10 |
| EVWAP | 4 | 2 | 1 | 1 | 19 |
| composite | 3 | 3 | 0 | 0 | 9 |
| POC | 2 | 1 | 0 | 1 | 34 |
| other | 2 | 0 | 0 | 2 | 5 |
| PDVWAP | 2 | 2 | 0 | 0 | 7 |
| IB-L | 2 | 1 | 0 | 1 | 2 |
| PWH | 1 | 0 | 0 | 1 | 1 |
| AS-L | 1 | 0 | 0 | 1 | 1 |

Seated denominator is **2,649 level-version rows**, including the retained documents; selected denominator is **770 scenario-version rows**. These are different units: a level can host more than one scenario, and repeated versions repeat levels. Production `kernel/levels_role.go:303` has a fallback classification; the descriptive taxonomy deliberately preserves unknown/composite labels rather than calling every unknown a round number. Full scenario kind × session × grade counts are in supplement.json `scenario_levels_session_grade`. Full seated kind × session × grade counts: [tables.json](2026-09-08-the-strategy-data/tables.json) `seated_levels`; primary scenario kind × session: `scenario_levels_session`. [A]

| Matched seated grade | Scenario n | Top kinds (counts within grade) |
| --- | --- | --- |
| A | 667 | AS-L 1, DEMAND 5, EQH 30, EQL 34, EVWAP 4, IB-L 2, NPOC 6, OB 17, ONH 112, ONL 85, OR-H 10, OR-L 22, PDC 36, PDH 28, PDL 35, PDVWAP 2, POC 2, PWH 1, PWL 6, RN 3, RTH-H 38, RTH-L 25, SUPPLY 5, SWG-H 32, SWG-L 25, VWAP 97, composite 3, other 1 |
| B | 40 | OB 1, OR-L 3, PDC 29, PDH 1, RN 1, RTH-H 4, other 1 |
| C | 55 | DEMAND 6, OB 16, ONH 8, ONL 3, OR-H 3, OR-L 6, SUPPLY 13 |
| unresolved | 8 | unresolved 8 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `scenario_levels_grade`; every cell carries its contributing row/scenario IDs. [A]

Do not confuse **seated level grade** with **scenario quality**. Scenario qualities are **B=464, C=175, A=130, A+=1 (n=770)**; those counts do not measure predictive reliability. `min_grade=B` in saved session settings is not `min_scenario_quality`; the latter resolves separately to **C** when absent (`store/strategy.go:1497`). [A: scenarios.csv; config.json.]

**Exact distance from the planner’s input price at authoring is unavailable.** No complete, version-bound input snapshot supplies that price. Two explicitly limited views follow: a price statement extracted from the saved reasoning, and the latest closed stored one-minute bar no more than two minutes before publication. Publication may occur after a long model request; neither view proves input-time distance. Saved ATR is the document’s `### 5m` ATR14, not a later live ATR.

| Distance view | Scenario median points (n) | Seated median points (n) | Scenario median ATR (n) | Seated median ATR (n) |
| --- | --- | --- | --- | --- |
| Reasoning-stated price | 28.75 (n=334) | 62.75 (n=999) | 1.46 (n=331) | 2.96 (n=990) |
| Publication closed-bar proxy | 33.89 (n=677) | 59.50 (n=2383) | 1.59 (n=674) | 2.65 (n=2330) |

Evidence: [summary.json](2026-09-08-the-strategy-data/summary.json) `dist`, each distribution lists contributing IDs; [plan-inventory.csv](2026-09-08-the-strategy-data/plan-inventory.csv) preserves the extracted price quote, ATR and bar natural key. Reasoning price exists for **105/279** plans; the publication proxy for **240/279**; saved ATR for **266/279**. Scenario anchors sit nearer than the whole seated map in these available subsets [B], but differing denominators, repeated levels and missing inputs prevent an exact all-plan authoring-distance claim.

## C3 · Side and time

Session/day-type long–short counts are in C1. Overall the authored set is **341 long / 429 short (n=770)**. Eligible fills are **25 long / 40 short (n=65)**; authored counts and filled counts are separate populations. [tables.json](2026-09-08-the-strategy-data/tables.json) `eligible_position_side` and `sides_session` enumerate IDs. [A]

| Stored regime-label text | Long | Short | n |
| --- | --- | --- | --- |
| neutral | 12 | 19 | 31 |
| not-recorded | 271 | 356 | 627 |
| up | 58 | 54 | 112 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `sides_regime`; every cell carries its contributing row/scenario IDs. [A]

These are labels embedded in accepted `bias_label`, **not a reconstructed machine regime**. All **59 retained planner_read_facts rows** lack a plan link; their IDs are enumerated in [summary.json](2026-09-08-the-strategy-data/summary.json) `facts_without_plan_link_ids`. Joining them to nearby timestamps would invent version provenance. Thus a machine-regime-by-side verdict is unavailable. [A]

| Publication hour CT | Authored scenarios n | Distinct accepted orders first observed n |
| --- | --- | --- |
| 00:00–00:59 | 48 | 0 |
| 01:00–01:59 | 53 | 0 |
| 02:00–02:59 | 46 | 2 |
| 03:00–03:59 | 26 | 0 |
| 04:00–04:59 | 23 | 0 |
| 05:00–05:59 | 7 | 0 |
| 06:00–06:59 | 11 | 0 |
| 07:00–07:59 | 49 | 0 |
| 08:00–08:59 | 94 | 1 |
| 09:00–09:59 | 46 | 1 |
| 10:00–10:59 | 40 | 0 |
| 11:00–11:59 | 32 | 0 |
| 12:00–12:59 | 23 | 1 |
| 13:00–13:59 | 20 | 0 |
| 14:00–14:59 | 7 | 0 |
| 15:00–15:59 | 0 | 0 |
| 16:00–16:59 | 13 | 0 |
| 17:00–17:59 | 52 | 1 |
| 18:00–18:59 | 50 | 2 |
| 19:00–19:59 | 34 | 0 |
| 20:00–20:59 | 29 | 2 |
| 21:00–21:59 | 19 | 0 |
| 22:00–22:59 | 20 | 1 |
| 23:00–23:59 | 28 | 1 |

Authoring hour is the accepted document’s **publication time**, repeated for its scenarios—not model request start. Total authored n=770. Submission history cannot be reconstructed from `armed_orders.updated_at` or `created_at`: those are mutable authorization-state timestamps. The order column instead uses **12 distinct signal IDs**, deduplicating **24 accepted-risk rows** by earliest receipt; IDs **1/2, 3/4, …, 23/24** are paired observations of the same orders. It is **first observed acceptance**, not an exact wire-submit clock. Its coverage is only **6 September 18:30:00 CT through 8 September 17:23:16 CT**. [supplement.json](2026-09-08-the-strategy-data/supplement.json) `authored_hour`, `unique_accepted_orders`, `unique_acceptance_hour`; all first-receipt timestamps and arm IDs are preserved. Blank historical acceptance coverage is not zero placements. [A]

**01:30 CT and 08:00 CT are scheduled preparation times**, LONDON and NY respectively; they are not automatic trade times. The measured boot registry opens LONDON at **02:00 CT** and NY at **08:30 CT**; ASIA prepares **16:30 CT** and opens **17:00 CT**. The broad authored family emphasis differs: LONDON 125:74 (n=200 plus 1 ambiguous), NY 162:86 (n=250 plus 2 ambiguous), and short share is **114/200 versus 128/250**. This is a session-level comparison, not evidence that the clock alone causes the difference. The sparse acceptance subset cannot establish equivalent execution behavior at those exact half-hours. [A/B: runtime-log.json; C1.]

## C4 · What it skips—and what the record cannot call a skip

Definitions: authored = real scenario-version rows; arm evidence = those with a retained arm row matched on plan + `armed_under_version` (fallback `version`) + scenario; reached = stored one-minute high/low contains the primary anchor in the observation window; filled = eligible positions linked to the plan’s trade date. Multiple fills can belong to one scenario. **These columns are not a funnel:** old decision-path fills can have no surviving arm row. Arm rows are reused on reauthorization (`store/armed_orders.go:221`, `:310`), so “no retained arm evidence” must not be called “never armed”.

Reach window: after publication, before next retained version of the same plan, recorded dormant/invalidated/expired event, current-registry session close, or snapshot cutoff—whichever comes first. Only fully contained minute bars count. Those historical session-close boundaries are an audit convention, not a reconstruction of every historic config. A touch on incomplete history proves observed reach; absence proves “not observed” only with a complete window. No minute-level reconstruction proves that the full entry confirmation, order permission or tick path occurred. The **306 observed / 369 not observed in complete windows / 95 unknown (n=770)** are anchor-reach results, not eligible opportunities. Each observed row records its first touching bar ID in scenarios.csv. [A/B]

| Plan trade date (CT) | Plan row IDs | Authored n | With retained arm n | Reached yes / no / unknown | Filled scenarios n | Eligible fills n (position IDs) |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-08-15 | 1,2 | 3 | 0 | 0 / 0 / 3 | 0 | 0 (none in cohort) |
| 2026-08-16 | 3,4,5,6,7,8,9,10 | 20 | 0 | 0 / 0 / 20 | 0 | 0 (none in cohort) |
| 2026-08-17 | 11,12,13,14,15,16,17,18,19,20 | 25 | 0 | 0 / 0 / 25 | 0 | 0 (none in cohort) |
| 2026-08-18 | 21,22,23,24,25,26,27,28 | 14 | 0 | 0 / 0 / 14 | 0 | 0 (none in cohort) |
| 2026-08-19 | 29,30,31,32,33,34,35,36,37,38 | 30 | 0 | 9 / 6 / 15 | 6 | 7 (521,522,523,524,525,526,527) |
| 2026-08-20 | 39,40,41,42,43,44,45 | 21 | 0 | 15 / 6 / 0 | 8 | 10 (528,529,531,532,533,534,535,536,537,538) |
| 2026-08-21 | 46,47,48,49,50 | 15 | 0 | 10 / 5 / 0 | 3 | 5 (540,541,542,543,544) |
| 2026-08-22 | no retained plans | unknown | unknown | unknown | unknown | unknown |
| 2026-08-23 | 51,52,53,54 | 0 | 0 | 0 / 0 / 0 | 0 | 0 (none in cohort) |
| 2026-08-24 | 55,56,57,58,59,60,61,62,63,64,65,66,67,68 | 37 | 0 | 17 / 20 / 0 | 5 | 7 (547,548,549,550,551,552,553) |
| 2026-08-25 | 69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88 | 65 | 0 | 25 / 36 / 4 | 6 | 8 (554,555,556,557,558,559,560,561) |
| 2026-08-26 | 89,90,91,92,93,94,95,96,97,98,99,100,101,102,103,104,105,106,107,108,109,110,111,112,113,114,115,116,117,118,119,120,121 | 106 | 0 | 43 / 60 / 3 | 2 | 2 (562,563) |
| 2026-08-27 | 122,123,124,125,126,127,128,129,130,131,132,133,134,135,136,137,138,139,140,141,142,143,144,145,146 | 77 | 1 | 39 / 38 / 0 | 3 | 3 (564,565,567) |
| 2026-08-28 | 147,148,149,150,151,152,153,154,155,156,157,158,159 | 42 | 6 | 13 / 29 / 0 | 3 | 3 (568,569,570) |
| 2026-08-29 | no retained plans | unknown | unknown | unknown | unknown | unknown |
| 2026-08-30 | 162,163,164 | 7 | 3 | 5 / 2 / 0 | 0 | 0 (none in cohort) |
| 2026-08-31 | 165,166,167,168,169,170,171,172,173,174,175,176,223 | 21 | 6 | 12 / 9 / 0 | 2 | 2 (575,578) |
| 2026-09-01 | 177,178,179,180,181,182,183,184,185,186,187,188,189,190,191,192,193,194,195,196,198 | 44 | 9 | 27 / 17 / 0 | 7 | 7 (581,582,583,584,585,586,587) |
| 2026-09-02 | 197,199,200,201,202,203,204,205,206,207,208,209,210,211,212,213,214,215,216,217,218,219,220,221,222,224,225,226,227,228,229,230,231 | 106 | 3 | 28 / 75 / 3 | 3 | 3 (588,589,590) |
| 2026-09-03 | 232,233,234,235,236,237,238,239,240,241,242,243,244,245,246,247 | 48 | 3 | 25 / 20 / 3 | 1 | 1 (591) |
| 2026-09-04 | 248,249,250,251,252,253,254,255,256 | 21 | 4 | 5 / 16 / 0 | 0 | 0 (none in cohort) |
| 2026-09-05 | no retained plans | unknown | unknown | unknown | unknown | unknown |
| 2026-09-06 | 258,259,260,261,262,263 | 17 | 4 | 5 / 9 / 3 | 1 | 1 (592) |
| 2026-09-07 | 257,264,265,266,267,268 | 16 | 4 | 9 / 7 / 0 | 1 | 1 (593) |
| 2026-09-08 | 269,270,271,272,273,274,275,276,277,278,279,280,281 | 35 | 9 | 19 / 14 / 2 | 4 | 5 (594,595,596,597,598) |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `calendar_days; session_days contains arm IDs as well as scenario and position IDs`; every cell carries its contributing row/scenario IDs. [A]

Rows dated 22 August, 29 August and 5 September have **no retained plan documents**, not a measured opportunity count. The 23 August row is a retained no-trade document, not an empty extraction. Plan trade date and calendar entry date differ for overnight ASIA; [supplement.json](2026-09-08-the-strategy-data/supplement.json) `fill_calendar_day` supplies the separate actual-entry-date view. No claim that filled = armed = reached is made.

Among **770 real scenarios**, **52 have retained arm evidence** and **718 do not**. Of the latter, **587 have no enabled arm specification**, while **131 have an enabled arm but no surviving matched row**. The **183 enabled complete geometries** therefore cannot be called 183 placed orders. This separation survives exact row/version joining. [supplement.json](2026-09-08-the-strategy-data/supplement.json) `arm_enabled`. [A]

| Condition | With arm evidence n=52 | Without retained arm n=718 |
| --- | --- | --- |
| reject | 29 | 242 |
| sweep_reclaim | 18 | 183 |
| reclaim | 3 | 69 |
| breakdown_continue | 0 | 3 |
| breakup_continue | 1 | 0 |
| fvg_entry | 0 | 0 |
| acceptance | 0 | 56 |
| hold | 0 | 39 |
| breakout_retest | 1 | 126 |

| Scenario quality | With arm evidence | Without retained arm |
| --- | --- | --- |
| A+ | 0 | 1 |
| A | 14 | 116 |
| B | 36 | 428 |
| C | 2 | 173 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `arm_presence, arm_presence_quality`; every cell carries its contributing row/scenario IDs. [A]

The publication-price proxy is nearer for scenarios with an arm row: median **25.64 points / 1.16 ATR (n=52)** versus **34.25 points (n=625), 1.60 ATR (n=622)** without. Those are associations in available data, not a reconstructed refusal reason. The no-arm group mixes legacy plans, shadow conditions, unavailable/invalid geometry, changed versions and possible gate refusals. The current code explicitly skips absent/disabled arm specs (`trader/armed_executor.go:317`) and shadow conditions; individual historic non-arm causes are not persistently linked for every scenario. [A/B: summary.json `dist.with_arm_*`, `dist.without_arm_*`.]

## C5 · Is it the same every day?

**No: a recognizable family of setups is reused, but side, condition and reference levels change both across days and within revised plans.** Below is every retained real session-day, with counts rather than an impressionistic label. Abbreviations: R=reject, S=sweep_reclaim, C=reclaim, A=acceptance, H=hold, B=breakout_retest, D=breakdown_continue, U=breakup_continue. F:W is annotated fade:follow; ? is ambiguous. Each row links to exact plan rows; scenario IDs and full raw labels are in the evidence JSON/CSV. [A]

| CT session-day | Plan row IDs | n | Declared types (n) | Plays (n) | F:W:?; ratio | Long:short | Primary level kinds (n) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-08-15:NY | 2 | 3 | balance 3 | B 1, C 1, R 1 | 1:2:0; 0.50:1 | 1:2 | EQH 1, PDC 1, other 1 |
| 2026-08-16:ASIA | 5,6,7,8,9 | 14 | balance 8, trend 6 | A 2, B 1, C 2, H 4, R 5 | 7:5:2; 1.40:1 | 7:7 | EQL 5, ONH 6, ONL 2, PDC 1 |
| 2026-08-16:NY | 3,4 | 6 | balance 6 | B 2, R 2, S 2 | 4:2:0; 2.00:1 | 4:2 | PDL 2, RN 2, RTH-H 2 |
| 2026-08-17:ASIA | 11,16,17,19,20 | 13 | balance 13 | B 1, C 1, H 1, R 5, S 5 | 10:2:1; 5.00:1 | 8:5 | EQH 1, EQL 2, ONH 4, ONL 2, PDC 1, composite 3 |
| 2026-08-17:LONDON | 12 | 3 | trend 3 | H 1, R 1, S 1 | 3:0:0; no follows | 2:1 | ONH 1, PDC 1, PDH 1 |
| 2026-08-17:NY | 13,14,15 | 9 | trend 9 | A 1, B 2, C 1, H 2, R 2, S 1 | 4:4:1; 1.00:1 | 6:3 | EQH 3, EQL 2, OR-L 2, PDC 1, PWH 1 |
| 2026-08-18:ASIA | 23,24,28 | 8 | balance 2, trend 6 | A 1, B 1, C 2, R 3, S 1 | 4:4:0; 1.00:1 | 3:5 | EQH 1, IB-L 1, ONH 3, OR-L 1, PDC 1, PDL 1 |
| 2026-08-18:LONDON | 21 | 3 | balance 3 | B 1, C 1, R 1 | 1:2:0; 0.50:1 | 1:2 | PDL 2, RTH-L 1 |
| 2026-08-18:NY | 22 | 3 | balance 3 | C 1, R 1, S 1 | 2:1:0; 2.00:1 | 2:1 | ONL 1, PDL 2 |
| 2026-08-19:ASIA | 35,36,37,38 | 12 | balance 12 | A 3, B 1, C 3, H 1, R 4 | 5:7:0; 0.71:1 | 4:8 | NPOC 1, ONH 3, ONL 2, OR-L 6 |
| 2026-08-19:LONDON | 29,30,31 | 9 | balance 9 | A 3, B 1, C 2, R 3 | 3:6:0; 0.50:1 | 3:6 | EQH 3, EQL 1, ONH 3, RTH-H 1, RTH-L 1 |
| 2026-08-19:NY | 32,33,34 | 9 | balance 9 | A 2, B 2, C 1, R 2, S 2 | 4:5:0; 0.80:1 | 4:5 | EQH 1, ONL 1, OR-L 1, PDC 3, PDL 1, RTH-H 1, RTH-L 1 |
| 2026-08-20:ASIA | 43,44,45 | 9 | trend 9 | A 2, B 1, C 2, R 4 | 4:5:0; 0.80:1 | 2:7 | EQH 1, ONH 3, ONL 1, OR-L 2, PDL 2 |
| 2026-08-20:LONDON | 39,40 | 6 | balance 3, trend 3 | A 1, B 2, R 3 | 3:3:0; 1.00:1 | 1:5 | EQH 1, ONH 2, ONL 1, PDC 2 |
| 2026-08-20:NY | 41,42 | 6 | balance 3, trend 3 | A 3, C 1, R 2 | 2:4:0; 0.50:1 | 1:5 | OR-H 1, PDC 1, PDL 1, RN 1, RTH-L 2 |
| 2026-08-21:LONDON | 46,47,48,49 | 12 | balance 12 | A 2, B 2, C 3, H 1, R 3, S 1 | 5:7:0; 0.71:1 | 5:7 | EQH 1, ONH 4, PDC 4, RTH-H 3 |
| 2026-08-21:NY | 50 | 3 | trend 3 | A 1, B 1, R 1 | 1:2:0; 0.50:1 | 0:3 | ONL 1, PDC 1, RTH-H 1 |
| 2026-08-24:ASIA | 64,66,67,68 | 13 | balance 4, trend 9 | A 2, B 3, C 1, R 5, S 2 | 7:6:0; 1.17:1 | 3:10 | EQL 1, NPOC 1, OB 1, ONH 3, ONL 1, OR-L 1, PDH 1, RN 1, RTH-H 2, SUPPLY 1 |
| 2026-08-24:NY | 56,57,58,59,60,61,62,63 | 24 | trend 24 | A 6, B 2, C 1, R 10, S 5 | 15:9:0; 1.67:1 | 6:18 | AS-L 1, EQL 5, IB-L 1, ONL 5, OR-L 3, PDL 2, PWL 6, other 1 |
| 2026-08-25:ASIA | 73,75,76,77,78,79,82,83,84,85,86,87,88 | 49 | balance 31, trend 18 | A 12, B 11, C 1, H 4, R 12, S 9 | 24:24:1; 1.00:1 | 23:26 | OB 7, ONH 18, ONL 7, PDC 8, PDH 5, RTH-H 1, RTH-L 2, unresolved 1 |
| 2026-08-25:LONDON | 69,70 | 8 | balance 5, trend 3 | A 1, B 2, C 1, H 2, R 1, S 1 | 4:4:0; 1.00:1 | 6:2 | EQL 1, ONH 2, PDC 1, PDH 2, RTH-H 2 |
| 2026-08-25:NY | 71,72 | 8 | trend 8 | A 2, B 1, C 1, H 1, R 2, S 1 | 4:4:0; 1.00:1 | 4:4 | EQH 2, EQL 1, OR-H 1, OR-L 1, PDH 1, RTH-H 2 |
| 2026-08-26:ASIA | 112,114,115,116,117,118,119,120,121 | 28 | balance 6, premium-fade 3, trend 19 | A 3, B 7, H 1, R 8, S 9 | 18:10:0; 1.80:1 | 3:25 | DEMAND 2, EQH 5, EQL 6, EVWAP 1, NPOC 1, OB 5, ONH 2, ONL 1, SUPPLY 1, VWAP 3, unresolved 1 |
| 2026-08-26:LONDON | 89,90,91,92,93,94,95,97,98,99,101,102,103 | 51 | balance 35, trend 16 | A 3, B 12, C 5, H 8, R 7, S 16 | 28:22:1; 1.27:1 | 21:30 | OB 9, ONH 9, PDC 14, PDL 8, RTH-H 1, RTH-L 9, unresolved 1 |
| 2026-08-26:NY | 104,105,106,107,108,109,110 | 27 | balance 24, trend 3 | B 7, H 3, R 12, S 5 | 19:8:0; 2.38:1 | 12:15 | NPOC 1, OB 2, ONH 3, OR-H 2, OR-L 3, PDC 4, PDL 5, RTH-H 5, SUPPLY 2 |
| 2026-08-27:ASIA | 135,136,137,138,139,140,141,142,143,144,145,146 | 45 | balance 32, trend 13 | B 10, C 1, H 3, R 12, S 19 | 32:12:1; 2.67:1 | 12:33 | NPOC 1, ONH 8, ONL 6, OR-L 1, PDC 2, PDH 3, PDL 2, POC 1, RTH-H 1, RTH-L 1, SUPPLY 1, SWG-H 6, SWG-L 1, VWAP 11 |
| 2026-08-27:LONDON | 122,123,124,125,126,127,128 | 22 | balance 14, trend 8 | A 2, B 7, R 4, S 9 | 13:9:0; 1.44:1 | 6:16 | DEMAND 1, EQH 2, EQL 2, EVWAP 1, OB 2, ONH 2, PDH 2, SUPPLY 3, VWAP 7 |
| 2026-08-27:NY | 131,132,133 | 10 | balance 7, premium_rejection 3 | A 1, B 1, R 4, S 4 | 8:2:0; 4.00:1 | 3:7 | DEMAND 1, EQL 1, EVWAP 1, ONH 1, OR-H 1, VWAP 5 |
| 2026-08-28:LONDON | 147,148,149,150,151,152 | 20 | balance 17, trend 3 | A 1, B 5, R 4, S 10 | 14:6:0; 2.33:1 | 4:16 | DEMAND 2, ONL 4, PDC 2, PDH 3, PDL 1, RTH-H 4, SWG-H 1, VWAP 2, unresolved 1 |
| 2026-08-28:NY | 153,154,155,156,157,158,159 | 22 | balance 16, trend 6 | B 7, H 1, R 7, S 7 | 15:7:0; 2.14:1 | 11:11 | EQH 1, EQL 2, ONH 1, ONL 2, OR-H 1, OR-L 1, PDH 4, PDL 2, SUPPLY 1, SWG-L 1, VWAP 6 |
| 2026-08-30:ASIA | 162,163 | 7 | balance 4, trend 3 | B 1, D 1, R 2, S 3 | 5:2:0; 2.50:1 | 5:2 | NPOC 1, ONL 2, PDC 1, PDL 1, VWAP 1, unresolved 1 |
| 2026-08-31:LONDON | 166 | 4 | balance 4 | B 1, R 3 | 3:1:0; 3.00:1 | 3:1 | PDC 1, PDH 1, RTH-L 1, VWAP 1 |
| 2026-08-31:NY | 170,171,172,173,174 | 17 | balance 14, trend 3 | B 2, C 1, H 1, R 7, S 6 | 13:3:1; 4.33:1 | 10:7 | OB 1, ONH 1, OR-H 1, OR-L 2, PDL 1, SWG-H 5, SWG-L 1, VWAP 5 |
| 2026-09-01:ASIA | 192,193,194,195,196,198 | 15 | balance 12, trend 3 | B 2, C 4, R 4, S 5 | 9:6:0; 1.50:1 | 14:1 | ONL 5, OR-L 2, PDC 2, RTH-H 1, SWG-L 1, VWAP 4 |
| 2026-09-01:LONDON | 178,179,180,181,182 | 13 | balance 3, trend 10 | A 1, D 1, R 8, S 3 | 11:2:0; 5.50:1 | 4:9 | DEMAND 1, ONL 5, RTH-L 1, SWG-L 4, VWAP 2 |
| 2026-09-01:NY | 183,184,185,186,187 | 16 | trend 16 | C 1, D 1, R 10, S 4 | 14:2:0; 7.00:1 | 5:11 | EQL 2, OB 1, ONL 4, OR-L 2, SUPPLY 1, SWG-H 1, SWG-L 4, VWAP 1 |
| 2026-09-02:ASIA | 216,217,218,219,220,221,222,224,225,226,227,228,229,230,231 | 45 | balance 43, trend 2 | B 2, C 2, R 25, S 16 | 41:4:0; 10.25:1 | 15:30 | ONH 8, ONL 17, OR-L 1, PDH 3, PDVWAP 2, RTH-L 1, SUPPLY 1, SWG-H 4, SWG-L 1, VWAP 7 |
| 2026-09-02:LONDON | 199,200,201,202,204 | 20 | balance 20 | B 4, C 2, H 1, R 8, S 5 | 14:6:0; 2.33:1 | 16:4 | ONH 4, ONL 4, PDC 1, PDL 1, RTH-H 4, RTH-L 2, VWAP 4 |
| 2026-09-02:NY | 203,205,206,207,208,209,210,211,212,213,214,215 | 41 | balance 37, trend 4 | B 6, C 9, H 1, R 15, S 10 | 26:15:0; 1.73:1 | 32:9 | ONH 5, ONL 3, OR-H 2, OR-L 1, PDC 5, RTH-H 7, RTH-L 3, SUPPLY 2, VWAP 13 |
| 2026-09-03:ASIA | 241,242,243,244,245,246,247 | 22 | balance 11, trend 11 | B 7, C 4, R 3, S 8 | 11:11:0; 1.00:1 | 14:8 | EVWAP 1, ONH 7, ONL 2, PDC 1, RTH-H 1, SWG-H 6, VWAP 2, unresolved 2 |
| 2026-09-03:LONDON | 232,234 | 5 | balance 5 | R 2, S 3 | 5:0:0; no follows | 2:3 | ONH 1, ONL 2, SWG-H 1, VWAP 1 |
| 2026-09-03:NY | 233,235,236,237,238,239,240 | 21 | balance 6, trend 15 | B 4, C 5, R 9, S 3 | 12:9:0; 1.33:1 | 12:9 | DEMAND 1, EQH 2, EQL 1, ONH 2, ONL 1, OR-H 2, OR-L 1, PDH 1, SWG-H 4, VWAP 6 |
| 2026-09-04:LONDON | 248,249,250 | 9 | balance 3, trend 6 | A 1, C 2, R 3, S 3 | 6:3:0; 2.00:1 | 4:5 | DEMAND 1, OB 2, ONH 2, PDC 1, SWG-H 1, VWAP 2 |
| 2026-09-04:NY | 252,253,254,255,256 | 12 | balance 12 | B 1, C 1, R 4, S 6 | 10:2:0; 5.00:1 | 3:9 | DEMAND 2, ONH 4, OR-H 1, POC 1, RTH-H 1, SUPPLY 1, VWAP 2 |
| 2026-09-06:ASIA | 258,259,260,261,262,263 | 17 | balance 15, trend 2 | C 3, R 9, S 5 | 14:3:0; 4.67:1 | 12:5 | EQH 1, ONH 2, ONL 2, PDC 2, PDL 1, RTH-H 1, SUPPLY 1, SWG-L 3, VWAP 3, unresolved 1 |
| 2026-09-07:ASIA | 264,265,266,267,268 | 16 | balance 12, trend 4 | B 1, C 1, H 2, R 9, S 3 | 12:4:0; 3.00:1 | 5:11 | EQH 1, OB 2, ONH 5, PDC 1, PDH 1, SWG-H 2, VWAP 4 |
| 2026-09-08:ASIA | 279,280,281 | 7 | balance 5, trend 2 | C 1, R 4, S 2 | 6:1:0; 6.00:1 | 3:4 | ONH 1, ONL 1, PDC 1, PDH 1, SWG-L 2, VWAP 1 |
| 2026-09-08:LONDON | 269,270,271,272,273 | 15 | balance 15 | C 2, R 9, S 3, U 1 | 12:3:0; 4.00:1 | 8:7 | EQH 2, OB 1, ONL 2, PDC 1, RTH-H 1, SWG-H 1, SWG-L 4, VWAP 3 |
| 2026-09-08:NY | 274,275,276,277,278 | 13 | balance 8, trend 5 | B 2, C 2, H 1, R 6, S 2 | 8:5:0; 1.60:1 | 6:7 | EQH 1, EQL 2, OB 1, ONL 1, OR-H 1, SUPPLY 3, SWG-L 3, VWAP 1 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `session_days`; every cell carries its contributing row/scenario IDs. [A]

The similarity comparison is **defined before interpreting the pairs**: compare same-session days with at least **6 scenarios and 2 retained versions**; distance is the equal-weight mean of total-variation distances of condition, side and primary-level-kind distributions. Zero means identical marginals, one disjoint marginals. This is an audit metric, not a trading score or significance test. There are **280 qualifying pairs**; [tables.json](2026-09-08-the-strategy-data/tables.json) `similarity_pairs` records all members, n, plan IDs and component distances. A different metric or inclusion floor can select different extrema. [A]

**Most similar: 2026-08-18:ASIA (n=8; plan rows [23, 24, 28]) versus 2026-08-20:ASIA (n=9; rows [43, 44, 45]).** Mean distance **0.208**; condition **0.167**, side **0.153**, level-kind **0.306**. Full plan text and version IDs: [supplement.json](2026-09-08-the-strategy-data/supplement.json) `closest_pair_quotes`. [A]

**Most different: 2026-08-25:LONDON (n=8; plan rows [69, 70]) versus 2026-09-01:LONDON (n=13; rows [178, 179, 180, 181, 182]).** Mean distance **0.705**; condition **0.673**, side **0.442**, level-kind **1.000**. Full plan text and version IDs: [supplement.json](2026-09-08-the-strategy-data/supplement.json) `farthest_pair_quotes`. [A]

The similar ASIA pair shows the same choice set in different markets: **row 23 S1** says “5m acceptance below 29568.25 IB-L and 29564.50 EQL”; **row 43 S1** says “Acceptance below ONL: price breaks and holds below 29323.0 with two 5m closes.” Both sit beside rejection shorts and reclaim longs. These are quotations from the stored plans, not assertions that their conditions were satisfied. [A]

The different LONDON pair changes emphasis: **row 69 S1** says “After the 01:00 CT sweep-up, 29243.75 RTH-H is support; any dip that prints 2×5m closes back above 29243.75 is a long.” **Row 182 S1** says “Price retests ONL 29123.25 and rejects with a 5m close below it; one-sided delivery continues.” The first day has **6 longs / 2 shorts (n=8)**; the second **4 / 9 (n=13)**. The comparison spans each day’s retained versions, not just these illustrative quotes. [A]

The variation is **partly associated with declared day type**, as C6 quantifies, but cannot be attributed to that label alone. Market inputs, changing level locations, time, version selection and code evolution also differ. Calling the residual “arbitrary” would exceed this record. [B]

## C6 · Day type and bias versus behavior

Balance contains **337 fades / 506 scenarios**, trend **155 / 258**: **66.60% (n=506)** versus **60.08% (n=258)**, a descriptive difference of **6.52 percentage points**. Follow-family shares are **163/506** and **101/258**. The minor premium labels each contain **2 fades / 1 follow (n=3)**. Excluding ambiguous holds does not turn trend into an all-follow strategy. C1 gives every condition cell; [supplement.json](2026-09-08-the-strategy-data/supplement.json) `family_session_daytype` gives the session-stratified cells. [A/B]

Equal weighting of retained versions rather than scenarios gives mean fade shares **67.78% across 156 balance versions** and **60.44% across 83 trend versions**; scenario denominators remain **506 and 258**. This sensitivity check reduces the influence of versions containing more scenarios, but does not make revisions independent or remove market confounding. Plan IDs are in [supplement.json](2026-09-08-the-strategy-data/supplement.json) `declared_type_per_plan`. [A]

| Declared bias | Long scenarios | Short scenarios | n |
| --- | --- | --- | --- |
| long | 193 | 78 | 271 |
| neutral | 61 | 72 | 133 |
| short | 87 | 279 | 366 |

Evidence: [tables.json](2026-09-08-the-strategy-data/tables.json), `bias_vs_side`; every cell carries its contributing row/scenario IDs. [A]

**The label is contextual, not a hard strategy switch.** The record shows an association with play mix, so “purely decorative” is too strong; it does not prove the declaration causes a choice. There is no blanket `day_type=trend → no entry` in `trader/armed_executor.go:1878` or `trader/entry_gate.go:147`. Strict checks the cited scenario’s side; only `plan_mode=direction` directly enforces plan bias (`trader/entry_gate.go:201`, `trader/armed_executor.go:1891`). Thus **78 short scenarios under long bias and 87 long scenarios under short bias** are present (**165/770**), without contradicting strict’s current scenario-level contract. They do contradict reading a bias label as an unconditional direction instruction. [A/B]

## C7 · Stops, targets and the shape of the trade

Complete geometry means positive parent-arm entry, stop and target, with entry unequal to stop. There are **183/770** such scenarios, all enabled; **587/770** lack it. A separate sign check found **0 wrong-side parent stops/targets among those 183**. These are authored parent geometries, not every split leg and not broker-accepted protection. [summary.json](2026-09-08-the-strategy-data/summary.json) `geometry_ids`, [supplement.json](2026-09-08-the-strategy-data/supplement.json) `geometry_side_errors`. [A]

| Measure | n | 25th percentile | Median | 75th percentile | Range |
| --- | --- | --- | --- | --- | --- |
| Authored stop points | 183 | 21.05 | 27.00 | 36.70 | 5.00–129.75 |
| Authored stop / saved ATR5m | 183 | 1.05 | 1.43 | 1.56 | 0.39–8.58 |
| Authored arm target / initial authored risk | 183 | 2.12 | 2.33 | 2.84 | 0.21–5.55 |
| Eligible hold minutes | 65 | 12.22 | 25.31 | 55.41 | 0.02–551.89 |
| Recorded filled initial-stop points | 49 | 26.00 | 34.25 | 45.25 | 3.75–92.75 |

To cover **all executable leg geometries**, expand explicit entry legs instead of counting their parent twice: **15 split scenarios contribute 30 legs**, and **168 unsplit scenarios contribute 168 legs**, for **198 complete leg geometries**. All 198 stops are on the correct risk side. This is an authored-leg inventory, not 198 placed orders. The current one-contract guard limits actual placement regardless of a document containing two entry legs. [A: supplement.json `effective_legs`; armed_executor.go:327 and :984.]

| Expanded arm-leg measure | n | 25th percentile | Median | 75th percentile | Range |
| --- | --- | --- | --- | --- | --- |
| stop_pts | 198 | 21.31 | 27.50 | 36.79 | 5.00–129.75 |
| stop_atr | 198 | 1.07 | 1.47 | 1.57 | 0.39–8.58 |
| target_R | 198 | 2.12 | 2.36 | 2.83 | 0.21–5.55 |

Every expanded-leg statistic carries its full IDs in [supplement.json](2026-09-08-the-strategy-data/supplement.json) `effective_leg_stats`. **101/198** expanded authored stops lie below 1.5×saved ATR; this remains authored arithmetic, not the observed live stop-composition winner. Parent and expanded-leg distributions are both shown so repeated split entries cannot silently change the unit. [A]

All distribution IDs are in [summary.json](2026-09-08-the-strategy-data/summary.json) `dist`; filled-stop IDs are **trade_excursions.id**, linked to position IDs in [excursions.json](2026-09-08-the-strategy-data/excursions.json). There are **60 excursion rows** attached to eligible positions, of which **49** have usable initial-stop prices; only **2/60** have positive entry ATR (excursion IDs **588,589**). These are insufficient to reconstruct a historical live-stop ATR distribution. [A]

The current stop is the **widest** of authored stop, nearest risk-side seated anchor plus clearance, and ATR floor (`trader/arm_stop_anchor.go:71`). The measured boot resolves floor **1.5×ATR5m** and maximum anchor distance **3.0×ATR5m**; clearance is **2 ticks** from `kernel/min_sl.go:40`. The composition occurs before arm R:R validation (`trader/armed_executor.go:429`). Studio resolves the current R:R floor to **2** (`trader/armed_executor.go:69`, `:1914`; runtime.json `/api/config/resolved`). A farther stop may therefore invalidate a once-valid authored target. These are current rules, not a claim every historical authored geometry passed them. [A]

**How often does the floor win instead of structure? Unanswerable from the retained record.** **98/183** authored stops are below 1.5×their saved ATR, but this does not say the live floor won: current live ATR can differ, a farther anchor can win, and stops are recomposed. `Bound` is not saved per historical acceptance. The log at `trader/armed_executor.go:432` prints selectively when changed or unanchored; it is not a complete denominator. Today’s retained examples show an **anchor winner at 17:01:20 CT** and an **authored winner at 17:21:16 CT** (runtime-log.json journal lines **29537,32566**); they do not establish a floor frequency. [A]

Targets are not all a clean first-obstacle 2R trade. Across **183** complete geometries the arm target’s median is **2.33R**, but the first listed target-chain node is **strictly positive and below 1R in 72/183**, and **nonpositive in 3/183** when measured directionally from the arm entry. Those are authored nodes, not executed partial exits. IDs and signed R are in [supplement.json](2026-09-08-the-strategy-data/supplement.json) `first_target_below_one_R`, `first_target_nonpositive`. The arm target and first obstacle must not be conflated. [A]

Eligible holds: **37/65 under 30 minutes**, **13/65 from 30 through 60**, **15/65 above 60**. All **65/65** entries are one contract; IDs are in [supplement.json](2026-09-08-the-strategy-data/supplement.json) `holds`, `quantity`. The median is **25.31 minutes**, with **12.22–55.41 minutes** covering the middle half. Extremes are position **597**, **1.192 seconds**, and **592**, **551.89 minutes**. Position 592’s long hold is not proof of an intended runner: the current source documents its historical protection incident (`trader/armed_executor.go:215`) and its stored exit was manual. No current open position was observed. [A]

**Trader’s reading [B]:** level-to-level intraday trading, including short holds and longer intraday swings. “Scalp only” does not fit the longer holds; “runner system” is unsupported by the one-contract bracket and currently suspended BE/trailing mechanisms. Holding-time shape is descriptive and includes manual/forced exits and past operational failures, not just ideal intended trades.

## C8 · What discretionary choices are actually implemented?

| Question | Current mechanism / retained evidence | Verdict and limit |
| --- | --- | --- |
| Stand aside on a trend day? | No day-type blanket veto in armed_executor.go:1878 or entry_gate.go:147; trend group contains 258 authored scenarios. Inactive/dormant plans cancel arms at armed_executor.go:239. | Refuted as a blanket trend-day rule. Specific gates can still stand aside. |
| Pick ONE trade of the day? | one_contract.go:119 checks working entries/open positions; armed_executor.go:984 guards both placement paths and permits one placement per pass. Saved max_daily_trades=3 is disabled, master guardrails=false (config.json). | No daily best-trade selector. One simultaneous commitment is not one daily trade. C4 has multiple eligible fills on several dates. |
| Size differently by conviction? | PlaceLimitEntry(..., side, 1, ...) at armed_executor.go:1069; current exit boot size=1. Eligible entry_quantity=1 for all 65 position IDs. | Refuted for current entry path and measured fills. The saved disabled max_contracts=2 is not current executable sizing. |
| Move a stop after entry? | BE mechanism auto_trader.go:157; trail auto_trader_trailing.go:115. Both refuse before wire when exitMechsSuspended() is true, exit_mechs_suspend.go:35. Boot BE=off, trail=off at 18:11:54 CT. | Mechanisms exist but are currently suspended. Do not call historical stop movement impossible or infer current behavior from saved enabled=true. |
| Take partials? | Current arm placement sends one contract with one stop and target; entry legs are not fractional exits. TCPTrader supports close quantity (trader/ninjatrader/tcp_trader.go:730), but that is not an automatic target-chain scale-out policy. | No automatic partial-plus-runner implementation demonstrated on this one-contract path; all 65 eligible entries are one contract. Target-chain nodes are plans, not partial-order receipts. |
| Re-enter after a stop? | store/armed_orders.go:310 keeps a terminal row terminal under the same version; a new version can reauthorize. Boot-sweep exception may reauthorize same version. auto_trader_loop.go:361 explicitly recognizes post_exit rescan. | Possible after fresh authorization; not an unconditional same-setup loop. A stop-specific empirical reentry rate cannot be recovered from mutable arm history. |
| Stop after N losses? | auto_trader_orders.go:112 reads ConsecutiveLossHalt; zero/absent is off. Field absent in saved config. Daily-loss and daily-trade limits also disabled. | Mechanism exists, currently off. No current N-loss-stop claim. Session caps are distinct and must not be substituted for it. |
| Avoid first minutes or news? | auto_trader_session.go:27/:103 checks first 5 minutes, lunch and T1 windows for decision-path entries; orders.go:281 calls it. kernel/calendar_blackout.go:14 sets T1 ±15m. auto_trader_clock.go:675 cancels arms and flattens around T1 before armed management. | Mechanisms exist, but complete arm-path entry exclusion is NOT established: see the call-site limit below. |

**Timing coverage limit [A/B]:** the strict arm path runs at `trader/auto_trader_loop.go:432`; `sessionEntryBlocked` is called from the decision-order path, not `entryGateForArm` (`trader/entry_gate.go:352`) or `runArmedPlacement` (`trader/armed_executor.go:952`). News cancellation/flattening is earlier at loop `:323`, but `enforceT1ForceFlatAt` returns false when flat and there was no cancel work (`trader/auto_trader_clock.go:708`), allowing the cycle to continue. The inspected path therefore does **not prove an unconditional first-five-minute/lunch/news ban on fresh arms**. No live placement was induced to test this. A code mechanism or a painted band is not proof every entry path is blocked.

A concrete observation reinforces the narrower lunch conclusion: accepted-risk rows **21/22** first observe the same order at **8 September 2026 12:54:19 CT**, inside the code’s **12:00–13:30 CT** lunch band (`kernel/no_trade_band.go:42`). This is acceptance evidence, not reconstructed exact submission time. The saved `blackout_enabled=false` is a separate risk-control setting; it does not erase the session/news functions just traced. Current session trade caps **ASIA=7, LONDON=10, NY=10** resolve from per-session saved settings and are checked in `trader/auto_trader_session.go:81`; the same call-path limitation applies to claiming they cover arms. [A]

## D · Verdict, differences from the system’s words, and unanswered questions

**D1 — Strategy:** Your system repeatedly maps named intraday references and proposes both a failed-break fade and a break/reclaim continuation. The retained proposals favor fades and shorts, but allow either side. The current strict executor turns only enabled, permitted scenario geometry into a one-contract order after its gates and broker-book checks. It uses a protective bracket and current fixed-stop/fixed-target posture, with lifecycle and scheduled forced exits. This is a level-based intraday strategy family, not proof of a profitable edge. [B, supported by C1–C8.]

**D2 — Consistency:** Same vocabulary, different daily composition. The session ratios and all **49** session-day rows show the variation; the defined **280-pair** comparison ranges from **0.208** to **0.705**. Balance/trend mix differs modestly, and both remain fade-heavy (**337/506**, **155/258**). Neither “identical every day” nor “random changes” is supported. [A/B: C5–C6.]

**D3 — Three discretionary differences:** (1) no single daily best-trade selection: competing authored scenarios plus mechanical admission; (2) no current conviction sizing: one contract; (3) no demonstrated automatic partial-plus-runner management: one-contract brackets, BE/trail suspended. These describe the choices a discretionary trader *could* exercise and this implementation does not currently exercise; they do not claim all discretionary traders behave the same way. [A/B: C8.]

| D4 · Words in the system | What the evidence supports |
| --- | --- |
| Saved prompt: “avoid … sideways oscillation”; custom note “No trade inside sideway zone”. | 506/770 authored scenarios carry a balance-family label. Balance is not a universal stand-aside label; this does not prove every entry was inside a precisely defined forbidden zone (that zone is not retained). C1/C6; config.json. |
| Saved frequency prompt: “2-4 trades per day”; “Single position holding time ≥ 30-60 minutes”. | Not hard current daily/hold constraints. C4 records plan-date fill counts including 10 on 20 August (IDs in that row); 37/65 holds were below 30 minutes. Code/current config has no enabled daily=one/three rule. |
| Saved frequency prompt: “50 point move stop loss to breakeven”. | Saved knob is 40 points, but runtime suspension prevents both BE and trail. Neither prompt 50 nor saved 40 describes current wire behavior. config.json; exit-runtime.json; exit_mechs_suspend.go:35. |
| Saved entry prompt: avoid “immediately restarting after closing positions”. | post_exit rescan is explicit at auto_trader_loop.go:361. Fresh-version reauthorization exists; no blanket post-close stand-down can be inferred. Rescan itself is not a fill. |
| Plan bias interpreted as a one-way instruction. | 165/770 authored scenarios oppose non-neutral declared bias. Strict validates scenario direction; direction mode is the separate bias gate. C6. |
| “Trend day” interpreted as continuation-only or a day to sit out. | 155 fade-family scenarios among 258 trend-labelled scenarios; no direct blanket trend veto found. C6/C8. |
| Guide scenario quality: “INFORMATIONAL — nothing gates on it” (web/src/guide/content/planCard.ts:96). | Overbroad: the code supports a min_scenario_quality gate at armed_executor.go:1898. Current missing setting resolves C, so current A+/A/B/C labels all satisfy it. It is not a measured win probability. |
| Guide says “lunch 11:30–13:30 ET → no entries” (web/src/guide/content/plays.ts:231). | That is 10:30–12:30 CT, whereas kernel/no_trade_band.go:42 defines 12:00–13:30 CT. Further, complete arm-path enforcement is not established; accepted rows 21/22 are at 12:54:19 CT. |
| Guide no-trade-band prose: “Three sources, all of them enforcing” (web/src/guide/content/planCard.ts:101). | Decision-path mechanisms exist, and news cancels/flat exists. The claim about every armed entry is stronger than the inspected call sites support. C8. |
| Guide target-chain / runner language (web/src/guide/content/plays.ts:45, :221). | 183 complete geometries have a target-chain description, but that is not a partial ladder. 72/183 first nodes lie at positive <1R and 3/183 are nonpositive from entry; all 65 eligible fills are one contract. C7. |
| Grade described as “trustworthy” (web/src/guide/content/levels.ts:12). | The audit counts labels and selected levels; it does not calibrate their reliability or identify a preferred level. No edge inference is justified. |
| Settings Guide describes BE and trailing as suspended (web/src/guide/content/settings.ts:401, :419). | Supported by the current exit boot; this is not a contradiction. Saved enabled flags alone would misdescribe behavior. |

D4 is exhaustive for the concrete prompt/Guide/label claims examined above, **not a claim to have audited every sentence of the entire Guide**. No published research ranking or previous audit performance claim is imported as evidence of this system’s edge.

**D5 — What cannot be answered from this retained record:** exact model-input price/time for every version; exact historical machine regime attached to each version; every placement timestamp; every historical arm authorization/cancel/rearm because rows are mutable; every non-arm reason; whether an anchor touch would have passed full confirmation and all gates; exact live floor-versus-anchor stop winner frequency; complete broker-accepted stop/target histories; an empirical stop-specific reentry rate; the causal effect of day-type declarations versus changing markets; calibrated grade reliability; the effect of a different strategy, target or threshold; and profitability. The CSV/JSON missing fields and enumerated unmatched/unknown subsets define those limits. These are evidence gaps, not proposed strategy changes.

## Source freshness and provenance

The following are literal `git log -1 --format="%H %cI %s" -- "<file>"` outputs for the source at the report worktree HEAD. The planner-audit command additionally pins `6095ca58`, as stated above. Source line references refer to that inspected tree. Generated evidence files have source queries and row IDs instead of fictitious preexisting git history. Machine-readable commands/output: [provenance.json](2026-09-08-the-strategy-data/provenance.json).

`docs/superpowers/SYSTEM-MAP.md`

```text
954f11b15f2e7615678f7d2b708c47895faebf1e 2026-09-08T18:38:16-05:00 fix(research): resolve archive paths before SQLite URI construction
```

`docs/superpowers/AUDIT-CHECKLIST.md`

```text
954f11b15f2e7615678f7d2b708c47895faebf1e 2026-09-08T18:38:16-05:00 fix(research): resolve archive paths before SQLite URI construction
```

`docs/superpowers/research/2026-09-08-trading-policy/README.md`

```text
0ec5bd2cb4c678e524cc21033d7622a7783765c1 2026-09-08T08:30:15-05:00 docs(research): publish four-policy trading study with 28 cited sources
```

`docs/superpowers/reports/2026-09-04-two-day-audit.md`

```text
f3c640c3f9799e6fa80ce124ae87ee915cad63ed 2026-09-04T07:26:52-05:00 docs(two-day audit D3): why the blindness went unalerted — a note, not a build
```

`store/attribution.go`

```text
60333cd5e7f8e66794bdbf2a5a72c0f40e378a77 2026-09-03T18:32:06-05:00 feat(data-integrity): D6 backfill (three states) + D7 boot lines + Guide + checklist 63
```

`kernel/plan_doc.go`

```text
d5e2414e0d30a275b6239f5d609f9e22f8e381d7 2026-09-08T16:02:39-05:00 feat(scenario-economics): enforce new authoring contract and preserve legacy unknowns
```

`kernel/levels_role.go`

```text
0cb06ae57ac8c818eabf58630f972ffe27e7d58f 2026-08-28T18:40:39-05:00 fix(pre-reopen): waterfall scenarioConds, GORM pool + persist watchdog, armed re-arm/cancel/dedup, stamp_pending lineage, SWG kinds, wake log verbs, WSL2 clock script
```

`store/strategy.go`

```text
5710cb5d353046aca531eb9ce049ab3bd780fdcb 2026-09-08T09:00:25-05:00 test(plan-liveness): pin unknown qualifiers and versioned readers
```

`store/armed_orders.go`

```text
0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb 2026-09-08T16:22:32-05:00 feat(research): link attempts, permissions, broker receipts and corrected outcomes
```

`trader/armed_executor.go`

```text
0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb 2026-09-08T16:22:32-05:00 feat(research): link attempts, permissions, broker receipts and corrected outcomes
```

`trader/entry_gate.go`

```text
01ce808839becd61120140e178bffa7cbc225d30 2026-09-05T12:12:00+00:00 fix(risk,planner): wire RiskForceFlat and BiasArmWarning — both shipped uncalled
```

`trader/arm_stop_anchor.go`

```text
4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep
```

`kernel/min_sl.go`

```text
4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep
```

`kernel/condition_status.go`

```text
d4c811698a819226a7a66d2acd6016591059c859 2026-08-31T16:53:40-05:00 0C shadow demotion: config-driven condition-status map (fvg_entry+breakout_retest shadow), arm-seam refusal + telemetry counter, inert 'shadowed' ledger state (boot sweep cancels resting), E8 complete counterfactual rows (R/ATR units, time bars, net-of-friction, ambiguous flag), resolved-map boot line; tests: wire-loopback silence, boot cancel, config flip, E8 coverage
```

`trader/one_contract.go`

```text
7797b36f8fcfca80c7a391d16e4b0eddbd054963 2026-09-07T10:25:18-05:00 fix(cancel): D4 — every cancel sender is guarded, and the census proves it
```

`trader/auto_trader.go`

```text
6310eaf8941f53194fa2c5e7368552eed1ed8d64 2026-09-07T23:51:51-05:00 merge: reconcile placement confirmation with pre-send identity and owner ruling
```

`trader/auto_trader_trailing.go`

```text
4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep
```

`trader/exit_mechs_suspend.go`

```text
4657560bbaf616fcde7457816da5b8aff22431dc 2026-09-02T07:33:39-05:00 fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep
```

`trader/auto_trader_loop.go`

```text
0eddf90e95184c106d5a8c6f8fabaa8059d667f5 2026-09-07T19:30:13-05:00 feat(session-calendar): D5 the MODE line, D6 the boot line, the Guide, the map and class 84
```

`trader/auto_trader_orders.go`

```text
0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb 2026-09-08T16:22:32-05:00 feat(research): link attempts, permissions, broker receipts and corrected outcomes
```

`trader/auto_trader_session.go`

```text
9a230d26da8194b8206e872f5daef7867c4e7c25 2026-09-02T22:46:11-05:00 fix(no-trade-band): one definition each for first-N and lunch — gate, grader and card read the same window (owner ruling 2026-09-02)
```

`trader/auto_trader_clock.go`

```text
0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb 2026-09-08T16:22:32-05:00 feat(research): link attempts, permissions, broker receipts and corrected outcomes
```

`kernel/calendar_blackout.go`

```text
0b28dc788d58c955845519596af6ffc06109111e 2026-09-02T22:46:11-05:00 feat(no-trade-band): read-time evaluation + drop the false clock-drift claim
```

`kernel/no_trade_band.go`

```text
6ce5bdc290e72922802d04067bfd7d5141d774f3 2026-09-03T00:33:49-05:00 fix(no-trade-band): close the seam — the instruction names no machine window at all
```

`trader/ninjatrader/tcp_trader.go`

```text
1fb3c21ecd3f7adb144cc9c40e108cc097349b09 2026-09-08T01:30:26-05:00 fix(signal-clock): the last bar-clock on an outgoing command, and class 89
```

`web/src/guide/content/plays.ts`

```text
7b1af151e2025ffe9d7bb2678e8d658a908be7ec 2026-09-05T22:45:55-05:00 docs(guide+checklist): the gate ORDER production runs, and class 77
```

`web/src/guide/content/planCard.ts`

```text
c40bb45a6e47c796810441cbf15069f2bee19e1e 2026-09-08T16:43:32-05:00 docs(scenario-economics): state legacy UNKNOWN intent and superseding C5 correction
```

`web/src/guide/content/levels.ts`

```text
4d159022c114983ad1b58aae1b73314afbf9f418 2026-09-02T12:30:48-05:00 docs(bars): class-45 report — measured cache depth, the Fri→Thu weekly finding that reshaped the ladder, per-TF retention trap, consumers re-pointed (exactly one), F1 red/green, storage projection, export, A15
```

`web/src/guide/content/settings.ts`

```text
09e110b5a5c62c76308471de12aeb0eaa21b9f5e 2026-09-03T22:39:03-05:00 fix(guide): the drift banner could never have been right, plus the label rules
```

`docs/superpowers/reports/2026-09-07-planner-preparation-audit/README.md`

```text
6095ca58fe5901ba398be374e4f9d3488d0bed6b 2026-09-07T23:09:02-05:00 docs: record owner approval for public audit evidence publication
```

## Validation and publication record

Validation commands and artifact fingerprints: [validate.py](2026-09-08-the-strategy-data/validate.py) and [validation.json](2026-09-08-the-strategy-data/validation.json).

Offline validation checked complete scenario totals, all condition/session/day-type marginals, unique scenario keys, arm and position joins, geometry signs, duplicate accepted-risk receipts, missing-value denominators and artifact privacy. Product tests were not run: this is a documentation/evidence-only change and executes no trading code. The source snapshot and current runtime are deliberately distinguished from historical behavior. Final publication verification is recorded in the PR and owner closeout using a commit-pinned report URL and byte-size/hash comparison; no branch-path fetch is treated as pinned evidence.
