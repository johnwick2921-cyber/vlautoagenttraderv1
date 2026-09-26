package updaterjob

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 24, 18, 2, 11, 123456789, time.UTC)

func mustNew(t *testing.T, jobID, releaseID string, now time.Time) Job {
	t.Helper()
	j, err := New(jobID, releaseID, now)
	if err != nil {
		t.Fatalf("New(%q,%q): %v", jobID, releaseID, err)
	}
	return j
}

func mustWrite(t *testing.T, dd string, j Job) {
	t.Helper()
	if err := Write(dd, j); err != nil {
		t.Fatalf("Write(%s %s/%s): %v", j.JobID, j.State, j.Phase, err)
	}
}

// step is the runner's shape at the persistence boundary: Enter + Write
// BEFORE the effect; receipt + Finish + Write after it.
func step(t *testing.T, dd string, j *Job, now *time.Time, to State) {
	t.Helper()
	*now = now.Add(time.Second)
	if err := j.Enter(to, *now); err != nil {
		t.Fatalf("Enter(%s): %v", to, err)
	}
	mustWrite(t, dd, *j)
	if j.Phase == PhaseStarted {
		r, _ := Lookup(to)
		*now = now.Add(time.Second)
		if err := j.AddReceipt(Receipt{Step: string(r.Effect), StartedAt: now.Add(-time.Second), EndedAt: *now, OK: true}, *now); err != nil {
			t.Fatal(err)
		}
		if err := j.Finish(*now); err != nil {
			t.Fatalf("Finish(%s): %v", to, err)
		}
		mustWrite(t, dd, *j)
	}
}

// restoreSeams puts every seam back after a test replaces it.
func restoreSeams(t *testing.T) {
	e, o, f, r, d := geteuid, ownerOf, fsyncFile, renameFile, fsyncDir
	t.Cleanup(func() { geteuid, ownerOf, fsyncFile, renameFile, fsyncDir = e, o, f, r, d })
}

func tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		fi, _ := os.Lstat(p)
		out = append(out, rel+" "+fi.Mode().String())
		return nil
	})
	sort.Strings(out)
	return out
}

var errCrash = errors.New("simulated crash")

