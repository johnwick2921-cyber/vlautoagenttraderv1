# CTO assessment: does the system behave like a disciplined level trader?

Status: baseline source review complete; reviewed repair candidate `cd2978b77da54e2fceddfb19e1d3d148bd2bfb62`. Final verification is recorded only in [CHECKPOINT.md](CHECKPOINT.md#final-verification). Not a deployment or profitability approval.

Source references below are repository-relative. Strategy source was inspected at
`63968be62e44db2fb07a92883e02127b9064b0be`; repair-specific behavior is at
the repair commits named in [REPAIR-STATUS.md](REPAIR-STATUS.md). Named
function boundaries below avoid carrying baseline line numbers onto changed source. The repair branch is undeployed.
[A] means source inspected or an explicitly named test run; [B] means inference.
Owner-supplied backtest numbers are not independently recomputed here. This is a
repository engineering assessment, not a new literature review or evidence that
a discretionary trading doctrine is profitable.

## Executive judgment

The system has many useful controls, but their number does not establish a
coherent trading process. I cannot sign off on complete trading correctness.
All 30 scoped review reports are complete:28 source slices cover1,049 files and
250,582 lines; two cross-boundary reviews examined repair 99a06543. Subsequent
repairs receive separate focused review/tests. Final combined verification has its own ledger in CHECKPOINT.md. The number of reviews is not the number of simultaneous agents.

The core standard is consistency: the same setup identity, contract, account,
entry, stop, target, permission and lifecycle must survive from market data to
planner to risk admission to the broker and finally into performance records.
A convincing prompt cannot compensate for a different execution rule. A green
unit test cannot establish profitability. A successful socket write cannot
establish an accepted order, a fill, a cancellation or protection.

## 1. The job is to select a worthwhile trade, including selecting no trade

[B] “Find the best trade” must mean best among currently eligible candidates
under an explicit, validated objective. It cannot mean always produce an order,
choose the highest model confidence, or choose the largest displayed R multiple.
The repository review has not established that the ranking identifies the best
future outcome or a positive-expectancy subset.

A reviewable selection record needs: observation time; concrete contract and
account; session; frozen level identity; scenario/version; evidence for the
setup; entry condition; invalidation; opposing obstacles; composed geometry;
all admission results; and why the chosen candidate outranked alternatives.
This is an engineering acceptance requirement, not a new enabled trading rule.

[A] `trader/armed_executor.go` separates placement from authoring. Repairs
2dc94a19 and 456b38d4 exercise the actual cycle and loopback transport: missing
permission or current quality refusal cannot leave an inherited authorization
eligible. Placement consumes IDs admitted in the current cycle. Retirement
write failure prevents placement. These are enforcement tests, not merely
assertions about the model's prose.

## 2. Stop location: invalidate the identified setup, then report its exposure

[A] `ResolveEntryGeometryZone` in `trader/structural_geometry.go` resolves the entry zone from the scenario's
frozen identity and source provenance. It refuses missing or ambiguous identity.
This is stronger than attaching the stop to whatever unrelated level is nearest
when the order is submitted.

[A] For a valid reject-fade zone, `composeGeometry` in that file sets the long stop
below the lower zone edge by the configured buffer, rounding outward to the
contract tick. A short stop is above the upper edge. Invalid/missing buffer is a
refusal. The ATR fallback recorded when provenance is missing also returns
`no_provenance`: it is diagnostic geometry, not permission to trade.

[A] The production arm caller at `trader/armed_executor.go` (the `structuralFade` branch) selects this
structural branch for the reject play, excluding explicit exit legs. Other
plays retain their legacy stop construction. Therefore “ATR has been removed
from all stops” would be false.

Unresolved: a configured/calibrated buffer is not automatically a validated
noise allowance. Its sample must be causal, contract/session appropriate and
out of sample. An overshoot distribution conditioned only on levels that later
held excludes breakdowns; it cannot by itself establish total stop-out risk.
The research review records that limitation without claiming a new calibration.

## 3. The planner and execution contract now agree on reject-fade geometry

[A] Repair 710ea1c8 updates the actual planner prompt builder, legacy stop-floor
facts and feasibility warning consumer. Reject fades use frozen structural
geometry; the legacy ATR floor is explicitly scoped to other plays. The prompt
no longer suggests switching execution routes to escape a refused arm. Authored
reject stops are not evaluated as if they were the later composed trade. Focused
production builder/warning tests pass and the map golden remains unchanged.
This is a consistency repair; it does not validate the buffer or trading edge.

## 4. Target selection: the next level is precise, but not proved optimal

[A] `FirstGeometryTarget`, `trader/structural_geometry.go`, chooses the nearest
complete sourced zone strictly beyond the entry zone's profit-side edge. It does
not rank by source count, grade, timeframe, minimum reward distance or measured
quality. The long target uses the target zone's near lower edge; shorts use its
near upper edge, with conservative tick rounding.

This answers WHICH level the code selects. It does not establish that all zones
represent equally material obstacles. A very dense map can still leave little
room. Conversely, skipping a nearby real obstacle merely to display 2R does not
prove the farther target is reachable.

[B] Keep “first obstacle” and “trade objective” conceptually distinct in research.
If a different target policy is proposed, compare it prospectively with the
current nearest-zone rule using identical entry events and execution assumptions.
Do not silently reinterpret historical target results or install an untested
quality ranking. A refusal because there is insufficient room is a valid output;
persistent refusals are a diagnostic to investigate, not permission to weaken
the gate until trades appear.

## 5. Entry, stop and target must describe one trade

[A] `composeGeometry` freezes tick-normalized entry, stop and target before
admission. It requires positive directional risk/reward, known costs, positive
reward after those costs, and the configured gross reward/risk threshold. It
reports planned one-contract dollar loss, then leaves daily risk and other
admission checks to their existing gates. Quantity is not authorized by this
geometry function alone.

Important distinction: the current R gate uses gross gain divided by stop
 distance. Costs are checked separately; this is not a net-R gate and not an
expected-value calculation. A 2R objective can lose money if it is reached too
rarely. A lower-R trade can have positive expectation with a sufficiently high
realized win rate. Neither ratio establishes an edge by itself.

For owner-supplied average win W, average loss L, win probability p and constant
round-trip cost c, the simplified expectancy is pW-(1-p)L-c. With W=4.12,
L=24.57, p=0.899 and c=2 points, this is approximately -0.778 points/trade.
The break-even win rate is (L+c)/(W+L), approximately92.6%. Thus “no win rate
saves it” is too absolute for those averaged outcomes; the supplied observed
win rate does not save it. This arithmetic is not a fresh backtest and does not
explain all fill-assumption differences.

## 6. Daily loss is an owner control; it is not a stop-placement algorithm

[A] The owner clarified DAILY loss, not a mandatory additional per-trade cap.
The structural core explicitly retains that distinction at
`composeGeometry` in `trader/structural_geometry.go`. Do not restore the removed cap or silently
set a new value. Contract value converts distance to planned exposure; it does
not decide where the setup is invalidated.

[A] `bootRiskFacts` in `trader/session_risk.go` reads the guardrails master and
individual daily-loss enable switch for boot reporting. This is a reporting
boundary, not evidence by itself that admission enforced the configured amount.
`entryGateForArm` and `entryGateForDecision` supply `DailyForceFlatReason` to
`EntryGate`; its daily-force-flat leg refuses when that resolver reports a trip.
`SessionRiskBootLine` labels a configured but disabled limit decorative. A displayed dollar value alone is
not proof of enforcement. Earlier observed settings are historical snapshots;
this assessment did not read or change current settings.

Required verification: define and test the precise daily boundary, realized and
unrealized components, commissions, account/trader scope, loss-trigger response,
resting-entry cancellation, and protection/flattening behavior. Whether to reserve
remaining daily budget for a prospective stop is a separate explicit owner
policy, not an assumption to introduce during a repair.

## 7. The broker lifecycle matters as much as the setup

[A] Committed repairs now distinguish authorization, placement pending, working,
fill, cancel pending and confirmed cancellation. Tests reproduce and repair:
startup checks occurring after autostart; entry paths escaping the boot latch;
same-version terminal arms being reauthorized; cancellation send being called
settlement; pre-request empty snapshots being used as cancellation evidence;
and previous-process retries exhausting a new process's retry allowance.

[A] A production placement-loop test also reproduced a refused stop entry
cancelling other scenarios as if it had placed. Commit 7b2eb894 makes successful
pre-send registration the commitment boundary. A separate test retains that
commitment after an ambiguous send error. Repair b63747ea covers the analogous limit path through the actual TCP adapter
and registration callback. Both commit on durable registration; neither calls
an ambiguous send a fill or confirmed cancellation.

All these are offline fixture results. They do not prove the installed AddOn
accepted an order or that every asynchronous interleaving is correct.

## 8. NT8 protection and account routing: repaired source, bounded evidence

[A] C# source repairs 9140f6c9/f1b7cc10 resolve explicit accounts without fallback,
select the actual held expiry, refuse ambiguous bare roots, and preserve SIM,
connection and session-account restrictions. Entry cancellation retains deferred
protection until terminal broker evidence. Submit ambiguity retains bracket
identity; cumulative fills amend one pair. Actual leg quantities confirm an
amendment, and synchronous terminal receipts survive submission reentrancy.

[A] Independent review reproduced two gaps in the first repair, then verified
both fixes. All33 extracted-production-method harness assertions pass and all
five AddOn sources compile against installed NT8 assemblies. These checks do
not recreate NT8 scheduling, the broker's OCO implementation or real fills.
The source AddOn has not been copied, compiled in the live NT8 installation or
restarted. No runtime protection claim follows from the temporary DLL.

[A/report] Additional cross-boundary tests reproduced completed partial exits
being recorded as whole-position closes and positive cumulative ENTRY fills on
terminal cancellation being omitted. Repair a982cc74 now records actual exit
quantity, receipt identity, fill and residual cost basis atomically, and handles
cumulative entry growth without overwriting partial-exit accounting. Repair 3f21431a also emits valid positive cumulative EXIT evidence on terminal cancellation/rejection; the report records 89 extracted-method assertions and five-source reference compilation. This is distinct from terminal ENTRY materialization. Current exit wire also lacks commission data;
zero additional recorded fee is unreported commission, not measured zero cost.

### Receive order, positive evidence and replacement lifetime

[A/source and reported offline tests] Repair 9b379c8c installs one exact account/symbol execution owner at successful trader construction. OrderUpdate, Fill and PositionClose are applied from TCP readLoop before advisory fanout. Internal handled flags prevent older consumers from reapplying events. Registration replacement and cleanup compare owner identity. Tests exercise the actual TCP path, both cumulative-entry/exit orders, raw and advisory replay, foreign accounts, cache resurrection and reentrant outbound progress.

For entry1@100 → cumulative entry2@105 → exit1@120, the expected residual is one contract at105 and realized MNQ P&L30USD. For entry1@100 → exit1@120 → cumulative entry2@105, it is one contract at110 with realized P&L40USD. Later cumulative growth of the same immutable order is observed exposure, not permission for another entry. Earlier exits and realized P&L are retained; corrected final P&L becomes unresolved while exposure continues. These fixtures establish their accounting cases, not exchange execution chronology.

Repair c20d0a82 retains a valid exit exceeding currently materialized entry quantity as pending rather than discarding it, and refuses absent/stale first position snapshots. Follow-up d7b70a90 preserves positive entry exposure even on rejection and fences a snapshot received before positive execution evidence. A rejection alarm cannot erase an actual partial fill.

Repair cd2978b7 moves the positive-entry receipt and cumulative deduplication state into the shared TCPServer, keyed by canonical symbol/account. Position data, receipt time and entry watermark are read together under one mutex. Replacing an adapter therefore neither forgets the prior entry nor lets an old duplicate renew its fence against a newer snapshot. The two replacement directions were reproduced before repair and covered by focused race tests; this summary reads their report rather than claiming another full source review.

The guarantee is receive order for an installed owner. There is no durable inbound journal or reconstruction of events received before ownership existed. Synchronous storage callbacks can backpressure transport and must not wait for broker replies. Failures still require later evidence/replay; committed database changes cannot guarantee subsequent process-local hooks across a crash. Same-order continuation does not retroactively repair all terminal analytics or excursion rows. These substantive limits remain after the named defects are fixed.

## 9. One contract and management rules must remain executable

One contract cannot be partially reduced. “Take some off and leave a runner” is
not an executable MNQ instruction for this account size. Every planned response
must map to hold, full exit, or another expressly supported action. Do not add
contracts to imitate a book's scaling example.

The historical task description says no stop movement, no re-entry and a session
cutoff. Those premises must be reconciled with current implemented stop repair,
management, same-version retirement and new-version authorization rules. A
protective-stop restoration is different from discretionary tightening, but the
UI and logs must identify which occurred. New plan versions must not become an
accidental loophole around the owner's intended re-entry policy.

Delayed flatten repair 94e08cf0 checks immutable position/entry lineage, invalidates
timers on Stop and preserves protection after close refusal. A broker-side atomic
position fence is still absent: a stale local row cannot prove that no unseen
replacement exists. Broker observers intentionally outlive ordinary Stop while
positions may remain. Repair d3e4638e retires a replaced reconciliation worker after its old close channel drains; an already executing pass can finish. Ordinary Stop still preserves protection observation. This bounded replacement handoff does not certify every shutdown path.

## 10. What establishes success, and the order of work

1. Close account-routing, protection, asynchronous settlement and stale
   authorization defects. Preserve SIM restrictions and real owner settings.
2. Align planner, geometry, admission, execution and displayed explanations.
   Every refusal must name the actual reason and retire incompatible permissions.
3. Baseline frontend/runtime-source and independent cross-boundary reviews are
   complete. Run the full suite on the final combined repaired source, relevant
   race tests, frontend verification and controlled broker lifecycle tests. Mark external/runtime
   checks unavailable until actually observed.
4. Validate the strategy separately: causal detector inputs and frozen levels;
   realistic limit fills, gaps and ambiguity; all costs; one-account chronological
   order/position constraints; actual session exits; and untouched evaluation data.
5. Compare policies with all candidates and refusals retained, not only executed
   winners. Report net expectancy, uncertainty with session dependence, tail loss,
   drawdown from the initial equity baseline, exposure and execution sensitivity.
   A hold rate alone is not trade win rate or expectancy.
6. If credible out-of-sample results remain nonpositive across realistic fills,
   do not label more code or a prettier R ratio an edge. Keep the strategy
   unapproved for promotion while recording the failed hypothesis honestly.

No precise buffer, optimum number of trades, universal target rule or profitable
regime classifier has been established by this source audit. The engineering job
is to make those hypotheses measurable and execution faithful. The trading job
is then to demonstrate that the selected opportunities pay after losses and costs.

## Evidence and completion limits

See README.md, CHECKPOINT.md, reviews/01 through reviews/30, coverage-validation.json
and [REPAIR-STATUS.md](REPAIR-STATUS.md), which separates repaired baseline
findings from concrete remaining source/runtime limitations. Frontend 28a6f32e
passed451 tests/build and was integrated as 6c4092bf. Full Go/build/focused race checks passed at 99a06543, before later changes. These are historical checkpoints; the single final verification ledger is in CHECKPOINT.md.
No deployment, owner-setting change, live database write or real order occurred.
Historical runtime snapshots from the earlier daily-loss dispatch are not new
observations. The earlier usage-blocked report is archived under interim/.

The final report must distinguish fixed/reproduced defects, static concerns,
intentional policies, dormant legacy paths and checks requiring real runtime.
Neither complete source coverage nor a green suite demonstrates profitable
trade selection. Owner-supplied backtest statistics remain supplied context,
not an independently repeated experiment in this engineering dispatch.
