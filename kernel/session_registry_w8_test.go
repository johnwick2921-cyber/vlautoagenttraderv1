package kernel

import "testing"

// W8 — ValidateSessionRegistry REFUSES an edit that would break the gates (unlike
// LoadSessionRegistry, which fail-safes to the default on bad input).
func TestValidateSessionRegistry(t *testing.T) {
	if err := ValidateSessionRegistry(DefaultSessionRegistry()); err != nil {
		t.Fatalf("the shipped default must validate, got %v", err)
	}
	if err := ValidateSessionRegistry(SessionRegistry{}); err == nil {
		t.Fatal("empty registry (no sessions) must be refused")
	}
	// missing name.
	bad := SessionRegistry{Sessions: []SessionDef{{WindowStartCT: "08:30", WindowEndCT: "15:00", ReadCT: "08:25", FlatCT: "14:45"}}}
	if err := ValidateSessionRegistry(bad); err == nil {
		t.Fatal("a session with no name must be refused")
	}
	// non-HH:MM anchor (hour out of range).
	badTime := SessionRegistry{Sessions: []SessionDef{{Name: "NY", WindowStartCT: "25:00", WindowEndCT: "15:00", ReadCT: "08:25", FlatCT: "14:45"}}}
	if err := ValidateSessionRegistry(badTime); err == nil {
		t.Fatal("a non-HH:MM window anchor must be refused")
	}
	// malformed killzone bound.
	badKZ := SessionRegistry{Sessions: []SessionDef{{
		Name: "NY", WindowStartCT: "08:30", WindowEndCT: "15:00", ReadCT: "08:25", FlatCT: "14:45",
		Killzones: []KillzoneCT{{Name: "am", StartCT: "08:30", EndCT: "nope"}},
	}}}
	if err := ValidateSessionRegistry(badKZ); err == nil {
		t.Fatal("a malformed killzone bound must be refused")
	}
}
