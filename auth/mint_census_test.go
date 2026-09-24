package auth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// M3 red-team H1 census over the REAL tree: an UNSCOPED token (GenerateJWT —
// a user identity, admitted to the credential routes) is minted ONLY where a
// user proves their password or first creates the account: the login and
// register handlers in api/handler_user.go. Every other minting site — the
// Telegram bot (telegram/agent.GenerateBotToken), cmd/gate-jwt, anything
// added later — must mint a MACHINE token (GenerateScopedJWT), which
// authMiddleware denies on the credential, Telegram-config and update routes.
//
// Walk: every non-test .go file under the module root. Directories are
// skipped the way the go tool skips them (a name starting with "." or "_",
// or "testdata") plus, at the module ROOT only, web/ and vendor/ — a nested
// directory that merely happens to be named web/ is still walked.
func TestOnlyLoginAndRegisterMintUnscopedTokens(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]int{"api/handler_user.go": 2} // register + login
	seen := map[string]int{}
	scanned := 0
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			n := d.Name()
			if p != root && (strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "testdata") {
				return filepath.SkipDir
			}
			if rel == "web" || rel == "vendor" || rel == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		scanned++
		// The names this file reaches package nofx/auth by (an alias counts;
		// a dot-import makes a bare GenerateJWT the auth one).
		aliases, dot := map[string]bool{}, f.Name.Name == "auth"
		for _, im := range f.Imports {
			if strings.Trim(im.Path.Value, `"`) != "nofx/auth" {
				continue
			}
			switch {
			case im.Name == nil:
				aliases["auth"] = true
			case im.Name.Name == ".":
				dot = true
			default:
				aliases[im.Name.Name] = true
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr: // auth.GenerateJWT (any reference, not only a call)
				if id, ok := x.X.(*ast.Ident); ok && x.Sel.Name == "GenerateJWT" && aliases[id.Name] {
					seen[rel]++
				}
				return false
			case *ast.FuncDecl: // the declaration's own name is not a use
				if x.Body != nil {
					ast.Inspect(x.Body, func(m ast.Node) bool {
						if id, ok := m.(*ast.Ident); ok && dot && id.Name == "GenerateJWT" {
							seen[rel]++
						}
						if s, ok := m.(*ast.SelectorExpr); ok {
							if id, ok := s.X.(*ast.Ident); ok && s.Sel.Name == "GenerateJWT" && aliases[id.Name] {
								seen[rel]++
							}
							return false
						}
						return true
					})
				}
				return false
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 200 {
		t.Fatalf("positive control: scanned only %d Go files — the walk did not cover the module", scanned)
	}
	if seen["api/handler_user.go"] != allowed["api/handler_user.go"] {
		t.Fatalf("positive control: api/handler_user.go mints %d unscoped tokens, want %d (register + login)",
			seen["api/handler_user.go"], allowed["api/handler_user.go"])
	}
	var offenders []string
	for rel, n := range seen {
		if _, ok := allowed[rel]; !ok {
			offenders = append(offenders, rel)
			_ = n
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("unscoped user tokens minted outside login/register (must use GenerateScopedJWT): %v", offenders)
	}
}
