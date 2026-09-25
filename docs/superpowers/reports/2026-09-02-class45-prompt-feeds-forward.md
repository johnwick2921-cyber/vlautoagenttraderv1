# CLASS 45 — THE PROMPT FEEDS FORWARD

Wave: dispatch **class 45** · checklist slot **50** (slot 45 was already the
"pantry nobody could reach" class; 46 is left free to avoid a double meaning
with the class-46 instrument-honesty dispatch, which filed under 49).
Branch `fix/class45-prompt-feeds-forward` · worktree `/home/hoang/nofx-class45`.
Status: **BUILT, GREEN, STAGED — awaiting owner GO. Not deployed.**

## THE CLASS IN ONE SENTENCE

The planner prompt and the plan validator are two statements of the same law,
written in different places by different waves, and nobody diffed them — so the
prompt ordered plays the validator refuses, withheld a threshold the executor
enforces, and corrected the model about one mistake at a time.

---

## D1 — WHERE THE PROMPT AND THE VALIDATOR DISAGREED

| # | the prompt said | the enforcer does | site |
|---|---|---|---|
| 1 | "If price sits BELOW PDL you **MUST** write a **continuation short**" — an unconditional MUST naming a CONDITION | `BreakdownContinueState` **voids** a breakdown continuation once a close comes back across the level. On a reclaimed level the only compliant answer is a rejected one. | `kernel/planner_prompt.go:589` |
| 2 | nothing about void levels | the validator already knows, per level, whether the breakdown is void — it computes the verdict to reject with | `kernel/plan_doc.go` (`BreakdownContinueState`) |
| 3 | nothing about a minimum stop | since 0B the arm composer floors every stop at **1.5×ATR5m** and the **R:R gate then judges the widened stop** | `kernel/min_sl.go`, `trader/arm_stop_anchor.go` |
| 4 | gap-down plan doc: "MUST include a continuation short" | same contradiction as (1), in the day-plan document rules | `kernel/plan_doc.go` gap messages |
| 5 | the reject block named **the last defect only**, at the **tail** of a ~6,600-token prompt | the chain accumulates defects; the model is judged against all of them | `trader/auto_trader_planner.go:1244` |

**(1) is the load-bearing one.** The WATERFALL PLAY rule binds "continuation" to
`breakdown_continue`/`breakup_continue`, so when every breakdown level in reach
was already reclaimed, the prompt's MUST was **unsatisfiable** — obey it and be
rejected, disobey it and be rejected.

## D2 — WHAT IT COST, MEASURED

Read-only, `planner_rejected_prompts`, `trade_date='2026-09-02'`:

| class of reject | count |
|---|---|
| breakdown-void (`a close came back across … the breakdown is void`) | **8** |
| `fade_requires_touch` | 3 |
| other (unreachable retest, too many levels) | 2 |
| **total rejects, one day** | **13** |

Void share across two days: **09-01 6/23 · 09-02 8/13** — 14 of 36.

**The chain shape, twice in one day** (both 3-attempt chains ended this way):

```
92 | 01:32:54 CT | LONDON | att1 | S1 breakdown_continue: a close came back across 29021.25 — the breakdown is void
93 | 01:35:04 CT | LONDON | att2 | scenario[0].confirm.rule "1x5m_close" — fade_requires_touch
94 | 01:37:44 CT | LONDON | att3 | S2 breakdown_continue: a close came back across 29021.25 — the breakdown is void
```
```
98 | 14:22:33 CT | NY | att1 | S2 breakdown_continue: a close came back across 29167.66 — the breakdown is void
99 | 14:23:37 CT | NY | att2 | scenario[1].confirm.rule "1x5m_close" — fade_requires_touch
100 | 14:25:17 CT | NY | att3 | S3 breakdown_continue: a close came back across 29109.65 — the breakdown is void
```

Read it as a loop: **void → fade → void.** Attempt 3 was told about the fade and
nothing else, so it fixed the fade and walked straight back into the defect it
had been corrected about two attempts earlier. Both sessions lost the read. A
third chain (ASIA 23:38/23:39) shows the same first two steps before it ended.

This is why the owner's addition — *cumulative* distinct rejects, not the last
one — is the fix rather than a refinement.

## E — WHAT SHIPPED

