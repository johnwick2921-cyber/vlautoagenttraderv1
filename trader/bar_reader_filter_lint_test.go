package trader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ── ROLL WAVE E2 — NO LIVE BAR READER BYPASSES THE CONTRACT FILTER ──────────
//
// Every store read of bars in LIVE code goes through a contract-aware form:
// LastNBarsOn / BarsBetweenOn (with a contract), or WindowContract first. The
// unfiltered LastNBars / BarsBetween exist for research tooling and for the
// rehydrate's filtered-out COUNT, and are allowed only where listed here — an
// allow-list that must SHRINK, never grow silently.
//
// The class-113 caveat applies and is stated: this matches TEXT, not the call
// graph. It catches a new `.BarsBetween(` in trader/; it cannot catch a wrapper
// that hides one. It is the cheap tripwire, not the proof — the proof is the
// C3 grep in the report, re-run at merge.
func TestNoLiveBarReaderBypassesTheContractFilter(t *testing.T) {
	// Files where the UNFILTERED form is legitimate, and why.
	allowed := map[string]string{
		"ninjatrader/bar_persist_wire.go": "the rehydrate reads the unfiltered count ONLY to report how many bars the filter kept out (A9)",
	}
	unfiltered := regexp.MustCompile(`\.(LastNBars|BarsBetween)\(`)
	var offenders []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for i, ln := range strings.Split(string(b), "\n") {
			code := ln
			if k := strings.Index(code, "//"); k >= 0 {
				code = code[:k]
			}
			if !unfiltered.MatchString(code) {
				continue
			}
			if why, ok := allowed[path]; ok {
				_ = why
				continue
			}
			offenders = append(offenders, fmt.Sprintf("%s:%d  %s", path, i+1, strings.TrimSpace(ln)))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Errorf(`%d unfiltered bar read(s) in live trader code. Every live reader must ask for a contract:

  %s

Use LastNBarsOn(symbol, tf, contract, n) or BarsBetweenOn(symbol, tf, contract, from, to),
with the contract from at.currentContract() (current tape) or
BarHistory().WindowContract() (a historical window). A reader that reads across
the roll is the 292-point tape this wave ended. If the read is genuinely
research-only, add it to the allow-list above WITH its reason.`, len(offenders), strings.Join(offenders, "\n  "))
	}
}
