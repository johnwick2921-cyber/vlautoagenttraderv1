package trader

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// ── W-EXEC-TRUTH W0b — the Guide and the gate-blocks panel do not lie about
// the binary (L5) ───────────────────────────────────────────────────────────
//
// The one admission chain counts each gate under its own name on every entry
// path; the panel must label every one. The guide must carry Picture's strict
// refusal VERBATIM (the card and the 📷 line print the constant) and name the
// CLI flag the withdraw is written with.
func TestW0bGuideAndGateLabelsMatchTheBinary(t *testing.T) {
	var src strings.Builder
	for _, f := range []string{"entry_admission.go", "arm_admission.go", "reconcile_owned.go", "session_risk.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	counted := map[string]bool{}
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`admitRefuse\(in, "([a-z_]+)"`),
		regexp.MustCompile(`IncGateBlock\(at\.id, "([a-z_]+)"`),
		regexp.MustCompile(`Class: "([a-z_]+)"`),
	} {
		for _, m := range re.FindAllStringSubmatch(src.String(), -1) {
			counted[m[1]] = true
		}
	}
	if len(counted) < 15 {
		t.Fatalf("fixture: expected the chain's gate classes, found %v", counted)
	}
	panel, err := os.ReadFile("../web/src/components/plan/GateBlocksPanel.tsx")
	if err != nil {
		t.Fatal(err)
	}
	for gate := range counted {
		if !strings.Contains(string(panel), "\n  "+gate+": {") {
			t.Errorf("GateBlocksPanel has no human label for gate %q", gate)
		}
	}
	guide, err := os.ReadFile("../web/src/guide/content/status.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{PictureStrictRefusal, "--withdraw-entries", "arm_not_admitted", "reconcile_owned", "plan_gate="} {
		if !strings.Contains(string(guide), want) {
			t.Errorf("the guide (status.ts) does not carry %q", want)
		}
	}
}
