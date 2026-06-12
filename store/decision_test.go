// Plan 4 Task 23 — decision audit trail field tests.

package store

import (
	"encoding/json"
	"testing"
	"time"
)

// Tests that the new Plan 4 T23 fields are present on DecisionRecord and round-trip via JSON.
func TestDecisionRecordAuditFieldsJSON(t *testing.T) {
	fillPrice := 21503.25
	fillLatencyMs := int64(842)
	rec := &DecisionRecord{
		ID:              7,
		TraderID:        "trader-1",
		PromptVersion:   "v1-abc123",
		AIModel:         "claude-opus-4-7",
		AILatencyMs:     1500,
		RiskCheckPassed: true,
		RiskCheckError:  "",
		ExecutionStatus: "filled",
		FillPrice:       &fillPrice,
		FillLatencyMs:   &fillLatencyMs,
		CreatedAt:       time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
	}

	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{
		"prompt_version",
		"ai_model",
		"ai_latency_ms",
		"risk_check_passed",
		"risk_check_error",
		"execution_status",
		"fill_price",
		"fill_latency_ms",
		"created_at",
	} {
		if _, ok := asMap[key]; !ok {
			t.Errorf("decision record JSON missing key %q in %s", key, string(raw))
		}
	}
}

// Tests that toRecord() copies all new audit fields from the DB model.
func TestDecisionRecordDBToRecordCopiesAuditFields(t *testing.T) {
	fillPrice := 100.0
	fillLatencyMs := int64(50)
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	db := &DecisionRecordDB{
		ID:              99,
		TraderID:        "t1",
		Timestamp:       now,
		PromptVersion:   "hash-xyz",
		AIModel:         "deepseek",
		AILatencyMs:     1234,
		RiskCheckPassed: true,
		RiskCheckError:  "",
		ExecutionStatus: "filled",
		FillPrice:       &fillPrice,
		FillLatencyMs:   &fillLatencyMs,
		CreatedAt:       now,
		CandidateCoins:  "[]",
		ExecutionLog:    "[]",
		Decisions:       "[]",
	}
	rec := db.toRecord()
	if rec.PromptVersion != "hash-xyz" {
		t.Errorf("PromptVersion: want hash-xyz, got %q", rec.PromptVersion)
	}
	if rec.AIModel != "deepseek" {
		t.Errorf("AIModel: want deepseek, got %q", rec.AIModel)
	}
	if rec.AILatencyMs != 1234 {
		t.Errorf("AILatencyMs: want 1234, got %d", rec.AILatencyMs)
	}
	if !rec.RiskCheckPassed {
		t.Error("RiskCheckPassed: want true")
	}
	if rec.ExecutionStatus != "filled" {
		t.Errorf("ExecutionStatus: want filled, got %q", rec.ExecutionStatus)
	}
	if rec.FillPrice == nil || *rec.FillPrice != 100.0 {
		t.Errorf("FillPrice: want 100.0, got %v", rec.FillPrice)
	}
	if rec.FillLatencyMs == nil || *rec.FillLatencyMs != 50 {
		t.Errorf("FillLatencyMs: want 50, got %v", rec.FillLatencyMs)
	}
	if !rec.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt: want %v, got %v", now, rec.CreatedAt)
	}
}
