package activation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// RollbackTo restores ALL THREE halves. "Restore the binary" is the version of
// this step people write, and it leaves the UI serving the new bundle and the
// marker claiming the new sha — the box lying about itself at exactly the
// moment someone is trying to find out what it is running.
func TestRollbackRestoresTheDistAndTheMarkerNotJustTheBinary(t *testing.T) {
	prev, install := twoReleases(t)
	// prev is the OLD release we roll back TO; install is where the process
	// reads from, currently holding the NEW content.
	withSystem(t, &system{
		ReadStat: func(pid int) (string, error) { return statLine(pid, 7), nil },
		Kill:     func(int) error { return nil },
		MainPID:  func() (int, error) { return 99, nil },
		Now:      time.Now,
		Sleep:    func(time.Duration) {},
	})
	next, rc, err := RollbackTo(prev, install, Identity{PID: 42, StartTicks: 7})
	if err != nil {
		t.Fatalf("RollbackTo: %v", err)
	}
	bin, _ := os.ReadFile(install.Binary)
	if string(bin) != "NEWBIN" {
		t.Fatalf("binary = %q, want the rolled-back content", bin)
	}
	dist, _ := os.ReadFile(filepath.Join(install.Dist, "index.html"))
	if string(dist) != "NEW" {
		t.Fatalf("dist = %q — the dist half was NOT restored", dist)
	}
	marker, _ := os.ReadFile(install.ReleaseFile)
	if strings.TrimSpace(string(marker)) != prev.SHA {
		t.Fatalf("RELEASE = %q, want %s — the marker half was NOT restored", marker, prev.SHA)
	}
	if next.PID != 99 || !rc.OK {
		t.Fatalf("expected the NEW identity and an OK receipt, got pid %d ok=%v", next.PID, rc.OK)
	}
}

func TestRollbackRefusesToSignalARecycledPID(t *testing.T) {
	prev, install := twoReleases(t)
	killed := false
	withSystem(t, &system{
		ReadStat: func(pid int) (string, error) { return statLine(pid, 424242), nil },
		Kill:     func(int) error { killed = true; return nil },
		MainPID:  func() (int, error) { return 99, nil },
		Now:      time.Now,
		Sleep:    func(time.Duration) {},
	})
	if _, _, err := RollbackTo(prev, install, Identity{PID: 42, StartTicks: 7}); err == nil {
		t.Fatal("RollbackTo signalled a pid it could not confirm")
	}
	if killed {
		t.Fatal("RollbackTo KILLED a recycled pid")
	}
}

// Under the release-dir layout all three halves move with ONE symlink, and the
// swap must never leave a moment with no `current` at all.
func TestRollbackRepointsCurrentAtomicallyAndRecordsWhatItDid(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, strings.Repeat("a", 40))
	newDir := filepath.Join(root, strings.Repeat("b", 40))
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	current := filepath.Join(root, "current")
	if err := os.Symlink(newDir, current); err != nil {
		t.Fatal(err)
	}
	withSystem(t, &system{
		ReadStat: func(pid int) (string, error) { return statLine(pid, 5), nil },
		Kill:     func(int) error { return nil },
		MainPID:  func() (int, error) { return 7, nil },
		Now:      time.Now,
		Sleep:    func(time.Duration) {},
	})
	prev := Release{Dir: oldDir, SHA: strings.Repeat("a", 40)}
	_, rc, err := Rollback(prev, Identity{PID: 3, StartTicks: 5})
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	got, err := os.Readlink(current)
	if err != nil {
		t.Fatalf("current is not a symlink after rollback: %v", err)
	}
	if got != oldDir {
		t.Fatalf("current -> %s, want %s", got, oldDir)
	}
	// Evidence must describe what was ACTUALLY done. An earlier draft of this
	// function copied a file onto itself and then reported
	// restored="binary,dist,RELEASE" — a fabricated value (A24) that would
	// have read as a successful three-half restore in the job record.
	if rc.Evidence["method"] != "current symlink repointed" {
		t.Fatalf("evidence must name the mechanism, got %+v", rc.Evidence)
	}
	if rc.Evidence["restored"] != "" {
		t.Fatal("evidence claims a file-copy restore that this path does not perform")
	}
}

// A `current` that is left dangling is worse than a failed rollback: the next
// boot serves nothing and the failure is reported nowhere.
//
// THE FIRST VERSION OF THIS TEST DID NOT WORK, and the mutation run is the only
// reason I know. It asserted the END STATE (current points at b, no staging
// link left behind) — and an implementation that does `os.Remove(name)` and
// then re-creates it passes that assertion perfectly, because the gap exists
// only DURING the call. An end-state assertion cannot see an invariant that is
// about the middle. So this observes the middle: a poller watches the name
// while the swap runs many times, and any single moment where it is absent
// fails the test.
func TestAtomicSymlinkNeverLeavesTheNameMissing(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	name := filepath.Join(root, "current")
	if err := atomicSymlink(a, name); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	missing := make(chan string, 1)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := os.Lstat(name); os.IsNotExist(err) {
				select {
				case missing <- "current was absent mid-swap":
				default:
				}
				return
			}
		}
	}()

	for i := 0; i < 400; i++ {
		target := a
		if i%2 == 0 {
			target = b
		}
		if err := atomicSymlink(target, name); err != nil {
			close(stop)
			t.Fatalf("repoint %d: %v", i, err)
		}
	}
	close(stop)

	select {
	case msg := <-missing:
		t.Fatalf("%s — a reader between the unlink and the create sees no release at all", msg)
	default:
	}

	got, err := os.Readlink(name)
	if err != nil {
		t.Fatalf("current unreadable after the swaps: %v", err)
	}
	if got != a && got != b {
		t.Fatalf("current -> %q, want one of the two targets", got)
	}
	if _, err := os.Lstat(name + ".swapping"); !os.IsNotExist(err) {
		t.Fatal("the staging symlink was left behind")
	}
}
