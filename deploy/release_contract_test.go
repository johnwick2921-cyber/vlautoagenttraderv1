package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// W-ONE-BUTTON WAVE 3a — the release workflow's guarantees, pinned.
//
// A release workflow is a thing nobody reads again until the day it matters,
// and by then the person reading it is under pressure. These tests state what
// it must do in words, so a change that quietly removes a guard fails here
// instead of shipping a package with a secret in it.

func repoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", rel))
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return string(b)
}

// The trigger is the whole safety story: a release is cut from a TAG that a
// human approved, never from a push to a branch.
func TestReleaseWorkflowTriggersOnTagAndNeverOnPushToDev(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(y, "tags:") || !strings.Contains(y, "v*") {
		t.Fatalf("release.yml must trigger on a v* TAG")
	}
	for _, forbidden := range []string{"branches:", "- dev", "pull_request:"} {
		if strings.Contains(y, forbidden) {
			t.Fatalf("release.yml must NEVER trigger on %q — a release is cut from an approved tag", forbidden)
		}
	}
	if !strings.Contains(y, "environment:") || !strings.Contains(y, "release") {
		t.Fatalf("release.yml must run in the protected 'release' Environment (owner = required reviewer)")
	}
}

// The binary must be provably built from clean, tagged source.
func TestReleaseWorkflowEnforcesCleanVcsStampAndTheGuideRev(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(y, "vcs.modified") {
		t.Fatalf("release.yml must enforce vcs.modified=false — a dirty build is not a release")
	}
	if !strings.Contains(y, "VITE_GUIDE_BUILT_REV") {
		t.Fatalf("release.yml must pass VITE_GUIDE_BUILT_REV so the guide cannot lie about the binary")
	}
}

// The artifact repository is an OPEN OWNER DECISION. The fail-closed default is
// THIS repo; the partner repo must never be reachable by accident.
func TestReleaseWorkflowDefaultsToThisRepoAndNeverThePartner(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(y, "RELEASE_REPO") {
		t.Fatalf("the artifact target must be ONE variable, RELEASE_REPO, so the owner changes it in one line")
	}
	if !strings.Contains(y, "johnwick2921-cyber/nofx") {
		t.Fatalf("RELEASE_REPO must DEFAULT to this repo")
	}
	if strings.Contains(y, "vlautoagenttraderv1") {
		t.Fatalf("the partner repo must never appear in the release workflow")
	}
}

// --- executable proofs: the scripts must REFUSE, not warn ---

func runScript(t *testing.T, script string, args ...string) (string, error) {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", script))
	if err != nil {
		t.Fatal(err)
	}
	// A MISSING script exits 127, which is also non-zero — so without this a
	// "it refused" assertion would be satisfied by "it was not there". That is
	// the shape that has bitten this wave twice.
	if st, serr := os.Stat(p); serr != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("%s must exist and be executable before its refusals mean anything (%v)", script, serr)
	}
	out, err := exec.Command("bash", append([]string{p}, args...)...).CombinedOutput()
	return string(out), err
}

// A package carrying a secret is rejected. This is the one that matters most:
// everything else costs a rebuild, this costs a credential.
func TestSecretScanRejectsASecretInTheStagedTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nofx-bin"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The deny-list entry that has actually bitten this repo before.
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DEEPSEEK_API_KEY=sk-live-should-never-ship\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runScript(t, "deploy/release/secret-scan.sh", dir)
	if err == nil {
		t.Fatalf("a staged tree containing .env MUST be rejected; scan exited 0.\n%s", out)
	}
	if !strings.Contains(out, ".env") {
		t.Fatalf("the refusal must NAME the offending path, got:\n%s", out)
	}
}

func TestSecretScanAcceptsACleanTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nofx-bin"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := runScript(t, "deploy/release/secret-scan.sh", dir); err != nil {
		t.Fatalf("a clean tree must pass, got error %v:\n%s", err, out)
	}
}

// A path that escapes the extraction root is rejected before it is written.
// The workflow must refuse with an INSTRUCTION when the owner has not yet
// created the release keypair, rather than a file-not-found from ssh-keygen.
func TestReleaseWorkflowRefusesWithInstructionsWhenThePublicKeyIsMissing(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(y, "deploy/release.pub") {
		t.Fatalf("the workflow must verify against the COMMITTED public key")
	}
	if !strings.Contains(y, "RELEASE_SIGNING_KEY") {
		t.Fatalf("the private half must come from the release Environment secret")
	}
	if !strings.Contains(y, "deploy/release/README.md") {
		t.Fatalf("the refusal must point at the one-time owner instructions")
	}
}

// A rollback pair is advertised tested ONLY when the compat job proved it.
func TestReleaseWorkflowNeverFabricatesARollbackPair(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(y, "ROLLBACK_PAIRS: '[]'") {
		t.Fatalf("until the DB-compat job supplies proven pairs, ROLLBACK_PAIRS must be EMPTY, never a plausible default")
	}
	if strings.Contains(y, "needs.dbcompat") {
		t.Fatalf("the workflow must not reference a job that does not exist — it parses, then fails at run time")
	}
}

func TestPackagerRejectsAnArchivePathThatEscapesTheRoot(t *testing.T) {
	out, err := runScript(t, "deploy/release/check-archive-paths.sh", "../../etc/passwd")
	if err == nil {
		t.Fatalf("a path escaping the root MUST be rejected; exited 0.\n%s", out)
	}
	if out2, err2 := runScript(t, "deploy/release/check-archive-paths.sh", "web/dist/index.html"); err2 != nil {
		t.Fatalf("an ordinary relative path must be accepted, got %v:\n%s", err2, out2)
	}
}
