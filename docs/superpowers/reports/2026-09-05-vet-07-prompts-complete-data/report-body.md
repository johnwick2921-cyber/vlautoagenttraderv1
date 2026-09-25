# Section 7 — The prompts and the trading system, complete reassessment

## One-page summary

**Verdict: I would keep the planner as a supervised author of conditional trade plans, but I would not regard a shorter prompt, a valid plan, or fewer rejects as evidence of trading edge.** The current instructions mix an auction thesis, historical claims, order permissions and validator requirements. A junior can obey one paragraph and become unable to act under another. This is a trading-method problem: it obscures the distinction between a market view, an executable opportunity and a decision to sit out. My first-person judgments are **[I] analytical judgment, untested here**, not a claim to a professional trading biography.

My three biggest problems:

1. **An attractive thesis can have no executable expression.** Thirteen actual executor proposals were refused by strict mode while citing S1; S1 in `2026-09-03:ASIA` v5 was unarmed. The planner simultaneously says only arms can trade and offers the model path when an arm fails feasibility. The result is repeated debate about an unavailable entry, not a measured missed-profit opportunity. The executor also offers discretionary closes without disclosing the hold lock that can suppress them (§2.5). Evidence: `D/strict-refusals.json:1`, `D/plan-asia-v5.json:1`, `kernel/planner_prompt.go:730`, `:733`, `trader/entry_gate.go:186`.
2. **The instructions overstate what the evidence says.** The corrected reject condition has 14 wins in 31 trades, 45.2% (Wilson 95% 29.2–62.2), not a current 75% rate. The cited overnight study describes a whole-day break of either boundary, not failure of each touched level. Neither establishes an entry policy for this MNQ book. Evidence: `D/population.json:1`, §2 below.
3. **Repair can change the trade instead of repairing its expression.** Five no-confirming-close rejects lead to five pullback-arm rejects; another three late retries choose levels with insufficient displacement. Removing the arm or substituting a fade would alter the opportunity, not simply fix the document. The existing repair hint does not explain that distinction. Evidence: reject IDs 119–122 and 124–132 in `D/rejects-seven-days.csv:1`; `kernel/breakdown_continue.go:261`.

My three biggest opportunities:

1. Put the **thesis, observable trigger, legal order path, effective stop/target and sit-out condition** on one checklist. I would review whether the trade can actually be expressed before debating conviction [I].
2. Use the delivered **28-item, 2,414-token instruction appendix** as a reviewable compression baseline: 49.39% of the 4,888-token current-contract instruction/schema text with the same tokenizer. It retains the inconvenient policies and contradictions. It is not applied; policy corrections are listed separately.
3. Judge future prompt changes on **valid executable plans available before their opportunities, unchanged risk, and forward outcomes by session day**. Lower token use and fewer rejection rows are operational improvements, not a substitute for positive expectancy [I].

The independently recomputed primary population is **58 closed eligible trades, −$466.428572, 18 wins / 38 losses / 2 flats, 12 CME days beginning at 17:00 CT**. There are **zero eligible entries after the September 3 11:10:33 CT strict-enforcement boot**. I make no current-strict profitability claim. `trade_excursions` has zero rows; position MAE/MFE are proxies and are not used to infer initial risk, optimal targets or a new stop. `D/population.json:1`, `D/eligible-positions.csv:1`.

## Evidence, scope and corrections to my previous report

I own only Section 7. This replaces the previous report and its Appendix A in full. Older `2026-09-05-vet-07-prompts-data/` files are historical, superseded evidence, not primary results for this revision. In particular I withdraw the unsupported approximate rewrite count, deletion of FVG policy from a supposedly equivalent rewrite, weakened three-per-side mandate, retained capitalized guards, generic immediate-entry/fade repair recommendation, and any implication that those changes were applied.

Worktree `/home/hoang/nofx-vet-07-complete`; branch `docs/vet-07-0905-complete`; session `codex-vet-07-complete-0905`; base `b4376246c2c502ecedd119c6a44a27956ed2f616`. Parent owns integration into dev; this worktree remains until integration. Scratch is exclusively `/home/hoang/nofx-analysis/vet-07-complete-0905`. Only this report and its own new evidence directory are changed.

Here **D** means `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data`. **P** means `D/planner-132-actual.txt`; **C** means `D/planner-132-current-contract-replay.txt`; **X** means `D/executor-37768-system_prompt.txt`; **Y** means `D/executor-37768-input_prompt.txt`. All `path:line` source references refer to the pinned base unless identified as actual stored prompts. [T] denotes measured own-tape evidence; [R] identifies a checked external source with transfer limits; [I] is my untested analytical judgment. Statements about code/text are direct inspection, not market research.

