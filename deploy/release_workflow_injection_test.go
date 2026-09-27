package deploy

// W-PR-B fold [11] — github.event.inputs.tag and steps.src.outputs.release_id
// are USER-INFLUENCED expressions spliced raw into run: bodies of the job that
// holds RELEASE_SIGNING_KEY. GitHub substitutes the expression's VALUE into the
// script text BEFORE bash sees it, so a tag input like `"; rm -rf …; "` is
// arbitrary code beside the signing key. git tag names can carry $, (), quotes
// and semicolons too, so github.ref_name is equally tainted when raw in a body.
//
// Rule pinned here: inside any run: body, every ${{ … }} expression must be in
// the FIXED allow-list below (values the repo controls, never the tag/input).
// User-influenced values travel ONLY via env: (substituted into the
// environment, where bash reads them as DATA, quoted).
//
// Class-250 probe: this test greps the REAL workflow file — the production
// lines. The mutation that proves it: splicing github.event.inputs.tag (or
// steps.src.outputs.release_id, or github.ref_name) back into a run: body
// fails the assertions below.

import (
	"regexp"
	"strings"
	"testing"
)

var exprRe = regexp.MustCompile(`\$\{\{\s*([^{}]+?)\s*\}\}`)

// fixedExprs is the allow-list: expressions whose value the REPO controls.
var fixedExprs = map[string]bool{
	"steps.src.outputs.sha": true, // git rev-parse HEAD, validated 40-hex
}

func TestReleaseWorkflowRunBodiesCarryNoUserInfluencedExpressions(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	lines := strings.Split(y, "\n")

	inRun := false
	runIndent := 0
	bad := []string{}
	for _, l := range lines {
		trim := strings.TrimLeft(l, " ")
		if inRun {
			cur := len(l) - len(trim)
			if trim != "" && cur <= runIndent {
				inRun = false
			} else {
				for _, m := range exprRe.FindAllStringSubmatch(l, -1) {
					expr := strings.TrimSpace(m[1])
					if !fixedExprs[expr] {
						bad = append(bad, expr+" in: "+strings.TrimSpace(l))
					}
				}
				continue
			}
		}
		if strings.HasPrefix(trim, "run: |") || strings.HasPrefix(trim, "run: >") {
			inRun = true
			runIndent = len(l) - len(trim)
			continue
		}
	}
	if len(bad) > 0 {
		t.Fatalf("user-influenced (or unknown) expressions inside run: bodies — they execute beside RELEASE_SIGNING_KEY:\n  %s",
			strings.Join(bad, "\n  "))
	}

	// The input itself must still be threaded — through env:, never raw.
	if !strings.Contains(y, "TAG: ${{ github.event.inputs.tag") {
		t.Fatalf("the tag input must be passed via a step env: TAG, not spliced into a run body")
	}
	if !strings.Contains(y, "RELEASE_ID: ${{ steps.src.outputs.release_id }}") {
		t.Fatalf("release_id must be passed via env: RELEASE_ID to every step that uses it")
	}
}
