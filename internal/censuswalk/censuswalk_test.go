package censuswalk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// PIN (M3 fold M5, red-team 4 #1): the skip names bind ONLY at the module
// root. Every probe dir below the root (api/web, internal/node_modules/p,
// api/.git, x/testdata/y, _x, api/.hidden, …) is walked; the root-level skip
// dirs are not; _test.go files never are.
func TestWalkSkipsOnlyAtTheModuleRoot(t *testing.T) {
	root := t.TempDir()
	var want []string
	for _, n := range RootSkips() {
		write(t, root, n+"/a.go", "package a\n")
		write(t, root, n+"/deeper/b.go", "package deeper\n")
	}
	for _, d := range NestedProbeDirs() {
		write(t, root, d+"/a.go", "package "+PackageName(d)+"\n")
		write(t, root, d+"/a_test.go", "package "+PackageName(d)+"\n")
		want = append(want, d+"/a.go")
	}
	write(t, root, "web.go", "package main\n") // a FILE named like a skip dir is ordinary
	want = append(want, "web.go")
	files, err := NonTestGoFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Rel)
		if f.Path != filepath.Join(root, filepath.FromSlash(f.Rel)) {
			t.Fatalf("Path %q does not match Rel %q", f.Path, f.Rel)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("walked:\n%s\nwant exactly:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The root-level skip set is pinned: widening it is a reviewed act.
func TestRootSkipsArePinned(t *testing.T) {
	want := ".Codex,.claude,.git,.understand-anything,node_modules,vendor,web"
	if got := strings.Join(RootSkips(), ","); got != want {
		t.Fatalf("root-level skips = %s, want exactly %s", got, want)
	}
}

// Ask the tool: every module package the toolchain links from anything
// `go list ./...` matches (outside the root-level skips) lies inside the walk.
// This is what makes a skip that hides a compiled package impossible to add
// silently — wherever it is and however it is spelled.
func TestWalkCoversEveryPackageTheToolchainLinks(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	uncovered, roots, err := ToolchainUncovered(root)
	if err != nil {
		t.Fatal(err)
	}
	if roots < 50 {
		t.Fatalf("go list ./... matched only %d packages — the toolchain is not seeing the module", roots)
	}
	if len(uncovered) > 0 {
		t.Fatalf("packages the build links but the census walk does not cover:\n%s", strings.Join(uncovered, "\n"))
	}
}

// ToolchainUncovered has teeth: on a synthetic module, a linked package under
// a root-level skip dir (web/evil) is reported; linked packages in every
// nested probe dir — including _x and x/testdata/y, which `go list ./...`
// itself never matches — are covered (walked). Negative control first.
func TestToolchainUncoveredControls(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	var imports []string
	for _, d := range NestedProbeDirs() {
		if strings.Contains("/"+d+"/", "/vendor/") {
			continue // nested vendor is not importable in module mode
		}
		write(t, root, d+"/a.go", "package "+PackageName(d)+"\n")
		imports = append(imports, "import _ \"nofx/"+d+"\"\n")
	}
	write(t, root, "api/api.go", "package api\n\n"+strings.Join(imports, ""))
	write(t, root, "main.go", "package main\n\nimport _ \"nofx/api\"\n\nfunc main() {}\n")
	uncovered, roots, err := ToolchainUncovered(root)
	if err != nil || len(uncovered) != 0 || roots == 0 {
		t.Fatalf("every nested probe dir is walked: uncovered=%v roots=%d err=%v", uncovered, roots, err)
	}
	write(t, root, "web/evil/e.go", "package evil\n")
	write(t, root, "api/evil.go", "package api\n\nimport _ \"nofx/web/evil\"\n")
	uncovered, _, err = ToolchainUncovered(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(uncovered) != 1 || !strings.HasPrefix(uncovered[0], "nofx/web/evil ") {
		t.Fatalf("a linked package under the root-level web/ must be reported, got %v", uncovered)
	}
}

// A SYMLINKED PACKAGE DIRECTORY IS REFUSED, NOT SKIPPED (CTO CENSUS-GUARDS
// 1790306266164 [32]). filepath.WalkDir does not descend into a symlinked
// dir, so a package the toolchain compiles THROUGH the link (api/hid ->
// elsewhere/hid, imported by api/api.go) was invisible to every per-census
// walk — only the suite-level ToolchainUncovered pin reported it. The walk
// must fail loud on the link so every census that uses NonTestGoFiles sees
// the package instead of silently skipping it.
func TestWalkRefusesSymlinkedPackageDir(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	write(t, root, "api/api.go", "package api\n\nimport _ \"nofx/api/hid\"\n")
	write(t, root, "elsewhere/hid/x.go", "package hid\n\nvar X = 1\n")
	if err := os.Symlink(filepath.Join(root, "elsewhere", "hid"), filepath.Join(root, "api", "hid")); err != nil {
		t.Fatal(err)
	}
	if _, err := NonTestGoFiles(root); err == nil {
		t.Fatal("the walk accepted a symlinked package directory — api/hid is compiled through the link but the walk does not descend into it, so every census using NonTestGoFiles was blind to nofx/api/hid")
	} else if !strings.Contains(err.Error(), filepath.Join("api", "hid")) {
		t.Fatalf("error does not name the symlinked dir: %v", err)
	}
}

// nonTestImporters asks the toolchain which packages matched by ./... import
// target from a NON-test file (.Imports never carries TestImports or
// XTestImports). .Imports is the CURRENT GOOS/GOARCH/tags only (census-repair
// verify, LOW note), so two more sources are read for imports: every
// non-test file a build constraint excludes from a matched package (another
// platform, a tag — go list's IgnoredGoFiles), and every non-test file of
// the census walk (build tags ignored) — because ./... does not even MATCH a
// package whose every file is excluded here (a _windows.go-only package, a
// //go:build ignore generator: "build constraints exclude all Go files").
// A failing go command, undecodable output or an unparsable file is an
// error (fail closed).
func nonTestImporters(root, target string) (importers []string, matched int, err error) {
	cmd := goCommand(root, "list", "-e", "-json=ImportPath,Dir,Imports,IgnoredGoFiles", "./...")
	out, err := cmd.Output()
	if err != nil {
		return nil, 0, err
	}
	seen := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p struct {
			ImportPath, Dir         string
			Imports, IgnoredGoFiles []string
		}
		if err := dec.Decode(&p); err != nil {
			return nil, 0, fmt.Errorf("go list: undecodable output: %v", err)
		}
		matched++
		hit := false
		for _, ip := range p.Imports {
			hit = hit || ip == target
		}
		for _, name := range p.IgnoredGoFiles {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Dir, name), nil, parser.ImportsOnly)
			if err != nil {
				return nil, 0, fmt.Errorf("%s: ignored by this build but cannot be read for imports: %v", filepath.Join(p.Dir, name), err)
			}
			for _, im := range f.Imports {
				if ip, _ := strconv.Unquote(im.Path.Value); ip == target {
					hit = true
				}
			}
		}
		if hit {
			seen[p.ImportPath] = true
		}
	}
	module, err := ModulePath(root)
	if err != nil {
		return nil, 0, err
	}
	files, err := NonTestGoFiles(root)
	if err != nil {
		return nil, 0, err
	}
	for _, file := range files {
		f, err := parser.ParseFile(token.NewFileSet(), file.Path, nil, parser.ImportsOnly)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: cannot be read for imports: %v", file.Rel, err)
		}
		for _, im := range f.Imports {
			if ip, _ := strconv.Unquote(im.Path.Value); ip == target {
				pkg := module
				if d := path.Dir(file.Rel); d != "." {
					pkg += "/" + d
				}
				seen[pkg] = true
			}
		}
	}
	for p := range seen {
		importers = append(importers, p)
	}
	sort.Strings(importers)
	return importers, matched, nil
}

