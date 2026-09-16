// W1 — deriving a scenario's price anchor, and admitting when there isn't one.
//
// PlanScenario names no level. The only prices it carries are the confirm
// reference and, when armed, the arm entry. Those are what the link can compare
// against — and a scenario with neither cannot be anchored at all, which is a
// COUNTED outcome, not a silent skip.

package trader

import (
	"testing"

	"nofx/kernel"
)

func TestConfirmRefPriceIsThePreferredAnchor(t *testing.T) {
	sc := kernel.PlanScenario{ID: "S1", Confirm: &kernel.PlanConfirm{RefPrice: 29600}}
	a, ok := scenarioAnchorFor(sc)
	if !ok || a.Price != 29600 || a.ID != "S1" {
		t.Fatalf("confirm ref price should anchor the scenario, got %+v ok=%v", a, ok)
	}
}

// The arm entry is the fallback, not the first choice: confirm is what the
// scenario is ABOUT, the arm is where it would be entered.
func TestArmEntryIsTheFallbackAnchor(t *testing.T) {
	sc := kernel.PlanScenario{ID: "S2", Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 29580}}
	a, ok := scenarioAnchorFor(sc)
	if !ok || a.Price != 29580 {
		t.Fatalf("arm entry should anchor when there is no confirm, got %+v ok=%v", a, ok)
	}

	both := kernel.PlanScenario{ID: "S3",
		Confirm: &kernel.PlanConfirm{RefPrice: 29600},
		Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 29580}}
	if a, _ := scenarioAnchorFor(both); a.Price != 29600 {
		t.Errorf("confirm must win over the arm entry, got %v", a.Price)
	}
}

// A scenario with no price at all is UNANCHORABLE. It must report that, not
// return a zero price that would then "match" a level at 0.
func TestScenarioWithNoPriceIsUnanchorable(t *testing.T) {
	for _, sc := range []kernel.PlanScenario{
		{ID: "S1"},
		{ID: "S2", Confirm: &kernel.PlanConfirm{RefPrice: 0}},
		{ID: "S3", Arm: &kernel.PlanArmSpec{Enabled: false, Entry: 29580}},
	} {
		if a, ok := scenarioAnchorFor(sc); ok {
			t.Errorf("%s has no usable price but anchored at %v — a 0 anchor would match a level at 0", sc.ID, a.Price)
		}
	}
}

// The band the link uses must be the map's OWN cluster width, not a second
// tolerance invented here (class 97: one source, both readers).
func TestLinkBandIsTheMapClusterWidth(t *testing.T) {
	got := scenarioLinkBand()
	want := float64(kernel.LevelClusterTicks) * 0.25
	if got != want {
		t.Fatalf("link band = %v, want the map's cluster width %v — two tolerances would drift apart", got, want)
	}
	if got != 3.0 {
		t.Errorf("the documented width is 12 ticks = 3.00 points, got %v", got)
	}
}
