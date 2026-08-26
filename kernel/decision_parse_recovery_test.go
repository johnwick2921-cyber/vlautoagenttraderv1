package kernel

import (
	"strings"
	"testing"
)

// P0 — the decision extractor must recover a decision object embedded in prose,
// and must return an ERROR (not a silent safe-wait) when the model emitted only
// prose with no JSON at all, so callWithSchemaRetry re-asks for the JSON envelope.
// Before this, a fully-reasoned setup (prose + truncated/missing JSON) was
// silently swallowed into a `wait`, with no retry and no record of the loss.

func TestExtractDecisions_ProsePlusSingleObject(t *testing.T) {
	// The model reasons fully, then emits ONE decision object (not the array
	// envelope) without fences.
	resp := "5m/15m EMAs aligned bullish. Planned long:\n" +
		`{"symbol": "MNQ", "action": "open_long", "leverage": 1, "position_size_usd": 60000, "stop_loss": 21480.00, "take_profit": 21560.00, "confidence": 80}`
	decisions, err := extractDecisions(resp)
	if err != nil {
		t.Fatalf("single-object recovery failed: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Action != "open_long" || decisions[0].Symbol != "MNQ" {
		t.Fatalf("recovered decision wrong: %+v", decisions)
	}
	if decisions[0].StopLoss != 21480.00 || decisions[0].TakeProfit != 21560.00 {
		t.Errorf("stop/tp = %v/%v, want 21480/21560", decisions[0].StopLoss, decisions[0].TakeProfit)
	}
}

func TestExtractDecisions_ProseOnlyErrors(t *testing.T) {
	// Prose with a reasoned setup but NO JSON anywhere. This must be an ERROR so
	// the caller retries, not a silent wait.
	resp := "Planned long: entry 30256, stop 30236.75, target 30313.75, R/R 3.00. No JSON emitted."
	decisions, err := extractDecisions(resp)
	if err == nil {
		t.Fatalf("prose-only response must return an error (so the retry fires), got decisions %+v", decisions)
	}
	if !strings.Contains(err.Error(), "no decision JSON") {
		t.Fatalf("error must name the missing JSON, got %v", err)
	}
}

func TestExtractDecisions_FencedAndUnfencedArrays(t *testing.T) {
	fenced := "<reasoning>bullish</reasoning>\n<decision>\n```json\n[{\"symbol\": \"MNQ\", \"action\": \"wait\"}]\n```\n</decision>"
	unfenced := "<reasoning>bearish</reasoning>\n<decision>\n[{\"symbol\": \"MNQ\", \"action\": \"close_long\"}]\n</decision>"

	for name, resp := range map[string]string{"fenced": fenced, "unfenced": unfenced} {
		decisions, err := extractDecisions(resp)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(decisions) != 1 {
			t.Fatalf("%s: got %d decisions, want 1", name, len(decisions))
		}
	}
}

func TestExtractDecisions_TruncatedJSONErrors(t *testing.T) {
	// The response was cut before the JSON array closed. Recovery must not invent
	// a decision; it must error so the retry asks for the JSON only.
	resp := "Long setup.\n<decision>\n[{\"symbol\": \"MNQ\", \"action\": \"open_long\", \"stop_loss\": 21480"
	decisions, err := extractDecisions(resp)
	if err == nil {
		t.Fatalf("truncated JSON must error, got %+v", decisions)
	}
}