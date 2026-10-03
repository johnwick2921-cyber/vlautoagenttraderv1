package deploy

import (
	"regexp"
	"strings"
	"testing"

	"vl/internal/updatersource"
)

// TestReleaseSourceConstantEqualsWorkflowRepo pins the B1 fold: the build-time
// release source (internal/updatersource.ReleaseRepo) and the release
// workflow's RELEASE_REPO must be the SAME value. One value, two pins — if
// either changes alone, this test goes RED before any release can be cut.
// (Audit fold B1; the partner repo carries the same test with its own value.)
func TestReleaseSourceConstantEqualsWorkflowRepo(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	m := regexp.MustCompile(`(?m)^\s*RELEASE_REPO:\s*(\S+)`).FindStringSubmatch(y)
	if m == nil {
		t.Fatalf("release.yml must define RELEASE_REPO exactly once")
	}
	if m[1] != updatersource.ReleaseRepo {
		t.Fatalf("workflow RELEASE_REPO %q != build-time constant %q — one side changed alone", m[1], updatersource.ReleaseRepo)
	}
	if updatersource.ReleaseRepo == "" {
		t.Fatalf("ReleaseRepo must not be empty")
	}
}

// The repo is public; the constant must name the owner repo, never the partner.
// The partner name is assembled from parts so the census never sees the whole
// literal (the same rule the census itself uses for its own tokens).
func TestReleaseSourceConstantNeverNamesThePartnerRepo(t *testing.T) {
	partner := "vlautoagent" + "traderv1" // parts, never the literal
	if strings.Contains(updatersource.ReleaseRepo, partner) {
		t.Fatalf("the build's release source must be the owner repo, not the partner repo")
	}
}
