# S3 — structure-first seating + validator-stamped relations (DS-102)

**Branch:** `fix/structure-seats-and-relation` · **PR:** #140 · **HEAD:** `3325dd39`
**Base:** `origin/dev 1e3ad705` (S1 CLASS 131 + recorder CLASS 132) · **Lane:** DS-102 (DeepSeek code lane 102)
**Dispatches:** S3 (22:58Z) + amendment (02:23Z) + STOP-line ruling (02:35Z). Owner-approved 2026-09-16 17:55 CT.

## 1. The class-NN find that changed the wave

`seatHTF`'s "2 guaranteed HTF seats" have been a **no-op since G2/G3 (2026-08-24)**.
The pre-seat sort (`levels_score.go:594-606`) and the "restore strict seating order"
sort inside `seatHTF` (~:1082-1096) use the **identical comparator**; the head IS the
top-N of that comparator, so a promoted tail candidate is restored to the tail by
construction, and the caller's `[:maxLevels]` slice keeps it out. The shipped pin test
`TestSeatHTFPromotesSwingLevels` passed vacuously (its HTF candidates outscored the head
fillers — the final sort alone seated them). Reported as a STOP-line; the CTO confirmed
[A] and ruled Option A, knob-gated. **CLASS NN appended** to `AUDIT-CHECKLIST.md`.

## 2. What shipped, per ruling

### (a) `day_plan.htf_seats` — effective promotion, legacy byte-identical
- `seatHTF(scored, maxLevels, seats *int)`: **nil → `seatHTFLegacy`** (the pre-S3 body
  verbatim, 2 seats, whole-list restore sort — today's table exactly, nullification
  included); **saved 0–6 → effective** (promote `*seats`, then `sortSeatBlock(head)` and
  `sortSeatBlock(tail)` separately — promoted slots survive). `*seats ≤ 0` = no-op.
- Plumbing is `*int` end-to-end: `resolveSessionPlanCfg` returns nil unless saved
  (clamped 0–6); `scoreLevelsPool`, `ScoreLevelsMinGradeFullSeats`, the three
  `Assemble*` entry points, engine_analysis and the research harnesses thread it.
- Goldens re-pinned at nil (identity/one-setup/weekly run the production path at nil).
- Tests: `TestSeatHTFPromotesSwingLevels` **rewritten** — HTF candidates score BELOW the
  head fillers; seat 0 under nil, 2 under saved=2; today-priority never demoted.
  `TestSeatHTFSeatsKnob` un-skipped (4/2/0/6 → 4/2/0/5). `TestSeatHTFZeroSeatsNoOp`.
  Parity: `ScoreLevelsMinGradeFull == FullSeats(nil)` byte-identical.

### (b) Structure zones do NOT count against max_levels
Pinned: 12 entry levels + `Structure` zones pass `ValidatePlanDocWithCaps`; 13 hit the
ceiling. The structure block was already a separate `PlanDoc.Structure` field (S1).

### (c) Relation fields — validator-stamped, counter-trend is a flag, never a block
- `PlanScenario` gains `relation_d` / `relation_4h` (enum with-trend|counter-trend|range)
  and `relation_claimed`. `StampScenarioRelations` runs inside
  `ValidatePlanDocWithFactsMachine`: model-supplied relation values are moved to
  `relation_claimed` and overwritten; absent structure or missing TF → fields stay EMPTY.
- **Fail-closed rate does not move** — pinned by `TestCounterTrendScenarioNotRejected`
  (a 4h-opposing short without any relation passes validation) and the existing
  class-38/34 guard suites.
- Prompt contract: `plannerOutputContract` always renders the relation law; a new
  `PromptContract` row asserts it. Repair prompts carry `RepairStructureRelationLaw`
  only when the 4h trend is up/down (`plannerRejectBlock` + all 9 call sites).
- Hint guard: new `HintFieldRelation` with a `*-trend` shape scan — an invented
  "against-trend" spelling fails the build like `reject_retest` did.
- Census: `dayplan_scenarios_by_relation:<session>:<relation>` counters
  (`store/dayplan_relation_counter.go`), incremented at the planner write site from the
  stamped doc (4h relation preferred, D fallback; absent → not counted).

### (d) `day_plan.htf_score_multiplier` — 1.0–1.5, default 1.2
- `ResolveHtfScoreMultiplier` (clamp 1.0–1.5); `scoreLevelsPool` reads `htfMult` instead
  of the const; `TFReadLine` and the assemble/engine paths print the resolved value.
- Boot line at trader load (manager): `🧮 htf_seats=<v>(legacy, non-effective|saved,
  effective) · htf_mult=<v>(default|saved) (S3)` — READ from the resolver.
- UI: `DayPlanEditor` fields (0–6 / 1.0–1.5) + i18n labels; TS `DayPlanConfig` type
  extended. Guide cards for both knobs (L5).

## 3. Twin probe (ruling item d) — seatBothSides is HEALTHY

`seatBothSides` returns ONLY its own maxLevels list — its swaps change membership and
its restore sort only reorders, so the swapped-in below-side level survives.
Pinned by `TestSeatBothSidesSwapSurvivesOwnSort` (gap-day fixture: 8 today-priority kinds
above price, below-side candidates that lose the comparator → `MinSideLevels` below seats
still land). One interaction documented: the effective seatHTF runs BEFORE seatBothSides,
so a promoted HTF seat may be swapped out by the side rule (P0.1 wins) — noted, not changed.

## 4. Verification

- `go build ./...` rc=0 · `go vet ./...` **VET EXIT 0** · `go test ./...` **TEST EXIT 0,
  zero FAIL lines** at HEAD `3325dd39` (tails in /tmp/s3-vet.log, /tmp/s3-full-test.log).
- `cd web && npm run build` rc=0 (tsc green; prettier/eslint ran via lint-staged).
- Knob-off byte-identity: goldens re-pinned at nil + `FullSeats(nil)` parity pin.

## 5. NOT DONE / OWED

- **OWED (owner-gated):** all knobs default OFF/unset — nothing changes the live plan
  until the owner flips `structure_map` ON after S4 final. No deploy, no boot, no
  GUIDE_BUILT_REV bump (deploy lane's step).
- `TFBootLine` (main boot, pre-strategy) still prints the const 1.2 with `[I]`; the
  per-trader `🧮` line is the source of truth once a strategy loads.
- The seatHTF↔seatBothSides interaction (P0.1 may evict a promoted HTF seat) is
  documented, not resolved — needs a ruling only if S4 shows HTF seating matters.
- `kernel/plan_doc_provenance_merged_test.go` and `store/knob_registry_table.go` are
  gofmt-flagged on dev itself (aligned-table style) — left untouched.

**Report commit-ref URL:** raw verified 200 after push.
