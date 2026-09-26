package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// ── W-ONE-BUTTON M2 site 6 — nothing in the trading API writes the hold ────
//
// The hold is written and cleared ONLY by the local operator CLI (and, from
// M4, the updater worker — admitted BY NAME in M3, below). No API route —
// resume, clear-freeze, anything future — may lift an installation-wide hold.
// This scans every non-test Go file in the module for calls to the three
// writers.
//
// M3 admission (design note §4.5, fail-closed default Q5): the worker's hold
// writer is admitted as the exact file internal/updaterworker/hold.go, which
// does not exist yet. An admitted-but-absent path grants nothing; when M4
// creates it, only that one file may write, and
// TestTradingAppNeverLinksTheUpdaterWorkerSide keeps the trading app from
// ever importing it.
var (
	holdWriterFiles = map[string]bool{
		"store/maintenance_hold.go":      true, // the definitions
		"internal/holdcli/holdcli.go":    true, // cmd/maintenance-hold
		"internal/updaterworker/hold.go": true, // M4 updater worker (admitted M3, by name)
	}
	// M2.1 (review 3 F13): names alone let os.Remove / os.WriteFile on the hold
	// path through. The path itself is confined too: MaintenanceHoldPath only in
	// the store, the CLI and the worker, the literal "hold.json" only in the store.
	holdPathUserFiles = map[string]bool{
		"store/maintenance_hold.go":      true,
		"internal/holdcli/holdcli.go":    true,
		"internal/updaterworker/hold.go": true, // M4 updater worker (admitted M3, by name)
	}
)

func TestOnlyTheOperatorCLIWritesTheMaintenanceHold(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	offenders, scanned, err := holdWriterOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scan saw only %d files — the walk is not covering the module", scanned)
	}
	if len(offenders) > 0 {
		t.Fatalf("the maintenance hold may be written/cleared only by the operator CLI (and the M4 updater, when it lands):\n%s", strings.Join(offenders, "\n"))
	}
}

// The census is proved on a synthetic module (M3): the worker's hold writer
// is admitted by its EXACT repo path. internal/updaterworker/hold.go calling
// the writers and MaintenanceHoldPath is clean (positive control); the same
// code under any other path — another file in the worker package, a deeper
// file with the same base name, the worker binary's main, the app's update
// handler, the wire — offends; and even the admitted file may not name
// "hold.json" (that literal stays in the store).
func TestHoldWriterCensusAdmitsTheWorkerOnlyByName(t *testing.T) {
	const worker = "package updaterworker\n\nimport \"nofx/store\"\n\n" +
		"func HoldForJob(d string) error {\n\t_ = store.MaintenanceHoldPath(d)\n\treturn store.WriteMaintenanceHold(d, store.MaintenanceHold{})\n}\n\n" +
		"func ReleaseJob(d string) error { return store.ClearMaintenanceHold(d) }\n"
	write := func(root, rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := func() string {
		root := t.TempDir()
		write(root, "go.mod", "module nofx\n\ngo 1.25\n")
		write(root, "store/maintenance_hold.go", "package store\n\nconst holdFile = \"hold.json\"\n\n"+
			"func MaintenanceHoldPath(d string) string { return d + \"/\" + holdFile }\n\n"+
			"func WriteMaintenanceHold(d string, h any) error { return nil }\n")
		write(root, "internal/holdcli/holdcli.go", "package holdcli\n\nimport \"nofx/store\"\n\nfunc Set(d string) error { return store.WriteMaintenanceHold(d, nil) }\n")
		write(root, "internal/updaterworker/hold.go", worker)
		write(root, "api/handler_updates.go", "package api\n")
		return root
	}
	// positive control: the admitted worker file writes, clears and resolves the path
	root := base()
	off, scanned, err := holdWriterOffenders(root)
	if err != nil || len(off) != 0 || scanned != 4 {
		t.Fatalf("clean synthetic module: offenders=%v scanned=%d err=%v (want none, 4 files)", off, scanned, err)
	}
	for name, c := range map[string]struct{ rel, body, want string }{
		"other file in the worker package": {"internal/updaterworker/other.go", strings.Replace(worker, "HoldForJob", "H2", 1), "internal/updaterworker/other.go: WriteMaintenanceHold"},
		"same base name, deeper path":      {"internal/updaterworker/sub/hold.go", worker, "internal/updaterworker/sub/hold.go: WriteMaintenanceHold"},
		"the worker binary's main":         {"cmd/nofx-updater/main.go", "package main\n\nimport \"nofx/store\"\n\nfunc main() { store.ClearMaintenanceHold(\"d\") }\n", "cmd/nofx-updater/main.go: ClearMaintenanceHold"},
		"the app's update handler":         {"api/handler_updates.go", "package api\n\nimport \"nofx/store\"\n\nfunc clear() { store.ForceClearMaintenanceHold(\"d\") }\n", "api/handler_updates.go: ForceClearMaintenanceHold"},
		"the wire resolving the hold path": {"internal/updaterwire/dial.go", "package updaterwire\n\nimport \"nofx/store\"\n\nvar p = store.MaintenanceHoldPath(\"d\")\n", "internal/updaterwire/dial.go: references MaintenanceHoldPath"},
		"admitted file naming hold.json":   {"internal/updaterworker/hold.go", worker + "\nvar raw = \"updater/hold.json\"\n", "internal/updaterworker/hold.go: names the hold file"},
	} {
		t.Run(name, func(t *testing.T) {
			root := base()
			write(root, c.rel, c.body)
			off, _, err := holdWriterOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasPrefix(o, c.want)
				// every fixture compiles: a hit must never ride on a
				// package the go/types leg could not type
				if strings.Contains(o, "cannot be typed") {
					t.Fatalf("the fixture does not compile: %s", o)
				}
			}
			if !hit {
				t.Fatalf("offenders = %v, want one starting %q", off, c.want)
			}
		})
	}
}

