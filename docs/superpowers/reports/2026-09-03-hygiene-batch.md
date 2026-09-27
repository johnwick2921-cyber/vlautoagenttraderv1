# HYGIENE BATCH — the owed small fixes

Branch `fix/hygiene-0903`. No cutover of its own; rides the attribution boot
(14:45–16:30 CT). Highest occupied checklist class at merge: **53**.

## ITEM BY ITEM

| # | before → after | where |
|---|---|---|
| H1a | header "THE 26 BUG CLASSES" / "18 proven" → "THE BUG CLASSES" + a *highest occupied: 53* note | `AUDIT-CHECKLIST.md:10` |
| H1b | slots 27/28/29 empty → netting-orphan · canonical casing · silent-aggregate family | `AUDIT-CHECKLIST.md:239–265` |
| H2 | canon laws live only in reports → **NOT DONE ON THIS BRANCH** (see below) | `CLAUDE.md` |
| H3 | no bias alias → `NormalizeBiasDirection` at the chokepoint | `kernel/plan_doc.go:487` |
| H4 | `No-trade: …` → `No-trade NOTES (… NOT machine-enforced …)` | `kernel/plan_render.go:169` |
| H5 | `ARMED_TEST_SEAM=on` → off | **NOT DONE ON THIS BRANCH** (see below) |
| H6 | comment said the key excludes the trader id; code includes it | `trader/auto_trader_planner.go:812` |
| H7 | four guide corrections, two no-ops | `web/src/guide/content/*` |
| H8 | `VL_BUILD_ID "2026-08-30-e7"` → `"2026-09-03-hygiene"` | `ninjascript/VLTraderTCPClient.cs:51` |
| H9 | **nothing to do** — no duplicate exists | — |

## THE FOUR ITEMS WHERE THE DISPATCH WAS WRONG

**H9 — there is no duplicate "52".** PART 1 contains each of 50, 51, 52, 53
exactly once (`awk` over PART 1, `uniq -d` empty). 52 is the no-trade-band class
and 53 is void-parity. Nothing renumbered.

**H3 — `bearish→bear` would have MANUFACTURED rejects.** The day-plan validator
accepts `long|short|neutral` (`kernel/plan_doc.go:316 biasDirections`);
`bull|bear` is the **weekly** doc's vocabulary (`kernel/weekly_prompt.go:360`).
Aliasing a day-plan bias to "bear" produces a value the validator rejects. The
aliases therefore target the validator's own set: `bearish|bear|down|downside|
sell → short`, `bullish|bull|up|upside|buy → long`. **A store grep found ZERO
bias-spelling rejects to date**, so this is precautionary, not evidence-driven.

**H7a — the resolved value is 1.5, not 0.3 (guide) and not 1.0 (dispatch).** The
saved strategy config reads `day_plan.proximity_filter_atr = 1.5`, and
`kernel.ResolveProximityK` passes it through (clamp 0.1–3.0). The card claimed
"⭐ 0.3 — LIVE since 2026-08-28" and is now corrected to the resolved 1.5 with
the band described as `K × dATR`.

**H7d — REVERSED 2026-09-03 after a measurement error of my own.** I first read
`SELECT config FROM strategies LIMIT 1`, which returned the UNBOUND preset
`均衡策略` (rr=3), and reported the market-entry floor as 3.0. The trader binds
`traders.strategy_id = a5b7662e-…` — the strategy named **"MNQ"**, whose
`min_risk_reward_ratio` is **2** since the 2026-09-01 08:13 CT save. So BOTH
paths refuse below **2.0** (arm path: `armMinRR()` default 2.0), and the Guide
card is corrected to say so.

> **`SELECT … LIMIT 1` on a multi-row table is never a measurement.** The bound
> row is found by TRADER BINDING (`traders.strategy_id`), never by `is_active`,
> `is_default`, or row order. This is **checklist class 9 (is_active-vs-binding)**,
> which already names these exact two rows — 4104ca0a advisory vs the bound
> a5b7662e — and I walked into it anyway. The probe existed; I did not run it.

**H7e — already correct, no change made.**
`ResolveMaxContracts` ends in `ClampStageAContracts`, which caps at **1** since
0B, so "capacity=1" is true and needs no owner click.

**H7c was broader than stated.** All three session read times in the Guide were
the pre-2026-08-31 values. Corrected to the registry
(`kernel/session_registry.go:90/99/108`): ASIA 16:55→**16:30**, LONDON
01:55→**01:30**, NY 08:25→**08:00**.

## H2 AND H5 CANNOT RIDE A BRANCH

`CLAUDE.md` is **gitignored** (`.gitignore:8`) and `.env` is untracked. Both exist
only in the main tree working directory, so neither can be delivered by a branch
— and A2b forbids editing the main tree outside the lock. **Both are staged as
exact text in this report and applied under the lock at cutover**, immediately
before the boot that reads them. `.env` is read at boot, so H5 takes effect on
that same restart:

```
H5:  .env:37   ARMED_TEST_SEAM=on   →   ARMED_TEST_SEAM=off
```
Boot line before/after is quoted in the cutover report (expect `test_seam=OFF`).
The H2 canon-law block is prepared and applied to `CLAUDE.md` in the same step.

## WHAT MY OWN CHANGES BROKE, AND WHY THAT WAS USEFUL

**H3's first cut aliased `flat|sideways|range → neutral` and broke two tests.**
`"sideways"` is the invalid-direction sentinel in `plan_doc_test.go:51` and
`w4_overlay_executor_test.go:83`. Aliasing it made an invalid value valid and
silently weakened both. Rather than edit the tests, I narrowed the table: those
mappings are **semantic judgements, not spellings**, and `neutral` is already
writable directly. An alias table fixes spelling; it must never decide meaning.
Both fixtures pass untouched, and the pin now asserts `sideways|flat|range`
remain invalid.

**H4 changed a prompt golden**, correctly: `futures-plan` line 28. The golden was
updated deliberately and the boot self-check passes.

## TESTS

`TestH3BiasAliasesTargetTheValidatorsVocabulary` (every alias lands in
`biasDirections`; unknown and semantic spellings pass through and stay invalid;
the chokepoint applies it) · `TestH4NoTradeProseRendersAsNotes` (renders as
NOTES, says "NOT machine-enforced", the bare constraint label is gone).

Go **27 ok / 0 FAIL**, goldens PASS, vet clean, tsc clean, **vitest 39 files /
302 tests**.

## A15 — LIVE-SURFACE TRUTH

- **H8 does not change the running AddOn.** `VL_BUILD_ID` is a C# constant; the
  DLL in NT8 still reports `2026-08-30-e7` until the next NT8 restart (copy →
  F5 → full restart). No AddOn reload was performed by this wave.
- **H5 is not yet in effect.** The seam remains ON until the .env edit lands
  under the lock at cutover.
- **H6 is comment-only.** The claim key still includes the trader id, so two
  traders on the same session are NOT collapsed to one AI call. Whether they
  SHOULD be is left open, unchanged, and flagged here.
- **H3 is precautionary.** Zero observed bias-spelling rejects; the aliases may
  never fire.
