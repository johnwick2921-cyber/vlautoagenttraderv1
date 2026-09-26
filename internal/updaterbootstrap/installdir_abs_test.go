package updaterbootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/updateauth"
)

// PR #200 fold #18: --install-dir is absolutized at the CLI entry. Before
// the fold a RELATIVE --install-dir passed every precondition (the DB stat
// resolves against the cwd) and the operator typed the confirmation — then
// updateauth refused the relative data dir inside ("data dir unset or not
// absolute"). The pins drive the real entry (Run) from the directory the
// relative path is relative to.

// A relative --install-dir enrolls into — and authorizes from — the same
// absolute data dir the absolute spelling resolves.
func TestRelativeInstallDirIsAbsolutizedAtEntry(t *testing.T) {
	attended(t)
	inst := install(t)
	t.Setenv("DB_PATH", "x")
	os.Unsetenv("DB_PATH")
	t.Chdir(filepath.Dir(inst))
	rel := filepath.Base(inst)
	if filepath.IsAbs(rel) {
		t.Fatalf("precondition: %q is not relative", rel)
	}

	rc, out, errb := run(rel, enrollLine(bEmail), "enroll", bEmail)
	if rc != 0 {
		t.Fatalf("relative --install-dir %q: enroll rc=%d stderr=%s", rel, rc, errb)
	}
	d := DataDirFor(inst) // the absolute spelling's data dir
	if !filepath.IsAbs(d) {
		t.Fatalf("precondition: %q is not absolute", d)
	}
	if a, err := updateauth.LoadAdmin(d); err != nil || a.Email != bEmail {
		t.Fatalf("the enrollment is not in %s: admin=%+v err=%v", d, a, err)
	}
	if !strings.Contains(out, "dir="+updateauth.Dir(d)) {
		t.Fatalf("enroll did not report the absolute dir %s: %q", updateauth.Dir(d), out)
	}

	rc, out, errb = run(rel, authorizeLine(bRel), "authorize", bRel)
	if rc != 0 || !strings.Contains(out, "hmac") {
		t.Fatalf("relative --install-dir %q: authorize rc=%d out=%q stderr=%s", rel, rc, out, errb)
	}
	g, err := updateauth.ParseInstallRequest(strings.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	key, _ := updateauth.LoadDeviceKey(d)
	a, err := updateauth.LoadAdmin(d) // red-4 #4b: the MAC carries the enrolled admin's user_id
	if err != nil {
		t.Fatal(err)
	}
	if !updateauth.VerifyMAC(key, a.UserID, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Fatal("the grant minted through the relative spelling does not verify under the install's key")
	}
}

// A relative --install-dir that names nothing is still refused at the
// precondition (rc 2, before any prompt), and the refusal names the
// ABSOLUTE path it looked at. Nothing is created under the cwd.
func TestRelativeNonexistentInstallDirIsStillRefused(t *testing.T) {
	attended(t)
	t.Setenv("DB_PATH", "x")
	os.Unsetenv("DB_PATH")
	cwd := t.TempDir()
	t.Chdir(cwd)
	for _, args := range [][]string{{"enroll", bEmail}, {"authorize", bRel}} {
		stdin := enrollLine(bEmail)
		if args[0] == "authorize" {
			stdin = authorizeLine(bRel)
		}
		rc, out, errb := run("no-such-install", stdin, args...)
		want := filepath.Join(cwd, "no-such-install", "data", "data.db")
		if rc != 2 || !strings.Contains(errb, "no bot database at "+want) {
			t.Fatalf("%v: rc=%d stderr=%q, want rc 2 naming %s", args, rc, errb, want)
		}
		if strings.Contains(errb, "Type exactly") || strings.Contains(out, "hmac") {
			t.Fatalf("%v: prompted or minted: out=%q stderr=%q", args, out, errb)
		}
		if _, err := os.Lstat(filepath.Join(cwd, "no-such-install")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%v: something was created under the cwd: %v", args, err)
		}
	}
}
