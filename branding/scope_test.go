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
func preserveImports(before, after []byte) error {
	old, err := importTargets(before)
	if err != nil {
		return err
	}
	current, err := importTargets(after)
	if err != nil {
		return err
	}
	for target := range old {
		if strings.HasPrefix(target, "nofx/") && !current[target] {
			return fmt.Errorf("existing import target changed or removed: %s", target)
		}
	}
	return nil
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
		if err := preserveImports(before, after); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
func TestImportScopeRejectsRenamedTarget(t *testing.T) {
	err := preserveImports([]byte("package p; import \"nofx/config\""), []byte("package p; import \"vl/config\""))
	if err == nil || !strings.Contains(err.Error(), "nofx/config") {
		t.Fatalf("renamed import was not rejected: %v", err)
	}
}

func TestImportScopeAllowsObsoleteStandardLibraryRemoval(t *testing.T) {
	before := []byte("package p; import (\"crypto/sha256\"; \"encoding/hex\"; \"nofx/config\")")
	after := []byte("package p; import \"nofx/config\"")
	if err := preserveImports(before, after); err != nil {
		t.Fatal(err)
	}
}
