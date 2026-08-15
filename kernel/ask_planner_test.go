package kernel

import (
	"strings"
	"testing"
)

// P5.4 — the anti-sycophancy enforcement is the load-bearing rule; these lock it
// in CODE so a flattering model can't route a bare pushback into a patch.

func TestParsePlannerReplyProposeMerge(t *testing.T) {
	raw := `{"evidence":"1h supply, score B","point_class":"NEW-INFO","verdict":"PROPOSE-MERGE","summary":"HTF outranks","patch":[{"op":"replace","path":"/levels/0/grade","value":"A"}]}`
	r, err := ParsePlannerReply(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Verdict != "PROPOSE-MERGE" || !hasPatchOps(r.Patch) {
		t.Fatalf("PROPOSE-MERGE with patch must survive: %+v", r)
	}
}

func TestBareDisagreementNeverPatches(t *testing.T) {
	// even if the model tries to attach a patch to a bare pushback, code strips it
	// AND forces DEFEND.
	raw := `{"evidence":"bias long, above EMA200","point_class":"BARE-DISAGREEMENT","verdict":"PROPOSE-MERGE","summary":"x","patch":[{"op":"replace","path":"/bias/direction","value":"short"}]}`
	r, err := ParsePlannerReply(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Verdict != "DEFEND" {
		t.Fatalf("bare disagreement must be DEFEND, got %q", r.Verdict)
	}
	if hasPatchOps(r.Patch) {
		t.Fatal("bare disagreement must NEVER carry a patch")
	}
}

func TestProposeMergeWithoutPatchDowngrades(t *testing.T) {
	raw := `{"evidence":"e","point_class":"NEW-INFO","verdict":"PROPOSE-MERGE","summary":"s","patch":[]}`
	r, err := ParsePlannerReply(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Verdict != "DEFEND" {
		t.Fatalf("empty-patch PROPOSE-MERGE must downgrade to DEFEND, got %q", r.Verdict)
	}
}

func TestConcedeDropsPatch(t *testing.T) {
	raw := `{"evidence":"e","point_class":"NEW-INFO","verdict":"CONCEDE","summary":"s","patch":[{"op":"replace","path":"/x","value":1}]}`
	r, _ := ParsePlannerReply(raw)
	if hasPatchOps(r.Patch) {
		t.Fatal("only PROPOSE-MERGE carries a patch; CONCEDE must drop it")
	}
}

func TestInvalidEnumsRejected(t *testing.T) {
	if _, err := ParsePlannerReply(`{"point_class":"MAYBE","verdict":"DEFEND"}`); err == nil {
		t.Fatal("unknown point_class must be rejected")
	}
	if _, err := ParsePlannerReply(`{"point_class":"NEW-INFO","verdict":"AGREE"}`); err == nil {
		t.Fatal("unknown verdict must be rejected")
	}
}

// hardening — a PROPOSE-MERGE whose patch has an unsupported/empty op is not
// appliable, so it downgrades to DEFEND rather than exposing a dead Apply button.
func TestMislabeledProposeMergeDowngrades(t *testing.T) {
	for _, patch := range []string{
		`[{"op":"move","path":"/levels/0","from":"/levels/1"}]`, // unsupported op
		`[{"op":"","path":""}]`,                                 // empty op/path
		`[{"op":"replace","path":"/x"}]`,                        // replace with no value
	} {
		raw := `{"evidence":"e","point_class":"NEW-INFO","verdict":"PROPOSE-MERGE","summary":"s","patch":` + patch + `}`
		r, err := ParsePlannerReply(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if r.Verdict != "DEFEND" || hasPatchOps(r.Patch) {
			t.Fatalf("un-appliable patch must downgrade to DEFEND: patch=%s verdict=%s", patch, r.Verdict)
		}
	}
}

func TestParsePlannerReplyTolerantOfProse(t *testing.T) {
	raw := "Here is my reply:\n```json\n{\"evidence\":\"e\",\"point_class\":\"BARE-DISAGREEMENT\",\"verdict\":\"DEFEND\",\"summary\":\"holding\"}\n```\n"
	r, err := ParsePlannerReply(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.PointClass != "BARE-DISAGREEMENT" {
		t.Fatalf("got %+v", r)
	}
}

func TestBuildAskPlannerUserPrompt(t *testing.T) {
	p := BuildAskPlannerUserPrompt("PLAN BLOCK", "STATUS", "why supply here?")
	for _, want := range []string{"TODAY'S PLAN", "PLAN BLOCK", "LIVE STATUS", "STATUS", "OWNER ASKS", "why supply here?"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
}
