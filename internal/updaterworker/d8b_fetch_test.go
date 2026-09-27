package updaterworker

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"nofx/internal/updaterjob"
)

// PIN (D8b, CTO 1790279155144 (4) / 1790280128238 — U4's crash suite): a
// fetch KILLED between its rename and its verdict link leaves a verdict-less
// release dir; the NEXT fetch recovers (the dir is quarantined, the release
// re-materialized, the verdict written) — never "already exists (it may be
// the running release)" forever — and the verdict-less dir is never read as
// a verified release. The fetch's own pending marker (written before the
// rename) is what makes the dir provably the interrupted fetch's.
func TestFetchKilledBetweenRenameAndLinkRecoversOnTheNextRun(t *testing.T) {
	r := buildRelease(t, releaseOpts{})
	e := newFetchEnv(t)
	orig := fetchAfterRename
	t.Cleanup(func() { fetchAfterRename = orig })
	fetchAfterRename = func() { panic("killed between rename and link") }
	func() {
		defer func() { recover() }()
		FetchRelease(e.cfg(r))
	}()
	if got := dirNames(t, e.releaseRoot); len(got) != 2 || got[0] != ".pending-"+testSHA || got[1] != testSHA {
		t.Fatalf("after the kill the release root is %v, want the verdict-less [.pending-%s %s]", got, testSHA, testSHA)
	}
	if _, err := os.Lstat(e.verdictPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the kill left a verdict: %v", err)
	}
	if _, err := updaterjob.ReadVerdict(e.dataDir, testReleaseID); err == nil {
		t.Fatal("a verdict-less release reads as verified")
	}
	fetchAfterRename = orig
	v, err := FetchRelease(e.cfg(r))
	if err != nil {
		t.Fatalf("the next fetch is blocked by the interrupted one: %v", err)
	}
	got := dirNames(t, e.releaseRoot)
	if len(got) != 2 || got[1] != testSHA || !strings.HasPrefix(got[0], ".orphan-"+testSHA+"-") {
		t.Fatalf("release root after recovery = %v, want [.orphan-%s-<unix> %s]", got, testSHA, testSHA)
	}
	if n, err := RehashRelease(v); err != nil || n == 0 {
		t.Fatalf("the recovered release does not re-prove: %d, %v", n, err)
	}
}