// TestWriteIsTmpFsyncRenameAndPrivate: the production Write goes temp file
// in the SAME dir (0600 from birth) → fsync → rename over the target → fsync
// of the dir, in that order; the target is never partial at any point — a
// crash before the rename leaves the old job whole, one after it the new job
// whole; the files are 0600 in 0700 dirs; and an existing target that is a
// symlink, a loose dir or a foreign owner is refused, never repaired.
func TestWriteIsTmpFsyncRenameAndPrivate(t *testing.T) {
	restoreSeams(t)
	// w is what the seams observe: the target of the write in progress and
	// its bytes before the write (nil = absent).
	var w struct {
		t      *testing.T
		target string
		old    []byte
		ops    []string
		crash  string
	}
	seeTarget := func(where string) {
		b, err := os.ReadFile(w.target)
		switch {
		case w.old == nil && !errors.Is(err, fs.ErrNotExist):
			w.t.Errorf("%s: target visible before the rename (%v, %d bytes)", where, err, len(b))
		case w.old != nil && !bytes.Equal(b, w.old):
			w.t.Errorf("%s: target changed before the rename:\n%s", where, b)
		}
	}
	fsyncFile = func(f *os.File) error {
		if filepath.Dir(f.Name()) != filepath.Dir(w.target) || !strings.HasPrefix(filepath.Base(f.Name()), ".job-") || !strings.HasSuffix(f.Name(), ".tmp") {
			w.t.Errorf("fsync of %s: not a temp file in the jobs dir", f.Name())
		}
		seeTarget("fsync")
		w.ops = append(w.ops, "fsync-file")
		if w.crash == "fsync-file" {
			return errCrash
		}
		return f.Sync()
	}
	renameFile = func(from, to string) error {
		fi, err := os.Lstat(from)
		if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
			w.t.Errorf("rename source %s: %v %v, want a regular 0600 file", from, err, fi)
		}
		if b, _ := os.ReadFile(from); !json.Valid(b) {
			w.t.Errorf("rename source %s is not a complete JSON document", from)
		}
		if filepath.Dir(from) != filepath.Dir(to) || to != w.target {
			w.t.Errorf("rename %s → %s: not a same-dir rename onto the target", from, to)
		}
		seeTarget("rename")
		w.ops = append(w.ops, "rename")
		if w.crash == "rename" {
			return errCrash
		}
		return os.Rename(from, to)
	}
	fsyncDir = func(dir string) error {
		jobs := filepath.Dir(w.target)
		switch dir {
		case jobs:
			w.ops = append(w.ops, "fsync-dir")
			if w.crash == "fsync-dir" {
				return errCrash
			}
		case filepath.Dir(jobs), filepath.Dir(filepath.Dir(jobs)):
			// the parent of a dir this write's Mkdir created
			// (TestFirstWriteMakesEveryNewDirDurable owns that rule)
			w.ops = append(w.ops, "fsync-parent")
		default:
			w.t.Errorf("fsync-dir %s, want %s or one of its parents", dir, jobs)
		}
		return realFsyncDir(dir)
	}

	dd := t.TempDir()
	j := mustNew(t, "job-0001", "v1.2.0", t0)
	target, err := Path(dd, j.JobID)
	if err != nil {
		t.Fatal(err)
	}
	jobs := filepath.Dir(target)
	w.t, w.target = t, target
	mustWrite(t, dd, j)
	// a first write into a fresh data dir creates <data>/updater and its jobs
	// dir, each made durable (parent fsync) before the job file is written
	if got := strings.Join(w.ops, ","); got != "fsync-parent,fsync-parent,fsync-file,rename,fsync-dir" {
		t.Fatalf("durability order = %s, want fsync-parent,fsync-parent,fsync-file,rename,fsync-dir", got)
	}
	for p, want := range map[string]os.FileMode{target: 0o600, jobs: 0o700 | fs.ModeDir, filepath.Dir(jobs): 0o700 | fs.ModeDir} {
		fi, err := os.Lstat(p)
		if err != nil || fi.Mode() != want {
			t.Errorf("%s: mode %v (%v), want %v", p, fi.Mode(), err, want)
		}
	}
	v1, _ := os.ReadFile(target)
	if got, err := Read(dd, j.JobID); err != nil || got.State != StateRequested {
		t.Fatalf("read back: %+v, %v", got, err)
	}

	// Crash at every durability point: the target is the old job or the new
	// job, whole — never partial — and a failed dir fsync is still an error
	// (the caller must not run the side effect it persisted for).
	for _, crashAt := range []string{"fsync-file", "rename", "fsync-dir"} {
		t.Run("crash at "+crashAt, func(t *testing.T) {
			dd := t.TempDir()
			now := t0
			j := mustNew(t, "job-0002", "v1.2.0", now)
			target, _ := Path(dd, j.JobID)
			w.t, w.target, w.old, w.crash = t, target, nil, ""
			mustWrite(t, dd, j)
			before, _ := os.ReadFile(target)
			next := j
			now = now.Add(time.Second)
			if err := next.Enter(StateDownloaded, now); err != nil {
				t.Fatal(err)
			}
			w.old, w.crash, w.ops = before, crashAt, nil
			if err := Write(dd, next); !errors.Is(err, errCrash) {
				t.Fatalf("Write with a crash at %s = %v, want the crash error", crashAt, err)
			}
			w.crash = ""
			after, _ := os.ReadFile(target)
			got, err := Read(dd, j.JobID)
			if err != nil {
				t.Fatalf("after a crash at %s the job is unreadable: %v\n%s", crashAt, err, after)
			}
			wantState := StateRequested
			if crashAt == "fsync-dir" {
				wantState = StateDownloaded // renamed: the new job, whole
			} else if !bytes.Equal(after, before) {
				t.Errorf("crash at %s changed the target", crashAt)
			}
			if got.State != wantState {
				t.Errorf("crash at %s: state %s, want %s", crashAt, got.State, wantState)
			}
			if tmps, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".job-*.tmp")); len(tmps) != 0 {
				t.Errorf("crash at %s left %v (Write's own error path must remove its temp file)", crashAt, tmps)
			}
		})
	}
	w.t, w.target, w.old = t, target, v1
	fsyncFile, renameFile, fsyncDir = func(f *os.File) error { return f.Sync() }, os.Rename, realFsyncDir

	// A real crash leaves its temp file behind: readers never see it.
	_ = os.WriteFile(filepath.Join(jobs, ".job-123.tmp"), []byte(`{"schema":1,"job_id":"job-00`), 0o600)
	if got, err := Read(dd, j.JobID); err != nil || got.JobID != j.JobID {
		t.Fatalf("a leftover temp file broke Read: %v", err)
	}
	if all, err := List(dd); err != nil || len(all) != 1 {
		t.Fatalf("a leftover temp file broke List: %d jobs, %v", len(all), err)
	}
	if b, _ := os.ReadFile(target); !bytes.Equal(b, v1) {
		t.Fatal("the job changed")
	}

	// Refusals: never written through, never repaired.
	t.Run("target is a symlink", func(t *testing.T) {
		dd := t.TempDir()
		j := mustNew(t, "job-0003", "v1.2.0", t0)
		mustWrite(t, dd, j)
		target, _ := Path(dd, j.JobID)
		outside := filepath.Join(dd, "outside.json")
		_ = os.WriteFile(outside, []byte("OUTSIDE"), 0o600)
		_ = os.Remove(target)
		_ = os.Symlink(outside, target)
		if err := Write(dd, j); !errors.Is(err, ErrUnsafe) {
			t.Errorf("Write over a symlink = %v, want ErrUnsafe", err)
		}
		if b, _ := os.ReadFile(outside); string(b) != "OUTSIDE" {
			t.Error("wrote through the symlink")
		}
	})
	for name, mode := range map[string]os.FileMode{"jobs dir 0755": 0o755, "updater dir 0750": 0o750} {
		t.Run(name, func(t *testing.T) {
			dd := t.TempDir()
			j := mustNew(t, "job-0004", "v1.2.0", t0)
			mustWrite(t, dd, j)
			target, _ := Path(dd, j.JobID)
			dir := filepath.Dir(target)
			if strings.HasPrefix(name, "updater") {
				dir = filepath.Dir(dir)
			}
			_ = os.Chmod(dir, mode)
			if err := Write(dd, j); err == nil {
				t.Errorf("Write with %s succeeded", name)
			}
			if fi, _ := os.Stat(dir); fi.Mode().Perm() != mode {
				t.Errorf("%s was repaired to %v", name, fi.Mode().Perm())
			}
		})
	}
	t.Run("target file 0644", func(t *testing.T) {
		dd := t.TempDir()
		j := mustNew(t, "job-0005", "v1.2.0", t0)
		mustWrite(t, dd, j)
		target, _ := Path(dd, j.JobID)
		_ = os.Chmod(target, 0o644)
		if err := Write(dd, j); !errors.Is(err, ErrUnsafe) {
			t.Errorf("Write over a 0644 job = %v, want ErrUnsafe", err)
		}
	})
	t.Run("foreign owner", func(t *testing.T) {
		dd := t.TempDir()
		j := mustNew(t, "job-0006", "v1.2.0", t0)
		mustWrite(t, dd, j)
		geteuid = func() int { return os.Geteuid() + 1 }
		defer func() { geteuid = os.Geteuid }()
		if err := Write(dd, j); err == nil {
			t.Error("Write into another uid's dir succeeded")
		}
		if _, err := Read(dd, j.JobID); err == nil {
			t.Error("Read of another uid's job succeeded")
		}
	})
}

func realFsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// TestReadRefusesUnsafeAndForgedFiles: the loader refuses what updateauth's
// loaders refuse (symlink, non-regular, loose mode, foreign owner, oversize,
// a symlinked dir) and every content forgery — unknown or re-cased keys, a
// duplicate key, trailing data, another schema, a job id that is not the
// file's, a history the table does not allow, a hand-edited layout.
func TestReadRefusesUnsafeAndForgedFiles(t *testing.T) {
	restoreSeams(t)
	fresh := func(t *testing.T) (string, string, []byte) {
		dd := t.TempDir()
		now := t0
		j := mustNew(t, "job-0010", "v1.2.0", now)
		mustWrite(t, dd, j)
		step(t, dd, &j, &now, StateDownloaded)
		p, _ := Path(dd, j.JobID)
		b, _ := os.ReadFile(p)
		return dd, p, b
	}
	for name, mut := range map[string]func(t *testing.T, dd, p string, b []byte){
		"symlink": func(t *testing.T, dd, p string, b []byte) {
			o := filepath.Join(dd, "o.json")
			_ = os.WriteFile(o, b, 0o600)
			_ = os.Remove(p)
			_ = os.Symlink(o, p)
		},
		"fifo": func(t *testing.T, dd, p string, b []byte) {
			_ = os.Remove(p)
			if err := syscall.Mkfifo(p, 0o600); err != nil {
				t.Skip("mkfifo:", err)
			}
		},
		"mode 0640": func(t *testing.T, dd, p string, b []byte) { _ = os.Chmod(p, 0o640) },
		"oversize": func(t *testing.T, dd, p string, b []byte) {
			_ = os.WriteFile(p, bytes.Repeat([]byte(" "), MaxFileBytes+1), 0o600)
		},
		"jobs symlink": func(t *testing.T, dd, p string, b []byte) { mvDirToSymlink(t, filepath.Dir(p)) },
		"file of another uid": func(t *testing.T, dd, p string, b []byte) {
			real := ownerOf
			ownerOf = func(fi fs.FileInfo) (uint32, bool) {
				uid, ok := real(fi)
				if fi.Name() == filepath.Base(p) {
					uid++
				}
				return uid, ok
			}
			t.Cleanup(func() { ownerOf = real })
		},
	} {
		t.Run(name, func(t *testing.T) {
			dd, p, b := fresh(t)
			mut(t, dd, p, b)
			if _, err := Read(dd, "job-0010"); !errors.Is(err, ErrUnsafe) {
				t.Errorf("Read = %v, want ErrUnsafe", err)
			}
		})
	}
	for name, edit := range map[string]func(s string) string{
		"unknown key grant": func(s string) string { return strings.Replace(s, `"schema": 1,`, `"schema": 1, "grant": "x",`, 1) },
		"unknown key requested_by": func(s string) string {
			return strings.Replace(s, `"schema": 1,`, `"schema": 1, "requested_by": "u1",`, 1)
		},
		"re-cased key": func(s string) string { return strings.Replace(s, `"state":`, `"STATE":`, 1) },
		"duplicate key": func(s string) string {
			return strings.Replace(s, `"schema": 1,`, `"schema": 1, "state": "complete",`, 1)
		},
		"trailing data": func(s string) string { return s + "{}\n" },
		"schema 2":      func(s string) string { return strings.Replace(s, `"schema": 1,`, `"schema": 2,`, 1) },
		"other job id":  func(s string) string { return strings.Replace(s, `"job_id": "job-0010"`, `"job_id": "job-0011"`, 1) },
		"state rewound": func(s string) string {
			return strings.Replace(s, `"state": "downloaded",`+"\n  \"phase\"", `"state": "activated",`+"\n  \"phase\"", 1)
		},
		"minified":       func(s string) string { return strings.Join(strings.Fields(s), "") },
		"not an object":  func(s string) string { return "[]\n" },
		"empty":          func(s string) string { return "" },
		"null receipts":  func(s string) string { return replaceBlock(s, `"receipts": [`, `"receipts": null`) },
		"history forged": func(s string) string { return strings.Replace(s, `"state": "requested",`, `"state": "verified",`, 1) },
	} {
		t.Run(name, func(t *testing.T) {
			dd, p, b := fresh(t)
			s := edit(string(b))
			if s == string(b) {
				t.Fatalf("edit %q did not change the file", name)
			}
			_ = os.WriteFile(p, []byte(s), 0o600)
			if _, err := Read(dd, "job-0010"); !errors.Is(err, ErrCorrupt) {
				t.Errorf("Read = %v, want ErrCorrupt", err)
			}
		})
	}
	// Crafted in the writer's exact byte form (so only the history rules can
	// refuse them): a job BORN mid-flight — parked at nt8_updated, where the
	// attended resume would act on it — and one whose history skips a state.
	born := Job{Schema: SchemaVersion, JobID: "job-0010", ReleaseID: "v1.2.0", State: StateNT8Updated, Phase: PhaseDone,
		CreatedAt: t0, UpdatedAt: t0, Transitions: []Transition{{State: StateNT8Updated, Phase: PhaseDone, At: t0}}, Receipts: []Receipt{}}
	// (the done step carries its receipt and honest counts, so only the skip
	// can refuse it — TestDoneStepCarriesItsReceipt owns the receipt rule)
	skip := Job{Schema: SchemaVersion, JobID: "job-0010", ReleaseID: "v1.2.0", State: StatePreflightOK, Phase: PhaseStarted, Attempts: 1,
		CreatedAt: t0, UpdatedAt: t0, Transitions: []Transition{{State: StateRequested, Phase: PhaseDone, At: t0}, {State: StateDownloaded, Phase: PhaseStarted, At: t0},
			{State: StateDownloaded, Phase: PhaseDone, At: t0, Receipts: 1}, {State: StatePreflightOK, Phase: PhaseStarted, At: t0, Receipts: 1}},
		Receipts: []Receipt{{Step: "download", StartedAt: t0, EndedAt: t0, OK: true}}}
	for name, j := range map[string]Job{"born at nt8_updated": born, "verified skipped": skip} {
		t.Run(name, func(t *testing.T) {
			dd, p, _ := fresh(t)
			b, err := encode(j)
			if err != nil {
				t.Fatal(err)
			}
			_ = os.WriteFile(p, b, 0o600)
			if _, err := Read(dd, "job-0010"); !errors.Is(err, ErrCorrupt) {
				t.Errorf("Read = %v, want ErrCorrupt", err)
			}
			d2 := t.TempDir()
			if err := Write(d2, j); err == nil {
				t.Error("Write accepted it")
			}
		})
	}

	// positive control: the unedited file reads.
	dd, _, _ := fresh(t)
	if j, err := Read(dd, "job-0010"); err != nil || j.State != StateDownloaded || j.Phase != PhaseDone {
		t.Fatalf("positive control: %+v %v", j, err)
	}
}

