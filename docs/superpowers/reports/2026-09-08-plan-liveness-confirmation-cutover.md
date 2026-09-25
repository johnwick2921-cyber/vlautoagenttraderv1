# Combined PLAN-LIVENESS / CONFIRMATION-TRUTH cutover

**Status: combined boot VERIFIED at 2026-09-08 15:17:31 CT. Both wave boot
lines are present; integrity/goldens and five references agree on f8bc7044.**
The preparation sections below are historical; this update is the postboot marker.

## Authority, ownership and provenance

[A] Owner explicitly authorized **both waves in one boot** at the next A7
window and named PLAN-LIVENESS the deploy owner. The earlier A31 hold was an
over-broad interpretation: A31 forbids this lane from authoring executor changes;
it does not forbid booting a merged head legitimately changed by the other lane.
The owner clarified that distinction before the combined cutover resumed.

**This lane built and gated the merged head; it did not author all of it.**
Provenance below rests on the named branches, their captured commits, session
claims and the owner's explicit attribution, never on git's shared author field.

| Lane / captured branch | Implementation and validation commits carried by this boot | Responsibility |
|---|---|---|
| PLAN-LIVENESS — `fix/plan-liveness`, session `plan-liveness-c22ee052/root[unlisted]`, pre-combination tip `3ef10c1f93c49b0513d3cc22d97e4f402f915716` | `9c754e369f397d04a462fbcbefb47c2feef37ab0`, `5710cb5d353046aca531eb9ce049ab3bd780fdcb`, `94f0d7df8601eec585b38029ccafb90239bee90d` | Version/anchor-bound death records, authored-condition validation, exhaustion WARN/counter, liveness surfaces, qualifier/reader pins and telemetry error handling. |
| CONFIRMATION-TRUTH — `fix/confirmation-truth`, session `confirmation-truth-96604090/root[unlisted]`, captured tip `f8bc7044cc44d58e84904a0a7761e78b420404af` | `e020885b86621d68534fbc237472fc3171688d62`, `b99963857c453f909570eba7a854a8ec23eed087`, `2166a072339131fdb0a8e8e816e12148b261872b`; audit/receipt publication `c19faeed`, `ba833a9d`, `f8bc7044` | Closed-bucket and ordered-sequence semantics, `1m_displacement`, confirmation evidence/surfaces and the reviewed `trader/armed_executor.go` call-site changes. Those changes belong to this lane. |
| Combined deploy owner — PLAN-LIVENESS | Fast-forward onto `f8bc7044`, followed by Guide/RELEASE/report metadata commits recorded at handoff | Own fresh merged-head suite, clean-clone binary, frontend, broker gate, backup, ordered swap/VERIFY, printed owner kill and postboot proof. No additional Go or executor behavior authored for this cutover. |

[A] At **14:54:38 CT** the foreign lock had been released and main was clean
on dev `f8bc7044`. At **14:55:17 CT**, the deploy owner acquired its own lock
and started an independent heartbeat at acquisition. No reclaim occurred.
PLAN-LIVENESS then fast-forwarded to current dev. All three remote refs were
read as `f8bc7044` before the combined suite. The applicable window is
**14:45–16:30 CT**; no mid-session override was used.

## 23 + 1 validation REJECT→PASS is the approved replay result

[A] The embedded `kernel/confirmation_replay_receipt.json`, read at the combined
head, records **23 closure-only REJECT→PASS observations across two scenario
identities**, then **one additional observation under the separately authorized
`1m_displacement` rule: 24 total**. This is an approved semantic change, not a
newly discovered regression. These are historical revalidations at retained
decision instants, not original planner-write rejection counts, trades, or live
refusal counters. Replay coverage remains **627 evaluated / 162 unevaluated
scenarios** as documented in the confirmation audit.

The 23 closure-only observation IDs are:

- Plan **163/S1**, n=22: decisions **34787, 34788, 34789, 34791, 34792, 34793,
  34794, 34795, 34796, 34797, 34798, 34799, 34800, 34801, 34802, 34803,
  34804, 34805, 34806, 34807, 34808, 34809**.
- Plan **183/S1**, n=1: decision **35691**.
- Additional authorized case, n=1: **plan 163/S1 / decision 34790**.

