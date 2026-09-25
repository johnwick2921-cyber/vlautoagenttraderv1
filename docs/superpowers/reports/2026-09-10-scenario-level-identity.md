# Dispatch 105 — scenario-level identity: corrected premises and implementation

> **OPEN POST-BOOT DATA INCIDENT:** the shared chart/planner ring contains
> ~292-point discontinuities after the MNQ September→December subscription
> change. Binary identity is verified; market-data integrity is NOT certified.
> See [read-only investigation](2026-09-10-identity-chart-discontinuity.md).

**Current status: implementation built and verified on the owner's corrected
scope; ready for review and a separate cutover GO. Not deployed.** The owner approved
separate `formed_close_ms`, unchanged evaluator/gate/wake inputs, complete-input
backfills and WARN-first authoring. The original STOP audit below remains the
historical evidence for that correction; its unperformed-work section describes
the pre-approval state. The implementation addendum supersedes that status.


**Lane:** `fix/scenario-level-identity`, session
`scenario-identity-ac5a801d/root[unlisted]`. Claimed with `deploy/nofx-claim.sh new`,
including the expected file footprint. The branch's observed `git ls-remote`
SHA immediately after acceptance was `8e8965be22fe60e0f6cfc94c616bf546fce6998f`;
that observation is not a durable claim identifier. The branch is the claim.

**[A] Running source:** `770e2297d2188d09de0dcf76c3722e19e022e4c6`.
At **2026-09-10 17:32:15 CT**, `/api/health` returned revision
`770e2297d218`, status `ok`; `/proc/2605112/exe` independently reported that full
revision and **`vcs.modified=false`**. Systemd reports process start
**17:10:33 CT** and `Restart=on-failure`. Receipt:
[runtime.json](2026-09-10-scenario-level-identity-data/runtime.json).

**[A] Source isolation and freshness:** the claimed worktree began at the actual
dev tip `0975ef1111d5ec88ad509e2ded5aece046a9de7d`; a second locked, read-only
worktree was pinned to the running revision for the database census. Source
excerpts below refer to that running revision. Dev advanced during the audit to
`5b5106e7d83002a183bb14091ef0b6217c2adf8c`; the claim was rebased before
publication. Before rebase, `git diff --stat origin/dev` showed 83 apparent
deletions in another lane's checklist/report additions. After rebase it showed
none. Those changes were retained. No main-tree operation or deploy lock was
needed for this report.

Before publication, cleanup batch 1 landed at
`29eda2b39d8d6e707e455838f449fe0df623070d`; this report branch was rebased again
onto that actual merged dev tip. Its diff against dev contains only this report
and its evidence files, with no deletions. The final two-format `uniq -c`
[census](2026-09-10-scenario-level-identity-data/checklist-census.txt) reads
**highest occupied 114**, not the dispatch's earlier 112. No class was assigned
or reserved: this is a correction report, not a fix. The final checklist
last-change line was:

```text
2ee867afdb9b674a7caeaad133f2b1df7b38347b 2026-09-10T17:32:19-05:00 docs(checklist): class 114 — the verifier is wrong, and its wrongness reads as a result
```

The standing census command's before-first-CLASS boundary also admitted PART 3
step **0**, whose bold title ends in a period. The receipt retains that initial
output and the corrected reader bounded to PART 1: **67 PART 1 entries + 29
CLASS headings**, with no extra heading formats found. Both give highest 114;
the five existing collisions are 75/76/77/92/93. This reader error is recorded,
not repaired in the shared checklist. The newer tracked canon was also read;
its last change is `07b53e6500c18b2531ec30a78675393cf09caef4` (2026-09-10
17:31:05 CT, “cleanup batch 1 — B5 the mutation harness that cannot fake a
verdict, B6 the worktree-add check”).

Evidence labels: **[A]** directly read/measured; **[B]** consequence inferred
from the verified call chain. The audit follows
`docs/superpowers/AUDIT-CHECKLIST.md`, PART 2 R1–R10 and PART 3, with the dispatch's
newer lock/build instructions taking precedence. Its last change read after
rebase was:

```text
2227e926e7e9e771122138f4f5ea078e1366b037 2026-09-10T17:28:42-05:00 docs(checklist): class 113 — a gate that certifies a name, not a path
```

## Corrections required before D1–D6 can be implemented

1. **Formation open and formation close are different facts.** [A]
   `kernel/levels.go:86-90` defines existing `FormedAtMs` as the **bar open**
   of the candle completing a pattern. S/D, FVG, iFVG and OB actually write
   `OpenTime`. `trader/auto_trader_wake_levels.go:102-159` reads that same field
   for plan-age comparisons and wake keys. [B] Replacing it with close time can
   change wakes, prohibited by A31. Adding TF to existing blank line-level
   `TF` fields can also change deduplication/grading. The recording seam must
   preserve these existing inputs; close/availability metadata needs an
   explicitly separate meaning. No such reinterpretation was made here.

2. **D4's shared evaluator is a trading consumer.** [A] The actual chain is
   `entryGateForArm -> scenarioInvalidationResolver -> scenarioInvalidationAt
   -> EvaluatePlanScenarios -> EvaluateScenario -> ScenarioAnchor`. The facts
   evaluated at that anchor determine the invalidation refusal. Another path,
   `kernel/engine_position.go -> MinSLAnchorFor -> ScenarioAnchor`, determines
   stop clearance. [B] An ID price winning over the current anchor can change
   refusal outcomes without editing either gate file. To keep A31, identity
   should be authoritative for the recorded/displayed candidate while the
   existing evaluation anchor, facts, verdicts and stop inputs remain unchanged.
   If D4 means changing those decisions, it needs an explicit scope amendment.

3. **W-TF did not make formation part of deduplication identity.** [A]
   `kernel/levels_assemble.go:272` compares kind, TF and price within one tick:

   ```go
   if o.Kind == l.Kind && o.TF == l.TF && math.Abs(o.Price-l.Price) <= dedupeTick {
   ```

   Formation is absent. The hash can distinguish two raw formations, but the
   existing selection can still discard one. This wave must preserve that
   selection under A31 and must not claim both will survive onto the map.

