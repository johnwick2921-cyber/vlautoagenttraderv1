package trader

import (
	"testing"

	"nofx/kernel"
)

func TestSessionGateDecision(t *testing.T) {
	reg := kernel.DefaultSessionRegistry() // NY enabled; ASIA/LONDON disabled

	cases := []struct {
		name       string
		h, m       int
		wantBlock  bool
		reasonHint string
	}{
		{"NY midday allowed", 9, 30, false, ""},
		{"NY first-5m blocked", 8, 32, true, "first-5m"},
		{"NY lunch blocked", 12, 30, true, "lunch"},
		{"NY post-lunch allowed", 13, 45, false, ""},
		{"Asia not enabled", 18, 0, true, "not enabled"},
		{"London not enabled", 3, 0, true, "not enabled"},
		{"overnight gap blocked", 16, 0, true, "outside"},
	}
	for _, c := range cases {
		reason, blocked := sessionGateDecision(reg, ctAt(t, c.h, c.m), nil, nil)
		if blocked != c.wantBlock {
			t.Fatalf("%s: blocked=%v want %v (reason %q)", c.name, blocked, c.wantBlock, reason)
		}
		if c.wantBlock && c.reasonHint != "" && !contains(reason, c.reasonHint) {
			t.Fatalf("%s: reason %q missing %q", c.name, reason, c.reasonHint)
		}
	}
}

func TestSessionEntryBlockedDormant(t *testing.T) {
	// day_plan off → the gate is dormant regardless of the clock.
	off := mkTrader("ninjatrader", nil, "5m")
	if _, blocked := off.sessionEntryBlocked(); blocked {
		t.Fatalf("plan-off trader must not engage the session gate")
	}
	crypto := mkTrader("binance", boolp(true), "5m")
	if _, blocked := crypto.sessionEntryBlocked(); blocked {
		t.Fatalf("crypto trader must never engage the session gate")
	}
}

func boolp(b bool) *bool { return &b }

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
