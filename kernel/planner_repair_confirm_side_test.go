package kernel

import (
	"strings"
	"testing"
)

// FOLD (2) — re-aimed item 3: the kill that is REALLY entry-shape is
// 412-414, "scenario[0].confirm.side \"\" invalid (above|below)" — a
// confirm-side shape defect. The persisted error must route to an excerpt
// stating the string enum (the PromptContract row-346 pin states it in the
// author prompt; the REPAIR side had no case).
func TestFpConfirmSideRepairExcerptRoutes(t *testing.T) {
	persisted := `scenario[0].confirm.side "" invalid (above|below)`
	p := BuildPlannerRepairPrompt("{}", persisted, nil)
	for _, want := range []string{
		`"side": "above" | "below"`,
		"STRING",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("confirm.side repair excerpt lacks %q:\n%s", want, p)
		}
	}
}