4. **PDH/PDL/PDC currently mean prior calendar day.** [A]
   `kernel/levels_multiday.go:38-41,68-85` groups by CT calendar date, not CME
   session. Stamping a CME prior-session close would misdescribe the actual
   contributing bars. Capture the real existing bucket's completion evidence;
   changing its definition is outside this wave. OR/IB and overnight families
   also emit while developing. Their future final-window times cannot be
   recorded as already-known births; incomplete or unsupported completion stays
   NULL. Swing pivot time similarly differs from its later confirmation time.

5. **Formation alone does not establish backfill recomputability.** [A]
   `touch_outcomes` lacks the hash's bounds, origin date and timeframe. A
   nonzero formation cannot reconstruct them. An exact archived `raw_origin`
   match may supply the missing inputs; otherwise D6 needs an honest
   `unrecomputable:missing_identity_inputs` result in addition to
   `unrecomputable:no_formation`. Never infer the missing fields from price or a
   label. A candidate ID also cannot replace the closer's scenario-ID key:
   multiple scenarios may name one candidate, so that join must preserve
   scenario/version attribution and explicitly represent ambiguity.

These are source and contract conflicts, not requests to expand this wave into
detector, gate, wake, scoring or seating repairs. Full call chains and exact
`git log -1` receipts for every cited source file are in
[source evidence](2026-09-10-scenario-level-identity-data/source-evidence.md),
[episode evidence](2026-09-10-scenario-level-identity-data/episode-evidence.md), and
[formation evidence](2026-09-10-scenario-level-identity-data/formation-evidence.md).

## C1 — scenarios lack candidate identity: PROVEN, with a comment correction

[A] At the running revision, `kernel/plan_doc.go:52-97` carries:

```text
Economics, ID, Trigger, Condition, Direction, TargetChain, Invalid, Confirm,
Quality, Fvg, Breakdown, Confirm2, Consumed, ChainAfter, Arm
```

No field names a candidate ID. The requested historical comment remains at
`kernel/scenario_state.go:19-22`:

```go
// DESIGN CONSTRAINT that shapes everything below: PlanScenario carries NO price
// (kernel/plan_doc.go:29-37). Trigger and Invalid are free text the planner
// wrote ("sweep 21480 then reclaim", "2x5m < 21470"). To evaluate a scenario we
// must first decide WHICH LEVEL it is about, and that resolution is a heuristic.
```

The literal “NO price” is stale: structured confirm, arm and FVG prices exist.
The verified premise is **no candidate identity**. [A] The stored-plan census
has **299 versions / 852 scenarios**, **0 scenarios carrying `level_id`**.
All `(plan_id, version)` pairs are in `census.json` under `plans.ids`.

## C2 — link basis since W1's third boot: measured, not a guessed success rate

[A] Marker `580e88b39412f6b8cb77021f31651509ccfc1e4d` records the third boot of
`adae3bb41b3151e12c9c7e0553f522860fccbea4` at **16:51:28 CT**. Fresh journal
reads independently show its integrity line at 16:51:31 CT. The cohort uses
`touch_outcomes.created_at >= 2026-09-10T21:51:28Z`, because the recorder may scan
historical tape; filtering by episode-open time would answer a different
question. `store/touch_outcomes.go:183-185` stamps write time in `SaveOutcome`.

At **17:32:50 CT**, the cohort contains **32 rows, IDs 5051–5082 inclusive**:

| ScenarioLinkBasis | n | Row IDs |
|---|---:|---|
| `price_proximity` | 0 | `[]` |
| `unresolved:two_scenarios_within_band` | 0 | `[]` |
| `unresolved:nearest_outside_band` | 0 | `[]` |
| `unresolved:no_scenario_at_seat` | 32 | 5051–5082 |
| NULL/empty/other | 0 | `[]` |

**0/32 linked by proximity; 32/32 unresolved.** This sample does not establish
that the nearest guess was right or wrong: no row in it has a linked scenario.
The earlier 17:29:46 CT census was empty; the planner read arriving during this
audit supplied the final sample. No read or plan was manually triggered.

For context only, across all **5,082** rows: basis NULL on IDs **1–4860**,
`unresolved:no_scenario_at_seat` on **4861–5082** (222 rows); no other basis.
This historical population is not substituted for the requested boot cohort.

[A] W1 `ScenarioNearest` is a nullable **scenario ID on a touch row**. It is not
the evaluator's anchor price. W1 reads `Confirm.RefPrice`, then enabled
`Arm.Entry`, and links only when exactly one anchor is within
`LevelClusterTicks * 0.25 = 3.00` points. The evaluator separately prioritizes
the FVG distal edge, then prose-to-plan snapping within **2.00** points. The
map/card has a third nearest-candidate lookup within **3.00** points. D4 must
keep those types and decision uses distinct.

## C3 — PlanLevel drops identity: PROVEN

[A] `PlanLevel` (`kernel/plan_doc.go:25-36`) retains exactly **Price, Label,
Grade, Instruction, MachineGrade**. Kind, Lo/Hi, OriginDate, TF, FormedAtMs and
ID do not survive typed JSON parsing and publication. The 299 stored versions
have no such identity keys in their `levels` entries; the census records the
empty key-count map instead of inventing IDs.

[A] `kernel/planner_prompt.go:545` renders price, label, grade, freshness, role
and distance. Its schema at lines 746–747 exposes no candidate-ID column or
scenario `level_id`. The model has no stable candidate identifier to cite.

## C4 — formation coverage: corrected denominator and a fresh W-TF sample

[A] `touch_outcomes`, **17:32:50 CT**, all 5,082 row IDs listed by category in
[census.json](2026-09-10-scenario-level-identity-data/census.json):