The first two lists were independently extracted from the pinned confirmation
`breakdown-observations.jsonl` by selecting `validation_old != PASS` and
`validation_new == PASS`; their count is asserted as 23. The extra case is
named by the embedded receipt and its production validator pin.

## PLAN-LIVENESS refusal evidence remains n=1

[A] The born-dead refusal ships on **n=1 measured case**, plan row **265**,
ASIA v2 S1, authored **2026-09-07 22:03:44.933209 CT**. Its 5m-close-below
condition was **29664.50**; completed 5m bar row **451050** closed **29661.50**
at **22:00 CT**, with constituent minute rows **451031, 451034, 451037,
451039, 451051**. The original v4 born-dead claim was cross-version contamination
and does not add a second case. The owner ruled that this measured case permits
the refusal to ship. **Representativeness remains unproven at n=1**; the recorded
refusal counter and named evidence will tell us how often it recurs. UNKNOWN
accepts with a warning. Exhaustion remains WARN + counter, with no added wake.

## Own merged-head verification and binary

[A] Fresh ordinary clone: `/tmp/nofx-plan-liveness-combined-build/nofx`.
At merged HEAD `f8bc7044cc44d58e84904a0a7761e78b420404af`, before building:

- `go test ./... -count=1`: PASS, exit 0.
- Explicit kernel golden/self-check run: PASS, exit 0.
- Vitest: **51 files / 368 tests PASS**, exit 0.
- TypeScript: PASS, exit 0.

`go build -o nofx-bin .` followed those checks in the same clean clone:

```
vcs.revision=f8bc7044cc44d58e84904a0a7761e78b420404af
vcs.time=2026-09-08T19:50:07Z
vcs.modified=false
SHA256=e2c2ce8602ca61e52d180309743593b3bf538337693e4f2c61ae83c21d457918
```

`GUIDE_BUILT_REV` was parsed from this binary before rebuilding dist. Later
Guide/RELEASE/report commits are metadata; they do not change the binary's
embedded revision. Logs and artifact receipts: `/tmp/plan-liveness-combined/`.

## Required live proof contract (defined before boot)

Both boot lines must be read from the new service process, not copied from
an isolated function run or this expectation list:

1. **Confirmation:** `close-requires-closed-bucket=on`,
   `sequence-order=enforced`, `missing-reference=UNKNOWN(not met)`,
   `immediate-displacement=1m_displacement`, forming-bucket and out-of-order
   refusal counters, and the separately labeled **23 / 24 (+1)** replay counts.
2. **Plan liveness:** tradeable count availability, recorded exhaustion WARN
   count, born-dead refusals, recorded deaths and authored UNKNOWN count.
   The boot code prints **tradeable=n/a** before a current-version snapshot
   exists; it does not fabricate 0/M. Actual **N/M** must be checked from the
   first current-version API/card/desk snapshot after feed warm-up.

**If either boot line is absent, that half is not proved live and will be
reported as such.** Even both boot lines do not prove a future organically
occurring forming-bucket refusal, out-of-order refusal, born-dead refusal or
scenario death. Unobserved events stay unobserved; no forced trade/read is
used to manufacture proof.

Fresh five-leg gate (including broker-snapshot leg 4 and no in-flight read),
backup/integrity, RELEASE → `mv` → independent VERIFY → printed owner kill,
90-second boot integrity, five-reference check and pushed postboot marker are
still required below. No unattended kill or timed deployment is scheduled.

## Source freshness at merged head

