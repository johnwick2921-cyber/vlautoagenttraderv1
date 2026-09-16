package trader

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// PINS (wiring wave 2026-09-05) for the two mechanisms that shipped unwired.

// --- 1. The daily-guardrail leg on BOTH order paths ---------------------
//
// Before this wave entry_gate.go contained neither "daily" nor "guardrail",
// so a resting arm filled straight through a tripped daily loss limit. These
// fail if the leg is removed or stops refusing.

func TestEntryGateRefusesWhenDailyForceFlatTripped(t *testing.T) {
	for _, path := range []string{"arm", "decision"} {
		reason, refused := EntryGate(EntryIntent{
			Path:           path,
			Action:         "open_long",
			DailyForceFlat: func() string { return "daily loss limit hit (realized today=-492.00, limit=-450.00)" },
		})
		if !refused {
			t.Fatalf("%s path: a tripped daily loss limit MUST refuse the entry; gate allowed it", path)
		}
		if !strings.Contains(reason, "daily_force_flat") {
			t.Errorf("%s path: refusal must name the daily_force_flat class, got %q", path, reason)
		}
		if !strings.Contains(reason, "limit=-450.00") {
			t.Errorf("%s path: refusal must carry the trip reason, got %q", path, reason)
		}
	}
}

// Fail-open contract: no evidence of a trip is never a refusal.
func TestEntryGateDailyLegFailsOpen(t *testing.T) {
	if _, refused := EntryGate(EntryIntent{Path: "arm", Action: "open_long"}); refused {
		t.Error("nil DailyForceFlat resolver must leave the leg off (fail-open), gate refused")
	}
	if _, refused := EntryGate(EntryIntent{
		Path: "arm", Action: "open_long",
		DailyForceFlat: func() string { return "" },
	}); refused {
		t.Error("empty trip reason is not evidence of a trip; gate must not refuse")
	}
	if _, refused := EntryGate(EntryIntent{
		Path: "arm", Action: "open_long",
		DailyForceFlat: func() string { return "   " },
	}); refused {
		t.Error("blank trip reason is not evidence of a trip; gate must not refuse")
	}
}

// THE PIN THAT FAILS WHEN THE CALL IS REMOVED — both adapters must populate
// the resolver, or the leg is dead on that path.
func TestBothEntryGateAdaptersPopulateDailyForceFlat(t *testing.T) {
	src, err := os.ReadFile("entry_gate.go")
	if err != nil {
		t.Fatalf("read entry_gate.go: %v", err)
	}
	// Whitespace-insensitive: gofmt aligns the two struct literals differently.
	re := regexp.MustCompile(`DailyForceFlat:\s+func\(\) string \{ return kernel\.DailyForceFlatReason\(at\.id\) \}`)
	if n := len(re.FindAllString(string(src), -1)); n != 2 {
		t.Errorf("both EntryIntent adapters (arm + decision) must populate DailyForceFlat; found %d of 2. "+
			"An unpopulated path silently reopens the hole this wave closed.", n)
	}
}

// --- 2. BiasArmWarning at the plan-write path ---------------------------
//
// BiasArmWarning shipped 2026-09-04 with ZERO production callers: written,
// tested, never called. This fails if it goes back to being uncalled.
func TestPlanWritePathCallsBiasArmWarning(t *testing.T) {
	src, err := os.ReadFile("auto_trader_planner.go")
	if err != nil {
		t.Fatalf("read auto_trader_planner.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "kernel.BiasArmWarning(d,") {
		t.Fatal("the plan-write path must call kernel.BiasArmWarning — without a caller it is " +
			"dead code and a plan whose bias direction carries no armed scenario ships silently")
	}
	if !strings.Contains(s, "bias-coherent arms:") {
		t.Error("the BiasArmWarning result must be surfaced in a log line the owner can grep")
	}
	// Owner ruling 2026-09-04 is WARN-first: the write must still proceed.
	if strings.Contains(s, "kernel.BiasArmWarning") && strings.Contains(s, "return fmt.Errorf(\"bias") {
		t.Error("BiasArmWarning must stay WARN-first per the owner ruling of 2026-09-04; it must not reject the write")
	}
}
