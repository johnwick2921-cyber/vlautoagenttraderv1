package branding

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
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
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "dist", ".understand-anything", ".claude", ".Codex":
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
