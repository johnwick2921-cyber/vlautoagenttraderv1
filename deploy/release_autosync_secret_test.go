package deploy

// DS-105 auto-sync (2026-10-17) — vl-auto-sync.yml is a CALLER of release.yml,
// never a signer. The signing key (the `release` Environment secret
// RELEASE_SIGNING_KEY) must live ONLY inside release.yml, behind the `release`
// Environment's required-reviewer gate. vl-auto-sync.yml runs on a schedule and
// on workflow_dispatch with only GITHUB_TOKEN (contents / pull-requests /
// issues), so a secret referenced there would be a key that fires without a
// human approval.
//
// Pinned here: vl-auto-sync.yml contains no `secrets.` reference at all, and it
// calls release.yml via `uses:` with only the `tag` string input — never a
// `secrets:` mapping. The mutation that proves it: writing
// `secrets: inherit` (or `secrets.RELEASE_SIGNING_KEY`) into the caller fails
// the assertions below.

import (
	"strings"
	"testing"
)

func TestAutoSyncWorkflowCarriesNoSecrets(t *testing.T) {
	y := repoFile(t, ".github/workflows/vl-auto-sync.yml")

	if strings.Contains(y, "secrets.") {
		t.Fatalf("vl-auto-sync.yml must never reference a secret — the signing key lives only in release.yml behind the `release` Environment")
	}
	if strings.Contains(y, "secrets:") {
		t.Fatalf("vl-auto-sync.yml must not pass a secrets: mapping to the called workflow")
	}
	if !strings.Contains(y, "uses: ./.github/workflows/release.yml") {
		t.Fatalf("vl-auto-sync.yml must call release.yml as a local reusable workflow")
	}

	// The caller hands the called workflow only the `tag` string input (data,
	// not a secret). The called workflow resolves its own Environment and key.
	if !strings.Contains(y, "with:") {
		t.Fatalf("vl-auto-sync.yml must pass the tag input to release.yml via with:")
	}
	if !strings.Contains(y, "tag: ${{ needs.merge-and-tag.outputs.tag }}") {
		t.Fatalf("vl-auto-sync.yml must pass the tag output from merge-and-tag to the release call")
	}
}

// The signing key still resolves from the called workflow's own protected
// Environment — this test ties the two halves together so neither can drift.
func TestAutoSyncSigningKeyStaysInReleaseEnvironment(t *testing.T) {
	r := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(r, "environment: release") {
		t.Fatalf("release.yml must run its signing job in the protected `release` Environment")
	}
	if !strings.Contains(r, "secrets.RELEASE_SIGNING_KEY") {
		t.Fatalf("release.yml must take the signing key only from RELEASE_SIGNING_KEY")
	}
}
