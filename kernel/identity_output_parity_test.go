package kernel

import (
	"encoding/json"
	"math"
	"nofx/market"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Golden generated on origin/dev 08aea8b8, before identity capture. It includes
// the production emitter, dedupe, scoring, seating and legacy map-render output.
func TestIdentityLegacyOutputParity(t *testing.T) {
	now := time.Date(2030, 9, 12, 12, 0, 0, 0, CTLocation())
	start := now.Add(-48 * time.Hour)
	bars := make([]market.Kline, 48*60)
	for i := range bars {
		s := start.Add(time.Duration(i) * time.Minute)
		p := 20000 + 60*math.Sin(float64(i)/13) + 20*math.Cos(float64(i)/71)
		bars[i] = market.Kline{OpenTime: s.UnixMilli(), CloseTime: s.Add(time.Minute).UnixMilli() - 1, Open: p - 1, High: p + 3, Low: p - 3, Close: p + 1, Volume: float64(100 + i%31)}
	}
	seated, pool, price, datr, raw := AssembleResearchLevels("identity-parity", bars, DefaultSessionRegistry(), "MNQ", 12, now, 2, "")
	rendered := RenderMapBlock(BuildMapCandidates(seated, price, 20, MapCandidateOpts{}), price)
	data, err := json.MarshalIndent(struct {
		Raw          []DetectedLevel
		Pool, Seated []ScoredLevel
		Price, DATR  float64
		Rendered     string
	}{raw, pool, seated, price, datr, rendered}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "identity_legacy_output.json")
	if os.Getenv("IDENTITY_WRITE_GOLDEN") == "1" {
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(expected) != string(append(data, '\n')) {
		t.Fatal("identity metadata changed legacy production output; compare against pinned dev golden")
	}
}
