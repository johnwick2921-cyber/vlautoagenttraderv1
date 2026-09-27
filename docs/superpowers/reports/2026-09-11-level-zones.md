# Level zones — boot verified at 13:58:32 CT, running 6c96683c

## C1 — the owner's four lines, traced first

[A] Frozen live planner read: **2026-09-11 10:30:27.269319419 CT**, snapshot
`7f9db41a-815f-4be8-a9e2-ea124db26b67`, research input row **23094445**.
Candidate rows **23093602–23094444**, n=843, writer revision
`802fb00b09e51f9801e8d4fbd1bf156c86865d95`. Price=29485,
ATR5m=42.11359506516323, dATR=465.75, final seats=12, intermediate pool=24.
This is one observed read, not a claim about every read today. Full whitelisted
candidate evidence: [snapshot.json](2026-09-11-level-zones-evidence/snapshot.json).

For the owner's approximate 29700 and 29000 lines, this audit explicitly uses
±15 points, matching half the widest supplied 30-point band. This is an audit
matching convention, not a detector parameter. Other bands use inclusive exact
endpoints. Matching an anchor does not establish that it is the owner's original
Sep 4 feature. A wide historical zone overlapping a band is also not that proof.

| Owner band | Detected anchors | Recorded collapse / selection | Model map |
|---|---:|---|---|
| 29685–29715 | 8 | RN 29700, EQH 4h/1h/15m, EQL 4h, Demand/IFVG/OB 4h. None in the recorded 24-row pool or final seats; exact later cut cause is NULL. No identified Sep 4 high with the requested provenance. | None of these anchors in `Levels`; none in capped `HTFZones`. |
| 29570–29590 | 11 | RN 29575 → VWAP+2σ 29573.0987; EQH·15m 29587 and EQL·15m 29583.75 → EQH·1h 29585. OB·1h 29585.875 is in Pool but absent from Levels. Other exact later cut causes are not all recorded. | No anchor in final map or capped HTF section. |
| 29470–29500 | 33 | SWG-H·15m 29475 **already collapses into SWG-H·5m 29475** (rows 23094168 → 23094164). The 5m survivor reaches Pool at score 1.36, then is absent from final Levels. | The shared swing loses its final seat. ONH 29500.75 is outside the strict band; do not silently count it as inside. |
| 28985–29015 | 1 | Demand·1d 29006.625, row 23094181: `proximity distance=478.375 exceeds band=465.75`. Never reaches collapse or seating. | Absent. **No 1d/4h/1h SWG-L trio detected.** |

[A] There are only 12 SWG records in the entire captured universe, rows
23094161–23094172, all 5m or 15m. `SwingPointLevels` explicitly runs 5m/15m;
the HTF detector registry runs EqualHighsLows, SupplyDemandZones, FairValueGaps,
OrderBlocks, not SwingPointLevels. W-TF's report explicitly says base swings
remain 5m/15m by owner ruling. E1's mandatory three higher-timeframe swing lows
cannot be constructed from this tape without inventing references or expanding
detection scope. Both violate this dispatch.

[B] For 29475, pool membership plus final absence proves final selection loss,
but the archive does not identify the exact overriding seat operation. A NULL
exclusion is not evidence for a specific cut. Research pointers are shared across
subsequent scoring passes; raw recorded final scores/reasons must not be treated
as an immutable transcript of each intermediate pass.

## Operative basis and revision

Owner clarification: Round 21 ran in chat; the dispatch's RESEARCH LAW governs.
The full report is **not yet committed** at
`docs/superpowers/research/2026-09-11-level-zones/` in the accepted tree. No full
Round 21 report or SHA has been fabricated. Pin it when it lands; until then
this limitation remains explicit. The dispatch is the user attachment
`fdf096e7-6f1b-4371-b360-fa3b31fbf123/pasted-text.txt`.

[A] `/api/health` returned `{"revision":"802fb00b09e5","status":"ok","time":null}`.
`go version -m /proc/3366586/exe` returned
`vcs.revision=802fb00b09e51f9801e8d4fbd1bf156c86865d95`,
`vcs.time=2026-09-11T13:41:30Z`, `vcs.modified=false`.
Audited code paths have an empty diff against that running SHA.
Branch `fix/level-zones`, accepted dev tip
`616b52a9def4042ce40308deba46529723ba01d7`; isolated locked worktree
`/tmp/nofx-level-zones`. Claim session `level-zones-fdf096e7/root[unlisted]`.

Playbook: `docs/superpowers/AUDIT-CHECKLIST.md`; apply provenance, sample IDs,
no inferred zeros, binding, call-site evidence, and merged-head validation.
No cutover was attempted. Every cited code/report file's `git log -1` receipt is
in [source-freshness.txt](2026-09-11-level-zones-evidence/source-freshness.txt).

## C2 — bands versus points

[A] **511 real bands, 332 points, 843 total** (not approximately 350).
Real band means `0 < lo < hi`; point means equal positive lo/hi. The struct
already has serialized `Price`, `Lo`, `Hi`, with Price documented as a zone's
midpoint. OB, FVG/IFVG and supply/demand retain their real bounds in the captured
raw origin. A claim that these detectors discarded their bounds is NOT
REPRODUCED. The presentation and merging can still fail to use those bounds.
New display bounds must not overwrite serialized legacy Lo/Hi if E6 holds.