// holdWriterOffenders scans every non-test .go file under root and reports
// each call to a hold writer, each MaintenanceHoldPath reference and each
// "hold.json" literal outside the admitted files (repo-relative slash paths).
func holdWriterOffenders(root string) (offenders []string, scanned int, err error) {
	allowed := holdWriterFiles
	writers := map[string]bool{"WriteMaintenanceHold": true, "ClearMaintenanceHold": true, "ForceClearMaintenanceHold": true}
	pathUsers := holdPathUserFiles
	// M3 fold M5: the ONE root-only walk (censuswalk) — a skip-named dir below
	// the root (api/web, internal/node_modules/x, …) is a compiled package.
	files, err := censuswalk.NonTestGoFiles(root)
	if err != nil {
		return nil, 0, err
	}
	for _, file := range files {
		p, rel := file.Path, file.Rel
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if perr != nil {
			offenders = append(offenders, rel+": cannot be parsed, so it cannot be checked ("+perr.Error()+")")
			continue
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if writers[name] && !allowed[rel] {
				offenders = append(offenders, rel+": "+name)
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if x.Sel.Name == "MaintenanceHoldPath" && !pathUsers[rel] {
					offenders = append(offenders, rel+": references MaintenanceHoldPath")
				}
			case *ast.Ident:
				if x.Name == "MaintenanceHoldPath" && !pathUsers[rel] {
					offenders = append(offenders, rel+": references MaintenanceHoldPath")
				}
			case *ast.BasicLit:
				if strings.Contains(x.Value, "hold.json") && rel != "store/maintenance_hold.go" {
					offenders = append(offenders, rel+": names the hold file (\"hold.json\")")
				}
			}
			return true
		})
	}
	return offenders, scanned, nil
}

// The admission list is pinned exactly: widening it is a reviewed act, not a
// drive-by line in some other PR.
func TestHoldWriterAdmissionsArePinned(t *testing.T) {
	want := []string{"internal/holdcli/holdcli.go", "internal/updaterworker/hold.go", "store/maintenance_hold.go"}
	for name, m := range map[string]map[string]bool{"writers": holdWriterFiles, "path users": holdPathUserFiles} {
		var got []string
		for k, v := range m {
			if v {
				got = append(got, k)
			}
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("hold %s admitted = %v, want exactly %v", name, got, want)
		}
	}
	for _, api := range []string{"api/handler_updates.go", "api/handler_maintenance.go", "internal/updaterwire/dial.go", "internal/updaterwire/wireserver/server.go"} {
		if holdWriterFiles[api] || holdPathUserFiles[api] {
			t.Fatalf("%s must never be admitted as a hold writer — the app never writes the hold", api)
		}
	}
}

// ── W-ONE-BUTTON M3 — the trading app never links the worker side ─────────
//
// The app DIALS the updater worker (nofx/internal/updaterwire); only the
// worker binary may LISTEN (nofx/internal/updaterwire/wireserver) or hold
// the worker's hold writer (nofx/internal/updaterworker, M4). If api/,
// trader/, kernel/, agent/, telegram/, store/ or the root main package could
// reach either — directly or through any chain of module packages — an
// app-side bug could serve forged worker verbs or write the hold. (store/ is
// in the design note's list; the worker's hold.go imports store, so the
// reverse edge must never appear.) Build tags are ignored (every non-test
// file counts), which can only over-report.
//
// M3 fold M4 (red-team 4 #2(b)): the attended CLI's package
// nofx/internal/updaterbootstrap is forbidden too — its Run reaches
// updateauth.Authorize → ComputeMAC, the one door that mints an install MAC,
// and nothing on the app side may mint (CTO ruling Q1(a)). Only its own
// binary, cmd/updater-bootstrap, links it.
var (
	tradingAppDirs          = []string{"api", "trader", "kernel", "agent", "telegram", "store"}
	forbiddenWorkerPackages = []string{"nofx/internal/updaterwire/wireserver", "nofx/internal/updaterworker", "nofx/internal/updaterbootstrap",
		// M4 3b-B U5b (f), OQ-8/C2 (CTO 1790259689740): the kill/restart
		// library — only the worker binary links it
		// (TestWorkerImportGuardRefusesTheActivationLibrary).
		"nofx/internal/activation"}
)

