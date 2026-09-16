package trader

import (
	"fmt"
	"strings"
	"testing"

	"nofx/kernel"
)

// CLASS 34 (owner ruling 2026-08-31) — reproduction of tonight: the
// breakdown-void reject + the reject block must carry only legal, live
// condition names. The model complied with "author a reject/retest play" and
// was punished with parse/schema rejects on "reject_retest" — twice, in both
// ASIA chains, both fail-closed.
func TestClass34RejectBlockNamesOnlyLiveConditions(t *testing.T) {
	err := fmt.Errorf("S3 breakdown_continue: a close came back across 29517.00 — the breakdown is void; %s", kernel.BreakdownReclaimedHint)
	live := kernel.ResolvedLiveConditions(nil, nil, "")
	block := plannerRejectBlock(err, live)

	// 1. the verbatim reject reason survives (retry-append contract).
	if !strings.Contains(block, "breakdown is void") {
		t.Fatalf("reject block lost the verbatim reason: %q", block)
	}
	// 2. the hint names the legal token and never the composite instruction.
	if !strings.Contains(block, "`reject` play") || strings.Contains(block, "author a reject/retest") {
		t.Fatalf("reject block hint wrong: %q", block)
	}
	// 3. the valid-conditions suffix is present.
	open := strings.Index(block, "Valid conditions: [")
	if open < 0 {
		t.Fatalf("reject block missing the valid-conditions suffix: %q", block)
	}
	close := strings.Index(block[open:], "]")
	if close < 0 {
		t.Fatalf("unterminated valid-conditions list: %q", block)
	}
	list := block[open+len("Valid conditions: [") : open+close]
	known := map[string]bool{}
	for _, c := range kernel.KnownConditions() {
		known[c] = true
	}
	resolved := kernel.ResolvedConditionStatuses(nil, nil, "")
	for _, tok := range strings.Split(list, ", ") {
		if !known[tok] {
			t.Errorf("suffix names unknown condition %q", tok)
		}
		if resolved[tok] == kernel.ConditionShadow {
			t.Errorf("suffix names shadowed condition %q", tok)
		}
	}
	// 4. the shadowed conditions must not appear as valid vocabulary.
	for _, shadowed := range []string{"breakout_retest", "fvg_entry"} {
		if strings.Contains(list, shadowed) {
			t.Errorf("valid-conditions list must exclude shadowed %q: %q", shadowed, list)
		}
	}
}

// The repair-prompt excerpt path uses the same registry constants.
func TestClass34RepairLawsNameOnlyLegalLiveConditions(t *testing.T) {
	for _, law := range []string{kernel.RepairBreakdownLaw, kernel.RepairArmSplitLaw, kernel.RepairEntryConfirmLaw} {
		if strings.Contains(law, "reject/retest") {
			t.Errorf("repair law still instructs the composite: %q", law)
		}
	}
	if err := kernel.ValidateValidatorHints(); err != nil {
		t.Fatalf("ValidateValidatorHints: %v", err)
	}
}

// CLASS 45 E4 (2026-09-02): class 34's canon is that the validator's verbatim
// reason and its legal-token hint REACH the model. Since 45 the live re-author
// carries them through the cumulative header/tail, not through the legacy
// single-defect tail above — so the canon is pinned on the path actually used.
func TestClass34HintSurvivesTheClass45Blocks(t *testing.T) {
	err := fmt.Errorf("S3 breakdown_continue: a close came back across 29517.00 — the breakdown is void; %s", kernel.BreakdownReclaimedHint)
	live := kernel.ResolvedLiveConditions(nil, nil, "")
	h := addDistinctReject(nil, err)

	for name, block := range map[string]string{"header": plannerRejectHeader(h, live), "tail": plannerRejectTail(h, live)} {
		if !strings.Contains(block, "breakdown is void") {
			t.Errorf("%s lost the verbatim reason: %q", name, block)
		}
		if !strings.Contains(block, "`reject` play") || strings.Contains(block, "author a reject/retest") {
			t.Errorf("%s lost the legal-token hint: %q", name, block)
		}
		if !strings.Contains(block, "Valid conditions: [") {
			t.Errorf("%s lost the valid-conditions suffix: %q", name, block)
		}
	}
}
