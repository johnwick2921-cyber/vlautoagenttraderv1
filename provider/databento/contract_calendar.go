// Package databento — contract calendar / expiry helpers.
//
// Plan 2 Task 19: CME index futures (NQ / MNQ / ES / MES) expire on the 3rd
// Friday of March / June / September / December. Symbol codes follow the
// CME convention: "<root><month-code><year-digit>" (e.g. MNQM6 = MNQ Jun 2026).
//
// This file provides:
//   - NextExpiryFromSymbol — parse a contract code into its expiry date,
//     disambiguating the single-digit year against `now`.
//   - DaysUntilExpiry — convenience wrapper for engine-side gating.
//   - thirdFridayOf — CME index-futures expiry convention.
package databento

import (
	"fmt"
	"strings"
	"time"
)

// CME month codes for futures contract symbology.
var cmeMonthCodes = map[byte]time.Month{
	'F': time.January,
	'G': time.February,
	'H': time.March,
	'J': time.April,
	'K': time.May,
	'M': time.June,
	'N': time.July,
	'Q': time.August,
	'U': time.September,
	'V': time.October,
	'X': time.November,
	'Z': time.December,
}

// NextExpiryFromSymbol returns the expiry date of the given CME contract code,
// disambiguating the single-digit year against `now`. Format: last 2 chars are
// month code (F/G/H/J/K/M/N/Q/U/V/X/Z) + last digit of year.
//
// Year disambiguation rule: assume the contract is in the current decade.
// If that would place the contract more than 1 year in the past, assume next
// decade. This handles the normal case (front-month contract within ~1 year
// of now) without breaking when the year-digit wraps (e.g. 2030).
//
// Examples (now=2026-05-22):
//
//	"MNQM6" → 2026-06 (current decade, current year)
//	"MNQU6" → 2026-09 (current decade, this year)
//	"MNQH7" → 2027-03 (current decade, next year)
//	"MNQM0" → 2030-06 (current decade, but year 2020 would be 6 years ago → bump to 2030)
func NextExpiryFromSymbol(symbol string, now time.Time) (time.Time, error) {
	if len(symbol) < 2 {
		return time.Time{}, fmt.Errorf("contract code too short: %q", symbol)
	}
	code := strings.ToUpper(symbol)
	monthChar := code[len(code)-2]
	yearChar := code[len(code)-1]

	month, ok := cmeMonthCodes[monthChar]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid CME month code %q in %q", monthChar, symbol)
	}
	if yearChar < '0' || yearChar > '9' {
		return time.Time{}, fmt.Errorf("invalid year digit %q in %q", yearChar, symbol)
	}
	yearDigit := int(yearChar - '0')

	decade := (now.Year() / 10) * 10
	year := decade + yearDigit
	// If the candidate year is more than 1 year before now, the contract code
	// refers to the next decade.
	if year < now.Year()-1 {
		year += 10
	}
	return thirdFridayOf(year, month), nil
}

// DaysUntilExpiry returns calendar days from now until contract expiry.
// Returns 999 if the symbol cannot be parsed (treat as "not near expiry").
func DaysUntilExpiry(symbol string, now time.Time) int {
	exp, err := NextExpiryFromSymbol(symbol, now)
	if err != nil {
		return 999
	}
	return int(exp.Sub(now).Hours() / 24)
}

// thirdFridayOf returns the 3rd Friday of the given month — CME index futures
// expiry convention.
func thirdFridayOf(year int, month time.Month) time.Time {
	// Start at day 1, advance to first Friday, then add 14 days.
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(time.Friday) - int(first.Weekday()) + 7) % 7
	return first.AddDate(0, 0, offset+14)
}