func TestTradingAppNeverLinksTheUpdaterWorkerSide(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	offenders, guarded, err := workerImportOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	// every guarded root must actually have been walked — a walk that found
	// nothing (wrong module name, a skipped dir) would otherwise pass vacuously
	seen := map[string]bool{}
	for _, g := range guarded {
		seen[g] = true
	}
	for _, want := range append([]string{"nofx"}, prefixed("nofx/", tradingAppDirs)...) {
		if !seen[want] {
			t.Fatalf("guarded root %s was not walked (walked %d packages: %v) — the guard is not covering the app", want, len(guarded), guarded)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("the trading app must never link the updater worker side:\n%s", strings.Join(offenders, "\n"))
	}
}

// The guard itself is proved on a synthetic module: a direct import, a
// transitive import and a root-main import are each caught, and the same
// tree with the offending import removed is clean (positive control).
func TestWorkerImportGuardCatchesDirectAndTransitiveImports(t *testing.T) {
	write := func(root, rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := func() string {
		root := t.TempDir()
		write(root, "go.mod", "module nofx\n\ngo 1.25\n")
		write(root, "internal/updaterwire/dial.go", "package updaterwire\n")
		write(root, "internal/updaterwire/wireserver/server.go", "package wireserver\nimport _ \"nofx/internal/updaterwire\"\n")
		write(root, "internal/updaterworker/hold.go", "package updaterworker\n")
		write(root, "internal/helper/h.go", "package helper\n")
		write(root, "api/server.go", "package api\nimport _ \"nofx/internal/updaterwire\"\nimport _ \"nofx/internal/helper\"\n")
		write(root, "trader/t.go", "package trader\n")
		write(root, "main.go", "package main\nimport _ \"nofx/api\"\n")
		write(root, "cmd/updater-worker/main.go", "package main\nimport _ \"nofx/internal/updaterwire/wireserver\"\nimport _ \"nofx/internal/updaterworker\"\n")
		return root
	}
	// positive control: the app dials, the worker binary listens — clean
	root := base()
	if off, guarded, err := workerImportOffenders(root); err != nil || len(off) != 0 || strings.Join(guarded, ",") != "nofx,nofx/api,nofx/trader" {
		t.Fatalf("clean synthetic module: offenders=%v guarded=%v err=%v (want none; guarded nofx, nofx/api, nofx/trader)", off, guarded, err)
	}
	for name, c := range map[string]struct{ rel, body, want string }{
		"direct api":        {"api/worker.go", "package api\nimport _ \"nofx/internal/updaterwire/wireserver\"\n", "api"},
		"transitive helper": {"internal/helper/h.go", "package helper\nimport _ \"nofx/internal/updaterworker\"\n", "api"},
		"trader":            {"trader/w.go", "package trader\nimport w \"nofx/internal/updaterworker\"\nvar _ = w.X\n", "trader"},
		"root main":         {"main_worker.go", "package main\nimport _ \"nofx/internal/updaterwire/wireserver\"\n", "(root main)"},
		"store":             {"store/s.go", "package store\nimport _ \"nofx/internal/updaterworker\"\n", "store"},
	} {
		t.Run(name, func(t *testing.T) {
			root := base()
			write(root, c.rel, c.body)
			off, _, err := workerImportOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasPrefix(o, c.want+":")
			}
			if !hit {
				t.Fatalf("offenders = %v, want one starting %q", off, c.want+":")
			}
		})
	}
	// a _test.go file linking the worker is not production linkage
	root = base()
	write(root, "api/worker_test.go", "package api\nimport _ \"nofx/internal/updaterwire/wireserver\"\n")
	if off, _, _ := workerImportOffenders(root); len(off) != 0 {
		t.Fatalf("a test-only import must not count as production linkage: %v", off)
	}
}

// workerImportOffenders builds the module's package import graph from every
// non-test .go file under root (module path from root/go.mod) and reports,
// for each trading-app package, any path to a forbidden worker package.
// guarded lists the trading-app packages that were checked, sorted.
func workerImportOffenders(root string) (offenders []string, guarded []string, err error) {
	modBytes, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, nil, err
	}
	module := ""
	for _, line := range strings.Split(string(modBytes), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "module" {
			module = f[1]
		}
	}
	if module == "" {
		return nil, nil, fmt.Errorf("no module line in go.mod")
	}
	imports := map[string]map[string]bool{} // import path → module-internal imports
	// M3 fold M5: the ONE root-only walk (censuswalk). testdata is walked too:
	// a package under x/testdata/ is importable and linked like any other.
	files, err := censuswalk.NonTestGoFiles(root)
	if err != nil {
		return nil, nil, err
	}
	for _, file := range files {
		p := file.Path
		relDir, _ := filepath.Rel(root, filepath.Dir(p))
		pkg := module
		if relDir != "." {
			pkg = module + "/" + filepath.ToSlash(relDir)
		}
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if perr != nil {
			offenders = append(offenders, file.Rel+": cannot be parsed, so its imports cannot be checked")
			continue
		}
		if imports[pkg] == nil {
			imports[pkg] = map[string]bool{}
		}
		for _, im := range f.Imports {
			ip, _ := strconv.Unquote(im.Path.Value)
			if ip == module || strings.HasPrefix(ip, module+"/") {
				imports[pkg][ip] = true
			}
		}
	}
	forbidden := isForbiddenWorkerPackage
	var starts []string
	for pkg := range imports {
		if pkg == module {
			starts = append(starts, pkg)
			continue
		}
		for _, d := range tradingAppDirs {
			if pkg == module+"/"+d || strings.HasPrefix(pkg, module+"/"+d+"/") {
				starts = append(starts, pkg)
				break
			}
		}
	}
	sort.Strings(starts)
	for _, start := range starts {
		// BFS with parent links so the report names the chain
		parent := map[string]string{start: ""}
		queue := []string{start}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if forbidden(cur) {
				chain := []string{cur}
				for p := parent[cur]; p != ""; p = parent[p] {
					chain = append([]string{p}, chain...)
				}
				label := strings.TrimPrefix(strings.TrimPrefix(start, module), "/")
				if label == "" {
					label = "(root main)"
				}
				offenders = append(offenders, label+": "+strings.Join(chain, " → "))
				break
			}
			var next []string
			for ip := range imports[cur] {
				next = append(next, ip)
			}
			sort.Strings(next)
			for _, ip := range next {
				if _, seen := parent[ip]; !seen {
					parent[ip] = cur
					queue = append(queue, ip)
				}
			}
		}
	}
	return offenders, starts, nil
}

