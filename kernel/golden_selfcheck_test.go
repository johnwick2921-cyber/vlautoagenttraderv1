package kernel

import "testing"

// P1 — the boot self-check must AGREE with the golden tests. If this fails, either
// the fixtures drifted apart or a prompt builder changed: both are real findings.
func TestVerifyPromptGoldensPasses(t *testing.T) {
	results, ok := VerifyPromptGoldens()
	for _, r := range results {
		t.Logf("  %-18s ok=%v hash=%s %s", r.Name, r.OK, r.GotHash, r.Detail)
	}
	if !ok {
		t.Fatal("boot goldens self-check FAILED — the embedded contract and the renderers disagree")
	}
}
