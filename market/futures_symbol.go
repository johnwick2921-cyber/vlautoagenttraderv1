// Task 12 / Cluster D — CME futures symbol detection.
//
// Single source of truth for "is this a CME futures symbol?" used by
// market.Normalize (case-preservation bypass) and market.GetWithTimeframes
// (route to Databento instead of CoinAnk). store/strategy.go has a small
// duplicate of this helper because store deliberately does not import market;
// keep the two in sync.

package market

import "strings"

// cmeFuturesRoots is the deny-list of crypto-collision-safe CME root symbols.
// Extend as new CME products are needed; keep conservative so this never
// matches a crypto ticker (BTC, ETH, SOL, etc.).
var cmeFuturesRoots = map[string]struct{}{
	"NQ":  {}, // E-mini Nasdaq-100
	"MNQ": {}, // Micro E-mini Nasdaq-100
	"ES":  {}, // E-mini S&P 500
	"MES": {}, // Micro E-mini S&P 500
	"RTY": {}, // E-mini Russell 2000
	"M2K": {}, // Micro E-mini Russell 2000
	"YM":  {}, // E-mini Dow
	"MYM": {}, // Micro E-mini Dow
	"CL":  {}, // Crude oil
	"MCL": {}, // Micro WTI Crude Oil
	"NG":  {}, // Henry Hub Natural Gas
	"GC":  {}, // Gold
	"MGC": {}, // Micro Gold
	"SI":  {}, // Silver
	"ZB":  {}, // 30-Year U.S. Treasury Bond
	"ZN":  {}, // 10-Year U.S. T-Note
	"ZF":  {}, // 5-Year U.S. T-Note
	"ZT":  {}, // 2-Year U.S. T-Note
}

// IsCMEFuturesSymbol reports whether a symbol is a CME futures symbol that
// must bypass crypto normalization (no ToUpper, no USDT append) and route to
// Databento.
//
// Matches:
//   - continuous form `<ROOT>.c.<N>` (e.g. NQ.c.0, MNQ.c.0) — case-sensitive
//     on the lowercase `.c.` segment per Databento's continuous symbology
//     convention.
//   - known CME roots, optionally followed by a contract suffix
//     (NQ, NQM6, MNQ, MNQU6, etc.) — uppercased for matching.
//
// Never matches crypto tickers (BTC, ETH, SOL, …) because the root set is
// conservative.
func IsCMEFuturesSymbol(symbol string) bool {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return false
	}
	// Continuous form: any symbol containing ".c." (lowercase c) is a
	// Databento continuous symbol.
	if strings.Contains(s, ".c.") {
		return true
	}
	// Known root, optionally followed by ".something" or "<month><year>".
	upper := strings.ToUpper(s)
	if _, ok := cmeFuturesRoots[upper]; ok {
		return true
	}
	for root := range cmeFuturesRoots {
		if strings.HasPrefix(upper, root+".") {
			return true
		}
		// Contract code form: <ROOT><month-letter HMUZ><year-digit>
		// (NQM6, MNQU6, etc.). Require exact root prefix + 2 more chars
		// where the first is a CME quarterly month code and the second
		// is a single digit.
		if strings.HasPrefix(upper, root) && len(upper) == len(root)+2 {
			tail := upper[len(root):]
			if isContractMonth(root, tail[0]) && tail[1] >= '0' && tail[1] <= '9' {
				return true
			}
		}
	}
	return false
}

// futuresPointValues maps a CME root to the USD value of a 1.00-point move
// for ONE contract (the contract multiplier). Used to size futures positions
// in contracts: contracts = notional / (price × pointValue).
var futuresPointValues = map[string]float64{
	"NQ":  20.0,   // E-mini Nasdaq-100   ($20/pt)
	"MNQ": 2.0,    // Micro E-mini Nasdaq ($2/pt)
	"ES":  50.0,   // E-mini S&P 500      ($50/pt)
	"MES": 5.0,    // Micro E-mini S&P    ($5/pt)
	"RTY": 50.0,   // E-mini Russell 2000 ($50/pt)
	"M2K": 5.0,    // Micro E-mini Russell($5/pt)
	"YM":  5.0,    // E-mini Dow          ($5/pt)
	"MYM": 0.5,    // Micro E-mini Dow    ($0.50/pt)
	"CL":  1000.0,  // Crude oil           ($1000 per $1 — 1,000 bbl)
	"MCL": 100.0,   // Micro WTI Crude     ($100 per $1 — 100 bbl; 1/10 CL)
	"NG":  10000.0, // Henry Hub Nat Gas   ($10,000 per $1 — 10,000 MMBtu)
	"GC":  100.0,   // Gold                ($100 per $1 — 100 oz)
	"MGC": 10.0,    // Micro Gold          ($10 per $1 — 10 oz; 1/10 GC)
	"SI":  5000.0,  // Silver              ($5000 per $1 — 5,000 oz)
	"ZB":  1000.0,  // 30Y T-Bond          ($1000 per pt — $100k face)
	"ZN":  1000.0,  // 10Y T-Note          ($1000 per pt — $100k face)
	"ZF":  1000.0,  // 5Y T-Note           ($1000 per pt — $100k face)
	"ZT":  2000.0,  // 2Y T-Note           ($2000 per pt — $200k face)
}

