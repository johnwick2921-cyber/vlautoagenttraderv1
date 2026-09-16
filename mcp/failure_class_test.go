package mcp

import (
	"errors"
	"fmt"
	"testing"
)

// E2 — the re-classification table. Each row is a REAL failure shape from the
// 50-failure DeepSeek audit, with the label the old two-system scheme produced
// and the label the single classifier produces. The old `timeout_source`
// defaulted to "transport" and was overridden for four sentinels only.
func TestClass46ReclassificationTable(t *testing.T) {
	type row struct {
		name         string
		err          error
		status       int
		oldSource    string // what timeout_source printed
		want         FailureClass
		providerSide bool
	}
	rows := []row{
		{"peer FIN mid-body", errors.New("stream interrupted: unexpected EOF"), 200, "transport", ClassTransport, true},
		{"peer RST", errors.New("stream interrupted: read tcp 1.2.3.4:1->5.6.7.8:443: read: connection reset by peer"), 200, "transport", ClassTransport, true},
		{"503 overloaded", errors.New(`API error (status 503): {"error":{"message":"Server Overloaded"}}`), 503, "transport", ClassHTTP5xx, true},
		{"502 bad gateway", errors.New("API returned error (status 502): bad gateway"), 502, "transport", ClassHTTP5xx, true},
		{"429 rate limited", errors.New("API error (status 429): rate limited"), 429, "transport", ClassHTTP5xx, true},
		{"400 bad request", errors.New("API error (status 400): invalid request"), 400, "transport", ClassHTTP4xx, false},
		{"401 unauthorized", errors.New("API error (status 401): unauthorized"), 401, "transport", ClassHTTP4xx, false},
		// parse is MODEL-side: it wrote a document that will not parse, and the
		// repair path exists for that. Only an ABSENT body (empty_200) is
		// provider-side.
		{"parse: no JSON", errors.New("no JSON object found in planner output"), 200, "transport", ClassParse, false},
		{"parse: type error", errors.New("plan JSON unmarshal: json: cannot unmarshal number 0.5 into Go struct field PlanArmLeg.size of type int"), 200, "transport", ClassParse, false},
		{"empty 200", errors.New("stream produced no result"), 200, "transport", ClassEmpty200, true},
		{"too long", fmt.Errorf("wrapped: %w", ErrTooLong), 200, "transport", ClassTooLong, true},
		{"planner total ceiling", fmt.Errorf("x: %w", ErrStreamTotalDeadline), 200, "planner_total", ClassTotalDeadline, true},
		{"watchdog idle", fmt.Errorf("x: %w", ErrWatchdogIdle), 200, "transport", ClassIdle, true},
		{"legacy idle sentinel", fmt.Errorf("x: %w", ErrStreamIdleDeadline), 200, "stream_idle", ClassIdle, true},
		{"client ceiling", errors.New("failed to read response: context deadline exceeded (Client.Timeout or context cancellation while reading body)"), 0, "client", ClassTotalDeadline, true},
		{"no api key", errors.New("AI API key not set"), 0, "transport", ClassAuthConfig, false},
		{"caller cancelled", errors.New("context canceled"), 0, "context", ClassContext, true},
	}
	mislabelled := 0
	t.Logf("%-24s %-14s → %-16s provider_side", "failure", "old source", "class 46")
	for _, r := range rows {
		got := ClassifyFailure(r.err, r.status)
		if got != r.want {
			t.Errorf("%s: got %q want %q", r.name, got, r.want)
		}
		if ps := FailureIsProviderSide(got); ps != r.providerSide {
			t.Errorf("%s: provider_side=%v want %v", r.name, ps, r.providerSide)
		}
		if r.oldSource == "transport" && r.want != ClassTransport {
			mislabelled++
		}
		t.Logf("%-24s %-14s → %-16s %v", r.name, r.oldSource, got, FailureIsProviderSide(got))
	}
	if mislabelled == 0 {
		t.Fatal("the fixture must contain rows the old default mislabelled")
	}
	t.Logf("rows the old timeout_source default tagged 'transport' but are NOT transport: %d of %d", mislabelled, len(rows))
}

// A validator reject is the ONLY class the model can act on.
func TestClass46OnlyValidatorIsModelSide(t *testing.T) {
	if FailureIsProviderSide(ClassValidator) {
		t.Fatal("a validator reject is the model's to fix — it must NOT be provider-side")
	}
	for _, c := range []FailureClass{ClassTransport, ClassHTTP5xx, ClassTotalDeadline,
		ClassIdle, ClassEmpty200, ClassTooLong} {
		if !FailureIsProviderSide(c) {
			t.Fatalf("%s must be provider-side — feeding it back to the model is poisoned feedback", c)
		}
	}
	// 4xx is OUR request being wrong: not the model's defect, and not
	// retryable by resending the same thing either.
	if FailureIsProviderSide(ClassHTTP4xx) {
		t.Fatal("http_4xx is a malformed REQUEST, not a provider failure to resend through")
	}
	if FailureIsProviderSide(ClassParse) {
		t.Fatal("a parse failure of a document the model WROTE is model-side — resending would loop on the same malformed output")
	}
}

// The observed status outranks text sniffing: a 503 body whose text happens to
// mention EOF is still http_5xx.
func TestClass46StatusOutranksText(t *testing.T) {
	e := errors.New("API error (status 503): upstream EOF while reading")
	if got := ClassifyFailure(e, 503); got != ClassHTTP5xx {
		t.Fatalf("got %q want http_5xx — the status must win over the word EOF", got)
	}
	// With no status observed and no status in the text, the same words are transport.
	if got := ClassifyFailure(errors.New("unexpected EOF"), 0); got != ClassTransport {
		t.Fatalf("got %q want transport", got)
	}
}
