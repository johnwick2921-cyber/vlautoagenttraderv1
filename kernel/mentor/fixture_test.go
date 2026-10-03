package mentor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vl/market"
)

// fixture is the on-disk shape written by testdata/extract_fixtures.py:
// recorded MNQ bars copied READ-ONLY from the DB copy (the bot's own tape).
type fixture struct {
	Symbol   string                  `json:"symbol"`
	Contract string                  `json:"contract"`
	Bars     map[string][][5]float64 `json:"bars"` // tf -> [ot,o,h,l,c]
}

// loadFixture reads a recorded fixture and returns the given timeframe's bars
// as market.Kline (canon 53: tests run on the bot's REAL recorded bars, not
// synthetic inputs).
func loadFixture(t *testing.T, name, tf string) []market.Kline {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	rows, ok := f.Bars[tf]
	if !ok {
		t.Fatalf("fixture %s: no tf %q", name, tf)
	}
	out := make([]market.Kline, 0, len(rows))
	for _, r := range rows {
		out = append(out, market.Kline{
			OpenTime:  int64(r[0]),
			Open:      r[1],
			High:      r[2],
			Low:       r[3],
			Close:     r[4],
			CloseTime: int64(r[0]) + 59_999,
		})
	}
	return out
}