**E1 · the MUST orders a DIRECTION, not a play** (`kernel/planner_prompt.go`)

before:
```
If price sits BELOW PDL you MUST write a continuation short; ABOVE PDH, a continuation long.
```
after:
```
If price sits BELOW PDL the plan MUST include a SHORT-direction scenario (ANY legal
condition — reject, breakdown_continue, acceptance, sweep_reclaim, hold, reclaim);
ABOVE PDH, a LONG-direction scenario. Pick the condition the TAPE supports: if a
breakdown level is listed as VOID above, author a different condition there.
```
The directional intent (be short below PDL) is preserved exactly; only the
condition lock is released.

**E1b · the same correction in the day-plan document rules** (`kernel/plan_doc.go`)
— gap-down/gap-up now demand a direction, via one shared `gapDownDirectionMessage`
so the message and the rule cannot drift.

**E2 · void levels are listed, decided by the validator's own code**
(`kernel/class45_feeds_forward.go`). `ComputeVoidBreakdownLevels` builds the
minimal `PlanScenario` and **calls `BreakdownContinueState`** — the single source
of the verdict. It is a level-oriented entry point over the same predicate, not a
copy. Rendered:

```
## VOID breakdown levels (a close came back across since the break — the write-site validator REFUSES a waterfall play at these)
29021.25 breakdown (reclaimed 01:14 CT) · 29058.75 breakdown (reclaimed 12:41 CT)
- do NOT author breakdown_continue or breakup_continue at these prices. Any other condition is legal there.
```

**E3 · the stop floor is stated with its live reading**

```
## Minimum stop distance this cycle
39.0 pts (1.5×ATR5m 26.02, resolved). Stops tighter than this are WIDENED by the
executor before the R:R gate sees them — author stops AND targets consistent with
it, or your R:R will not survive the widening.
```

