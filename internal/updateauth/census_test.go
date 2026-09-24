package updateauth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ── W-ONE-BUTTON M3 census: who may touch the enrollment, and who may mint ──
//
// Scans every non-test Go file in the module (the walk of
// store/maintenance_hold_writers_test.go):
//
//  1. the enrollment/seen file names are spelled ONLY in paths.go (so no
//     other file can build their paths by hand — os.Remove/os.WriteFile on a
//     hand-built path included);
//  2. Enroll (creates/replaces admin.json + device.key), Authorize and
//     ComputeMAC (mint a MAC) are called from outside this package ONLY by
//     the attended CLI — nothing in api/, the reset/password handlers, the
//     Telegram agent or the trader can enroll or mint (CTO ruling Q1(a));
//  3. LoadDeviceKey is called from outside this package ONLY by the API's
//     install handler/gate;
//  4. only the update handler, the Server wiring and updater-side packages
//     (internal/updater*, cmd/updater*) may import this package at all, and
//     never as a dot-import.
func TestUpdateAuthCensus(t *testing.T) {
	literalHome := "internal/updateauth/paths.go"
	fileNames := []string{adminFileName, deviceKeyName, seenFileName, enrollLockName, seenLockName}
	callers := map[string]map[string]bool{
		"Enroll":        {"internal/updaterbootstrap/bootstrap.go": true},
		"Authorize":     {"internal/updaterbootstrap/bootstrap.go": true},
		"ComputeMAC":    {"internal/updaterbootstrap/bootstrap.go": true},
		"LoadDeviceKey": {"api/handler_updates.go": true},
	}
	importers := func(rel string) bool {
		return rel == "api/handler_updates.go" || rel == "api/server.go" ||
			strings.HasPrefix(rel, "internal/updater") || strings.HasPrefix(rel, "cmd/updater")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	scanned := 0
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

		// 4. imports
		alias := ""
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			if path != "nofx/internal/updateauth" {
				continue
			}
			if !importers(rel) {
				offenders = append(offenders, rel+": imports nofx/internal/updateauth")
			}
			alias = "updateauth"
			if im.Name != nil {
				alias = im.Name.Name
				if alias == "." || alias == "_" {
					offenders = append(offenders, rel+": "+alias+"-imports nofx/internal/updateauth")
				}
			}
		}

		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit: // 1. file names
				if x.Kind == token.STRING && rel != literalHome {
					for _, name := range fileNames {
						if strings.Contains(x.Value, name) {
							offenders = append(offenders, rel+": spells "+name)
						}
					}
				}
			case *ast.SelectorExpr: // 2/3. outside callers (any reference, not only calls)
				if id, ok := x.X.(*ast.Ident); ok && alias != "" && id.Name == alias {
					if allowed, ok := callers[x.Sel.Name]; ok && !allowed[rel] {
						offenders = append(offenders, rel+": references updateauth."+x.Sel.Name)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scan saw only %d files — the walk is not covering the module", scanned)
	}
	if len(offenders) > 0 {
		t.Fatalf("update-authorization census:\n%s", strings.Join(offenders, "\n"))
	}
}
