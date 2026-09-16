package trader

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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
func TestNoCallSiteTreatsCancelReturnAsConfirmation(t *testing.T) {
	var offenders []string
	scanned := 0
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == "node_modules" || base == ".git" || base == "web" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
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
				offenders = append(offenders, path+":"+strconv.Itoa(i+1)+"  "+l)
			}
		}
		return nil
	})
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
