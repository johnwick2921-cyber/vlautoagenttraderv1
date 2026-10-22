package deploy

// partner_first_time_setup_test — the test suite for
// deploy/partner-first-time-setup.sh (release #32, partner one-time setup).
//
// The suite runs the REAL script against a scratch HOME with PATH-injected
// fakes for systemctl and loginctl (the script's only systemd calls), a pinned
// VL_UPDATER_PARTNER_SHA (no network), and the real partner trust anchor. It
// proves --dry-run prints every action and writes NOTHING, and that the
// refusals (wrong signer key, missing install dir, bad email) fire before any
// write. No secret is ever asserted from the output — the token is never
// printed, and the test checks the script's output for the token only to prove
// absence.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// partnerFirstTimeSetupScript is the script under test, relative to this file.
func partnerFirstTimeSetupScript(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "partner-first-time-setup.sh")
}

// fakeBin writes PATH fakes for systemctl and loginctl. go/git/curl come from
// the real PATH (the script only checks their presence; git ls-remote is
// skipped by VL_UPDATER_PARTNER_SHA, and curl is only used in the live check).
func fakeBin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	systemctl := filepath.Join(dir, "systemctl")
	loginctl := filepath.Join(dir, "loginctl")
	// is-system-running succeeds; start also succeeds (never reached in dry-run).
	if err := os.WriteFile(systemctl, []byte("#!/usr/bin/env bash\ncase \"$1 $2\" in\n  \"--user is-system-running\") exit 0 ;;\n  \"--user start\") exit 0 ;;\n  \"--user enable\") exit 0 ;;\n  *) exit 1 ;;\nesac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loginctl, []byte("#!/usr/bin/env bash\ncase \"$1\" in\n  show-user) echo yes ;;\n  *) exit 1 ;;\nesac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// partnerInstallTree builds a fake install dir with a .env and the partner
// trust anchor (the key read from the repo's own release_allowed_signers).
func partnerInstallTree(t *testing.T, home, signerLine string) string {
	t.Helper()
	install := filepath.Join(home, "vl")
	if err := os.MkdirAll(filepath.Join(install, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, ".env"), []byte("JWT_SECRET=test-secret-0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "deploy", "release_allowed_signers"), []byte(signerLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return install
}

// snapshotFiles returns the sorted relative paths of every file under root.
func snapshotFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func runSetup(t *testing.T, home, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{partnerFirstTimeSetupScript(t)}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PATH="+bin+":"+os.Getenv("PATH"),
		"VL_UPDATER_PARTNER_SHA=38296df95249e95aa62d48273ffcd1d28843d19c",
	)
	var so, se strings.Builder
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run: %v", err)
	}
	return so.String(), se.String(), code
}

func partnerSignerLine(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("release_allowed_signers")
	if err != nil {
		t.Fatalf("read deploy/release_allowed_signers: %v", err)
	}
	return strings.TrimSpace(string(b))
}

// --dry-run prints every action and writes NOTHING, and the token never appears.
func TestPartnerFirstTimeSetupDryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := partnerInstallTree(t, home, partnerSignerLine(t))
	before := snapshotFiles(t, home)

	stdout, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", install, "--release-dir", filepath.Join(home, "releases"), "owner@example.com")
	if code != 0 {
		t.Fatalf("dry-run exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, want := range []string{"would run: go run", "would run:", "nothing was written", "would run: systemctl --user start vl-updater",
		"VL_RELEASE_SOURCE=github", "would: mkdir -p"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, stdout)
		}
	}
	after := snapshotFiles(t, home)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("dry-run wrote files:\nbefore: %v\nafter: %v", before, after)
	}
	if strings.Contains(stdout+stderr, "VL_CUTOVER_TOKEN=") || strings.Contains(stdout+stderr, "eyJ") {
		t.Fatalf("a token leaked into the output:\n%s\n%s", stdout, stderr)
	}
}

// The wrong trust anchor (the OWNER repo's key) refuses before anything runs.
func TestPartnerFirstTimeSetupRefusesOwnerKey(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := partnerInstallTree(t, home, "release ssh-ed25519 "+vlReleaseKey+" owner-release")
	_, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", install, "--release-dir", filepath.Join(home, "releases"), "owner@example.com")
	if code != 2 || !strings.Contains(stderr, "OWNER repo's key") {
		t.Fatalf("owner key: exit %d stderr %q, want refusal naming the owner key", code, stderr)
	}
}

