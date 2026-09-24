package store

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
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
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "web", "vendor", ".claude", ".Codex", ".understand-anything":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if perr != nil {
			offenders = append(offenders, rel+": cannot be parsed, so it cannot be checked ("+perr.Error()+")")
			return nil
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
		return nil
	})
	return offenders, scanned, err
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
	forbiddenWorkerPackages = []string{"nofx/internal/updaterwire/wireserver", "nofx/internal/updaterworker", "nofx/internal/updaterbootstrap"}
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
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "web", "vendor", ".claude", ".Codex", ".understand-anything", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		relDir, _ := filepath.Rel(root, filepath.Dir(p))
		pkg := module
		if relDir != "." {
			pkg = module + "/" + filepath.ToSlash(relDir)
		}
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if perr != nil {
			rel, _ := filepath.Rel(root, p)
			offenders = append(offenders, filepath.ToSlash(rel)+": cannot be parsed, so its imports cannot be checked")
			return nil
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
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	forbidden := func(ip string) bool {
		for _, f := range forbiddenWorkerPackages {
			if ip == f || strings.HasPrefix(ip, f+"/") {
				return true
			}
		}
		return false
	}
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

func prefixed(prefix string, xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, prefix+x)
	}
	return out
}
