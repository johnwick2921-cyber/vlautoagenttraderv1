package auth

import (
	"os"
	"path/filepath"
	"testing"

	"nofx/internal/censuswalk"
)

// Class 258 inside M3's own auth censuses (found by the M3 class finalizer):
// TestOnlyLoginAndRegisterMintUnscopedTokens and TestServerNeverMintsAFutureIat
// walked the tree themselves and SkipDir'd every ".*", "_*" and testdata
// directory at ANY depth, and said that is how the go tool skips them. It is
// not: an imported package in api/.hidden, _x or x/testdata/y is compiled and
// linked. Both now walk with internal/censuswalk; this pin plants a minter in
// every censuswalk.NestedProbeDirs() directory of a synthetic module (never
// the real tree) and drives both production census functions.
func TestMintCensusesSeeNestedSkipNamedDirs(t *testing.T) {
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
	const (
		unscoped = "import \"nofx/auth\"\n\nfunc m() { _, _ = auth.GenerateJWT(\"u\", \"e\") }\n"
		direct   = "import \"github.com/golang-jwt/jwt/v5\"\n\nfunc m() { _ = jwt.NewWithClaims(nil, nil) }\n"
	)
	for _, dir := range append([]string{"api"}, censuswalk.NestedProbeDirs()...) { // api = positive control
		t.Run(dir, func(t *testing.T) {
			root := t.TempDir()
			write(root, "go.mod", "module nofx\n\ngo 1.25\n")
			pkg := "package " + censuswalk.PackageName(dir) + "\n\n"
			write(root, dir+"/unscoped.go", pkg+unscoped)
			write(root, dir+"/direct.go", pkg+direct)

			seen, _, err := unscopedMintSites(root)
			if err != nil {
				t.Fatal(err)
			}
			if seen[dir+"/unscoped.go"] != 1 {
				t.Errorf("%s/unscoped.go mints an unscoped user token (a compiled, importable package) and the mint census counts %d — want 1 (seen %v)", dir, seen[dir+"/unscoped.go"], seen)
			}
			minters, _, _, _, err := futureIatCensus(root)
			if err != nil {
				t.Fatal(err)
			}
			if minters[dir+"/direct.go"] != 1 {
				t.Errorf("%s/direct.go calls jwt.NewWithClaims outside signToken and the future-iat census counts %d — want 1 (minters %v)", dir, minters[dir+"/direct.go"], minters)
			}
		})
	}
}
