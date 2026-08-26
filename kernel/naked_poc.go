package kernel

import (
	"time"

	"nofx/market"
)

// P1.3 — naked-POC detector + provider seam.
//
// A prior session's Point of Control is "naked" (untested) until price trades
// BACK THROUGH it. Once a later session's bar brackets the POC (low ≤ POC ≤
// high), it is consumed. Naked POCs are strong magnets; daily-rank < weekly-rank
// (weekly nPOCs get the HTF grade bump).
//
// The stored profiles live in the DB (store layer); the kernel is DB-blind, so
// the trader layer reads store.SessionProfile, converts rows to PriorPOC, and
// wires the result through NakedPOCProvider (mirror of market.FuturesBarsProvider).

// PriorPOC is one stored session's POC for the naked-POC scan.
type PriorPOC struct {
	SessionDate string  // YYYY-MM-DD CME session-day of origin
	POC         float64 // point of control price
	Weekly      bool    // true → weekly-rank POC (older/composite) → HTF grade
}

// NakedPOCProvider, when set by the trader layer, returns the naked POCs for a
// symbol as DetectedLevels (already filtered to untested). nil → no nPOC input.
var NakedPOCProvider func(symbol string) []DetectedLevel

// NakedPOCs returns the untested POCs among `pocs` given the recent `bars` — a
// POC is dropped once a bar from a LATER session brackets it. Emitted as
// KindNPOC DetectedLevels (weekly → HTF).
func NakedPOCs(pocs []PriorPOC, bars []market.Kline, now time.Time) []DetectedLevel {
	nowMs := now.UnixMilli()
	var out []DetectedLevel
	for _, p := range pocs {
		if p.POC <= 0 || p.SessionDate == "" {
			continue
		}
		tested := false
		for i := range bars {
			b := bars[i]
			if b.CloseTime >= nowMs {
				continue
			}
			bDate := CMESessionDayKey(time.UnixMilli(b.OpenTime))
			if bDate > p.SessionDate && b.Low <= p.POC && b.High >= p.POC {
				tested = true
				break
			}
		}
		if tested {
			continue
		}
		label := "nPOC·" + p.SessionDate
		if p.Weekly {
			label = "nPOC·wk·" + p.SessionDate
		}
		out = append(out, DetectedLevel{
			Kind:       KindNPOC,
			Price:      p.POC,
			Lo:         p.POC,
			Hi:         p.POC,
			Label:      label,
			OriginDate: p.SessionDate,
			HTF:        p.Weekly,
		})
	}
	return out
}