| Stored formation state | n / 5,082 | Meaning |
|---|---:|---|
| Positive timestamp | 183 / 5,082 = **3.60%** | Existing captured formation value |
| Zero | 4,222 / 5,082 | Existing writer's “no formation” sentinel |
| SQL NULL | 677 / 5,082 | Older uncaptured column |
| Negative | 0 / 5,082 | None observed |
| SQL non-NULL | 4,405 / 5,082 = **86.68%** | Includes all 4,222 unknown zeros; not formation coverage |

The dispatch's **3.8%** described an older 4,860-row denominator
(`183/4860 = 3.77%`). It is neither the current positive rate nor the literal
SQL non-NULL rate. `FormedAtMs == 0` means unknown in the running Go type; this
wave's future NULL contract must not recast those zeros as real timestamps.
The positive-row IDs begin 732 and end 4839 and are enumerated in the artifact;
no contiguous-range claim is made for them.

Since W-TF process start **17:10:33 CT**, there are **32 new touch rows**,
IDs **5051–5082**, **0 positive, 32 zero, 0 NULL** formation values. Touch rows
have no TF column, so a per-TF touch breakdown cannot be manufactured.

[A] The research archive supplies the richer forward sample:
**1,130 `candidate:planner_read` records**, IDs **14780870–14781999 inclusive**,
all written by `770e2297d2188d09de0dcf76c3722e19e022e4c6`. These are raw
candidate records, not distinct surviving map seats. They contain **789/1130
(69.82%)** positive `formation_ms`; **341/1130** are NULL.

| Recorded TF | Candidate records | Positive formation | NULL formation |
|---|---:|---:|---:|
| 15m | 71 | 31 | 40 |
| 1h | 84 | 45 | 39 |
| 4h | 121 | 86 | 35 |
| 1d | 64 | 32 | 32 |
| 5m | 6 | 0 | 6 |
| UNKNOWN | 784 | 595 | 189 |
| **Total** | **1,130** | **789** | **341** |

UNKNOWN TF is kept UNKNOWN, never inferred to be 1m. All row IDs per TF and
formation category are in `archive_candidates_since_wtf.by_event.planner_read`.
An additional **12 `candidate:prior_episodes`** records, IDs
**14780858–14780869**, are reported separately; they are not newly emitted map
levels and do not enter the table above.

**W-TF's forward capture is now observed, but it did not fill line formation.**
All 789 positive values belong to existing zone families: DEMAND **58/58**,
SUPPLY **63/63**, FVG **33/33**, IFVG **256/256**, OB **379/379**. EQH **0/142**
and EQL **0/126** remain unformed. In the new sample, PDH/PDL/PDC **0/1 each**,
ONH/ONL **0/1 each**, OR-H/OR-L **0/1 each**, IB-H/IB-L **0/3 each**, VWAP
**0/3**, VWAP±2σ **0/2**, and RN **0/29** have no formation. IDs for every
family are in the artifact, not inferred from labels.

The source confirms W-TF's decorator assigns TF and LookbackBars; it does not
assign formation or a stored age field. The existing zone timestamps retain
their documented open-time meaning.

### Formation-capture matrix — recording seams, not detector changes

| Family | Captured now | Evidence a recording seam could retain |
|---|---|---|
| S/D, FVG/iFVG, OB | Completing candle **open** | Preserve existing timestamp; close and availability must be separate facts. |
| PDH/PDL/PDC | No | Actual final bar of the existing **calendar-day** bucket; not a substituted CME-session boundary. |
| RTH-H/L | No | Actual final contributing bar under existing NY registry membership. |
| AS/LDN/ON H/L | No | Actual completed existing window; NULL while developing or terminal evidence absent. |
| OR H/L | No | Actual completed 08:30–08:35 window; no future 08:35 birth on a developing row. |
| IB H/L/extensions | No | Actual completed 08:30–09:30 window; same developing constraint. |
| VWAP / bands | No | Existing session anchor at 17:00; anchor time and first computability differ. |
| eVWAP | No | Existing 15:00 anchor, not session VWAP's 17:00 anchor. |
| POC/VAH/VAL, pdVWAP | No | Final actual prior-session bar and cached output metadata. |
| nPOC | No | Internal `birth` already holds a bar close but is dropped at output. |
| SETT | No | Same final prior-session bar currently supplying the price. |
| MID-O | No | Existing contributing bars; current inclusive 08:30-open boundary prevents blindly calling 08:30 its completion. |
| Gap | No | Existing completing bar available at output. |
| EQH/EQL | No | Carry pivot/confirmation metadata through unchanged clustering, or NULL; final price-only arrays cannot recover it. |
| SWG H/L | TF only | Pivot close exists internally; first availability requires later confirmation bars. |
| RN | No, by nature | NULL formation and NULL ID, with reason. |
| Owner / other unknown origin | Not established by this sample | NULL unless actual source lineage establishes formation; no inferred timestamp. |

The expanded matrix, source line numbers and complete freshness ledger are in
[formation-evidence.md](2026-09-10-scenario-level-identity-data/formation-evidence.md).

## C5 — hash exists, but “wired once” is false

[A] `trader/research_snapshot.go:228-232`:

```go
func researchCandidateID(symbol string, l kernel.DetectedLevel) string {
    identity := fmt.Sprintf("%s|%s|%g|%g|%s|%s|%d", symbol, l.Kind, l.Lo, l.Hi, l.OriginDate, l.TF, l.FormedAtMs)
    h := sha256.Sum256([]byte(identity))
    return hex.EncodeToString(h[:])
}
```

[A] There are **two production calls**, both archive recording:
`recordResearchCandidatesAt` at line **31** (`candidate:planner_read`), and
`recordResearchEpisodesAt` at line **240** (`candidate:prior_episodes`). The
first explicitly records:

```text
symbol/kind/bounds/origin date/timeframe/formation; unknown formation may alias episodes
```

