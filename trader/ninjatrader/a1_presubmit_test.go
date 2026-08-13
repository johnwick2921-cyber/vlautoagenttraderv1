// A1 (G3) — pre-submit identity invariant: an order's frame account must equal the
// emitting trader's bound account, else the submit is refused.
package ninjatrader

import "testing"

func TestAssertBoundAccount(t *testing.T) {
	cases := []struct {
		name         string
		frameAccount string
		boundAccount string
		wantErr      bool
	}{
		{"match", "Sim101", "Sim101", false},
		{"mismatch", "Sim102", "Sim101", true},   // the cross-account emission this blocks
		{"both empty legacy", "", "", false},      // account-less legacy frame on an unbound trader
		{"frame set, bound empty", "Sim101", "", true},
		{"frame empty, bound set", "", "Sim101", true},
		{"case-different is a mismatch", "sim101", "Sim101", true}, // exact-match; NT account names are case-exact
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := assertBoundAccount("entry", "MNQ", c.frameAccount, c.boundAccount)
			if (err != nil) != c.wantErr {
				t.Fatalf("assertBoundAccount(%q,%q) err=%v, wantErr=%v", c.frameAccount, c.boundAccount, err, c.wantErr)
			}
		})
	}
}