func isForbiddenWorkerPackage(ip string) bool {
	for _, f := range forbiddenWorkerPackages {
		if ip == f || strings.HasPrefix(ip, f+"/") {
			return true
		}
	}
	return false
}

// ── M3 fold M5: the same guard, answered by the toolchain ─────────────────
//
// workerImportOffenders models the import graph from source (every non-test
// file, build tags ignored — it can only over-report). This asks the Go
// toolchain what the trading app ACTUALLY links in the default build context
// (`go list -deps` of the root main package and every package under the
// trading-app dirs): whatever the source model misses — a directory the walk
// skipped, an import it mis-resolved — the linker's own answer cannot.
// patterns is what was asked (a vacuity check for callers).
func toolchainWorkerLinkOffenders(root string) (offenders []string, patterns []string, err error) {
	module, err := censuswalk.ModulePath(root)
	if err != nil {
		return nil, nil, err
	}
	patterns = []string{"."}
	for _, d := range tradingAppDirs {
		if fi, serr := os.Stat(filepath.Join(root, d)); serr == nil && fi.IsDir() {
			patterns = append(patterns, "./"+d+"/...")
		}
	}
	pkgs, err := censuswalk.ListPackages(root, true, patterns...)
	if err != nil {
		return nil, patterns, err
	}
	for _, p := range pkgs {
		if p.Module != module {
			continue
		}
		if p.Error != "" && !strings.Contains(p.Error, "build constraints exclude all Go files") {
			offenders = append(offenders, p.ImportPath+": the toolchain could not load it, so its links are unknown ("+p.Error+")")
			continue
		}
		if isForbiddenWorkerPackage(p.ImportPath) {
			offenders = append(offenders, p.ImportPath+": linked by the trading app (go list -deps "+strings.Join(patterns, " ")+")")
		}
	}
	sort.Strings(offenders)
	return offenders, patterns, nil
}

func TestTradingAppLinkageFromTheToolchain(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	offenders, patterns, err := toolchainWorkerLinkOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 1+len(tradingAppDirs) {
		t.Fatalf("asked the toolchain for %v — every trading-app dir must exist and be asked about", patterns)
	}
	if len(offenders) > 0 {
		t.Fatalf("the toolchain says the trading app links the updater worker side:\n%s", strings.Join(offenders, "\n"))
	}
}

func prefixed(prefix string, xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, prefix+x)
	}
	return out
}

// ── M4 3b-B U5b (g) — only the operator CLI ever SETS withdraw_entries ─────
//
// Owner rule (dispatch §0/§2, CTO 1790258770876): the updater never cancels
// orders, so the worker's hold NEVER carries withdraw_entries. The hold-writer
// census above admits internal/updaterworker/hold.go as a WRITER by exact
// name; that admission does not extend to this field. Only the operator CLI
// (internal/holdcli/holdcli.go — cmd/maintenance-hold --withdraw) may set it;
// store/maintenance_hold.go only DEFINES it (the struct tag). Reading it
// (trader/withdraw.go, the worker's own foreign-hold check) is fine.
//
// A "set" is any of: a composite-literal key WithdrawEntries / a
// "withdraw_entries" key; an assignment (any operator) to x.WithdrawEntries;
// taking &x.WithdrawEntries; a POSITIONAL MaintenanceHold{...} literal (it sets
// every field); or a string literal naming withdraw_entries (raw JSON) outside
// the two admitted files. U5f (U5b verifier D2) adds: a string literal naming
// WithdrawEntries (reflect by name), and a go/types leg (withdrawTypedOffenders)
// that resolves every composite literal's type — a KEYLESS literal whose type
// is ELIDED ([]MaintenanceHold{{…}}, map[…]MaintenanceHold{k: {…}}, pointer
// elements), written through a type ALIAS (this package's or another's), or a
// struct IDENTICAL to the hold's (converted) is a set.
var (
	withdrawSetterFiles  = map[string]bool{"internal/holdcli/holdcli.go": true}
	withdrawLiteralFiles = map[string]bool{"internal/holdcli/holdcli.go": true, "store/maintenance_hold.go": true}
)

// TestOnlyTheOperatorCLISetsWithdrawEntries is the pin over the real module.
//
// NAMED LIMITS (not chased, per the CTO's syntactic-belt ruling from M3 — a
// tripwire for honest code, not a sandbox; the full list is on
// withdrawTypedOffenders): a JSON key assembled at run time ("withdraw_" +
// "entries"); reflect with a COMPUTED field name (FieldByName(x), Field(i),
// FieldByIndex); unsafe and assembly.
func TestOnlyTheOperatorCLISetsWithdrawEntries(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	offenders, scanned, err := withdrawSetterOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scan saw only %d files — the walk is not covering the module", scanned)
	}
	if len(offenders) > 0 {
		t.Fatalf("withdraw_entries may be set only by the operator CLI (internal/holdcli/holdcli.go):\n%s", strings.Join(offenders, "\n"))
	}
	// the go/types leg is not vacuous: it typed every file the walk scanned
	// and resolved the two real hold literals (holdcli's set, the worker's
	// holdFor) to the hold type
	files, err := censuswalk.NonTestGoFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	_, stats, err := withdrawTypedOffenders(root, files)
	if err != nil {
		t.Fatal(err)
	}
	if stats.files != scanned || stats.holdLits < 2 {
		t.Fatalf("go/types leg typed %d of %d files and resolved %d hold literals (want every file, at least 2)", stats.files, scanned, stats.holdLits)
	}
}