## C3 — fixed tolerance and measured pairs

[A] `clusterToleranceFor` returns `LevelClusterTicks * 0.25`, with ticks=12:
3.00 points. Production calls at this revision:
`levels_score.go:590`; `map_candidates.go:138`, `:227`, `:539`.
The first changes pre-seat scoring output; the other three build/merge map or
projection views. They are not equivalent mutation surfaces.

Unordered pairs from all 843 raw references (354903 possible pairs), strictly
more than 3 points apart, inclusive upper bound:

| m | m × captured ATR5m | Pair count |
|---|---:|---:|
| .25 | 10.528398766290808 | 7165 |
| .50 | 21.056797532581616 | 17077 |
| .75 | 31.585196298872425 | 26506 |

[A] Computed from exact archived anchors, no rounding before comparison. These
are cumulative pair counts, not resulting cluster counts or independent sources;
raw duplicates remain because this is the detected-universe census. ATR5m at
this read is 42.11, not the dispatch's illustrative 20–35.

## C4 — name retention

[A] `collapseLevelClusters` excludes bands from point clustering, records a
loser's label only when its TF differs from the keeper, and propagates prior
CollapsedNames. Same-TF distinct labels are omitted. The 29475 cross-TF pair
already merges; E1's assertion that it must fail today as separate levels is
NOT REPRODUCED. Seat loss still hides both labels because the model map is built
from `in.Levels`, not the raw universe (`planner_prompt.go:552`).

## C5 — ordering, score, daily census

[A] Map order: absolute distance ascending, score descending on equal distance.
Merge keeper order uses score. Scoring/seating also applies today-priority,
score, distance, HTF/volume/both-side seat passes, grade filtering, and final
nearest ordering. There is **no independent 1.2 multiplier in map sorting** to
remove. The 1.2 is inside the non-zone score when `HTF=true`; zones instead use
zoneTFMult. The HTF detection set includes 15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w;
actual configured input here emitted 1d/4h/1h/15m references. Daily/weekly zone
tier maps to 4h. PDH/PDL and other anchors can also carry HTF=true.

`zoneSizeMult`: size=(hi-lo)/dATR; thresholds ≤.30→1.25, ≤.60→1.10,
≤1→1, ≤1.5→.85, ≤2.5→.70, otherwise .50; invalid inputs→1.
[A] Daily rows 23094176–23094196: 21 raw; seven in the ±465.75 band including
one same-kind duplicate; six with recorded scores; zero final daily seats.
Highest recorded daily score is **1.27296** (row 23094194), not .862.
The old .862/1.260 figures are **NOT REPRODUCED** on this snapshot. No causal
claim that zoneSizeMult alone explains daily seat loss is established.

## C6 — prior touches are not a free universal count

[A] `touch_outcomes` contains ordinal/outcome and identity fields. Its
`NextOrdinal` scopes by trader, root symbol, exact price and session-day lower
bound, then MAX+1. It does not match formation identity or require the same
contract, and does not impose an upper as-of bound. It is not a safe direct
prior-touch-ranking query for all newly merged references.

The read records 12 `candidate:prior_episodes` facts separately. Row 23093275
(ONH identity) records two detector episodes; row 23093276 (PDH identity) one.
Both say `formation_verified=false`, with basis explicitly limited to the
available detector window. The 843 candidate planner-read records have
`prior_episodes=null`. Do not replace this unknown with zero or interpret ordinal
as a globally independent, formation-correct historical count. Ranking needs a
specified as-of identity and missing-data policy; this audit has not established
a complete count for all 843 references.

## C7 — round numbers

[A] `RoundNumberLevels` uses price ± proximityK×dATR, grids 100/50/25,
deduplicating identical prices. Captured rows 23093613–23093650: 38 references,
29025 through 29950. The measured window is 29019.25–29950.75. Proximity to these
already detected references is available without changing detection. Distance
to an abstract grid outside this window would be a distinct ranking choice.

## Original blocking corrections — resolved by owner ruling

1. E1's higher-timeframe swing-low trio is absent by detector design; the
   29475 pair already collapses. Use the actual recorded pair and its final
   seat loss as the live acceptance case. Keep missing higher-timeframe swings
   explicitly outside this wave rather than fabricate them.
2. D4 changes what the existing SCORE term counts, but E6 requires the full
   serialized score output unchanged. Existing code already counts distinct
   families, with a different taxonomy. Altering that input can alter scores
   even when its numerical weight is unchanged. W3's pinned-basis report §2
   records this same conflict and its owner ruling: presentation/order only.
3. D2 says replace every caller, including score-time collapse. That operation
   alters serialized survivor sets and Confluence, while E6 compares Seated
   and Pool JSON. A render-only carrier cannot hide those changes. Do not
   silently weaken E6 or change the scorer to satisfy the new presentation.

