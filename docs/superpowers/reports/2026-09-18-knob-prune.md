# W-KNOB-PRUNE — Day Plan knob prune (owner order 2026-09-18 00:3x CT, "full fix 6")

Branch `feat/knob-prune` cut from origin/dev TIP `0dd27940` (release d7f86fc5 live 00:09 CT).
Claim `808c6ea8` pushed 00:46:02 CT. Verdict implemented EXACTLY as saved in
`~/.claude/projects/-home-hoang-nofx/memory/project_knob_prune_list.md` (2026-09-17 evening,
parked by the owner, re-ordered 2026-09-18). No merge, no deploy.

Spec freshness: `git log -1 -- docs/superpowers/plans/…` not applicable — the spec is the
memory file (untracked); its text is quoted in the dispatch and matches the file read at
00:46 CT.

## 1. Live stored-value audit (read-only, `sqlite3 -readonly data.db`, 2026-09-18 00:47 CT)

Nine strategies carry a `day_plan` block. Eight are Studio seeds:
`{"acceptance_rule":"5m_close","approval_required":false,"eod_flat_ct":"14:45","evening_digest":true,
"last_entry_ct":"13:00","max_levels":8,"plan_enabled":…,"plan_mode":"advisory",…,"replan_cap":2,
"scenario_cap":3,"sessions_enabled":["NY"]}` — every pruned knob at its default except
`evening_digest:true` (honoured, see §3).

The owner's **MNQ** strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` stores NON-defaults on
five pruned knobs — each is honoured by the surviving path and logged once at trader load:

| stored on MNQ            | value | fate      | surviving path                                                    |
|--------------------------|-------|-----------|-------------------------------------------------------------------|
| `structure_map`          | true  | FOLDED    | `ResolveStructureMap` unchanged; UI control gone                  |
| `scenario_cap`           | 5     | FOLDED    | `DayPlanConfig.ScenarioCapResolved()` → 5                         |
| `realign_cap`            | 10    | FOLDED    | `DayPlanConfig.RealignCapResolved()` → 10                         |
| `wake_on_htf_ob`         | true  | COLLAPSED | `WakeOnHTFOrderBlocks()` keeps the OB wake class for this strategy |
| `levels_fresh_by_tf`     | true  | FOLDED*   | grader kept; `LevelsFreshByTFEnabled()` → by-tf                    |
| `evening_digest`         | true  | FOLDED    | `EveningDigestOn()` → true (all nine rows store true)              |
| `acceptance_rule`        | 5m_close (+3 session rows) | FOLDED | the one rule; stored value inert (it already was since 2026-08-30) |
| `last_entry_ct` / `eod_flat_ct` | 13:00 / 14:45 | DELETED | ignored on load (encoding/json), never read — proof §5           |

\* `levels_fresh_by_tf` is on the verdict's REMOVE list. Deleting the code path would have
changed the owner's live prompt (the HTF freshness strings) — L3 forbids a removal that
silently changes a set value, so the knob is treated as FOLDED: control gone, registry
`folded`, stored true honoured + logged. When the owner clears it, a follow-up deletes
`kernel/levels_fresh_by_tf.go`. **Not re-decided; flagged.**

Nothing stores `htf_score_multiplier`, `seat_1h_zone`, `wake_min_interval_min`,
`wake_on_15m_zone/htf_zone/ifvg/seated_invalidation`.

## 2. Knob table