Fresh read-only capture: **September 5, 17:31:40 CT**. SQLite URI `file:/home/hoang/nofx/data/data.db?mode=ro`, `PRAGMA query_only=ON`, read transaction, no schema initialization. GET `/api/health` returned 200, revision `36648655cfe0`; authenticated GET `/api/config/resolved?…&session=NY` returned 200. Authentication was signed only in memory from the existing local configuration; no `cmd/gate-jwt`, `store.New`, credential output or credential file was used. `D/capture.py:1`, `D/capture-meta.json:1`, `D/api-health.json:1`, `D/api-resolved.json:1`.

Spec freshness at the pinned base: SYSTEM-MAP `a96224dd`, 2026-09-04 09:07:37−05:00; AUDIT-CHECKLIST `15340faa`, 2026-09-04 13:22:07−05:00. The renewed user instruction to cut from origin/dev governs over PART 2 R9's running-revision wording. There is no deployment in this section.

The eligible query uses `entry_time >= 1786770000000`, CLOSED, non-null `pnl_corrected`, excludes `source=e7_farside_test` and `plan_id=UNRESOLVABLE`. The latter excludes IDs **530,539,545,546,566,571,580**, in addition to test **572–574** and null-corrected **576,577,579**. The full query, included IDs, condition joins and session days are preserved. A 65-row population is invalid for the primary book. No mutable arm stop or exit distance is used as initial risk. I make no level retirement/flip proposal from raw touch counts, and no broker-slippage claim about position 591.

## 1. Rendered prompts and measurements

### Actual reads, not fixtures

The planner artifact is the **exact stored string** from `planner_rejected_prompts.id=132`, NY attempt 3, created September 4 **10:34:21 CT**, with an input clock of **10:28 CT**. The executor artifacts are exact system and input strings from `decision_records.id=37768`, input clock **13:27 CT**, row timestamp **13:28:01 CT**. A stored row timestamp is not the time every included bar became available. P has data through different horizons; Y explicitly says its latest 5m close is 4,068 seconds old (`Y:98`). I did not initiate a planner/executor call to obtain these artifacts.

Planner row 132 predates the running revision. C is an explicitly labelled **textual current-contract replay on those same historical facts**, not a newly executed prompt: `measure.py` changes only the old armable/non-armable schema lists to the current derived lists, and the general arm floor from 1.0 to 1.5. The remaining waterfall 1.0 literal stays. The exact source diff and helper definitions are preserved in `D/source-evidence.txt:1`; `kernel/planner_prompt.go:697`, `:733`, `kernel/arm_kind.go:89`, `:103`, `kernel/armed.go:17`, `kernel/condition_status.go:104`. The original report missed the now-fixed derived schema list; I do **not** recommend that fix again.

**UNMEASURABLE:** an actual planner provider request generated by the current running revision. No such stored full request is available. Missing inputs are the next real request bytes plus its exact revision and provider usage record. The replay is useful for current text review but does not fill that gap.

### Boundary definition and results

Facts means *supplied context by provenance*, not verified truth: numerical observations, bars, indicators, prior-plan content, diary text and machine statuses. Instruction includes all demands, static market priors, repair directions and operational promises; schema is separately counted but included in the instruction-half rewrite denominator. Mixed clock/floor/calendar lines are split. The embedded premium veto is retained with its computed context and also mapped as a constraint. A prior plan's instruction strings count as supplied context because the executor receives them as a prior document. This convention is disclosed, not a claim that those strings have no instructional influence.

`D/prompt-boundaries.csv:1` assigns every character a category and original line number. `D/measure.py:1` tokenizes each concatenated category with the same encoding. A few boundary tokens differ from whole-string tokenization; percentages use the sum of category counts, never an unexplained rounded total. The previous 57.5% planner-facts share classified entire mixed sections as facts; the new finer boundary gives **55.36% facts / 44.64% instruction-plus-schema**. That is a boundary correction, not a changed market read.

| Actual payload | Characters | o200k tokens | Facts tokens | Instruction tokens | Schema tokens | Instruction + schema share |
|---|---:|---:|---:|---:|---:|---:|
| Planner 132 | 31,595 | 10,948 | 6,063 | 4,049 | 839 | 44.64% |
| Executor 37768 system | 11,120 | 3,318 | 1,778 | 1,421 | 120 | 46.43% |
| Executor 37768 input | 16,447 | 8,504 | 8,440 | 64 | 0 | 0.75% |
| Executor both messages | 27,567 | 11,822 | 10,218 | 1,485 | 120 | 13.58% |

