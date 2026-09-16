package kernel

import (
	"testing"

	"nofx/market"
)

// TRADE EXCURSION LOGGING (wave 1A) — F1, THE PIN.
//
// Position 589: MNQ LONG, filled 29192.50 at 1788360064833 (2026-09-02
// 14:41:04 UTC), stopped out at 29115.00 at 1788361167398 (14:59:27). The
// decision that opened it (decision_records 36394) carried stop_loss 29115 and
// take_profit 29317.25 — a 77.5 pt stop, the widened one the forensic names.
//
// These are the nineteen 1m bars the store holds for the hold, verbatim:
//
//	sqlite3 data/data.db "select open_time_ms,o,h,l,c from bars
//	  where symbol='MNQ' and tf='1m'
//	    and open_time_ms between 1788360060000 and 1788361140000"
//
// The recorded row says mae=80.5. Rebuilt from those closed bars it is 81.25 —
// see TestExcursion589AgainstTheRecordedRow for that 0.75.
var bars589 = []market.Kline{
	{OpenTime: 1788360060000, Open: 29193.75, High: 29197.50, Low: 29178.75, Close: 29182.00},
	{OpenTime: 1788360120000, Open: 29181.75, High: 29200.75, Low: 29181.50, Close: 29186.00},
	{OpenTime: 1788360180000, Open: 29186.00, High: 29196.75, Low: 29182.00, Close: 29194.50},
	{OpenTime: 1788360240000, Open: 29195.00, High: 29202.75, Low: 29192.00, Close: 29201.50},
	{OpenTime: 1788360300000, Open: 29201.75, High: 29202.50, Low: 29182.50, Close: 29183.25},
	{OpenTime: 1788360360000, Open: 29183.00, High: 29185.00, Low: 29175.00, Close: 29176.00},
	{OpenTime: 1788360420000, Open: 29176.75, High: 29179.50, Low: 29171.25, Close: 29173.25},
	{OpenTime: 1788360480000, Open: 29173.25, High: 29175.50, Low: 29159.00, Close: 29160.25},
	{OpenTime: 1788360540000, Open: 29159.75, High: 29162.25, Low: 29149.00, Close: 29160.50},
	{OpenTime: 1788360600000, Open: 29160.50, High: 29160.50, Low: 29140.00, Close: 29140.25},
	{OpenTime: 1788360660000, Open: 29140.00, High: 29154.50, Low: 29139.50, Close: 29151.50},
	{OpenTime: 1788360720000, Open: 29151.25, High: 29153.00, Low: 29140.75, Close: 29142.50},
	{OpenTime: 1788360780000, Open: 29142.75, High: 29143.75, Low: 29128.25, Close: 29132.75},
	{OpenTime: 1788360840000, Open: 29132.75, High: 29136.25, Low: 29120.25, Close: 29129.00},
	{OpenTime: 1788360900000, Open: 29130.25, High: 29140.75, Low: 29128.00, Close: 29133.00},
	{OpenTime: 1788360960000, Open: 29132.50, High: 29136.25, Low: 29122.75, Close: 29126.75},
	{OpenTime: 1788361020000, Open: 29126.75, High: 29136.00, Low: 29123.00, Close: 29135.00},
	{OpenTime: 1788361080000, Open: 29135.25, High: 29137.25, Low: 29116.00, Close: 29120.75},
	{OpenTime: 1788361140000, Open: 29120.75, High: 29121.75, Low: 29111.25, Close: 29120.25},
}

const (
	pos589Entry  = 29192.50
	pos589Stop   = 29115.00
	pos589Target = 29317.25
	pos589Enter  = int64(1788360064833)
	pos589Exit   = int64(1788361167398)
)

func TestExcursionPin589(t *testing.T) {
	got := ComputePathExcursion(pos589Entry, "LONG", pos589Stop, pos589Target,
		bars589, pos589Enter, pos589Exit, "1m")

	if !got.Computed {
		t.Fatal("589 has full 1m coverage — the path must be computed, not abandoned")
	}
	// Adverse extreme: low 29111.25 in the 14:59 bar → 81.25 pts, 18 bars in.
	if got.MAEPts != 81.25 {
		t.Errorf("MAE = %v, want 81.25 (entry 29192.50 − low 29111.25)", got.MAEPts)
	}
	if got.MAETs != 1788361140000 || got.MAEBars != 18 {
		t.Errorf("MAE at ts=%d bar=%d, want ts=1788361140000 bar=18", got.MAETs, got.MAEBars)
	}
	// Favorable extreme: high 29202.75 in the 14:44 bar → 10.25 pts, 3 bars in.
	if got.MFEPts != 10.25 {
		t.Errorf("MFE = %v, want 10.25 (high 29202.75 − entry 29192.50)", got.MFEPts)
	}
	if got.MFETs != 1788360240000 || got.MFEBars != 3 {
		t.Errorf("MFE at ts=%d bar=%d, want ts=1788360240000 bar=3", got.MFETs, got.MFEBars)
	}
	// The fill landed 4.8s into the 14:41 bar. That bar is part of the hold.
	if got.BarsHeld != 19 {
		t.Errorf("bars_held = %d, want 19 — the entry bar is INCLUDED (the old computation dropped it)", got.BarsHeld)
	}
	// Stop 29115 to target 29317.25 is 202.25 points; no 1m bar here is close.
	if got.AmbiguousBars != 0 {
		t.Errorf("ambiguous_bars = %d, want 0 — no bar spans a 202.25 pt range", got.AmbiguousBars)
	}
	if got.Resolution != "1m" {
		t.Errorf("resolution = %q, want \"1m\"", got.Resolution)
	}
}

