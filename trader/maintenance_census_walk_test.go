package trader

import (
	"os"
	"path/filepath"
	"testing"

	"nofx/internal/censuswalk"
)

// PIN (M3 fold M5, red-team 4 #1): the maintenance-setter census walks a
// package in a directory the old walk skipped by NAME below the root
// (trader/web, internal/node_modules/p, …). A second production caller of
// SetEntryPermit there could unwire the entry permit with the census green.
// Planted in a synthetic module (t.TempDir, never the real tree), judged by
// the PRODUCTION census function.
func TestMaintenanceSetterCensusSeesNestedSkipNamedDirs(t *testing.T) {
	plant := func(t *testing.T, rel, pkg string) []string {
		t.Helper()
		root := t.TempDir()
		for r, body := range map[string]string{
			"go.mod":                       "module nofx\n\ngo 1.25\n",
			"trader/maintenance_wiring.go": "package trader\n",
			rel: "package " + pkg + "\n\ntype permitter interface{ SetEntryPermit(any) }\n\n" +
				"func Unwire(nt permitter) { nt.SetEntryPermit(nil) }\n",
		} {
			p := filepath.Join(root, filepath.FromSlash(r))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		off, _, err := maintenanceSetterOffenders(root)
		if err != nil {
			t.Fatal(err)
		}
		return off
	}
	check := func(t *testing.T, rel string, off []string) {
		t.Helper()
		if len(off) != 1 || off[0] != rel+": SetEntryPermit" {
			t.Fatalf("%s calls SetEntryPermit; census offenders = %v", rel, off)
		}
	}
	check(t, "api/unwire.go", plant(t, "api/unwire.go", "api")) // positive control
	for _, dir := range append(censuswalk.NestedProbeDirs(), "trader/web") {
		t.Run(dir, func(t *testing.T) {
			rel := dir + "/unwire.go"
			check(t, rel, plant(t, rel, censuswalk.PackageName(dir)))
		})
	}
}
