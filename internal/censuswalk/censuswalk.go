// Package censuswalk is the ONE definition of "the module's non-test Go
// source" for the census tests that confine a capability to named files:
// the hold-writer census and the worker import guard (store), the
// update-authorization census (internal/updateauth), the worker-socket
// literal census (internal/updaterwire) and the maintenance-setter census
// (trader).
//
// W-ONE-BUTTON M3 fold M5 (red-team 4 #1): each of those censuses carried its
// own copy of a skip list — .git node_modules web vendor .claude .Codex
// .understand-anything (and testdata in the import guard) — matched against a
// directory NAME AT ANY DEPTH. The Go toolchain compiles and links a package
// in api/web, internal/node_modules/x, api/.hidden, x/testdata/y or _x like
// any other ([A] go1.25.13: `go build` of an importer succeeds and
// `go list -deps` names them), so every census was blind to a package there.
//
// Here the skips apply ONLY to direct children of the module root — the SPA
// tree web/ (whose node_modules can carry third-party .go files:
// web/node_modules/flatted/golang/pkg/flatted in the deploy tree), a root
// node_modules/ or vendor/, and the tool dirs. Nothing below the root is
// skipped by name. Build tags are ignored (every non-test .go file counts),
// which can only over-report.
//
// The walk is also checked against the toolchain itself: ToolchainUncovered
// runs `go list -deps` and reports any file the build links that the walk
// does not cover (TestWalkCoversEveryPackageTheToolchainLinks, real module).
//
// Test tooling only: nothing but _test.go files may import this package.
package censuswalk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// rootOnlySkips are skipped ONLY as direct children of the module root.
var rootOnlySkips = map[string]bool{
	".git":                 true,
	"node_modules":         true,
	"web":                  true,
	"vendor":               true,
	".claude":              true,
	".Codex":               true,
	".understand-anything": true,
}

// SkippedAtRoot reports whether a direct child of the module root with this
// name is outside the census walk. The same name below the root is walked.
func SkippedAtRoot(name string) bool { return rootOnlySkips[name] }

// RootSkips returns the root-level skip names, sorted (for pins and reports).
func RootSkips() []string {
	out := make([]string, 0, len(rootOnlySkips))
	for k := range rootOnlySkips {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// NestedProbeDirs returns package directories that every census walk MUST
// cover: for each root-level skip name N, api/N and internal/N/p (the same
// name below the root is an ordinary compiled package), plus a testdata, an
// underscore and a dot directory. Each census pins itself against these
// (synthetic modules only — never plant in the real tree).
func NestedProbeDirs() []string {
	var out []string
	for _, n := range RootSkips() {
		out = append(out, "api/"+n, "internal/"+n+"/p")
	}
	return append(out, "x/testdata/y", "_x", "api/.hidden")
}

// PackageName is a valid Go package name for a probe dir.
func PackageName(dir string) string {
	n := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, filepath.Base(dir))
	if n == "" || n[0] >= '0' && n[0] <= '9' {
		n = "p" + n
	}
	return n
}

// File is one non-test Go source file of the module.
type File struct {
	Path string // absolute (root-joined) path
	Rel  string // module-relative, slash-separated
}

// NonTestGoFiles returns every non-test .go file under root, in walk order,
// except those under a root-level skipped directory.
func NonTestGoFiles(root string) ([]File, error) {
	root = filepath.Clean(root)
	var out []File
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && filepath.Dir(p) == root && SkippedAtRoot(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		// CTO CENSUS-GUARDS 1790306266164 [32]: WalkDir does not descend into
		// a symlinked dir, but the toolchain compiles THROUGH the link — a
		// package there would be invisible to every census using this walk.
		// Refuse it loudly instead of silently skipping it.
		if d.Type()&fs.ModeSymlink != 0 {
			if fi, serr := os.Stat(p); serr == nil && fi.IsDir() {
				target, _ := os.Readlink(p)
				return fmt.Errorf("symlinked package directory %s -> %s: the census walk refuses symlinked dirs — the toolchain compiles through them but the walk does not descend; move the package into the tree", p, target)
			}
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		out = append(out, File{Path: p, Rel: filepath.ToSlash(rel)})
		return nil
	})
	return out, err
}

// underRootSkip reports whether dir (absolute) lies under a root-level
// skipped directory of root.
func underRootSkip(root, dir string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(dir))
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	first := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
	return SkippedAtRoot(first)
}

// Package is one package as the toolchain reports it.
type Package struct {
	ImportPath string
	Dir        string
	Module     string   // module path; "" for the standard library
	Error      string   // go list -e error text, "" when none
	Files      []string // absolute paths of its non-test .go files (GoFiles, CgoFiles, IgnoredGoFiles)
}