// The 0.75 the recorded row is missing. Kept as its own test so the number is
// asserted rather than described in a comment nobody runs.
func TestExcursion589AgainstTheRecordedRow(t *testing.T) {
	const recordedMAE = 80.5 // trader_positions.mae for id=589
	got := ComputePathExcursion(pos589Entry, "LONG", pos589Stop, pos589Target,
		bars589, pos589Enter, pos589Exit, "1m")
	if got.MAEPts <= recordedMAE {
		t.Fatalf("the closed tape must be at least as adverse as the row recorded live: got %v, row %v", got.MAEPts, recordedMAE)
	}
	if d := got.MAEPts - recordedMAE; d != 0.75 {
		t.Errorf("recorded-vs-tape gap = %v, want 0.75 — the live computation read the final bar while it was still forming (low 29112.00 then, 29111.25 closed)", d)
	}
	// MFE landed mid-hold, long before the close, so it has no such gap.
	if got.MFEPts != 10.25 {
		t.Errorf("MFE = %v, want 10.25 — matches the recorded row exactly", got.MFEPts)
	}
}

// F2 — side awareness, and the entry bar counted. A synthetic path where the
// adverse and favorable moves are unmistakably on opposite sides.
func TestExcursionSideAwareAndEntryBarIncluded(t *testing.T) {
	// entry lands 30s into the first bar, which prints the LOW of the whole run.
	bars := []market.Kline{
		{OpenTime: 1_000_000, Open: 100, High: 102, Low: 90, Close: 101}, // entry bar
		{OpenTime: 1_060_000, Open: 101, High: 115, Low: 99, Close: 114},
		{OpenTime: 1_120_000, Open: 114, High: 116, Low: 108, Close: 110},
	}
	const entry = 100.0
	entryMs := int64(1_030_000) // half-way through bar 0
	exitMs := int64(1_150_000)

	long := ComputePathExcursion(entry, "LONG", 0, 0, bars, entryMs, exitMs, "1m")
	if long.MAEPts != 10 || long.MAEBars != 0 {
		t.Errorf("LONG: MAE %v at bar %d, want 10 at bar 0 — the entry bar's low 90 IS the adverse extreme and the old code dropped it",
			long.MAEPts, long.MAEBars)
	}
	if long.MFEPts != 16 || long.MFEBars != 2 {
		t.Errorf("LONG: MFE %v at bar %d, want 16 at bar 2 (high 116)", long.MFEPts, long.MFEBars)
	}

	short := ComputePathExcursion(entry, "SHORT", 0, 0, bars, entryMs, exitMs, "1m")
	if short.MAEPts != 16 || short.MAEBars != 2 {
		t.Errorf("SHORT: MAE %v at bar %d, want 16 at bar 2 — up is adverse for a short", short.MAEPts, short.MAEBars)
	}
	if short.MFEPts != 10 || short.MFEBars != 0 {
		t.Errorf("SHORT: MFE %v at bar %d, want 10 at bar 0", short.MFEPts, short.MFEBars)
	}
	if long.BarsHeld != 3 || short.BarsHeld != 3 {
		t.Errorf("bars_held %d/%d, want 3 each", long.BarsHeld, short.BarsHeld)
	}
}

// F3 — a bar reaching BOTH the stop and the target. The tape carries no
// ordering inside a bar, so the row is flagged and the exit is read AGAINST
// the trade (the confirm-cost study's rule): the stop, never the target.
func TestExcursionAmbiguousBarResolvesAgainstTheTrade(t *testing.T) {
	bars := []market.Kline{
		{OpenTime: 2_000_000, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: 2_060_000, Open: 100, High: 120, Low: 80, Close: 95}, // spans both
	}
	const entry, stop, target = 100.0, 90.0, 110.0

	got := ComputePathExcursion(entry, "LONG", stop, target, bars, 2_000_000, 2_120_000, "1m")
	if got.AmbiguousBars != 1 {
		t.Fatalf("ambiguous_bars = %d, want 1 — bar 2 ran 80..120 through both 90 and 110", got.AmbiguousBars)
	}

	px, amb := ResolveAmbiguousExit("LONG", stop, target, bars[1])
	if !amb {
		t.Fatal("the spanning bar must report itself ambiguous")
	}
	if px != stop {
		t.Errorf("resolved exit = %v, want the STOP %v — ambiguity resolves against the trade", px, stop)
	}
	pxS, ambS := ResolveAmbiguousExit("SHORT", 110, 90, bars[1]) // short: stop above
	if !ambS || pxS != 110 {
		t.Errorf("SHORT resolved exit = %v (amb=%v), want the stop 110", pxS, ambS)
	}

	// A bar that reaches neither, or only one, is not ambiguous and resolves to
	// nothing — the caller keeps the real fill.
	if _, amb := ResolveAmbiguousExit("LONG", stop, target, bars[0]); amb {
		t.Error("a bar touching neither level is not ambiguous")
	}
}

// A hold with no bar coverage must produce NOTHING, not a confident zero.
func TestExcursionNoCoverageIsNotZero(t *testing.T) {
	got := ComputePathExcursion(100, "LONG", 90, 110, nil, 1, 2, "1m")
	if got.Computed || got.Resolution != "none" {
		t.Fatalf("no bars must yield computed=false resolution=none, got %+v", got)
	}
	far := []market.Kline{{OpenTime: 9_000_000, Open: 100, High: 101, Low: 99, Close: 100}}
	if g := ComputePathExcursion(100, "LONG", 90, 110, far, 1_000_000, 1_060_000, "1m"); g.Computed {
		t.Fatalf("bars outside the hold are no coverage at all, got %+v", g)
	}
}
