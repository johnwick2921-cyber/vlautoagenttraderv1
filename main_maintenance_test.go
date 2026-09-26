package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/config"
	"nofx/internal/holdcli"
	"nofx/store"
)

// MUST-2 (CTO review of cb513079) — the production call-site proof: with a
// real installation layout (DB_PATH set ONLY in .env, as on the owner's and
// partners' machines), the data dir the running bot resolves (loadDotEnv →
// config.Init → resolveMaintenanceDataDir, the exact calls main makes) is the
// data dir the operator CLI writes — even when the CLI is run from another
// directory. A CLI that holds a file the bot never reads is a hold that does
// not exist.
func TestMaintenanceHoldCLIWritesTheFileTheBotReads(t *testing.T) {
	inst := t.TempDir()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/data.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The bot's database exists where .env says (a real install has it; the
	// CLI refuses an --install-dir without it — M2.1, review 2 N5).
	if err := os.MkdirAll(filepath.Join(inst, "var", "db"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst, "var", "db", "data.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH") // DB_PATH comes ONLY from .env, as on a real install

	// ── the bot, from its WorkingDirectory ──
	t.Chdir(inst)
	loadDotEnv(".env")
	config.Init()
	botDir := resolveMaintenanceDataDir(config.Get().DBPath)
	if want := filepath.Join(inst, "var", "db"); botDir != want {
		t.Fatalf("bot data dir = %q, want %q", botDir, want)
	}
	if store.ReadMaintenanceHold(botDir).Held {
		t.Fatal("precondition: no hold")
	}

	// ── the operator, from somewhere else, pointing at the installation ──
	t.Chdir(t.TempDir())
	os.Unsetenv("DB_PATH") // the operator's shell does not have the bot's env
	var out, errb bytes.Buffer
	if rc := holdcli.Run([]string{"--install-dir", inst, "set", "--job", "job-mustwo"}, &out, &errb); rc != 0 {
		t.Fatalf("cli set rc=%d: %s", rc, errb.String())
	}
	st := store.ReadMaintenanceHold(botDir)
	if !st.Held || st.Hold.JobID != "job-mustwo" {
		t.Fatalf("the bot does not see the hold the CLI wrote (bot reads %s; cli said %q): %+v", store.MaintenanceHoldPath(botDir), strings.TrimSpace(out.String()), st)
	}
}

// mainTopLevel returns main()'s top-level statements. The wiring below must be
// a TOP-LEVEL statement of main(): a commented-out call is not in the AST, and
// a call nested in an if/closure is not unconditional (review F1, test
// integrity — the old strings.Index pins passed with the call commented out).
func mainTopLevel(t *testing.T) []ast.Stmt {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "main" && fd.Recv == nil {
			return fd.Body.List
		}
	}
	t.Fatal("func main not found")
	return nil
}

// isSel reports whether e is the selector x.sel.
func isSel(e ast.Expr, x, sel string) bool {
	s, ok := e.(*ast.SelectorExpr)
	if !ok || s.Sel.Name != sel {
		return false
	}
	id, ok := s.X.(*ast.Ident)
	return ok && id.Name == x
}

// directCall returns the call when stmt IS an unconditional call statement.
func directCall(stmt ast.Stmt) *ast.CallExpr {
	es, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return nil
	}
	c, _ := es.X.(*ast.CallExpr)
	return c
}

// containsCall reports whether n contains a call to x.sel anywhere.
func containsCall(n ast.Node, x, sel string) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if c, ok := m.(*ast.CallExpr); ok && isSel(c.Fun, x, sel) {
			found = true
		}
		return !found
	})
	return found
}

// The data dir is configured by an UNCONDITIONAL top-level statement of main(),
// with the resolver over cfg.DBPath, AFTER the os.Args DB-path override and
// BEFORE the traders load (their TCP server and every gate read it).
func TestMainResolvesTheMaintenanceDataDirBeforeTradersLoad(t *testing.T) {
	stmts := mainTopLevel(t)
	set, load, override := -1, -1, -1
	for i, st := range stmts {
		if c := directCall(st); c != nil && isSel(c.Fun, "trader", "SetMaintenanceDataDir") && len(c.Args) == 1 {
			if arg, ok := c.Args[0].(*ast.CallExpr); ok {
				if id, ok := arg.Fun.(*ast.Ident); ok && id.Name == "resolveMaintenanceDataDir" && len(arg.Args) == 1 && isSel(arg.Args[0], "cfg", "DBPath") {
					set = i
				}
			}
		}
		if load < 0 && containsCall(st, "traderManager", "LoadTradersFromStore") {
			load = i
		}
		ast.Inspect(st, func(m ast.Node) bool {
			if as, ok := m.(*ast.AssignStmt); ok && len(as.Lhs) == 1 && isSel(as.Lhs[0], "cfg", "DBPath") {
				if ix, ok := as.Rhs[0].(*ast.IndexExpr); ok && isSel(ix.X, "os", "Args") {
					override = i
				}
			}
			return true
		})
	}
	if set < 0 {
		t.Fatal("main() must call trader.SetMaintenanceDataDir(resolveMaintenanceDataDir(cfg.DBPath)) as an unconditional TOP-LEVEL statement")
	}
	if load < 0 || override < 0 {
		t.Fatalf("anchors missing: load=%d override=%d", load, override)
	}
	if !(override < set && set < load) {
		t.Fatalf("order must be: DB path override (stmt %d) < SetMaintenanceDataDir (stmt %d) < LoadTradersFromStore (stmt %d)", override, set, load)
	}
}

// The 🔒 boot line is printed by an unconditional top-level statement of
// main(), after the data dir is configured (it READS the hold file).
func TestMainPrintsTheMaintenanceBootLine(t *testing.T) {
	stmts := mainTopLevel(t)
	set, line := -1, -1
	for i, st := range stmts {
		c := directCall(st)
		if c == nil {
			continue
		}
		if isSel(c.Fun, "trader", "SetMaintenanceDataDir") {
			set = i
		}
		if isSel(c.Fun, "logger", "Infof") && containsCall(c, "trader", "MaintenanceBootLine") {
			line = i
		}
	}
	if line < 0 {
		t.Fatal("main() must print trader.MaintenanceBootLine(...) from an unconditional top-level logger.Infof")
	}
	if set < 0 || set > line {
		t.Fatalf("the boot line (stmt %d) must follow SetMaintenanceDataDir (stmt %d)", line, set)
	}
}
