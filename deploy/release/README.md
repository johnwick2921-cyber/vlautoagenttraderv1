# Release packaging (W-ONE-BUTTON M4)

`.github/workflows/release.yml` cuts a signed, scanned release from an
**approved tag**. These scripts are the parts that must be runnable — and
testable — outside GitHub, so their guarantees are pinned by
`deploy/release_contract_test.go` rather than trusted.

| Script | Refuses when |
|---|---|
| `secret-scan.sh <tree>` | a denied path (`.env`, `*.db`, `*.key`, `id_*`, `data/`, `backups/`, …) or secret-shaped content is present; also runs `gitleaks` when installed, and SAYS SO when it is not — an absent scanner is never reported as a clean bill |
| `check-archive-paths.sh <path>…` | a path is absolute, contains `..`, is home-relative, or carries a newline |
| `package.sh <repo> <stage>` | a required allow-list artifact is missing, a staged path is unsafe, or a symlink is staged |
| `manifest.sh <stage> <sha40> <id>` | the source sha is not 40 hex |

The staged tree and the unpacked archive are scanned **separately**. They are
different things, and a check that only sees one of them can be bypassed by
whatever happens in between.

## Owner action, once: the release keypair

The workflow signs `manifest.json` and then verifies that signature against the
**committed public key** before anything is uploaded, so a signature the shipped
key cannot check never leaves the runner. That requires a keypair the owner
creates once:

```
ssh-keygen -t ed25519 -C nofx-release -f nofx-release-key   # no passphrase
cp nofx-release-key.pub deploy/release.pub                  # COMMIT this half
# paste the PRIVATE half (nofx-release-key) into the repo's
# Settings → Environments → release → secret RELEASE_SIGNING_KEY
shred -u nofx-release-key                                   # keep no local copy
```

Until `deploy/release.pub` exists the workflow **refuses** with that instruction
rather than a file-not-found. The private half never appears in the repository,
in a log, or on a developer machine.

## Open owner decision

`RELEASE_REPO` is where artifacts are published. It defaults **fail-closed** to
this repository (`johnwick2921-cyber/nofx`) and is one line to change. The
partner mirror is never a valid target, and the contract test fails if its name
appears in the workflow.
