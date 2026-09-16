package api

import (
	"os"
	"regexp"
	"testing"

	"nofx/kernel"
)

// CLEANUP BATCH 2, B4 — THE SVP DEPTH IS THE CONSTANT, NOT A RETYPED 2000.
// Batch 1 fixed the comment at :50; the literal at :63 stayed. A literal that
// equals a constant is a second copy nobody can compare (class 105).
func TestSVPHandlerCarriesNoBare2000(t *testing.T) {
	src, err := os.ReadFile("handler_svp.go")
	if err != nil {
		t.Fatal(err)
	}
	// strip comments (prose may cite the number as history), then look for a
	// bare 2000 in code
	code := regexp.MustCompile(`(?m)//.*$`).ReplaceAllString(string(src), "")
	if regexp.MustCompile(`\b2000\b`).MatchString(code) {
		t.Fatalf("handler_svp.go still types 2000 in code — use kernel.AISVPBarCount (= %d)", kernel.AISVPBarCount)
	}
	if !regexp.MustCompile(`provider\(symbol, interval, kernel\.AISVPBarCount\)`).MatchString(code) {
		t.Fatal("the SVP provider call must read its depth from kernel.AISVPBarCount")
	}
	if kernel.AISVPBarCount != 2000 {
		t.Logf("note: kernel.AISVPBarCount is %d — the handler follows it", kernel.AISVPBarCount)
	}
}
