package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"vl/store"
)

// The USDT-denominated evidence regexes are gone (P-R crypto residue). This
// pins the generic min-size path: an NT-style error string — no USDT — still
// lands the hasAmountTooSmall diagnosis with the generic wording, and no USDT
// text leaks into the answer.
func TestTraderDiagnosisGenericMinSizePathAnswersForNTErrorString(t *testing.T) {
	ev := traderDiagnosisEvidence{
		TraderName:   "t",
		TraderConfig: &store.Trader{ID: "t1", Name: "t", AIModelID: "m1", ExchangeID: "e1", IsRunning: true},
		Model:        &safeModelToolConfig{ID: "m1", Enabled: true},
		Exchange:     &safeExchangeToolConfig{ID: "e1", Enabled: true},
		Account:      map[string]any{"total_equity": 100000.0},
	}
	if err := json.Unmarshal([]byte(`{"records":[{"success":false,
		"error_message":"order rejected: opening amount too small (123.45), must be ≥ 60.00",
		"execution_log":["opening amount too small (123.45), must be ≥ 60.00"],
		"candidate_coins":[],"decision_json":"","decisions":[]}]}`), &ev.Decisions); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	got := formatTraderDiagnosisEvidence("zh", ev)
	if !strings.Contains(got, "低于系统最小下单要求") || !strings.Contains(got, "被拦下") {
		t.Fatalf("generic min-size path must still answer for an NT-style (no USDT) error, got: %s", got)
	}
	if strings.Contains(got, "USDT") {
		t.Fatalf("no USDT text should leak into the diagnosis, got: %s", got)
	}
}
