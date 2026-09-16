package kernel

import (
	"math"
	"nofx/market"
	"strings"
	"testing"
	"time"
)

func TestAuthoredInvalidationCompletedWindowAndUnknown(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-09-08T08:03:00-05:00")
	var bars []market.Kline
	end := now.UnixMilli() / 300000 * 300000
	for ms := end - 600000; ms < end; ms += 60000 {
		bars = append(bars, market.Kline{OpenTime: ms, CloseTime: ms + 60000, Close: 100})
	}
	cases := []struct {
		name, rule  string
		change      func([]market.Kline) []market.Kline
		known, dead bool
	}{
		{"above", "5m close above 99 kills the setup", nil, true, true},
		{"below", "5m close below 101 kills the setup", nil, true, true},
		{"equal", "5m close above 100 kills the setup", nil, true, false},
		{"two closes", "2x5m<101", nil, true, true},
		{"two needs both", "2x5m<101", func(b []market.Kline) []market.Kline { b[4].Close = 102; return b }, true, false},
		{"missing minute", "5m close below 101", func(b []market.Kline) []market.Kline { return b[:len(b)-1] }, false, false},
		{"duplicate minute", "5m close below 101", func(b []market.Kline) []market.Kline { return append(b, b[9]) }, false, false},
		{"nonfinite", "5m close below 101", func(b []market.Kline) []market.Kline { b[9].Close = math.NaN(); return b }, false, false},
		{"forming bucket ignored", "5m close above 101", func(b []market.Kline) []market.Kline {
			return append(b, market.Kline{OpenTime: end, CloseTime: end + 60000, Close: 105})
		}, true, false},
		{"sequential unknown", "5m close back below 101 negates breakout", nil, false, false},
		{"compound unknown", "5m close above 99 or a 1m MSS above 105 cancels the fade.", nil, false, false},
		{"qualified parentheses unknown", "5m close below 101 (only after breakout) kills the setup", nil, false, false},
		{"subjective unknown", "auction changes character", nil, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]market.Kline(nil), bars...)
			if tc.change != nil {
				b = tc.change(b)
			}
			v := EvaluateAuthoredInvalidationAt(PlanScenario{ID: "S1", Invalid: tc.rule}, b, now)
			if v.Known != tc.known || v.Invalidated != tc.dead {
				t.Fatalf("%+v", v)
			}
			if !v.Known && !strings.Contains(v.Reason, "UNKNOWN") {
				t.Fatalf("unknown unnamed: %+v", v)
			}
		})
	}
}