// PIN (M3 census repair, verifier N1): the package doc says "Test tooling
// only: nothing but _test.go files may import this package" — asked of the
// toolchain, not asserted: no non-test file of the module imports censuswalk,
// and neither the app nor any cmd/ binary links it.
func TestCensusWalkIsTestToolingOnly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module, err := ModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	target := module + "/internal/censuswalk"
	importers, matched, err := nonTestImporters(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if matched < 50 {
		t.Fatalf("go list ./... matched only %d packages — the toolchain is not seeing the module", matched)
	}
	if len(importers) > 0 {
		t.Fatalf("non-test code imports %s (test tooling only):\n%s", target, strings.Join(importers, "\n"))
	}
	linked, err := ListPackages(root, true, ".", "./cmd/...")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range linked {
		if p.ImportPath == target {
			t.Fatalf("the app or a cmd/ binary links %s (go list -deps . ./cmd/...)", target)
		}
	}
}

// nonTestImporters has teeth: a non-test importer is reported, a _test.go
// importer is not.
func TestNonTestImportersControls(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	write(t, root, "internal/censuswalk/w.go", "package censuswalk\n")
	write(t, root, "api/api.go", "package api\n")
	write(t, root, "api/api_test.go", "package api\n\nimport _ \"nofx/internal/censuswalk\"\n")
	got, _, err := nonTestImporters(root, "nofx/internal/censuswalk")
	if err != nil || len(got) != 0 {
		t.Fatalf("a _test.go importer is test tooling: got %v err %v", got, err)
	}
	write(t, root, "api/uses.go", "package api\n\nimport _ \"nofx/internal/censuswalk\"\n")
	got, _, err = nonTestImporters(root, "nofx/internal/censuswalk")
	if err != nil || len(got) != 1 || got[0] != "nofx/api" {
		t.Fatalf("a non-test importer must be reported: got %v err %v", got, err)
	}
}

