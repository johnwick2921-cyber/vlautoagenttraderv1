package deploy

// W-PR-B fold [28] — secret-scan's content pass used `find -size -2M`, and
// find's -size rounds UP to the unit: a file matches only when
// ceil(size/1MiB) < 2, i.e. ≤ 1 MiB. The README promised "skips files ≥ 2 MB",
// so everything 1–2 MiB — and the 2.6 MB JS bundle — was never content-scanned
// outside gitleaks. The bundle must be content-scanned: the bound is raised to
// 4 MB (files ≥ 4 MB are skipped, in practice the binary) and documented
// honestly.
//
// Class-250 probe: THIS TEST IS THE CALLER — it extracts the REAL find line
// from secret-scan.sh and runs that exact expression against fixture files.
// The mutation that proves it: reverting to `-size -2M` drops the 2.1 MB and
// 2.7 MB files from the matched set and fails the assertions below.

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSecretScanContentPassScansUpToTheDocumentedBound(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "deploy/release/secret-scan.sh"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)

	m := regexp.MustCompile(`find ("?\$ROOT"?|"[^"]*") -type f -size (-[0-9]+[kMG])`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("could not extract the content-pass find expression from secret-scan.sh")
	}
	findExpr := m[0]

	// fixture files around the real bound
	dir := t.TempDir()
	sizes := map[string]int{
		"a-1m.js":   1000000, // ≤ 1 MiB — always scanned
		"b-2.1m.js": 2100000, // 1–2 MiB — the old find silently skipped it
		"c-2.7m.js": 2700000, // the bundle class — MUST be content-scanned
		"d-5m.bin":  5000000, // ≥ 4 MB — the binary class, skipped by design
	}
	for name, n := range sizes {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, make([]byte, n), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("bash", "-c", strings.Replace(findExpr, `"$ROOT"`, `"`+dir+`"`, 1)+` | tr '\0' '\n'`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("find expression failed: %v\n%s", err, out)
	}
	matched := strings.TrimSpace(string(out))
	for _, name := range []string{"a-1m.js", "b-2.1m.js", "c-2.7m.js"} {
		if !strings.Contains(matched, name) {
			t.Fatalf("%s must be content-scanned; the find expression returned:\n%s", name, matched)
		}
	}
	if strings.Contains(matched, "d-5m.bin") {
		t.Fatalf("the ≥ 4 MB binary-class file must be skipped; find returned:\n%s", matched)
	}

	// the DOCUMENTED bound must agree with the code (never hand-maintained)
	readme, err := os.ReadFile(filepath.Join(root, "deploy/release/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "≥ 4 MB") {
		t.Fatalf("README must document the raised bound (≥ 4 MB) exactly as the code enforces it")
	}
}