Proposed corrected scope for owner ruling: preserve legacy scorer, collapse,
and anchor outputs byte-for-byte; build zones and five-family counts from the
uncut detector universe on a separate render/shortlist path; rank that path
without reading HTF-weighted Score. Apply volatility merge there and explicitly
retain the legacy scorer's tolerance. Use measured available references for E1.
The owner subsequently accepted both corrections: E1 uses the observed 29475
seat-loss pair plus daily demand, and D4 family count is presentation-only.
The score confluence input remains untouched. The chart's “1D/4H/1H” label is
the owner's reading of structure, not a claim about emitted detector references.
The daily demand **band** overlaps the read's range; its **anchor** fails the
legacy proximity cut. The earlier measured rejection is not withdrawn.

No production edits, mutation experiments, changed fixtures, builds, merges,
DB writes, or boots were performed in this audit. No RED/GREEN or golden
preservation claim is made. The score incompatibility is source-proven design
coupling, not a claimed executed mutant. The original scope stop is resolved; the independent native-overlap stop below
now prevents implementation.

## A15 / remaining proof

Full Round 21 SHA outstanding. Exact cut causes are missing for some archive
rows. Full-map prior-touch identity policy remains unresolved. The owner will
still see the existing point-oriented, seated map because nothing deployed.
No backfill was run; proposed zones remain read-time computation. Any future
zone-vs-line or multi-source claim remains UNTESTED; widths, merge factors,
caps and ranking weights remain [I]. The dispatch's proposed experiment and
sample-size/power figures are not independently validated by this audit.

Rollback: documentation-only; running binary and data are untouched. Final
cutover proof, tests at merged HEAD, Guide stamp, boot counts, marker and
SHA-pinned HTTP byte verification remain owed after corrected scope and build.

## Follow-up C replay — D2 native overlap bridges the whole local map

**[A] New STOP under A23, after the corrected scope was accepted.** The
presentation/scoring separation is feasible. The obstacle is the requested
merge relation applied to the actual native bands, not that separation.

Read-only replay uses all 843 recorded detector outputs, their original native
Lo/Hi, and the same frozen ATR5m. It adds an edge when two native intervals
overlap or two anchors are within m×ATR5m, then takes connected components.
No new point width, score, gate, cap, age filter or detector is introduced.
This is an independent audit calculation, not an implemented production merge
or a mutation claim. It does not rely on a greedy input order.

| Rule | Components | Component sizes | Component holding both chart cases |
|---|---:|---|---|
| Native overlap only (m=0) | 4 | 827, 13, 2, 1 | 28313–31104.50 |
| Native overlap or .25×ATR5m | 4 | 827, 13, 2, 1 | 28313–31104.50 |
| Native overlap or .50×ATR5m | 4 | 827, 13, 2, 1 | 28313–31104.50 |
| Native overlap or .75×ATR5m | 4 | 827, 13, 2, 1 | 28313–31104.50 |

The dominant zone is **2791.50 points wide**. The two E1 cases would both reach
a full-map render, but as members of this same 827-source zone. That is not a
claim that the requested useful local zones have been delivered.

A three-reference witness suffices, without any long transitive chain:

| Archive id | Reference | Original bounds |
|---|---|---|
| 23094164 | SWG-H·5m, anchor 29475 | 29475–29475 |
| 23094181 | Demand·1d, anchor 29006.625 | 28810.75–29202.50 |
| 23094275 | OB(bull)·4h, anchor 29252.875, origin 2026-06-09 | **28500.75–30005** |

The 4h OB directly contains both other references. Keeping its real bounds
(D1), merging all overlaps (D2), and retaining every old reference (D6) forces
these into one component. The 15m swing at 29475 (row 23094168) joins too.
Adding point widths cannot split a component that native overlap already joins;
changing m within the proposed test range cannot resolve this case.

Reproduce:

```sh
python3 docs/superpowers/reports/2026-09-11-level-zones-evidence/overlap_replay.py
```

[Complete output and component row IDs](2026-09-11-level-zones-evidence/overlap-result.json).
The independent union-find replay agrees with a separate iterative union replay:
four components with sizes 827/13/2/1 at m=.5.

**Ruling needed:** retain broad historical bands as separately rendered context,
and make overlap alone insufficient to merge them with local reference zones?
That would preserve every bound and name while allowing separate local zones,
but it changes D2's unconditional overlap rule. No width cutoff, age cutoff,
containment exception or deletion has been silently installed. The criterion
separating contextual bands from mergeable zones must be explicit and [I].

No production code changed. No build, DB write, or cutover occurred. The worktree
remains locked for this dispatch while the ruling is pending.

## Compatibility ruling accepted — implementation in progress

Owner replaced unconditional overlap with BOTH fixed-anchor distance ≤m×ATR5m
(default .5 [I]) and resulting width ≤1×ATR5m [I]. Each source joins its one
nearest compatible cluster; clusters never join one another. Broad bands stay
separate context. The earlier native-overlap STOP is resolved by this ruling.

