# Partner sync verification — nofx@92d50acdb (2026-10-02)

- Source: nofx `92d50acdb` ("Merge pull request #293 … release2", password
  install booted 10:17 CT 2026-10-02).
- Branch: `sync/vl-92d50acdb-20261002`, stacked on `sync/vl-pb-pc-20261002`
  @ `1f50c8687` (PR #18, rebased by DS-106 onto #16 head `638556547`;
  base move only, tree identical) — quoted as the PR base.
- Method: tree-level net-diff (`git ls-tree -r --format='%(objectname) %(path)'`)
  between nofx@92d50acdb and the partner head. 4,576 paths byte-identical;
  69 paths taken from nofx (50 overwrites + 19 adds), including the P0
  test-isolation guard
  `internal/updaterbootstrap/home_isolation_guard_test.go` (nofx blob
  `8178a19f4078…`, commit `a0470035a` ancestor of `92d50acdb` [A]) — present
  and byte-identical in the partner tree. The 9 remaining differences are the
  partner carve-outs below — every path, blob sha, and reason.

## Match table (the ONLY differences)

| path | nofx blob | partner blob | reason |
|---|---|---|---|
| .github/workflows/release.yml | f8d975e2ad5f | 8b593a6bb9aa | partner guard: no `push: tags` trigger; release only via owner-approved workflow_dispatch (P-B) |
| deploy/RELEASE | a4f35975f83e | 709d3400c2c0 | partner stamp: names the partner build commit (re-stamped after build) |
| deploy/install-updater-worker.sh | 10c0546434c1 | 3c70a3e6939f | partner REPO_URL default (vlautoagenttraderv1) |
| deploy/release_allowed_signers | 13abac5bf01b | ac459624c12d | partner placeholder for the partner's OWN signing key (owner step B2) |
| deploy/release_contract_test.go | 0dc3bf6dfbae | 9ffab2eb135a | partner asserts (pins the partner ReleaseRepo) |
| deploy/updater_worker_install_test.go | 0a9d41d61e81 | 6d1c27bc2c92 | partner REPO_URL assert |
| internal/updatersource/source.go | 447b9b640c90 | 5053fb7b13eb | ONE-line carve-out: `ReleaseRepo = "johnwick2921-cyber/vlautoagenttraderv1"`; the rest is nofx's P-A implementation byte-for-byte |
| docs/superpowers/reports/2026-09-22-vl-partner-verification.md | — | 6942ad5673c4 | partner-only doc (prior sync verification report) |
| docs/superpowers/runbooks/2026-09-22-vl-partner-update.md | — | c74d36977c99 | partner-only doc (partner update runbook) |

## Gates

- `go build ./...` — green.
- `go vet ./...` — green.
- Secret scan (sk-/AKIA/ghp_/private-key patterns over the sync diff) — 0 hits.
- Targeted package tests + full suite + one race: run per the box timing
  rules (lunch band 12:00–13:30 CT excludes the full trader-inclusive suite;
  race after 15:30 CT). Results appended below when they land.
