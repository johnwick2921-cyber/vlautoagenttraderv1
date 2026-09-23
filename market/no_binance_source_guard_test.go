package market

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── W-NO-BINANCE A — the source guard ──────────────────────────────────────
//
// No non-test file on the futures path's packages (market/, trader/, kernel/)
// may carry a Binance host except the sites below, each behind the crypto
// branch or dead. And the two Binance market-data fetchers may be CALLED only
// from the else-branch of an `if isFutures` — the crypto branch.

// binanceHostAllowlist: "<file>:<enclosing func or const/var name>" → why.
var binanceHostAllowlist = map[string]string{
	"market/data.go:getOpenInterestData": "crypto-perp OI; called only from the crypto branch (checked below)",
	"market/data.go:getFundingRate":      "crypto-perp funding; called only from the crypto branch (checked below)",
	"market/api_client.go:baseURL":       "APIClient's base URL; NewAPIClient is constructed only inside the two fetchers above",
	"market/historical.go:binanceFuturesKlinesURL": "GetKlinesRange — no caller anywhere (dead on the live path); deleted in W-NO-BINANCE Part B",
}

// Crypto-only renderers of funding that are NOT Binance hosts, for the record
// (CTO): kernel/grid_engine.go's 'Funding Rate' lines serve crypto grid
// strategies only; agent/agent.go and market.Format serve crypto tickers.

func TestNoBinanceHostOnTheFuturesPathOutsideTheAllowlist(t *testing.T) {
	fset := token.NewFileSet()
	found := map[string]bool{}
	for _, dir := range []string{"../market", "../trader", "../kernel"} {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				t.Fatalf("parse %s: %v", path, perr)
			}
			rel := strings.TrimPrefix(filepath.ToSlash(path), "../")
			encl := func(pos token.Pos) string {
				for _, d := range f.Decls {
					if d.Pos() <= pos && pos <= d.End() {
						switch d := d.(type) {
						case *ast.FuncDecl:
							return d.Name.Name
						case *ast.GenDecl:
							for _, s := range d.Specs {
								if vs, ok := s.(*ast.ValueSpec); ok && vs.Pos() <= pos && pos <= vs.End() {
									return vs.Names[0].Name
								}
							}
						}
					}
				}
				return "?"
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING || !strings.Contains(strings.ToLower(lit.Value), "binance.com") {
					return true
				}
				key := rel + ":" + encl(lit.Pos())
				found[key] = true
				if _, ok := binanceHostAllowlist[key]; !ok {
					t.Errorf("%s (%s): a Binance host outside the allowlist — the futures path must never reach Binance (W-NO-BINANCE A)", key, fset.Position(lit.Pos()))
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for key := range binanceHostAllowlist {
		if !found[key] {
			t.Errorf("allowlist row %q matches nothing — delete it (the allowlist only shrinks)", key)
		}
	}
}

// The two Binance fetchers are called ONLY from the crypto branch: inside the
// else of an `if isFutures { … }` in this package.
func TestBinanceFetchersAreCalledOnlyFromTheCryptoBranch(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	for _, p := range pkgs {
		for _, f := range p.Files {
			var stack []ast.Node
			ast.Inspect(f, func(n ast.Node) bool {
				if n == nil {
					stack = stack[:len(stack)-1]
					return false
				}
				stack = append(stack, n)
				ce, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := ce.Fun.(*ast.Ident)
				if !ok || (id.Name != "getOpenInterestData" && id.Name != "getFundingRate") {
					return true
				}
				calls++
				inCrypto := false
				for i := len(stack) - 1; i > 0; i-- {
					if ifs, ok := stack[i-1].(*ast.IfStmt); ok && ifs.Else == stack[i] {
						if c, ok := ifs.Cond.(*ast.Ident); ok && c.Name == "isFutures" {
							inCrypto = true
							break
						}
					}
				}
				if !inCrypto {
					t.Errorf("%s: %s is called outside the else-branch of `if isFutures` — the futures path would reach Binance", fset.Position(ce.Pos()), id.Name)
				}
				return true
			})
		}
	}
	if calls < 4 {
		t.Fatalf("expected the four crypto-branch calls (two per market read), found %d — the guard is going vacuous", calls)
	}
}
