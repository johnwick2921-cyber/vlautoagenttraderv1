# VL — THE MASTER AUDIT CHECKLIST
*Every category of defect this system has actually produced. Any audit that doesn't check all of these is incomplete.*
*v1 · 2026-08-19 · built from the bugs found 08-14 → 08-19*

---

## HOW TO USE THIS

Give this file to any auditing agent as the spec. Every section must return a verdict with a **file:line receipt or a live query result** — never "looks fine", never "the prior report says so".

**The standard of proof, in one line:** *a passing unit test proves a function works; only production data proves the seam works.*

---

## 1 · UNITS, WINDOWS, ZONES AND SCOPE (the "correct code, wrong context" class)
*This class produced the worst bugs: the 33-hour death check, "2×5m" on 1-minute bars, levels burned on the whole cache, and the AI reading UTC clock numbers against CT windows.*

| # | Check | What "wrong" looks like |
|---|---|---|
| 1.1 | **Every rendered time is in ONE labelled zone** — prompt, card, alerts, digests, logs, settings | The model reasoning "12:12 UTC is inside the 12:00–13:30 lunch window" — a 5-hour self-block |
| 1.2 | Current-time line and every window bound share that zone | Mixed CT/UTC/ET inside one prompt |
| 1.3 | ET-anchored rules converted and displayed as CT (15:45 ET = 14:45 CT) | Two clocks disagreeing about when the session ends |
| 1.4 | DST correct across **both** annual transitions | Windows drifting an hour for weeks |
| 1.5 | **Every bar-interval assumption matches the configured rule** | "2×5m" counted on 1m bars = acceptance 5× too fast, at 7 sites |
| 1.6 | **Every lookback window is bounded and stated** — no calculation silently reading the whole cache | Death check judging a day plan against 33 hours; closes-beyond = 84 (cache) vs 11 (since birth) |
| 1.7 | **Every state has a birth timestamp and is judged only from it** | Levels marked consumed for events before they existed |
| 1.8 | **Every scope key is complete** (trader, session, date, symbol) | One trader's plan governing another's executor; a digest keyed without trader |
| 1.9 | Money units consistent everywhere ($/point, $/tick, contracts) | P&L right in one place, wrong in another |
| 1.10 | Percentages vs ratios vs multiples never conflated | An ATR multiplier read as a percentage |

---

## 2 · CONFIG AUTHORITY (the "shadows-config" class)
*A literal beating the owner's setting. Nine sites found; three caused live harm.*

| # | Check |
|---|---|
| 2.1 | For **every** user-visible setting: does the value the ENGINE used equal the value in the DB equal the value the UI shows? (three-way check, from a real cycle) |
| 2.2 | No hardcoded literal duplicates a config field anywhere — grep/AST sweep, verdict per hit: SHADOWS-CONFIG / SAFE-DEFAULT / TRUE-CONSTANT |
| 2.3 | No default applied in more than one file (name the single source of truth) |
| 2.4 | An unset value defaults in the **SAFE** direction, never the loosest (an unset R:R must not become 1.0) |
| 2.5 | No frontend constant gates behavior the backend also decides |
| 2.6 | Config survives **both** halves of any hand-rolled codec AND any FE normalize/field list |
| 2.7 | Contradictory fields cannot coexist (`sessions.enable` vs `sessions_enabled`) |
| 2.8 | A setting the UI offers cannot fail-close the system (max_levels 12 rejecting every plan) |
| 2.9 | Config-truth 4-step on every field: save → row JSON → reload → UI → **the enforcing path reads it** |

---

## 3 · WIRING (the "built but never connected" class)
*10 dead wires + 11 dead controls found. Everything passed its own unit test.*

| # | Check |
|---|---|
| 3.1 | Every function/endpoint/component has a **production caller** — sweep for zero-caller code |
| 3.2 | Every button: does the click reach the API, change server state, and **survive a hard reload**? |
| 3.3 | Every alert/event type has a real emit site (not just a definition) |
| 3.4 | Every producer has a consumer and every consumer has a producer (list orphans both ways) |
| 3.5 | Every stored value that the UI displays is actually written by something |
| 3.6 | Every displayed value has a real origin (nothing invented at render time) |
| 3.7 | The full chain works end to end in **production data**, not only in tests |

---

## 4 · TRUTH OF WHAT'S SHOWN (the "captured then discarded" class)
*Refused entries were recorded as successes; wait-reasoning existed in the row but rendered blank.*

| # | Check |
|---|---|
| 4.1 | An outcome recorded equals the outcome that happened (no success flag overwriting a refusal) |
| 4.2 | Reasoning captured is reasoning displayed |
| 4.3 | Counters shown to the owner are the counters the engine incremented |
| 4.4 | Nothing computed is silently dropped before the UI |
| 4.5 | Nothing displayed is stale beyond its meaning (a dot that outlived its condition) |
| 4.6 | Test/demo/seed data is absent from live statistics, equity and grades |

---

## 5 · DEPLOYMENT INTEGRITY
*Cost a full session of TRADING REFUSED, and produced three false "it's broken" reports.*

