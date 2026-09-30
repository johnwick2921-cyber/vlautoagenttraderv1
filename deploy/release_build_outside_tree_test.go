package deploy

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestReleaseWorkflowGoBuildsWriteOutsideTheTree pins the rule the first
// release run (v2026.09.27.1, 2026-09-27) broke: every `go build -o <target>`
// in release.yml must write either OUTSIDE the checkout ($RUNNER_TEMP or an
// absolute path) or to a git-IGNORED path. A binary written to an untracked,
// non-ignored path makes every LATER build in the same job read
// vcs.modified=true, and the clean-build check refuses the release.
func TestReleaseWorkflowGoBuildsWriteOutsideTheTree(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	re := regexp.MustCompile(`go build[^\n]*?-o\s+("[^"]+"|\S+)`)
	ms := re.FindAllStringSubmatch(y, -1)
	if len(ms) == 0 {
		t.Fatal("release.yml has no `go build -o` — the pin reads nothing")
	}
	root := repoRoot(t)
	for _, m := range ms {
		target := strings.Trim(m[1], `"`)
		if strings.HasPrefix(target, "$RUNNER_TEMP") || strings.HasPrefix(target, "/") || strings.HasPrefix(target, "$OUT/") {
			continue
		}
		cmd := exec.Command("git", "check-ignore", "-q", target)
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			t.Fatalf("release.yml builds into %q: not outside the tree and not git-ignored — the next build in the job would read vcs.modified=true", target)
		}
	}
	if strings.Contains(y, `OUT="$RUNNER_TEMP/updater-bin"`) == false && strings.Contains(y, "$OUT/") {
		t.Fatal(`$OUT is used but not defined as "$RUNNER_TEMP/updater-bin"`)
	}
}
