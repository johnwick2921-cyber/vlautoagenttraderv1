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

## REL19 — partner sync (release #19, vl `192ef87d9`, DS-105, 2026-10-13)

Syncs the partner tree to release #19 (vl `192ef87d9c045…`, tag
`v2026.10.13.2`). **AddOn change in this release** — the partner machine MUST
copy the AddOn files and fully restart NT8, not just build the Go binary.

1. **Build at the sha named in `deploy/RELEASE`** (`afb7ce565…` — the REL19
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   then build at that sha from a clean clone and verify
   `go version -m vl-bin` shows `vcs.revision=afb7ce565…` `vcs.modified=false`.
   For a FRESH install, build at `main` HEAD and re-arm `deploy/RELEASE` to the
   built sha (`echo -n "$HEAD_SHA" > deploy/RELEASE`) plus `GUIDE_BUILT_REV` in
   `web/src/guide/types.ts`, then `(cd web && npm run build)`.
2. **AddOn (HARD RULE):** copy ALL 5 `ninjascript/*.cs` files
   (`VLTraderTCPClient.cs`, `VLBarsSubscriptionManager.cs`,
   `VLContractResolver.cs`, `VLHistoryPull.cs`, `VLInstrumentLookup.cs`) to the
   NT8 `bin/Custom/AddOns/` folder, **F5 compile inside NT8, then FULL NT8
   restart** (AddOns do NOT hot-reload). The new build id is
   `VL_BUILD_ID = 2026-10-13-a` (`SendAllOrderSnapshots` for every SIM + bound
   account, sent on connect / periodic / account_register).
3. **Boot-line check:** after restart the boot line must read
   `nt8 addon: build_id=2026-10-13-a … match=yes` (Go's
   `ExpectedAddonBuild` is `2026-10-13-a`; `match=yes` means the wire
   build-id matches the Go constant). A `match=no` means the AddOn copy/F5/
   restart was missed — repeat step 2.
4. **Restart the bot** and verify `BOOT INTEGRITY OK … goldens PASS`,
   `bundle-rev == binary rev`, `hello handshake OK`, positions `count=0`.
5. **Carve-outs on this tree** (differ from vl@192ef87d9 BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL20 — partner sync (release #20, vl `2afdd65fa`, DS-105, 2026-10-13)

Syncs the partner tree to release #20 (vl `2afdd65fafee…`, tag
`v2026.10.13.3`). No NEW AddOn change in #20 — the AddOn build id stays
`2026-10-13-a` (same 5 `.cs` files as REL19).

1. **Build at the sha named in `deploy/RELEASE`** (`a91575e129…` — the REL20
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto` to fetch it) and verify `go version -m vl-bin` shows
   `vcs.revision=a91575e129…` `vcs.modified=false`. A FRESH install builds at
   `main` HEAD and re-arms `deploy/RELEASE` + `GUIDE_BUILT_REV`, then
   `(cd web && npm run build)`.
2. **AddOn** — unchanged from REL19 (`VL_BUILD_ID 2026-10-13-a`). If the
   partner machine is still on `2026-10-04-d1`, copy all 5 `ninjascript/*.cs`
   files, F5 compile, FULL NT8 restart (HARD RULE — no hot reload). If it is
   already on `2026-10-13-a`, skip this step.
3. **DB backup timer** — the timer moved to `America/Chicago`: re-run
   `bash deploy/install-db-backup.sh` once after install.
4. **Boot-line check:** `nt8 addon: build_id=2026-10-13-a … match=yes`.
5. **Restart the bot** and verify `BOOT INTEGRITY OK … goldens PASS`,
   `bundle-rev == binary rev`, `hello handshake OK`, positions `count=0`.
6. **Carve-outs on this tree** (differ from vl@2afdd65fa BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL22 — partner sync (releases #21 + #22, vl `f3ec96d85`, CTO lane, 2026-10-14)

Syncs the partner tree to release #22 (vl `f3ec96d85f07…`, tag
`v2026.10.14.3`, booted on the owner box 08:22:40 CT 10-14), and with it
release #21 (vl `269745e1a`). Net diff vl `2afdd65fa..f3ec96d85`. **No AddOn
change in #21 or #22**: `ninjascript/` is byte-identical to REL19/REL20, and
the AddOn build id stays `2026-10-13-a`. No `go.mod` or `web/package*.json`
change either (still **Go 1.26.9**).

1. **Build at the sha named in `deploy/RELEASE`** (`e56f7f02e42d…` — the REL22
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=e56f7f02e42d…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`. A FRESH install builds at
   `main` HEAD and re-arms `deploy/RELEASE` to the built sha instead.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-13-a`). If the partner
   machine is still on `2026-10-04-d1`, copy all 5 `ninjascript/*.cs` files,
   F5 compile, FULL NT8 restart (HARD RULE — no hot reload). If it is already
   on `2026-10-13-a`, skip this step.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-13-a … match=yes`, positions `count=0`.
4. **New boot lines to expect on a partner machine:**
   - `🧾 manual origin (R1): … this boot marked=0 [] already=0 skipped=4 ·
     backup=none`. The R1 migration (owner ruling, release #22) is pinned to
     four row ids AND the owner's own trader id AND a reconcile-adoption
     fingerprint, so on a partner database it writes nothing and takes no
     backup. A partner line that says `marked=` anything but 0 is a defect —
     stop and report it.
   - `daily-loss gate: …` — the gate is now **OFF by default**, and an
     unsaved switch reads OFF. If you want it, turn it ON with a dollar limit
     in Studio and save; when ON it refuses new entries, cancels resting
     entries, flattens bot positions and latches until 17:00 CT.
5. **Behaviour changes worth knowing** (all ship ON unless noted):
   mentor trail after the NT8-confirmed +1R partial (knob
   `mentor_trail_mode`: `candle` default, `off` = the pre-#22 rule);
   swing break-even at +1R; sweep setups wait for their trigger; the Studio
   prompt reaches the planner; the weekly read uses CME weeks; holiday early
   close (11-27, 12-24 flat + cancel at 12:15 CT); new entries freeze while an
   Update job is in preflight; `/api/equity-history` now needs login.
   Optional env overrides `CLOCK_STEP_DETECT_MS` / `CLOCK_STEP_FREEZE_MS` /
   `CLOCK_STEP_HOLD_S` default to 100 / 1000 / 120 — no action needed.
6. **Updater worker** (optional for partners): the #22 updater check-order
   change only takes effect once the updater binary is rebuilt; the bot-side
   preflight freeze works without it.
7. **Carve-outs on this tree** (differ from vl@f3ec96d85 BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL23 — partner sync (release #23, vl `7129b7646`, CTO lane, 2026-10-14)

Syncs the partner tree to release #23 (vl `7129b7646af0…`, tag
`v2026.10.14.4`, PR #521, booted on the owner box 11:22:42 CT 10-14), on top of
REL22 (already in partner `main`). Net diff vl `f3ec96d85..7129b7646`. **No AddOn
change**: `ninjascript/` is byte-identical to REL19–REL22, and the AddOn
build id stays `2026-10-13-a`. No `go.mod` or `web/package*.json` change
(still **Go 1.26.9**). No DB migration.

1. **Build at the sha named in `deploy/RELEASE`** (`678254c1caf3…` — the REL23
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** and verify
   `go version -m vl-bin` shows `vcs.revision=678254c1caf3…`
   `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-13-a`). Copy the 5
   `ninjascript/*.cs` + F5 + FULL NT8 restart ONLY if the machine is still on
   `2026-10-04-d1`.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-13-a … match=yes`, positions `count=0`.
4. **What changes (#519, #520):**
   - **Bar gaps refill themselves.** After an NT8 feed outage the bot finds
     any hole of 3+ open minutes (`BARS_HOLE_MIN_GAP_MIN`, optional env,
     default 3) and re-asks NT8 for history at +1/+3/+10/+30 min, never while
     the feed is down. If NT8 still has nothing, ONE
     `🚨 bars hole UNFILLED … Reload All Historical Data` line is logged and
     the bot retries every 15 min — the fix is an NT8 history reload. Holes
     show in `/api/health` `bars_holes`.
   - **ORB / trendline follow the course:** the ORB escape needs the candle
     BODY outside, the ORB side follows the LATEST escape (the old all-day
     lock is gone), the trendline breaks on a 5m CLOSE read at the candle's
     close time. A 2m entry trails on 2m candles (dormant until
     `exec_2m_after_30m` is ON).
   - `mentor_trail_tf` values `30s`/`45s` are now REFUSED at config load
     with a log line and run as `1m` (the bot has no sub-minute bars).
5. **Carve-outs on this tree** (differ from vl@7129b7646 BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL24 — partner sync (releases #24 + #25, vl `e52203393`, DS-108, 2026-10-15)

Syncs the partner tree to release #24 (vl `e522033935dcd…`, tag
`v2026.10.14.5`, booted on the owner box 07:43:57 CT 10-15), and with it #25,
on top of REL23 (already in partner `main`). Net diff vl
`7129b7646..e52203393` (121 commits, 17 merges). No `go.mod` or
`web/package*.json` change (still **Go 1.26.9**, `GOTOOLCHAIN=auto`).

**AddOn CHANGES this release**: `VL_BUILD_ID 2026-10-14-a` (was
`2026-10-13-a`). One `.cs` file changed — `ninjascript/VLTraderTCPClient.cs`
(+ `vltrader_tcp_PROTOCOL.md` doc). The fix (AC-1): `close_position` honours
its account field, fail-closed — one AddOn serving two MNQ traders on two SIM
accounts no longer flattens the wrong account on an Emergency Flat / EOD flat.
No wire shape change; `ExpectedAddonBuild` is `2026-10-14-a` in lockstep.

1. **Build at the sha named in `deploy/RELEASE`** (`06413856e2a9…` — the REL24
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=06413856e2a9…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — copy `ninjascript/VLTraderTCPClient.cs` (build
   `2026-10-14-a`), F5 compile, FULL NT8 restart (HARD RULE — no hot reload).
   The other 4 `ninjascript/*.cs` files are unchanged from REL19/REL20.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-14-a … match=yes`, positions `count=0`.
4. **Behaviour changes worth knowing** (release #24; all ship ON unless
   noted): planner touch-fills wait for the touch reaction; exec follow-ups
   (a BE stop is not ignored, a NOT APPLIED fill is not re-sent, dead-man needs
   an order snapshot, the cutoff holds); mentor seed-state replay is not
   re-seeded on a latch; the updater preflight freeze also withdraws resting
   entries; machine tokens may READ and REDUCE risk but never ADD it (deny by
   default); Emergency Flat / Positions → Close now honour the bound account
   end-to-end (the AddOn fix above).
5. **Updater worker** (optional for partners): the W26 PART C updater-addon-proof
   fix lives in the updater binary and only takes effect after it is rebuilt
   via `deploy/install-updater-worker.sh`; the bot-side behaviour needs no
   updater change.
6. **Carve-outs on this tree** (differ from vl@e52203393 BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL27 — partner sync (releases #26 + #27, vl `4ee511df8`, DS-108, 2026-10-15)

Syncs the partner tree to release #27 (vl `4ee511df8bd75…`, tag
`v2026.10.15.2`, booted on the owner box 13:43:31 CT 10-15), and with it
release #26 (`v2026.10.15.1`, `dde67b760`). Net diff vl
`e52203393..4ee511df8` (62 commits, 20 merges, the W26 wave folds + the #27
stale-print tf-bound). **No AddOn change** (`VL_BUILD_ID 2026-10-14-a`), no
`go.mod` or `web/package*.json` change (still **Go 1.26.9**,
`GOTOOLCHAIN=auto`), no DB migration.

1. **Build at the sha named in `deploy/RELEASE`** (`a5ddd43e8549…` — the REL27
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=a5ddd43e8549…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-14-a`); no F5 needed if the
   machine is already on `2026-10-14-a`.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-14-a … match=yes`, positions `count=0`.
4. **What changes (#26 + #27, all ship ON unless noted):** the W26 folds —
   risk-ledger truth, planner stale-zone truth, dead-man escalation, bar
   stale-print, UI/studio truth, alert down-episodes, mentor P2S, mentor
   test-clock, fdr-clock (Stop wakes promptly in the halt backoff),
   updater-addon-proof (already-compiled AddOn is not re-F5'd), test-clock-arms
   (arm fixtures pinned off the wall clock) — plus #27's stale-print tf-bound.
   Machine tokens may READ and REDUCE risk but never ADD it (deny by default).
5. **Carve-outs on this tree** (differ from vl@4ee511df8 BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL28 — partner sync (release #28, vl `2444616cc`, DS-108, 2026-10-15)

Syncs the partner tree to release #28 (vl `2444616cc6929…`, tag
`v2026.10.15.3`, booted on the owner box 16:34:51 CT 10-15; RELEASE marker
`0e1b74721`), on top of REL27 (already in partner `main`). Net diff vl
`4ee511df8..2444616cc` (6 commits: #548 test race + #549 addon stale-reemit
HTF cross-check). **No AddOn change** (`VL_BUILD_ID 2026-10-14-a`), no `go.mod`
or `web/package*.json` change (still **Go 1.26.9**).

1. **Build at the sha named in `deploy/RELEASE`** (`9ca59b0da728…` — the REL28
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=9ca59b0da728…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-14-a`); no F5 needed on a machine
   already at `2026-10-14-a`.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-14-a … match=yes`, positions `count=0`.
4. **What changes (#548, #549):** a test-race fix in
   `trader/run_owner_alert_episode_test.go` (test-only); the AddOn stale-reemit
   cross-check — the stale-print path re-verifies the bar on the HTF tape
   before re-emitting (`provider/ninjatrader/bar_stale_print.go`,
   `bar_cache.go`), its tests, the Updates guide line (`status.ts`), and an
   AUDIT-CHECKLIST addendum.
5. **Carve-outs on this tree** (differ from vl@2444616cc BY DESIGN, the same 8
   as every prior sync): `.github/workflows/release.yml`,
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`, this
   runbook, `internal/updatersource/source.go`.

## REL30 — partner sync (release #30, vl `a27936e28`, DS-107, 2026-10-16)

Syncs the partner tree to release #30 (vl `a27936e2812…`, tag `v2026.10.16.1`,
owner "FULL FOR VLTRADER TOO"; RELEASE marker published by the CTO). Net diff
vl `2444616cc..a27936e28` (release #29 #550/#551 + release #30 #555/#556/#557/
#558/#560 + #553/#554 CI/test). **No AddOn change** (`VL_BUILD_ID 2026-10-14-a`),
no `go.mod` or `web/package*.json` change (still **Go 1.26.9**).

1. **Build at the sha named in `deploy/RELEASE`** (`b2d43e1df032…` — the REL30
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=b2d43e1df032…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-14-a`); no F5 needed on a machine
   already at `2026-10-14-a`.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-14-a … match=yes`, positions `count=0`.
4. **What changes (#29 + #30):** box watermark re-judgement (P2-N2), one-entry-
   per-side (no flip), the day-seed replay + its testdata, the planner-HTF
   veto-snapshot, stale-print HTF cross-check follow-ups, CI coverage timeout
   30m→60m (#553/#554). All ship ON unless a knob says otherwise; machine
   tokens may READ and REDUCE risk but never ADD it.
5. **Carve-outs on this tree — now 9** (differ from vl@a27936e28 BY DESIGN):
   `.github/workflows/release.yml` (partner `RELEASE_REPO`, no `if: false`
   guards, release job `permissions: contents: write`), `deploy/release_allowed_signers`
   (the PARTNER key), `deploy/release_contract_test.go` (partner version),
   `deploy/partner_signers_test.go` (partner-only), `branding/census_test.go`,
   `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_source_pin_test.go`, this runbook, and
   `internal/updatersource/source.go` (ReleaseRepo line).

## REL31 — partner sync (release #31, vl `f31beef13`, DS-108, 2026-10-16)

Syncs the partner tree to release #31 (vl `f31beef1345…`, tag `v2026.10.16.2`,
published by the CTO). Net diff vl `a27936e28..f31beef13` (release #31: #559
fix/course-exits + #561 fix/course-defaults-30). **No AddOn change**
(`VL_BUILD_ID 2026-10-14-a`), no `go.mod` or `web/package*.json` change (still
**Go 1.26.9**).

1. **Build at the sha named in `deploy/RELEASE`** (`80ad38870f85…` — the REL31
   tree-sync commit): `git checkout main && git pull --ff-only origin main`,
   build at that sha from a clean clone with **Go 1.26.9** (or
   `GOTOOLCHAIN=auto`) and verify `go version -m vl-bin` shows
   `vcs.revision=80ad38870f85…` `vcs.modified=false`. Build the web with
   `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn** — unchanged (`VL_BUILD_ID 2026-10-14-a`); no F5 needed on a machine
   already at `2026-10-14-a`.
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-14-a … match=yes`, positions `count=0`.
4. **What changes (#31):** mentor exits follow the course — R1 marketable leg-1
   take-profit modify through `modify_bracket` (no AddOn change), R2 swing 4h
   trail, defect-6 PHL EMA, the candle-3 partial is a marketable limit; and
   course-defaults-30 — mentor defaults aligned with the course (R8 ceiling,
   reduced tier reach 30; `htf_gate_news_only` + `stop_after_loss` reverted to
   the course text). The #559 swing changes were REVERTED to dev before the
   merge.
5. **Carve-outs on this tree — 10** (differ from vl@f31beef13 BY DESIGN):
   `.github/workflows/release.yml` (partner `RELEASE_REPO`, no `if: false`
   guards, release job `permissions: contents: write`),
   `deploy/release_allowed_signers` (the PARTNER key),
   `deploy/release_contract_test.go` (partner version),
   `deploy/partner_signers_test.go` (partner-only), `branding/census_test.go`,
   `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/release_source_pin_test.go`, this runbook, and
   `internal/updatersource/source.go` (ReleaseRepo line).

## REL32 — partner sync (release #32, vl `a22dd3fad`, DS-105, 2026-10-17)

Syncs the partner tree to release #32 (vl `a22dd3fad6…` — the release #32 tip,
which includes the THE ONE RULE home-wipe guards #575 and the census token fix
that followed). Net diff vl `f31beef13..a22dd3fad` (release #32: #564
planner-4-split, #568 mentor-size-4 (4 = 2+2), #570 risk-knob-truth, #569
planner-knob-truth, #563 leg-3 bracket, #565 orb-course-rules, #572
done-after-win readers, #574 exited-candle pin, #576 mentor-knob-truth, #577
knob-recheck-fixes, #575 home-wipe-guards, then the homeguard round-3 + census
token fix).
**AddOn CHANGED** (`VL_BUILD_ID 2026-10-14-a` → `2026-10-16-a`); no `go.mod` or
`web/package*.json` change (still **Go 1.26.9**).

1. **Build at the sha named in `deploy/RELEASE`** (the REL32 tree-sync commit):
   `git checkout main && git pull --ff-only origin main`, build at that sha from
   a clean clone with **Go 1.26.9** (or `GOTOOLCHAIN=auto`) and verify
   `go version -m vl-bin` shows `vcs.revision=<that sha>` `vcs.modified=false`.
   Build the web with `VITE_GUIDE_BUILT_REV=<that sha> npm run build`.
2. **AddOn — CHANGED** (`VL_BUILD_ID 2026-10-16-a`; leg-3 bracket): copy
   `ninjascript/VLTraderTCPClient.cs` into the NT8 AddOns folder, **F5 compile,
   full NT8 restart** (the HARD RULE — copy, compile, full restart, no hot
   reload).
3. **Restart the bot in a flat window** and verify `BOOT INTEGRITY OK …
   goldens PASS`, `bundle-rev == binary rev`, `hello handshake OK`,
   `nt8 addon: build_id=2026-10-16-a … match=yes`, positions `count=0`.
4. **What changes (#32):** mentor 4-contract = 2+2 sizing, the leg-3 bracket (the
   AddOn change), planner 4-way split, mentor/risk/planner knob truth fixes,
   orb course rules, done-after-win readers, the exited-candle double-exit guard,
   and the THE ONE RULE home-wipe guards (`internal/safepath`,
   `internal/testhome`, `internal/homeguard` + a `homeguard_main_test.go` in
   every test package).
5. **Carve-outs on this tree — 16, machine-pinned in `deploy/partner_sync_carveouts`**
   (differ from vl@a22dd3fad BY DESIGN): `.github/workflows/release.yml`
   (partner `RELEASE_REPO`, release job `permissions: contents: write`),
   `branding/census_test.go`, `deploy/RELEASE`, `deploy/install-updater-worker.sh`,
   `deploy/partner-first-time-setup.sh` (partner-only),
   `deploy/partner_first_time_setup_test.go` (partner-only),
   `deploy/partner_signers_test.go` (partner-only), `deploy/partner_sync_carveouts`
   (the pinned list itself), `deploy/partner_sync_carveouts_test.go` (partner-only),
   `deploy/release_allowed_signers` (the PARTNER key),
   `deploy/release_contract_test.go`, `deploy/release_source_pin_test.go`,
   `deploy/systemd-user/vl-updater.service` (partner-only),
   `docs/superpowers/runbooks/2026-09-27-one-button-update.md` (partner-only),
   this runbook, and `internal/updatersource/source.go` (ReleaseRepo line).

## AUTO — partner sync (vl v2026.10.17.4, DS-105 auto-sync, 2026-10-17)

Auto-synced to vl tag `v2026.10.17.4` (`80e5464b6a2e273a8547d18e377213d81e9d4739…`). Build commit `de63614ea22f90fe1a67044843d7e122932c7ff9…` (see deploy/RELEASE).
AddOn VL_BUILD_ID: 2026-10-16-a
Carve-outs: 19 machine-pinned paths kept (deploy/partner_sync_carveouts).
