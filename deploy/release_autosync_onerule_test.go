package deploy

// Auto-sync retry safety (2026-10-17, runs 39069638349 / 39070410198).
//
// The first run past the cache-path fix pushed auto-sync/<tag> and then failed
// at `gh pr create`; every retry was then rejected at the push because the
// branch already existed. THE ONE RULE forbids deleting it or force-pushing
// over it, so a retry pushes to a fresh name. Pinned here, with the rest of
// THE ONE RULE for this pipeline:
//   - the merge never passes --delete-branch, and nothing force-pushes;
//   - an existing auto-sync branch is detected and a fresh name is used;
//   - the last-synced-tag state file is committed BEFORE the one push (a
//     best-effort second push could fail silently and merge a sync that never
//     records its tag);
//   - the workflow runs one sync at a time (the 10-min schedule).

import (
	"strings"
	"testing"
)

func TestAutoSyncNeverDeletesOrForcePushes(t *testing.T) {
	y := repoFile(t, ".github/workflows/vl-auto-sync.yml")
	s := repoFile(t, "deploy/vl-auto-sync.sh")
	for name, body := range map[string]string{"vl-auto-sync.yml": y, "vl-auto-sync.sh": s} {
		for _, bad := range []string{"--delete-branch", "push --force", "push -f ", "--force-with-lease", "push origin +", "push origin :", "--delete "} {
			if strings.Contains(body, bad) {
				t.Errorf("%s contains %q — THE ONE RULE: no branch deletion, no force push", name, bad)
			}
		}
	}
}

func TestAutoSyncRetryUsesAFreshBranch(t *testing.T) {
	s := repoFile(t, "deploy/vl-auto-sync.sh")
	if !strings.Contains(s, `git ls-remote --exit-code origin "refs/heads/$BRANCH"`) {
		t.Fatal("vl-auto-sync.sh must detect an auto-sync branch left by a failed run before pushing")
	}
	if !strings.Contains(s, `BRANCH="auto-sync/${NEW_TAG}-${GITHUB_RUN_ID:-`) {
		t.Fatal("vl-auto-sync.sh must push a retry to a fresh branch name, never over the old one")
	}
}

func TestAutoSyncStateCommittedBeforeThePush(t *testing.T) {
	s := repoFile(t, "deploy/vl-auto-sync.sh")
	state := strings.Index(s, `git commit --quiet -m "record last synced vl tag: $NEW_TAG"`)
	push := strings.Index(s, `git push origin "HEAD:$BRANCH"`)
	if state < 0 || push < 0 {
		t.Fatalf("anchors missing (state commit %d, push %d)", state, push)
	}
	if state > push {
		t.Fatal("the last-synced-tag commit must come BEFORE the push, so the PR always carries it")
	}
	if strings.Count(s, `git push origin "HEAD:$BRANCH"`) != 1 {
		t.Fatal("exactly one push of the sync branch — no best-effort second push")
	}
}

func TestAutoSyncRunsOneAtATime(t *testing.T) {
	y := repoFile(t, ".github/workflows/vl-auto-sync.yml")
	if !strings.Contains(y, "concurrency:\n  group: vl-auto-sync\n  cancel-in-progress: false") {
		t.Fatal("vl-auto-sync.yml must hold a workflow-level concurrency group (no cancel) — one sync at a time")
	}
}
