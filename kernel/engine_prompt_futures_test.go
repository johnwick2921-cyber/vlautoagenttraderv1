package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

// futuresTestEngine builds a StrategyEngine with a minimal futures-flavoured
// config for the BuildFuturesDecisionSystemPrompt round-trip tests below.
func futuresTestEngine() *StrategyEngine {
	cfg := &store.StrategyConfig{
		Language: "en",
		RiskControl: store.RiskControlConfig{
			MinConfidence:      75,
			MinRiskRewardRatio: 1.5,
			MaxPositions:       1,
		},
	}
	return &StrategyEngine{config: cfg}
}

// TestFuturesDecisionPrompt_ParseableEnvelope is the core round-trip proof: a
// futures-prompt-style AI response (the envelope BuildFuturesDecisionSystemPrompt
// instructs) must parse cleanly through the SAME extractor the crypto path
// uses, into a valid Decision with the standard 6-action enum.
func TestFuturesDecisionPrompt_ParseableEnvelope(t *testing.T) {
	aiResponse := "<reasoning>\n5m and 15m EMAs aligned bullish, RSI 58, ATR ~20pt. Long setup.\n</reasoning>\n\n" +
		"<decision>\n```json\n[\n" +
		`  {"symbol": "MNQ", "action": "open_long", "leverage": 1, "position_size_usd": 60000, "stop_loss": 21480.00, "take_profit": 21560.00, "confidence": 80}` +
		"\n]\n```\n</decision>\n"

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		t.Fatalf("extractDecisions failed on futures envelope: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("got %d decisions, want 1", len(decisions))
	}
	d := decisions[0]
	if d.Symbol != "MNQ" || d.Action != "open_long" {
		t.Errorf("got symbol=%q action=%q, want MNQ/open_long", d.Symbol, d.Action)
	}
	if d.StopLoss != 21480.00 || d.TakeProfit != 21560.00 {
		t.Errorf("stop/tp = %v/%v, want 21480/21560", d.StopLoss, d.TakeProfit)
	}
	if d.Confidence != 80 {
		t.Errorf("confidence = %d, want 80", d.Confidence)
	}
}

// TestFuturesDecisionPrompt_WaitParses confirms the no-trade form (a single
// wait array) parses — the shape that replaces today's empty-parse "safe wait".
func TestFuturesDecisionPrompt_WaitParses(t *testing.T) {
	aiResponse := "<reasoning>No clean setup; chop.</reasoning>\n" +
		"<decision>\n```json\n[{\"symbol\": \"MNQ\", \"action\": \"wait\"}]\n```\n</decision>"
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		t.Fatalf("extractDecisions failed on wait: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Action != "wait" || decisions[0].Symbol != "MNQ" {
		t.Fatalf("wait decision parsed wrong: %+v", decisions)
	}
}

// TestFuturesDecisionPrompt_Content asserts the futures system prompt carries
// futures framing + the exact parser envelope, and does NOT frame MNQ as a
// crypto perp (the framing that confused the model into empty JSON).
func TestFuturesDecisionPrompt_Content(t *testing.T) {
	e := futuresTestEngine()
	p := e.BuildFuturesDecisionSystemPrompt("MNQ", 50000)

	for _, s := range []string{"MNQ", "$2", "0.25", "<reasoning>", "<decision>", "open_long | open_short", "JSON array"} {
		if !strings.Contains(p, s) {
			t.Errorf("futures system prompt missing %q", s)
		}
	}
	// Must not FRAME MNQ as a crypto perp. (Note: the prompt may MENTION
	// "funding rate" only to tell the model to ignore it, so that term is
	// allowed; USDT / BTC-ETH leverage tiers must be absent.)
	lower := strings.ToLower(p)
	for _, s := range []string{"usdt", "btc/eth"} {
		if strings.Contains(lower, s) {
			t.Errorf("futures system prompt unexpectedly contains crypto concept %q", s)
		}
	}
}

// TestFuturesVariant_Routing confirms BuildSystemPrompt routes "futures" to the
// futures prompt, while other variants still produce the crypto assembly.
func TestFuturesVariant_Routing(t *testing.T) {
	e := futuresTestEngine()
	if !strings.Contains(e.BuildSystemPrompt(50000, "futures", "MNQ"), "Micro E-mini Nasdaq-100") {
		t.Errorf("futures variant did not route to the futures prompt")
	}
	if strings.Contains(e.BuildSystemPrompt(50000, "balanced", "MNQ"), "Micro E-mini Nasdaq-100") {
		t.Errorf("balanced variant unexpectedly produced the futures prompt")
	}
}

