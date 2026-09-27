# W3 — CANDIDATES, NOT ENTITLEMENTS (Dispatch 103)

**Branch** `fix/candidates-not-entitlements` · claim `a9bea0a3`, session
`candidates-not-entitlements-51524a30/nofx-66 [fd21dd]`
**Code merged** in the combined boot head `8941ec68` (with 102 bars-horizon, 101 rider, 104
session-risk), booted and marked `18bf1e01` 2026-09-10 06:58 CT
**Docs merged** `240e8cd4` (`--ff-only`, +53 −1)
**Rev the wave was built and measured against** `954f11b1`, verified three ways: `/api/health`
→ `954f11b15f2e`; `/proc/438/exe` → `vcs.revision=954f11b15f2e…`, `vcs.modified=false`;
`deploy/RELEASE` → `954f11b1`.
**Evidence classes** [A] verified · [B] inferred · [C] speculation

---

## 1. C1–C6 — two premises did not reproduce

Full working: `2026-09-09-candidates-data/C1-C6-measured.md`.

| # | premise | verdict | measured |
|---|---|---|---|
| C1 | max_levels=12; 12/12 seated; 181 cut rows all score 0 | **partly** | `max_levels`=12 resolved ✓. Live reads show **both** `seated 12/…` and `seated 24/…` — 24 is the **2× pre-seat pool** (`levels_score.go:646`, `eff*2`), not a second cap. "181 rows all score 0" was the **09-04** sample; on 09-09 all 84 cut rows carried real scores — Stage A works, no A24 regression. The cap was cutting **grade-A rows at 1.344–1.36**. |
| C2 | 22% of seats >100 pt | **NOT REPRODUCED — worse** | **29.8%** (n=84, ids 1009–1176, mean 150.1 pt) |
| C3 | correlated labels count as confluence | **confirmed + sharpened** | `conf` counts distinct families within `confBand = 0.10×dATR` (**≈±20.3 pt**) while clusters collapse at a fixed **3.00 pt** — **~6.8× apart, and both are commented "cluster tolerance"** |
| C4 | OB 8/117 (6.8%), EQH 0/4 | **NOT REPRODUCED** | **OB 34/187 = 18.2%** · SUPPLY 33.3% · DEMAND 40.0% · **EQH 17/146 = 11.6%**. Tier-1 anchors seat 100% |
| C5 | nothing exists beyond the map | **confirmed** | measured-move **0**, ATR-projected extreme **0** |
| C6 | PWH/PWL can never seat | **confirmed + sharpened** | `priorWeekMinBars=4320` on a ~33 h ring; the const block's own comment says this "can never pass, which is CORRECT: those anchors must come from a multi-day source, not the ring." Zero PWH/PWL rows in `candidate_pool`, ever. Daily bars exist: `tf='1d'`, n=1901 |

**A third premise was withdrawn by me, not by the dispatch.** I reported the levels boot line as
printing a literal `seats=8`. It does not: `0e016635` (2026-09-04) fixed it to
`seats=per-trader (default 8, hard cap 12)` and that fix **is in the running rev**. My claim came
from grepping `nofx_2026-09-0*.log` and quoting the **09-03** line as current.

## 2. The owner's ruling, and what it changed