// A missing install dir refuses before anything runs.
func TestPartnerFirstTimeSetupRefusesMissingInstallDir(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	_, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", filepath.Join(home, "nonexistent"), "--release-dir", filepath.Join(home, "releases"), "owner@example.com")
	if code != 2 || !strings.Contains(stderr, "install dir does not exist") {
		t.Fatalf("missing dir: exit %d stderr %q, want the install-dir refusal", code, stderr)
	}
}

// A malformed email refuses before anything runs.
func TestPartnerFirstTimeSetupRefusesBadEmail(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := partnerInstallTree(t, home, partnerSignerLine(t))
	_, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", install, "--release-dir", filepath.Join(home, "releases"), "not-an-email")
	if code != 2 || !strings.Contains(stderr, "must look like an address") {
		t.Fatalf("bad email: exit %d stderr %q, want the email refusal", code, stderr)
	}
}

// A release dir INSIDE the install refuses (the containment rule).
func TestPartnerFirstTimeSetupRefusesReleaseDirInsideInstall(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := partnerInstallTree(t, home, partnerSignerLine(t))
	_, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", install, "--release-dir", filepath.Join(install, "releases"), "owner@example.com")
	if code != 2 || !strings.Contains(stderr, "must be OUTSIDE the install") {
		t.Fatalf("inside install: exit %d stderr %q, want the containment refusal", code, stderr)
	}
}

// The installed systemd unit hardcodes %h/vl, so a non-default install dir
// refuses (exit 2) with the exact reason the fold ordered.
func TestPartnerFirstTimeSetupRefusesNonDefaultInstallDir(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := filepath.Join(home, "elsewhere")
	_ = partnerInstallTree(t, home, partnerSignerLine(t)) // the $HOME/vl tree (unused)
	// A DIFFERENT, valid install dir (exists + has .env + signers).
	if err := os.MkdirAll(filepath.Join(install, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, ".env"), []byte("JWT_SECRET=test-secret-0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "deploy", "release_allowed_signers"), []byte(partnerSignerLine(t)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runSetup(t, home, bin,
		"--dry-run", "--install-dir", install, "--release-dir", filepath.Join(home, "releases"), "owner@example.com")
	if code != 2 || !strings.Contains(stderr, "a different install dir is not supported") {
		t.Fatalf("non-default install: exit %d stderr %q, want the unit-path refusal", code, stderr)
	}
}

// An existing ~/.config/vl-updater/env that is not mode 0600 refuses (the same
// condition install-updater-worker.sh enforces) — AND, because the release-dir
// mkdir (m385) runs BEFORE that refusal, the release dir is created first.
// MUTANT: drop the mkdir from step 2 → the release dir is missing → RED.
func TestPartnerFirstTimeSetupRefusesEnvNot0600(t *testing.T) {
	home := t.TempDir()
	bin := fakeBin(t)
	install := partnerInstallTree(t, home, partnerSignerLine(t)) // $HOME/vl
	// Already enrolled: admin.json present so enroll is skipped, and an env
	// file with a token line but the WRONG mode (644) so the env-file step
	// refuses before anything is written.
	if err := os.MkdirAll(filepath.Join(install, "data", "updater"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "data", "updater", "admin.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(home, ".config", "vl-updater", "env")
	if err := os.MkdirAll(filepath.Dir(envFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte("VL_CUTOVER_TOKEN=planted\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runSetup(t, home, bin,
		"--install-dir", install, "--release-dir", filepath.Join(home, "releases"), "owner@example.com")
	if code != 2 || !strings.Contains(stderr, "not mode 0600") {
		t.Fatalf("env 0600: exit %d stderr %q, want the mode refusal", code, stderr)
	}
	// m385: the release-dir mkdir (step 2) runs BEFORE the env-0600 refusal
	// (step 4), so the dir must exist even though the run was refused.
	rel, err := os.Stat(filepath.Join(home, "releases"))
	if err != nil || !rel.IsDir() {
		t.Fatalf("the release dir must be created before the env-0600 refusal: %v", err)
	}
}