[A] The production snapshot pin now runs `BuildLevelZones` over all 843 archived
outputs: **222 total zones = 58 local + 164 broad context**. Of the local zones,
**44 contain multiple references**, 14 are singletons. It absorbs 621 references.
The widest merged envelope is **41.50 points**, below the resolved
**42.11359506516323** cap. The “tens” count refers to local/merged zones, not total
zones: preserving 164 broad contexts makes a total in the tens impossible.
The 29475 5m/15m pair remains together; 29006.625 daily demand remains its own
28810.75–29202.50 context band. The actual model prompt contains both cases.
These census numbers use archived native bounds and missing point evidence,
not a claim that the archived JSON contains newly added defining-wick metadata.

The breadth threshold resolves independently (`LEVEL_ZONE_BROAD_ATR`, default
1 [I]); any band also wider than `LEVEL_ZONE_MAX_WIDTH_ATR` remains standalone.
No context reference is dropped. Merge ordering is stable ascending original
anchor, retaining detector emission order on ties; this deterministic ordering
is [I]. A cluster anchor never moves. Unknown widths are preserved per source;
merged envelopes with unknown members explicitly read incomplete.

D4 is presentation-only. Five display families, capped at 3, are carried beside
every source. The scorer's old taxonomy and confluence input remain unchanged.
New zone ranking reads no Score or HTF multiplier. Its four weights default 1 [I]:
log(1+known prior touches), 1/(1+distance to detected RN/ATR5m), capped family
count, minus distance/ATR5m. Unknown touches receive no bonus and display NULL.
Counts require known formation and complete available post-formation 1m tape;
no stored legacy ordinal is relabelled as a complete history.

D1 adds output-only exact pivot-wick evidence to swing results, after detection.
Other points without derivable defining-bar/ATR inputs stay NULL and counted.
Native bounds are retained; rounds receive a resolved 2-point width [I].
All parameters and proposed sensitivity ranges are in the Guide. Source
FormedAtMs, legacy identity, scoring output and all execution anchors remain
unchanged. Historical plans are not backfilled; new plans freeze `zone_map`.
The card panel displays that frozen map; the candle chart still draws authored
levels, not every machine context zone (A15).

Validation so far: initial chart/width/family/compatibility pins RED on undefined
new view symbols; all now GREEN. `TestStageAScoreParityLegacy` PASS with unchanged
binary JSON golden. Kernel/trader production-wiring pins PASS. Card and Guide
checks: 2 files / 7 tests PASS. Width mutation initially NOT-APPLIED (spacing),
then corrected through `scripts/mutate.sh`: `o.WidthK*in.ATR` → `.01*in.ATR`,
mutant builds and width pin KILLS it. Merge mutation `d > o.MergeATR*atr5m` →
`d > .01*atr5m`: builds, compatibility pin KILLS it. Further mutations/full
suites, clean build, binary-derived Guide revision, merge and live proof remain
pending. No deployment GO has been requested or used for this new wave.

## Validation checkpoint before A23 stop — 2026-09-11

[A] Full frontend suite **62 files / 427 tests PASS**, TypeScript PASS.
The first sandboxed run failed to spawn Go (`EPERM`) in the branding suite;
rerunning with process permission passes all eight branding tests. The dispatch's
anticipated brand-scope failure was therefore NOT REPRODUCED in this run.

[A] Five required semantic mutations all build and are KILLED through
`scripts/mutate.sh`: width (`o.WidthK*in.ATR` → `.01*in.ATR`); merge distance
(`o.MergeATR*atr5m` → `.01*atr5m`); name carrier (`Label: l.Label` →
`Label: "lost"`); family count (`len(z.Families)` → `len(z.Sources)`);
HTF ordering (insert `if z.Sources[0].TF == "1d" { rank *= 1.2 }`).
The family pin includes one/two/three-family progression, not just the cap.
A production dataflow mutation `in.Zones = &zoneView` → `in.Zones = nil`
also builds and is KILLED by the wiring guard. Mutants were restored.
The model now has one zone ENTRY SHORTLIST, while the older score/identity
reference rows remain available without presenting a competing shortlist.

[A] `go test ./...` is **FAIL**, solely on these two test failures in
`nofx/trader`; kernel (including Stage A parity), store, API, deploy and the
other listed packages pass:

- `TestOneSetupDeclinedPreBootAuthorizationRetiredNeverPlaced`
- `TestLiveConditionPlacesOnLoopback`

Feature suite logs: `book age 35m0s exceeds the 1m0s bound`.
**A/B on unchanged RUNNING SOURCE** in clean clone
`/tmp/level-zones-baseline/nofx`, detached at
`802fb00b09e51f9801e8d4fbd1bf156c86865d95`:

```sh
go test ./trader -run '^(TestOneSetupDeclinedPreBootAuthorizationRetiredNeverPlaced|TestLiveConditionPlacesOnLoopback)$' -count=1
```

Both fail again, `nofx/trader 4.403s`, now showing `book age 30m0s exceeds the
1m0s bound`. [A] `shadowWireHarness` seeds `OrderSnapshots().PutAt(...,
time.Now())`; both tests then run the arm manager at `armTestClock(t, at)`.
That helper searches forward/backward in five-minute steps to escape blocked
session windows. During lunch the selected evaluation time is 30–35 minutes
later than the fixture receipt. **Pre-existing, not introduced by this wave.**
The one-minute freshness guard correctly refuses it; it must not be weakened.
[A/B receipt](2026-09-11-level-zones-evidence/arm-fixture-ab.txt).

