# Wake-predicate wave — step 2: the 1B wiring (steps 3-4 gated on a boot)

Owner: hoang · 2026-09-03 · branch `fix/wake-predicate` · worktree `/home/hoang/nofx-wake`

---

## WAKE-PREDICATE wave — STEP 2 of 4 COMPLETE (1B wiring). Branch `fix/wake-predicate` @ `f1d7cf51`, NOT deployed.

**Live rev, read from `/proc/365128/exe` (owner correction accepted).** `vcs.revision=4d846e26`,
`vcs.modified=false`, built `2026-09-04T02:11:20Z`. Five-reference agreement: `deploy/RELEASE`
= binary stamp = `/api/health` = `4d846e26`. My earlier `f478ed88` was quoted from a peer's
message rather than the binary — my error. `042ff360` is also not the running rev.
**1B IS in that binary** (`trader/detector_record.go` present; boot line printed 21:19:00) —
it simply writes nothing.

**The A29 finding (survives the rev correction).** `recordDetectorOutputs`: **0 production call
sites** at the running rev AND at dev HEAD. My earlier count of "5" was a miscount — `git grep -h`
strips filenames, so the `_test` filter was inert. Live boot line: `touch_outcomes=0 · candidate_pool=0`.

| | |
|---|---|
| Wired at | `trader/auto_trader_planner.go:2347`, beside `kernel.ResolveVoidScope` in `assemblePlannerInputWithCtx` |
| Why there | the hook re-resolves that identical scope (`detector_record.go:38`) — the detector judges the SAME tape the prompt and validator read |
| `planVersion` | the version IN FORCE at read time, not the one being authored (a read can fail and author nothing — A24) |
| Live write proof | one production read → `candidate_pool=11 · touch_outcomes=137` |

**Tests, both RED→GREEN verified.**
- `TestPlannerReadWritesDetectorStores` (`trader/detector_production_path_test.go`) drives the
  REAL read. Call site removed → `candidate_pool EMPTY after a planner read`. Under that same
  condition the pre-existing `TestDetectorWritesThroughTheProductionPath` **stays GREEN** — it
  calls the hook directly, which is why it guarded nothing for the whole unwired period.
- `TestEveryClaimedProductionPathHasACallSite` (`trader/wiring_gate_test.go`) — the standing gate.
  Parses every non-test `.go`, collects wiring claims from function docs **and file-level banners**
  (1B's claim was a banner; a func-doc-only scan missed the exact case it exists for), fails on
  0 sites. Verified RED naming both `detector_record.go` and `recordDetectorOutputs`.

**Suite:** `go test ./...` EXIT=0, 28 ok, 0 FAIL at the pushed sha. One caveat, stated honestly:
`TestFanOutClosesLastResortIsHonest` (`provider/ninjatrader`, untouched by this wave) failed
**once** in an earlier full-suite run, then passed 3× isolated, full-package on both trees, and in
two subsequent full `./...` runs. Load-sensitive/intermittent; **not reproduced**, so not claimed
as proven pre-existing either.

**Checklist:** entry "Reported wired, called by nobody" — renumbered **69** on the rebase after
another lane landed 67/68; theirs untouched (A16).

**A18:** remote `f1d7cf51` == local sha the suite ran on ✓. **A14:** raw commit-ref URL → **HTTP 200**.
Process failure owned: I pushed once mid-rebase (remote briefly held the pre-rebase commit) —
corrected by completing the rebase, re-running the suite, and `--force-with-lease` on this
session's own branch.

**NOT DONE (needs your GO):** step 3 — boot this, let a session accumulate `touch_outcomes` /
`candidate_pool`, and only then build `WakeChangesAt` on real data so F1 is a genuine ASIA-09-02
replay rather than a test that asserts UNKNOWN. Until it boots, the wake predicate stays as-is:
75 `level_event` wakes / 7 days, 11 of ASIA 09-02's 15 versions.

**A15 — what you will STILL see wrong after this boots:** the wake drumbeat is unchanged (that is
step 4); `touch_outcomes` starts at 0 and only fills from the first read after the boot, so the
first session's counts are thin by construction, not by defect.
