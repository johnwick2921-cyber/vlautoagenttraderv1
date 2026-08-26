package kernel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTZGuardSingleTimeSource — the owner's P0 guard (2026-08-19): no
// owner-facing or prompt time may be emitted without going through the
// single timezone helpers in kernel/tz.go. It scans every non-test Go
// source under kernel/, trader/, api/ and agent/ and fails on:
//
//  1. bare time layouts ("15:04", "01-02 15:04", "2006-01-02 15:04",
//     with optional seconds) used outside kernel/tz.go — every clock
//     render must go through FormatCT / ClockCT / ClockCTAndUTC /
//     TableTimeCT (or carry an explicit zone suffix in the layout);
//  2. prompt table headers that still label bars "Time(UTC)";
//  3. the lunch no-trade window rendered without a CT label.
//
// One documented exemption: `TimeCT` storage fields (HH:MM by contract,
// rendered under a CT-labelled header) may keep a bare "15:04" layout on
// the SAME line.
var (
	tzGuardBareLayout = regexp.MustCompile(`^\s*(2006-01-02 |01-02 )?15:04(:05)?\s*$`)
	tzGuardScanDirs   = []string{"kernel", "trader", "api", "agent"}
)

func TestTZGuardSingleTimeSource(t *testing.T) {
	repo, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// kernel tests live in kernel/; step up to the repo root.
	for !strings.HasSuffix(repo, "nofx") && repo != "/" {
		repo = filepath.Dir(repo)
	}

	var violations []string
	for _, dir := range tzGuardScanDirs {
		root := filepath.Join(repo, dir)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if err := tzGuardCheckFile(t, path, root); err != nil {
				violations = append(violations, err.Error())
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("timezone guard failed (%d):\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}

func tzGuardCheckFile(t *testing.T, path, root string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, path)
	if rel == "tz.go" {
		return nil // the helpers themselves
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val := lit.Value
			if len(val) < 2 {
				return true
			}
			inner := val[1 : len(val)-1] // strip quotes
			switch {
			case strings.Contains(inner, "Time(UTC)") || strings.Contains(inner, "时间(UTC)"):
				t.Errorf("%s: prompt table header still labels bars UTC — must be Time(CT)", rel)
			case strings.Contains(inner, "12:00-13:30 lunch") && !strings.Contains(inner, "CT"):
				t.Errorf("%s: lunch no-trade window rendered without a CT label", rel)
			case tzGuardBareLayout.MatchString(inner):
				// Exempt the TimeCT storage field (HH:MM by contract).
				line := tzGuardLineAt(fset, rel, lit)
				if !strings.Contains(line, "TimeCT") {
					t.Errorf("%s:%s: bare time layout %q must route through kernel/tz.go helpers (FormatCT/ClockCT/ClockCTAndUTC/TableTimeCT) or carry an explicit zone suffix", rel, fset.Position(lit.Pos()), inner)
				}
			}
			return true
		})
	}
	return nil
}

func tzGuardLineAt(fset *token.FileSet, rel string, lit *ast.BasicLit) string {
	pos := fset.Position(lit.Pos())
	// Re-read the file — the parser is one file per call, cheap enough for a guard.
	src, err := os.ReadFile(pos.Filename)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(src), "\n")
	if pos.Line-1 < len(lines) {
		return lines[pos.Line-1]
	}
	return ""
}