**STOP under A23.** The narrowly scoped correction would make these loopback
fixtures use one chosen clock for session, plan/bars and broker snapshot receipt
before calling the existing `...At(now)` entry point. No arm/gate production
behaviour change is needed or proposed. These arm test fixtures are outside
this lane's stated level-map footprint; owner ruling is required before editing
them. No merge, Guide binary stamp, deploy or kill has occurred. Remaining
call-removal mutations, clean build and merged-head suite are still owed.

### Intermediate envelope counts — superseded by strict known-width pin below

[Production replay output](2026-09-11-level-zones-evidence/compatibility-result.json)
records: native unconditional overlap gave 4 components (largest 827 sources,
2791.50pt); corrected compatibility gives **222 zones, 44 actual merges,
164 broad contexts, widest merged envelope 41.50pt**, cap 42.11359506516323pt.
There are 38 cross-TF and 37 cross-family merged zones; 19 candidate/cluster
comparisons fail the width limit. Source widths missing in the archived
snapshot remain NULL (294); incomplete envelopes are explicitly labelled.
Thus 41.50pt is the maximum **known envelope**, not an assertion that missing
member widths were measured. The final strict cap guarantee with unknown
members still needs review before release; do not read the native-bound census
as a completed proof for every newly derivable D1 point band.

E4 distribution (different denominators, **not** a score experiment): original
recorded score-family counts across 843 raw references are 1:4, 2:52, 3:11,
4:132, 5:247, 6:173, 7:80, NULL:144. The new capped display count across
222 zones is **1:185, 2:11, 3:26**. Source-name multiplicities range 1–45;
the full histogram is in the replay JSON. No count is fed back into scoring.

Exact model-table excerpts produced by the view:

```text
  anchor 29006.62 · band 28810.75–29202.50 · broad context · families=1 · incomplete-width=false
    Demand·1d · tf=1d · anchor 29006.62 · formed_at=1778130000000 · width=native detector bounds
  anchor 29456.00 · band 29454.75–29485.00 · local · families=3 · incomplete-width=true
    SWG-H·5m · tf=5m · anchor 29475.00 · formed_at=NULL · width=NULL: defining wick or ATR(tf) unavailable
    SWG-H·15m · tf=15m · anchor 29475.00 · formed_at=NULL · width=NULL: defining wick or ATR(tf) unavailable
```

The grouped local anchor is a display representative; each source retains its
own anchor. These rows are an offline production-render replay, **not a live
post-boot plan**. The full generated zone block is 109422 bytes; this increases
the model input. No token/cadence/timeout knob was changed to hide that cost.

A15: full Round 21 report remains absent from the accepted tree, no SHA invented.
Legacy plans have no zone panel by design. Historical display widths/touch
counts are not backfilled. Actual post-boot model adoption and clean merged-head
validation remain unproved. The isolated worktree stays locked pending ruling.

## Final in-scope correction: unknown width cannot pass the cap

The intermediate 222-zone result above is **superseded**. Review found that it
used an unknown-width source's anchor as a zero-width proxy in compatibility.
That cannot prove D2(b). The implementation now keeps unknown-width references
separate until defining evidence exists; no incomplete cluster can absorb a
reference. This is enforcement of the approved BOTH condition, not a new
trading gate. The strict-width focused tests and Stage A golden pass.

The archived raw JSON never carried the new defining-wick metadata. For the two
chart pivots, [pivot-width-evidence.json](2026-09-11-level-zones-evidence/pivot-width-evidence.json)
records supplemental store rows: MNQ 09-26 / source=live, 5m open
1789134900000 and 15m open 1789134300000. Both highs are 29475, upper defining
wicks are 3 points, and the existing k=2 confirmation relationship matches
archived formation closes 1789135800000 and 1789137000000 respectively.
The 5m ATR is the archived input's exact 42.11359506516323; the 15m ATR is
65.0852 at its recorded four-decimal precision. The selected swing-side wick
is used, not its opposite tail. This supplements missing old output metadata;
it is not a claim that the old snapshot serialized those values.

**Final measured snapshot result: 843 sources → 498 total zones/references,
25 multi-source merges, 164 broad contexts, 292 unknown-width singletons.
The widest actual merged band is 41.50pt, below the 42.11359506516323 cap.**
Every merged band is known and under the cap, pinned in the chart test. The
remaining 17 singleton local references plus 25 merged local zones account for
the rest. Thus the merger count is in the tens; total retained references are
not, because broad and unknown references are preserved rather than discarded.

Both E1 cases reach the production-rendered model table:

```text
  anchor 29006.62 · band 28810.75–29202.50 · broad context · families=1 · incomplete-width=false
  anchor 29457.88 · band 29454.75–29491.27 · local · families=3 · incomplete-width=false
    SWG-H·5m · tf=5m · anchor 29475.00 · formed_at=NULL · width=max(defining wick,k×ATR(tf)) [I]
    SWG-H·15m · tf=15m · anchor 29475.00 · formed_at=NULL · width=max(defining wick,k×ATR(tf)) [I]
```

