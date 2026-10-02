package agent

import "testing"

func TestIsStockSymbol(t *testing.T) {
	tests := []struct {
		sym  string
		want bool
	}{
		// CME futures symbols — must NOT be detected as stock
		{"MNQ", false},
		{"mnq", false},
		{"ES", false},
		{"MNQU6", false},

		// Real stock tickers — must be detected as stock
		{"AAPL", true},
		{"TSLA", true},
		{"NVDA", true},
		{"MSFT", true},
		{"GOOGL", true},
		{"AMZN", true},
		{"META", true},
		{"AMD", true},
		{"PLTR", true},
		{"BA", true},
		{"F", true},   // Ford — 1 letter
		{"GM", true},  // 2 letters
		{"JPM", true}, // 3 letters

		// Mixed / edge cases
		{"btc", true},     // legacy crypto ticker — now just a 3-letter ticker
		{"aapl", true},    // lowercase stock (uppercased internally)
		{"BTC123", false}, // not pure letters
		{"123456", false}, // digits
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.sym, func(t *testing.T) {
			got := isStockSymbol(tt.sym)
			if got != tt.want {
				t.Errorf("isStockSymbol(%q) = %v, want %v", tt.sym, got, tt.want)
			}
		})
	}
}
