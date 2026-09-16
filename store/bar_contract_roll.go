package store

import (
	"fmt"
	"strings"
)

// ── ROLL WAVE (2026-09-10) — THE CONTRACT COLUMN AND ITS ONE-TIME BACKFILL ──
//
// WHAT HAPPENED. The AddOn's VLContractResolver picks the front month by a UTC
// date rule (expiry minus eight days). September 2026 expires the 18th, so its
// roll date is the 10th, and at 00:00 UTC on the 11th — 19:00 CT on the 10th —
// the rule flipped to December. Nothing changed on the wire until the next
// reconnect: at 21:15:03 CT the subscription re-ACKed as "MNQ 12-26" and the
// bars that followed were on the December scale, ~292 points above September.
//
// THREE BARS PER SYMBOL ARRIVED WITH ONE CONTRACT'S OPEN AND THE OTHER'S CLOSE.
// The historical replay for 21:15 landed first with a September open; the live
// update for the same minute then landed with a December close; Go's upsert
// keeps whole bars, not halves, so the persisted row is the last frame's — and
// the last frame carried both. MNQ 21:15/21:16/21:17 show bodies of +284.5,
// +296.75 and +276.0 on a tape whose bodies are otherwise single digits.
//
// The boundaries below are READ FROM THE TAPE, not from the resolver's date
// rule: the 19:00 CT flip is where the AddOn DECIDED, 21:15 CT is where the
// bars CHANGED, and the 135 minutes between are September bars. Every row id
// is quoted so the backfill can be checked against the record by anyone.

// rollBoundary is one symbol's roll as observed in the persisted 1m tape.
type rollBoundary struct {
	Symbol string
	Old    string // the retired contract
	New    string // the current contract
	// MixedFromMs is the OPEN time of the first bar whose OHLC straddles the
	// roll. NewFromMs is the open time of the first bar wholly on the new
	// scale. Bars whose [open, open+tf) window intersects
	// [MixedFromMs, NewFromMs) are on neither scale and get ContractMixed.
	MixedFromMs int64
	NewFromMs   int64
	// Quoted evidence — the row ids the boundary rests on (A21).
	LastOldRow, FirstMixedRow, LastMixedRow, FirstNewRow int64
}

// The observed roll of 2026-09-10, ASIA session. Both symbols re-subscribed on
// the same reconnect; ES's contamination began one bar earlier than MNQ's.
//
//	MNQ  464732 21:14 o29134.00 c29132.00  last clean September
//	     464738 21:15 o29131.50 c29416.00  MIXED  (+284.50 body)
//	     464739 21:16 o29124.50 c29421.25  MIXED  (+296.75)
//	     464744 21:17 o29129.25 c29405.25  MIXED  (+276.00)
//	     464749 21:18 o29406.50 c29406.50  first clean December
//	ES   464728 21:13 o7609.50  c7610.75   last clean September
//	     464729 21:14 o7610.50  c7675.50   MIXED  (+65.00)
//	     464737 21:15 o7610.25  c7674.75   MIXED
//	     464740 21:16 o7609.25  c7675.00   MIXED
//	     464741 21:17 o7610.00  c7673.25   MIXED
//	     464745 21:18 o7673.25  c7673.25   first clean December
var roll20260910 = []rollBoundary{
	{Symbol: "MNQ", Old: "MNQ 09-26", New: "MNQ 12-26",
		MixedFromMs: 1789092900000, NewFromMs: 1789093080000, // 21:15:00 / 21:18:00 CT
		LastOldRow: 464732, FirstMixedRow: 464738, LastMixedRow: 464744, FirstNewRow: 464749},
	{Symbol: "ES", Old: "ES 09-26", New: "ES 12-26",
		MixedFromMs: 1789092840000, NewFromMs: 1789093080000, // 21:14:00 / 21:18:00 CT
		LastOldRow: 464728, FirstMixedRow: 464729, LastMixedRow: 464741, FirstNewRow: 464745},
}

// tfMillis is the bar duration for the timeframes the store holds. Unknown
// timeframes are treated as 1m — the narrowest window — so an unrecognised tf
// can only be UNDER-marked as mixed, never over-marked as clean.
func tfMillis(tf string) int64 {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1m":
		return 60_000
	case "3m":
		return 180_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1h":
		return 3_600_000
	case "2h":
		return 7_200_000
	case "4h":
		return 14_400_000
	case "6h":
		return 21_600_000
	case "8h":
		return 28_800_000
	case "12h":
		return 43_200_000
	case "1d":
		return 86_400_000
	case "3d":
		return 259_200_000
	case "1w":
		return 604_800_000
	}
	return 60_000
}

