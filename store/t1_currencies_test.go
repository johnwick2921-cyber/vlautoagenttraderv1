package store

import (
	"reflect"
	"testing"
)

// W-T1-CURRENCIES (2026-09-18) — the ONE resolver. Empty → ["USD"] (shipped
// default); canonical upper-case, trimmed, deduped; ALL/* wins over everything.
func TestT1CurrenciesForResolver(t *testing.T) {
	cases := []struct {
		name  string
		cfg   *DayPlanConfig
		want  []string
		saved bool
	}{
		{"nil config", nil, []string{"USD"}, false},
		{"absent", &DayPlanConfig{}, []string{"USD"}, false},
		{"empty list", &DayPlanConfig{T1Currencies: []string{}}, []string{"USD"}, false},
		{"blank entries", &DayPlanConfig{T1Currencies: []string{"", "  "}}, []string{"USD"}, false},
		{"saved USD prints saved", &DayPlanConfig{T1Currencies: []string{"USD"}}, []string{"USD"}, true},
		{"lower, spaces, dupes", &DayPlanConfig{T1Currencies: []string{" usd", "EUR ", "eur", "Gbp"}}, []string{"USD", "EUR", "GBP"}, true},
		{"ALL", &DayPlanConfig{T1Currencies: []string{"all"}}, []string{T1CurrencyAll}, true},
		{"star", &DayPlanConfig{T1Currencies: []string{"USD", "*"}}, []string{T1CurrencyAll}, true},
	}
	for _, c := range cases {
		if got := c.cfg.T1CurrenciesFor(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: T1CurrenciesFor = %v want %v", c.name, got, c.want)
		}
		if got := c.cfg.T1CurrenciesSaved(); got != c.saved {
			t.Errorf("%s: T1CurrenciesSaved = %v want %v", c.name, got, c.saved)
		}
	}
	if DefaultT1Currencies()[0] != "USD" || len(DefaultT1Currencies()) != 1 {
		t.Fatalf("shipped default must be exactly [USD]: %v", DefaultT1Currencies())
	}
}