The current helper hashes zero formation; it does not implement the proposed
NULL-ID rule. The new safe resolver must not promote those old archive hashes
to established identity for unknown formations. [A] The Stage A pinned source
at `954f11b1` has the same two-call research wiring.

[A] No candidate identity is persisted in typed `plans.doc`, `touch_outcomes`
or `armed_orders`. Their unrelated primary keys are not candidate IDs. The
archive stores `stable_id` and full `raw_origin`; episode rows do not carry all
hash inputs. D6's recomputed/unrecomputable classification has therefore not
been built or run. No backfill counts are claimed for this unimplemented wave.

## C6 — merged candidates lack an ID: PROVEN

[A] `kernel/map_candidates.go:57-89` defines **Price, Names, Kinds, Grade,
Score, Fresh, MergedCount, MergedCredit, Distance, DistanceATR, HasATR, Role,
EntryCandidate, RefusedReason, Projection, ProjectionMethod**. The dispatch's
listed fields were incomplete. It has no ID,
TF, formation or retained primary-reference identity object.
`kernel/planner_prompt.go:552` renders
`RenderMapBlock(BuildMapCandidates(...), ...)`; these are the candidates the
model sees. W3's merging is a display/selection operation that must keep its
existing order, anchor choice, width and scores under A31.

## Basis publication and spec freshness

The exact basis text was read at these pins, and fetched again from immutable
raw URLs. **Every HTTP result below was 200, its download size equalled
`git ls-tree -l`, and the downloaded bytes equalled the local Git blob.**
Full URLs, SHA256s, command receipts and last-change lines are in
[basis-receipts.json](2026-09-10-scenario-level-identity-data/basis-receipts.json).

| Basis | Read pin | Bytes | Last-change commit at that pin |
|---|---|---:|---|
| `2026-09-10-episode-contract.md` | `770e2297d2188d09de0dcf76c3722e19e022e4c6` | 33,143 | `5dd0e8c7f9ef32038bf5f2574e63b2f02b84633c` |
| `2026-09-09-candidates-not-entitlements.md` | same running pin | 9,269 | `d0df134d8badc10704b178442111338a179e3bd1` |
| `2026-09-10-every-detector-every-timeframe.md` | same running pin | 14,489 | `050cd5c1549b656ed06f6eaae8954b55a723ede4` |
| `SYSTEM-MAP.md` | same running pin | 85,621 | `8d10b55b249de36289978aa7f0d83be8bd244849` |
| `VL-TRADING-RULEBOOK-v1.md`, §A/§C | `daeb654978b0c739592e8589a393c5e79171560d` | 36,422 | same rulebook pin |

The named W1 “next wave's basis” heading is absent at the running pin; the
relevant facts are present in its link discussion and were verified against
source rather than quoted as an invented section. The corrected rulebook is
**not on dev at the observed `5b5106e7` tip**. It exists on
`docs/rulebook-trading-corrections-20260910`; its text says canonical effect
begins when PR #99 merges. This report does not transplant it or claim that
merge occurred. The required future §A edit must follow that document's real
landing; §B is untouched.

## Unperformed work and the concrete corrected scope

No E1–E11 implementation fixtures or mutations were run: A23 stopped the build
before production changes. There is no red-to-green claim. The read-only
census used individually consistent SQLite transactions with `mode=ro` and
`PRAGMA query_only=ON`, fixed boot cutoffs, explicit IDs and a query deadline.
Main and archive snapshots are separate, not falsely described as atomic.
Their query source is [census.py](2026-09-10-scenario-level-identity-data/census.py).

The proposed scope for owner confirmation is:

- Preserve existing `FormedAtMs`, TF, detector definitions, dedupe and all
  gate/wake inputs; add separately defined recording metadata where needed.
- Make ID authoritative for candidate attribution on records/cards; keep the
  trading evaluator's existing anchor and decisions. Record disagreement
  between those two explicitly.
- Keep line formations tied to the actual current detector buckets; leave
  developing/unknown formation NULL. Do not hash a guessed timestamp.
- Backfill only with every exact hash input; preserve legacy scenario NULLs
  and name missing-input reasons rather than asserting recomputability.
- D3 remains **WARN and counted**, accepted for this boot and next; no REFUSE
  until the owner rules after at least five measured plans. There is no new
  refusal in this correction report.

Legacy attribution remains a guess by construction. No identity boot line,
named/unnamed after-count, first authored-ID plan, or live card proof exists
yet. All are owed after corrected implementation and the owner's separate GO.
No database mutation means no migration backup was required here. No binary,
RELEASE, Guide source, dist, account, order or process was changed; rollback
of this report requires only reverting its documentation commit. A future
cutover must perform the full backup, merged suite, clean-clone build, source
Guide stamp, fresh gate, RELEASE/swap/VERIFY sequence and then observe live
proof. This report is not a deploy authorization or a five-leg gate result.


## Implementation addendum — owner-approved corrected scope

**[A] Branch:** `fix/scenario-level-identity`; isolated locked worktree
`/tmp/nofx-scenario-level-identity`. Rebased onto dev
`08aea8b81124bab5f1e18e9eccd76e35f108668c`, preserving the intervening collapse-keeps-names
changes. The deletion comparison now includes no deletions of another lane's
files. No main-tree edit, deploy lock, live DB write or boot occurred.

### Contract and separation

- `levelidentity.ID` is the sole seven-input hash implementation. Formation is
  the separately captured close. Every missing input names its reason and returns
  NULL; no zero-formation or partial hash. Bounds must be present and finite.
- `DetectedLevel.FormedAtMs`, existing `TF` and all trading inputs stay unchanged.
  `formed_close_ms` rides separately into the frozen plan map and research record.
  PlanLevel metadata comes from the machine map, not model-supplied extra fields.
- `LevelByID` validates a stored ID against every frozen input. Cards, desk,
  evaluation recording and episode attribution use that resolver. The existing
  `ScenarioAnchor` and `EvaluateScenario` code is unchanged. The evaluator's own
  anchor remains beside the resolved candidate; > existing merge width is a
  recorded disagreement, never a decision change.
