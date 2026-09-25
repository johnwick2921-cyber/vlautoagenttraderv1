package updateauth

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateAuthCensusRefusesAModuleWhoseGoListCannotRun is the DS-105
// CENSUS-AUTH [13] fold RED (CTO 1790310827826): constTypeInfo used to return
// an EMPTY map when go list failed, silently turning the typed constant pass
// OFF — the census then passed blind on the per-file name fold. The census
// must FAIL, not pass, over a module the toolchain cannot read.
//
// The fixture: a go.mod whose module line is well-formed (censuswalk's
// ModulePath reads it leniently, and the walk itself succeeds) but whose
// REST is not a valid go.mod — `go list` refuses it. Old code: constTypeInfo
// degrades silently, the planted package is scanned with the name fold and
// the census returns no offenders and no error — the subtest passes and this
// test FAILS. New code: constTypeInfo t.Fatalf's inside the subtest, so the
// subtest FAILS and this test passes.
func TestUpdateAuthCensusRefusesAModuleWhoseGoListCannotRun(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module censusbroken\n\nnot a go.mod directive\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(tmp, "p")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	// A plain, non-offending package: on the degraded name fold the census
	// has nothing to report, so old code returns (nil offenders, nil error).
	if err := os.WriteFile(filepath.Join(pkg, "x.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := t.Run("census over the broken module", func(st *testing.T) {
		// Must not pass silently: on old code this returns no error and the
		// subtest PASSES; on new code constTypeInfo st.Fatalf's and the
		// subtest FAILS — which is the refusal this test demands.
		_, _, _ = updateAuthOffenders(tmp)
	})
	if sub {
		t.Fatalf("the census passed a module whose go list cannot run — constTypeInfo degraded to the name fold silently")
	}
}
