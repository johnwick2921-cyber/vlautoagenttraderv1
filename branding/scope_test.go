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

	"vl/internal/censuswalk"
)

// modulePrefix is the module path the census sees (vl today; the R5 rename
// makes it vl — the pins below must not hardcode it).
func modulePrefix() string {
	if m, err := censuswalk.ModulePath(".."); err == nil {
		return m
	}
	return "nofx" // pre-go.mod synthetic dirs only
}

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

// preserveImports protects the vl/ namespace: a vl/… import target the
// base file had must not disappear by RENAME (nofx/X → vl/X). W-EXEC-TRUTH W0
// (CTO ruling on M1): a target that left THIS file but is still imported by
// another tracked file (stillImported) was MOVED — a legitimate refactor — and
// is preserved; a target that vanished from the module, or one whose suffix
// reappears under a non-vl module-internal path in this file, is rejected.
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
		if !strings.HasPrefix(target, modulePrefix()+"/") || current[target] {
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
	suffix := strings.TrimPrefix(target, modulePrefix()+"/")
	for imp := range current {
		root, rest, ok := strings.Cut(imp, "/")
		if !ok || root == modulePrefix() || strings.Contains(root, ".") {
			continue // the module itself, or an external module (github.com/…)
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
	// Z21 (plan v7 FINAL R1b.10, owner ruling 2026-09-30): the protected
	// namespace is now vl/… and the old module prefix is forbidden (see
	// TestNoOldModuleImport). The base is re-pinned from the pre-rename commit
	// to the D2-DEAD item-12 tip (module vl, cmd/vl-*, no provider/nofxos) —
	// a pre-rename base would make every target skip once the prefix is vl/.
	// The R1b PR is merged with a MERGE COMMIT, never a squash: a squash drops
	// the pinned sha and the cat-file check below would skip the test.
	const base = "4bed716cdd5ae90dd7069f83030aa9867804fd34"
	// A mirror clone (the VL partner repo) does not carry vl history, so the
	// pin cannot be evaluated there: skip with the reason stated instead of
	// failing on `git diff` exit 128. In vl itself the commit exists and the
	// check runs unchanged.
	if _, err := git("cat-file", "-e", base+"^{commit}"); err != nil {
		t.Skipf("base commit %s is not in this repository (mirror clone) — import-target pin not evaluable here", base[:8])
	}
	// Z21: the pinned base must declare the module path the census sees at
	// HEAD. A re-pinned base whose go.mod disagrees is a bad pin and FAILS,
	// never skips — a vacuous base is the whole reason for the re-pin.
	goMod, err := git("show", base+":go.mod")
	if err != nil {
		t.Fatal(err)
	}
	want, err := censuswalk.ModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := moduleLine(goMod); got != want {
		t.Fatalf("base %s declares module %q, census sees %q — re-pin base", base[:8], got, want)
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
	err := preserveImports([]byte("package p; import \"vl/config\""), []byte("package p; import \"renamedroot/config\""), func(string) bool { return true })
	if err == nil || !strings.Contains(err.Error(), "vl/config") || strings.Contains(err.Error(), "renamedroot/config") {
		t.Fatalf("renamed import was not rejected: %v", err)
	}
}

func TestImportScopeAllowsObsoleteStandardLibraryRemoval(t *testing.T) {
	before := []byte("package p; import (\"crypto/sha256\"; \"encoding/hex\"; \"vl/config\")")
	after := []byte("package p; import \"vl/config\"")
	if err := preserveImports(before, after, nil); err != nil {
		t.Fatal(err)
	}
}

// W-EXEC-TRUTH W0 (CTO M1): an import MOVED to another file (the freeze gate
// left auto_trader_orders.go for entry_admission.go and took vl/discipline
// with it) is preserved; the same removal with no other importer is not.
func TestImportScopeAllowsMovedTarget(t *testing.T) {
	before := []byte("package p; import (\"vl/config\"; \"vl/discipline\")")
	after := []byte("package p; import \"vl/config\"")
	if err := preserveImports(before, after, func(t string) bool { return t == "vl/discipline" }); err != nil {
		t.Fatalf("a target still imported elsewhere was moved, not removed: %v", err)
	}
	if err := preserveImports(before, after, func(string) bool { return false }); err == nil {
		t.Fatal("a target no tracked file imports any more must still be rejected")
	}
}

// moduleLine extracts the module path a go.mod declares.
func moduleLine(goMod []byte) string {
	for _, ln := range strings.Split(string(goMod), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "module "))
		}
	}
	return ""
}

// TestNoOldModuleImport (Z21, owner ruling 2026-09-30): the protected namespace
// is now vl/…; the old module prefix is FORBIDDEN in every tracked .go file's
// imports. The token is assembled at runtime so this guard cannot itself trip a
// grep for the old name.
func TestNoOldModuleImport(t *testing.T) {
	old := "no" + "fx"
	out, err := exec.Command("git", "-C", "..", "ls-files", "-z", "*.go").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if f == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join("..", f))
		if err != nil {
			t.Fatal(err)
		}
		targets, err := importTargets(b)
		if err != nil {
			t.Fatal(err)
		}
		for tg := range targets {
			if seg, _, _ := strings.Cut(tg, "/"); strings.EqualFold(seg, old) {
				t.Errorf("%s imports the old module prefix: %s", f, tg)
			}
		}
	}
}
