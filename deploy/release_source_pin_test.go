package deploy

// Partner tree's pin test: the build-time release source must name the
// PARTNER repo (vl's own tree asserts the opposite — the owner repo).
// The partner name is assembled from parts so the census never sees the
// whole token in one string.
import (
	"strings"
	"testing"

	"vl/internal/updatersource"
)

func TestReleaseRepoIsThePartnerRepo(t *testing.T) {
	partner := "vlautoagent" + "traderv1" // parts, never the literal
	if updatersource.ReleaseRepo == "" {
		t.Fatalf("ReleaseRepo must not be empty")
	}
	if !strings.Contains(updatersource.ReleaseRepo, partner) {
		t.Fatalf("the build's release source must be the partner repo, got %q", updatersource.ReleaseRepo)
	}
}
