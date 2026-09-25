P='/home/hoang/nofx-vet-05/docs/superpowers/reports/2026-09-05-vet-05-execution.md'
s=open(P,encoding='utf-8').read(); E=[]
def rep(old,new,tag):
    global s
    if s.count(old)!=1: print("!! FAILED (%d) %s"%(s.count(old),tag)); E.append(tag); return
    s=s.replace(old,new); print("ok",tag)

rep("""**Verdict: BROKEN stop-entry construction and guard; demonstrated SIM limit lifecycle; execution readiness remains unproven.**""",
"""**Verdict: BROKEN stop-entry construction and guard; demonstrated SIM limit lifecycle; execution readiness remains unproven.** *(Revised 2026-09-05 after three adversarial verification lenses — numbers, code+checklist, research labels. The verdict and both stop-entry defects survive unchanged and I could not refute either. What did not survive: my primary population carried seven rows the governing rule excludes, so every performance figure is re-cut on 58 rows; the daily-loss item is the owner's dated ruling, not a defect; the invalidation episode is checklist class 59, whose entry half is already shipped and live; and the empty excursion table was already explained by a commit date. 23 findings, 23 accepted, 0 rejected as findings — two of the verifiers' proposed replacement values are refused with the query below. Full dispositions: VERIFICATION RECORD.)*""","verdict")

rep("""**[T] The requested `trade_excursions` answer is unavailable: zero rows.** Stored `trader_positions.mae/mfe` is a separate bar-based proxy.""",
"""**[T] The requested `trade_excursions` answer is unavailable: zero rows — and the reason is settled, not open.** The writer landed in `44d4bbb7` (2026-09-02 23:46:19 CT); the last stored position 591 opened 09-03 09:05:14 under rev `33de2bef`, which does not contain it (`git merge-base --is-ancestor 44d4bbb7 33de2bef900d` → false), and nothing has opened since. No trade has ever closed under a binary carrying the hook. Stored `trader_positions.mae/mfe` is a separate bar-based proxy.""","q5-exc-reason")

# Section 09 cross-reference + disagreements, inserted before Surprises
rep("""## Surprises found, never acted on""",
"""## Cross-reference: section 09 (`2026-09-05-vet-09-top-ten.md`)

Read from `origin/dev`; I edited nothing of theirs.

**Where it disputes me, it is right and I have conceded.** `vet-09-top-ten.md:33` names sections 01, 05 and 06 as running the broader 65-row population and rules that "their performance/exit/excursion rates are not compliant primary estimates under this dispatch," listing the identical seven sentinel ids and Σ **−$97.50**, and reconciling **−563.93 + 97.50 = −466.43**. My independent re-derivation (`revise/r01_compliant.py`) lands on the same 58 ids, the same **−$466.43 / mean −$8.04 / 18W-38L-2F**, and its note that q01 "missed the attribution sentinel by searching correction notes" is exactly my defect too: `q31_verified.py:19` searched `close_reason` and `pnl_correction_note` and never `plan_id`. Their `q03` and my `r01` agree row for row.

**Where it endorses me, I record the agreement.** `vet-09-top-ten.md:109` adopts this section's Q1 finding — position 591's broker stop and fill both at **29355**, therefore **zero broker-referenced stop slippage**, with the 3.3715-pt gap being ledger geometry drift — and rules that "section 04's desk-strip example and section 08's slippage label must not be used as execution-cost evidence; section 05's raw NT8 line receipts supersede them." That is the disagreement I filed at Q1 being upheld against two other sections.

**Where I agree on substance and differ on framing.** Their rank 1 is my rank 1: the stop-entry contract, citing the same `VLTraderTCPClient.cs:972`/`:978` slot swap and the same `armed_executor.go:940` → `:993` inverted predicate. Their rank 3 asks to "separately verify the already-merged daily entry-blocking change in the running binary before treating the configured $450 limit as protection" — which is precisely the revision gap I have now made my rank 2, and it is a better statement of the item than my draft's "resolve daily-loss policy." I have adopted their framing. Their rank 3 also observes that `risk_limits.go:165` blocks entries and does not close positions and that "the name ForceFlat is not its behavior"; I read the same lines as a *deliberate documented scope* rather than a naming defect, which is a difference of emphasis, not of fact.

**One thing I would add to their rank 9.** They list excursion emptiness under "make the record of a trade complete." It belongs there, but the empty table specifically needs no engineering: it needs one trade under a binary carrying `44d4bbb7`. The engineering that *is* needed is the different, forward-looking hole at `armed_executor.go:1287` sitting below the `:1246-1248` early return — see my rank 4.

## Surprises found, never acted on""","xref09")

open(P,'w',encoding='utf-8').write(s)
print("\nFAILED:",E if E else "none")