These are deterministic text shares, not binomial rates; Wilson intervals do not apply. The old export had an extra trailing newline per file, explaining its +1 character counts. Token totals reproduce. DeepSeek's [token-usage documentation](https://api-docs.deepseek.com/quick_start/token_usage/) identifies returned API usage as authoritative. **These are fallback BPE counts, not DeepSeek billing estimates guaranteed accurate for this model.** The exact provider tokenizer/model revision and reliably joined usage for these payloads are missing; I withdraw extrapolated daily cost claims derived from a completion-length heuristic.

| Actual payload | Uppercase MUST / NEVER / SHOULD | Any-case must / never / should | Paragraphs | Mean / maximum heuristic sentences per paragraph |
|---|---|---|---:|---|
| Planner 132 | 12 / 8 / 5 | 18 / 25 / 6 | 43 | 2.674 / 52 |
| Executor system | 3 / 1 / 0 | 8 / 5 / 1 | 30 | 1.800 / 6 |
| Executor input | 0 / 0 / 0 | 0 / 2 / 0 | 39 | 1.333 / 11 |

The **Rules paragraph alone** (`P:271`, current source `kernel/planner_prompt.go:704`) is **9,951 characters, 1,623 whitespace words, 2,480 o200k tokens, 52 heuristic sentences**. It contains uppercase MUST/NEVER/SHOULD **8/2/4**. The splitter is explicitly a punctuation heuristic: it splits “e.g.” and merges some lowercase starts. It is not linguistic ground truth. Every paragraph's sentence count is in `D/measurements.json:1`, not only an average.

My junior-checklist verdict [I]: fewer words help only if the page distinguishes “I see a short thesis” from “I can place this particular short.” Compressing 2,480 tokens of conflicting Rules without resolving authority yields a shorter conflict. That is why Appendix A is a faithful baseline and §2's policy decisions remain separate.

## 2. Both prompts as a junior's checklist

### Unanswerable or underspecified demands

* P asks for daily/4h structure while those summaries are unavailable; fresh full-prompt IDs in `D/prompt-population-checks.json:1` show **47/47** missing both, Wilson **92.4–100%**. VIX is likewise unavailable in **47/47**. The daily-candle heading promises eight but row 132 contains two candles. I cannot prove from this that a model cannot reason directionally; I can prove that the requested historical inputs are absent. Missing: sufficient session-aligned closed bars and the actual VIX source, or instructions to abstain from those claims (`P:18`, `:163`, `:164`; daily block in P).
* The FVG demand is conditional, not an impossible unconditional instruction: **47/47** full prompts show an empty fresh list (Wilson **92.4–100%**). The original “demand was never satisfiable” wording overstates it—the empty-list branch tells the model to author none. Current source explicitly allows shadow scenarios to be authored/scored while barring their orders (`kernel/condition_status.go:8`). Deleting the whole FVG contract would remove research/scoring behavior. I withdraw that recommendation as a mere compression.
* Acceptance duration is named as `ACCEPT_HOLD_MIN`; the general minimum reward/risk is referenced symbolically; the waterfall floor conflicts with the resolved floor. A junior cannot safely invent values or choose which sentence is authoritative. `P:271`; `kernel/plan_confirm.go:225`; `kernel/min_sl.go:34`.
* Executor asks for “roughly 1.5-3x ATR” and “~15-50 points” without identifying the timeframe (`X:19`), while Y contains six ATRs. It asks for 50-point breakeven (`X:28`), while saved strategy says 40 and the exit mechanism is suspended (`D/saved-trading-settings.json:1`, `trader/exit_mechs_suspend.go:33`). These are three different stop-management instructions, not three independent trading signals.
* “Trade quality,” confidence 60, and 30–60-minute holding time are not definitions of expected value (`X:21`, `:29`, `:31`). I would demand observable setup and invalidation definitions and calibrated outcomes before treating any as an edge [I]. Exiting at a valid stop in ten minutes is not evidence of impatience.

### Contradictions, with the trading consequence

