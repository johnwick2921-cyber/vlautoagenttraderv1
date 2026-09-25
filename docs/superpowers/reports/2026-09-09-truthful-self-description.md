# W5 — The system describes itself truthfully

**Wave:** fix/truthful-self-description · **claim:** `truthful-self-description-50f0837d/nofx-6b [a7075a]`
**Basis:** `docs/superpowers/reports/2026-09-08-the-strategy.md` @ `5519d4941c6742418f3124f38227d23893413f86` (65,178 bytes), section D4.
**Running rev when the wave started:** `954f11b15f2e`, measured two ways — `/api/health` → `954f11b15f2e`, and `/proc/438/exe` → `/home/hoang/nofx/nofx-bin` with `vcs.revision=954f11b15f2e7615678f7d2b708c47895faebf1e`, `vcs.modified=false`.

Strings and docs only. Not one line of behaviour.

## The headline

The wave was dispatched to fix wrong *numbers*. The largest thing it found is a wrong *mechanism*:

```
grep -cE 'InLunchNoTrade|InFirstNoTradeMinutes|sessionEntryBlocked' trader/armed_executor.go   →  0
the band's only production enforcement call site: trader/auto_trader_orders.go:281 (decision path)
bound strategy a5b7662e "MNQ": plan_mode = strict
```

Under `strict` a resting order is the only way into the market, and **nothing on the arm path reads the no-trade band**. The Guide said "no entries" and "Three sources, all of them enforcing". Both bands — lunch and first-N — refuse nothing in the live configuration. This is a CODE defect; per A24 the wave corrects the CLAIM and files the defect.

## Every D4 row, reproduced at the running rev

| Row | Verdict | Where it actually is |
| --- | --- | --- |
| C1 sideways oscillation | **REPRODUCED** | `data/data.db` → `strategies.config.ai_config.prompt_sections.entry_standards` (NOT source) |
| C2 2-4 trades/day, 30-60 min hold | **REPRODUCED** | same, `prompt_sections.trading_frequency` |
| C3 "50 point move → breakeven" | **REPRODUCED** | same; saved knob is 40 and BE/trail are suspended |
| C4 avoid restarting after close | **REPRODUCED** | same; `trader/auto_trader_loop.go` arms a post_exit cycle |
| C5 bias as one-way instruction | **REPRODUCED** | `kernel/planner_prompt.go:741` — an AUTHORING instruction, out of scope (A31) |
| C6 trend day = continuation-only | **REPRODUCED** | `kernel/planner_prompt.go:713` — same, out of scope |
| C7 quality "nothing gates on it" | **REPRODUCED** | `web/src/guide/content/planCard.ts:96` (+ `settings.ts:225`) |
| C8 lunch 11:30–13:30 ET | **REPRODUCED** | `web/src/guide/content/plays.ts:231` — 01's citation was **exact** |
| C9 "three sources, all enforcing" | **REPRODUCED** | `web/src/guide/content/planCard.ts:101` |
| C10 target-chain / runner | **REPRODUCED** | `web/src/guide/content/plays.ts:45`, `:221` |
| C11 grade "trustworthy" | **REPRODUCED** | `web/src/guide/content/levels.ts:12` |
| C12 BE/trail suspended | **REPRODUCED as TRUE** | `settings.ts:401,:417` — **left unchanged**, the code confirms it |

**Twelve of twelve reproduced. None had been quietly fixed.** One citation was off: C7's code line is `trader/armed_executor.go:1899-1904`, not `:1898`.

### SPEC-FRESHNESS note for the next reader of 01

01's D4 line numbers are **not** stale. `plays.ts:231` is exactly the sentence it names. An early draft of this report claimed otherwise; that was my error, not 01's — I read a ~700-character line through `cut -c1-165`, saw only its waterfall opening, and explained away a `grep` hit that had been correct. **Truncating output while auditing prose is how you miss the prose.** Re-pin 01 by all means, but its D4 citations held.

## Every string changed