Final capped family histogram: {1: 477, 2: 20, 3: 1}. It is a
presentation distribution, not a score change. The replay JSON above now holds
these final counts. Prior intermediate counts remain labelled for auditability.

The pre-existing two arm-fixture failures remain the external blocker. Full
suite and additional call-removal mutations must be rerun after the fixture
ruling; no deployment, merge or Guide-built revision is claimed here.

Final in-scope verification: complete `go test ./kernel` PASS (0.814s), focused
kernel/trader zone + parity + wiring tests PASS, `go build ./...` PASS. Dev
freshness rechecked: still accepted tip `616b52a9`; no spec drift. Full trader
suite remains blocked by the independently reproduced fixture-clock defect.


## Owner-authorized clock-only fixture correction (2026-09-11)

The owner's explicit ruling supersedes the earlier A23 fixture STOP: fix both
fixtures with one clock, test-only, no production line, then cut over on this
lane's own fresh gate. No gate or arm predicate was changed.

[A] Both fixtures now own `2026-09-11T15:00:00Z` (10:00 CT, outside lunch) and
explicitly install the persisted whole-day TEST session. That same value feeds
the broker snapshot receipt, plan creation, existing active-plan provider clock
seam, bar timestamps, and `maybeManageArmedOrdersAt(nil, now)`. The helpers and
the two consumers are entirely in `_test.go` files. Existing helper callers
retain their wall-clock provider. No production line is part of this correction.

| Fixture | Before, unchanged running source `802fb00b09e51f9801e8d4fbd1bf156c86865d95` | After, same production source + test-only patch |
|---|---|---|
| `TestOneSetupDeclinedPreBootAuthorizationRetiredNeverPlaced` | FAIL (2.23s): `book age 30m0s exceeds the 1m0s bound`; `the allowed scenario's pre-boot authorization did NOT place — the retire pass over-reached` | PASS (1.55s): allowed S1 places, declined S2 retires, no second signal or spurious broker cancel |
| `TestLiveConditionPlacesOnLoopback` | FAIL (2.17s): `book age 30m0s exceeds the 1m0s bound`; `live condition did NOT place — regression in the arm seam` | PASS (0.93s): nonempty signal reaches the loopback wire |

[A] The unchanged-source before run was at 13:01 CT in
`/tmp/level-zones-baseline/nofx`, HEAD exactly the running SHA above, initially
porcelain-clean. The identical test-only patch was then applied there; the two
tests passed together (`nofx/trader 2.489s`). Thus these are **pre-existing fixture
failures, not regressions introduced by level zones**. Feature before-run age
was 35 minutes; baseline age was 30 minutes because the real clock advanced;
both failures are the identical stale-book predicate, not identical elapsed age.
The freshness refusal remains intact. Test patch and concise output receipts:
[clock patch](2026-09-11-level-zones-evidence/arm-fixture-clock.patch),
[after on running production source](2026-09-11-level-zones-evidence/arm-fixture-after.txt).

The first corrected feature run also passed (`nofx/trader 0.733s`). Full suite,
remaining call-removal mutations, merged-head validation and deployment evidence
are recorded below as they complete. No cutover has occurred at this entry;
A7's next permitted window is 14:45–16:30 CT. The owner's GO stands, subject to
the deployment lock and this lane's fresh five-leg gate at cutover.


### Validation after the authorized fixture correction

[A] `go test ./...` PASS, exit 0, including `nofx/trader 223.667s`.
The two arm fixture files are the only Go diff against the preceding feature
commit `5e81db95c1ac9788fd26caa23fa4bfaee119dfff`; the production delta for
this correction is empty. No Stage A golden changed. Earlier full frontend
validation remains 62 files / 427 tests PASS and tsc PASS; this correction
contains no frontend code change. Merged-head checks remain required at cutover.

[A] All five semantic mutations were rerun against the final strict-width
implementation through `scripts/mutate.sh`: width, merge, names, family count,
and reintroduced HTF ranking multiplier. All applied, built, and were KILLED.
Six additional call-removal/replacement mutations also applied, built, and were
KILLED: production width-input assembly, zone-map build input, planner rendering,
boot invocation, family classifier, and width input lookup. The source was
restored after every run. [Exact changes and verdicts](2026-09-11-level-zones-evidence/mutation-receipts.txt).

### A15 — CI is not claimed green

[A] PR #104 at `5e81db95` reports CI failures despite the passing local full
suite. Test job `34631884981` shows the branding Go history guard exiting 128;
the frontend job explicitly reports `fatal: bad object
954f11b15f2e7615678f7d2b708c47895faebf1e`, and Vite denies
`/home/runner/work/nofx/nofx/branding/product.txt?raw`. The relevant workflow,
branding guard and Vite configuration have no diff in this wave. These are
reported setup surfaces, not reasons to weaken the guards. The summarized log initially hid the cause of coverage run `34631884694`.
Retrieving the complete job log establishes it: the same two named arm fixtures
fail with `book age 15m0s exceeds the 1m0s bound` (2.73s and 2.10s), and no
other `--- FAIL:` appears. This is the same fixture-clock defect now corrected;
the different age reflects the later CI wall time. Dev's same-tip coverage
run `34607468613` also failed, but matching status alone is not proof of its cause. Security run `34631884852` reports 24 called
standard-library vulnerabilities; go.mod/go.sum and the workflow are unchanged.
No CI configuration or dependency changes are included in this wave.