```
docs/superpowers/reports/2026-09-08-plan-liveness.md
3ef10c1f93c49b0513d3cc22d97e4f402f915716 2026-09-08T14:05:14-05:00 docs(plan-liveness): stamp A7 candidate from verified binary
docs/superpowers/reports/2026-09-08-confirmation-truth.md
f8bc7044cc44d58e84904a0a7761e78b420404af 2026-09-08T14:50:07-05:00 docs(confirmation): publish verified candidate and stamp Guide from binary
docs/superpowers/AUDIT-CHECKLIST.md
78eed09b7969022408e3279d89c895c553885843 2026-09-08T14:32:14-05:00 docs(confirmation): assign class 91 at integration
docs/superpowers/SYSTEM-MAP.md
e020885b86621d68534fbc237472fc3171688d62 2026-09-08T14:21:34-05:00 fix(confirmation): require closed buckets and ordered reference evidence
kernel/confirmation_replay_receipt.json
e020885b86621d68534fbc237472fc3171688d62 2026-09-08T14:21:34-05:00 fix(confirmation): require closed buckets and ordered reference evidence
kernel/confirmation_telemetry.go
e020885b86621d68534fbc237472fc3171688d62 2026-09-08T14:21:34-05:00 fix(confirmation): require closed buckets and ordered reference evidence
trader/plan_liveness.go
9c754e369f397d04a462fbcbefb47c2feef37ab0 2026-09-08T08:56:19-05:00 fix(plan-liveness): bind death evidence to version and validate authored closes
trader/class33_cutover_gate.go
268ee6097b1aa2c7979552018f004b548592f182 2026-09-03T19:46:13-05:00 feat(F12): cutover leg 4 reads the broker; the override guard becomes a check
api/ui_serving.go
1560aeb21f9004227ebdd5ac68793b22696131df 2026-09-03T21:09:35-05:00 fix(ui): the bot serves its own UI + fix the gate-jwt 401 (owner rulings 2026-09-03)
deploy/RELEASE
11803c092221108b7da1f00944ccbf666fd9316d 2026-09-08T01:34:28-05:00 deploy: boot 20 marker — RELEASE=33672fdd + GUIDE_BUILT_REV=33672fdd, from the MAIN TREE after the passed boot
deploy/RESTORE.md
986a8fbeb6d16a2bc846349bdb6e78796e87ff16 2026-08-16T09:54:59-05:00 docs(deploy): RESTORE.md — binary rollback + the MANDATORY RELEASE re-arm
web/src/guide/types.ts
f8bc7044cc44d58e84904a0a7761e78b420404af 2026-09-08T14:50:07-05:00 docs(confirmation): publish verified candidate and stamp Guide from binary
```

## Fresh gate, backup and RELEASE preparation

[A] Production dist built after the binary-derived Guide stamp, exit 0. The
generated JavaScript embeds `f8bc7044cc44d58e84904a0a7761e78b420404af`.
The original worktree commit-hook attempt lacked its dependency symlink and
could not start ESLint; the normal hook passed after restoring that link.
No hook was bypassed and no production code changed.

[A] Deploy owner's fresh five-leg gate at **15:03:39 CT**, n=1 running trader,
HTTP 200, ready=true:

1. DB open positions: **0** (row IDs: `[]`), `sqlite trader_positions`.
2. API positions: **0**, `trader.GetPositions`.
3. NT8 positions snapshot: **count=0**, `NT8 positions frame`.
4. Working orders: **0 at broker, ledger agrees: 0**; source
   `broker — NT8 order_snapshot frame (age 8s, build 2026-09-07-h1)`.
5. Planner in flight: **no planner read claimed**, `plannerReadInFlight claim`.

All five legs are from one returned payload. The trailing legacy note saying
no NT8 working-order frame exists is stale; leg 4's actual source is quoted
above. A fresh read will be required again immediately before `mv`.

[A] Online SQLite backup completed **15:04:25 CT**, source opened `mode=ro`,
copy **750,182,400 bytes**, `PRAGMA integrity_check = ok`. Location:
`/home/hoang/nofx-backups/plan-liveness-confirmation-20260908-150422/data.db`.
No schema/data migration was performed by the deploy owner.

The same backup directory preserves `RELEASE.before` (**33672fdd**) and
`dist.before`. Preserved executable:
`/home/hoang/nofx-backups/plan-liveness-confirmation-20260908-150422/nofx-bin.old.33672fdd2cd2fee60a2c562a9693e06ab3b13551`.
Both `/proc/3260027/exe` and the disk/backup executable were independently
read as **33672fdd2cd2fee60a2c562a9693e06ab3b13551**, `vcs.modified=false`,
SHA-256 **25af1ec9714be825287fd697e9148340381ac1b702180cc392fb3dba0a5319c2**.
The backup name comes from that observed revision.

