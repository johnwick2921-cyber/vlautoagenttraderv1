# Dispatch 105: pinned source evidence and scope correction

Read-only audit by `/root/identity_source_evidence`, 2026-09-10. No production
files changed, no build or tests run, no database access. This document is the
only file written by this subtask.

Source inspected with `git show <revision>:<path> | nl -ba`:

- Running revision supplied by the parent lane: `770e2297d2188d09de0dcf76c3722e19e022e4c6`.
- Current fetched dev during this audit: `0975ef1111d5ec88ad509e2ded5aece046a9de7d`.
- [A] `git diff --stat 770e2297 origin/dev -- <all source files listed below>`
  was empty. The quoted source and last-change entries therefore apply at both
  revisions. This subtask did not independently read the running process or API;
  the parent owns that verification.
- [A] Tracked instructions are `docs/superpowers/CLAUDE-canon.md`. Root
  `AGENTS.md`, root `CLAUDE.md`, and `docs/superpowers/Codex-canon.md` are absent
  in this worktree. The tracked canon was read; its acquire-started keeper and
  prohibition on manual heartbeat take precedence over the stale copied text.

## C1 — no explicit scenario level identity: verified, with wording correction

[A] `kernel/plan_doc.go:52-97` defines these scenario fields:
`Economics`, `ID`, `Trigger`, `Condition`, `Direction`, `TargetChain`, `Invalid`,
`Confirm`, `Quality`, `Fvg`, `Breakdown`, `Confirm2`, `Consumed`, `ChainAfter`,
`Arm`. There is no candidate ID or level ID field.

Exact historical comment, `kernel/scenario_state.go:19-22`:

```go
// DESIGN CONSTRAINT that shapes everything below: PlanScenario carries NO price
// (kernel/plan_doc.go:29-37). Trigger and Invalid are free text the planner
// wrote ("sweep 21480 then reclaim", "2x5m < 21470"). To evaluate a scenario we
// must first decide WHICH LEVEL it is about, and that resolution is a heuristic.
```

The comment's literal **“NO price” is no longer accurate**. The schema contains
`Confirm.RefPrice` (`plan_doc.go:46-50`), arm entry/stop/target
(`plan_doc.go:106-117`), and FVG bounds (`plan_doc.go:248-255`). The valid premise
is **no explicit candidate identity**, not absence of structured prices.

[A] `ScenarioEval` (`scenario_state.go:43-57`) carries ID, Status, Anchor,
HasAnchor, Facts, Reason, and Basis, with no level ID. Its actual resolver also
has an existing structured FVG exception before prose matching
(`scenario_state.go:85-96`):

```go
func ScenarioAnchor(s PlanScenario, levels []PlanLevel) (float64, bool) {
    // FVG ENTRY MODEL (2026-08-26) — the scenario's anchor is the DISTAL edge
    // (long → fvg_lo, short → fvg_hi): the min-SL clearance leg must be measured
    // against the invalidation side, and the level-clearance rule then accepts
    // SL = distal ± 2 ticks naturally.
    if s.Fvg != nil && s.Fvg.Lo > 0 && s.Fvg.Hi > 0 {
        if strings.EqualFold(s.Fvg.Direction, "long") {
            return s.Fvg.Lo, true
        }
        return s.Fvg.Hi, true
    }
    for _, text := range []string{s.Trigger, s.Invalid} {
```

Indentation in quoted blocks is normalized for readability; quoted text and
expressions are unchanged.

## C3 — identity metadata is dropped from the plan: verified

[A] `kernel/plan_doc.go:25-36` has only Price, Label, Grade, Instruction,
MachineGrade. Kind, bounds, origin date, TF, formation, and ID are absent.
`PlanDoc.Levels` is `[]PlanLevel` (`plan_doc.go:258-264`).

[A] Model parsing uses a typed destination (`plan_doc.go:457-460`):

