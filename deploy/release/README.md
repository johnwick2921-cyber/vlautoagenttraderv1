# Release packaging (W-ONE-BUTTON M4)

`.github/workflows/release.yml` cuts a signed, scanned release from an
**approved tag**. These scripts are the parts that must be runnable — and
testable — outside GitHub, so their guarantees are pinned by
`deploy/release_contract_test.go` rather than trusted.

| Script | Refuses when |
|---|---|
| `secret-scan.sh <tree>` | a denied path (`.env`, `*.db`, `*.key`, `id_*`, `data/`, `backups/`, …) or secret-shaped content is present; also runs `gitleaks`. With `GITLEAKS_REQUIRED=1` (set by the release job) an absent gitleaks **fails**; locally it prints a NOTE — an absent scanner is never reported as a clean bill |
| `check-archive-paths.sh <path>…` | a path is absolute, contains `..`, is home-relative, or carries a newline |
| `package.sh <repo> <stage> <sha40>` | a required allow-list artifact is missing, the source sha is not 40 hex, a staged path is unsafe, or a symlink is staged. It **writes** `deploy/RELEASE` from the source sha rather than copying the checked-in one — that file is the marker the *boot procedure* wrote for the *previous* boot, so copying it ships a sha that disagrees with the binary beside it |
| `manifest.sh <stage> <sha40> <id>` | the source sha is not 40 hex |

The staged tree and the unpacked archive are scanned **separately**. They are
different things, and a check that only sees one of them can be bypassed by
whatever happens in between.

## Owner action, once: the release keypair

The workflow signs `manifest.json` and then verifies that signature **before
anything is uploaded**, so a signature the shipped key cannot check never leaves
the runner. That requires a keypair the owner creates once:

```
ssh-keygen -t ed25519 -C nofx-release -f nofx-release-key   # no passphrase
echo "release $(cat nofx-release-key.pub)" > deploy/release_allowed_signers
git add deploy/release_allowed_signers                      # COMMIT this half
# paste the PRIVATE half (nofx-release-key) into the repo's
# Settings → Environments → release → secret RELEASE_SIGNING_KEY
shred -u nofx-release-key                                   # keep no local copy
```

**It must be an allowed-signers file, not a bare `.pub`.** `ssh-keygen -Y verify
-f <file>` reads `<file>` as `<principal> <keytype> <base64> [comment]`. A bare
public key begins with the *keytype*, so ssh-keygen takes `ssh-ed25519` as the
principal and `-I release` then matches nothing — verification refuses every
signature, including good ones. `TestReleaseSignatureVerifiesOnlyWithAnAllowedSignersFile`
signs with a real throwaway keypair and proves both directions, so this is not
taken on faith. The 3b updater verifies the same way.

Until `deploy/release_allowed_signers` exists the workflow **refuses** with that
instruction rather than a file-not-found. The private half never appears in the
repository, in a log, or on a developer machine.

## Known limit

The content pass skips files ≥ 2 MB — in practice the binary — because scanning
it byte-wise on every release buys little against a deny-list that already
refuses the shapes secrets arrive in. `gitleaks` covers what the content pass
skips, and in CI gitleaks is **required** (`GITLEAKS_REQUIRED=1`), so the gap
exists only in a local run, where the NOTE says so.

## Open owner decision

`RELEASE_REPO` is where artifacts are published. It defaults **fail-closed** to
this repository (`johnwick2921-cyber/nofx`) and is one line to change. The
partner mirror is never a valid target, and the contract test fails if its name
appears in the workflow.