`deploy/RELEASE` is prepared as **f8bc7044**, derived from the verified combined
binary. This metadata commit is not the postboot marker. The main tree is
fast-forwarded under the lock after final metadata-head checks, and RELEASE
therefore precedes the swap and the owner's kill. The final gate, swap and
independent verification receipts are appended after they occur.

## Postboot marker — combined head built and gated by PLAN-LIVENESS

[A] **PLAN-LIVENESS acted as deploy owner for the merged head, not as the
author of all changes in it.** The lane/commit provenance table above remains
the attribution record. In particular the reviewed executor/confirmation
changes came from CONFIRMATION-TRUTH commits `e020885b`, `b9996385` and
`2166a072`; PLAN-LIVENESS's implementation commits are `9c754e36`, `5710cb5d`
and `94f0d7df`. This cutover authored deployment metadata and verified the
combined head; no new Go or executor change was introduced by the deploy owner.

### Ordered swap and observed restart

[A] Final pre-swap gate, **15:09:52 CT**, n=1 running trader, all five legs
PASS: DB 0 open (IDs `[]`), API 0 positions, NT8 count=0, broker 0 working
orders matching ledger 0 (snapshot **age 21s, build 2026-09-07-h1**), no planner
read claimed. RELEASE **f8bc7044** was already active on the clean main dev tree.

[A] **15:10:28 CT:** binary `mv` exit **0**; old-dist preservation and new-dist
`mv` each exit **0**. The prior executable was already preserved and verified
under its actual revision as recorded above. **15:10:37 CT:** separate VERIFY
exit **0**: RELEASE file, HEAD:RELEASE, disk binary and GUIDE matched f8bc7044,
every dist hash matched, and HTTP-served index/JavaScript matched the new dist.
The old process was still PID **3260027**, revision **33672fdd2cd2**, at this
verification. The owner then received exactly `kill -9 3260027`.

[A] Owner acknowledged completion. systemd recorded **15:17:26 CT**:
`Main process exited, code=killed, status=9/KILL`; it started the replacement
service at **15:17:31 CT**. The owner-shell command exit code was not supplied;
it is not invented as zero. Successful SIGKILL/restart is independently observed
in systemd and in the changed executable/PID. The agent did not issue the kill.
The observed boot is inside **14:45–16:30 CT**; no mid-session override occurred.

[A] The read-only watcher captured the following from **new PID 3566770** at
**15:17:31 CT**, verified by **15:17:32 CT**, within the 90-second boot window:

```text
🔐 BOOT INTEGRITY OK — rev f8bc7044cc44 · built 2026-09-08T19:50:07Z · expected f8bc7044 · goldens PASS
🖥 ui: served-by=go-static build=2026-09-08T20:02:55Z
🔎 confirmation: close-requires-closed-bucket=on · sequence-order=enforced · missing-reference=UNKNOWN(not met) · immediate-displacement=1m_displacement · forming-bucket refusals=0 · out-of-order refusals=0 · validation-REJECT→PASS=23 observations/2 scenarios (closure-only audit; evaluated=627 unevaluated=162) · with-1m_displacement=24 (+1 audit observation)
🧭 plan liveness: tradeable=n/a · exhausted-warnings=0 · born-dead refusals=0 · deaths recorded=0 · authored UNKNOWN=0 · exhaustion=warn-only
```

**Both halves' boot lines are present.** Confirmation's zero forming-bucket and
out-of-order counts are the new process's measured startup counters. The
**23 closure-only observations / 24 with 1m_displacement (+1)** are labeled
historical replay results, not live counters and not a regression. PLAN-LIVENESS's
startup counts are read from recorded rows; `tradeable=n/a` is the truthful
startup absence of a computed current-version snapshot.

The boot sweep separately logged **cancelled 0 pre-boot arms**, with **0
authorized-but-never-placed** arms left for the new process. Process census:
**n=1, PID list [3566770]**. No second service process was observed.

### Five-reference and feed verification

[A] Independent five-reference check at **15:18:16 CT** passed:

