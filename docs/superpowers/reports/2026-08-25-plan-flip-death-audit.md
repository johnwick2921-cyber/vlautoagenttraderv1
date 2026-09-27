# 14-Plan Full Audit — Flip/Death/Label Correctness (2026-08-25)

Owner question: "why old point for flip when plan re-read, it said new flip
point than bias". Answered + fixed. 4 agents audited all 14 plan versions
(LONDON/NY/ASIA, 2026-08-23/24).

## 1. The flip-point bug (answered)

ASIA v4 (written 22:39 CT 08-24) anchored `flip` AND `death` at **29263.25
labeled "PDL"** — ~109 pts above price, ~108 pts above v3's flip (29154.38).

**Root cause (fixed in bbb8c677):** `ExtractMultiDayLevels`
(kernel/levels_multiday.go) computed PDH/PDL/PDC from the sliding 2000-bar 1m
ring with NO coverage guard. The v4 read ran seconds after a bot restart + a
stalled feed, so the prior-day bucket only contained Friday afternoon — and the
"low" of that truncated bucket was the CLOSE region (PDC 29266.25 − 3.00 =
29263.25). The model copied the Go label faithfully (the prompt says Go facts
are to be trusted, not re-derived). The input was wrong, not the model.

**Fix:** prior-day buckets with <15h of closed bars are omitted; prior
week/month need 3/7 days (the ~33h ring can never fabricate those again —
also kills the bogus PWL 29220.25 traded as overhead supply in 7 NY plans).

## 2. What the audit found across all 14 plans

### Flip/death (7 of 11 active plans broken)
- **NY v2**: death (15m_close above 29082.50) preempts flip (15m_close above
  29116.75) by construction → flip can never fire.
- **NY v4 / v6 / v7**: flip == death (identical price+side+rule) → flip_to
  fires at the same tick the plan dies; void.
- **NY v8 / ASIA v1**: death below flip with the easier rule → flip
  effectively unreachable.
- **ASIA v3**: flip references 29154.38 which exists NOWHERE in its own level
  list, labeled "4h bear OB" while the list's only OB is OB(bull)·4h @ 29162.75
  (bull/bear contradiction).
- **ASIA v4**: stale PDL anchor (the owner's bug — §1).

### Label vs side semantics (9 of 14 plans)
- PDL 29263.25 above price traded as overhead resistance (NY v1–v8, ASIA v4).
- PWL 29220.25 above price as supply (7 plans).
- EQL above price used as upside targets; EQH below price as breakdown
  triggers (NY v1/v7/v8, ASIA v1/v3).
- Same price gets DIFFERENT labels across sessions (29220.25 = PDL on 08-23,
  PWL on 08-24).

### Scenarios
- 88% of scenarios quality C (29/33) — G5 demotes every scenario whose trigger
  is consumed, and the C-gate revert makes C tradable → quality field provides
  zero prioritization right now.
- Orphan triggers: ASIA v4 S1 "29156.50 EQH" not in levels; NY v3 S2 target
  28947.75 never a listed level.
- All 3 ASIA plans carry NY-session no_trade boilerplate into the Asian
  session.

## 3. Shipped fixes (bbb8c677, deployed 01:15 CT, BOOT OK)

1. Multi-day coverage guard (root cause) + tests.
2. Write-time WARNs (never gates): orphan flip/death anchor, flip==death
   degeneracy, death-preempts-flip.
3. Prompt: flip and death MUST be different events; short-biased flip sits
   below death or uses a stricter rule.

## 4. Still open (owner decisions — not changed)

- Quality-C monopoly: every consumed-trigger scenario demotes to C → no
  differentiation. Options: keep C tradable but weight A/B in the executor, or
  require at least one non-consumed scenario per plan.
- Label provenance check at write: compare plan level labels vs the machine
  label at the same price (machineGrades map already keys by rounded price —
  extend to compare labels, warn on mismatch).
- Death-preempts-flip: promote the WARN to a reject after 2-3 sessions of
  observing the warning rate.
- EQL/EQH misplacement: consider rejecting EQL-as-upside-target label misuse.
