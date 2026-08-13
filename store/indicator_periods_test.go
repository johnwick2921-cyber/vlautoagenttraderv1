// F11c — indicator period lists are validated at save: each must be 1..500. An
// absurd value (9999) is a hard error, never a silently-empty series.
package store

import "testing"

func cfgWith(ind IndicatorConfig) *StrategyConfig {
	return &StrategyConfig{Indicators: ind}
}

func TestValidateIndicatorPeriods(t *testing.T) {
	cases := []struct {
		name    string
		ind     IndicatorConfig
		wantErr bool
	}{
		{"defaults valid", IndicatorConfig{EMAPeriods: []int{20, 50}, RSIPeriods: []int{7, 14}, ATRPeriods: []int{14}, BOLLPeriods: []int{20}}, false},
		{"empty lists valid", IndicatorConfig{}, false},
		{"boundary 500 valid", IndicatorConfig{EMAPeriods: []int{500}}, false},
		{"above 500 rejected", IndicatorConfig{EMAPeriods: []int{501}}, true},
		{"9999 rejected", IndicatorConfig{EMAPeriods: []int{9, 21, 9999}}, true},
		{"zero rejected", IndicatorConfig{RSIPeriods: []int{0}}, true},
		{"negative rejected", IndicatorConfig{ATRPeriods: []int{-5}}, true},
		{"bad BOLL rejected", IndicatorConfig{BOLLPeriods: []int{700}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := cfgWith(c.ind).ValidateIndicatorPeriods()
			if (err != nil) != c.wantErr {
				t.Fatalf("ValidateIndicatorPeriods() err=%v, wantErr=%v", err, c.wantErr)
			}
		})
	}
}