// futuresTickSizes maps a CME root to its minimum price increment (tick), in the
// same decimal units the bot's bars use. Index ticks are simple decimals;
// Treasuries quote in 32nds/64ths/128ths expressed as decimals (ZB 1/32=0.03125).
// RESOLVING families only (index + treasury); energy/metals are parked (Phase 2.5)
// so they are absent and callers default safely. Used by the futures system prompt
// to align stops to the real instrument's tick.
var futuresTickSizes = map[string]float64{
	"NQ": 0.25, "MNQ": 0.25, "ES": 0.25, "MES": 0.25,
	"RTY": 0.10, "M2K": 0.10, "YM": 1.0, "MYM": 1.0,
	"ZB": 0.03125, "ZN": 0.015625, "ZF": 0.0078125, "ZT": 0.0078125,
}

// FuturesTickSize returns the tick size for a CME root (any symbol form), or 0 if
// unknown (caller must treat 0 as "unknown"). Index + treasury only.
func FuturesTickSize(symbol string) float64 {
	if root := futuresRoot(symbol); root != "" {
		return futuresTickSizes[root]
	}
	return 0
}

// FuturesPointValue returns the USD value of a 1.00-point move for one
// contract of the given CME futures symbol (e.g. MNQ=2, NQ=20). Accepts any
// symbol form (continuous "NQ.c.0", contract "NQM6", bare "MNQ", qualified
// "MNQ 06-26"). Returns 0 for non-futures / unknown roots — callers must
// treat 0 as "unknown" and NOT divide by it.
func FuturesPointValue(symbol string) float64 {
	if root := futuresRoot(symbol); root != "" {
		return futuresPointValues[root]
	}
	return 0
}

// futuresRoot extracts the CME root from any symbol form. Longest-root-first
// so "MNQ" wins over "NQ". Returns "" if no known root matches.
func futuresRoot(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	best := ""
	for root := range cmeFuturesRoots {
		matched := s == root ||
			strings.HasPrefix(s, root+".") ||
			strings.HasPrefix(s, root+" ")
		if !matched && strings.HasPrefix(s, root) && len(s) == len(root)+2 {
			tail := s[len(root):]
			matched = isContractMonth(root, tail[0]) && tail[1] >= '0' && tail[1] <= '9'
		}
		if matched && len(root) > len(best) {
			best = root
		}
	}
	return best
}

// isQuarterlyMonth reports whether b is one of the CME quarterly month
// codes (H=Mar, M=Jun, U=Sep, Z=Dec). Index + Treasury futures roll
// quarterly. Retained for reference / quarterly-only callers; contract-code
// recognition now uses the broader isFuturesMonth (energy/metals list all 12).
func isQuarterlyMonth(b byte) bool {
	switch b {
	case 'H', 'M', 'U', 'Z':
		return true
	}
	return false
}

// futuresMonthCodes maps a CME root to the single-letter month codes it actually
// lists, for contract-code-form recognition (<ROOT><month><year>). Per-root, not
// blanket: index + Treasury roll QUARTERLY (H/M/U/Z), so NQF6 (Jan) is NOT a real
// contract and must not match; energy (CL/MCL/NG) lists ALL 12, so NGF6 IS valid;
// metals list their own cycles. (All verified against cmegroup.com.) Keep in sync
// with cmeFuturesRoots — every root needs an entry.
var futuresMonthCodes = map[string]string{
	// Equity index (E-mini + Micro) — quarterly H/M/U/Z.
	"NQ": "HMUZ", "MNQ": "HMUZ", "ES": "HMUZ", "MES": "HMUZ",
	"RTY": "HMUZ", "M2K": "HMUZ", "YM": "HMUZ", "MYM": "HMUZ",
	// CBOT Treasuries — quarterly H/M/U/Z.
	"ZB": "HMUZ", "ZN": "HMUZ", "ZF": "HMUZ", "ZT": "HMUZ",
	// Energy — every calendar month (F..Z).
	"CL": "FGHJKMNQUVXZ", "MCL": "FGHJKMNQUVXZ", "NG": "FGHJKMNQUVXZ",
	// Metals — gold Feb/Apr/Jun/Aug/Oct/Dec; silver Jan/Mar/May/Jul/Sep/Dec.
	"GC": "GJMQVZ", "MGC": "GJMQVZ", "SI": "FHKNUZ",
}

// isContractMonth reports whether month code b is one that the given root lists
// (per futuresMonthCodes). Roots absent from the map (none expected) match no
// contract-code month. This is how energy/metals recognize their monthly codes
// while index/Treasury stay quarterly-only.
func isContractMonth(root string, b byte) bool {
	return strings.IndexByte(futuresMonthCodes[root], b) >= 0
}