| Conflict | Evidence | My assessment |
|---|---|---|
| Every concrete bias-direction scenario must arm; if infeasible omit arm and let the AI path take it | `kernel/planner_prompt.go:730`, `:733`; `trader/entry_gate.go:186` | Under strict, an unarmed thesis has no decision-path entry. Fixing the sentence must not authorize a new market-entry path. |
| Schema mandates ≥3 levels each side; validator only requires nonzero each side | `kernel/planner_prompt.go:696`; `kernel/plan_doc.go:882` | This is an unnecessary prompt quota still present after the owner deleted the code quota. My prior appendix silently weakened it; this appendix preserves it and proposes removal separately. |
| “At least 3 points apart” and duplicate “merging”; validator rejects distance ≤3 | `kernel/planner_prompt.go:704`; `kernel/plan_doc.go:860` | Exact 3.0 is a boundary mismatch as well as a false merge promise. A junior needs >3.0 with explicit rejection, if owner accepts corrected wording. |
| Premium longs forbidden; all biases labels without mandate | `kernel/planner_prompt.go:162`, `:187`, `:619` | Ambiguous permission to trade a trend above the prior midpoint. The branch is not independently established as an entry filter. |
| General arm stop 1.5×5m ATR; immediate waterfall 1.0× | `kernel/planner_prompt.go:733`, `:752` | A model can design the reward around the wrong risk. The floor fix at :733 already shipped; only remaining inconsistency needs correction. |
| Strict blocks decision entries; executor advertises off-plan setups, open examples and assured retrace arms | `X:34`, `:90`, `:110`, `:112`; strict refusals in D | Proposal validity, machine confirmation and actual resting orders are different states. Thirteen refusals are measured wasted proposals, not demonstrated saved or missed profits. |
| Executor says reasoning then decision; later says decision first; example begins reasoning | `X:74`, `:81`, `:83` | Output priority itself conflicts. Keep one unambiguous order; do not attribute profitability to formatting. |
| Morning weighting 07:30–10:00 CT versus graded 08:30–11:00 CT | `kernel/planner_prompt.go:654`; `kernel/session_registry.go:111` | A junior can follow the instruction and receive a timing penalty. Owner must select intended advisory versus grading windows. This does not establish which window earns more. |

I also withdraw the old inference that “two closes beyond a level 430 points away” proves a corrupt per-level counter. Closes can be beyond a distant level without a touch. `X:149`–`:159` alone cannot establish the semantics or a bug; touch, closes-beyond and confirmation should be inspected separately.

### The trader-method assessment and research check

**[T] The session preference is unsupported as a ranking.** Corrected reject trades by stored `plan_session` are Asia **2/11 wins, 18.2% [5.1,47.7], −$234**; London **5/8, 62.5% [30.6,86.3], +$538.50**; NY **7/12, 58.3% [32.0,80.7], +$281.50**. Every ID is in `D/reject-condition-sessions.json:1`, derived from the 58-row population. These are plan-session labels, not independently classified exchange RTH intervals. All cells n<30: **no verdict on superior session or executable edge**. The prior report's API fallback could not establish this comparison; the direct versioned-plan join now can describe it, but still cannot validate a preference. Acceptance is **1/6, 16.7% [3.0,56.4], +$4.571428**; sweep_reclaim **1/6, 16.7% [3.0,56.4], −$207**. Neither is current-strict evidence (`D/population.json:1`).

