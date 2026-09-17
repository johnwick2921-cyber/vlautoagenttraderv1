package trader

import (
	"errors"
	"strings"
	"testing"

	"nofx/kernel"
)

// S3(c) — the repair block carries the relation vocabulary ONLY when the 4h
// structure is directional (up/down), never on range or an absent table.
func TestPlannerRejectBlockCarriesRelationLaw(t *testing.T) {
	err := errors.New("fixture reject")
	with := plannerRejectBlock(err, []string{"reject"}, "up")
	if !strings.Contains(with, kernel.RepairStructureRelationLaw) {
		t.Fatalf("4h=up reject block must carry the relation law: %q", with)
	}
	down := plannerRejectBlock(err, []string{"reject"}, "down")
	if !strings.Contains(down, kernel.RepairStructureRelationLaw) {
		t.Fatalf("4h=down reject block must carry the relation law: %q", down)
	}
	for _, absent := range []string{"range", ""} {
		block := plannerRejectBlock(err, []string{"reject"}, absent)
		if strings.Contains(block, kernel.RepairStructureRelationLaw) {
			t.Fatalf("4h=%q must NOT carry the relation law: %q", absent, block)
		}
	}
	if got := plannerRejectBlock(nil, []string{"reject"}, "up"); got != "" {
		t.Fatalf("nil error must render an empty block, got %q", got)
	}
}