| # | knob | fate | now | evidence | pin |
|---|------|------|-----|----------|-----|
| 1 | `levels_fresh_by_tf` | REMOVE → folded (owner stores ON) | OFF unless stored | R23 S4c: freshness separates nothing | `trader/knob_prune_pin_test.go::TestKnobPrunePin_Resolvers` (Freshness per stored shape) |
| 2 | `htf_score_multiplier` | REMOVE, hard-wire **1.0** | `kernel.HTFScoreMultiplier = 1.0`; field + `ResolveHtfScoreMultiplier` deleted | R23 Q-C: 1.2 promotes a group that holds LESS; owner: "1.0 is the same" | identity golden re-pinned; `TestKnobPrunePin_Seating`; `TestT2EQL15mHTFGradeAReproducible` derives from the const |
| 3 | `seat_1h_zone` | REMOVE | 1h S/D seat guarantee unconditional (was ON, never stored) | R24: no zone kind/TF beats random | `TestKnobPrunePin_Seating`, `TestKnobPrunePin_ExecutorKeyLevels` (byte-identical) |
| 4 | `structure_map` | REMOVE UI, keep path (owner stores ON) | OFF unless stored | advisory text only, keep OFF until R25 | `TestKnobPrunePin_PlannerPromptDefaults` (OFF prompt byte-identical) + `TestKnobPrunePin_Resolvers` |
| 5–7 | `wake_on_15m_zone`, `wake_on_htf_ob`, `wake_on_ifvg` (+ `wake_on_htf_zone`, `wake_on_seated_invalidation`) | COLLAPSE → `wake_on_level_events` | one switch, nil = ON; legacy five: any ON → ON; OB class only via legacy stored `wake_on_htf_ob=true` | R23/R24 kept HTF zones + seated invalidation | `TestKnobPrunePin_WakeCandidates` — candidate lists per stored shape (nil/empty/seed/owner/all-off) byte-identical to the pre-prune golden |
| 8 | `acceptance_rule` | FOLD | `AcceptanceRuleFor` returns the one rule always; dropdown + session override row gone | one option since 2026-08-30 | `TestKnobPrunePin_Resolvers`, `TestAcceptanceRuleForSelfHeals`, `TestW15AcceptanceRuleResolution` |
| 9 | `realign_cap` | FOLD (into the re-plan section) | `store.DefaultRealignCap=5` unless stored | overlaps replan_cap | `TestKnobPrunePin_Resolvers` (5 / owner 10) |
| 10 | `evening_digest` | FOLD | OFF unless stored true | owner verdict | `TestKnobPrunePin_Resolvers` (seed/owner true; empty false) |
| 11 | `wake_min_interval_min` | FOLD | 30 unless stored | keep value not control | `TestKnobPrunePin_WakeCandidates` (Interval per shape), `TestDayPlanWakeKnobDefaults` |
| 12 | `scenario_cap` | FOLD | 3 unless stored | owner verdict | `TestKnobPrunePin_Resolvers` (3 / owner 5), `TestKnobPrunePin_PlannerPromptDefaults` |
| 13–14 | `last_entry_ct`, `eod_flat_ct` | DELETE | struct fields, readers, `dayplan-arm` flags, registry rows gone | unreachable in the clock since P2 (2026-08-18) | `store.TestDayPlanLegacyClockFieldsIgnoredOnLoad` (reflection: no owner, not enumerated; old row loads) |

KEEP (17) untouched: plan_enabled, planner_model, plan_mode, planner_timeframes, sessions,
structural_stop, proximity_filter_atr, max_levels, htf_seats, one_setup_enabled + min_grade,
min_scenario_quality, replan_cap, approval_required, flip_reread, (wake_on_seated_invalidation
and wake_on_htf_zone are what the single switch means). `t1_currencies` (feat/t1-currencies)
not touched; expect a small merge in `web/src/guide/content/settings.ts` / knob count.

## 3. Byte-identical proofs (L8) — how they were made

The pins were written FIRST and their goldens generated at the base commit (`013256c3`,
tree = origin/dev `0dd27940` + the two test files), then the prune landed and the same tests
ran again. Goldens live in `kernel/testdata/knob_prune/` and `trader/testdata/knob_prune/`;
regenerate with `KNOB_PRUNE_WRITE_GOLDEN=1`.

HELD unchanged after the prune (before == after, bytes):
- `trader/testdata/knob_prune/wake_candidates.json` — collector output for nil / `{}` / the
  Studio seed / the owner's MNQ JSON verbatim / all-five-false: 5 / 5 / 5 / **7 (2 OB)** / 0
  candidates, same keys, same priorities, same interval (30).
- `kernel/testdata/knob_prune/executor_keylevels.txt` — executor KEY LEVELS block
  (max_levels 8, seat guarantee) — identical (scores are not rendered there).
- `kernel/testdata/knob_prune/planner_prompt_defaults.txt` — planner prompt at scenario_cap 3,
  structure OFF; and `ScenarioCap:0` renders the same bytes.
- `kernel/testdata/futures_mnq_plan.golden` etc. — the existing prompt goldens still pass.

RE-PINNED (the deliberate change, htf mult 1.2 → 1.0) — diffs measured before overwriting:
- `kernel/testdata/identity_legacy_output.json` (identity tape, 12 seats): **0 seat-membership
  changes, 0 order changes, rendered map block identical**; only PDC and PDH scores 1.92 → 1.60
  (grade A → A). At 8 seats: same result (0 / 0).
