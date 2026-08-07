package ninjatrader

import (
	"strings"
	"sync"
)

// Priced-close fallback cache.
//
// A position_close frame carries the REAL exit price, but it routes by SYMBOL to
// ONE trader's close-sync (dispatchBySymbol). When owner-routing still finds no open
// row (a frame-before-entry race, or a row already gone), the price would otherwise
// be lost and reconcile would fabricate exit=entry pnl=0. Instead the price is parked
// here; reconcile's orphan-close consults it before writing a fake 0.
//
// Package-level + mutex-guarded on purpose: the frame can be received by one trader's
// close-sync goroutine while a DIFFERENT trader's reconcile goroutine (the row owner)
// is the one that eventually closes the row.

type pricedClose struct {
	exitPrice float64
	qty       float64
	tsMs      int64
}

var (
	pricedCloseMu    sync.Mutex
	pricedCloseByKey = map[string]pricedClose{}
)

// pricedCloseGraceMs bounds retention: past it, a parked price is too stale to trust
// for a reconcile that only just observed the row flat.
const pricedCloseGraceMs = 120_000 // 2 minutes

func pricedCloseKey(account, symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(account)) + "|" +
		strings.ToUpper(strings.TrimSpace(symbol)) + "|" +
		strings.ToUpper(strings.TrimSpace(side))
}

// putPricedClose parks an unmatched priced close for later reconcile consumption.
func putPricedClose(account, symbol, side string, exitPrice, qty float64, tsMs int64) {
	if exitPrice <= 0 {
		return
	}
	pricedCloseMu.Lock()
	pricedCloseByKey[pricedCloseKey(account, symbol, side)] = pricedClose{exitPrice, qty, tsMs}
	pricedCloseMu.Unlock()
}

// takePricedClose returns and REMOVES a parked exit price for (account,symbol,side)
// if one arrived within the grace window. Idempotent: a second reconcile gets nothing
// (so the same priced close is consumed exactly once).
func takePricedClose(account, symbol, side string, nowMs int64) (float64, bool) {
	key := pricedCloseKey(account, symbol, side)
	pricedCloseMu.Lock()
	defer pricedCloseMu.Unlock()
	pc, ok := pricedCloseByKey[key]
	if !ok {
		return 0, false
	}
	delete(pricedCloseByKey, key)
	if nowMs-pc.tsMs > pricedCloseGraceMs {
		return 0, false // stale
	}
	return pc.exitPrice, true
}