// Proved on a synthetic module: the CLI's set and every read are clean
// (positive control), and each way of setting the field elsewhere — including
// in the worker's census-ADMITTED hold writer — is caught.
func TestWithdrawSetterCensusCatchesEveryForm(t *testing.T) {
	const worker = "package updaterworker\n\nimport \"nofx/store\"\n\n" +
		"func holdFor(j string) store.MaintenanceHold {\n\treturn store.MaintenanceHold{Held: true, JobID: j, Owner: \"updater\"}\n}\n\n" +
		"func ours(st store.MaintenanceHoldState, j string) bool {\n\treturn st.Hold.JobID == j && !st.Hold.WithdrawEntries\n}\n"
	base := func() string {
		root := t.TempDir()
		censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
		censusWrite(t, root, "store/maintenance_hold.go", "package store\n\ntype MaintenanceHold struct {\n\tHeld bool `json:\"held\"`\n\tJobID string `json:\"job_id\"`\n\tOwner string `json:\"owner,omitempty\"`\n\tWithdrawEntries bool `json:\"withdraw_entries,omitempty\"`\n}\n\ntype MaintenanceHoldState struct{ Hold MaintenanceHold }\n")
		censusWrite(t, root, "internal/holdcli/holdcli.go", "package holdcli\n\nimport (\n\t\"fmt\"\n\t\"nofx/store\"\n)\n\nfunc Set(w bool) store.MaintenanceHold {\n\th := store.MaintenanceHold{Held: true, WithdrawEntries: w}\n\tfmt.Printf(\"withdraw_entries=%v\\n\", h.WithdrawEntries)\n\th.WithdrawEntries = w\n\treturn h\n}\n")
		censusWrite(t, root, "internal/updaterworker/hold.go", worker)
		censusWrite(t, root, "trader/withdraw.go", "package trader\n\nimport \"nofx/store\"\n\nfunc wants(st store.MaintenanceHoldState) bool { return st.Hold.Held && st.Hold.WithdrawEntries }\n")
		// keyed literals — type elided, through an alias, pointer elements —
		// that never name the field are clean (the go/types leg's control)
		censusWrite(t, root, "internal/updaterworker/list.go", "package updaterworker\n\nimport \"nofx/store\"\n\ntype L = store.MaintenanceHold\n\nvar (\n\tsl = []store.MaintenanceHold{{Held: true, JobID: \"j\"}}\n\tpl = []*L{{Held: true}}\n\tml = map[string]L{\"a\": {JobID: \"j\"}}\n\tal = L{Owner: \"updater\"}\n\tel = []store.MaintenanceHold{{}}\n)\n")
		return root
	}
	root := base()
	if off, scanned, err := withdrawSetterOffenders(root); err != nil || len(off) != 0 || scanned != 5 {
		t.Fatalf("clean synthetic module: offenders=%v scanned=%d err=%v (want none, 5 files)", off, scanned, err)
	}
	for name, c := range map[string]struct{ rel, body, want string }{
		"the admitted hold writer setting it in its literal": {"internal/updaterworker/hold.go", strings.Replace(worker, `Owner: "updater"}`, `Owner: "updater", WithdrawEntries: true}`, 1), "internal/updaterworker/hold.go: composite literal sets WithdrawEntries"},
		"an assignment in the worker":                        {"internal/updaterworker/set.go", "package updaterworker\n\nimport \"nofx/store\"\n\nfunc f(h *store.MaintenanceHold) { h.WithdrawEntries = true }\n", "internal/updaterworker/set.go: assigns WithdrawEntries"},
		"an op-assignment":                                   {"internal/updaterworker/set.go", "package updaterworker\n\nimport \"nofx/store\"\n\nfunc f(h *store.MaintenanceHold, b bool) { h.WithdrawEntries = h.WithdrawEntries || b }\n", "internal/updaterworker/set.go: assigns WithdrawEntries"},
		"the field's address taken":                          {"cmd/nofx-updater/main.go", "package main\n\nimport \"nofx/store\"\n\nfunc main() { var h store.MaintenanceHold; p := &h.WithdrawEntries; *p = true }\n", "cmd/nofx-updater/main.go: takes the address of WithdrawEntries"},
		"a positional literal":                               {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\nvar h = store.MaintenanceHold{true, \"j\", \"updater\", true}\n", "internal/updaterworker/pos.go: positional MaintenanceHold literal"},
		"raw JSON naming the key":                            {"cmd/nofx-updater/main.go", "package main\n\nconst raw = `{\"held\":true,\"withdraw_entries\":true}`\n\nfunc main() {}\n", "cmd/nofx-updater/main.go: names withdraw_entries"},
		"a map literal keyed withdraw_entries":               {"api/handler_updates.go", "package api\n\nvar m = map[string]any{\"withdraw_entries\": true}\n", "api/handler_updates.go: composite literal sets withdraw_entries"},
		// U5b verifier D2 — the forms a syntactic walk cannot type (go/types leg)
		"a keyless literal, type elided, in a slice":             {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\nvar hs = []store.MaintenanceHold{{true, \"j\", \"updater\", true}}\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"a keyless literal, type elided, in a map":               {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\nvar m = map[string]store.MaintenanceHold{\"a\": {true, \"j\", \"updater\", true}}\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"a keyless literal, type elided, pointer elements":       {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\nvar hs = []*store.MaintenanceHold{{true, \"j\", \"updater\", true}}\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"a positional literal through a type alias":              {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\ntype H = store.MaintenanceHold\n\nvar h = H{true, \"j\", \"updater\", true}\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"a positional literal through another package's alias":   {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/trader/holdalias\"\n\nvar h = holdalias.H{true, \"j\", \"updater\", true}\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"a positional literal of an identical struct, converted": {"internal/updaterworker/pos.go", "package updaterworker\n\nimport \"nofx/store\"\n\ntype twin struct {\n\tHeld bool\n\tJobID, Owner string\n\tWithdrawEntries bool\n}\n\nvar h = store.MaintenanceHold(twin{true, \"j\", \"updater\", true})\n", "internal/updaterworker/pos.go: keyless MaintenanceHold literal"},
		"reflect naming the field":                               {"internal/updaterworker/refl.go", "package updaterworker\n\nimport (\n\t\"reflect\"\n\n\t\"nofx/store\"\n)\n\nfunc f(h *store.MaintenanceHold) { reflect.ValueOf(h).Elem().FieldByName(\"WithdrawEntries\").SetBool(true) }\n", "internal/updaterworker/refl.go: names WithdrawEntries in a string literal"},
		"a keyless literal in a file outside the host build":     {"internal/updaterworker/pos_other.go", "//go:build plan9\n\npackage updaterworker\n\nimport \"nofx/store\"\n\nvar hs = []store.MaintenanceHold{{true, \"j\", \"updater\", true}}\n", "internal/updaterworker/pos_other.go: keyless MaintenanceHold literal"},
	} {
		t.Run(name, func(t *testing.T) {
			root := base()
			if strings.Contains(c.body, "nofx/trader/holdalias") {
				censusWrite(t, root, "trader/holdalias/alias.go", "package holdalias\n\nimport \"nofx/store\"\n\ntype H = store.MaintenanceHold\n")
			}
			censusWrite(t, root, c.rel, c.body)
			off, _, err := withdrawSetterOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasPrefix(o, c.want)
				// every fixture compiles: a hit must never ride on a
				// package the go/types leg could not type
				if strings.Contains(o, "cannot be typed") {
					t.Fatalf("the fixture does not compile: %s", o)
				}
			}
			if !hit {
				t.Fatalf("offenders = %v, want one starting %q", off, c.want)
			}
		})
	}
}

// The go/types leg fails closed: a package it cannot type is an offender,
// never a silent pass.
func TestWithdrawTypedLegFailsClosed(t *testing.T) {
	root := t.TempDir()
	censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	censusWrite(t, root, "store/maintenance_hold.go", "package store\n\ntype MaintenanceHold struct {\n\tHeld bool `json:\"held\"`\n\tWithdrawEntries bool `json:\"withdraw_entries,omitempty\"`\n}\n")
	censusWrite(t, root, "internal/updaterworker/bad.go", "package updaterworker\n\nimport \"nofx/store\"\n\nvar hs = []store.MaintenanceHold{{true, true}}\n\nvar broken int = \"not an int\"\n")
	off, _, err := withdrawSetterOffenders(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(off, "\n")
	if !strings.Contains(joined, "internal/updaterworker/bad.go: its package does not") || !strings.Contains(joined, "cannot be typed") {
		t.Fatalf("offenders = %v, want the untypable package named as an offender", off)
	}
}

// The admissions are pinned exactly: the worker (any file) is never one.
func TestWithdrawSetterAdmissionsArePinned(t *testing.T) {
	for name, pair := range map[string]struct {
		m    map[string]bool
		want string
	}{
		"setters":  {withdrawSetterFiles, "internal/holdcli/holdcli.go"},
		"literals": {withdrawLiteralFiles, "internal/holdcli/holdcli.go,store/maintenance_hold.go"},
	} {
		var got []string
		for k, v := range pair.m {
			if v {
				got = append(got, k)
			}
		}
		sort.Strings(got)
		if strings.Join(got, ",") != pair.want {
			t.Fatalf("withdraw %s admitted = %v, want exactly %s", name, got, pair.want)
		}
	}
	for f := range withdrawSetterFiles {
		if strings.HasPrefix(f, "internal/updaterworker/") || strings.HasPrefix(f, "cmd/nofx-updater/") || strings.HasPrefix(f, "internal/updaterjob/") || strings.HasPrefix(f, "api/") {
			t.Fatalf("%s must never be admitted to set withdraw_entries — the updater never cancels orders", f)
		}
	}
}

func withdrawSetterOffenders(root string) (offenders []string, scanned int, err error) {
	files, err := censuswalk.NonTestGoFiles(root)
	if err != nil {
		return nil, 0, err
	}
	isField := func(e ast.Expr) bool {
		sel, ok := e.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "WithdrawEntries"
	}
	isHoldType := func(e ast.Expr) bool {
		switch x := e.(type) {
		case *ast.SelectorExpr:
			return x.Sel.Name == "MaintenanceHold"
		case *ast.Ident:
			return x.Name == "MaintenanceHold"
		}
		return false
	}
	for _, file := range files {
		rel := file.Rel
		f, perr := parser.ParseFile(token.NewFileSet(), file.Path, nil, 0)
		if perr != nil {
			offenders = append(offenders, rel+": cannot be parsed, so it cannot be checked ("+perr.Error()+")")
			continue
		}
		scanned++
		setter := withdrawSetterFiles[rel]
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				for _, el := range x.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						if isHoldType(x.Type) && !setter {
							offenders = append(offenders, rel+": positional MaintenanceHold literal (sets every field, withdraw_entries included)")
							return true
						}
						continue
					}
					switch k := kv.Key.(type) {
					case *ast.Ident:
						if k.Name == "WithdrawEntries" && !setter {
							offenders = append(offenders, rel+": composite literal sets WithdrawEntries")
						}
					case *ast.BasicLit:
						if strings.Contains(k.Value, "withdraw_entries") && !setter {
							offenders = append(offenders, rel+": composite literal sets withdraw_entries")
						}
					}
				}
			case *ast.AssignStmt:
				for _, l := range x.Lhs {
					if isField(l) && !setter {
						offenders = append(offenders, rel+": assigns WithdrawEntries")
					}
				}
			case *ast.IncDecStmt:
				if isField(x.X) && !setter {
					offenders = append(offenders, rel+": assigns WithdrawEntries")
				}
			case *ast.UnaryExpr:
				if x.Op == token.AND && isField(x.X) && !setter {
					offenders = append(offenders, rel+": takes the address of WithdrawEntries")
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING && strings.Contains(x.Value, "withdraw_entries") && !withdrawLiteralFiles[rel] {
					offenders = append(offenders, rel+": names withdraw_entries in a string literal")
				}
				if x.Kind == token.STRING && strings.Contains(x.Value, "WithdrawEntries") && !withdrawLiteralFiles[rel] {
					offenders = append(offenders, rel+": names WithdrawEntries in a string literal (reflect by name)")
				}
			}
			return true
		})
	}
	typed, _, err := withdrawTypedOffenders(root, files)
	if err != nil {
		return nil, scanned, err
	}
	return append(offenders, typed...), scanned, nil
}

