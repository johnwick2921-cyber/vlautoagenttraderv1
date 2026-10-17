package deploy

// DS-105 auto-sync (2026-10-17) — vl-auto-sync.yml STARTS release.yml (workflow_dispatch),
// never a signer. The signing key (the `release` Environment secret
// RELEASE_SIGNING_KEY) must live ONLY inside release.yml, behind the `release`
// Environment's required-reviewer gate. vl-auto-sync.yml runs on a schedule and
// on workflow_dispatch with only GITHUB_TOKEN (contents / pull-requests /
// issues), so a secret referenced there would be a key that fires without a
// human approval.
//
// Pinned here: vl-auto-sync.yml contains no `secrets.` reference at all, and it
// starts release.yml with only the `tag` string input — never a
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
	// Run 39073507478: called as a reusable workflow, release.yml saw the
	// `release` Environment secret EMPTY and refused to sign. The caller must
	// START release.yml as its own run on the tag instead.
	if strings.Contains(y, "uses: ./.github/workflows/release.yml") {
		t.Fatalf("vl-auto-sync.yml must not call release.yml as a reusable workflow — the Environment key is empty there (run 39073507478)")
	}
	if !strings.Contains(y, `gh workflow run release.yml -R "$GITHUB_REPOSITORY" --ref "$TAG" -f tag="$TAG"`) {
		t.Fatalf("vl-auto-sync.yml must start release.yml on the tag (--ref the tag, tag input = the tag)")
	}
	// The tag reaches bash only as env DATA from merge-and-tag, and is
	// refused unless it is a v* tag.
	if !strings.Contains(y, "TAG: ${{ needs.merge-and-tag.outputs.tag }}") {
		t.Fatalf("vl-auto-sync.yml must pass the merge-and-tag output to the release step via env:")
	}
	if !strings.Contains(y, `case "$TAG" in v*)`) {
		t.Fatalf("vl-auto-sync.yml must refuse a non-v* tag before starting a release")
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
