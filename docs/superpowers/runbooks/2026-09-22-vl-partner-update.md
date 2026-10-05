> _Old project name replaced with VL on 2026-10-05._

# VL update for Tin and Binnie — September 22, 2026

Prepared on a fresh clone of `johnwick2921-cyber/vlautoagenttraderv1`, based on
published main `78d35c25eca2e51853339bc5cf10ec30340cf40d`. Shared source changes
from vl `8dcaca66` through `0960a6ac` were applied using format-patch → am.
The partner baseline matched every pre-existing file touched by that patch;
partner-specific files outside the patch were preserved. This is a partner
commit history, not a merge or reset to the unrelated vl history.

## Included

- Picture HTF SIM strategy, native bar completion evidence, broker event
  handling and reconciliation, and its UI/configuration surfaces.
- AddOn `2026-09-20-p1`.
- Roll-aware charts and the Planner candle-response correction (PR 179).
- Shared tests, Guide content, and the other changes in the cumulative patch.

## Update each partner machine

1. Record the running Go revision, received AddOn build, selected strategy,
   and current positions/orders. Back up the database using SQLite backup
   and preserve the existing binary, frontend, RELEASE, and AddOn sources.
2. Require a fresh flat gate: broker and store positions flat, no working
   entries or orphan protective orders, and pending arms retired through
   the supported control path. Prevent re-arming during the attended update.
3. Fetch the approved partner branch and update by fast-forward/merge without
   overwriting `.env`, local data, credentials, account bindings, or saved
   strategy settings. This package does not change those values.
4. Use NT8's own trace to identify its loaded user-data AddOns directory.
   Back up the old sources under their build ID; copy the updated repository
   C# files there, compare hashes, F5 compile, and fully restart NT8.
   Verify a received frame carrying `2026-09-20-p1`; source files alone are
   not a receipt that the compiled AddOn is running.
5. Build at the exact approved partner code commit in a clean clone named
   `vl`. Require vcs.modified=false. Derive deploy/RELEASE and
   GUIDE_BUILT_REV from that binary, THEN build the frontend. The vl source
   SHA and the partner binary SHA are different; do not copy Hoang's SHA
   into the partner's release stamp.
6. Install the matching binary/RELEASE/frontend in the owner's attended
   safe window, using the machine's existing service restart procedure.
   Verify boot integrity, health revision, UI bundle, and AddOn receipt.
   Restore the saved binary/RELEASE/frontend if verification fails.
7. Picture HTF defaults OFF. If the partner requests activation, verify the
   selected SIM account and at least 120 completed native 4H bars for the
   resolved contract in the evaluator cache, including completion evidence.
   Count bars, not detected levels. DB rows alone are not cache readiness.
   Native history must not create retroactive live entries. Enable through
   Strategy Studio only after readiness passes.
8. Return actual receipts: Go revision, AddOn build, data readiness, saved
   mode/runtime mode, and first natural entry/fill/protective-order frames.
   State pending evidence explicitly; installing code is not trade proof.

Neither partner machine was accessed or deployed by preparation of this branch.
The owner performs the partner-repository push under the standing repo rule.

---

## Boot 5 / 6 / 7 update steps (PARTNER-SYNC-BOOT7, 2026-09-30)

The boot-7 sync (`sync/vl-4d538206-20260930`) carries the vl tree at
`4d5382069643` (booted 01:00:43 CT 2026-09-28) — three boots ahead of the
boot-4 tree this runbook was written for. Boots 5, 6 and 7 add: the updater
installs with NT8 closed (`trader/installation_gate_nt8absent_test.go`), the
flat-live allowance's predecessors, the one-button release-build fixes, and
every W-EXEC-TRUTH fold since boot 4.

Same procedure as steps 1–8 above, with these deltas:

1. **Build at the commit-of-build proof sha.** The sync PR's body names the
   partner build commit (the tree-sync commit, not the stamp tip). Build from
   a clean clone at THAT sha; `go version -m` must show
   `vcs.revision=<that sha>` and `vcs.modified=false`.
2. **AddOn.** The boot-7 tree's `VLTraderTCPClient.cs` is build
   `2026-09-23-m21`. If a machine's NT8 runs an older AddOn build, copy the
   file + F5 + full NT8 restart (the HARD RULE: copy, compile, full restart —
   no hot reload).
3. **One-order check** at the next market open: a single SIM order round-trip,
   and report the boot line (machine, sha, time) to the CTO.
4. **Updater worker is OPTIONAL** for partners and is NOT installed by this
   sync. If a partner later chooses it, it builds from the partner repo
   (`install-updater-worker.sh` defaults `REPO_URL` to
   `johnwick2921-cyber/vlautoagenttraderv1`).
5. **No release capability.** The partner workflow has no trigger that can
   fire; no partner CI run can create a release or tag in
   `johnwick2921-cyber/vl`.

## R2 — VL rename sync (`sync/vl-0b45081d-20261001`, DS-106, 2026-10-01)

Syncs the partner tree to the R2 rename boot: vl `0b45081d33e3f372fd2fde83fffda9f4268bb97a`
(booted 23:07 CT 2026-09-30; docs-only dev content up to `db412e61c` is NOT included —
this tree is `0b45081d` exactly, verified by blob-sha match table). Partner-machine steps:

1. **Build at the commit-of-build** `ab8effac4fdb5e45b7d1915c82d7608b13d8f83f`
   (named in `deploy/RELEASE`), from the partner repo, never from vl.
2. **Migrate**: `deploy/migrate-to-vl.sh --dry-run`, review, then the real run.
3. **AddOn**: copy `VLTraderTCPClient.cs` for `VL_BUILD_ID 2026-09-30-m22`,
   F5 compile, full NT8 restart (HARD RULE — no hot reload).
4. **Updater re-enroll**: `deploy/install-updater-worker.sh` (defaults REPO_URL
   to `johnwick2921-cyber/vlautoagenttraderv1`).
5. **Carve-outs on this tree** (differ from vl@0b45081d BY DESIGN):
   `.github/workflows/release.yml` (workflow_dispatch-only, both jobs
   `if: ${{ false }}`, `contents: read`, `RELEASE_REPO` = partner repo),
   `deploy/release_contract_test.go` (partner assertions),
   `deploy/install-updater-worker.sh` (`REPO_URL` defaults to the partner repo),
   `deploy/RELEASE` (names the partner build commit), this runbook.
