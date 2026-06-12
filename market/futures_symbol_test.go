// Task 12 / Cluster D tests — verify IsCMEFuturesSymbol detection +
// Normalize bypass for CME futures. Crypto path assertions live in
// data_test.go and remain unchanged.

package market

import "testing"

func TestIsCMEFuturesSymbol(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Continuous form — must match exactly as Databento returns/expects.
		{"NQ.c.0", true},
		{"MNQ.c.0", true},
		{"ES.c.0", true},
		{"MES.c.0", true},
		// Lowercase root + .c. is still a futures symbol (root is
		// uppercased internally before lookup).
		{"nq.c.0", true},
		// Bare CME root.
		{"NQ", true},
		{"MNQ", true},
		{"ES", true},
		// Specific contract code (NQM6 = June 2026).
		{"NQM6", true},
		{"MNQU6", true},
		{"ESZ6", true},
		// Crypto must NOT match.
		{"BTC", false},
		{"BTCUSDT", false},
		{"ETH", false},
		{"ETHUSDT", false},
		{"SOL", false},
		{"DOGE", false},
		// Stock-flavored xyz symbols must NOT match (those have their
		// own xyz: prefix path).
		{"TSLA", false},
		{"AAPL", false},
		// Whitespace tolerance.
		{"  NQ.c.0  ", true},
		// Empty + malformed.
		{"", false},
		{".c.", true}, // matches the literal substring; documented behavior
		{"XYZ", false},
		// Edge case: CME root followed by single char is NOT a contract
		// code (need month-letter + digit, total root+2 chars).
		{"NQM", false},
		// Wrong month-letter (only HMUZ for quarterly index futures).
		{"NQF6", false},
	}
	for _, c := range cases {
		got := IsCMEFuturesSymbol(c.in)
		if got != c.want {
			t.Errorf("IsCMEFuturesSymbol(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNormalize_CMEFutures_PreservesCaseAndSkipsUSDT(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Continuous form: case preserved, no USDT.
		{"NQ.c.0", "NQ.c.0"},
		{"MNQ.c.0", "MNQ.c.0"},
		{"ES.c.0", "ES.c.0"},
		// Specific contract: case preserved.
		{"NQM6", "NQM6"},
		// Whitespace stripped, case preserved.
		{"  NQ.c.0  ", "NQ.c.0"},
	}
	for _, c := range cases {
		got := Normalize(c.in)
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q (CME futures must skip ToUpper + USDT append)", c.in, got, c.want)
		}
	}
}

func TestFuturesPointValue(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		// Index futures (the ones we trade): bare root, continuous, contract,
		// and qualified forms all resolve to the same multiplier.
		{"MNQ", 2.0},
		{"MNQ.c.0", 2.0},
		{"MNQU6", 2.0},
		{"MNQ 06-26", 2.0},
		{"NQ", 20.0},
		{"NQ.c.0", 20.0},
		{"ES", 50.0},
		{"MES", 5.0},
		{"RTY", 50.0},
		{"M2K", 5.0},
		{"YM", 5.0},
		{"MYM", 0.5},
		// Phase 1 — new CME roots (recognition). Each its own multiplier
		// (verified against cmegroup.com).
		{"CL", 1000.0},
		{"MCL", 100.0},
		{"NG", 10000.0},
		{"GC", 100.0},
		{"MGC", 10.0},
		{"SI", 5000.0},
		{"ZB", 1000.0},
		{"ZN", 1000.0},
		{"ZF", 1000.0},
		{"ZT", 2000.0},
		// Monthly contract-code recognition (energy lists all 12 months).
		{"NGF6", 10000.0},  // F = January
		{"MCLZ6", 100.0},   // Z = December (MCL wins over CL — longest root)
		// Non-futures / unknown → 0 (caller must not divide by it).
		{"BTCUSDT", 0},
		{"TSLA", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := FuturesPointValue(c.in); got != c.want {
			t.Errorf("FuturesPointValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestFuturesPointValue_MicroMiniDivergence asserts a micro NEVER inherits its
// mini's multiplier — same underlying, DIFFERENT contract value (a micro is 1/10
// of its mini). A wrong multiplier mis-values every PnL on that symbol (the id=45
// missing-×pv class). This is the cheap catch for a copy-paste error in the table.
func TestFuturesPointValue_MicroMiniDivergence(t *testing.T) {
	pairs := []struct {
		mini, micro     string
		miniPV, microPV float64
	}{
		{"NQ", "MNQ", 20.0, 2.0},
		{"ES", "MES", 50.0, 5.0},
		{"YM", "MYM", 5.0, 0.5},
		{"RTY", "M2K", 50.0, 5.0},
		{"GC", "MGC", 100.0, 10.0},
		{"CL", "MCL", 1000.0, 100.0},
	}
	for _, p := range pairs {
		gotMini, gotMicro := FuturesPointValue(p.mini), FuturesPointValue(p.micro)
		if gotMini != p.miniPV {
			t.Errorf("%s point value = %v, want %v", p.mini, gotMini, p.miniPV)
		}
		if gotMicro != p.microPV {
			t.Errorf("%s point value = %v, want %v", p.micro, gotMicro, p.microPV)
		}
		if gotMicro == gotMini {
			t.Errorf("micro %s (%v) must NOT equal mini %s (%v) — micro inherited the mini's multiplier",
				p.micro, gotMicro, p.mini, gotMini)
		}
		if gotMicro*10 != gotMini {
			t.Errorf("micro %s should be 1/10 of mini %s: micro=%v mini=%v", p.micro, p.mini, gotMicro, gotMini)
		}
	}
}

func TestNormalize_Crypto_UnchangedByTask12(t *testing.T) {
	// Sanity: crypto path must be byte-unchanged. If this test fails,
	// the Task 12 branch leaked into the crypto path.
	cases := []struct {
		in, want string
	}{
		{"btc", "BTCUSDT"},
		{"BTC", "BTCUSDT"},
		{"BTCUSDT", "BTCUSDT"},
		{"eth", "ETHUSDT"},
		{"SOL", "SOLUSDT"},
		// xyz dex assets keep their xyz: prefix path.
		{"TSLA", "xyz:TSLA"},
		{"AAPL", "xyz:AAPL"},
	}
	for _, c := range cases {
		got := Normalize(c.in)
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q (crypto/xyz path must be unchanged by Task 12)", c.in, got, c.want)
		}
	}
}
