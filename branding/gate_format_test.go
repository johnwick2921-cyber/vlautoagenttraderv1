package branding

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCryptoUnionGateCanonicalFormat drives scripts/crypto-union-gate.sh in a
// synthetic repo and pins the CTO-ruled canonical table format:
//
//	| path | line | token | DELETE|CUT|KEEP | OWNER | reason |
//
// plus the single-ownership invariant: a line claimed by TWO tables is a
// DOUBLE-CLAIM FAIL, a prose/range line is an unparseable FAIL, and a clean
// canonical union exits 0. (The production call site is the script itself.)
func TestCryptoUnionGateCanonicalFormat(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmp := t.TempDir()
	tblDir := t.TempDir() // tables live OUTSIDE the swept repo

	// minimal guard: the script extracts the literal and the allowlist from here
	write(t, filepath.Join(tmp, "branding/no_crypto.go"),
		"package branding\n\nconst SweepRegexLiteral = `bybit`\n\n"+
			"var LineLevelOwnershipPaths = []string{\n\t\"agent/agent.go\",\n\t\"agent/tools.go\",\n}\n\n"+
			"var RiskCapAssertSites = []string{\n\t\"src/kept.go\", \"package\",\n}\n")

	// the real gate script
	scriptSrc, err := os.ReadFile(filepath.Join("..", "scripts", "crypto-union-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(tmp, "scripts", "crypto-union-gate.sh")
	write(t, script, string(scriptSrc))
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}

	write(t, filepath.Join(tmp, "src/kept.go"), "package src\nvar K = \"bybit api\" // line 2, covered by a KEEP row\n")
	write(t, filepath.Join(tmp, "web/src/components/plan/ExecutorVerdict.tsx"), "arm.state === 'mixed'\n")
	write(t, filepath.Join(tmp, "web/src/components/trader/TraderConfigModal.tsx"), "// cut complete\n")
	// meet the 2,000-file enumeration floor
	for i := 0; i < 2000; i++ {
		write(t, filepath.Join(tmp, "fill", fmt.Sprintf("f%05d.txt", i)), "")
	}

	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		// CI runners have no git identity; pin author + committer for every
		// git call this synthetic repo makes (commits must not fail with
		// "empty ident name").
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=gate-test",
			"GIT_AUTHOR_EMAIL=gate-test@example.invalid",
			"GIT_COMMITTER_NAME=gate-test",
			"GIT_COMMITTER_EMAIL=gate-test@example.invalid",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "init")
	head := strings.TrimSpace(git("rev-parse", "HEAD"))

	const guardLine = 3 // branding/no_crypto.go line holding the literal
	const keptLine = 2  // src/kept.go line holding "bybit api"

	tableA := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| branding/no_crypto.go | %d | bybit | KEEP | CR-B | guard's own exported literal |
| src/kept.go | %d | bybit | KEEP | CR-A | deliberate keep for the proof |
`, head, head, guardLine, keptLine)
	tableAPath := filepath.Join(tblDir, "tblA.md")
	write(t, tableAPath, tableA)

	runGate := func(args ...string) (string, int) {
		cmd := exec.Command("bash", append([]string{script}, args...)...)
		cmd.Dir = tmp
		out, err := cmd.CombinedOutput()
		rc := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				rc = ee.ExitCode()
			} else {
				t.Fatalf("gate run: %v\n%s", err, out)
			}
		}
		return string(out), rc
	}

	// clean canonical union -> exit 0
	out, rc := runGate(tableAPath)
	if rc != 0 {
		t.Fatalf("clean canonical union must exit 0, got %d:\n%s", rc, out)
	}
	for _, want := range []string{"sweep complete", "DOUBLE-CLAIM"} {
		if want == "DOUBLE-CLAIM" && strings.Contains(out, want) {
			t.Fatalf("clean union must not report DOUBLE-CLAIM:\n%s", out)
		}
	}

	// double-claim: a second table claims the SAME kept.go line as its own
	tableB := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| src/kept.go | %d | bybit | KEEP | CR-B | conflicting claim |
`, head, head, keptLine)
	tableBPath := filepath.Join(tblDir, "tblB.md")
	write(t, tableBPath, tableB)
	out, rc = runGate(tableAPath, tableBPath)
	if rc == 0 {
		t.Fatalf("double-claim union must FAIL, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "DOUBLE-CLAIM") {
		t.Fatalf("expected DOUBLE-CLAIM in output:\n%s", out)
	}

	// non-canonical line: prose ranges must be refused, not silently parsed
	tableC := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
CUT src/kept.go [2] -- a prose range, not a canonical row
`, head, head)
	tableCPath := filepath.Join(tblDir, "tblC.md")
	write(t, tableCPath, tableC)
	out, rc = runGate(tableCPath)
	if rc == 0 {
		t.Fatalf("prose-range table must FAIL, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "unparseable") {
		t.Fatalf("expected 'unparseable' in output:\n%s", out)
	}

	// file-level exclusivity (Finding 3): the same PATH claimed by two tables
	// on DIFFERENT lines — per-line check passes, the file-level check must FAIL
	tableD := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| src/kept.go | %d | bybit | KEEP | CR-B | same path, different line, different owner |
`, head, head, keptLine+1)
	tableDPath := filepath.Join(tblDir, "tblD.md")
	write(t, tableDPath, tableD)
	out, rc = runGate(tableAPath, tableDPath)
	if rc == 0 {
		t.Fatalf("file-level double-claim must FAIL, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "FILE DOUBLE-CLAIM") || !strings.Contains(out, "src/kept.go") {
		t.Fatalf("expected FILE DOUBLE-CLAIM on src/kept.go:\n%s", out)
	}

	// blanket row + dialect (Finding 4 + CR-C): a dated-export file with a hit
	// is excluded by one blanket row with count=N — the count is VERIFIED; the
	// CR-C dialect (bullet headers, backticked fields, extraction-note regex)
	// parses too
	write(t, filepath.Join(tmp, "src/gen/report.tsv"), "bybit\thit\n")
	git("add", "-A")
	git("commit", "-qm", "gen-export")
	head = strings.TrimSpace(git("rev-parse", "HEAD"))
	tableE := fmt.Sprintf(`- branch point: %s (origin/dev tip, cut at accept)
- integrator tip at generation: %s
- paths: .
- regex: extracted programmatically from plan v10 line 108 (len 240)
- line rows: 2 · blanket-KEEP paths: 1
| path | line | token | disposition | OWNER | reason |
|---|---|---|---|---|---|
| `+"`branding/no_crypto.go`"+` | %d | `+"`bybit`"+` | KEEP | CR-B | guard's own exported literal |
| `+"`src/kept.go`"+` | %d | `+"`bybit`"+` | KEEP | CR-A | deliberate keep for the proof |
| `+"`src/gen/report.tsv`"+` | - | count=1 | KEEP | CR-C | dated research export — historical record, not shipped code, KEEP byte-identical |
`, head, head, guardLine, keptLine)
	tableEPath := filepath.Join(tblDir, "tblE.md")
	write(t, tableEPath, tableE)
	out, rc = runGate(tableEPath)
	if rc != 0 {
		t.Fatalf("dialect union with a count-verified blanket row must exit 0, got %d:\n%s", rc, out)
	}
	if !strings.Contains(out, "excluded with count=1") {
		t.Fatalf("expected the blanket exclusion to be REPORTED:\n%s", out)
	}

	// a blanket count that disagrees with the sweep is a silent-skip-proof FAIL
	tableF := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| src/gen/report.tsv | - | count=7 | KEEP | CR-C | dated research export |
| src/nope.tsv | - | count=1 | KEEP | CR-C | typo row, matches nothing |
`, head, head)
	tableFPath := filepath.Join(tblDir, "tblF.md")
	write(t, tableFPath, tableF)
	out, rc = runGate(tableFPath)
	if rc == 0 {
		t.Fatalf("count mismatch + stale blanket must FAIL, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "count mismatch") {
		t.Fatalf("expected the count-mismatch FAIL:\n%s", out)
	}
	if !strings.Contains(out, "matches NO tracked file") {
		t.Fatalf("expected the stale-blanket FAIL:\n%s", out)
	}

	// line-level ownership (HOLD-LIFTED ruling): on the three allowlisted paths
	// a same-line double-claim still FAILS (line-level), while two tables owning
	// DIFFERENT lines of an allowlisted path is legal (no FILE DOUBLE-CLAIM).
	write(t, filepath.Join(tmp, "agent/tools.go"), "package agent\nvar A = \"bybit agent\"\nvar B = \"bybit helper\"\n")
	git("add", "-A")
	git("commit", "-qm", "agent-tools")
	head = strings.TrimSpace(git("rev-parse", "HEAD"))
	tableG := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| branding/no_crypto.go | %d | bybit | KEEP | CR-B | guard literal |
| src/kept.go | %d | bybit | KEEP | CR-A | keep |
| src/gen/report.tsv | - | count=1 | KEEP | CR-C | dated research export |
| agent/tools.go | 2 | bybit | KEEP | CR-A | payment-branch line |
| agent/tools.go | 3 | bybit | KEEP | CR-B | other line, CR-B owns |
`, head, head, guardLine, keptLine)
	tableGPath := filepath.Join(tblDir, "tblG.md")
	write(t, tableGPath, tableG)
	out, rc = runGate(tableGPath)
	if rc != 0 {
		t.Fatalf("allowlisted path, different lines, two owners must PASS (line-level ruling), got %d:\n%s", rc, out)
	}
	if strings.Contains(out, "FILE DOUBLE-CLAIM") {
		t.Fatalf("allowlisted path must not trip the file-level check:\n%s", out)
	}

	// same LINE, two owners, allowlisted path -> line-level DOUBLE-CLAIM
	tableH := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| agent/tools.go | 2 | bybit | KEEP | CR-B | conflicting claim on the same line |
