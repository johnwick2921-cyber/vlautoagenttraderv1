package trader

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// E7 / D5 — NO CALL SITE MAY TREAT THE CANCEL RETURN AS CONFIRMATION.
//
// nt.CancelOrder returns the result of putting a frame on a socket. Five sites
// in armed_executor.go used to read `cerr == nil` as proof the order was gone
// and wrote the ledger terminal on it. This test is what stops the sixth.
//
// It is a SOURCE scan on purpose: the defect is a shape, not a value, and no
// runtime assertion can see a branch that was never taken.
//
// THE RULE IS DELIBERATELY BROADER THAN THE DEFECT: no block may OPEN on a
// cancel send returning nil, even one that merely logs. Branching on the
// success of a send is where the defect grows, and "does this block also write
// a terminal state?" is not a question a text scan can answer honestly. Write
// the failure branch instead — `if err != nil { … } else { … }` — which reads
// the same and cannot drift into treating a send as a settlement.
//
// The walk is internal/censuswalk.NonTestGoFiles (CLASS 258): skip names apply
// ONLY as direct children of the module root.
func TestNoCallSiteTreatsCancelReturnAsConfirmation(t *testing.T) {
	scanned, offenders, err := scanCancelSites("..")
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Fatal("the scan read no files — this pin cannot fail and is therefore not a pin")
	}
	if len(offenders) > 0 {
		t.Fatalf("a cancel SEND is being read as a CONFIRMATION at %d site(s) — move the row to cancel_pending and let the settlement pass confirm it:\n\t%s",
			len(offenders), strings.Join(offenders, "\n\t"))
	}
	t.Logf("scanned %d production .go files; no call site treats the cancel return as confirmation", scanned)
}

// scanCancelSites is the shared scan: every non-test .go file under root
// (censuswalk) is checked for the defect shape.
func scanCancelSites(root string) (scanned int, offenders []string, err error) {
	files, err := censuswalk.NonTestGoFiles(root)
	if err != nil {
		return 0, nil, err
	}
	for _, f := range files {
		b, rerr := os.ReadFile(f.Path)
		if rerr != nil {
			continue
		}
		scanned++
		for i, line := range strings.Split(string(b), "\n") {
			l := strings.TrimSpace(line)
			if !strings.Contains(l, "CancelOrder(") {
				continue
			}
			// THE DEFECT SHAPE: the send's error is compared to nil in the same
			// statement that opens a block — i.e. the block runs BECAUSE the
			// send returned nil, which is what "treating a send as a
			// settlement" looks like in Go.
			if strings.Contains(l, "== nil") && strings.HasSuffix(l, "{") {
				offenders = append(offenders, f.Rel+":"+strconv.Itoa(i+1)+"  "+l)
			}
		}
	}
	return scanned, offenders, nil
}

// TestCancelCensusSeesNestedSkipNamedDirs plants the defect in EVERY
// censuswalk.NestedProbeDirs directory of a synthetic module and asserts the
// census sees every one. With the old any-depth SkipDir the dirs named like a
// root skip (api/web, internal/node_modules/p, api/.git, …) were invisible
// (CLASS 258).
func TestCancelCensusSeesNestedSkipNamedDirs(t *testing.T) {
	root := t.TempDir()
	dirs := censuswalk.NestedProbeDirs()
	for _, dir := range dirs {
		full := filepath.Join(root, filepath.FromSlash(dir))
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + censuswalk.PackageName(dir) + "\n\n" +
			"func offender() {\n\tif nt.CancelOrder(\"MNQ\") == nil {\n\t\t_ = 1\n\t}\n}\n"
		if err := os.WriteFile(filepath.Join(full, "offender.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, offenders, err := scanCancelSites(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, o := range offenders {
		seen[filepath.ToSlash(filepath.Dir(strings.SplitN(o, ":", 2)[0]))] = true
	}
	var missed []string
	for _, dir := range dirs {
		if !seen[dir] {
			missed = append(missed, dir)
		}
	}
	if len(missed) > 0 {
		t.Fatalf("the cancel census skipped %d of %d nested probe dirs — a skip by NAME at depth exempts compiled packages (CLASS 258):\n\t%s",
			len(missed), len(dirs), strings.Join(missed, "\n\t"))
	}
}
