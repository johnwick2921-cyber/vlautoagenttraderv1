package kernel

import "testing"

// gateArgs mirrors validateDecision's risk params for a typical $50k SIM
// account (btcEthLev=10, altLev=5, btcEthRatio=5, altRatio=1).
func gateArgs() (equity float64, btcEthLev, altLev int, btcEthRatio, altRatio float64) {
	return 50000, 10, 5, 5.0, 1.0
}

// TestFuturesGate_AcceptsRealisticMNQOpen proves the futures notional
// exemption: a 1-contract-ish MNQ open (~$60k notional) now PASSES the gate,
// where the crypto equity×ratio cap ($50k) previously rejected it.
func TestFuturesGate_AcceptsRealisticMNQOpen(t *testing.T) {
	eq, btcEthLev, altLev, btcEthRatio, altRatio := gateArgs()
	d := &Decision{
		Symbol:          "MNQ",
		Action:          "open_long",
		Leverage:        1,
		PositionSizeUSD: 60000, // ~1 MNQ contract notional (> the old $50k cap)
		StopLoss:        21480.00,
		TakeProfit:      21560.00, // SL<TP, 0.2-entry placement => R/R 4:1
	}
	if err := validateDecision(d, eq, btcEthLev, altLev, btcEthRatio, altRatio, 0, 0, 20, nil); err != nil {
		t.Fatalf("expected MNQ $60k open to PASS the futures gate, got: %v", err)
	}
}

// TestFuturesGate_RejectsAbsurdMNQNotional confirms the cap is REAL, not
// accept-everything: a notional above equity×futuresMaxNotionalLeverage
// ($50k×20 = $1M) is rejected.
func TestFuturesGate_RejectsAbsurdMNQNotional(t *testing.T) {
	eq, btcEthLev, altLev, btcEthRatio, altRatio := gateArgs()
	d := &Decision{
		Symbol:          "MNQ",
		Action:          "open_long",
		Leverage:        1,
		PositionSizeUSD: 2_000_000, // > $1M ceiling
		StopLoss:        21480.00,
		TakeProfit:      21560.00,
	}
	if err := validateDecision(d, eq, btcEthLev, altLev, btcEthRatio, altRatio, 0, 0, 20, nil); err == nil {
		t.Fatal("expected absurd $2M MNQ notional to be REJECTED, but it passed")
	}
}

// TestFuturesGate_CryptoCapUnchanged is the regression guard: a crypto open
// above equity×ratio ($50k) is STILL rejected (the futures branch must not
// loosen crypto).
func TestFuturesGate_CryptoCapUnchanged(t *testing.T) {
	eq, btcEthLev, altLev, btcEthRatio, altRatio := gateArgs()
	d := &Decision{
		Symbol:          "SOLUSDT", // altcoin, ratio 1x => $50k cap
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 60000, // > $50k crypto cap
		StopLoss:        100.0,
		TakeProfit:      130.0,
	}
	if err := validateDecision(d, eq, btcEthLev, altLev, btcEthRatio, altRatio, 0, 0, 20, nil); err == nil {
		t.Fatal("expected crypto SOLUSDT $60k open to STILL be rejected by the $50k cap")
	}
}

// TestFuturesGate_WaitAlwaysValid — a wait decision (the common futures
// output) validates regardless of symbol.
func TestFuturesGate_WaitAlwaysValid(t *testing.T) {
	eq, btcEthLev, altLev, btcEthRatio, altRatio := gateArgs()
	d := &Decision{Symbol: "MNQ", Action: "wait"}
	if err := validateDecision(d, eq, btcEthLev, altLev, btcEthRatio, altRatio, 0, 0, 20, nil); err != nil {
		t.Fatalf("wait should always validate, got: %v", err)
	}
}

// mnqOpen builds a valid MNQ open with a given confidence (notional well under
// the equity×20 ceiling). With the F1 real-R:R entry reference of 21496 (see
// mnqRRCtx), SL 21480 / TP 21560 gives risk 16, reward 64 → R:R = 4.0.
func mnqOpen(confidence int) *Decision {
	return &Decision{
		Symbol: "MNQ", Action: "open_long", Leverage: 1,
		PositionSizeUSD: 60000, StopLoss: 21480.00, TakeProfit: 21560.00,
		Confidence: confidence,
	}
}

// mnqRRCtx supplies an entry reference of 21496 → the mnqOpen decision computes a
// real R:R of exactly 4.0 (64/16).
func mnqRRCtx() *Context { return mdCtx("MNQ", 21496.00) }

// TestStrategyStudio_MinRRConfigDriven proves the R/R floor is the CONFIGURED
// value, not a hardcoded 3.0: the same real-4:1 decision PASSES at minRR=3.0 and is
// REJECTED at minRR=5.0.
func TestStrategyStudio_MinRRConfigDriven(t *testing.T) {
	eq, b, a, br, ar := gateArgs()
	if err := validateDecision(mnqOpen(90), eq, b, a, br, ar, 3.0, 0, 20, mnqRRCtx()); err != nil {
		t.Fatalf("real-4:1 decision should PASS at minRR=3.0, got: %v", err)
	}
	if err := validateDecision(mnqOpen(90), eq, b, a, br, ar, 5.0, 0, 20, mnqRRCtx()); err == nil {
		t.Fatal("real-4:1 decision should be REJECTED at minRR=5.0 (config-driven, not hardcoded 3.0)")
	}
}

// TestStrategyStudio_MinConfidenceGate proves min_confidence is now ENFORCED:
// a confidence-50 open PASSES when the gate is disabled (0) and is REJECTED at
// min_confidence=75. Entry ctx makes the R:R (4.0) pass so confidence is reached.
func TestStrategyStudio_MinConfidenceGate(t *testing.T) {
	eq, b, a, br, ar := gateArgs()
	if err := validateDecision(mnqOpen(50), eq, b, a, br, ar, 0, 0, 20, mnqRRCtx()); err != nil {
		t.Fatalf("confidence-50 open should PASS with the gate disabled (0), got: %v", err)
	}
	if err := validateDecision(mnqOpen(50), eq, b, a, br, ar, 0, 75, 20, mnqRRCtx()); err == nil {
		t.Fatal("confidence-50 open should be REJECTED at min_confidence=75")
	}
}
