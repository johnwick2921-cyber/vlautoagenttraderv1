package ninjatrader

import (
	"testing"
	"time"
)

// ── ROLL WAVE PINS (server half) ─────────────────────────────────────────────
//
// The boundary is a FIXTURE CONSTANT (A28). 2026-09-10 21:15 CT, epoch ms.
const rollAt int64 = 1789092900000

func rollServer(t *testing.T) *TCPServer {
	t.Helper()
	s := NewTCPServer(nil)
	s.barCache = NewBarCache(2500)
	return s
}

// mixedRing seeds the ring the way the incident left it: September bars up to
// the roll, December bars after, one ~292-point step between them.
func mixedRing(s *TCPServer, symbol string) {
	var bars []Bar
	for i := 0; i < 60; i++ { // 60 September minutes
		ms := rollAt - int64(60-i)*60_000
		bars = append(bars, Bar{T: ms, O: 29130, H: 29132, L: 29128, C: 29131, V: 1})
	}
	for i := 0; i < 10; i++ { // 10 December minutes, +292
		ms := rollAt + int64(180_000) + int64(i)*60_000
		bars = append(bars, Bar{T: ms, O: 29422, H: 29424, L: 29420, C: 29423, V: 1})
	}
	s.barCache.SeedHistorical(symbol, "1m", bars)
}

func rangeOf(bars []Bar) float64 {
	hi, lo := -1e18, 1e18
	for _, b := range bars {
		if b.H > hi {
			hi = b.H
		}
		if b.L < lo {
			lo = b.L
		}
	}
	return hi - lo
}

// E1 — THE GAP PIN. A ring holding bars either side of the roll reads a
// ~294-point range — the roll gap, not the market. After the AddOn names the
// new contract, the ring holds the CURRENT contract only and the range is the
// market's.
//
// RED on the pre-wave server: observeContract did not exist, nothing purged,
// and the range stayed 294.
func TestGapPinRingReadsCurrentContractOnlyAfterRoll(t *testing.T) {
	s := rollServer(t)
	mixedRing(s, "MNQ")
	if r := rangeOf(s.barCache.Get("MNQ", "1m")); r < 200 {
		t.Fatalf("fixture must present the roll gap before the fix acts; range=%.2f", r)
	}
	// The first ACK of the process names September — that is not a roll.
	s.observeContract("MNQ", "MNQ 09-26", time.UnixMilli(rollAt-3_600_000))
	if n := s.barCache.Count("MNQ", "1m"); n != 70 {
		t.Fatalf("the first ACK must not purge anything (no previous contract); ring=%d", n)
	}
	// The reconnect ACK names December: THIS is the roll.
	s.observeContract("MNQ", "MNQ 12-26", time.UnixMilli(rollAt))
	if n := s.barCache.Count("MNQ", "1m"); n != 0 {
		t.Fatalf("a roll must PURGE the symbol's ring; %d bar(s) survived", n)
	}
	// The AddOn's post-subscribe replay refills it on the new contract.
	var dec []Bar
	for i := 0; i < 10; i++ {
		ms := rollAt + int64(180_000) + int64(i)*60_000
		dec = append(dec, Bar{T: ms, O: 29422, H: 29424, L: 29420, C: 29423, V: 1})
	}
	s.barCache.SeedHistorical("MNQ", "1m", dec)
	got := s.barCache.Get("MNQ", "1m")
	for _, b := range got {
		if b.T < rollAt {
			t.Fatalf("a September bar (%d) is back in the ring after the roll", b.T)
		}
	}
	if r := rangeOf(got); r > 50 {
		t.Fatalf("range after the roll must be the market's, not the gap: %.2f", r)
	}
	f, ok := s.CurrentContract("MNQ")
	if !ok || f.Contract != "MNQ 12-26" || f.Previous != "MNQ 09-26" {
		t.Fatalf("CurrentContract after the roll: %+v ok=%v", f, ok)
	}
}

// E3 — ROLL PIN. A new instrument on the ACK → purged, recorded, listeners
// notified exactly once. The same instrument again → nothing at all.
func TestRollFiresOnceAndRepeatedAckIsSilent(t *testing.T) {
	s := rollServer(t)
	mixedRing(s, "MNQ")
	fired := 0
	var gotFrom, gotTo string
	OnContractRoll(func(sym, from, to string, at time.Time) { fired++; gotFrom, gotTo = from, to })
	t.Cleanup(func() { rollListenersMu.Lock(); rollListeners = nil; rollListenersMu.Unlock() })

	s.observeContract("MNQ", "MNQ 09-26", time.UnixMilli(rollAt-60_000))
	s.observeContract("MNQ", "MNQ 09-26", time.UnixMilli(rollAt-30_000)) // reconnect, same contract
	if fired != 0 || s.barCache.Count("MNQ", "1m") != 70 {
		t.Fatalf("a repeated ACK naming the same contract must do nothing; fired=%d ring=%d", fired, s.barCache.Count("MNQ", "1m"))
	}
	s.observeContract("MNQ", "MNQ 12-26", time.UnixMilli(rollAt))
	if fired != 1 || gotFrom != "MNQ 09-26" || gotTo != "MNQ 12-26" {
		t.Fatalf("the roll must fire ONCE with both names; fired=%d from=%q to=%q", fired, gotFrom, gotTo)
	}
	s.observeContract("MNQ", "MNQ 12-26", time.UnixMilli(rollAt+60_000)) // reconnect after the roll
	if fired != 1 {
		t.Fatalf("a repeated ACK after the roll must not re-fire; fired=%d", fired)
	}
	f, _ := s.CurrentContract("MNQ")
	if !f.RolledAt.Equal(time.UnixMilli(rollAt)) {
		t.Fatalf("the roll timestamp must be the ACK's receipt, got %v", f.RolledAt)
	}
}

// E5 — SOURCE PIN. CurrentContract reads the FRAME. With no ACK there is no
// contract — not a date-derived one, not a config literal, nothing.
func TestCurrentContractComesOnlyFromTheFrame(t *testing.T) {
	s := rollServer(t)
	if f, ok := s.CurrentContract("MNQ"); ok {
		t.Fatalf("no frame has named a contract, yet CurrentContract answered %+v — that answer came from somewhere other than the AddOn", f)
	}
	// A pending or errored subscription is not a named contract either — EVEN
	// IF a contract string is present on the state. The STATE gates, not the
	// string: a mutation that dropped the state check survived the first draft
	// of this pin because every non-subscribed case it tried also had an empty
	// contract, so the second clause masked the first.
	s.setSubState("MNQ", "pending", "MNQ 12-26", "")
	if _, ok := s.CurrentContract("MNQ"); ok {
		t.Fatal("a PENDING subscription names nothing, whatever string sits beside it")
	}
	s.setSubState("MNQ", "error", "MNQ 12-26", "no such instrument")
	if _, ok := s.CurrentContract("MNQ"); ok {
		t.Fatal("an ERRORED subscription names nothing, whatever string sits beside it")
	}
	s.setSubState("MNQ", "subscribed", "", "")
	if _, ok := s.CurrentContract("MNQ"); ok {
		t.Fatal("a subscribed state with no contract string names nothing")
	}
	// Only the ACK does, and it carries its receipt time.
	before := time.Now()
	s.setSubState("MNQ", "subscribed", "MNQ 12-26", "")
	f, ok := s.CurrentContract("MNQ")
	if !ok || f.Contract != "MNQ 12-26" || f.Source != "subscribed" || f.ReceivedAt.Before(before) {
		t.Fatalf("the ACK must be the source, with its receipt time: %+v ok=%v", f, ok)
	}
}
