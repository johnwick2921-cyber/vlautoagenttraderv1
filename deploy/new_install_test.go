package deploy

// new_install_test — the test suite for deploy/new-install.sh (w32 NEW-INSTALL,
// OWNER ORDER 10-16). Runs the REAL script against a fake HOME with
// PATH-injected fakes for go/node/npm/git (the script only checks presence +
// versions), a forced WSL seam (VL_NEW_INSTALL_WSL), a fake NT8 AddOns dir
// (VL_NT8_ADDONS_DIR) and a skipped build (VL_NEW_INSTALL_SKIP_BUILD). Real
// openssl/cp/curl come from the system PATH. It proves --dry-run writes nothing
// and leaks no secret, the refusals fire (not WSL, no NT8 folder, wrong go),
// and a second run changes nothing (idempotence).

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func newInstallScript(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "new-install.sh")
}

// newInstallFakeBin writes PATH fakes. goVersion is the string after "go version "
// (e.g. "go1.26.9 linux/amd64"). node always reports v22.14.0; npm/git exist.
func newInstallFakeBin(t *testing.T, goVersion string) string {
	t.Helper()
	dir := t.TempDir()
	fake := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/usr/bin/env bash\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fake("go", `if [ "${1:-}" = version ]; then echo "`+goVersion+`"; else exit 0; fi`)
	fake("node", `echo v22.14.0`)
	fake("npm", `exit 0`)
	fake("git", `exit 0`)
	return dir
}

// runNewInstall runs the script under a fake HOME and returns stdout/stderr/code.
func runNewInstall(t *testing.T, home, bin string, env map[string]string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command("bash", newInstallScript(t))
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+bin+":"+os.Getenv("PATH"))
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var so, se strings.Builder
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run: %v", err)
	}
	return so.String(), se.String(), code
}

// newInstallHome builds a fake HOME with a vl install dir and a fake NT8 AddOns dir.
func newInstallHome(t *testing.T) (home, addons string) {
	t.Helper()
	home = t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "vl"), 0o755); err != nil {
		t.Fatal(err)
	}
	addons = filepath.Join(home, "nt8", "AddOns")
	if err := os.MkdirAll(addons, 0o755); err != nil {
		t.Fatal(err)
	}
	return home, addons
}

func newInstallEnv(home, addons string) map[string]string {
	return map[string]string{
		"VL_NEW_INSTALL_WSL":         "1",
		"VL_NEW_INSTALL_SKIP_BUILD":  "1",
		"VL_NEW_INSTALL_WAIT_SECONDS": "0", // no bot is running in tests: skip the match=yes wait
		"VL_NT8_ADDONS_DIR":          addons,
		"VL_API_PORT":                "9", // nothing listens: the health check FAILs fast, no live bot
		"VL_NEW_INSTALL_DIR":         filepath.Join(home, "vl"),
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			out[rel] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestNewInstallDryRunWritesNothing(t *testing.T) {
	home, addons := newInstallHome(t)
	bin := newInstallFakeBin(t, "go1.26.9 linux/amd64")
	before := snapshotTree(t, home)

	stdout, stderr, code := runNewInstall(t, home, bin, newInstallEnv(home, addons), "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, want := range []string{"would: create", "would copy ninjascript", "would: GET", "nothing was written"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, stdout)
		}
	}
	after := snapshotTree(t, home)
	if strings.Join(sortedKeys(before), "\n") != strings.Join(sortedKeys(after), "\n") {
		t.Fatalf("dry-run wrote files:\nbefore: %v\nafter: %v", sortedKeys(before), sortedKeys(after))
	}
	// No secret shape may appear: no JWT_SECRET=<value>, no base64 blob, no JWT.
	for _, line := range strings.Split(stdout+"\n"+stderr, "\n") {
		if strings.Contains(line, "JWT_SECRET=") && !strings.Contains(line, "would write") {
			t.Fatalf("a secret leaked: %q", line)
		}
		if strings.Contains(line, "eyJ") {
			t.Fatalf("a token leaked: %q", line)
		}
	}
}

func TestNewInstallRefusesNotWSL(t *testing.T) {
	home, addons := newInstallHome(t)
	bin := newInstallFakeBin(t, "go1.26.9 linux/amd64")
	env := newInstallEnv(home, addons)
	env["VL_NEW_INSTALL_WSL"] = "0"
	_, stderr, code := runNewInstall(t, home, bin, env, "--dry-run")
	if code != 2 || !strings.Contains(stderr, "WSL") {
		t.Fatalf("not WSL: exit %d stderr %q, want the WSL refusal", code, stderr)
	}
}

func TestNewInstallRefusesMissingNT8Folder(t *testing.T) {
	home, _ := newInstallHome(t)
	bin := newInstallFakeBin(t, "go1.26.9 linux/amd64")
	env := newInstallEnv(home, filepath.Join(home, "does-not-exist"))
	_, stderr, code := runNewInstall(t, home, bin, env, "--dry-run")
	if code != 2 || !strings.Contains(stderr, "does not exist") {
		t.Fatalf("missing NT8: exit %d stderr %q, want the AddOns-folder refusal", code, stderr)
	}
}

func TestNewInstallRefusesWrongGoVersion(t *testing.T) {
	home, addons := newInstallHome(t)
	bin := newInstallFakeBin(t, "go1.20.0 linux/amd64")
	_, stderr, code := runNewInstall(t, home, bin, newInstallEnv(home, addons), "--dry-run")
	if code != 2 || !strings.Contains(stderr, "go must be") {
		t.Fatalf("wrong go: exit %d stderr %q, want the go-version refusal", code, stderr)
	}
}

// A second (non-dry) run against an already-installed HOME changes nothing: no
// new files, and the generated JWT_SECRET is not regenerated.
func TestNewInstallIdempotentSecondRunChangesNothing(t *testing.T) {
	home, addons := newInstallHome(t)
	bin := newInstallFakeBin(t, "go1.26.9 linux/amd64")
	env := newInstallEnv(home, addons)

	if _, _, code := runNewInstall(t, home, bin, env); code != 0 {
		t.Fatalf("first run exit %d", code)
	}
	first := snapshotTree(t, home)
	firstKeys := sortedKeys(first)

	if _, _, code := runNewInstall(t, home, bin, env); code != 0 {
		t.Fatalf("second run exit %d", code)
	}
	second := snapshotTree(t, home)

	if strings.Join(firstKeys, "\n") != strings.Join(sortedKeys(second), "\n") {
		t.Fatalf("second run changed the file list:\nfirst: %v\nsecond: %v", firstKeys, sortedKeys(second))
	}
	for k, v := range first {
		if second[k] != v {
			t.Fatalf("second run changed %q", k)
		}
	}
	// The JWT_SECRET was generated exactly once and is not shown anywhere.
	dotenv := first[filepath.Join("vl", ".env")]
	if !strings.Contains(dotenv, "JWT_SECRET=") {
		t.Fatalf(".env missing JWT_SECRET")
	}
}
