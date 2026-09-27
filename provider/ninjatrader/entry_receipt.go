package ninjatrader

import (
	"strings"
	"time"
)

// entryReceiptState shares execution knowledge with the account snapshot's
// lifetime. Adapter replacement must not forget which empty snapshots predate
// entry execution, or treat replayed cumulative evidence as a new execution.
// Protected by TCPServer.acctMu.
type entryReceiptState struct {
	received   time.Time
	quantities map[string]int
}

// NoteEntryExecution records positive entry-execution evidence for one signal
// on one (symbol, account). Only growth of a cumulative observation advances
// the fence — repeated old frames cannot invalidate a newer broker snapshot
// forever.
func (s *TCPServer) NoteEntryExecution(symbol, account, signal string, quantity int) {
	if strings.TrimSpace(symbol) == "" || strings.TrimSpace(account) == "" || signal == "" || quantity <= 0 {
		return
	}
	key := subKey(symbol, account)
	s.acctMu.Lock()
	defer s.acctMu.Unlock()
	if s.entryReceipts == nil {
		s.entryReceipts = make(map[string]*entryReceiptState)
	}
	r := s.entryReceipts[key]
	if r == nil {
		r = &entryReceiptState{quantities: make(map[string]int)}
		s.entryReceipts[key] = r
	}
	if quantity > r.quantities[signal] {
		r.quantities[signal] = quantity
		r.received = time.Now()
	}
}

// PositionsForExecutionReceipt returns a single consistent view of account
// positions, their local receipt time, and the symbol's latest entry receipt.
// The receipt and cumulative dedup persist through socket reconnect and owner
// replacement. A new server has neither a snapshot nor a receipt: unknown.
func (s *TCPServer) PositionsForExecutionReceipt(account, symbol string) ([]OpenPosition, time.Time, time.Time, bool) {
	s.acctMu.RLock()
	defer s.acctMu.RUnlock()
	var entry time.Time
	if r := s.entryReceipts[subKey(symbol, account)]; r != nil {
		entry = r.received
	}
	if account == "" {
		return nil, time.Time{}, entry, false
	}
	// W117 F1 — key present with a nil slice keeps TODAY's known-flat
	// convention (a snapshot WAS received, empty book). The nil=unknown
	// refinement is F4's job with its own RED.
	v, ok := s.acctPositions[account]
	if !ok {
		return nil, time.Time{}, entry, false
	}
	out := make([]OpenPosition, len(v))
	copy(out, v)
	return out, s.acctPositionsReceived[account], entry, true
}