- `kernel/testdata/knob_prune/seating.json`: same fixture, same 0/0.
- `kernel/testdata/stage_a_score_legacy.json` (64 scorer fixtures, `ScoreLevelsMinGradeFull`
  across grade × cap × freshness): **33 of 64 fixtures change seat membership, 0 change order
  only, 31 change scores/grades only; 112 seated rows change grade.** This is the honest size
  of the owner's "1.0 is the same": on a tape where HTF and non-HTF candidates compete at the
  cut line, the seat set moves. The identity fixture has no such competition.
- `kernel/leveltruth_t2_test.go`: EQL·15m (HTF) 0.70×1.2×1.0 = 0.84 → B (was A at 1.2);
  plain 0.70 → B. The expectation is now derived from the constant.
- `trader/testdata/knob_prune/resolvers.json`: `HTFMult` 1.2 → 1.0 on every shape; `nil`
  shape `Digest` true → false (production-unreachable: `maybeWriteDigests` returns before
  the resolver when no day plan is enabled; every real block stores true).

## 4. Folded-knob log line

`manager/trader_manager.go` at trader load, after the 🧮 HTF line, one line per folded knob
whose STORED value differs from its constant:
`⚙ folded knob <name>=<value> honoured from stored config`. For the owner's MNQ strategy the
boot will print: structure_map=true, scenario_cap=5, realign_cap=10, evening_digest=true,
levels_fresh_by_tf=true, `wake_on_15m_zone/htf_zone/htf_ob/seated_invalidation/ifvg →
wake_on_level_events=true`, `wake_on_htf_ob (order-block class)=true`. A stored
`acceptance_rule` that is not the rule logs "NOT honoured" (it never was since 08-30).
Pure function `store.DayPlanConfig.FoldedKnobLines()`; a nil block or an all-default block
prints nothing.

## 5. Dead fields — grep proof

```
grep -rn --include=*.go -E 'LastEntryCT|EODFlatCT|last_entry_ct|eod_flat_ct' . \
  | grep -v '^./docs/' | grep -v _test.go | grep -v '^\S*:\s*//'
trader/auto_trader_clock.go:312:func effectiveEODFlatCT(reg kernel.SessionRegistry, sessionDayKey, configFlat string) string {
```
The single survivor is a session-calendar helper whose `configFlat` comes from the session
registry (it could not compile against a deleted field). `store.TestDayPlanLegacyClockFieldsIgnoredOnLoad`
proves by reflection that no struct owns either tag and the enumerator no longer yields them,
and that a live-shaped row carrying both still loads with the per-session offsets intact.
`AuditDeadKnobs2026_09_03` drops the two (they are no longer schema fields).

## 6. Registry

New status `KnobFolded` ("folded — no control; stored value honoured (reason)"), counted in
`KnobStatusSummary` / boot line `folded=N` / `GET /api/config/resolved` `summary.folded` /
the Studio resolved panel. Rows: 4 deleted (`htf_score_multiplier`, `seat_1h_zone`,
`last_entry_ct`, `eod_flat_ct`), 11 reclassified `folded` (acceptance_rule, evening_digest,
levels_fresh_by_tf, structure_map, realign_cap, scenario_cap, wake_min_interval_min, the five
legacy wake switches), 1 added live (`wake_on_level_events`, consumer
`trader/auto_trader_wake_levels.go:103`). `TestWakeKnobsAreLiveThroughTheirAccessors` now
asserts folded/live per leaf and still requires a detector-found call site for each.

## 7. Guide (L7)

52 → 43 knob cards (`GuidePage.test.tsx`). Removed cards: Structure table (S1), HTF score
multiplier (S3), Max scenarios, Acceptance window, Evening digest, Re-align cap, 1h anchor
seat, HTF freshness by own timeframe, Min wake interval. Rewritten: "Wake triggers (5
toggles)" → "Wake on level events (1 switch)"; Max re-plans names the folded re-align budget;
Session overrides drops the acceptance row; registry-status table gains `folded`; new
section "Folded and removed knobs (W-KNOB-PRUNE)" with the fate/evidence table. `levels.ts`
freshness paragraph and `planCard.ts` structure / re-align text rewritten. i18n: 13 keys
removed, 1 added, no orphans (grep). **`GUIDE_BUILT_REV` not bumped here** — per Boot 5 it is
bumped to the shipped rev by the deploy lane before the boot.

## 8. Not done / owner decisions surfaced

