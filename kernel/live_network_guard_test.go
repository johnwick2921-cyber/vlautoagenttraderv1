package kernel

// tests-no-live-network GUARD — a unit test must NEVER call a live external
// service (CI killer 2026-09-26: TestBaseCoinSymbolsNoArgs called the real
// CoinAnk API with context.TODO() and no timeout, hanging the Backend and
// Go-coverage jobs on a slow API; seen on #243 and #255 runs).
//
// A test may exercise a live endpoint ONLY as an opt-in integration probe:
// the offending FILE must carry the NOFX_LIVE_TESTS gate (or call the coinank
// liveNetworkGate helper, which is that gate) and run under a bounded context.
// Everything else dials a fixture/httptest server or is loopback by
// construction — loopback and variable-built URLs (httptest servers) pass.
//
// Offenders this guard catches:
//   - http.Get / http.Post / http.PostForm / http.Head with a NON-loopback
//     string-literal URL (variable-built URLs are fixtures — e.g.
//     httptest.Server.URL, "http://127.0.0.1:"+port)
//   - net.Dial / net.DialTimeout with a NON-loopback string-literal host
//     ("unix" sockets and in-process 127.0.0.1 pass)
//   - websocket.Dial with a non-loopback literal URL
//   - the CoinAnk live entry points (BaseCoinSymbols, Kline, DepthWsConn,
//     KlineWsConn) in any form (same-package or coinank_api. selector)
//
// RED: fails on the pre-fix tree (the coinank tests dial with no gate).
// GREEN: the coinank tests are gated and everything else is loopback.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// liveNetworkHttpVerbs are the http verbs whose call means a real request.
var liveNetworkHttpVerbs = map[string]bool{
	"Get":      true,
	"Post":     true,
	"PostForm": true,
	"Head":     true,
}

// liveCoinankEntryPoints are the CoinAnk API client entry points that dial the
// live endpoint. Extend this set if another live client package's tests appear.
var liveCoinankEntryPoints = map[string]bool{
	"BaseCoinSymbols": true,
	"Kline":           true,
	"DepthWsConn":     true,
	"KlineWsConn":     true,
}

// literalIsLoopback reports whether a string literal is a loopback host or a
// unix socket name — in-process, never a live external service.
func literalIsLoopback(lit string) bool {
	l := strings.ToLower(lit)
	if strings.HasPrefix(l, "unix") {
		return true
	}
	for _, lb := range []string{"127.0.0.1", "localhost", "::1"} {
		if strings.Contains(l, lb) {
			return true
		}
	}
	return false
}

// literalURLIsLive reports whether a string literal is a non-loopback URL.
func literalURLIsLive(lit string) bool {
	if literalIsLoopback(lit) {
		return false
	}
	l := strings.ToLower(lit)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "ws://") || strings.HasPrefix(l, "wss://")
}

// TestNoTestDialsALiveNetworkHostWithoutTheOptIn is the guard itself.
func TestNoTestDialsALiveNetworkHostWithoutTheOptIn(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	err = filepath.Walk(root, func(p string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "vendor", "web", ".understand-anything", ".Codex":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, src, 0)
		if perr != nil {
			return nil
		}
		live := false
		gated := strings.Contains(string(src), "NOFX_LIVE_TESTS")
		pos := fset.Position(f.Package)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// liveNetworkGate(t) is the opt-in gate itself.
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "liveNetworkGate" {
				gated = true
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				if id, ok := call.Fun.(*ast.Ident); ok && liveCoinankEntryPoints[id.Name] {
					live = true
					pos = fset.Position(call.Pos())
				}
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if id.Name == "http" && liveNetworkHttpVerbs[sel.Sel.Name] && len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING && literalURLIsLive(strings.Trim(lit.Value, `"`)) {
					live = true
					pos = fset.Position(call.Pos())
				}
				return true
			}
			if id.Name == "net" && (sel.Sel.Name == "Dial" || sel.Sel.Name == "DialTimeout") && len(call.Args) >= 2 {
				if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING && !literalIsLoopback(strings.Trim(lit.Value, `"`)) {
					live = true
					pos = fset.Position(call.Pos())
				}
				return true
			}
			if id.Name == "websocket" && sel.Sel.Name == "Dial" && len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING && literalURLIsLive(strings.Trim(lit.Value, `"`)) {
					live = true
					pos = fset.Position(call.Pos())
				}
				return true
			}
			if id.Name == "coinank_api" && liveCoinankEntryPoints[sel.Sel.Name] {
				live = true
				pos = fset.Position(call.Pos())
			}
			return true
		})
		if !live || gated {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		offenders = append(offenders, rel+":"+pos.String()+": test dials a live network host without the NOFX_LIVE_TESTS=1 opt-in (fixture/httptest or the gate)")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range offenders {
		t.Errorf("%s", o)
	}
	if len(offenders) == 0 {
		t.Log("no test dials a live network host without the opt-in")
	}
}