// TestFuturesDecisionPrompt_SymbolParameterized (Phase 3) asserts the prompt
// reflects the ACTIVE symbol's real instrument + point value — not a hardwired
// MNQ. The micro≠mini divergence (P1) must flow through: NQ says $20, MNQ $2.
func TestFuturesDecisionPrompt_SymbolParameterized(t *testing.T) {
	e := futuresTestEngine()
	cases := []struct {
		symbol   string
		wantName string   // instrument description
		wantHave []string // substrings that MUST appear (point value etc.)
		wantGone []string // substrings that must NOT appear (no wrong instrument)
	}{
		{"MNQ", "Micro E-mini Nasdaq-100", []string{"$2.00 per index point", "Symbol: MNQ"}, nil},
		{"NQ", "E-mini Nasdaq-100", []string{"$20.00 per index point", "Symbol: NQ"}, []string{"Symbol: MNQ"}},
		{"ES", "E-mini S&P 500", []string{"$50.00 per index point", "Symbol: ES"}, []string{"Micro E-mini Nasdaq-100"}},
		{"MES", "Micro E-mini S&P 500", []string{"$5.00 per index point", "Symbol: MES"}, nil},
		{"ZB", "30-Year U.S. Treasury Bond", []string{"$1000.00 per point", "Symbol: ZB"}, []string{"index point"}},
	}
	for _, c := range cases {
		p := e.BuildFuturesDecisionSystemPrompt(c.symbol, 50000)
		if !strings.Contains(p, c.wantName) {
			t.Errorf("%s prompt missing instrument name %q", c.symbol, c.wantName)
		}
		for _, s := range c.wantHave {
			if !strings.Contains(p, s) {
				t.Errorf("%s prompt missing %q (wrong/missing point value or symbol)", c.symbol, s)
			}
		}
		for _, s := range c.wantGone {
			if strings.Contains(p, s) {
				t.Errorf("%s prompt unexpectedly contains %q (leaked another instrument)", c.symbol, s)
			}
		}
	}

	// micro≠mini: the same prompt builder must NOT give a micro its mini's $/point.
	nq := e.BuildFuturesDecisionSystemPrompt("NQ", 50000)
	mnq := e.BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if strings.Contains(mnq, "$20.00 per index point") {
		t.Error("MNQ prompt wrongly shows NQ's $20 point value (micro inherited the mini's multiplier)")
	}
	if strings.Contains(nq, "$2.00 per index point") {
		t.Error("NQ prompt wrongly shows MNQ's $2 point value")
	}

	// Parked/unknown symbol defaults to MNQ (never a blank/wrong instrument).
	parked := e.BuildFuturesDecisionSystemPrompt("CL", 50000)
	if !strings.Contains(parked, "Micro E-mini Nasdaq-100") || !strings.Contains(parked, "Symbol: MNQ") {
		t.Error("a parked symbol (CL) should default the prompt to MNQ, not emit a blank/wrong instrument")
	}
}

func TestBuildFuturesSystemPrompt_NoCryptoVocab(t *testing.T) {
	p := BuildFuturesSystemPrompt(FuturesPromptConfig{
		Symbol:             "MNQ",
		ContractMultiplier: 2.0, // MNQ = $2/point
		TickSize:           0.25,
		MinStopPoints:      15,
		MaxStopPoints:      50,
		MinRiskReward:      1.5,
	})

	// Must NOT contain crypto vocabulary
	forbidden := []string{
		"cryptocurrency", "altcoin", "BTC", "ETH", "USDT", "perpetual",
		"funding rate", "coins simultaneously",
	}
	for _, f := range forbidden {
		if strings.Contains(p, f) {
			t.Errorf("futures prompt contains forbidden crypto term %q", f)
		}
	}

	// Must contain futures-specific framing
	required := []string{
		"NQ", "tick", "contract", "stop loss", "take profit", "MNQ",
	}
	for _, r := range required {
		if !strings.Contains(p, r) {
			t.Errorf("futures prompt missing required term %q", r)
		}
	}
}

func TestBuildFuturesUserPrompt_IncludesIndicators(t *testing.T) {
	p := BuildFuturesUserPrompt(FuturesContext{
		Symbol:       "MNQ",
		CurrentPrice: 21500.00,
		EMA20:        21495.00,
		EMA50:        21480.00,
		RSI14:        58.3,
		MACD:         3.21,
		ATR14:        12.5,
		BollUpper:    21540.00,
		BollLower:    21460.00,
	})
	for _, s := range []string{"21500.00", "EMA20", "RSI14", "ATR14", "Bollinger"} {
		if !strings.Contains(p, s) {
			t.Errorf("user prompt missing %q", s)
		}
	}
}
