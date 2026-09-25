# CTO assessment: does the system behave like a disciplined level trader?

Status: INCOMPLETE AUDIT; not a deployment or profitability approval.

Source references below are repository-relative. Strategy source was inspected at
`63968be62e44db2fb07a92883e02127b9064b0be`; repair-specific behavior is at
`7b2eb89455b3a73e911de64e66f5ea8c358c5913`. The repair branch is undeployed.
[A] means source inspected or an explicitly named test run; [B] means inference.
Owner-supplied backtest numbers are not independently recomputed here. This is a
repository engineering assessment, not a new literature review or evidence that
a discretionary trading doctrine is profitable.

## Executive judgment

The system has many useful controls, but their number does not establish a
coherent trading process. I cannot sign off on complete trading correctness.
The audit has 20 completed review reports covering 774 assigned source files;
10 reviews, important repairs and final combined verification remain unfinished.

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

[A] `trader/armed_executor.go:1174` separates placement from authoring. Existing
armed rows are consumed later. Consequently “the latest model declined it” is
insufficient unless the old authorization is retired too. Review20 A2 identifies
missing-verdict and ordinary-refusal cases requiring production-loop regression.
The geometry-refusal path already explicitly retires old authorizations; that
stronger behavior must be checked across other refusal paths.

## 2. Stop location: invalidate the identified setup, then report its exposure

[A] `trader/structural_geometry.go:28` resolves the entry zone from the scenario's
frozen identity and source provenance. It refuses missing or ambiguous identity.
This is stronger than attaching the stop to whatever unrelated level is nearest
when the order is submitted.

[A] For a valid reject-fade zone, `composeGeometry` at line123 sets the long stop
below the lower zone edge by the configured buffer, rounding outward to the
contract tick. A short stop is above the upper edge. Invalid/missing buffer is a
refusal. The ATR fallback recorded when provenance is missing also returns
`no_provenance`: it is diagnostic geometry, not permission to trade.

[A] The production arm caller at `trader/armed_executor.go:484` selects this
structural branch for the reject play, excluding explicit exit legs. Other
plays retain their legacy stop construction. Therefore “ATR has been removed
from all stops” would be false.

Unresolved: a configured/calibrated buffer is not automatically a validated
noise allowance. Its sample must be causal, contract/session appropriate and
out of sample. An overshoot distribution conditioned only on levels that later
held excludes breakdowns; it cannot by itself establish total stop-out risk.
The research review records that limitation without claiming a new calibration.

## 3. The planner currently contradicts that stop contract

[A] `kernel/planner_prompt.go:798` still instructs arms to satisfy a universal
ATR floor and suggests omitting the arm and using the AI path if both conditions
cannot be met. `kernel/class45_feeds_forward.go:178` labels the legacy floor as
this cycle's minimum and says tighter stops are widened.

[A] `kernel/plan_doc.go:581` also calculates feasibility warnings from authored
stops/targets, including reject scenarios whose production geometry is composed
later. These warnings can describe a different trade from the one the executor
will evaluate. This is a verified source contradiction; its prepared regression
has not run because approval review hit the usage limit.

Required repair: distinguish reject structural geometry from legacy plays in
all prompt/facts/warning surfaces, preserve actual configured admission rules,
and remove any suggestion that changing routes bypasses refusal. Verify the
production prompt builder and production arm call site together. Do not merely
edit a sentence while leaving its facts block and warnings contradictory.

## 4. Target selection: the next level is precise, but not proved optimal

[A] `FirstGeometryTarget`, `trader/structural_geometry.go:79`, chooses the nearest
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
`trader/structural_geometry.go:206`. Do not restore the removed cap or silently
set a new value. Contract value converts distance to planned exposure; it does
not decide where the setup is invalidated.

[A] `trader/session_risk.go:279` reads both the guardrails master and the
individual daily-loss enable switch. `SessionRisk` reporting explicitly labels
a configured but disabled limit decorative. A displayed dollar value alone is
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
cancelling other scenarios as if it had placed. Commit7b2eb894 makes successful
pre-send registration the commitment boundary. A separate test retains that
commitment after an ambiguous send error. The analogous limit-send-error path
remains an open item; it is not covered by claiming the stop repair fixed both.

All these are offline fixture results. They do not prove the installed AddOn
accepted an order or that every asynchronous interleaving is correct.

## 8. Open NT8 protection and account-routing concerns prevent sign-off

[A] `ninjascript/VLTraderTCPClient.cs:1152` handles close requests using a
symbol-to-account map and active-account fallback, rather than the request's
explicit account. The map is keyed by root symbol, which is insufficient to
represent two accounts holding the same symbol. The exact held expiry also
needs verification against front-month instrument resolution. These are source
findings; no wrong-account close was observed during this audit.

[A] `HandleCancelOrder` at line1735 attempts entry cancellation, but line1813
removes deferred bracket intent even after a caught cancellation failure.
`SubmitBracketOnEntryFill` at line2102 consumes that intent. [B] A cancel/fill
race can therefore lose future bracket placement intent. Existing placed
brackets are not cancelled by this handler; saying this code always removes an
existing stop would be false. Terminal state, partial fills and ambiguous bracket
submission need a coherent lifecycle repair with behavioral tests.

[A] `HandlePlaceProtectiveStop` at line1850 also needs explicit-account fallback
review. Current SIM restrictions remain essential and must not be weakened.
The unchanged AddOn compiled offline against installed NT8 references, but these
findings are not repaired or behaviorally verified. Compilation is not protection
proof. These concerns take priority over optimizing signal frequency.

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

These are acceptance checks still requiring the remaining runtime/UI review,
not assertions that every management rule currently agrees.

## 10. What establishes success, and the order of work

1. Close account-routing, protection, asynchronous settlement and stale
   authorization defects. Preserve SIM restrictions and real owner settings.
2. Align planner, geometry, admission, execution and displayed explanations.
   Every refusal must name the actual reason and retire incompatible permissions.
3. Complete runtime/frontend reviews and independent cross-boundary review.
   Run the full suite on the final combined source, relevant race tests, frontend
   verification and controlled broker lifecycle tests. Mark external/runtime
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

See README.md, CHECKPOINT.md, reviews/01 through reviews/20 and the separate
repair report. Reviews21/22 have partial artifacts;23–30 are unfinished.
Automatic approval review blocked the next test because Codex usage was exhausted;
several agents stopped for the same reason. No workaround was used to claim a
blocked test passed. No deployment, owner setting change, live DB write or real
order was performed for this assessment. This report is a traced interim CTO
assessment, not the promised final end-to-end completion certificate.