- `StampAuthoredIdentity` is called after ordinary accepted-plan validation.
  Missing and unknown IDs are WARN plus recorded event, not a refusal. Events
  are scoped to trader/plan/version/scenario. Disagreement counts are unique
  version/scenario observations; polling does not increase them. Counter-write
  errors warn, and recording panics are contained.
- `touch_outcomes.level_id` is the sole added episode column. Existing ordinal,
  k, delta, horizon, formation-open scan floor, watermark and proximity fields
  retain their meanings. Exact primary references and merged members recorded by the existing merge
  receive the named primary ID; unnamed or changed references remain NULL.
  Membership is stored as exact source IDs, never reconstructed by price. A candidate named by two
  scenarios remains ambiguous at the scenario join. A named row cannot read a
  different active plan version's scenario facts merely because both are S1.
- Backfill population is **episode rows**, not inferred legacy scenario counts.
  Before the measured W-TF creation cutoff, rows are untouched. Later rows need
  all seven inputs from an already-named frozen candidate in the exact plan
  version. Missing fields are `unrecomputable:<fields>`. Legacy plan documents
  are never rewritten. Classification rows, with IDs, are preserved in the
  trader-scoped `level_identity_backfill:` sidecar. No live migration was run.

**[A] Formation output matrix:** prior-calendar PDH/PDL/PDC and NY-subset RTH
use the actual last source close of their existing buckets, not an invented
CME boundary. AS/London/overnight and OR/IB require an actual completed-window
last close; developing or missing-final-bar outputs retain NULL. Weekly/monthly
references and profile/pdVWAP/nPOC use the existing bucket's last source close.
Session/eVWAP use their explicitly defined anchor only when its source bar is
present. Gap, S/D, FVG, iFVG, OB and swings carry their source-completion close;
iFVG records inversion close separately from the unchanged original-gap open.
Round numbers, owner references and durable legacy extras without source
formation remain NULL. Lookback is recorded alongside, not substituted for TF.

### Scope correction approved during implementation

The full Go run found the brand wave's `TestExistingGoImportTargetsPreserved`
rejecting removal of obsolete `crypto/sha256` and `encoding/hex` imports from
`trader/research_snapshot.go`. Moving the hash to the shared helper made these
imports unused. The guard compared all future changes to `954f11b1`, extending
that wave's import constraint indefinitely. The owner explicitly approved
narrowing it to protect `nofx/...`; the existing rename-to-`vl/...` rejection
pin remains and a standard-library-removal pin was added. No existing project
import target or Go module path was renamed. Evidence:
[initial full Go result](2026-09-10-scenario-level-identity-data/go-before-guard-correction.txt).

The corrected RULEBOOK was absent from dev. Its unchanged baseline is carried
from `daeb654978b0c739592e8589a393c5e79171560d` (branch
`docs/rulebook-trading-corrections-20260910`); only the identity note in §A is
this lane's addition. No §B policy is authored or amended here. Guide and
SYSTEM-MAP plan/levels notes accompany it. All three explicitly distinguish
source implementation from live proof.

### Verification record

**[A] E1/E5 original red:** the actual parser discarded `level_id`; the OR emitter
had no separate close. See [red output](2026-09-10-scenario-level-identity-data/E1-E5-red.txt).
Compilation failures while writing fixtures are not counted as semantic red pins.

**[A] Green pins:** actual accepted planner write → frozen map → `LevelByID` →
production episode row; missing and unknown IDs remain ACTIVE and counted;
different closes yield different IDs and every missing hash input yields NULL;
P versus P+5 preserves evaluator output and records attribution disagreement;
legacy stored plan reads NULL; PDH prior calendar source close, OR 08:35, IB
09:30 and existing extensions, VWAP source anchor, and round-number NULL are
pinned with explicit clocks. The production-path episode test also recomputes
its IDs from complete frozen inputs. UI pins cover legacy NULL, unknown WARN
and the distinct candidate/evaluator prices.

**[A] Behavior golden:** generated on the exact dev baseline `08aea8b8` in a
separate locked worktree; 302,824 bytes. The current production detector → dedupe
→ scorer → seat output and old map text compare byte-for-byte. Golden:
`kernel/testdata/identity_legacy_output.json`; test
`TestIdentityLegacyOutputParity`. This is a deterministic fixture proof, not a
claim that every possible tape has been enumerated.

**[A] E7:** removing only the added `id=` column from the new model map text
returns the previous text exactly. The other planner prompt change is the new
scenario schema field; no authoring policy instruction was changed.

**[A] E8:** the AST wiring gate names actual production files and counts their
calls; declarations and tests cannot satisfy it. [Caller receipts and parity
result](2026-09-10-scenario-level-identity-data/parity-wiring.txt).

**[A] E10:** seven mutations, seven confirmed source replacements, seven
successful package builds, seven failing pins. The mutations omit formation
from the hash, turn WARN into REFUSE, substitute the evaluator anchor for the
ID's candidate price, and disconnect episode identity, plan stamping, boot
logging and formation capture. Each artifact quotes BEFORE/AFTER and the test
failure: [manifest](2026-09-10-scenario-level-identity-data/mutations.json).
The first authority run had an overly strict script confirmation check; it is
not counted. The corrected confirmation checks the exact mutated file, and
all mutations were restored before final suites.

**[A] Frontend:** full vitest 59 files / 417 tests PASS; `tsc --noEmit` PASS.
The sandboxed frontend run could not spawn the existing brand test's Go
subprocess (EPERM); rerunning with that execution permission passed, without
changing the test. [Vitest output](2026-09-10-scenario-level-identity-data/vitest-full.txt).
Full Go final result and final tested commit are recorded below when complete.

### Remaining cutover proof and rollback