// replaceBlock replaces the JSON array that starts at open (through its
// matching "]") with repl.
func replaceBlock(s, open, repl string) string {
	i := strings.Index(s, open)
	if i < 0 {
		return s
	}
	depth := 0
	for k := i + len(open) - 1; k < len(s); k++ {
		switch s[k] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[:i] + repl + s[k+1:]
			}
		}
	}
	return s
}

func mvDirToSymlink(t *testing.T, dir string) {
	t.Helper()
	real := dir + ".real"
	if err := os.Rename(dir, real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, dir); err != nil {
		t.Fatal(err)
	}
}

// forgedIDs: the ids api.TestJobRoutesNeverServeAFile sends (URL-decoded),
// plus every shape that could leave the jobs dir or name another file.
var forgedIDs = []string{
	"secret", "secret.json", "../secret.json", "../../secret.json", "..", strings.Repeat("a", 300),
	"device.key", "admin.json",
	"", ".", "/", "/etc/passwd", "job-0001/../../x", "a/b/c/d/e/f", "job-0001/", "job-0001.json",
	"JOB-00000001", "-job-0001", "job-0001 ", " job-0001", "job-0001\n", "job-0001\x00", "job\\0001",
	"%2e%2e", "..%2fsecret.json", "ｊｏｂ-０００１", "job-0001​", ".jobs.lock", ".job-123.tmp",
	"hold", "worker.sock", "seen_job_ids", "a1234567", // a1234567: valid shape — control below
	strings.Repeat("a", 7), strings.Repeat("a", 65),
}

