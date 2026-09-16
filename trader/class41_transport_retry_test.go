package trader

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"

	"nofx/kernel"
)

// CLASS 41 M0 pin (transport-resets dispatch, 2026-09-02): a TRANSPORT or
// DEADLINE failure of the planner call re-sends the IDENTICAL prompt with NO
// reject block. Only validator/parse rejects append the PREVIOUS ATTEMPT block.
// On the pre-fix code attempt 2 re-authored with the transport error text as
// its "validator reason" (2026-09-01 23:47:33 CT) — this test MUST FAIL there.
func TestClass41TransportFailureResendsIdenticalPrompt(t *testing.T) {
	at := plannerTestTrader(t)
	facts := kernel.PlanFacts{Price: 29614, DATR: 300}
	machine := map[float64]string{29500: "PDL", 29550: "RN 29550", 29600: "RN 29600", 30000: "RN 30000", 30100: "RN 30100", 30200: "RN 30200"}
	var prompts []string
	transportErr := fmt.Errorf("still failed after 2 retries: %w", errors.New("stream interrupted: unexpected EOF"))
	_, lc, err := at.runPlannerReadCoreWithFactsGrades("ASIA", "2026-09-02", "scheduled",
		"deepseek-v4-pro", "hashT", "", "", "", "FULLPROMPT", facts, nil, machine, nil, true,
		func(userPrompt string) (string, error) {
			prompts = append(prompts, userPrompt)
			if len(prompts) == 1 {
				return "", transportErr
			}
			return thinAbovePlanJSON, nil
		})
	if err != nil || lc != "active" {
		t.Fatalf("attempt 2 must land the plan: lc=%q err=%v", lc, err)
	}
	if len(prompts) != 2 {
		t.Fatalf("want 2 calls, got %d", len(prompts))
	}
	h := func(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }
	if h(prompts[0]) != h(prompts[1]) {
		t.Fatalf("transport failure must re-send the IDENTICAL prompt: attempt1 hash %s != attempt2 hash %s\nattempt2 tail: %q", h(prompts[0])[:12], h(prompts[1])[:12], tail(prompts[1], 200))
	}
	// Both shapes: the legacy tail and the class-45 top/tail corrections. A
	// transport failure is not a rejection and must teach the model nothing.
	for _, marker := range []string{"PREVIOUS ATTEMPT REJECTED", "CORRECTIONS FROM THIS READ"} {
		if strings.Contains(prompts[1], marker) {
			t.Fatalf("no reject block on a transport failure, found %q", marker)
		}
	}
}

// Validator rejects keep the class-34 append (unchanged behaviour, pinned so
// M0 cannot over-reach).
func TestClass41ValidatorRejectStillAppendsBlock(t *testing.T) {
	at := plannerTestTrader(t)
	facts := kernel.PlanFacts{Price: 29614, DATR: 300}
	machine := map[float64]string{29500: "PDL", 29550: "RN 29550", 29600: "RN 29600", 30000: "RN 30000", 30100: "RN 30100", 30200: "RN 30200"}
	var prompts []string
	_, _, _ = at.runPlannerReadCoreWithFactsGrades("ASIA", "2026-09-02", "scheduled",
		"deepseek-v4-pro", "hashV", "", "", "", "FULLPROMPT", facts, nil, machine, nil, true,
		func(userPrompt string) (string, error) {
			prompts = append(prompts, userPrompt)
			if len(prompts) == 1 {
				return "not json at all", nil
			}
			return thinAbovePlanJSON, nil
		})
	if len(prompts) < 2 {
		t.Fatalf("want ≥2 calls, got %d", len(prompts))
	}
	if prompts[1] == prompts[0] {
		t.Fatalf("a parse/validator reject must NOT re-send the identical prompt (repair or reject block expected)")
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