```go
var doc PlanDoc
if err := json.Unmarshal([]byte(js), &doc); err != nil {
    return nil, fmt.Errorf("plan JSON unmarshal: %w", err)
}
```

Machine grading stamps only `MachineGrade` by rounded price, not identity
(`trader/auto_trader_planner.go:1864-1872`). The published JSON is marshalled
from that typed document and assigned to `PlanDB.Doc`
(`auto_trader_planner.go:1976-1992`):

```go
docJSON, _ := json.Marshal(doc)
version, err := at.store.Plan().AppendPlan(&store.PlanDB{
    // other fields omitted from this excerpt
    Doc: string(docJSON),
})
```

[A] Ranked model row rendering contains price, label, grade, freshness, role,
and distance; no ID (`kernel/planner_prompt.go:545`):

```go
fmt.Fprintf(&b, "  %-9.2f %-20s grade %s  %-8s %-15s %s%.1f\n", l.Price, label, l.Grade, l.Fresh, role, sign, absF(l.Distance))
```

[A] The output schema similarly asks for price/label/grade/instruction, with no
identity (`planner_prompt.go:746`):

```go
fmt.Sprintf(`  "levels": [{"price": <n>, "label": "<PDH|ONH|nPOC…>", "grade": "A|B|C", "instruction": "<verb>"}],  // max %d, MUST include ≥3 below AND ≥3 above the current price`, maxL)
```

The scenario schema at `planner_prompt.go:747` has no `level_id` either.

## C5 — existing hash and persistence boundary: verified; two callers

[A] Exact implementation, `trader/research_snapshot.go:228-232`:

```go
func researchCandidateID(symbol string, l kernel.DetectedLevel) string {
    identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s|%d", symbol, l.Kind, l.Lo, l.Hi, l.OriginDate, l.TF, l.FormedAtMs)
    h := sha256.Sum256([]byte(identity))
    return hex.EncodeToString(h[:])
}
```

The direct production caller count is **two, not one**. C5's “single production
caller” premise is false; both calls are quoted here as the correction:

1. `recordResearchCandidatesAt`, `research_snapshot.go:22-32`, records
   `candidate:planner_read`, sets `stable_id` at line 31, and sets the explicit
   caveat at line 32:

   ```go
   f.Set("stable_id", researchCandidateID(symbol, l))
   f.Set("identity_basis", "symbol/kind/bounds/origin date/timeframe/formation; unknown formation may alias episodes")
   ```

   Its production entry is `trader/auto_trader_planner.go:2516`:

   ```go
   recordResearchCandidates(in.ResearchSnapshotID, symbol, researchRaw, scored, now)
   ```

2. `recordResearchEpisodesAt`, `research_snapshot.go:238-245`, records
   `candidate:prior_episodes` and sets `stable_id` at line 240. Its production
   entry is `trader/detector_record.go:79-81`:

   ```go
   eps := kernel.DetectTouchOutcomes(bars, lv.Price, k, delta, horizon, exitOn)
   if len(researchIDs) > 0 {
       recordResearchEpisodes(researchIDs[0], symbol, lv.DetectedLevel, eps, k, delta, horizon, exitOn, now)
   }
   ```

`git grep -n researchCandidateID 770e2297 -- '*.go'` returns exactly those two
calls and the definition. Both calls write research snapshot facts. The current
function returns a hash even when `FormedAtMs == 0`; unknown formation can
therefore alias otherwise identical reference episodes. This is the behavior
Dispatch 105's NULL identity rule must correct for its new identity contract.

[A] There is no candidate/level/stable ID field in the persisted plan struct,
`TouchOutcomeRow` (`store/touch_outcomes.go:22-111`), or `ArmedOrderDB`
(`store/armed_orders.go:20-100`). `PlanDB.Doc` is the typed serialized plan JSON
(`store/plan.go:34-60`, `auto_trader_planner.go:1976-1992`). Source-scoped
`git grep -n 'candidate_id\|level_id\|stable_id'` over these structs and the
planner writer returns no matches. This verifies the production schema/writer
boundary; it is not a substitute for the parent's live database census.

### D6 correction: formation alone does not make a row recomputable

[A] `TouchOutcomeRow` stores `Symbol`, `LevelPrice`, `LevelKind`, and
`FormedAtMs`, but does **not** store Lo, Hi, OriginDate, or TF. The writer
(`trader/detector_record.go:91-105`) copies only those available identity pieces.
Consequently, [B] an episode with a nonzero formation timestamp still cannot
recompute the specified seven-component hash **from episode columns alone**.
It needs a uniquely linked preserved candidate record carrying every input.
Otherwise classification must remain unrecomputable with the missing cause
named; guessing bounds/timeframe/origin would violate Dispatch 105.

## C6 — MapCandidate has no identity and the model consumes it: verified

[A] `kernel/map_candidates.go:57-89` contains Price, Names, Kinds, Grade, Score,
Fresh, MergedCount, MergedCredit, Distance, DistanceATR, HasATR, Role,
EntryCandidate, RefusedReason, Projection, and ProjectionMethod. No ID, primary
anchor identity, bounds, formation timestamp, origin date, or TF survives on
this render view.

[A] The strongest reference becomes the merge keeper: `map_candidates.go:145-152`
sorts descending score, then ascending price; lines 157-164 append merged names
and kinds; lines 169-179 initialize the keeper from `s.Price`, `s.Label`,
`s.Kind`, grade, score, freshness and distance. This is evidence for preserving
that existing keeper as the primary identity anchor, not changing ordering.

[A] The model consumes the exact selection/render path
(`kernel/planner_prompt.go:552-555`):

```go
if mb := RenderMapBlock(BuildMapCandidates(in.Levels, in.Price, in.ATR5m, MapCandidateOpts{}), in.Price); mb != "" {
    b.WriteString("\n")
    b.WriteString(mb)
}
```

`RenderMapBlock` renders merged rows and then `EntryShortlist(cs)`
(`map_candidates.go:318-357`). Neither row format at lines 343-345 nor shortlist
format at line 353 contains an ID.

## STOP: globally changing ScenarioAnchor would change a refusal gate

[A] The legacy anchor helper is shared by recording/display and execution.
All direct non-test callers at the pinned revision are:

| Caller | Location | Use |
|---|---|---|
| Plan API | `api/handler_plan.go:423` | Card anchor |
| MinSLAnchorFor | `kernel/min_sl.go:88` | Stop-clearance gate anchor |
| staleConfirmAnnotation | `kernel/plan_confirm.go:120` | FVG stale annotation |
| CitationStructure | `kernel/plan_render.go:299` | Entry structure classification |
| EvaluateScenario | `kernel/scenario_state.go:174` | Scenario evaluation |

`kernel/min_sl.go:84-91` directly returns `ScenarioAnchor` for the cited scenario.
The actual refusal caller is `kernel/engine_position.go:235-251`:

```go
// Leg 2 — level clearance for a cited scenario anchor.
if anchor, ok := MinSLAnchorFor(ctx, d); ok {
    tick := market.FuturesTickSize(d.Symbol)
    if tick <= 0 {
        tick = 0.25
    }
    clear := float64(MinSLTickClearance) * tick
    violated := false
    if d.Action == "open_long" {
        violated = d.StopLoss > anchor-clear
    } else {
        violated = d.StopLoss < anchor+clear
    }
    if violated {
        telemetry.IncGateBlock(ctx.TraderID, "min_sl_gate")
        return fmt.Errorf("sl_too_tight: stop %.2f does not clear the cited level %.2f by ≥%d tick(s) — widen or skip", d.StopLoss, anchor, MinSLTickClearance)
    }
}
```

[B] Making `ScenarioAnchor` globally ID-authoritative can therefore change
whether the production stop-clearance gate refuses. D4's “every which-level
consumer” requirement cannot be implemented literally at this shared helper
while also claiming A31's absolute no-gate-behavior-change limit. A separate
recording/display identity resolver could preserve the existing execution
anchor path, but that boundary needs the owner's corrected scope ruling before
implementation. No code was built to choose the ruling on the owner's behalf.

## Last-change provenance for every cited source file

Each entry below is the exact output of
`git log -1 --format='%H %s' 770e2297 -- <file>`. The source is unchanged at
dev `0975ef1111d5ec88ad509e2ded5aece046a9de7d`; where checked again with
`origin/dev`, the last-change output is identical.

| File | Last-change output |
|---|---|
| `docs/superpowers/CLAUDE-canon.md` | `290044296c482afdee04acd740d189b89bfd040d docs(canon): two different rc 3s sat on adjacent lines` |
| `kernel/plan_doc.go` | `d5e2414e0d30a275b6239f5d609f9e22f8e381d7 feat(scenario-economics): enforce new authoring contract and preserve legacy unknowns` |
| `kernel/scenario_state.go` | `2eaf7ab59ff1cf89ce88d0317d71a3f3390eff74 FVG entry model — 5th scenario condition (pure-math play) (#79)` |
| `kernel/map_candidates.go` | `35fa69cc8b54d3493f544fe9ca5e7c075f282b99 candidates(D6 card): the merged map reaches the plan card` |
| `kernel/planner_prompt.go` | `a59b6c9d6a0cd1ee9f6409e2542e699adb0d024a merge 35fa69cc into the combined boot head — 103 candidates` |
| `trader/research_snapshot.go` | `0babd0902d1bb74e5f44fce1fc4350e42a8cbdeb feat(research): link attempts, permissions, broker receipts and corrected outcomes` |
| `trader/auto_trader_planner.go` | `c6f75756f3e54a56646ca3cfa9c86f116b5d4541 feat(W1 4/n): wire the link — the recorder stamps it, the gate keeps it wired` |
| `trader/detector_record.go` | `c6f75756f3e54a56646ca3cfa9c86f116b5d4541 feat(W1 4/n): wire the link — the recorder stamps it, the gate keeps it wired` |
| `store/touch_outcomes.go` | `c35dfecb2e11cbc763174516983919885cf0b4cc feat(W1 2/n): the touch → scenario link as a HEURISTIC, and the boot line that names its resolver` |
| `store/armed_orders.go` | `c11632c38be236663c2a065d20f750e30c3316f1 fix(settlement): four words in the ledger were written from something other than the broker's answer` |
| `store/plan.go` | `4e901261778399f2e09e3f7f9b7e5f4168afae8b feat(plan): a lifecycle log, so trigger_reason stops answering the wrong question` |
| `api/handler_plan.go` | `35fa69cc8b54d3493f544fe9ca5e7c075f282b99 candidates(D6 card): the merged map reaches the plan card` |
| `kernel/min_sl.go` | `4657560bbaf616fcde7457816da5b8aff22431dc fix(0B): exit sanity — stop floor 1.5xATR, stop anchored to seated structure, BE+trail suspended, size 1, re-arm after boot sweep` |
| `kernel/plan_confirm.go` | `b99963857c453f909570eba7a854a8ec23eed087 test(confirmation): pin the ruled carve-out and publish mutation evidence` |
| `kernel/plan_render.go` | `b71d424c0435806dbb30f8a1845ac817f8834e7f docs+fix(hygiene): the owed small fixes — checklist backfill, bias aliases, no-trade NOTES, guide corrections to RESOLVED values` |
| `kernel/engine_position.go` | `0b8b41c3704570eccfc4d918d26cbc880ac4c897 fix4+8k [F2]: honest C6 — GateRefusalError never parse-retried, planless prompt warning, risk_check_error logged; [F6] executor bias_ctx PDC from full universe` |
