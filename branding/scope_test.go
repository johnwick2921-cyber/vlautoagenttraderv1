package branding

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func importTargets(source []byte) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "source.go", source, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	targets := map[string]bool{}
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		targets[path] = true
	}
	return targets, nil
}

// preserveImports protects the nofx/ namespace: a nofx/… import target the
// base file had must not disappear by RENAME (nofx/X → vl/X). W-EXEC-TRUTH W0
// (CTO ruling on M1): a target that left THIS file but is still imported by
// another tracked file (stillImported) was MOVED — a legitimate refactor — and
// is preserved; a target that vanished from the module, or one whose suffix
// reappears under a non-nofx module-internal path in this file, is rejected.
// stillImported nil = no move is recognized (the strict, original reading).
func preserveImports(before, after []byte, stillImported func(string) bool) error {
	old, err := importTargets(before)
	if err != nil {
		return err
	}
	current, err := importTargets(after)
	if err != nil {
		return err
	}
	for target := range old {
		if !strings.HasPrefix(target, "nofx/") || current[target] {
			continue
		}
		if stillImported != nil && stillImported(target) && !renamedInto(target, current) {
			continue // moved to another file, not renamed
		}
		return fmt.Errorf("existing import target changed or removed: %s", target)
	}
	return nil
}

// renamedInto reports whether the after-file imports target's path under a
// different, non-nofx module-internal root (nofx/config → vl/config).
func renamedInto(target string, current map[string]bool) bool {
	suffix := strings.TrimPrefix(target, "nofx/")
	for imp := range current {
		root, rest, ok := strings.Cut(imp, "/")
		if !ok || root == "nofx" || strings.Contains(root, ".") {
			continue // nofx itself, or an external module (github.com/…)
		}
		if rest == suffix {
			return true
		}
	}
	return false
}

// Owner-approved 105 correction: protect the project namespace, not obsolete
// standard-library dependencies. Removing unused hashing imports is legitimate;
// renaming nofx/... to vl/... remains forbidden. The module path has its own pin.
func TestExistingGoImportTargetsPreserved(t *testing.T) {
	root := ".."
	git := func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		return cmd.Output()
	}
	const base = "954f11b15f2e7615678f7d2b708c47895faebf1e"
	// The base is a nofx commit. A mirror clone (the VL partner repo) does not
	// carry nofx history, so the pin cannot be evaluated there: skip with the
	// reason stated instead of failing on `git diff` exit 128. In nofx itself
	// the commit exists and the check runs unchanged.
	if _, err := git("cat-file", "-e", base+"^{commit}"); err != nil {
		t.Skipf("base commit %s is not in this repository (mirror clone) — import-target pin not evaluable here", base[:8])
	}
	// Every import target any tracked .go file has at HEAD (the move test).
	files, err := git("ls-files", "*.go")
	if err != nil {
		t.Fatal(err)
	}
	headSet := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(files)), "\n") {
		if f == "" {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(root, f))
		if rerr != nil {
			continue
		}
		if targets, perr := importTargets(b); perr == nil {
			for tg := range targets {
				headSet[tg] = true
			}
		}
	}
	headImports := func(target string) bool { return headSet[target] }
	paths, err := git("diff", "--name-only", base, "--", "*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range strings.Split(strings.TrimSpace(string(paths)), "\n") {
		if path == "" {
			continue
		}
		before, err := git("show", base+":"+path)
		if err != nil {
			continue
		} // a new file has no pre-existing import targets
		after, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := preserveImports(before, after, headImports); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
func TestImportScopeRejectsRenamedTarget(t *testing.T) {
	err := preserveImports([]byte("package p; import \"nofx/config\""), []byte("package p; import \"vl/config\""), func(string) bool { return true })
	if err == nil || !strings.Contains(err.Error(), "nofx/config") {
		t.Fatalf("renamed import was not rejected: %v", err)
	}
}

func TestImportScopeAllowsObsoleteStandardLibraryRemoval(t *testing.T) {
	before := []byte("package p; import (\"crypto/sha256\"; \"encoding/hex\"; \"nofx/config\")")
	after := []byte("package p; import \"nofx/config\"")
	if err := preserveImports(before, after, nil); err != nil {
		t.Fatal(err)
	}
}

// W-EXEC-TRUTH W0 (CTO M1): an import MOVED to another file (the freeze gate
// left auto_trader_orders.go for entry_admission.go and took nofx/discipline
// with it) is preserved; the same removal with no other importer is not.
func TestImportScopeAllowsMovedTarget(t *testing.T) {
	before := []byte("package p; import (\"nofx/config\"; \"nofx/discipline\")")
	after := []byte("package p; import \"nofx/config\"")
	if err := preserveImports(before, after, func(t string) bool { return t == "nofx/discipline" }); err != nil {
		t.Fatalf("a target still imported elsewhere was moved, not removed: %v", err)
	}
	if err := preserveImports(before, after, func(string) bool { return false }); err == nil {
		t.Fatal("a target no tracked file imports any more must still be rejected")
	}
}
