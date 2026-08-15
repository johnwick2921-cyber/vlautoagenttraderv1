package kernel

import (
	"encoding/json"
	"fmt"
	"strings"
)

// P5.4 — Ask-Planner: a plan-scoped Q&A with the reasoner that authored the plan.
// The anti-sycophancy contract (spec §44) is embedded VERBATIM in the system
// prompt AND enforced in code (ParsePlannerReply) so the model cannot flatter its
// way to a patch: a bare disagreement NEVER produces one.

// AskPlannerSystemPrompt is the verbatim anti-sycophancy contract. The reply MUST
// be the three-part structure; a patch is allowed ONLY on genuinely new
// information; nothing changes without the owner's explicit Apply.
const AskPlannerSystemPrompt = `You are the PLANNER that wrote today's trading day-plan. The owner is questioning it. You are NOT a sycophant: you do not fold to pressure, and you do not flatter. Answer with this exact structure, and be honest even when it means disagreeing with the owner.

CONTRACT (do not deviate):
- EVIDENCE — restate what you based the plan element on: its provenance, its score/grade, and the live facts (with numbers). No hedging.
- OWNER'S POINT — classify the owner's message as exactly one of:
    NEW-INFO           = they gave evidence you did not weight (a higher-timeframe level, a fresh structural read, a fact).
    BARE-DISAGREEMENT  = a confidence check or opinion with NO new evidence ("are you sure?", "I think it's wrong").
- VERDICT — exactly one of:
    DEFEND         = you hold your read; say where you would be wrong (the flip/invalidation line).
    CONCEDE        = the new info changes the picture; explain what and why.
    PROPOSE-MERGE  = the new info warrants a surgical edit; include a patch.

HARD RULES:
- A BARE-DISAGREEMENT can ONLY be DEFEND. It NEVER produces a patch. This is the anti-sycophancy rule — obey it.
- A patch is a SURGICAL RFC-6902 change to the plan (one or a few ops), never a full re-plan. Only on NEW-INFO.
- Nothing you say changes the plan; the owner applies the patch or not. Do not claim you changed anything.

OUTPUT: a single JSON object, nothing else:
{"evidence":"<text, with numbers>","point_class":"NEW-INFO|BARE-DISAGREEMENT","verdict":"DEFEND|CONCEDE|PROPOSE-MERGE","summary":"<one line>","patch":[<RFC-6902 ops>]}
Omit "patch" (or use []) unless verdict is PROPOSE-MERGE. Answer in the owner's language for the prose fields; keep JSON keys and controlled tokens in English.`

// PlannerReply is the structured Ask-Planner answer.
type PlannerReply struct {
	Evidence   string          `json:"evidence"`
	PointClass string          `json:"point_class"` // NEW-INFO | BARE-DISAGREEMENT
	Verdict    string          `json:"verdict"`     // DEFEND | CONCEDE | PROPOSE-MERGE
	Summary    string          `json:"summary"`
	Patch      json.RawMessage `json:"patch,omitempty"` // RFC-6902 ops, only when PROPOSE-MERGE
}

var validPointClass = map[string]bool{"NEW-INFO": true, "BARE-DISAGREEMENT": true}
var validVerdict = map[string]bool{"DEFEND": true, "CONCEDE": true, "PROPOSE-MERGE": true}

// BuildAskPlannerUserPrompt assembles the plan + live facts + the owner question.
func BuildAskPlannerUserPrompt(planBlock, liveStatus, question string) string {
	var sb strings.Builder
	sb.WriteString("TODAY'S PLAN:\n")
	sb.WriteString(planBlock)
	if strings.TrimSpace(liveStatus) != "" {
		sb.WriteString("\n\nLIVE STATUS (facts, not judgment):\n")
		sb.WriteString(liveStatus)
	}
	sb.WriteString("\n\nOWNER ASKS:\n")
	sb.WriteString(strings.TrimSpace(question))
	return sb.String()
}

// ParsePlannerReply extracts + validates the structured reply AND enforces the
// anti-sycophancy contract in CODE (belt-and-braces over the prompt):
//   - unknown enums are rejected;
//   - a BARE-DISAGREEMENT is forced to DEFEND and stripped of any patch;
//   - a PROPOSE-MERGE with no patch is downgraded to DEFEND (nothing to apply);
//   - a patch present on anything but PROPOSE-MERGE is dropped.
func ParsePlannerReply(raw string) (*PlannerReply, error) {
	js := extractJSONObject(raw)
	if js == "" {
		return nil, fmt.Errorf("no JSON object in planner reply")
	}
	var r PlannerReply
	if err := json.Unmarshal([]byte(js), &r); err != nil {
		return nil, fmt.Errorf("planner reply unmarshal: %w", err)
	}
	r.PointClass = strings.ToUpper(strings.TrimSpace(r.PointClass))
	r.Verdict = strings.ToUpper(strings.TrimSpace(r.Verdict))
	if !validPointClass[r.PointClass] {
		return nil, fmt.Errorf("invalid point_class %q", r.PointClass)
	}
	if !validVerdict[r.Verdict] {
		return nil, fmt.Errorf("invalid verdict %q", r.Verdict)
	}

	// THE ANTI-SYCOPHANCY ENFORCEMENT (code, not just prompt):
	// a bare disagreement can only DEFEND and never carries a patch.
	if r.PointClass == "BARE-DISAGREEMENT" {
		r.Verdict = "DEFEND"
		r.Patch = nil
	}
	// a patch only rides a PROPOSE-MERGE; empty/[] patch → downgrade to DEFEND.
	if r.Verdict != "PROPOSE-MERGE" {
		r.Patch = nil
	} else if !hasPatchOps(r.Patch) {
		r.Verdict = "DEFEND"
		r.Patch = nil
	}
	return &r, nil
}

// hasPatchOps reports whether the raw patch is a non-empty JSON op array.
func hasPatchOps(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var ops []PatchOp
	if json.Unmarshal(raw, &ops) != nil {
		return false
	}
	return len(ops) > 0
}