[A] Final focused race check: `go test -race ./trader -run
'^(TestOneSetupDeclinedPreBootAuthorizationRetiredNeverPlaced|TestLiveConditionPlacesOnLoopback)$'
-count=1` PASS (`nofx/trader 1.768s`). The full regular suite passed before the
mutations, and every mutation restored its source. A final running-process
read still returns health revision `802fb00b09e5`, PID3366586, executable
`vcs.revision=802fb00b09e51f9801e8d4fbd1bf156c86865d95`,
`vcs.modified=false`; service `Restart=on-failure`.

**Cutover pending A7, not a new approval request.** At 13:25 CT the permitted
14:45–16:30 CT window has not opened. No deploy lock is held by this lane, no
merge or binary swap occurred, and no kill is offered outside the window. The
owner's GO remains authorized. At cutover: acquire the free lock with its own
bounded keeper, merge current dev, run the merged-head suite in a clean clone
named `nofx`, derive the Guide revision from that binary before rebuilding dist,
then perform this lane's fresh five-leg gate and RELEASE → mv → VERIFY → print
the resolved owner-run kill. A gate measured now would not be a fresh cutover
gate for that later window. Boot and first-plan live proof remain unmeasured.


## Mid-session cutover authorization and merged-head build

The owner explicitly overrode A7's time window and A3's owner-run kill:
"boot now, mid-session" and "you run the kill yourself". This lane will execute
the resolved SIGKILL only after its own fresh gate, RELEASE → mv → VERIFY.
Open positions still block; any in-flight planner read is waited out. Resting
arms, if present, must be quoted before and their sweep result after. No gate
code was edited and no order-specific pass path exists in this wave.

[A] Acquired the free main-tree lock at 13:46:30 CT for 60 minutes. The lock's
own bounded keeper beats every 120 seconds through 14:46:30 CT. Main was clean,
on dev. Current dev at merge included `027da6f2` (PR #105, branch
`docs/forming-candle-test`, implementation/report commit `53accd5e`), a separate
read-only research lane. That lane's report is included, not authored by this
lane. This lane's work is PR #104, branch `fix/level-zones`: core `8cfc7394`,
strict-width correction `5e81db95`, test-only arm-clock correction `30d8c722`,
with report/checklist follow-ups. This lane **built and gated the merged head,
not every commit it contains**. Prior provenance markers `701637eb`/`616b52a9`
already described running `802fb00b`; they are not new code in this cutover.

[A] PR #104 merged as `6c96683c704f9a9ea5267af0ea33f0c5df631261`, with checklist
class 123 assigned from the fresh all-format census (previous ceiling 122).
Main fast-forwarded under the lock. The separate clean clone
`/tmp/level-zones-build/nofx` checked out that exact merged HEAD. Full
`go test ./...` PASS there; frontend 62 files / 427 tests PASS; `tsc --noEmit`
PASS. Only then was the binary built. Its embedded revision is
`6c96683c704f9a9ea5267af0ea33f0c5df631261`, `vcs.modified=false`, md5
`838ae0762ef5f00e04d857c18eabe033`. The Guide SOURCE stamp and RELEASE above
were derived from that binary, not from a guessed git head. Dist is rebuilt
from the stamped source next. This is a pre-cutover receipt, not a boot claim.


## Boot marker — 2026-09-11 13:58:32 CT, PID3497330

**[A] BOOT COMPLETE.** This supersedes the earlier A7 hold and pre-cutover
status entries. Under the owner's explicit mid-session and agent-run-kill
orders, this lane executed SIGKILL on old PID3366586 at **13:58:27 CT**.
Systemd (`Restart=on-failure`) relaunched PID**3497330** at **13:58:32 CT**.
Integrity and goldens passed within seconds. The running process is the clean
merged binary, not the old executable. No unattended timer was used.

[A] Fresh gate before swap: 13:57:11.543 CT, PASS; checked age33.4s at swap.
Fresh gate immediately before kill: **13:58:21.756 CT**, PASS; checked age5.5s
before SIGKILL. All five legs: DB open positions=0; API positions=0;
NT8 snapshot positions=0; broker working=0 and ledger working=0,
**armed/unplaced=0**; no planner read claimed. Broker snapshot age18s at the
last read. **No resting arm existed to sweep, and no in-flight read required
waiting.** No leg override or gate-script edit was used; the owner's override
changed only the time window and who executed the kill.
[Exact final gate](2026-09-11-level-zones-evidence/pre-kill-gate.json).

[A] RELEASE and its committed HEAD value were already `6c96683c` before the
swap. The old binary was moved to `nofx-bin.old.802fb00b`, verified to hold
`802fb00b09e51f9801e8d4fbd1bf156c86865d95`, md5
`bf71fabd7336ec2ecc183d22f048bb83`. Old dist is preserved at
`/tmp/level-zones-build/dist.old.802fb00b`. New binary and dist were moved into
place; VERIFY established the embedded revision and md5 **before** SIGKILL.
The stamped dist contains the full new revision in `assets/index-DMbSO-tR.js`.

