// A5 (G5) — prompt-ownership: the final assembly must skip a cycle whose context
// carries a field owned by another trader (cross-trader contamination), and pass
// (byte-identical) when every tag matches the deciding trader.
package kernel

import "testing"

func TestAssertPromptOwnership(t *testing.T) {
	// All tags match the deciding trader → no violation.
	ok := &Context{TraderID: "T1", OwnerTags: map[string]string{
		"account": "T1", "positions": "T1", "market_data": "T1",
	}}
	if _, _, violated := assertPromptOwnership(ok); violated {
		t.Fatal("a context whose tags all match the deciding trader must NOT violate")
	}

	// A field owned by ANOTHER trader → violation, and it names the offending field/owner.
	bad := &Context{TraderID: "T1", OwnerTags: map[string]string{
		"account": "T1", "positions": "T2", "market_data": "T1",
	}}
	field, badOwner, violated := assertPromptOwnership(bad)
	if !violated || field != "positions" || badOwner != "T2" {
		t.Fatalf("contamination must violate naming the field/owner; field=%q owner=%q violated=%v", field, badOwner, violated)
	}

	// Untagged context (empty OwnerTags) → no violation (legacy / goldens byte-identical).
	if _, _, violated := assertPromptOwnership(&Context{TraderID: "T1"}); violated {
		t.Fatal("an untagged context must not violate (byte-identical legacy path)")
	}
	// Empty deciding trader → no violation (nothing to assert against).
	if _, _, violated := assertPromptOwnership(&Context{OwnerTags: map[string]string{"account": "T9"}}); violated {
		t.Fatal("an empty TraderID must not violate")
	}
	// nil-safe.
	if _, _, violated := assertPromptOwnership(nil); violated {
		t.Fatal("nil context must not violate")
	}
}
