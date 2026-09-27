package updateauth

import (
	"os"
	"os/exec"
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
// REST is not a valid go.mod — `go list` refuses it.
//
// The mechanism is a re-exec child: the child test runs the REAL census over
// the broken module and must die inside constTypeInfo's t.Fatalf (the loud
// failure), so the child FAILS. The parent passes IFF the child failed — on
// the old silent-degradation code the child passes and this test FAILS.
func TestUpdateAuthCensusRefusesAModuleWhoseGoListCannotRun(t *testing.T) {
	if root := os.Getenv("CENSUS_FAILLOUD_ROOT"); root != "" {
		// CHILD: the census over this broken module must NOT return.
		_, _, err := updateAuthOffenders(t, root)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatal("the census returned over a module whose go list cannot run — constTypeInfo degraded to the name fold silently")
		return
	}
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
	cmd := exec.Command(os.Args[0], "-test.run=^TestUpdateAuthCensusRefusesAModuleWhoseGoListCannotRun$", "-test.count=1")
	cmd.Env = append(os.Environ(), "CENSUS_FAILLOUD_ROOT="+tmp)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("the census PASSED a module whose go list cannot run — constTypeInfo degraded to the name fold silently\n%s", out)
	}
}