// PIN (census-repair verify, LOW note): go list's .Imports is the CURRENT
// GOOS/GOARCH/tags only, so a non-test importer behind another platform's
// build constraint — a _plan9.go file, a package whose every file is
// _windows.go, a //go:build ignore generator a `go run` would link — was
// invisible to TestCensusWalkIsTestToolingOnly. Each is reported — so is one
// in a package under the root-level web/ (outside the census walk, but
// matched by ./..., so read through go list's IgnoredGoFiles); an
// other-platform _test.go importer still is not.
func TestNonTestImportersSeesEveryPlatform(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	write(t, root, "internal/censuswalk/w.go", "package censuswalk\n")
	write(t, root, "api/api.go", "package api\n")
	write(t, root, "api/uses_plan9.go", "package api\n\nimport _ \"nofx/internal/censuswalk\"\n")
	write(t, root, "api/x_windows_test.go", "package api\n\nimport _ \"nofx/internal/censuswalk\"\n")
	write(t, root, "winonly/w_windows.go", "package winonly\n\nimport _ \"nofx/internal/censuswalk\"\n")
	write(t, root, "tools/gen.go", "//go:build ignore\n\npackage main\n\nimport _ \"nofx/internal/censuswalk\"\n\nfunc main() {}\n")
	write(t, root, "web/gopkg/a.go", "package gopkg\n")
	write(t, root, "web/gopkg/b_plan9.go", "package gopkg\n\nimport _ \"nofx/internal/censuswalk\"\n")
	write(t, root, "clean/c.go", "package clean\n")
	write(t, root, "clean/c_windows_test.go", "package clean\n\nimport _ \"nofx/internal/censuswalk\"\n")
	got, _, err := nonTestImporters(root, "nofx/internal/censuswalk")
	if err != nil {
		t.Fatal(err)
	}
	if want := "nofx/api,nofx/tools,nofx/web/gopkg,nofx/winonly"; strings.Join(got, ",") != want {
		t.Fatalf("importers behind another platform's build constraint: got %v, want exactly %s", got, want)
	}
}