D2 as written ("the confluence credit is computed on MERGED candidates") and E7 ("a golden diff on
the score of every fixture level is EMPTY") are mutually exclusive: merging three names at one price
takes `confMult` from 1.40 to 1.00, a −28.6% score change, and `stage_a_parity_test.go:21-46`
byte-compares the full `[]ScoredLevel` across 64 fixtures against a 685,898-byte golden.

**Ruled: E7 and A31 govern — presentation and ordering only.** So every W3 datum rides a
render-time `MapCandidate` view built from `[]ScoredLevel`; `ScoredLevel` is untouched and the
golden cannot move. `MapRole` is a **separate axis** from the five `LevelRole` values, which are
serialized into that same golden.

The confluence findings are **recorded for experiment E4**, not fixed
(`2026-09-09-candidates-data/E4-recorded-findings.md`): F1 the 1.40×/1.60× premium, F2 the 6.8× gap
— and the corollary that merging at 3.00 pt would not have closed F1 anyway, since a pair 20 pt
apart never merges.

## 3. What shipped

| | |
|---|---|
| **D1** map kept whole | nothing is ever dropped; every candidate carries a role (class 93) |
| **D2** merge | references within one zone-width → ONE candidate, ALL names, ONE credit |
| **D3** candidacy `[I]` | an entry candidate needs an opposing reference ≥ the resolved minimum (default: the stop floor). The wave's only new refusal; it refuses candidacy, never an authored scenario |
| **D4** order `[I]` | shortlist ranks by reachability; score carried and shown, ranks second |
| **D5** projections `[I]` | PWH/PWL from the **daily** source, RN window, ATR-projected extreme, measured move — targets/obstacles only, never entries, no detector grade |
| **D6** | executor table · planner table · plan card |
| **D7** | boot line + per-read line |

**Two places the dispatch's literal shape was overridden, both on canon.** `cap=per-trader
(default 8, hard cap 12)` rather than `cap=12` — several traders, none of their values global; this
is the shape `VolumeWaveBootLine` adopted when `0e016635` fixed the same class, and **my own D7 test
asserted `cap=12` and was corrected, not the line.** Per-read fields print `n/a` at boot, never `0`.

**D4 is deliberately not applied to the card's array.** `EditSheet` patches by position
(`:136` replace, `:188` remove); reordering what the owner sees would make an in-flight edit rewrite
or **delete** a different level than the one clicked.

## 4. Tests — RED quoted, and the mutation that survived

E1 merge · E2/E2b/E2c no-target · E3/E3b reachability · E4 the 09-03 projection · E5/E5b/E5c
PWH/PWL · E6 exclusion-is-not-invalidation · E7 golden · D5b/c/d. RED for the first batch was
`undefined: BuildMapCandidates` / `MapCandidate` / `MapCandidateOpts`.

**E9 mutations.** M1 (merge width → 0.01) and M3 (order → score-first) died with quoted failures.
**M2 (stop-floor default → 0.0) SURVIVED** — E2/E2b both pass `MinTargetDistance` explicitly and
never exercised the fallback. `TestE2c` was added to drive the default path; M2 now dies with
`the minimum failed to resolve from ATR5m; reason "no resolved minimum target distance — candidacy undecided"`.
A mutation that passes is not evidence (A8), and this one proved it.

**E7 held throughout:** `TestStageAScoreParityLegacy` passes; the golden is byte-identical.
Suites at the merged head: full Go suite green; `tsc` clean; **58 files / 414 vitest**.

## 5. A15 — what is still wrong

- **The scored confluence term still credits correlated labels** (F1) and still counts them over a
  window ~6.8× the merge width (F2). E4 owns both; nothing here changed a score.
- **`confBand` and `LevelClusterTicks` are both commented "cluster tolerance"** and are not the same
  thing — a reader of either comment will be misled.
- **Two render sites fabricate an uncomputed role as `react_zone`** (`levels_score.go:1148-1150`,
  `planner_prompt.go:507-509`) — a class-49 fabricated value. The new map-role column prints `n/a`;
  theirs was left alone.
- **`RoleForLabel` hard-codes `fresh=""`**, so the label path can never return `target_only`.
- **`armGateVerdict`** (`armed_executor.go:1268`) has **0 production callers**; production uses
  `armGateVerdictFor`.
- **`faq.ts` says "the fourteen questions"** with 19 present; **`glossary.ts` still defines "Thin
  side"** though two other pages record it as removed on 2026-08-31.
- **Vitest cannot run in a git worktree**: `.git` is a *file* there, so vite's workspace-root probe
  fails and `fs.allow` denies `../../../branding/*.txt?raw`. Worked around with an untracked config;
  every future worktree hits it.

## 6. What this wave got wrong

Recorded because the pattern repeated, and twice it was caught by a rule written earlier in the
same wave.

1. **`seats=8`** — quoted a 09-03 log line as current; the fix had shipped on 09-04.
2. **The class number** — filed 95, collided twice; landed 98 after the combined-boot lane
   renumbered it correctly (A16 protects the *landed* entry).
3. **The gate query** — retyped five of seven terminal states and invented a sixth. Reported
   resting arms to the owner as a reason to hold when the true count was **0**. → **class 99**.
   *(Corrected 2026-09-10: this said **11**; the query never returned 11 — every reading was
   **10**. The 11 was a hardcoded `echo` label printed above output that said 10, and it
   propagated from here into class 99 and into two lanes' messages. See the correction note in
   class 99.)*
4. **The stale base** — the branch was green and correct for eight hours, then diffed as **5,403
   deletions** of 102's and 101's landed work after a combined boot merged it. Nothing conflicted,
   nothing failed. → **class 100**, which then caught the same branch a **second** time the next
   morning (a 392-line research doc, plus `RELEASE` and `GUIDE_BUILT_REV` rollbacks).
5. **Lane ownership** — repeated the dispatch's "102 holds fix/episode-contract" without checking.
   It is **101's**; 102 is bars-horizon. I had verified that branch's *merge status* carefully and
   never questioned whose it was.
6. **A shared file** — wrote an ignore line into `.git/info/exclude`, which worktrees **share** with
   the main tree, so it would have changed every other lane's git behaviour. Reverted.

## 7. Rollback

The code rode the combined boot; rollback is that boot's, not this wave's — `18bf1e01` names
`RELEASE=8941ec68` and the prior binary is named for the rev it holds. These docs are additive
(`240e8cd4`, +53 −1) and revert with `git revert` without touching code.