`, head, head)
	tableHPath := filepath.Join(tblDir, "tblH.md")
	write(t, tableHPath, tableH)
	out, rc = runGate(tableGPath, tableHPath)
	if rc == 0 {
		t.Fatalf("same-line double-claim on an allowlisted path must FAIL, got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "DOUBLE-CLAIM") {
		t.Fatalf("expected the line-level DOUBLE-CLAIM:\n%s", out)
	}

	// CR-A header dialect (the third accepted form) + ceded '-' markers:
	// base sha (branch point): / integrator tip generated against: / a sweep-regex
	// header with the literal on the NEXT line / NO paths line (NOTE, not FAIL).
	// A ceded row carries only ownership; the real disposition must come from the
	// owner's table. Ceded-only = FAIL; paired with the real row = PASS.
	// Abbreviated shas accepted (CR-A carries 9-hex).
	write(t, filepath.Join(tmp, "web/src/components/plan/ExecutorVerdict.tsx"),
		"const ok = arm.state === 'mixed'\n")
	git("add", "-A")
	git("commit", "-qm", "trap-file")
	head = strings.TrimSpace(git("rev-parse", "HEAD"))
	tableI := fmt.Sprintf(`base sha (branch point): %s
integrator tip generated against: %s
sweep regex (exported from plan v10 line 108, never hand-typed):
  `+"`bybit`"+`