1. `levels_fresh_by_tf`: folded, not deleted (owner's stored ON). Delete the grader once
   cleared.
2. `evening_digest` for NEW strategies flips from seeded-true to off (Go and Studio no longer
   seed it) — the verdict's "constant off unless stored on". Every existing row stores true.
3. `realign_cap` "fold into replan_cap": implemented as constant 5 unless stored, described
   beside the re-plan card. A literal tie to the replan value would have changed the shipped
   default 5 → 2, which L8 forbids.
4. 1.0 HTF weight changes seating on 33/64 stage-A fixtures (§3) — the owner's call, now
   quantified; the identity tape shows 0.
5. Single-switch semantics: OFF disables every level-event class (15m/HTF/iFVG/invalidation);
   a legacy strategy with SOME switches off maps to ON (per-class suppression is not preserved
   — no live strategy stored one).
6. `GUIDE_BUILT_REV` bump — deploy lane.
6b. **New coupling (review of #172):** `WakeOnHTFOrderBlocks()` ANDs the legacy
   `wake_on_htf_ob` with `wake_on_level_events`. Pre-prune the OB class was gated only by its
   own field; now the single switch OFF disables OBs too (intended single-switch semantics).
   Inert for every stored strategy today (none stores the new switch). Pinned by
   `trader.TestKnobPrunePin_WakeCandidates_SingleSwitchOwnsOB`: `{wake_on_level_events:false,
   wake_on_htf_ob:true}` → zero candidates, while the same fixture yields OBs with the switch
   on. Named in the registry note for `wake_on_htf_ob`.
7. Go layer committed as one commit (`fdce92e6`): the struct/registry edits are shared by
   every knob group, so a per-group split would have left non-compiling intermediate commits.

## 9. Verification tails

All run under `nice -n 19 ionice -c 3` in the worktree, 2026-09-18 01:0x–01:26 CT.

```
$ gofmt -l <touched .go files>           → (empty)
$ go vet ./...                            → (empty)
$ go test ./... -count=1                  → 35 packages ok, EXIT=0
    ok  nofx/api      9.071s · ok  nofx/kernel  1.457s · ok  nofx/store  86.310s
    ok  nofx/trader 370.227s · ok  nofx/manager … · ok  nofx/trader/ninjatrader 13.183s
$ go test ./kernel ./trader ./store -race -count=1
    ok  nofx/kernel   4.885s
    ok  nofx/trader 422.007s
    ok  nofx/store  138.945s
    EXIT=0
$ cd web && npx tsc --noEmit -p .         → (empty)
$ npx vitest run                          → Test Files 71 passed (71) · Tests 463 passed (463)
$ npm run build                           → ✓ built in 4.92s
```

Commits on `feat/knob-prune` (all pushed): `808c6ea8` claim · `013256c3` pre-prune pins ·
`fdce92e6` Go · `d977af91` web editor · `3a48a8ec` guide · (this report).

## 10. Merge of origin/dev f820a642 (CLASS 149 #170 + CLASS 150 #171) — a32ed9d6, 01:4x CT

Conflicts (5), each resolved keeping BOTH sides: `store/strategy.go` (t1_currencies field +
folded acceptance_rule comment) · `web/src/types/strategy.ts` (same) ·
`web/src/i18n/plan-translations.ts` (t1Currencies + noTradeAdvisory kept, maxScenarios stays
removed) · `web/src/guide/content/settings.ts` (t1 card kept, 5-toggle wake card replaced by the
1-switch card) · `web/src/guide/GuidePage.test.tsx` (census 53 + t1 clause − 9 prune = **44**).
`git diff --check` + marker grep clean. AUDIT-CHECKLIST: CLASS 149/150 in place, `## CLASS NN`
prune-protocol section appended (number at merge).

Tails at a32ed9d6 (nice -n 19 ionice -c 3): `go build ./... && go vet ./...` rc 0 ·
`go test ./... -count=1` rc 0, 35 ok / 0 FAIL (store 71s, trader 339s, api 8.9s, kernel 1.4s) ·
`go test -race ./kernel ./store ./trader ./api` rc 0 (4.6s / 130s / 365s / 11.7s) ·
`tsc --noEmit` rc 0 · `vitest run` 72 files / 468 tests passed ·
`eslint src/guide src/components/strategy --max-warnings 0` rc 1 with **56 prettier/prettier
errors in five files this branch never touched** (GridConfigEditor, PublishSettingsEditor,
RiskControlEditor, TokenEstimateBar, scenarioEconomics.test.ts) — identical to origin/dev
(`git diff origin/dev` empty on them; the same command on a clean origin/dev checkout returns the
same 56, rc 1) — pre-existing dev debt, not this wave's; eslint on every file the branch touched
rc 0.