This addendum claims no live named-plan count, no live backfill count and no
identity boot. Those require the owner's separate GO and a fresh gate. The
boot reader is trader-bound; unavailable map/counter/backfill data prints n/a,
not a plausible zero. Legacy IDs remain NULL by design. Guide source revision
must be stamped from the clean-clone binary before dist. Checklist class number
is assigned at merge; none is reserved here.

Cutover still owes backup + integrity check before any migration, full suite at
the merged head, clean clone named `nofx`, binary `vcs.modified=false`, source
GUIDE_BUILT_REV then dist, own five-leg gate, RELEASE → mv → VERIFY, permitted
restart, owner boot acknowledgment, pushed same-tree marker and five-reference
verification before lock release. Preserve the old binary by its actual embedded
revision. Rollback restores that binary and matching RELEASE/dist/Guide marker;
the additive nullable column and versioned recording sidecars may remain.
No account, order, execution or detector policy changes are part of this wave.

[Implementation source freshness receipts](2026-09-10-scenario-level-identity-data/implementation-source-receipts.json)
quote `git log -1` against the dev base for each existing touched file. Original
running-revision evidence and its sample IDs remain in C1–C6 above.


### W2 integration checkpoint

**[A]** The first restored full Go suite passed at
`aa145338cae425afdd09e09aa06e49726a4eaf6e`. Dev then acquired W2 through
`de26d1e4867b2b1e356b7c90d7b93305060cff41`, so that run is not claimed as W2
integration proof. Rebase retained W2's fade episode fields, production stamp,
API data, chip, Guide and class 115. Two textual conflicts were resolved by
retaining both lanes: `SessionPlanCard.tsx` props and the checklist tail. The
identity lane did not author or modify `kernel/fade_permission.go` or the fade
stamp. A subsequent merged-source suite and build are required below.

The recorded formation TF now reads the existing `AISVPBarInterval` constant
for the base series. Exact merge membership is captured as `source_ids` beside
the primary identity; a merged member can attach to that named primary through
those recorded IDs. No later proximity guess, merge width change or score
change is involved. The legacy production-output golden still pins the output.

**Test chronology boundary:** E1 parser and E5 OR pins were run red before their
implementation. E2/E3/E4 have confirmed build-green mutation reds after the core
was introduced, followed by restored green runs; this report does not represent
those as pre-code baseline executions.


### Final prepared-candidate verification

**[A] Code candidate:** `b30afc6570936a628c8cef1f462dc9e8dd8216d9`, based on
`de26d1e4867b2b1e356b7c90d7b93305060cff41` (W2 included). The restored **full
`go test ./...` passes** on this combined source; kernel 1.296s, trader 133.945s.
[Full Go receipt](2026-09-10-scenario-level-identity-data/go-w2-final.txt).
The complete frontend suite after W2 integration and the binary-derived Guide
stamp passes **60 files / 421 tests**. `npm run build` passes both TypeScript
and Vite; Vite's existing chunk-size warning is advisory, not a failed build.
[Frontend receipt](2026-09-10-scenario-level-identity-data/vitest-w2-full.txt).

**[A] Clean clone:** `/tmp/identity-build/nofx`, porcelain-clean before `go build`.
The built `/tmp/identity-build/nofx-bin` reports:

```
vcs.revision=b30afc6570936a628c8cef1f462dc9e8dd8216d9
vcs.modified=false
MD5=42bf7006f44b6a0f1f7e4ca4137df8df
SHA256=169827b0621a809d8ba5bdfe34f014f909205d36fa7b9550b5567ba4d2975694
```

The Guide SOURCE in this branch and the build clone was then stamped to that
exact binary revision, **before** building dist. The subsequent documentation
and Guide-stamp commit is not represented as the binary's revision. Dist has
92 files, with [a file-by-file SHA256 manifest](2026-09-10-scenario-level-identity-data/candidate-dist-manifest.json).
[Build information](2026-09-10-scenario-level-identity-data/candidate-buildinfo.txt)
and [dist build output](2026-09-10-scenario-level-identity-data/candidate-dist-build.txt).
These are prepared artifacts, not live references.

**[A] Scope/deletion review:** compared against the W2 base, no changes to
`kernel/scenario_state.go`, `kernel/entry_gate.go`, `trader/armed_executor.go`,
`kernel/levels_score.go`, `kernel/fade_permission.go`, or W2's fade stamp wiring.
The detector changes are output metadata; baseline output parity passes.
No file from W2 or the collapse-keeps-names lane is deleted. The module remains
`nofx`; the new `nofx/levelidentity` import is additive. Standard-library hash
imports moved out of the research writer under the explicitly approved guard
correction. E8 production-call receipts remain available above.

**A15, test fixture observation:** the pre-existing `recorderFixture` in
`trader/detector_formation_test.go` returns `time.Now()` even when given a fixed
`endAt`; last changed at `c6f75756f3e54a56646ca3cfa9c86f116b5d4541`.
105's new integration fixture supplies its own fixed clock and does not use
that helper. No failure attributable to this observation was measured here,
and it was not changed under this wave.

**Still not claimed:** a merge/boot on dev, an identity boot line from the live
process, a migration against live data, or the first named live plan. The
original correction report is already on dev via PR #100; this implementation
addendum accompanies the implementation branch pending its merge. On owner GO,
rebase/merge against then-current dev, assign the checklist number and rerun the
merged-head suite/build/gate sequence. Do not deploy a stale prepared binary if
the merged source differs. No RELEASE, main-tree binary or running process was
changed while preparing this candidate.


### Remote CI and partner handoff — not a green-remote-CI claim

**[A] PR #101**, head `71127ef5d20787d279764f55190bf7d8643cae1d` at this
observation, is mergeable, but its remote checks are **not all green**. Completed
job logs identify failures in unchanged workflow setup:

- Security run `34543205125`: installation of `govulncheck@latest` selects
  `golang.org/x/vuln v1.8.0`, which requires Go >=1.26; the job runs 1.25.3 with
  `GOTOOLCHAIN=local`. The scanner did not complete. Its npm production-dependency
  job separately fails with exit 127 while running `husky`.