ownership: file-level per Finding 3
provider/alpaca and its live branches are out of scope (declined) — accepted prose, no pipes
  `+"`bybit|oldtoken`"+` — a pre-amendment embedded literal blob, skipped as a backticked line
| path | line | token | disposition | owner | reason |
|---|---|---|---|---|---|
| agent/tools.go | 2 | bybit | - | CR-B | ceded to CR-B (union coverage) |
`, head[:9], head[:9])
	tableIPath := filepath.Join(tblDir, "tblI.md")
	write(t, tableIPath, tableI)
	out, rc = runGate(tableIPath)
	if rc == 0 {
		t.Fatalf("ceded-only line must FAIL (no table carries a real disposition), got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "ONLY ceded markers") {
		t.Fatalf("expected the ONLY-ceded-markers FAIL:\n%s", out)
	}

	// the owner's table carries the real row for the same line -> union PASS,
	// no paths header -> NOTE not FAIL, literal taken from the next line
	tableJ := fmt.Sprintf(`base sha (branch point): %s
integrator tip generated against: %s
sweep regex (exported from plan v10 line 108, never hand-typed):
  `+"`bybit`"+`
ownership: file-level per Finding 3
| path | line | token | disposition | owner | reason |
|---|---|---|---|---|---|
| branding/no_crypto.go | %d | bybit | KEEP | CR-B | guard literal |
| src/kept.go | %d | bybit | KEEP | CR-A | keep |
| src/gen/report.tsv | - | count=1 | KEEP | CR-C | dated research export |
| agent/tools.go | 2 | bybit | KEEP | CR-B | real row from the owning part |
| agent/tools.go | 3 | bybit | KEEP | CR-A | other line |
| web/src/components/plan/ExecutorVerdict.tsx | 1 | `+"`'mixed' (single-quoted)`"+` | KEEP | CR-C | named trap (not regex-visible) |
`, head[:9], head[:9], guardLine, keptLine)
	tableJPath := filepath.Join(tblDir, "tblJ.md")
	write(t, tableJPath, tableJ)
	out, rc = runGate(tableIPath, tableJPath)
	if rc != 0 {
		t.Fatalf("CR-A dialect union with a satisfied cession must exit 0, got %d:\n%s", rc, out)
	}
	if !strings.Contains(out, "no paths header") {
		t.Fatalf("expected the no-paths-header NOTE:\n%s", out)
	}
	if strings.Contains(out, "ONLY ceded markers") {
		t.Fatalf("satisfied cession must not report ONLY-ceded-markers:\n%s", out)
	}

	// a multi-MB single-line file must not crash the sweep — the pathological-
	// line grep class that took DS-101's run down (2026-10-01). The gate must
	// COMPLETE and report the hit, never die.
	huge := strings.Repeat("x", 2<<20) + "bybit huge-line marker"
	write(t, filepath.Join(tmp, "src/huge.json"), huge+"\n")
	git("add", "-A")
	git("commit", "-qm", "huge-line")
	head = strings.TrimSpace(git("rev-parse", "HEAD"))
	tableK := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: .
regex: bybit
| branding/no_crypto.go | %d | bybit | KEEP | CR-B | guard literal |
| src/kept.go | %d | bybit | KEEP | CR-A | keep |
| src/gen/report.tsv | - | count=1 | KEEP | CR-C | dated research export |
| agent/tools.go | 2 | bybit | KEEP | CR-B | real row |
| agent/tools.go | 3 | bybit | KEEP | CR-A | other line |
`, head, head, guardLine, keptLine)
	tableKPath := filepath.Join(tblDir, "tblK.md")
	write(t, tableKPath, tableK)
	out, rc = runGate(tableKPath)
	if rc == 0 {
		t.Fatalf("the huge-line file's hit has no row and must FAIL (UNLISTED), got exit 0:\n%s", out)
	}
	if !strings.Contains(out, "sweep complete") {
		t.Fatalf("the gate must COMPLETE on a multi-MB line, not crash:\n%s", out)
	}
	if !strings.Contains(out, "src/huge.json") {
		t.Fatalf("expected the huge-line hit to be reported:\n%s", out)
	}

	// the gate's table inputs are never swept — a table living INSIDE the
	// repo, whose rows carry tokens, must not self-hit
	tableL := fmt.Sprintf(`branch-point: %s
integrator-tip: %s
paths: src
regex: bybit
| branding/no_crypto.go | %d | bybit | KEEP | CR-B | guard literal |
| src/kept.go | %d | bybit | KEEP | CR-A | keep |
| src/gen/report.tsv | - | count=1 | KEEP | CR-C | dated research export |
| src/huge.json | 1 | bybit | KEEP | CR-A | huge-line fixture |
| agent/tools.go | 2 | bybit | KEEP | CR-B | real row |
| agent/tools.go | 3 | bybit | KEEP | CR-A | other line |
`, head, head, guardLine, keptLine)
	write(t, filepath.Join(tmp, "docs/crypto-removal/tblL.md"), tableL)
	git("add", "-A")
	git("commit", "-qm", "in-repo-table")
	head = strings.TrimSpace(git("rev-parse", "HEAD"))
	out, rc = runGate("docs/crypto-removal/tblL.md")
	if rc != 0 {
		t.Fatalf("an in-repo table with only covered rows must exit 0, got %d:\n%s", rc, out)
	}
	for _, bad := range []string{
		"UNLISTED hit: docs/crypto-removal/tblL.md",
		"hit not covered by a KEEP row", // only tblL.md rows exist in this run
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("the table input was swept (self-hit):\n%s", out)
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