// goCommand returns the go command for the toolchain that built this test
// binary (runtime.GOROOT, with GOTOOLCHAIN=local so it never switches), else
// the go on PATH. The environment is offline and read-only: GOPROXY=off,
// GOFLAGS=-mod=readonly, GOWORK=off — a census never downloads anything or
// edits go.mod.
func goCommand(dir string, args ...string) *exec.Cmd {
	bin := filepath.Join(runtime.GOROOT(), "bin", "go")
	toolchain := "GOTOOLCHAIN=local"
	if _, err := os.Stat(bin); err != nil {
		bin = "go"
		toolchain = ""
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	env := append([]string{}, os.Environ()...)
	env = append(env, "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOWORK=off")
	if toolchain != "" {
		env = append(env, toolchain)
	}
	cmd.Env = env
	return cmd
}

// ListPackages runs `go list -e [-deps] -json <patterns>` in root and returns
// every package it reports. A failing go command is an error (fail closed).
func ListPackages(root string, deps bool, patterns ...string) ([]Package, error) {
	args := []string{"list", "-e"}
	if deps {
		args = append(args, "-deps")
	}
	args = append(args, "-json=ImportPath,Dir,Module,Error,GoFiles,CgoFiles,IgnoredGoFiles")
	args = append(args, patterns...)
	cmd := goCommand(root, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	var pkgs []Package
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var raw struct {
			ImportPath string
			Dir        string
			Module     *struct{ Path string }
			Error      *struct{ Err string }
			GoFiles    []string
			CgoFiles   []string
			Ignored    []string `json:"IgnoredGoFiles"`
		}
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("go list: undecodable output: %v", err)
		}
		p := Package{ImportPath: raw.ImportPath, Dir: raw.Dir}
		if raw.Module != nil {
			p.Module = raw.Module.Path
		}
		if raw.Error != nil {
			p.Error = strings.ReplaceAll(raw.Error.Err, "\n", " ")
		}
		for _, group := range [][]string{raw.GoFiles, raw.CgoFiles, raw.Ignored} {
			for _, f := range group {
				if strings.HasSuffix(f, ".go") && !strings.HasSuffix(f, "_test.go") {
					p.Files = append(p.Files, filepath.Join(raw.Dir, f))
				}
			}
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// ModulePath reads the module path from root/go.mod.
func ModulePath(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "module" {
			return f[1], nil
		}
	}
	return "", fmt.Errorf("no module line in %s/go.mod", root)
}

// ToolchainUncovered asks the toolchain which packages of the module are
// linked by any package it matches with ./... (minus those under a root-level
// skipped directory — the SPA's node_modules is not the app) and reports every
// non-test .go file of a linked module package that the walk did NOT cover,
// and every module package the toolchain could not load (fail closed; a
// package whose files are ALL excluded by build constraints is not linkable
// and is tolerated, its files still counted). A test-only package has no
// non-test file and so nothing to cover. roots is how many packages were
// matched (a vacuity check for callers).
func ToolchainUncovered(root string) (uncovered []string, roots int, err error) {
	module, err := ModulePath(root)
	if err != nil {
		return nil, 0, err
	}
	files, err := NonTestGoFiles(root)
	if err != nil {
		return nil, 0, err
	}
	walked := map[string]bool{}
	for _, f := range files {
		walked[filepath.Clean(f.Path)] = true
	}
	matched, err := ListPackages(root, false, "./...")
	if err != nil {
		return nil, 0, err
	}
	var starts []string
	for _, p := range matched {
		if p.Dir != "" && underRootSkip(root, p.Dir) {
			continue
		}
		starts = append(starts, p.ImportPath)
	}
	if len(starts) == 0 {
		return nil, 0, fmt.Errorf("go list ./... matched no package outside the root-level skips")
	}
	linked, err := ListPackages(root, true, starts...)
	if err != nil {
		return nil, len(starts), err
	}
	for _, p := range linked {
		if p.Module != module {
			continue
		}
		if p.Error != "" && !strings.Contains(p.Error, "build constraints exclude all Go files") {
			uncovered = append(uncovered, p.ImportPath+": the toolchain could not load it ("+p.Error+")")
			continue
		}
		for _, f := range p.Files {
			if !walked[filepath.Clean(f)] {
				uncovered = append(uncovered, p.ImportPath+" ("+f+"): linked by the build, but outside the census walk")
			}
		}
	}
	sort.Strings(uncovered)
	return uncovered, len(starts), nil
}
