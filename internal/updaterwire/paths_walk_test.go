package updaterwire

import (
	"os"
	"path/filepath"
	"testing"

	"nofx/internal/censuswalk"
)

// PIN (M3 fold M5, red-team 4 #1): the worker-socket literal census walks a
// package in a directory the old walk skipped by NAME below the root. The
// offender is planted in a synthetic module (t.TempDir, never the real tree)
// and judged by the PRODUCTION census function.
func TestWorkerSocketLiteralCensusSeesNestedSkipNamedDirs(t *testing.T) {
	plant := func(t *testing.T, rel, pkg string) []string {
		t.Helper()
		root := t.TempDir()
		for r, body := range map[string]string{
			"go.mod":                        "module nofx\n\ngo 1.25\n",
			"internal/updaterwire/paths.go": "package updaterwire\n\nconst socketName = \"worker.sock\"\n",
			rel:                             "package " + pkg + "\n\nvar squat = \"updater/worker.sock\"\n",
		} {
			p := filepath.Join(root, filepath.FromSlash(r))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		off, _, err := workerSocketLiteralOffenders(root)
		if err != nil {
			t.Fatal(err)
		}
		return off
	}
	check := func(t *testing.T, rel string, off []string) {
		t.Helper()
		if len(off) != 1 || off[0] != rel+": names the worker socket (\"worker.sock\")" {
			t.Fatalf("%s names the worker socket; census offenders = %v", rel, off)
		}
	}
	check(t, "api/squat.go", plant(t, "api/squat.go", "api")) // positive control
	for _, dir := range censuswalk.NestedProbeDirs() {
		t.Run(dir, func(t *testing.T) {
			rel := dir + "/squat.go"
			check(t, rel, plant(t, rel, censuswalk.PackageName(dir)))
		})
	}
}