| Reference | Observed value |
|---|---|
| Working-tree `deploy/RELEASE` | `f8bc7044` |
| Disk and `/proc/3566770/exe` vcs.revision | `f8bc7044cc44d58e84904a0a7761e78b420404af`, `vcs.modified=false` |
| `HEAD:deploy/RELEASE` | `f8bc7044` |
| `GUIDE_BUILT_REV` and served JavaScript | `f8bc7044cc44d58e84904a0a7761e78b420404af` |
| `/api/health` | revision `f8bc7044cc44`, status `ok` |

Disk binary SHA-256 remains
`e2c2ce8602ca61e52d180309743593b3bf538337693e4f2c61ae83c21d457918`.
Every dist hash and HTTP-served index/JavaScript match the built artifacts.

[A] NT8 `bars_historical` frames were received by the new process between
**15:17:32 and 15:17:56 CT**. At **15:21:17 CT**, the authenticated live
`/api/klines?symbol=MNQ&exchange=ninjatrader&interval=1m&limit=3` response
returned **n=3** bars from NT8 BarCache, identified by openTime values
**1788898740000, 1788898800000, 1788898860000**. The first two minutes are
complete; the last is the forming 15:21 minute, whose closeTime is in the
future at observation. It is evidence of a recovered live feed, not a completed
confirmation bucket. No forming bar is presented as a closed-bar proof.

[A] Fresh postboot five-leg gate at **15:19:40 CT** again passed all five:
0 DB/API/NT8 positions, broker 0 working orders matching ledger 0,
`broker — NT8 order_snapshot frame (age 10s, build 2026-09-07-h1)`, no planner
read claimed. No override was applied to leg 4.

### Live-surface limits and event proof still owed

[A] At **15:20:49 CT**, `/api/plan/today` returned HTTP 200 with
`found=false`, `active_session=""`, `is_active=false`, `night=true`,
`trade_date="2026-09-08"`. There is **no active version from which to report
tradeable N/M in this window**. No plan-card N/M screenshot or active desk count
is claimed. The new frontend is byte-verified as served; actual active-version
N/M remains event-dependent and must be observed when a current plan exists.

[A] At **15:20:11 CT**, read-only system_config prefix census found:
`scenario_death:` **n=0, row IDs []**; `plan_liveness_event:` **n=0, row IDs []**.
No organic new scenario-death record, exhaustion WARN event or born-dead refusal
was verified in this handoff. A live forming-bucket refusal, out-of-order refusal
or qualifying MET event also remains unverified. These are not manufactured
by forcing authoring, placing a trade or rewriting historical rows.

The n=1 born-dead evidence limit remains as ruled above. The confirmation
replay's **162 unevaluated scenarios remain unevaluated**. Legacy saved verdicts
without the new evidence can remain UNKNOWN until reevaluated. Both boot
surfaces are live; these future event proofs are explicitly outstanding.

### Publication, rollback and closeout

[A] Final pre-swap metadata head **cd5b9a6b9c479eae97eced7fb0abb40594bf1ac5**
passed the full Go suite, **51 files / 368 Vitest tests**, and TypeScript checks
in the clean clone before the main-tree RELEASE fast-forward. Go source differs
from compiled f8bc7044 only in subsequent deployment/report/Guide metadata.
Both origin/dev and fix/plan-liveness were read at that metadata SHA. Its
commit-pinned raw cutover report returned **HTTP 200 / 11,695 bytes**, exactly
matching the git blob.

This postboot marker is committed only after the passed boot. It is fast-forwarded
into the same clean main dev tree, checked at its own final SHA, pushed and
verified by commit-pinned raw bytes **before the deploy lock is released**. The
final publication and lock-release receipt is retained in
`/tmp/plan-liveness-combined/candidate.json`. No rollback was required. If a
later rollback is ordered, the preserved prior binary, RELEASE and dist listed
above must be restored together under the same fresh-gate/owner-kill protocol;
no historical DB rows are rewritten to make the result appear clean.

Additional source-freshness reads for the postboot surfaces:

```
api/handler_plan.go
5710cb5d353046aca531eb9ce049ab3bd780fdcb 2026-09-08T09:00:25-05:00 test(plan-liveness): pin unknown qualifiers and versioned readers
api/handler_klines.go
91faf354c2baf4a9b2ed311d275db98c9d58bdea 2026-05-30T13:07:06-05:00 fix(nt8): live MNQ bars reach chart via klines ninjatrader branch
```
