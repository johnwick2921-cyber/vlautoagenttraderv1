# Adversarial verify — armGateVerdict (peer claim: DEAD + class-53 test-shape defect)

Rev checked: worktree /home/hoang/nofx-conform HEAD acd92e26; lines re-derived at deployed
rev 70af663d and dev tip 492d2067 — :1305 (decl) and :430 (production call) are IDENTICAL at
all three revs.

## Reproduced (peer CORRECT)
- Exact-token grep, whole repo, all file types: 10 occurrences of `armGateVerdict` (excluding
  `armGateVerdictFor`) — 1 doc comment :1303, 1 decl :1305, 8 test calls. [A]
- Method-level survival check: `grep -rn "armGateVerdict[^F(]"` returns ONLY the doc comment,
  so there is no method-value reference (`at.armGateVerdict` passed as a func value), no
  interface declaring it, and no `MethodByName` reflection. The method is unexported, so
  package `trader` is the only reachable scope and the repo-wide grep is exhaustive. [A]
- => 0 production callers. A29 DEAD is CONFIRMED.
- Peer's file:line is CURRENT (:1305), unlike the two-day-audit report it descends from,
  which still cites :1268/:415.

## Refuted (peer's class-53 rationale is factually inverted)
1. `armGateVerdictFor` (:1316-1377) reads exactly THREE leg fields — `leg.Entry`, `leg.Stop`,
   `leg.Target` (:1347-1348, :1350-1351, :1357-1362). It reads no `leg.WaitConfirm`, no
   `leg.Rule`, no `leg.Kind`. Every other input comes from `sc`, `cfg`, `snap`, `atr5m`,
   `minQuality`, `session`. The wrapper at :1309 supplies precisely those three fields from
   `sc.Arm`. There is therefore NO "leg shape" the wrapper hides from the gate. [A]
2. Production builds the SAME triple for a single arm: armed_executor.go:306-308 —
   `legs = []kernel.PlanArmLeg{{Entry: sc.Arm.Entry, Stop: sc.Arm.Stop, Target: sc.Arm.Target, ...}}`
   whenever `len(sc.Arm.Legs) == 0`. [A]
3. MEASURED (A17/A21): the single-arm shape is the shape production PASSES, not a retired one.
   `sqlite3 file:/home/hoang/nofx/data/data.db?mode=ro "select leg_count,count(*) from armed_orders group by 1"`
   -> leg_count 0 = 34 rows, leg_count 2 = 4 rows, n=38. 34/38 (89%) of every armed row the
   system has ever written is the "legacy" single-arm shape. [A]
4. By design, too: kernel/plan_doc.go:1210 `normalizeArmLegs` COLLAPSES any non-sweep_reclaim
   multi-leg arm back to the single top-level arm (class 39). The single shape is the default
   production outcome, not a legacy leftover. [A]

## The real gap the peer MISSED (measure-first)
Production mutates `leg.Stop` via `composeArmStop` (0B stop anchoring, :385-407) BEFORE the
gate call at :430. The wrapper does not. The 8 tests therefore never see a stop the composer
WIDENED — which is exactly two-day-audit D37 (:921 block): "All 7 R:R-at-arm refusals were
authored at R:R >= 2.0 and pushed below the 2.0 floor by composeArmStop widening the stop."
That is a stop-VALUE coverage gap, not a leg-SHAPE gap, and re-pointing the tests at :430 with
a raw leg would not close it. Note also `trader/zerob_exit_sanity_test.go:60,70,155` already
calls `armGateVerdictFor` directly with an explicit leg, so the production signature IS
exercised by the suite — a fact that further weakens the peer's class-53 framing.