// TestJobPathRefusesEveryForgedID: Path, Read, Write and New refuse every
// forged id BEFORE touching the filesystem — nothing is created, nothing
// outside the jobs dir is read — and the valid shapes still work.
func TestJobPathRefusesEveryForgedID(t *testing.T) {
	restoreSeams(t)
	dd := t.TempDir()
	const secret = "TOP-SECRET-SENTINEL-9f1c"
	_ = os.MkdirAll(filepath.Join(dd, "updater"), 0o700)
	for _, p := range []string{filepath.Join(dd, "secret.json"), filepath.Join(dd, "updater", "secret.json"), filepath.Join(dd, "updater", "admin.json")} {
		_ = os.WriteFile(p, []byte(`{"s":"`+secret+`"}`), 0o600)
	}
	before := tree(t, dd)
	valid := map[string]bool{"a1234567": true}
	for _, id := range forgedIDs {
		if valid[id] {
			continue
		}
		if p, err := Path(dd, id); !errors.Is(err, ErrBadJobID) || p != "" {
			t.Errorf("Path(%q) = %q, %v — want ErrBadJobID", id, p, err)
		}
		if j, err := Read(dd, id); !errors.Is(err, ErrBadJobID) || j.JobID != "" || strings.Contains(j.Error, secret) {
			t.Errorf("Read(%q) = %+v, %v — want ErrBadJobID", id, j, err)
		}
		if _, err := New(id, "v1.2.0", t0); err == nil {
			t.Errorf("New(%q) accepted a forged id", id)
		}
		forged := mustNew(t, "job-0001", "v1.2.0", t0)
		forged.JobID = id
		if err := Write(dd, forged); err == nil {
			t.Errorf("Write accepted job id %q", id)
		}
	}
	if after := tree(t, dd); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Errorf("forged ids touched the filesystem:\nbefore %v\nafter  %v", before, after)
	}
	// controls: the valid shapes resolve INSIDE the jobs dir.
	for _, id := range []string{"a1234567", strings.Repeat("a", 8), strings.Repeat("z", 64), "0123456789abcdef0123456789abcdef", "job-0001"} {
		p, err := Path(dd, id)
		if err != nil || p != filepath.Join(dd, "updater", "jobs", id+".json") {
			t.Errorf("Path(%q) = %q, %v", id, p, err)
		}
	}
	for _, bad := range []string{"", "relative/data", "data"} {
		if _, err := Path(bad, "job-0001"); !errors.Is(err, ErrBadDataDir) {
			t.Errorf("Path(data dir %q) = %v, want ErrBadDataDir", bad, err)
		}
	}
	// forged release ids are refused too (a release id reaches a filesystem
	// join in the worker).
	for _, rid := range []string{"", "..", "../v1", "v1/../../x", "/abs", "v1 ", strings.Repeat("v", 65)} {
		if _, err := New("job-0001", rid, t0); err == nil {
			t.Errorf("New accepted release id %q", rid)
		}
	}
}

