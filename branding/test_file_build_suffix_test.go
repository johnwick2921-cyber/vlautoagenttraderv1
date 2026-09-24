package branding

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// A Go file whose name ends in _<GOOS>, _<GOARCH> or _<GOOS>_<GOARCH> is
// silently built ONLY for that target. A test named `…_arm_test.go` ("arm" as in
// armed orders) never runs on amd64: `go test -run` prints "no tests to run" and
// exits 0 (W2, checklist class "a test file silently excluded by a GOOS/GOARCH
// filename suffix"). An intentionally platform-specific test must be listed
// here with its reason; none exists today.
var platformTestFileAllowlist = map[string]string{}

// The names `go tool dist list` reports (go1.2x), as filename-suffix build
// constraints.
var goosNames = []string{"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos"}
var goarchNames = []string{"386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips", "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm"}

// platformSuffix reports the build-constraint suffix a test file name carries,
// or "".
func platformSuffix(name string) string {
	base := strings.TrimSuffix(name, "_test.go")
	if base == name {
		return ""
	}
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return ""
	}
	last := parts[len(parts)-1]
	for _, a := range goarchNames {
		if last == a {
			return "_" + a
		}
	}
	for _, o := range goosNames {
		if last == o {
			return "_" + o
		}
	}
	return ""
}

func TestPlatformSuffixRecognisesEveryShape(t *testing.T) {
	for name, want := range map[string]string{
		"confirm_resolver_arm_test.go":     "_arm",
		"x_linux_test.go":                  "_linux",
		"x_windows_amd64_test.go":          "_amd64",
		"confirm_resolver_armgate_test.go": "",
		"linux_test.go":                    "",
		"armed_executor_test.go":           "",
		"plan_arm_legs_test.go":            "",
		"not_a_test_arm.go":                "",
	} {
		if got := platformSuffix(name); got != want {
			t.Errorf("platformSuffix(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNoTestFileCarriesAPlatformBuildSuffix(t *testing.T) {
	seen := 0
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// CLASS 258: skip names come from censuswalk and apply ONLY to
			// direct children of the module root.
			if path != root && filepath.Dir(path) == root && censuswalk.SkippedAtRoot(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		seen++
		rel := strings.TrimPrefix(filepath.ToSlash(path), "../")
		if s := platformSuffix(d.Name()); s != "" {
			if _, ok := platformTestFileAllowlist[rel]; !ok {
				t.Errorf("%s ends in %q — Go builds it ONLY for that platform, so on amd64 its tests never run; rename it (e.g. %s)", rel, s, strings.Replace(d.Name(), s+"_test.go", s+"gate_test.go", 1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if seen < 500 {
		t.Fatalf("the guard saw only %d test files — it is going vacuous", seen)
	}
}

// TestSuffixGuardSeesNestedSkipNamedDirs plants a platform-suffixed test file
// in EVERY censuswalk.NestedProbeDirs directory of a synthetic module and
// asserts the guard reports every one. With the old any-depth SkipDir the dirs
// named like a root skip were invisible (CLASS 258).
func TestSuffixGuardSeesNestedSkipNamedDirs(t *testing.T) {
	root := t.TempDir()
	dirs := censuswalk.NestedProbeDirs()
	for _, dir := range dirs {
		full := filepath.Join(root, filepath.FromSlash(dir))
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + censuswalk.PackageName(dir) + "\n"
		if err := os.WriteFile(filepath.Join(full, "offender_linux_test.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var reported []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && filepath.Dir(path) == root && censuswalk.SkippedAtRoot(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if s := platformSuffix(d.Name()); s != "" {
			if _, ok := platformTestFileAllowlist[rel]; !ok {
				reported = append(reported, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range reported {
		seen[r] = true
	}
	var missed []string
	for _, dir := range dirs {
		if !seen[dir+"/offender_linux_test.go"] {
			missed = append(missed, dir)
		}
	}
	if len(missed) > 0 {
		t.Fatalf("the suffix guard skipped %d of %d nested probe dirs — a skip by NAME at depth exempts compiled packages (CLASS 258):\n\t%s",
			len(missed), len(dirs), strings.Join(missed, "\n\t"))
	}
}