| # | File:line | Before | After | Row |
| --- | --- | --- | --- | --- |
| 1 | `plays.ts:231` | `lunch 11:30–13:30 ET → no entries` | `lunch 12:00–13:30 CT — the window kernel.LunchWindowCT() resolves — refuses AI-decision entries only … with plan_mode=strict … it refuses nothing` | C8/C9 |
| 2 | `plays.ts:231` | `no pool swept by 10:30 ET` (×2) | `by 09:30 CT` — `ETtoCT("10:30")` = `09:30` | C8 |
| 3 | `planCard.ts:101` | `Three sources, all of them enforcing` | `Three sources, and they do NOT bind the same way …` + which path each binds | C9 |
| 4 | `planCard.ts:96` | `quality A+/A/B/C (INFORMATIONAL — nothing gates on it)` | judged against the `min_scenario_quality` floor `MinScenarioQualityFor` resolves | C7 |
| 5 | `settings.ts:225` | `Lowest grade … (INFORMATIONAL — nothing gates on it).` | same correction | C7 |
| 6 | `levels.ts:12` | `a grade says how trustworthy it is` | `a grade is a letter … a SEATING PRIORITY label, not a probability … no grade has ever been calibrated` | C11 |
| 7 | `plays.ts:45` | `T1 = first opposing pool; runner = the draw.` | + `T1 and "runner" are PLAN VOCABULARY, not an exit ladder … one bracket … nothing scales out` | C10 |
| 8 | `plays.ts:221` | `runner = the draw (…)` | + `vocabulary only: no partial exit exists, the whole position closes at once` | C10 |
| 9 | `guards.ts:116` | `no entries 12:00–13:30 CT` | + `— the window kernel.LunchWindowCT() resolves` | A11 |
| 10 | `settings.ts:569` | `12:00–13:30 CT (matches the lunch gate)` | + names the resolver | A11 |
| 11 | `planner_prompt.go` lunch line | `(hard-gated — entries inside it are refused)` | `(refused on the AI-decision path only — no band predicate exists in the arm path …)` | C8/C9 |
| 12 | `planner_prompt.go:643-647` | present-tense stale paragraph | prefixed `SUPERSEDED by faa3526f`, verbs moved to past tense | provenance |
| 13 | DB `prompt_sections.trading_frequency` | `2-4 trades per day`, `50 point move stop loss to breakeven`, `≥30-60 minutes` | marked GUIDANCE; states `guardrails_enabled=false`, `max_daily_trades_enabled=false`, BE/trail SUSPENDED | C2/C3 |
| 14 | DB `prompt_sections.entry_standards` | `sideways oscillation, or immediately restarting after closing positions` | states neither is a machine stand-aside, names `auto_trader_loop.go` | C1/C4 |

I corrected my own C8 rewrite once: the first version said "hard-gated: entries inside it are refused", which replaced a false claim with a different false claim. It was caught by C9's evidence before it shipped.

## Prompt instructions for mechanisms that do not exist (code-side findings, NOT changed)

Per A31/D4, these instruct the model what to author and are out of scope here:

1. `kernel/planner_prompt.go:741` — "ARMS FOLLOW THE BIAS … Every scenario in the plan's bias direction … MUST carry an arm." The bias-coherence warning exists (`BiasArmWarning`) but **warns and never gates**; under `strict` the entry gate's bias leg (`trader/entry_gate.go:203`) fires only `if in.PlanMode == "direction"`. What gates is Leg 2, scenario-direction consistency.
2. `kernel/planner_prompt.go:713` — "The scenario MIX must follow the regime + day_type." **No gate reads `day_type`.** Nothing makes a trend day continuation-only and nothing stands aside on one.

Note the Guide is already truthful about (1): `guards.ts:81` and the mode table at `guards.ts:200-214` correctly scope bias-refusal to `direction` and cited-scenario to `strict`. Left alone.

## A15 — what the owner will still read that is wrong after this boot

