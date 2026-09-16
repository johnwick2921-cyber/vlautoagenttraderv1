package store

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// P&L-TRUTH WAVE (2026-09-01) — THE LINT. A build-time scan over every
// prompt-facing package for an AGGREGATION over the raw realized_pnl column
// (or the coercing EffectivePnL accessor). The raw column is legitimately read
// only where pnl_corrected is WRITTEN (the per-close guard / correction
// tooling) — those sites are on the explicit allow-list below with a reason.
// Any new raw aggregation fails the build; the boot line states the contract.

// rawAggregationPatterns are the shapes that fold the raw column into a figure.
var rawAggregationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`SUM\(\s*realized_pnl`),
	regexp.MustCompile(`COALESCE\(\s*pnl_corrected\s*,\s*realized_pnl`),
	regexp.MustCompile(`IFNULL\(\s*pnl_corrected\s*,\s*realized_pnl`),
	regexp.MustCompile(`\+=\s*[\w.]*\.RealizedPnL\b`),
	regexp.MustCompile(`EffectivePnL\(\)`),
}

// pnlRawAllowList — file suffix → the reason the raw read is legitimate.
var pnlRawAllowList = map[string]string{
	"store/position.go":           "defines EffectivePnL (per-row display accessor; banned from aggregators) and WRITES realized_pnl at close",
	"store/pnl_correction.go":     "the correction tooling: reads realized_pnl to WRITE pnl_corrected",
	"trader/auto_trader_clock.go": "per-row close analytics on the row this process just closed (recompute check + watch backfill); not an aggregate",
}

// scanRawPnLAggregation walks root (non-test .go files) and returns every
// "path:line: text" hit that is not allow-listed. label is the repo-relative
// name of root (e.g. "store"), so allow-list suffixes read as repo paths.
func scanRawPnLAggregation(root, label string, allow map[string]string) []string {
	var hits []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relToRoot, rErr := filepath.Rel(root, path)
		if rErr != nil {
			relToRoot = path
		}
		rel := filepath.ToSlash(filepath.Join(label, relToRoot))
		for suffix := range allow {
			if strings.HasSuffix(rel, suffix) {
				return nil
			}
		}
		b, rErr := os.ReadFile(path)
		if rErr != nil {
			return nil
		}
		for i, line := range strings.Split(string(b), "\n") {
			code := line
			if idx := strings.Index(line, "//"); idx >= 0 {
				code = line[:idx]
			}
			for _, p := range rawAggregationPatterns {
				if p.MatchString(code) {
					hits = append(hits, rel+":"+itoa(i+1)+": "+strings.TrimSpace(line))
					break
				}
			}
		}
		return nil
	})
	return hits
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// TestPnlSurfaceGuardNoRawAggregation is the build gate.
func TestPnlSurfaceGuardNoRawAggregation(t *testing.T) {
	var hits []string
	for _, r := range []struct{ root, label string }{{".", "store"}, {"../api", "api"}, {"../trader", "trader"}, {"../agent", "agent"}, {"../kernel", "kernel"}} {
		hits = append(hits, scanRawPnLAggregation(r.root, r.label, pnlRawAllowList)...)
	}
	if len(hits) > 0 {
		t.Fatalf("corrected-column guard: %d raw P&L aggregation(s) outside the allow-list — read pnl_corrected via CorrectedPnL() and count the NULL rows as UNRESOLVED:\n  %s",
			len(hits), strings.Join(hits, "\n  "))
	}
	// Every registered surface must exist in its file (the boot line counts them).
	for _, sf := range PnLSurfaces() {
		b, err := os.ReadFile(filepath.Join("..", sf.File))
		if err != nil || !strings.Contains(string(b), "func "+sf.Name) && !strings.Contains(string(b), ") "+sf.Name+"(") {
			t.Errorf("registered P&L surface %s not found in %s (err=%v)", sf.Name, sf.File, err)
		}
	}
}

// F6 — the lint FAILS on a deliberately added raw aggregation and PASSES on an
// allow-listed write path.
func TestPnlSurfaceGuardCatchesRawAggregation(t *testing.T) {
	dir := t.TempDir()
	bad := "package x\nfunc f(rows []T) float64 { var s float64; for _, r := range rows { s += r.RealizedPnL }; return s }\n"
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	sql := "package x\nconst q = \"SELECT COALESCE(SUM(COALESCE(pnl_corrected, realized_pnl)), 0)\"\n"
	if err := os.WriteFile(filepath.Join(dir, "sql.go"), []byte(sql), 0o644); err != nil {
		t.Fatal(err)
	}
	if hits := scanRawPnLAggregation(dir, "x", nil); len(hits) != 2 {
		t.Fatalf("the lint must flag both raw aggregations, got %v", hits)
	}
	// Allow-listed write path → no hit.
	if hits := scanRawPnLAggregation(dir, "x", map[string]string{"x/bad.go": "write path", "x/sql.go": "correction tooling"}); len(hits) != 0 {
		t.Fatalf("allow-listed sites must pass, got %v", hits)
	}
	// A comment quoting the old shape is fine.
	ok := "package x\n// the old code did s += r.RealizedPnL — banned now\nfunc g() {}\n"
	_ = os.WriteFile(filepath.Join(dir, "bad.go"), []byte(ok), 0o644)
	_ = os.Remove(filepath.Join(dir, "sql.go"))
	if hits := scanRawPnLAggregation(dir, "x", nil); len(hits) != 0 {
		t.Fatalf("comments must not count, got %v", hits)
	}
}
