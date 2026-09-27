package trader

import "testing"

// WAVE PLANNER B3 — the ONE per-read line renders every field the plan file
// names: session, attempts, each reject's item class (the shared kernel
// classifier), the read→publish latency, the final lifecycle. A field the
// read cannot know prints n/a, never a fabricated number.
func TestPlannerReadLineRendersEveryField(t *testing.T) {
	r := int64(1_786_000_000_000)
	p := int64(1_786_000_001_500)
	got := plannerReadLine("NY", 3, []string{
		"born-dead authored scenario: S1: authored condition",
		"scenario[0].confirm.side \"at\" invalid (above|below)",
	}, &r, &p, "no_trade")
	want := "🧭 planner read: session=NY attempts=3 reject_classes=A6,A3 read→publish=1500ms lifecycle=no_trade"
	if got != want {
		t.Fatalf("read line:\n got %q\nwant %q", got, want)
	}
	// No reject history, no born-check (a fail-closed read records none): n/a.
	if got := plannerReadLine("ASIA", 3, nil, nil, nil, "no_trade"); got !=
		"🧭 planner read: session=ASIA attempts=3 reject_classes=none read→publish=n/a lifecycle=no_trade" {
		t.Fatalf("fail-closed line: %q", got)
	}
}