// ── U5f (U5b verifier D2) — the go/types leg ──────────────────────────────
//
// The syntactic walk above reads a literal's written type. It cannot see a
// literal whose type is ELIDED (an element of a []MaintenanceHold or
// map[…]MaintenanceHold literal), one written through a type ALIAS (type H =
// store.MaintenanceHold, in this package or another), or one of a struct type
// IDENTICAL to the hold's, converted. So this leg asks the compiler: every
// non-test file is type-checked (go/types, imports from the toolchain's own
// export data — go list -export), every CompositeLit's type is RESOLVED
// (aliases unaliased, a pointer element dereferenced), and a literal whose
// type is the hold — or a struct identical to it, tags ignored — is an
// offender outside the setter file when it is KEYLESS (it sets every field)
// or keyed WithdrawEntries.
//
// Fail closed: a host-build file that does not type-check is an offender; the
// toolchain failing is an error. A census file OUTSIDE the host build (another
// GOOS, //go:build ignore, a directory go list does not match) is checked with
// its directory's host-build files, errors tolerated, and a keyless literal
// whose type still does not resolve is an offender.
//
// NAMED LIMITS (not chased — the CTO's syntactic-belt ruling from M3: this is
// a tripwire for honest code, not a sandbox):
//   - a JSON key assembled at run time ("withdraw_" + "entries", a []byte
//     built by hand, a key read from a file) — only a literal naming it is seen;
//   - reflect with a COMPUTED field name (FieldByName(x) for a non-literal x,
//     Field(i) by index, FieldByIndex) — only the literal "WithdrawEntries" is;
//   - unsafe (pointer arithmetic onto the field, an unsafe.Pointer cast from a
//     look-alike struct) and assembly;
//   - generated or copied code that goes through encoding/gob, text/template
//     or any other by-name mechanism with a computed name.
func withdrawTypedOffenders(root string, files []censuswalk.File) ([]string, typedStats, error) {
	var stats typedStats
	module, err := censuswalk.ModulePath(root)
	if err != nil {
		return nil, stats, err
	}
	pkgs, err := goListExport(root)
	if err != nil {
		return nil, stats, err
	}
	exports := map[string]string{}
	for _, p := range pkgs {
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
	}
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		e, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %q", path)
		}
		return os.Open(e)
	})
	storePkg, err := imp.Import(module + "/store")
	if err != nil {
		return nil, stats, fmt.Errorf("import %s/store: %v", module, err)
	}
	holdObj, _ := storePkg.Scope().Lookup("MaintenanceHold").(*types.TypeName)
	if holdObj == nil {
		return nil, stats, fmt.Errorf("%s/store declares no type MaintenanceHold", module)
	}
	holdStruct, ok := holdObj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, stats, fmt.Errorf("%s/store.MaintenanceHold is not a struct", module)
	}
	isHold := func(t types.Type) bool {
		t = types.Unalias(t)
		if p, ok := t.Underlying().(*types.Pointer); ok {
			t = types.Unalias(p.Elem())
		}
		if n, ok := t.(*types.Named); ok && n.Obj().Name() == "MaintenanceHold" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == module+"/store" {
			return true
		}
		st, ok := t.Underlying().(*types.Struct)
		return ok && types.IdenticalIgnoreTags(st, holdStruct)
	}

	census := map[string]string{} // abs path → module-relative
	for _, f := range files {
		census[filepath.Clean(f.Path)] = f.Rel
	}
	var offenders []string
	covered := map[string]bool{}
	hostFiles := map[string][]string{} // dir → its host-build census files
	pkgPath := map[string]string{}     // dir → import path
	for _, p := range pkgs {
		if p.Module != module {
			continue
		}
		for _, g := range p.GoFiles {
			abs := filepath.Clean(filepath.Join(p.Dir, g))
			if _, ok := census[abs]; ok {
				hostFiles[p.Dir] = append(hostFiles[p.Dir], abs)
			}
		}
		pkgPath[p.Dir] = p.ImportPath
		if p.Error != "" && len(hostFiles[p.Dir]) > 0 {
			offenders = append(offenders, census[hostFiles[p.Dir][0]]+": its package does not load, so its literals cannot be typed ("+p.Error+")")
		}
	}
	check := func(dir string, paths []string, only map[string]bool, tolerant bool) {
		var parsed []*ast.File
		for _, p := range paths {
			f, perr := parser.ParseFile(fset, p, nil, 0)
			if perr != nil {
				continue // the syntactic leg already reports an unparsable file
			}
			parsed = append(parsed, f)
		}
		var typeErrs []error
		info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
		conf := types.Config{Importer: imp, Error: func(e error) { typeErrs = append(typeErrs, e) }}
		path := pkgPath[dir]
		if path == "" {
			path = "census/" + dir
		}
		_, _ = conf.Check(path, fset, parsed, info)
		if !tolerant && len(typeErrs) > 0 {
			offenders = append(offenders, census[paths[0]]+": its package does not type-check, so its literals cannot be typed ("+typeErrs[0].Error()+")")
		}
		for _, f := range parsed {
			abs := filepath.Clean(fset.File(f.Pos()).Name())
			rel := census[abs]
			if !only[abs] {
				continue
			}
			stats.files++
			setter := withdrawSetterFiles[rel]
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				keyless := false
				for _, el := range lit.Elts {
					if _, kv := el.(*ast.KeyValueExpr); !kv {
						keyless = true
					}
				}
				tv, ok := info.Types[lit]
				if !ok || tv.Type == nil || tv.Type == types.Typ[types.Invalid] {
					if keyless && !setter {
						offenders = append(offenders, rel+": keyless MaintenanceHold literal? its type does not resolve, so it cannot be cleared")
					}
					return true
				}
				if !isHold(tv.Type) {
					return true
				}
				stats.holdLits++
				if setter {
					return true
				}
				if keyless {
					offenders = append(offenders, rel+": keyless MaintenanceHold literal (go/types: "+tv.Type.String()+" — sets every field, withdraw_entries included)")
				}
				for _, el := range lit.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "WithdrawEntries" {
							offenders = append(offenders, rel+": composite literal sets WithdrawEntries (go/types: "+tv.Type.String()+")")
						}
					}
				}
				return true
			})
		}
	}
	dirs := make([]string, 0, len(hostFiles))
	for d := range hostFiles {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		only := map[string]bool{}
		for _, f := range hostFiles[d] {
			only[f], covered[f] = true, true
		}
		check(d, hostFiles[d], only, false)
	}
	// census files the host build does not compile: typed alongside their
	// directory's host-build files, errors tolerated
	rest := map[string][]string{}
	for abs := range census {
		if !covered[abs] {
			rest[filepath.Dir(abs)] = append(rest[filepath.Dir(abs)], abs)
		}
	}
	restDirs := make([]string, 0, len(rest))
	for d := range rest {
		restDirs = append(restDirs, d)
	}
	sort.Strings(restDirs)
	for _, d := range restDirs {
		sort.Strings(rest[d])
		only := map[string]bool{}
		for _, f := range rest[d] {
			only[f] = true
		}
		check(d, append(append([]string(nil), hostFiles[d]...), rest[d]...), only, true)
	}
	return offenders, stats, nil
}