**[R] The overnight claim uses the wrong event for the implied trading rule.** [TradingStats' original NQ study](https://tradingstats.net/overnight-high-low-breakout-strategy/) covers January 2015–December 2025, **2,827 days**. Its table says high-only 1,094, low-only 922, both 646, neither 165. Thus **2,662/2,827** days broke *at least one* boundary. It does not measure a particular entry's stop-out probability or this system's seated level at its formation time. The commercial publisher's NQ study is not an MNQ fill model; I have not independently reproduced its underlying bars. I would keep “overnight reference” as a description and require formation-safe, costed MNQ opportunities before a fade/break policy [I]. Source-count Wilson calculation is in `D/research-checks.json:1`.

**[R] A null for a raw gap does not prove the chained setup profitable.** [MPM's original FVG research](https://mpmmarkets.com/research/does-the-fair-value-gap-strategy-work) tests roughly 40,000 occurrences across ES, NQ, GC and SI, seven years and three timeframes. Its honest-execution results do not establish a profitable raw-gap system after costs. That evidence does **not** demonstrate the prompt's specific sweep→displacement→gap chain on this MNQ strategy. I label that positive-chain inference [I], untested here. Exact source attribution for the prompt's “20–80pts,” “1h+ fill ~70–80%,” and unnamed structure-shift floor is **UNMEASURABLE**: missing paper/version, instrument, sample, event definition and cost model; primary-source searches did not identify matching support. I do not promote or retire a condition from those assertions.

I would treat bias as a compact hypothesis, a level as a reference, confirmation as an observation, and an arm as an executable commitment [I]. For each proposed play I would ask: what must happen, what would disprove it, can the legal order capture that event, and is its target reachable after the actual stop floor and costs? This is the missing trader review. Strong language about “conviction” cannot answer those four questions.

### Exact text I would remove or replace — separate policy proposals, never applied

The complete quoted cut register is `D/policy-cuts.md:1`; it extracts the exact originals automatically, including entire multiline examples where applicable. **None of these cuts is smuggled into Appendix A.** My proposed cuts/replacements are:

1. Remove the unsupported weekday conviction and stale week-win claims; preserve observable context. No new weekday or condition veto [I].
2. Replace strict-incompatible model-entry escape hatches with an explicit executable/unexecutable outcome. Do not switch an infeasible continuation to a fade merely to pass validation [I].
3. Remove the obsolete three-per-side quota, false merge promise and stale stop-floor literal; replace with actual current validator constraints. This changes the instruction contract and needs its own reviewed implementation.
4. Replace the executor's off-plan permission, open example and “arm already covers it” promise with mode- and arm-state-specific text. Retain actual position management permissions as determined by the engine.
5. Replace the generic ATR/range sentence and the 50-point breakeven instruction with effective risk/exit state; remove 30–60-minute holding and “impulsive” language as a reason to disregard a valid stop [I].
6. Delete the earlier reasoning-first instruction/example or reorder it to the decision-first output contract.

I do not propose deleting raw bars, the entire indicator history, or all FVG instructions without an ablation. Those are input/policy removals, not semantically preserving compression. I would compare unchanged facts and guard outcomes before choosing a smaller factual payload [I].

## 3. Last seven days of rejects, rule by rule

The exact interval is **[2026-08-29 00:00 CT, 2026-09-05 00:00 CT)**, seven complete calendar days, selected by `created_at`, not `trade_date`. There are **64 recorded failed attempts, IDs 69–132**, dated September 1–4 within that interval: **47 full prompts and 17 repair prompts**. Six rows are transport failures, leaving 58 content failures; that 58 is unrelated to the 58-trade primary population.

This is a census of the retained rejection table, not proof of every model request made during the week. There are **95 plan-version records** written in the same timestamp interval, including **14 `planner_fail_closed`** and mechanical lifecycle copies (`D/plans-seven-days.json:1`). The previous 254 was an all-store plan total, not seven-day authored plans. **UNMEASURABLE:** overall rejection probability per attempted read/call. Missing: a complete request ledger with call/read IDs, successful attempt rows, transport outcomes and interrupted-read records. `64/(64+95)` would be an invalid rate.

Each percentage below is a share of **64 recorded failed attempts**, with nominal Wilson 95% limits. Retry rows are dependent within reads, so these intervals describe binomial sensitivity, not independent-session uncertainty. All rows have exactly one observed returned error; later validators could have found additional defects. The mapping says which sentences govern or conflict with the refusal, not that a sentence is experimentally proven to cause it.

<!-- REJECT_TABLE -->

The current code still has instances of the dispatch's “class-45 pattern”; the checklist now names this withheld-validator family at **class 50** (`AUDIT-CHECKLIST.md:843`), while class 45 is the pantry/bar-layer issue. I retain the question's meaning, not its obsolete numbering.

* **Gap trigger reachability** is absent from the Rules paragraph: directional presence alone does not satisfy `kernel/plan_doc.go:905`/`:910`. Rows **116,123** establish actual refusals. Both long and short mirror code was inspected. The exact future text must say short trigger ≤current price on gap-down; long trigger ≥current price on gap-up.
* **Maximum waterfall-level distance** 5×ATR5m (`kernel/breakdown_continue.go:74`) is not stated in the planner instruction. Row **95** establishes an actual refusal. A distant level may be a legitimate reference yet not an authorable waterfall trigger; make the distinction explicit.
* **Target proximity** 1.5×daily ATR, fallback 0.012×price (`kernel/plan_doc.go:914`) is missing as an explicit number. No corresponding row in the 64 recorded failures; this is a code/text gap, not an observed loss.
* **Duplicate boundary** rejects distance ≤3.0 whereas wording permits exactly 3 and promises merging (`kernel/plan_doc.go:860`). No corresponding failure in this seven-day set; row 97 is the separate 13-versus-12 cap failure.
* **The repair menu is incomplete for strict trading.** `kernel/breakdown_continue.go:261` suggests immediate; the arm check then requires pullback. `kernel/planner_repair.go:53` has no matching specific branch for the no-close/pullback/displacement combination; row **131** contains the generic fallback. The displacement display (`kernel/displacement_feeds_forward.go:111`) calls an up-break “authorable” without naming the condition/direction; rows **119,121,124,127,130** are short requests at 29481.50. These are observed conflicts, not evidence that a different condition would win.

For September 4 specifically, IDs **117–132**, **14/16** belong to minimum-displacement/no-close/pullback-arm families: **87.5% [64.0,96.5]**. The no-close→pullback pairs are **119→120,121→122,124→125,127→128,130→131**. IDs **126,129,132** then fail displacement. I do not call all five episodes completed 3/3 failures: the 121→122 London episode was interrupted by a boot (`D/runtime-log-evidence.txt:1`). In the retained attempt-1 population, **0/30** carries earlier-read corrections (Wilson **0–11.4%**); this supports a missing carry-forward context claim, not an unconditional demand to reuse stale corrections forever (`D/prompt-population-checks.json:1`).

Already fixed and not re-recommended: derived reclaim arm lists (`kernel/arm_kind.go:89`); general planner floor from resolver (`kernel/planner_prompt.go:733`); void/floor facts fed forward (`P:103`, `:108`, `:111`); cumulative current-read corrections at top/tail (`P:1`, `:274`). No observed post-fix void rejection is not proof of a future zero-reject rate. The next qualifying read remains the evidence gate.

## 4. Actual half-length rewrite and semantic coverage

Appendix A is the **actual full rewrite**, also preserved verbatim as `D/appendix-rewrite.txt:1`. It is not a placeholder, an approximate token promise or `{schema unchanged}`. Its numbered schema rules contain the fields, enums, conditional objects, bounds, split-arm shape and vocabulary distinctions. It includes the conditional warming and nearest-1h-zone instructions from current source even though neither rendered in row 132; their original length is **not** added to the denominator.

| Same-tokenizer comparison | Original instruction/schema text | Actual rewrite | Remaining length | Half-length requirement |
|---|---:|---:|---:|---|
| tiktoken o200k_base | 4,888 | 2,414 | 49.39% | Pass |
| tiktoken cl100k_base | 4,894 | 2,427 | 49.59% | Pass |

These numbers compare concatenated original instruction/schema spans from C against every character of the delivered numbered rewrite. Facts are preserved separately in C and `prompt-boundaries.csv`. This is an **instruction-half compression**, not a 50% reduction of the entire 10,948-token planner message. `D/measurements.json:1` records counts and hash; `D/measure.py:1` reproduces them. There are 28 numbered lines and zero all-capital words, including zero capitalized guard slogans. Single-letter machine enum values A/B/C and S1-style IDs retain required spelling.

**120 mapped units** cover all instruction/schema spans, all 52 heuristic Rules sentences, the computed premium prohibition, and the two extra conditional source branches (`D/constraint-map.md:1`, `D/constraint-map.json:1`). Composite units map all their subconstraints, e.g. Rules-16's seven condition/confirmation pairs to rewrite line 16, and schema L264's fields to lines 14/15/17/18/21/22/24. The mapping preserves each original constraint; repeated prose is consolidated. No instruction is removed to manufacture the reduction.

The scope of semantic equivalence is **constraint content**, not identical model behavior. Preserving contradictory constraints is possible; proving that a stochastic model resolves them identically is not. **UNMEASURABLE:** behavioral equivalence, acceptance-rate improvement, latency improvement and profitability of this rewrite. Missing: paired offline model outputs at pinned model/settings, validator replay for both long/short cases and unchanged facts, followed by forward SIM observations. No paid/production model call or validator mutation was made.

I would not hand a junior the contradictory text as a live runbook without the separate rulings in §2. This is the compliant preservation baseline the original task asked for. The existing exact-fragment guards (`kernel/prompt_contract.go:24`) will not accept wholesale rewording/case changes; I make no claim they passed. A future implementation must replace literal checks with equivalent constraint tests and review every proposed policy correction separately. No guard was changed or bypassed here.

## 5. Which laws protect money, which are hygiene, and what is missing

I assessed **PART 2 and PART 3 themselves**, not only an arbitrary count of “Law” sentences elsewhere. Hygiene is useful; I do not relabel all code hygiene as a demonstrated profitable rule. No controlled P&L attribution exists for these laws.

| Law | My classification and trading purpose | Evidence |
|---|---|---|
| PART 2 R1 fresh evidence; R2 independent math; R7 corrected P&L | Protect capital allocation indirectly: stop choosing a strategy or raising size from contaminated evidence. This run's 58-versus-65 correction shows why. | `AUDIT-CHECKLIST.md:1847`, `:1850`, `:1857`; `D/population.json:1` |
| PART 2 R3 long/short mirrors | Direct potential exposure protection: a correct short path does not excuse a broken long path. Saved P&L amount unmeasurable without counterfactual orders. | `AUDIT-CHECKLIST.md:1852`; gap mirrors `kernel/plan_doc.go:903` |
| PART 2 R4 file:line; R5 grades; R6 verdict grammar; R8 CT times | Evidence hygiene, with time conversion directly affecting entry/blackout timing. They make a ruling auditable; not an edge. | `AUDIT-CHECKLIST.md:1853` |
| PART 2 R9 read-only isolation | Direct protection against an audit accidentally altering risk, orders or live state. Origin/dev base exception is explicitly authorized for this dispatch. | `AUDIT-CHECKLIST.md:1861` |
| PART 3 step 0 claim; step 1 ownership/tree gate | Coordination hygiene with a concrete risk consequence: avoid overlapping work and unintended runtime replacement. | `AUDIT-CHECKLIST.md:1869`, `:1950` |
| PART 3 steps 2–3 build/marker | Integrity hygiene: know which decision system is running. | `AUDIT-CHECKLIST.md:1959` |
| PART 3 step 4 five-leg flat gate | Direct operational exposure protection: database, API, broker positions, working orders and in-flight planner work must be reconciled. No single “flat” number suffices. | `AUDIT-CHECKLIST.md:1965` |
| PART 3 step 5 owner acknowledgment/override sweep | Direct operational protection when someone accepts deployment risk; explicit order handling is essential. | `AUDIT-CHECKLIST.md:1977` |
| PART 3 steps 6–7 verified swap/boot/rollback | Recovery discipline that limits unknown-state exposure; no deploy is authorized or performed in this report. | `AUDIT-CHECKLIST.md:1983` |

From a trader's view I would add three **review requirements** [I], not new strategy knobs: (1) sign off the **expressibility** of each bias-direction opportunity—legal path and feasible order geometry, not only directional narrative; (2) freeze the **as-of evidence universe and initial order risk** so a later rewritten plan or moved stop cannot explain yesterday's trade; (3) keep a **change ledger with predeclared success/kill metrics**, scored by independent session days and net costs. The exact inputs currently missing for (2) include immutable accepted initial stop/target, complete fill path and usable excursion observations. I would not replace them with mutable arm or ledger-exit distances.

The checklist already has the Stage-A one-contract evidence gate (`AUDIT-CHECKLIST.md:646`) and owner-controlled cutover framework. I do not re-propose them as new discoveries. The question here is whether prompt and settings language tells the trader the truth about those existing rules.

## 6. Settings: daily, weekly, never, and controls that should not appear as live

The fresh endpoint reports **167 classified settings: 144 live, 7 ineffective, 16 candidate-unverified**, schema 57; it reports zero suspended/advisory/display-only/infra entries and `env_shadows=0`. This is a registry inventory, **not** proof that 144 knobs alter this MNQ runtime. Its three trader-context resolved rows are **minimum reward/risk 2**, **plan mode strict**, **HTF veto true from shipped default**. `D/api-resolved.json:1`; endpoint builds those rows through the actual resolvers at `api/config_resolved.go:83`.

Saved trading settings: maximum levels 12; scenario cap 5; replans 4; realigns 10; min confidence 60; daily-loss amount $450 with `daily_loss_enabled=false` and `guardrails_enabled=false`; trade-cap amount 3 with `max_daily_trades_enabled=false`; breakeven true at 40 points and trailing true; last-entry 13:00 and EOD-flat 14:45. These are **saved values**, not proof of effect (`D/saved-trading-settings.json:1`). The captured boot remains older than b4376246, so current-source daily-entry-leg wiring is not claimed active merely because it is on dev.

| Cadence | What I would permit/review [I] | What I would not infer or change from one day's tape | Metric |
|---|---|---|---|
| Daily, before the session | Review plan approval/session participation, calendar exclusions, feed/broker health and chosen thesis; inspect effective daily risk and clock state. Any participation/veto change remains an owner decision. | Do not “fix” yesterday by changing stop floor, reward/risk, confidence, condition shadow status or order type. | Plans available before their opportunity; count of unresolved order/risk states; approved session participation. |
| Weekly, deliberate review | Examine candidate input usefulness, scenario/level/replan caps and repair burden. Propose one isolated change only with frozen comparison data. | A smaller prompt or fewer plans does not imply fewer missed profitable trades. No automatic tuning from n<30 cells. | Tokens/call; valid feasible plans/read; event-to-valid-plan latency; same-session-day net outcomes. |
| Never as routine tuning | One-contract Stage A, strict authority, effective stop-floor enforcement, schema enums, tick geometry, clocks, broker binding and suspension safeguards. | Do not loosen a loss limit mid-session, enable trails because prose requests them, or turn on a shadow condition to satisfy its demand block. | Breaches and unexpected order actions; changes require a separately reviewed owner ruling. |
| Should not appear as active discretionary controls | Hide/disable irrelevant crypto funding, coin-ranking/leverage controls for this instrument; show saved and effective exit/clock state separately. | Do not delete stored fields or candidate-unverified settings on a field grep alone. | Visible controls with independently verified MNQ consumers; zero contradictory saved/effective labels. |

Specific corrections: `last_entry_ct` and `eod_flat_ct` are registry **ineffective**, while the session clock has its own rules. Breakeven/trail are mechanism-suspended even though saved booleans are true (`trader/exit_mechs_suspend.go:33`); the endpoint's suspended count of zero does not reflect that state. `htf_veto` is still registry **candidate-unverified** while the same endpoint resolves it to true; that is a classification discrepancy, not proof of no consumer. Candidate-unverified means unresolved method-level tracing, not dead. The endpoint's `env_shadows=0` also does not prove absence of all environment behavior. I would display effective risk fields before proposing more daily knobs [I].

The new daily gate on dev explicitly says **entries blocked, existing positions not closed by this leg** (`trader/entry_gate.go:171`). I would not describe a $450 field as a guaranteed maximum daily loss or a guaranteed flatten. Section 6 owns the risk framework; Section 7's ruling is that the prompt/settings must state the actual effect.

## Ordered recommendations — unapplied

| Priority | What | Why / evidence label | Implementation category | Number I would watch / evidence gate |
|---|---|---|---|---|
| 1 | Render scenario confirmation, actual arm state, legal entry/management permissions and effective risk together; strict-incompatible escape hatches become explicit “not executable.” | [T] 13 strict refusals citing unarmed S1; [I] expressibility before conviction. | Prompt + code; owner ruling only for actual authority changes. | Decision-path open proposals under strict: target 0; valid executable plans at opportunity time. No profit claim until forward sample exists. |
| 2 | Repair the same thesis, or explicitly decline it; give direction-specific displacement and the joint arm/confirmation rule. | [T] IDs 119–132; changing continuation to fade is a strategy substitution. | Prompt + code, data first for an alternative condition. | No-close→pullback loops/read; scheduled read fail-closures; include interrupted reads in ledger. |
| 3 | Remove stale statistical/weekday authority, and distinguish research reference from entry evidence. | [T] corrected n31 reject and n<30 session cells; [R] overnight event mismatch and gap-source transfer limits. | Prompt + owner ruling for policy removal; data first for replacement edge. | Every market assertion has event definition, source/as-of, n and uncertainty; forward net outcomes by day. |
| 4 | Use Appendix A as compression baseline; separately review §2's policy cuts and explicit withheld constraints. | Direct 2,414/4,888 measurement; [I] concise checklist helps review but is not an edge claim. | Prompt + code for guard parity; not applied. | ≥50% instruction reduction; zero unmapped constraints; paired validator behavior, both directions; no relaxed risk. |
| 5 | Separate saved values, resolved values and actual mechanism permissions on the prompt/settings surface. | Direct API/strategy/exit-suspension discrepancies; current source versus running revision mismatch. | Code + guide; owner ruling for controls exposed or removed. | Zero controls represented as effective without a verified consumer; no unintentional setting changes. |
| 6 | Preserve actual request IDs, revision, provider usage, successful/failed/interrupted attempts and immutable initial order risk. | [T] 64 failed rows are not a read-attempt denominator; 32 read-facts rows have incomplete linkage; no excursions. | Code + data first. | Link completeness; exact call/reject/latency denominators; effective sample in days. P&L judged only after the strict-era forward population exists. |

## Requirement coverage and honest measurement limits

<!-- COVERAGE_TABLE -->

## Appendix A — numbered instruction-half rewrite, not applied

Read with the unchanged historical fact context in C. The content below preserves source instructions, including unsupported assertions and mutually inconsistent policies; §2, not this appendix, contains the proposed policy removals. No evidence is rewritten to make the system look profitable. The only current-source changes relative to the actual stored prompt are recorded in §1. This artifact is ready for review, not installed in the trading loop.

```text
<!-- REWRITE -->
```

## Reproduction and custody

`D/README.md:1` gives the exact commands. The three scripts, raw selected prompts, all rejection IDs, explicit boundaries, rewrite, map, corrected positions and fresh API snapshots are preserved. No runtime tests were needed for docs-only changes. Artifact checks verify both tokenizers' half-length limits, all 28 numbering prefixes, zero uppercase words, nonempty mappings, all 64 rejection IDs classified, the 58-trade/sum/day invariants, and allowed changed paths. They do not claim an LLM parity test.

I leave merge and any site health check after integration to the parent. Fresh read-only health/config GETs returned 200 during this audit. I have neither removed my worktree nor modified a trading process, order, prompt, database, configuration or main-tree lock.