- Image run `34543205094`: generated image tags have an empty prefix, e.g.
  `nofx-backend:-4ed4efa-amd64`, and fail as invalid reference format before
  the image build. No image publication is claimed.
- Docker frontend job `103090143569` cannot resolve
  `../../../branding/product.txt?raw` from `src/constants/branding.ts` inside
  its build context. That branding import and Dockerfile are unchanged by 105.
- Other backend checks were still running at inspection. Their eventual
  outcomes are not inferred from the local suite or from another job.

[Exact failure excerpts](2026-09-10-scenario-level-identity-data/remote-ci-setup-failures.txt).
No `.github` workflow, Dockerfile, dependency manifest or Go toolchain was
changed by 105. Owner ruling at cutover GO: these three CI setup failure groups are
**pre-existing, owed to cleanup batch 2**. They are recorded under A15 and
not repaired under the identity dispatch. The full local Go suite, 421 frontend tests and candidate
build results above remain separately established.

**[A] Partner handoff:** `/home/hoang/vlautoagenttraderv1` is at
`f6ae7597fb3bc9caeaaedb25ce8c3c48bca72247` (2026-08-23), lacks the identity
wave's prerequisite `kernel/plan_doc.go`, `kernel/scenario_state.go` and
`store/touch_outcomes.go`, and already has uncommitted changes in
`agent/planner_runtime_state_test.go`, `agent/skill_dispatcher_test.go` and
`agent/trader_scope_test.go`. It was not modified or pushed. A complete
`format-patch` handoff from the W2 base is prepared at
`/tmp/identity-build/partner-identity.patch`. Applying/building it requires the
partner's prerequisite baseline/history synchronization; no successful mirror
application or mirror test result is claimed.


### Cutover GO and partner routing (2026-09-10)

Owner authorized 105 after W2 boots and releases its lock, with 105's own
fresh five-leg gate and RELEASE → mv → VERIFY → exact kill command.
W2 boot was directly observed at 18:47:08 CT: `BOOT INTEGRITY OK — rev
4fc670aa4508 · expected 4fc670aa · goldens PASS`; its fade-permission
boot line was present at 18:47:15. Lock release and 105's gate remain
separate prerequisites.

The owner routes the partner patch to **Binnie's lane, handoff only**.
105 must not apply it. Prepared patch: `/tmp/identity-build/partner-identity.patch`,
SHA256 `59c582b9684173f8cc3caf29cf8b56f9d4b2f0bb8418c37b44e418d0bdedc138`.
The prerequisite and dirty-checkout observations above travel with that handoff.
A handoff note is staged beside the patch; no delivery acknowledgement from
Binnie's lane has yet been observed.


### Merged cutover candidate — 19:01 CT, not yet booted

**[A]** PR #101 merged at `cd8f99780eafe20fbb0b51a9c87955d3a2a4e887`,
including W2's final dev report `b952f2c4` and marker `4dc0fae1`. 105 acquired
the free lock as `scenario-identity-ac5a801d/root[unlisted]` at 18:53:50 CT;
the acquire-started keeper runs until 19:53:50 CT. No hand-beater was started.

**[A]** Full `go test ./...` passed in clean clone
`/tmp/identity-cutover/nofx` at that merged head (trader: 176.580 seconds).
Full frontend suite passed: 60 files / 421 tests. The clone was porcelain-clean
before `go build`. The binary reports:

```text
vcs.revision=cd8f99780eafe20fbb0b51a9c87955d3a2a4e887
vcs.modified=false
MD5=b32d80dba7d9dcfb335a194c71fcd7d9
SHA256=0ba70b618b1e4697b9d95a23e109930867e8a98b8a9408da6854e6c759781ddb
```

The binary-derived revision was stamped into BOTH the source worktree and
clone's `web/src/guide/types.ts`, THEN dist built successfully. The main
checkout has not yet been advanced or swapped.

**[A]** SQLite online backup, including committed WAL contents, completed at
18:56:55 CT: `/home/hoang/nofx-backups/identity-cd8f9978-20260910/data.db`,
793,993,216 bytes; backup `PRAGMA integrity_check` returned `ok`. No live
migration or identity boot is claimed before the actual restart.

**[A]** Own browser-authenticated gate at 19:01:31 CT returned `ready:true`: DB
OPEN 0; API positions 0; NT8 positions count 0; broker working 0 / ledger
working 0 / armed-unplaced 0; no planner read claimed. Earlier reads at
18:52 and 18:57 correctly held on planner work. A new read is required at
the eventual swap; this observation is not a standing authorization.

**A15 main-tree residue:** untracked `CLAUDE.md.save`, 20,171 bytes, mtime
12:49:45 CT, predates this cutover. It does not equal `CLAUDE.md`. It has
not been edited or removed; permission to preserve it outside the main
checkout was requested because the literal clean-tree gate currently fails.

**Provenance:** 105 authored identity changes on `fix/scenario-level-identity`.
W2's fade-permission code and boot came from `fix/fade-permission`, as its
marker and report record. 105 built and tested their merged head; it did not
author W2's code. Protected trading files and W2's fade files have no 105 diff.
Class 116 was assigned after the three-format census: highest 115; existing
duplicates only 75, 76, 77. The intermediate prose listing 92/93 was corrected
to the measured set before merge in `6f2ff668`.

[Go receipt](2026-09-10-scenario-level-identity-data/cutover-go-merged.txt),
[frontend receipt](2026-09-10-scenario-level-identity-data/cutover-vitest-merged.txt),
[build metadata](2026-09-10-scenario-level-identity-data/cutover-buildinfo.txt),
[backup](2026-09-10-scenario-level-identity-data/cutover-backup.json),
[dist build](2026-09-10-scenario-level-identity-data/cutover-dist-build.txt),
[Binnie handoff](2026-09-10-scenario-level-identity-data/BINNIE-HANDOFF.md).