| What | Where | Wave that owns it |
| --- | --- | --- |
| The lunch and first-N bands do not reach the arm path | `trader/armed_executor.go` | **code wave** — the band should bind the arm path, or the Guide's "no entries" must stay qualified forever |
| Three unbound `New Strategy` presets still carry all four false sentences | `strategies` rows `a79ba71f`, `70695b25`, `08539e82` | **DB hygiene wave** — binding any of them resurrects every claim |
| `day_type` is authored, displayed, and read by no gate | prompt + API/UI | code wave |
| `GUIDE_BUILT_REV` cannot detect prose drift | `web/src/guide/` | guide-tooling wave |
| The 1.5×ATR5m stop floor, the 12-seat cap, the grading ladder, 14:45 flat | various | unvalidated `[I]` parameters — labelled as such where the Guide describes them |

## E2 — the no-behaviour pin

```
kernel/planner_prompt.go          | comment block + ONE description string
web/src/guide/content/guards.ts   | prose
web/src/guide/content/levels.ts   | prose
web/src/guide/content/planCard.ts | prose
web/src/guide/content/plays.ts    | prose
web/src/guide/content/settings.ts | prose
```
No gate, validator, executor, exit, level, wake, cadence, arm or order file is touched. Full Go suite at the merged head: **30 packages ok, 0 FAIL**, goldens included.

## E1 — the fixture, RED first

`kernel/guide_clock_contract_test.go`. The existing `TestNoTradeWindowsHaveNoSurfaceLiterals` forbids a hardcoded lunch window in **Go** source; the Guide — the surface the owner actually reads — was the one place it could drift unwatched, and it did.

The fixture is deliberately **not** "never type a clock": four Guide files typed the CORRECT window, and rewriting four true sentences to look busy is worse than changing nothing (Section H). `tradingDay.ts` already typed the window *and* named `kernel.LunchWindowCT`, so the rule is **never type it unsourced**.

RED:
```
plays.ts    states the lunch window as "11:30–13:30" — LunchWindowCT() resolves 12:00–13:30 CT
guards.ts   types the lunch window "12:00–13:30" without naming its resolver
planCard.ts types the lunch window "12:00-13:30" without naming its resolver
settings.ts types the lunch window "12:00–13:30" without naming its resolver
TestGuideStatesNoRawEasternClock: plays.ts states 3 raw Eastern clocks
```
GREEN after the corrections.

## Rollback

Prior binary preserved as `nofx-bin.old.<rev>` (named for the rev it holds, verified with `go version -m`). DB backed up before the config write to `~/nofx-backups/manual-w5-truthful/data.db` (762,155,008 bytes), taken with `sqlite3 .backup` against a read-only handle.

## E5 was owed and unrun; when run it refuted the fixture

The dispatch required E5 — "mutate the resolved value E1 reads; quote the line and the failure text" — and I did not run it. A sibling lane's class-97 warning ("one source, both readers — never two readers that happen to agree") prompted me to. It refuted my own pin:

```
-func LunchWindowCT() (startCT, endCT string) { return "12:00", "13:30" }
+func LunchWindowCT() (startCT, endCT string) { return "12:15", "13:45" }

go test ./kernel/ -run TestGuideDoesNotTypeTheLunchWindow   →   ok        ← should have FAILED
```

The pin built its forbidden-literal list FROM the resolver, so when the resolver moved, the Guide's stale `12:00–13:30` matched none of the new literals and the loop fell through. It could only ever confirm today's agreement — the same defect the wave was written to remove, inside the guard written to prevent it.

The shipped VALUE was never wrong (`12:00–13:30`, verified live), so nothing the owner read was false. What was missing was the guard against tomorrow.

Fixed on fix/guide-clock-drift-pin: the pin now finds every `HH:MM–HH:MM` range written near "lunch" and requires it to EQUAL the resolved pair. The same mutation now fails on three files at once:

```
plays.ts      states the lunch window as "12:00–13:30" but kernel.LunchWindowCT() resolves "12:15–13:45"
settings.ts   states the lunch window as "12:00–13:30" but …
tradingDay.ts states the lunch window as "12:00–13:30" but …
```

One more lesson from the fix: the first inversion used a ±160-character window and failed on tradingDay.ts's unrelated NY session range `08:30–14:45`. A guard calibrated by guesswork fails on correct text; the window was then calibrated against the real sentences. Checklist class 93, filed as the worked example for the inbound class 97.
