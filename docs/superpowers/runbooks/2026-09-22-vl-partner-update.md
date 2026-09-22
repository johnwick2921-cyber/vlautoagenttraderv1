# VL update for Tin and Binnie — September 22, 2026

Prepared on a fresh clone of `johnwick2921-cyber/vlautoagenttraderv1`, based on
published main `78d35c25eca2e51853339bc5cf10ec30340cf40d`. Shared source changes
from nofx `8dcaca66` through `0960a6ac` were applied using format-patch → am.
The partner baseline matched every pre-existing file touched by that patch;
partner-specific files outside the patch were preserved. This is a partner
commit history, not a merge or reset to the unrelated nofx history.

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
   `nofx`. Require vcs.modified=false. Derive deploy/RELEASE and
   GUIDE_BUILT_REV from that binary, THEN build the frontend. The nofx source
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
