package kernel

import (
	"nofx/market"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// E6 — RETIREMENT. No surface may render a rate from the two biased
// instruments. The Go side is pinned here; the Guide is pinned by its own copy.
func TestBiasedInstrumentsRenderNoRate(t *testing.T) {
	// Both files must carry the retirement notice, so the next reader cannot
	// use them without seeing why they are wrong.
	for _, f := range []string{"level_stats_calc.go", "touch_telemetry.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "NO SURFACE MAY RENDER A RATE FROM IT") {
			t.Errorf("%s must carry the D4 retirement notice", f)
		}
		if !strings.Contains(string(b), "D1′") && !strings.Contains(string(b), "detector_d1prime") {
			t.Errorf("%s must name its calibrated replacement", f)
		}
	}
}

// D7 — the boot line reports REAL counts and names the retirement.
func TestDetectorBootLineIsReadNotLiteral(t *testing.T) {
	line := DetectorBootLine(0, 0)
	for _, want := range []string{"detector: D1′", "k=3", "H=12", "exit_on=close", "touch_outcomes=0", "candidate_pool=0"} {
		if !strings.Contains(line, want) {
			t.Errorf("boot line missing %q: %s", want, line)
		}
	}
	// The counts are arguments, so a populated table shows through.
	if !strings.Contains(DetectorBootLine(41, 820), "touch_outcomes=41") {
		t.Error("the boot line must report the counts it is given, never a literal")
	}
	t.Logf("boot: %s", line)
}

// D5 — ONE GEOMETRY. The resolved band scales with the tape; the legacy fixed
// bands survive only for the retired episode shape and are documented as such.
func TestResolvedTouchBandScalesWithTheTape(t *testing.T) {
	quiet := make([]market.Kline, 0, 200)
	loud := make([]market.Kline, 0, 200)
	p, q := 29000.0, 29000.0
	for i := 0; i < 200; i++ {
		p += float64(1-2*(i%2)) * 1.0  // Δ ≈ 1 pt
		q += float64(1-2*(i%2)) * 12.0 // Δ ≈ 12 pt
		ts := int64(1788400000000 + i*60000)
		quiet = append(quiet, market.Kline{OpenTime: ts, Close: p, High: p + 1, Low: p - 1})
		loud = append(loud, market.Kline{OpenTime: ts, Close: q, High: q + 1, Low: q - 1})
	}
	bq, okq := ResolvedTouchBandPoints(quiet)
	bl, okl := ResolvedTouchBandPoints(loud)
	if !okq || !okl {
		t.Fatalf("Δ must resolve on both tapes: quiet ok=%v loud ok=%v", okq, okl)
	}
	if bl <= bq {
		t.Errorf("the band must WIDEN with volatility: quiet=%.2f loud=%.2f", bq, bl)
	}
	// The legacy band cannot do this — that is the defect.
	if TouchBandPoints() != TouchBandPoints() || bq == bl {
		t.Error("a fixed band means different things at different volatilities")
	}
	// No bars → the caller is TOLD it is a fallback, never silently given a
	// different geometry.
	if _, ok := ResolvedTouchBandPoints(nil); ok {
		t.Error("with no Δ the caller must learn the band is a fallback (ok=false)")
	}
	t.Logf("resolved band: quiet Δ→%.2f pts · loud Δ→%.2f pts · legacy fixed %.2f pts", bq, bl, TouchBandPoints())
}

// The boot line's retirement claim is only allowed because THIS grep passes.
// It runs on every build so the claim cannot outlive its evidence.
func TestNoSurfaceRendersARetiredRate(t *testing.T) {
	roots := []string{"..", "../web/src"}
	// A RATE from the retired instruments — the published artifacts and the
	// field names that would carry one into a rendered surface.
	banned := []string{"84%", "70.3%", "75.1%", "reaction rate", "rejection rate"}
	var hits []string
	for _, root := range roots {
		_ = filepathWalk(root, func(path string, data string) {
			if strings.Contains(path, "_test.go") || strings.Contains(path, "/docs/") ||
				strings.Contains(path, "detector_d1prime") || strings.Contains(path, "guards.ts") {
				return // tests, reports and the retirement notice itself may NAME them
			}
			// Scan RENDERED text only. A comment naming the retired artifacts
			// is documentation — this file and the retirement notices do it
			// deliberately — while the defect is a rate reaching a log line, a
			// format string or a UI surface. First pass flagged only comments,
			// including a RandomBaseline = 0.50 that AGREES with D1′.
			for _, line := range strings.Split(data, "\n") {
				t := strings.TrimSpace(line)
				if t == "" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
					continue
				}
				for _, b := range banned {
					if strings.Contains(line, b) {
						hits = append(hits, path+" → "+b+"  |  "+strings.TrimSpace(line))
					}
				}
			}
		})
	}
	if len(hits) > 0 {
		t.Errorf("the boot line claims legacy rates are retired, but a surface still renders one:\n  %s",
			strings.Join(hits, "\n  "))
	}
}

func filepathWalk(root string, fn func(path, data string)) error {
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".tsx") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		fn(p, string(b))
		return nil
	})
}