// TestNoJobFileReadsAsAbsentAndCreatesNothing (L4): with no job file — no
// updater dir, or no jobs dir — Read is ErrNotFound and List is empty, and
// neither creates a directory: the reader the API will call leaves today's
// filesystem byte-identical.
func TestNoJobFileReadsAsAbsentAndCreatesNothing(t *testing.T) {
	restoreSeams(t)
	for _, withUpdater := range []bool{false, true} {
		dd := t.TempDir()
		if withUpdater {
			_ = os.MkdirAll(filepath.Join(dd, "updater"), 0o700)
		}
		before := tree(t, dd)
		if _, err := Read(dd, "job-0001"); !errors.Is(err, ErrNotFound) {
			t.Errorf("updater=%v: Read = %v, want ErrNotFound", withUpdater, err)
		}
		all, err := List(dd)
		if err != nil || len(all) != 0 {
			t.Errorf("updater=%v: List = %v, %v; want empty", withUpdater, all, err)
		}
		if after := tree(t, dd); strings.Join(after, ",") != strings.Join(before, ",") {
			t.Errorf("updater=%v: a read created %v (was %v)", withUpdater, after, before)
		}
	}
}

// TestWriteRefusesAForbiddenEdgeOrARewrite: the validator binds at the
// persistence call site — a job moved outside the table, or a write that does
// not EXTEND the job on disk (rewound, identity changed, receipt rewritten,
// finished job edited, file born mid-flight), is refused and the file on
// disk is unchanged.
func TestWriteRefusesAForbiddenEdgeOrARewrite(t *testing.T) {
	restoreSeams(t)
	dd := t.TempDir()
	now := t0
	j := mustNew(t, "job-0020", "v1.2.0", now)
	mustWrite(t, dd, j)
	step(t, dd, &j, &now, StateDownloaded)
	step(t, dd, &j, &now, StateVerified)
	p, _ := Path(dd, j.JobID)
	onDisk, _ := os.ReadFile(p)

	unchanged := func(name string) {
		t.Helper()
		if b, _ := os.ReadFile(p); !bytes.Equal(b, onDisk) {
			t.Errorf("%s: the file on disk changed", name)
		}
	}

	// Enter refuses a forbidden edge and leaves the job as it was.
	k := j
	if err := k.Enter(StateMaintenanceHeld, now); !errors.Is(err, ErrForbiddenEdge) {
		t.Errorf("Enter(skip preflight) = %v, want ErrForbiddenEdge", err)
	}
	if k.State != j.State || len(k.Transitions) != len(j.Transitions) {
		t.Error("a refused Enter changed the job")
	}

	cases := map[string]func(k *Job){
		"state set directly (skips preflight_ok)": func(k *Job) { k.State, k.Phase = StateMaintenanceHeld, PhaseStarted },
		"forbidden edge appended by hand (skips preflight)": func(k *Job) {
			k.Transitions = append(k.Transitions, Transition{State: StateMaintenanceHeld, Phase: PhaseStarted, At: now, Receipts: len(k.Receipts)})
			k.State, k.Phase, k.Attempts = StateMaintenanceHeld, PhaseStarted, 1
		},
		"activated appended by hand": func(k *Job) {
			k.Transitions = append(k.Transitions, Transition{State: StateActivated, Phase: PhaseStarted, At: now, Receipts: len(k.Receipts)})
			k.State, k.Phase, k.Attempts = StateActivated, PhaseStarted, 1
		},
		"rewound to downloaded": func(k *Job) {
			k.Transitions = k.Transitions[:3]
			k.State, k.Phase = StateDownloaded, PhaseDone
			k.Receipts = k.Receipts[:1]
		},
		"release id changed": func(k *Job) { k.ReleaseID = "v9.9.9" },
		"created_at changed": func(k *Job) { k.CreatedAt = k.CreatedAt.Add(-time.Hour) },
		"receipt rewritten":  func(k *Job) { k.Receipts = append([]Receipt(nil), k.Receipts...); k.Receipts[0].OK = false },
		// (the rewritten time stays between its neighbours, so only the
		// append-only rule — not the time order — can refuse it)
		"transition rewritten": func(k *Job) {
			k.Transitions = append([]Transition(nil), k.Transitions...)
			k.Transitions[1].At = k.Transitions[1].At.Add(-time.Millisecond)
		},
	}
	for name, edit := range cases {
		k := j
		k.Transitions = append([]Transition(nil), j.Transitions...)
		k.Receipts = append([]Receipt(nil), j.Receipts...)
		edit(&k)
		if err := Write(dd, k); !errors.Is(err, ErrCorrupt) && !errors.Is(err, ErrRewrite) {
			t.Errorf("%s: Write = %v, want refused", name, err)
		}
		unchanged(name)
	}

	// A file is born only as New's job: nothing is created mid-flight.
	mid := j
	mid.JobID = "job-0021"
	if err := Write(dd, mid); !errors.Is(err, ErrRewrite) {
		t.Errorf("create mid-flight: Write = %v, want ErrRewrite", err)
	}
	if _, err := Read(dd, "job-0021"); !errors.Is(err, ErrNotFound) {
		t.Errorf("mid-flight create left a file: %v", err)
	}

	// A finished job is immutable.
	now = now.Add(time.Second)
	if err := j.Enter(StateCancelled, now); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dd, j)
	fin, _ := os.ReadFile(p)
	e := j
	e.Blocker = "edited after the end"
	if err := Write(dd, e); !errors.Is(err, ErrRewrite) {
		t.Errorf("edit a finished job: Write = %v, want ErrRewrite", err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, fin) {
		t.Error("a finished job changed")
	}
	mustWrite(t, dd, j) // an identical re-write is a no-op, not an error
	// positive control: the legal path writes.
	d2 := t.TempDir()
	now = t0
	g := mustNew(t, "job-0022", "v1.2.0", now)
	mustWrite(t, d2, g)
	for _, s := range []State{StateDownloaded, StateVerified, StatePreflightOK, StateMaintenanceHeld} {
		step(t, d2, &g, &now, s)
	}
}