### Owner-approved final preparation, 19:05 CT

The owner explicitly approved merging verification PR #102 and moving the
pre-existing save file. It was preserved intact as
`/tmp/identity-build/CLAUDE.md.save.pre-cutover` (20,171 bytes); main tree then
read porcelain-clean. Automatic approval review had rejected the direct dev
push and the follow-up PR merge before that explicit approval; no bypass was
used. All publication continues through the approved PR.

Own fresh gate at 19:05:04 CT: all five PASS, `ready:true`. DB/API/NT8 positions
all zero; broker working 0 / ledger working 0; **1 armed without a signal id**,
informational under the canonical leg-4 rule; no planner read claimed.
[Complete payload](2026-09-10-scenario-level-identity-data/cutover-pre-release-gate.json).
RELEASE is prepared as `cd8f9978` before the binary swap. This section is
preparation evidence, not a claim that the new process is running.


## Boot marker — owner-reported boot verified 2026-09-10 21:15 CT

**[A] Boot is verified, PID `2826476`.** Journal at 21:15:02 CT:

```text
🔐 BOOT INTEGRITY OK — rev cd8f99780eaf · built 2026-09-10T23:55:46Z · expected cd8f9978 · goldens PASS
🪪 level identity: map ids=n/a (no-formation=n/a; no captured map) · scenarios named=0 unnamed=0[WARN] unresolved=0[WARN] · heuristic-disagreed=0 · backfill recomputed=0 unrecomputable=321 untouched=5050
```

The identity recorder is live. The boot used a legacy plan without a captured
identity map, so `n/a` is deliberate unavailable data, not a map count of zero.
Legacy IDs remain NULL by design. Named/unnamed/unresolved counts are recorded
new-authoring events, not a retroactive classification of every legacy scenario.
A first newly authored plan with named IDs is a separate proof, pending below.
The nullable TEXT `touch_outcomes.level_id` column exists after boot. All 5,371
rows then read NULL; the durable backfill sidecar records all row ids and three
classifications: recomputed 0, unrecomputable 321, untouched 5,050. No partial
hash was used. [Row-id receipt](2026-09-10-scenario-level-identity-data/cutover-backfill-live.json).

**[A] Five revision references agree** (full SHA
`cd8f99780eafe20fbb0b51a9c87955d3a2a4e887`):

1. `/api/health`: `cd8f99780eaf`, status `ok`.
2. `/proc/2826476/exe`: full SHA above; `vcs.modified=false`.
3. `deploy/RELEASE`: `cd8f9978`.
4. committed Guide SOURCE: full SHA above.
5. served dist: full SHA above; all 92 file hashes match the prepared manifest.

The installed binary and running inode both have MD5
`b32d80dba7d9dcfb335a194c71fcd7d9`, exactly matching the clean-clone candidate.
[Reference receipt](2026-09-10-scenario-level-identity-data/cutover-postboot-references.json).

**[A] W2 remains live:** its 21:15:03 boot line says LABEL ONLY (no refusal),
today permitted=56 excluded=10 not-evaluated=0, and retains its E3-null coverage
statement. This is W2's work from `fix/fade-permission` (code `4fc670aa`, marker
`4dc0fae1`, report `b952f2c4`). Identity came from `fix/scenario-level-identity`
via PR #101 / merge `cd8f9978`. PR #102 / merge `d495d0b4` carries the
subsequent Guide, RELEASE and preparation receipts, with no Go changes after
the tested build. 105 built and gated the merged head, not authored all of it.
[Exact boot lines](2026-09-10-scenario-level-identity-data/cutover-boot-lines.txt).

**A15 — boot timing and sweep must not be misread.** The last pre-swap gate was
19:07:45 CT, all five PASS, broker working=0 / ledger working=0 / armed-unplaced=1.
105 swapped and verified the candidate, then printed `kill -9 2778818` for the
owner; 105 did not execute that command. The observed boot is timestamped
21:15:02 CT. The earlier gate is NOT a fresh gate for that later boot.
Arm **152**, ASIA plan v3 / S1, was created at 19:11:05 CT and carried signal
`62a9aa52-0c57-432b-bbc5-67a967550ca4`, entry 29094.00 by boot. The class-33
sweep cancelled it at 21:15:03; the row reads cancelled, snapshot-id 0.
This matches the separately recorded sweep limitation: it cancels prior-process
orders, including intended resting orders. It was not changed by 105.
[Exact arm receipt](2026-09-10-scenario-level-identity-data/cutover-live-data-proof.json).

The first post-boot browser gate at 21:15:41 CT was all PASS with no positions,
working orders or armed rows. At 21:17:03 a new planner read was in flight:
legs 1–4 remained PASS and leg 5 correctly read HOLD. This is normal activity
after boot, not permission for another restart.
[Later complete gate](2026-09-10-scenario-level-identity-data/cutover-postboot-gate.json).

**A15 — lock liveness:** the keeper ended at its declared 19:53:50 CT expiry
while waiting for the owner-executed kill. On resume it was STALE, still naming
this same 105 session; neither main HEAD nor the source tree had moved, both
were clean, and no other session owned it. No lock was silently seized or
hand-beaten. This marker must be published before that held lock is released.

The three CI setup failure groups remain **pre-existing, owed to batch 2**.
The partner patch remains a **Binnie-only handoff, unapplied by 105**; the
handoff note is linked above and no receipt from Binnie is invented.


### Targeted-restart ruling applied

The owner requested current-contract-only purge and store reseed, conditioned
on the store carrying separable contracts. It does not: contract is absent
from the bars schema and natural key, with contaminated OHLC already persisted.
No purge/reseed/additional restart was performed; counts 0/0. The current
contract evidence and exact RANGE line are in the incident report. This
STOP preserves the owner's prohibition against guessed contract attribution.
The first new identity plan remains unproven: a timezone-normalized
`julianday(created_at)` read after boot returned no authored plan yet.
