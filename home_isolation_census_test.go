package main

// home_isolation_census — the CI-runnable grep for real-HOME writes, run the
// way the repo's census guards run (parsed, so a planted probe body inside a
// string literal is not a reference). Rules over every _test.go file outside
// internal/updaterbootstrap (which has its own attended seam + guard):
//
//   - a file that NAMES updaterbootstrap.Run / updaterbootstrap.Enroll (any
//     import alias, dot-import included) must isolate HOME in the same file
//     (`Setenv("HOME"` — the enroll path writes $HOME/.config/vl-updater/env);
//   - a file that NAMES os.UserHomeDir under any import name is an offender:
//     no test outside the guard itself may resolve the real HOME directly.
//
// Born from TEST-WROTE-REAL-WORKER-ENV-2: main_updater_bootstrap_test.go ran
// the real enroll with a temp --install-dir but no HOME seam, and rewrote the
// live ~/.config/vl-updater/env.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestHomeIsolationCensus(t *testing.T) {
	const bootstrapPkg = "vl/internal/updaterbootstrap"
	var offenders []string
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == ".Codex" || name == ".understand-anything" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, "_test.go") {
			return nil
		}
		dir := path.Dir(filepath.ToSlash(p))
		if dir == "internal/updaterbootstrap" || strings.HasPrefix(dir, "internal/updaterbootstrap/") ||
			dir == "internal/testhome" || strings.HasPrefix(dir, "internal/testhome/") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if perr != nil {
			offenders = append(offenders, p+": cannot be parsed, so it cannot be checked ("+perr.Error()+")")
			return nil
		}
		aliases := map[string]bool{}
		dotBootstrap, osAlias := false, ""
		for _, im := range f.Imports {
			ip := strings.Trim(im.Path.Value, `"`)
			switch {
			case ip == bootstrapPkg:
				if im.Name == nil {
					aliases["updaterbootstrap"] = true
				} else if im.Name.Name == "." {
					dotBootstrap = true
				} else if im.Name.Name != "_" {
					aliases[im.Name.Name] = true
				}
			case ip == "os":
				osAlias = "os"
				if im.Name != nil && im.Name.Name != "." && im.Name.Name != "_" {
					osAlias = im.Name.Name
				}
			}
		}
		namesBootstrap, namesUserHomeDir := false, false
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok {
					if aliases[id.Name] && (x.Sel.Name == "Run" || x.Sel.Name == "Enroll") {
						namesBootstrap = true
					}
					if osAlias != "" && id.Name == osAlias && x.Sel.Name == "UserHomeDir" {
						namesUserHomeDir = true
					}
				}
			case *ast.Ident:
				if dotBootstrap && (x.Name == "Run" || x.Name == "Enroll") {
					namesBootstrap = true
				}
			}
			return true
		})
		if namesUserHomeDir {
			offenders = append(offenders, p+": names os.UserHomeDir — the real HOME may only be resolved by the internal/testhome guard")
		}
		if namesBootstrap && !fileNamesSetenvHome(p) {
			offenders = append(offenders, p+": names updaterbootstrap.Run/Enroll without Setenv(\"HOME\") — enroll writes $HOME/.config/vl-updater/env; isolate HOME in the same file")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("home-isolation census offenders:\n%s", strings.Join(offenders, "\n"))
	}
}

func fileNamesSetenvHome(p string) bool {
	b, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), `Setenv("HOME"`)
}