| Reference | Observed value |
|---|---|
| Disk `deploy/RELEASE` | `6c96683c` |
| `HEAD:deploy/RELEASE` | `6c96683c` |
| Guide SOURCE `GUIDE_BUILT_REV` | `6c96683c704f9a9ea5267af0ea33f0c5df631261` |
| `/api/health` | `6c96683c704f`, status `ok` |
| `/proc/3497330/exe` VCS revision | `6c96683c704f9a9ea5267af0ea33f0c5df631261`, modified=false |

[A] Built binary, on-disk binary and running executable md5 all equal
**`838ae0762ef5f00e04d857c18eabe033`**. The pre-marker source HEAD is
`aa5e90dbdfbe548710dcc462960b762cd0755186`; it is the Guide/RELEASE/report
stamp commit, not the binary's build revision. These distinct commit roles are
intentional; the five binary references above agree.
[Verification receipt](2026-09-11-level-zones-evidence/boot-verification.json).

### Actual boot lines, read from the new process

```text
09-11 13:58:32 [INFO] nofx/main.go:295 🔐 BOOT INTEGRITY OK — rev 6c96683c704f · built 2026-09-11T18:49:47Z · expected 6c96683c · goldens PASS
09-11 13:58:32 [INFO] trader/auto_trader.go:44 [trader_id=8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265 trader_name=hoang] 🗺 zones: width=max(wick,k×ATR) k=0.5[I] · merge=0.5×ATR5m[I] max-width=1×ATR5m[I] broad>1×ATR5m[I] (also standalone above max-width) round-width=2pt[I] · families capped=3[I] · rank=[touches,round,families,distance] weights=[1,1,1,1][I] htf-mult=removed-from-zone-order (score unchanged) · cap=12[O] · detected/merged/context/NULL-width=n/a (resolver=BuildLevelZones per read) · trader=8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265 bound-strategy=a5b7662e-7bf7-49bb-9f09-7efa48f95ac8 · session cap re-resolved each read; no backfill
09-11 13:58:33 [INFO] trader/auto_trader.go:44 [trader_id=8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265 trader_name=hoang] 🛡 cutover safety (class 33): gate legs=5 · leg4=ledger (no snapshot yet) · boot sweep cancelled 0 pre-boot arm(s) (0 authorized-but-never-placed left for this process)
09-11 13:58:33 [INFO] nofx/main.go:297 🗄 research snapshot: schema=1 · objects=5 · rows today market=UNKNOWN candidate=UNKNOWN plan=UNKNOWN scenario=UNKNOWN exec=UNKNOWN · null-fields=UNKNOWN · dropped=0 · added latency p50=UNKNOWN
09-11 13:58:33 [INFO] nofx/main.go:312 🖥 ui: served-by=go-static build=2026-09-11T18:55:43Z
```

[A] The zones line names bound strategy
`a5b7662e-7bf7-49bb-9f09-7efa48f95ac8`, resolved k=.5, merge=.5, max-width=1,
broad=1, RN width2, family cap3, four weights1, and session cap12. Per-read
counts are explicitly `n/a (resolver=BuildLevelZones per read)`; these are not
fabricated zeros or counts borrowed from the offline snapshot. The sweep line
reports **cancelled0 / authorized-but-never-placed0**. Its initial "no snapshot
yet" is a startup observation; the subsequent gate below reads the broker.
The separate research recorder reports **schema=1**, not schema=UNKNOWN;
UNKNOWN row counters remain exactly as the surface reports them.

[A] At **13:59:37.888 CT**, the post-boot gate again passes all five legs:
positions0, broker/ledger working0, armed0, no planner read; broker snapshot
age4s. Historical frames resumed and replay logged verified live-scale MNQ4h
rows at13:59:22. **First new-plan proof remains pending:** `/api/plan/today`
still serves NY version6 authored **12:41:53.97648234 CT**, with no `zone_map`.
It predates this boot; no backfill was performed and no forced planner read was
triggered. The process and zone boot surface are live, but a merged zone in a
new model table/card is not yet observed. The 843-reference acceptance fixture
is evidence for the implementation, not a substitute for that live read.

[A] Lane provenance remains the merge section above: this lane's PR104 plus
the already-merged read-only research PR105; this lane built and gated their
merged head, not all of their source. Full merged-head Go suite PASS
(`nofx/trader 188.857s`), vitest427/427, tsc PASS. The two clock-fixture
FAIL→PASS receipts on unchanged running source are pre-existing corrections,
not regressions attributable to this wave. No production arm/gate line changed.

Rollback, if needed: under the same safety protocol restore the preserved
`802fb00b` binary, matching RELEASE/Guide/dist, and restart only after a fresh
gate (or the owner's explicit scoped override). The old binary and dist remain
available; no rollback was triggered because integrity and health passed.

This marker is committed and pushed to dev from the same release tree before
the deploy lock is released. Final remote/HEAD equality and pinned HTTP bytes
are checked after the marker push; they are not inferred from this text.
