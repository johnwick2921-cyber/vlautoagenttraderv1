package databento

import (
	"testing"
	"time"
)

// TestNextExpiryFromSymbol_YearDisambiguation covers the single-digit-year
// wrap-around logic. Year digit '0' in a "now" of 2029-12-01 must map to
// year 2030, not 2020.
func TestNextExpiryFromSymbol_YearDisambiguation(t *testing.T) {
	cases := []struct {
		symbol string
		now    time.Time
		want   time.Time
	}{
		{"MNQM6", date(2026, 5, 22), date(2026, 6, 19)},  // current quarter
		{"MNQU6", date(2026, 5, 22), date(2026, 9, 18)},  // next quarter
		{"MNQH7", date(2026, 5, 22), date(2027, 3, 19)},  // next year
		{"MNQM0", date(2029, 12, 1), date(2030, 6, 21)},  // year-digit wrap into next decade
		{"MNQH0", date(2029, 12, 1), date(2030, 3, 15)},  // wrap, earliest month
	}
	for _, tc := range cases {
		got, err := NextExpiryFromSymbol(tc.symbol, tc.now)
		if err != nil {
			t.Errorf("symbol=%q: %v", tc.symbol, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("symbol=%q now=%v: got %v, want %v", tc.symbol, tc.now, got, tc.want)
		}
	}
}

// TestDaysUntilExpiry_ReturnsLargeOnParseError verifies the documented
// fallback: a symbol that can't be parsed returns 999 so callers in
// conditional logic (e.g. "if days <= 5") treat it as "not near expiry".
func TestDaysUntilExpiry_ReturnsLargeOnParseError(t *testing.T) {
	got := DaysUntilExpiry("BAD", date(2026, 5, 22))
	if got != 999 {
		t.Errorf("DaysUntilExpiry(\"BAD\") = %d, want 999", got)
	}

	// Also verify other malformed inputs.
	cases := []string{"", "X", "MNQA6", "MNQM!"}
	for _, sym := range cases {
		if got := DaysUntilExpiry(sym, date(2026, 5, 22)); got != 999 {
			t.Errorf("DaysUntilExpiry(%q) = %d, want 999", sym, got)
		}
	}
}

// TestThirdFridayOf pins the CME quarterly expiry dates for 2026. These are
// well-known calendar dates and serve as a regression check against any
// future refactor of thirdFridayOf.
func TestThirdFridayOf(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  time.Time
	}{
		{2026, time.March, date(2026, 3, 20)},
		{2026, time.June, date(2026, 6, 19)},
		{2026, time.September, date(2026, 9, 18)},
		{2026, time.December, date(2026, 12, 18)},
	}
	for _, tc := range cases {
		got := thirdFridayOf(tc.year, tc.month)
		if !got.Equal(tc.want) {
			t.Errorf("thirdFridayOf(%d, %v) = %v, want %v", tc.year, tc.month, got, tc.want)
		}
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
