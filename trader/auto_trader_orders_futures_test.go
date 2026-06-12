package trader

import "testing"

// TestFuturesOrderQuantity verifies notional → contracts conversion + clamp
// for CME futures. MNQ point value = $2/pt.
func TestFuturesOrderQuantity(t *testing.T) {
	const mnqPrice = 30323.75 // ≈ live MNQ; 1 contract notional ≈ $60,647

	cases := []struct {
		name        string
		symbol      string
		notionalUSD float64
		price       float64
		want        float64
	}{
		// ~1 contract: 60000 / (30323.75 × 2) = 0.989 → round → 1
		{"one contract", "MNQ", 60000, mnqPrice, 1},
		// ~3 contracts: 180000 / (30323.75 × 2) = 2.97 → 3
		{"three contracts", "MNQ", 180000, mnqPrice, 3},
		// Floor at 1 even for tiny notional
		{"floor at one", "MNQ", 1000, mnqPrice, 1},
		// Clamp at maxFuturesContracts (10) for huge notional
		{"clamp at max", "MNQ", 5_000_000, mnqPrice, maxFuturesContracts},
		// NQ ($20/pt): 1 contract notional ≈ $606k; 600000/(30323.75×20)=0.989→1
		{"NQ one contract", "NQ", 600000, mnqPrice, 1},
		// Safe default 1 when point value unknown (non-futures shouldn't reach
		// here, but defend): pv=0 → 1
		{"unknown pv default", "BTCUSDT", 60000, mnqPrice, 1},
		// Safe default when price is 0 (avoid div-by-zero)
		{"zero price default", "MNQ", 60000, 0, 1},
	}
	for _, c := range cases {
		got := futuresOrderQuantity(c.symbol, c.notionalUSD, c.price, int(maxFuturesContracts))
		if got != c.want {
			t.Errorf("%s: futuresOrderQuantity(%q, %.0f, %.2f) = %v, want %v",
				c.name, c.symbol, c.notionalUSD, c.price, got, c.want)
		}
	}
}
