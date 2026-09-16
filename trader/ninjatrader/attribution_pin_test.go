package ninjatrader

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// ATTRIBUTION E1-lite — the materializer must never create a row with an empty
// plan_id. "" is indistinguishable from "not yet stamped"; the sentinel says we
// looked and there was nothing to join on. Source pin, because the surrounding
// path needs a live NT8 snapshot to exercise end to end.
func TestAttributionMaterializerStampsSentinel(t *testing.T) {
	b, err := os.ReadFile("reconcile.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, `Source:             "reconcile",`)
	if i < 0 {
		t.Fatal("materialization site moved — re-locate before trusting this pin")
	}
	block := src[i:min(i+900, len(src))]
	// Whitespace-tolerant: gofmt aligns struct fields, so an exact-spacing
	// match is a pin that breaks on formatting rather than on behaviour.
	if !regexp.MustCompile(`PlanID:\s+store\.PlanUnresolvable`).MatchString(block) {
		t.Error("the materialized row must carry the sentinel, never an empty plan_id")
	}
	if !strings.Contains(block, "🔗 attribution: materialized") {
		t.Error("a lineage-less materialization must be LOUD (A9)")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