// migrateContractColumn adds the column (idempotent) and runs the one-time
// backfill for rows that predate it. It is safe to run on every boot: the
// backfill touches ONLY rows whose contract is still empty, so a row stamped by
// the live writer is never rewritten, and a second run is a no-op.
//
// The backfill is per (symbol, tf) because the mixed window is a TIME window
// and a bar is mixed if its own [open, open+tf) intersects it — a 5m bar
// opening at 21:15 and a 1h bar opening at 21:00 both straddle the roll even
// though only one of them opens inside it.
func (s *BarHistoryStore) migrateContractColumn() error {
	var has int64
	if err := s.db.Raw("SELECT COUNT(*) FROM pragma_table_info('bars') WHERE name='contract'").Scan(&has).Error; err != nil {
		return err
	}
	if has == 0 {
		if err := s.db.Exec("ALTER TABLE bars ADD COLUMN contract TEXT NOT NULL DEFAULT ''").Error; err != nil {
			return err
		}
	}
	// GORM's AutoMigrate runs BEFORE this and, seeing the struct field, adds
	// the column itself — as nullable TEXT, so every pre-existing row reads
	// NULL rather than ''. The first draft of this backfill filtered on
	// contract = '' and touched zero rows while reporting success. NULL and ''
	// are both "unstamped" and must be one value before anything filters on it.
	if err := s.db.Exec("UPDATE bars SET contract = '' WHERE contract IS NULL").Error; err != nil {
		return err
	}
	if err := s.db.Exec("CREATE INDEX IF NOT EXISTS idx_bars_contract ON bars(symbol, tf, contract, open_time_ms)").Error; err != nil {
		return err
	}
	var tfs []string
	if err := s.db.Raw("SELECT DISTINCT tf FROM bars").Scan(&tfs).Error; err != nil {
		return err
	}
	for _, rb := range roll20260910 {
		for _, tf := range tfs {
			d := tfMillis(tf)
			// MIXED: the bar's window intersects [MixedFromMs, NewFromMs).
			if err := s.db.Exec(`UPDATE bars SET contract = ? WHERE symbol = ? AND tf = ? AND contract = ''
				AND open_time_ms < ? AND open_time_ms + ? > ?`,
				ContractMixed, rb.Symbol, tf, rb.NewFromMs, d, rb.MixedFromMs).Error; err != nil {
				return fmt.Errorf("backfill mixed %s %s: %w", rb.Symbol, tf, err)
			}
			// OLD: wholly before the mixed window.
			if err := s.db.Exec(`UPDATE bars SET contract = ? WHERE symbol = ? AND tf = ? AND contract = ''
				AND open_time_ms + ? <= ?`,
				rb.Old, rb.Symbol, tf, d, rb.MixedFromMs).Error; err != nil {
				return fmt.Errorf("backfill old %s %s: %w", rb.Symbol, tf, err)
			}
			// NEW: wholly at or after the first clean bar.
			if err := s.db.Exec(`UPDATE bars SET contract = ? WHERE symbol = ? AND tf = ? AND contract = ''
				AND open_time_ms >= ?`,
				rb.New, rb.Symbol, tf, rb.NewFromMs).Error; err != nil {
				return fmt.Errorf("backfill new %s %s: %w", rb.Symbol, tf, err)
			}
		}
	}
	return nil
}

// ContractCensus reports, per symbol, how many rows carry each contract label —
// the boot line's figure, READ not literal (A11). Empty-label rows are the ones
// no backfill claimed and no writer stamped.
func (s *BarHistoryStore) ContractCensus(symbol string) (map[string]int64, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	type row struct {
		Contract string
		N        int64
	}
	var rows []row
	if err := s.db.Raw("SELECT contract, COUNT(*) AS n FROM bars WHERE symbol = ? GROUP BY contract", symbol).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Contract] = r.N
	}
	return out, nil
}

// LatestContract returns the contract label of the NEWEST usable bar for a
// symbol — the fallback for "what contract are we on" when the AddOn has not
// yet ACKed a subscription this process. It is the last thing the broker told
// us, which is not a date rule; and it is labelled as a fallback wherever it is
// printed, because the ACK is the source and this is its shadow.
func (s *BarHistoryStore) LatestContract(symbol string) (string, bool) {
	if s == nil || s.db == nil {
		return "", false
	}
	var c string
	err := s.db.Raw(`SELECT contract FROM bars WHERE symbol = ? AND tf = '1m' AND contract <> '' AND contract <> ?
		ORDER BY open_time_ms DESC LIMIT 1`, symbol, ContractMixed).Scan(&c).Error
	if err != nil || strings.TrimSpace(c) == "" {
		return "", false
	}
	return c, true
}

// ContractAt returns the contract label of the newest USABLE 1m bar at or
// before ms — "what contract was the tape on at that moment". ok=false when
// nothing usable exists at or before ms.
//
// Windowed readers use it to ask the honest question: not "the current
// contract" (a trade that closed in September must be measured on September
// bars) but "the contract this window was on". A window whose two ends resolve
// to DIFFERENT contracts straddles a roll and is unrecomputable — see
// WindowContract.
func (s *BarHistoryStore) ContractAt(symbol string, ms int64) (string, bool) {
	if s == nil || s.db == nil {
		return "", false
	}
	var c string
	err := s.db.Raw(`SELECT contract FROM bars WHERE symbol = ? AND tf = '1m' AND open_time_ms <= ?
		AND contract <> '' AND contract <> ? ORDER BY open_time_ms DESC LIMIT 1`, symbol, ms, ContractMixed).Scan(&c).Error
	if err != nil || strings.TrimSpace(c) == "" {
		return "", false
	}
	return c, true
}

// WindowContract resolves ONE contract for [fromMs, toMs]. It is the contract
// at both ends when they agree; when they do not, the window spans a roll and
// the result is ContractMixed with ok=false — the caller must treat the window
// as unrecomputable rather than read a mixed tape through it (A24).
func (s *BarHistoryStore) WindowContract(symbol string, fromMs, toMs int64) (string, bool) {
	a, okA := s.ContractAt(symbol, fromMs)
	b, okB := s.ContractAt(symbol, toMs)
	switch {
	case !okA && !okB:
		return "", false
	case !okA:
		return b, true // the window starts before any stamped bar; its body is on b
	case !okB:
		return a, true
	case a == b:
		return a, true
	}
	return ContractMixed, false
}
