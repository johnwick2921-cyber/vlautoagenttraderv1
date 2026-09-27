package deploy

// W-PR-B fold [30] — under `set -e`, if `ssh-keygen -Y sign` fails, the step
// exits BEFORE its `shred -u /tmp/sign.key` line, so the release signing key
// file survives on the runner for every later step. The key must be shredded
// on EVERY exit path (a trap), not just the success line.
//
// Class-250 probe: THIS TEST IS THE CALLER — it extracts the REAL sign-step
// run body from release.yml and executes it. The mutation that proves it:
// removing the `trap ... EXIT` line makes the failure-path assertion fail
// (the key file survives a failed sign).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowShredsTheSigningKeyOnEveryPath(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	body := stepRunBody(t, y, "Sign the manifest (ed25519; private key lives only in the Environment)")
	if !strings.Contains(body, "trap") || !strings.Contains(body, "EXIT") {
		t.Fatalf("the sign step must shred the key on EVERY exit path (trap ... EXIT); body:\n%s", body)
	}
	root := repoRoot(t)
	os.Remove("/tmp/sign.key")

	run := func(key string) (string, error) {
		t.Helper()
		cmd := exec.Command("bash", "-c", body)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "RELEASE_SIGNING_KEY="+key)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// (a) FAILURE path: a bogus key makes ssh-keygen -Y sign fail under set -e.
	//     The key file must NOT survive.
	out, err := run("not-a-private-key")
	if err == nil {
		t.Fatalf("expected the sign step to fail on a bogus key:\n%s", out)
	}
	if _, serr := os.Stat("/tmp/sign.key"); serr == nil {
		b, _ := os.ReadFile("/tmp/sign.key")
		t.Fatalf("sign FAILED but /tmp/sign.key survived on the runner: %q", string(b))
	}

	// (b) SUCCESS path: a real ed25519 key signs a staged manifest; the step
	//     then proceeds (and may refuse on the allowed-signers check) — the key
	//     must be gone in every case.
	if err := os.MkdirAll("/tmp/stage", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/stage/manifest.json", []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "k")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-q").CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen ed25519: %v\n%s", err, out)
	}
	key, rerr := os.ReadFile(keyPath)
	if rerr != nil {
		t.Fatal(rerr)
	}
	out2, _ := run(string(key))
	if _, serr := os.Stat("/tmp/sign.key"); serr == nil {
		t.Fatalf("after the sign step ran, /tmp/sign.key still exists:\n%s", out2)
	}
	os.Remove("/tmp/stage/manifest.json")
}