// typedStats is the leg's vacuity evidence: how many census files it typed,
// and how many literals it resolved to the hold type (setter files included).
type typedStats struct{ files, holdLits int }

// goListExport runs `go list -e -export -deps -json ./...` in root with the
// census's offline, read-only toolchain environment (censuswalk's: this
// binary's GOROOT with GOTOOLCHAIN=local, GOPROXY=off, GOFLAGS=-mod=readonly,
// GOWORK=off). A failing go command is an error (fail closed).
type listedExport struct {
	ImportPath, Dir, Export, Module, Error string
	GoFiles                                []string
}

func goListExport(root string) ([]listedExport, error) {
	bin := filepath.Join(runtime.GOROOT(), "bin", "go")
	env := append([]string{}, os.Environ()...)
	env = append(env, "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOWORK=off")
	if _, err := os.Stat(bin); err != nil {
		bin = "go"
	} else {
		env = append(env, "GOTOOLCHAIN=local")
	}
	args := []string{"list", "-e", "-export", "-deps", "-json=ImportPath,Dir,Export,Module,Error,GoFiles", "./..."}
	cmd := exec.Command(bin, args...)
	cmd.Dir, cmd.Env = root, env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	var pkgs []listedExport
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var raw struct {
			ImportPath, Dir, Export string
			Module                  *struct{ Path string }
			Error                   *struct{ Err string }
			GoFiles                 []string
		}
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("go list: undecodable output: %v", err)
		}
		p := listedExport{ImportPath: raw.ImportPath, Dir: raw.Dir, Export: raw.Export, GoFiles: raw.GoFiles}
		if raw.Module != nil {
			p.Module = raw.Module.Path
		}
		if raw.Error != nil {
			p.Error = strings.ReplaceAll(raw.Error.Err, "\n", " ")
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}
