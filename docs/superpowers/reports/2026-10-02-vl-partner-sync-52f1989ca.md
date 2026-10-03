# Partner sync verification — nofx@52f1989ca0a5 (2026-10-02, forward of the 92d50acdb sync)

- Source: nofx `52f1989ca0a5ba4a4f126caa8b9bd2c0d790b96b` (bundle #308, booted
  19:49:37 CT 2026-10-02 as v2026.10.02.6; RELEASE marker commit `d473be657`).
  CTO order 2026-10-02 ~20:21 CT: carry the sync forward from `92d50acdb` to
  `52f1989ca0a5` on the same branch.
- Branch: `sync/vl-92d50acdb-20261002` (continued per CTO order), stacked on
  `sync/vl-pb-pc-20261002` @ `1f50c8687` (PR #18 base, rebased by DS-106).
- Method: forward net-diff `92d50acdb..52f1989ca0a5` — 76 files (74 shared,
  2 carve-out paths), then full-tree `git ls-tree` net-diff of the partner head
  against nofx@52f1989ca0a5.

## Match table (full-tree net-diff vs nofx@52f1989ca0a5)

- **4,669 paths byte-identical** with nofx@52f1989ca0a5 (was 4,576 at 92d50acdb).
- Forward-range shared files: 73 taken byte-for-byte from nofx (the 74th is the
  new carve-out below).

| path | disposition | nofx blob | partner blob |
|---|---|---|---|
| .github/workflows/release.yml | carve-out (no `push: tags` trigger) | f8d975e2ad5f | 8b593a6bb9aa |
| deploy/RELEASE | carve-out (partner stamp; re-stamped this sync) | 437f7cf55420 | eef64c1550fe |
| deploy/install-updater-worker.sh | carve-out (partner REPO_URL) | 10c0546434c1 | 3c70a3e6939f |
| deploy/release_allowed_signers | carve-out (partner signing key placeholder) | 13abac5bf01b | ac459624c12d |
| deploy/release_contract_test.go | carve-out (partner asserts) | 0dc3bf6dfbae | 9ffab2eb135a |
| deploy/updater_worker_install_test.go | carve-out (partner REPO_URL assert) | 0a9d41d61e81 | 6d1c27bc2c92 |
| internal/updatersource/source.go | carve-out (ONE-line `ReleaseRepo = "johnwick2921-cyber/vlautoagenttraderv1"`) | 447b9b640c90 | 5053fb7b13eb |
| branding/census_test.go | carve-out (partner census pin, re-derived on the nofx@52f1989ca0a5 base) | 019b65e68d08 | 6624a7dc2e5c |
| docs/superpowers/reports/2026-09-22-vl-partner-verification.md | partner-only doc | — | 6942ad5673c4 |
| docs/superpowers/runbooks/2026-09-22-vl-partner-update.md | partner-only doc | — | c74d36977c99 |
| docs/superpowers/reports/2026-10-02-vl-partner-sync-92d50acdb.md | partner-only doc | — | 1afb9cc27de3 |
| deploy/release_source_pin_test.go | **NEW carve-out #11** — deleted on partner (see below) | 0f88fefb6cc8 | — |

**Carve-out #11 (new this sync):** nofx added `deploy/release_source_pin_test.go`
in the forward range. Both of its tests are nofx-side repo-identity pins that
cannot hold on the partner tree:
`TestReleaseSourceConstantEqualsWorkflowRepo` (declared again → redeclaration
with the partner's own B1 in the carve-out `deploy/release_contract_test.go`)
and `TestReleaseSourceConstantNeverNamesThePartnerRepo` (asserts the build's
release source never names the partner — the partner's constant MUST name the
partner). nofx's own file comment says the partner repo carries the same test
with its own value; the partner's mirror asserts (B1, both directions) already
live in `deploy/release_contract_test.go`. Removed on partner, commit
`1802c254c`.

## Census (carve-out re-derived, CTO ruling pattern)

- Base: nofx@52f1989ca0a5 `branding/census_test.go` (adds the #307
  `e1dcc173…` legacy-job fixture row, 11 hits; ceiling 1171).
- Partner table: 3 re-pins + 5 adds for partner-only content (self-row 10);
  ceiling = exact sum **1218** (no slack); dated reason lines on every re-pin.
- `go test ./branding/` green (30.957s and 36.5s in the two suite runs).

## Gates (recipe: `HOME=$(mktemp -d) GOMODCACHE=/home/hoang/go/pkg/mod GOMAXPROCS=4 nice -n 19 ionice -c 3`)

- `go build ./...` — green.
- `go vet ./...` — green (after carve-out #11).
- Secret scan (sk-/AKIA/ghp_/private-key patterns over the forward diff) — 0 hits.
- Web: `npm ci` rc 0 · `npm run build` green 4.90s (VITE_GUIDE_BUILT_REV = full sha).
- Full suite run 1: FAIL `vl/telegram TestBotReMintsItsTokenWhenAPasswordChangeRetiresIt`
  ("the re-minted bot token reads as stale", bot_token_test.go:167) — flake under
  concurrent npm build load; isolated re-run PASS 2.803s, full `./telegram/`
  package PASS 12.653s.
- Full suite run 2: `vl/agent` compile `signal: killed` (OOM; box memory
  pressure) — isolated `./agent/` PASS 85.513s; `vl/telegram` PASS 13.484s;
  every other package `ok`.
- **Full `-race` suite — GREEN**, rc 0, 47 packages, no DATA RACE lines:
  `ok vl/api 536.385s` · `ok vl/updaterbootstrap 112.724s` ·
  `ok vl/updaterworker 201.934s` · `ok vl/store 398.913s` ·
  `ok vl/trader 1486.819s` · `ok vl/trader/ninjatrader 190.324s`.

## Commit-of-build proof

- Binary built from a clean `--shared` clone of the partner repo at
  `1802c254c19a9a08904db7e435ed9a7db609c0e6` (linked worktrees do not carry
  `vcs.*` stamping).
- `go version -m`: `vcs.revision=1802c254c19a9a08904db7e435ed9a7db609c0e6`,
  `vcs.time=2026-10-03T01:29:02Z`, `vcs.modified=false`; md5
  `aea52c9e9980e9e3e5f238cc7c4130b9`.
- `deploy/RELEASE` re-stamped to `1802c254c…` (partner blob `eef64c1550fe`).

## PR

`DO NOT MERGE UNTIL OWNER — partner sync: 52f1989ca` on base
`sync/vl-pb-pc-20261002` @ `1f50c8687`; URL quoted in the bridge DONE report.
