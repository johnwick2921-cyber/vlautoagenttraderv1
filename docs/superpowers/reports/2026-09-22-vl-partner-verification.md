# VL partner update verification — September 22, 2026

Status: prepared and tested locally; not pushed or deployed to partner machines.
Branch: `sync/nofx-0960a6ac-20260922`.
Partner base: `78d35c25eca2e51853339bc5cf10ec30340cf40d`.
Shared nofx source target: `0960a6ac`.
Code/binary revision: `c5890387433dacfbee114224929cb7d8ca98d3d3`.
Release artifact commit: `b1f8559`.

## Actual checks

- Full `go test ./...`: exit 0; 33 packages passed.
- Go build: exit 0; linux/amd64, CGO_ENABLED=1, vcs.modified=false.
- Frontend: 77 files passed, 494 tests passed and 1 skipped. The skipped
  import-target pin names a nofx-history commit absent from the independent
  partner history; the mirror-specific skip predates this update.
- Guide tests after release stamp: 12 passed. TypeScript/Vite build: passed.
- RELEASE and GUIDE_BUILT_REV were derived from the built binary, then dist
  was built; the resulting bundle contains the matching revision.
- Pattern scan of changed files: zero matches for private keys, OpenAI-style
  keys, GitHub tokens, AWS access-key IDs, or runtime .env/database/binary files.
- Compiled binary scan: zero matches for the same credential patterns. This
  is a bounded pattern scan, not a claim of exhaustive secret detection.

Binary SHA-256: `e545bb54640d673e2fa678f39e332e99fbe6c7bb401d094e3b7d1ebb7207ea7c`.
Frontend entry: `assets/index-Dgkn6P1O.js`.

C# compilation/F5, received AddOn build, actual partner runtime/native-data
readiness, and SIM entry/protection receipts remain partner-machine checks.
No local account settings, credentials, databases, or mode settings were copied.

Follow `docs/superpowers/runbooks/2026-09-22-vl-partner-update.md`.
Owner push (reserved to owner by the standing partner-repo rule):

```sh
git -C /tmp/partner-update-20260922/nofx push -u origin sync/nofx-0960a6ac-20260922
```

Open a PR to partner main and recheck its merged HEAD before release if it differs.