**E4 · cumulative distinct rejects, at the TOP and the TAIL**
(`trader/auto_trader_planner.go`). `addDistinctReject` accumulates every distinct
defect of the read in first-seen order; `plannerRejectHeader` leads the prompt
("The standing rules below still apply EXCEPT where this correction overrides
them.") and `plannerRejectTail` closes it. Recorded at **all eight** reject
sites. The attempt line now reads `reauthor+block(top+tail, N distinct)`. The
legacy single-defect tail survives only as a degrade path when the history is
somehow empty, so behaviour falls back to the old shape rather than to nothing.

**Token cost, measured, not assumed:** header ≈131 tok · tail ≈105 tok ·
**≈236 tok added to a ~6,600-tok prompt** (≈3.6%).

**E5** · two rows in `PromptContracts()` so the pairing is enumerable.
**E6** · boot line, ordered by the owner's naming: `📜 prompt feeds forward:` →
`⏱ wakes:` (class 47, separate lane) → `🚫 no-chase:`.

```
boot: prompt feeds forward: void-levels=n/a (computed per read) · stop-floor=1.5×ATR5m (n/a — no ATR yet) · reject-block=top+tail (class 45)
live: prompt feeds forward: void-levels=2 · stop-floor=1.5×ATR5m=39.0pts · reject-block=top+tail (class 45)
```

**Boot-line honesty (self-caught, checklist class 49).** The first cut printed
`void-levels=0` at boot, which reads as *"zero levels are void"* when the truth
is *"not computed yet — there are no bars"*. A negative count now means
uncomputed and prints `n/a`; `0` means measured-and-empty. The floor MULTIPLIER
is known at boot even when ATR is not, so it is always stated. Caught by writing
`TestClass45BootLineDoesNotFakeAMeasurement` **red first**.

## F — TESTS

| id | what it pins | result |
|---|---|---|
| **F1** | E1: the prompt must not order a CONDITION | **RED first** on pre-45 surfaces, then green |
| **F2** | **parity**: the rendered void list agrees with `BreakdownContinueState` | **40/40 checks across 20 tapes** |
| **F3** | the floor line matches the resolver (`MinSLATRMult`), not a literal | green |
| **F4** | the re-author carries the cumulative distinct rejects at **top AND tail**, defect stated **exactly twice**, corrections **lead** the playbook; token delta logged | green |
| **F5** | replay pin of the LONDON 01:32 chain | green |
| — | boot line does not fake a measurement | **RED first**, then green |
| — | assembly source pin: ≥6 reject sites record history | green |

F1's red, quoted:
```
class45_pin_test.go:76: E1: the prompt still orders a CONDITION ("If price sits BELOW PDL you MUST write a continuation short")…
```
F2's parity line:
```
class45_feeds_forward_test.go:88: parity: 40/40 checks agree across 20 tapes
```

**Three existing pins were updated to the new canon, none relaxed:**

- `TestRunPlannerReadRepairFallbackAndRegression` asserted the old single tail.
  Now asserts the new header **and** tail, that the defect appears **exactly
  twice**, and that the corrections **precede** the playbook. Strengthened.
- `TestClass41TransportRetry…` asserted `!contains("PREVIOUS ATTEMPT REJECTED")`,
  which class 45 would have made **vacuously true**. Now checks both markers, so
  the "a transport failure is not a rejection" canon still bites.
- `TestClass34…` tested the hint on the legacy function, which is now only the
  degrade path. A new case pins class 34's canon on the path actually used
  (header + tail).

**Suites:** Go **27 ok / 0 FAIL** (full `./...`), goldens PASS, `go vet` clean,
frontend `tsc --noEmit` clean, **vitest 38 files / 298 tests passed**.

## GUIDE (content law)

`web/src/guide/content/guards.ts` — new section *"WHAT THE PLANNER IS TOLD, AND
WHAT IT WAS NOT (class 50)"*, two paragraphs: the three withheld facts, and the
memory failure with the London chain as its evidence.
`web/src/guide/content/faq.ts` — two entries: *"Why was a plan rejected twice for
the same reason?"* and *"Why does the plan sometimes fight the prompt?"*, each
with its mechanism line. `GUIDE_BUILT_REV` is bumped in the **marker commit**
after the boot, per A19.

## CHECKLIST

Class **50** appended to `docs/superpowers/AUDIT-CHECKLIST.md` with the probe:
*enumerate every MUST in the prompt and name the validator function that can
reject a document obeying it — any MUST that names a CONDITION rather than a
DIRECTION is a contradiction waiting for the right market;* plus *every gate that
can rewrite the model's output must state its threshold in the prompt,* and
*every retry states the defects of ALL prior attempts.*

## ROLLBACK

Additive and dormant-safe. `git revert` the wave commit, rebuild, restart —
nothing persists to the DB and no schema changes. Partial escapes, no rebuild
needed for the first two: an empty `VoidBreakdownLevels` renders **nothing**;
`StopFloorATR5m=0` renders **nothing**; an empty reject history falls back to the
legacy single-defect tail. Binary rollback: `nofx-bin.prev.boot`.

## A15 — WHAT I DID NOT DO

- **No cutover.** Not deployed; the five-leg gate has not been taken.
- **No change to the validator.** E2 reads the verdict, it does not restate it —
  a second opinion is the bug being fixed.
- **No change to the stop floor, the R:R gate, or sizing.** E3 only *tells* the
  model the number 0B already enforces.
- **No repair-path change.** Repair (the default retry) is untouched; E4 governs
  the full re-author only.
- **The gap-down/gap-up direction rule was already direction-only in code** —
  only its MESSAGE was corrected, so no behaviour moved there.
- **`GUIDE_BUILT_REV` not bumped here** — that belongs to the marker commit.
- **Slot 46 deliberately left free** in the checklist.

## SEQUENCE

Class 45 boots **before** class 47 stages (owner ruling). Class 47 rebases onto
this, and the boot order at that point must read
`📜 prompt feeds forward:` → `⏱ wakes:` → `🚫 no-chase:`.

## OWED FROM EARLIER WAVES (unchanged by this one)

- **0B:** a BE/trail trigger proving no `move_stop` frame; a sweep→re-arm sequence.
- **Class 36:** the 16:30 CT ASIA halt read quote (bypass line + plan-write time before 17:00).
- **Queued:** the wake-predicate fix (class 47 ships WARN-first first); the
  claim-key comment lie (hygiene); the ATR-less-cycle floor gap (0B finding);
  the tree guard (spec only, `docs/superpowers/plans/2026-09-02-tree-guard-spec.md`).
