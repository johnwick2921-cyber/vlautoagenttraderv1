package updaterwire

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/internal/installpath"
)

func TestSocketPathResolvesViaInstallpath(t *testing.T) {
	inst := t.TempDir()
	t.Setenv("DB_PATH", "")
	os.Unsetenv("DB_PATH")
	// default: <install>/data/updater/worker.sock
	got, err := SocketPathFor(inst)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(installpath.DataDir(inst, installpath.DefaultDBPath), "updater", "worker.sock")
	if got != want {
		t.Fatalf("SocketPathFor = %q, want %q", got, want)
	}
	// DB_PATH from <install>/.env moves the data dir — and the socket with it
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/bot.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = SocketPathFor(inst)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(inst, "var", "db", "updater", "worker.sock"); got != want {
		t.Fatalf("SocketPathFor with .env DB_PATH = %q, want %q", got, want)
	}
	// the process environment wins over .env, exactly as for the bot
	t.Setenv("DB_PATH", "/abs/elsewhere/x.db")
	got, _ = SocketPathFor(inst)
	if got != "/abs/elsewhere/updater/worker.sock" {
		t.Fatalf("env DB_PATH: %q", got)
	}
}

func TestSocketPathRefusesRelativeEmptyOrTooLong(t *testing.T) {
	for _, d := range []string{"", "data", "./data", "../data"} {
		if _, err := SocketPath(d); err == nil {
			t.Fatalf("SocketPath(%q) must refuse a non-absolute data dir", d)
		}
	}
	long := "/" + strings.Repeat("d", 120)
	if _, err := SocketPath(long); err == nil {
		t.Fatal("a socket path over the 107-byte sun_path limit must be refused, not truncated")
	}
	if p, err := SocketPath("/srv/nofx/data"); err != nil || p != "/srv/nofx/data/updater/worker.sock" {
		t.Fatalf("positive control: %q %v", p, err)
	}
}

// The socket filename literal lives in ONE file (as "hold.json" lives only in
// store/maintenance_hold.go): nothing else can name — and so unlink, chmod or
// squat — the worker socket without going through SocketPath.
func TestWorkerSocketLiteralIsConfined(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	scanned := 0
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "web", "vendor", ".claude", ".Codex", ".understand-anything":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if perr != nil {
			offenders = append(offenders, rel+": cannot be parsed")
			return nil
		}
		scanned++
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && strings.Contains(lit.Value, "worker.sock") && rel != "internal/updaterwire/paths.go" {
				offenders = append(offenders, rel+": names the worker socket (\"worker.sock\")")
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scan saw only %d files", scanned)
	}
	if len(offenders) > 0 {
		t.Fatalf("the worker socket may be named only by internal/updaterwire/paths.go:\n%s", strings.Join(offenders, "\n"))
	}
}
