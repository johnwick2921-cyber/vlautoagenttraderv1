package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// ── W-ONE-BUTTON M3 fold M5 (red-team 4 #1) — the census walk is root-only ──
//
// Both store censuses used to SkipDir any directory NAMED web, node_modules,
// vendor, .git, .claude, .Codex, .understand-anything (and testdata, in the
// import guard) at ANY depth. The toolchain compiles and links a package in
// api/web, internal/node_modules/nm, api/.hidden, x/testdata/y or _x like any
// other, so a hold writer or a worker-side import placed there was invisible.
// These pins plant each shape in a synthetic module (t.TempDir — never the
// real tree) and drive the PRODUCTION census functions; the toolchain
// (`go list -deps`) confirms each planted chain is really linked, so the pins
// cannot pass vacuously.

func censusWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// PIN (ported RT4-1b): a hold writer in a nested skip-named dir is an offender.
func TestHoldWriterCensusSeesNestedSkipNamedDirs(t *testing.T) {
	base := func() string {
		root := t.TempDir()
		censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
		censusWrite(t, root, "store/maintenance_hold.go", "package store\n\nfunc WriteMaintenanceHold(d string, h any) error { return nil }\nfunc ClearMaintenanceHold(d string) error { return nil }\n")
		censusWrite(t, root, "internal/holdcli/holdcli.go", "package holdcli\n\nimport \"nofx/store\"\n\nfunc Set(d string) error { return store.WriteMaintenanceHold(d, nil) }\n")
		return root
	}
	const writer = "import \"nofx/store\"\n\nfunc Lift() error { return store.ClearMaintenanceHold(\"d\") }\n"
	// positive control: the same writer in api/ is caught
	root := base()
	censusWrite(t, root, "api/lift.go", "package api\n\n"+writer)
	if off, _, err := holdWriterOffenders(root); err != nil || len(off) == 0 {
		t.Fatalf("positive control: api/lift.go calling ClearMaintenanceHold must be an offender (offenders=%v err=%v)", off, err)
	}
	for _, dir := range censuswalk.NestedProbeDirs() {
		t.Run(dir, func(t *testing.T) {
			root := base()
			rel := dir + "/lift.go"
			censusWrite(t, root, rel, "package "+censuswalk.PackageName(dir)+"\n\n"+writer)
			off, _, err := holdWriterOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasPrefix(o, rel+": ClearMaintenanceHold")
			}
			if !hit {
				t.Fatalf("%s calls store.ClearMaintenanceHold (a compiled, importable package) and the hold-writer census reports %v", rel, off)
			}
		})
	}
}

// PIN (ported RT4-1a): the import guard sees a chain through a nested
// skip-named dir. Ground truth from the toolchain first, so the probe is real.
func TestWorkerImportGuardSeesNestedSkipNamedDirs(t *testing.T) {
	base := func() string {
		root := t.TempDir()
		censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
		censusWrite(t, root, "internal/updaterwire/dial.go", "package updaterwire\n")
		censusWrite(t, root, "internal/updaterwire/wireserver/server.go", "package wireserver\n\nimport _ \"nofx/internal/updaterwire\"\n")
		censusWrite(t, root, "internal/updaterworker/hold.go", "package updaterworker\n")
		censusWrite(t, root, "api/server.go", "package api\n\nimport _ \"nofx/internal/updaterwire\"\n")
		censusWrite(t, root, "trader/t.go", "package trader\n")
		censusWrite(t, root, "main.go", "package main\n\nimport _ \"nofx/api\"\n\nfunc main() {}\n")
		return root
	}
	for _, dir := range censuswalk.NestedProbeDirs() {
		if strings.HasPrefix(dir+"/", "api/vendor/") || strings.Contains(dir, "/vendor/") {
			continue // not importable in module mode ("use of vendored package not allowed") — nothing to link; the hold census still walks it
		}
		t.Run(dir, func(t *testing.T) {
			root := base()
			censusWrite(t, root, dir+"/w.go", "package "+censuswalk.PackageName(dir)+"\n\nimport _ \"nofx/internal/updaterwire/wireserver\"\n")
			censusWrite(t, root, "api/uses.go", "package api\n\nimport _ \"nofx/"+dir+"\"\n")
			linked, err := censuswalk.ListPackages(root, true, ".")
			if err != nil {
				t.Fatal(err)
			}
			truth := false
			for _, p := range linked {
				truth = truth || p.ImportPath == "nofx/internal/updaterwire/wireserver"
			}
			if !truth {
				t.Fatalf("ground truth: the app binary does not link wireserver through nofx/%s — probe broken (%v)", dir, linked)
			}
			off, _, err := workerImportOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || (strings.HasPrefix(o, "api: ") && strings.Contains(o, "nofx/"+dir+" → nofx/internal/updaterwire/wireserver"))
			}
			if !hit {
				t.Fatalf("`go list -deps` links nofx/internal/updaterwire/wireserver via nofx/%s, but the import guard reports %v", dir, off)
			}
			// and the toolchain-answered guard says the same
			toff, _, err := toolchainWorkerLinkOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(toff) != 1 || !strings.HasPrefix(toff[0], "nofx/internal/updaterwire/wireserver: linked by the trading app") {
				t.Fatalf("toolchain guard via nofx/%s: %v", dir, toff)
			}
		})
	}
}

// The toolchain-answered guard is clean on a clean synthetic module (the app
// dials, the worker binary listens) and catches the direct import — positive
// and negative controls for toolchainWorkerLinkOffenders itself.
func TestToolchainWorkerLinkGuardControls(t *testing.T) {
	root := t.TempDir()
	censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	censusWrite(t, root, "internal/updaterwire/dial.go", "package updaterwire\n")
	censusWrite(t, root, "internal/updaterwire/wireserver/server.go", "package wireserver\n\nimport _ \"nofx/internal/updaterwire\"\n")
	censusWrite(t, root, "api/server.go", "package api\n\nimport _ \"nofx/internal/updaterwire\"\n")
	censusWrite(t, root, "main.go", "package main\n\nimport _ \"nofx/api\"\n\nfunc main() {}\n")
	censusWrite(t, root, "cmd/nofx-updater/main.go", "package main\n\nimport _ \"nofx/internal/updaterwire/wireserver\"\n\nfunc main() {}\n")
	if off, pats, err := toolchainWorkerLinkOffenders(root); err != nil || len(off) != 0 || strings.Join(pats, " ") != ". ./api/..." {
		t.Fatalf("clean synthetic module: offenders=%v patterns=%v err=%v", off, pats, err)
	}
	censusWrite(t, root, "api/worker.go", "package api\n\nimport _ \"nofx/internal/updaterwire/wireserver\"\n")
	if off, _, err := toolchainWorkerLinkOffenders(root); err != nil || len(off) != 1 {
		t.Fatalf("direct api import of wireserver: offenders=%v err=%v (want exactly one)", off, err)
	}
}