| # | Check |
|---|---|
| 5.1 | Running binary rev == HEAD == deploy/RELEASE, `vcs.modified` accounted for |
| 5.2 | Boot assertion actually refuses trading on mismatch (**prove it in a sandbox**, don't assume) |
| 5.3 | Deploy order enforced: pull → build → **then** write RELEASE → restart |
| 5.4 | Exactly one process; no orphan from a prior start |
| 5.5 | Goldens pass at boot, against the running binary |
| 5.6 | Any "bug" report is checked against the running rev before being believed |

---

## 6 · THE SAFETY SPINE
*Must be re-proven at every audit, from the current binary.*

| # | Check |
|---|---|
| 6.1 | Gate ORDER proven with a fixture per gate |
| 6.2 | Plan content can only ADD restriction, never bypass a gate |
| 6.3 | Owner overlays likewise |
| 6.4 | SIM lock: enumerate every account-selection site; no path to a non-SIM account |
| 6.5 | Hold-lock: the AI cannot widen a stop, cancel a bracket, or close by opinion |
| 6.6 | Mechanical exits still work when the brain is dead (bracket resting at the exchange) |
| 6.7 | Size caps master-independent |
| 6.8 | Every external dependency fails CLOSED (LLM, calendar, wire, bars, clock, disk) |
| 6.9 | Money math to the cent on a **real** closed trade |
| 6.10 | Which safety paths have **never fired live** — name them as unproven, don't claim them |

---

## 7 · THE MODEL'S INPUT AND OUTPUT
*The AI can only be as good as what it's told — and this is where the last three "why no trades" causes lived.*

| # | Check |
|---|---|
| 7.1 | Every claim in the prompt is TRUE (it says it read 4h — did it?) |
| 7.2 | One prompt = one price snapshot (no section 2 points older than another) |
| 7.3 | Instructions permit what the mode intends (advisory must inform, not forbid) |
| 7.4 | Numbers in the prompt match the numbers the gates enforce |
| 7.5 | Level/scenario states in the prompt are earned, not artifacts |
| 7.6 | Unparseable or truncated output is recovered, retried, and **its content preserved** |
| 7.7 | Latency fits inside the decision window; a decision whose bar closed is discarded |
| 7.8 | Every wait has a stated reason the owner can read |
| 7.9 | **The counterfactual**: how many waits had a viable setup available? (the number that separates "gates too strict" from "prompt suppressing") |

---

## 8 · DATA PROVENANCE
*Padded NT8 bars poisoned PDH/PDL and killed five plans before anyone looked at the bars themselves.*

| # | Check |
|---|---|
| 8.1 | No synthetic/placeholder/flat-filled bars in any store or cache |
| 8.2 | Levels derived from real bars — spot-check against an independent chart |
| 8.3 | Detector output matches an independent recomputation, tick-exact |
| 8.4 | Grades recomputed by hand match the displayed grade |
| 8.5 | If source data is unavailable, the system says so — it never computes from placeholders |
| 8.6 | Indicator math matches an external oracle (NT8's own values) |

---

## 9 · WHAT THE OWNER CAN SEE (answerability)
*If the answer isn't on a screen, the system will be debugged by accident.*

Can the owner answer, from the UI alone: today's plan · which play is armed · **why an entry was refused** · what he changed · what the AI proposed and what he did · why a plan died · how many decisions were lost · which levels are burned **and why** · what errors happened today and what they cost?

Name the screen for each, or record it as a GAP.

---

## 10 · MULTI-INSTANCE AND FRESH-INSTALL
*Because a partner will install this.*

| # | Check |
|---|---|
| 10.1 | Two traders cannot share plans, providers, configs or digests |
| 10.2 | No global singleton captures whichever instance started first |
| 10.3 | A fresh install gets the canonical timezone (CT) regardless of host locale, with no configuration step |
| 10.4 | A fresh install gets safe defaults, never the loosest |
| 10.5 | Migrations are additive and idempotent |

---

## 11 · THE AUDIT'S OWN HONESTY
*Two prior reports made claims that couldn't be reproduced.*

| # | Rule |
|---|---|
| 11.1 | Nothing is "verified" because a prior report said so — reproduce it or mark UNPROVEN |
| 11.2 | Where a prior claim can't be reproduced, **say which claim** |
| 11.3 | "Renders / compiles / a test passes" is never proof of a live path |
| 11.4 | State CANNOT PROVE + exactly what would prove it, rather than softening |
| 11.5 | Record the running rev at the start AND end (HEAD may move mid-audit) |
| 11.6 | A run that fixes as it goes cannot certify what it measured — separate the passes |
| 11.7 | Rank findings by **cost**: lost money → lost trade → lost clarity → cosmetic |

---

## THE FOUR QUESTIONS EVERY AUDIT MUST ANSWER

1. **Does every value the engine used equal the value the owner set?** (config authority)
2. **Does every number shown have a real, reachable origin?** (no invented values)
3. **Does every piece of built code have a live caller, proven in production data?** (no dead wires)
4. **Is every calculation using the right unit, window, interval, zone and scope?** (the class that hides best)

If an audit answers those four with receipts, it has done its job. If it skips one, that's where the next bug will be.

---

*Written after: 10 dead wires · 11 dead controls · 9 shadows-config sites · 5 scope errors · 3 deploy-integrity incidents · 2 timezone errors · 1 security chain · and roughly 40 hours of hunting them one at a time.*
