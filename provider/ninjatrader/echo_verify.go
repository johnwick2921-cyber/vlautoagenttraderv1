package ninjatrader

import (
	"net"
	"sort"
	"time"

	"nofx/discipline"
)

// A2 (G1) — echo verification. Every order/modify/cancel frame the Go server sends
// is registered as a pendingOp keyed by its monotonic seq; the AddOn echoes
// (trader_id, account, seq) on the paired ack/fill/close/reject. checkEcho compares
// the echo to the registered op so a fill/close for the WRONG trader/account — or a
// forged/unknown seq — is caught before it is processed.

// pendingOp is the identity the server stamped on an outbound op.
type pendingOp struct {
	traderID string
	account  string
	signalID string
}

// checkEcho compares an inbound echo against the registered pending op.
//
//   - pre-v3 AddOn (echoSeq == 0): no echo to verify → accept, no mismatch. This
//     keeps the lockstep deploy window safe (new Go + not-yet-F5'd C#).
//   - echoSeq > 0 AND a registered op whose trader_id AND account both match → accept.
//   - echoSeq > 0 AND a registered op with a DIFFERENT trader_id or account →
//     MISMATCH (a fill/close came back for the wrong originator — freeze).
//   - echoSeq > 0 but NO registered op (!found): TOLERATED with a warning (accept,
//     no mismatch). A retired op (a duplicate/late frame after its close retired the
//     entry) or a reconnect leaves legitimate echoes "unknown"; a false freeze halts
//     a live trader, so a bookkeeping gap must NOT freeze. A4's direct account check
//     (fill/close account ≠ the trader's bound account) is the hard net for a truly
//     cross-wired fill. `reason` is non-empty here so the caller logs the warning.
//
// A mismatch means: do NOT process the frame + 🚨 + freeze the owning trader (A4).
func checkEcho(pending pendingOp, found bool, echoSeq uint64, echoTraderID, echoAccount string) (accept, mismatch bool, owner, reason string) {
	if echoSeq == 0 {
		return true, false, "", "" // legacy / deploy window — nothing to verify
	}
	if !found {
		return true, false, echoTraderID, "unknown/retired seq — tolerated (bookkeeping gap, not a freeze)"
	}
	if pending.traderID != echoTraderID {
		return false, true, pending.traderID, "trader_id echo " + q(echoTraderID) + " ≠ pending " + q(pending.traderID)
	}
	if pending.account != echoAccount {
		return false, true, pending.traderID, "account echo " + q(echoAccount) + " ≠ pending " + q(pending.account)
	}
	return true, false, pending.traderID, ""
}

func q(s string) string { return "\"" + s + "\"" }

// RegisterBoundAccount (A3/G2) adds account to the session allowlist and, if a
// client is connected, immediately re-declares the full set to the AddOn so a
// late-binding trader's account is enforced without waiting for a reconnect. Empty
// account is a no-op.
func (s *TCPServer) RegisterBoundAccount(account string) {
	if account == "" {
		return
	}
	s.boundAcctMu.Lock()
	if s.boundAccounts == nil {
		s.boundAccounts = make(map[string]bool)
	}
	if s.boundAccounts[account] {
		s.boundAcctMu.Unlock()
		return // already declared — no frame needed
	}
	s.boundAccounts[account] = true
	s.boundAcctMu.Unlock()

	s.connMu.Lock()
	c := s.conn
	s.connMu.Unlock()
	if c != nil {
		s.sendAccountAllowlist(c)
	}
}

// sendAccountAllowlist declares the current bound-account allowlist to the AddOn.
// Best-effort: a failure here just means the AddOn keeps its prior set until the
// next connect/register.
func (s *TCPServer) sendAccountAllowlist(c net.Conn) {
	s.boundAcctMu.Lock()
	accts := make([]string, 0, len(s.boundAccounts))
	for a := range s.boundAccounts {
		accts = append(accts, a)
	}
	s.boundAcctMu.Unlock()
	if len(accts) == 0 {
		return // nothing declared yet — AddOn stays fail-open (legacy)
	}
	sort.Strings(accts) // deterministic frame
	s.writeMu.Lock()
	_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := WriteFrame(c, FrameAccountRegister, AccountRegisterPayload{Accounts: accts})
	s.writeMu.Unlock()
	if err != nil {
		s.logger.Warn("tcp_server: send account_register", "err", err)
		return
	}
	s.logger.Info("tcp_server: sent account_register allowlist", "accounts", accts)
}

// assignSeqRegister assigns the next monotonic seq to an outbound op and registers
// its identity for later echo-verify. Returns the seq to stamp on the frame.
func (s *TCPServer) assignSeqRegister(traderID, account, signalID string) uint64 {
	seq := s.seqCounter.Add(1)
	s.pendingMu2.Lock()
	if s.pendingOps == nil {
		s.pendingOps = make(map[uint64]pendingOp)
		s.pendingBySig = make(map[string]uint64)
	}
	s.pendingOps[seq] = pendingOp{traderID: traderID, account: account, signalID: signalID}
	if signalID != "" {
		s.pendingBySig[signalID] = seq
	}
	// Drop-oldest backstop so an un-retired run can't grow the registry unbounded.
	if len(s.pendingOps) > pendingOpsCap {
		oldest, first := uint64(0), true
		for k := range s.pendingOps {
			if first || k < oldest {
				oldest, first = k, false
			}
		}
		if op, ok := s.pendingOps[oldest]; ok {
			delete(s.pendingBySig, op.signalID)
		}
		delete(s.pendingOps, oldest)
	}
	s.pendingMu2.Unlock()
	return seq
}

// lookupPending resolves the pending op for an inbound echo. It prefers seq (the v3
// identity); if seq==0 (pre-v3 AddOn) it falls back to signal_id so a fill still
// correlates — checkEcho then treats seq==0 as legacy-tolerated.
func (s *TCPServer) lookupPending(seq uint64, signalID string) (pendingOp, bool) {
	s.pendingMu2.Lock()
	defer s.pendingMu2.Unlock()
	if seq != 0 {
		op, ok := s.pendingOps[seq]
		return op, ok
	}
	if signalID != "" {
		if seq2, ok := s.pendingBySig[signalID]; ok {
			return s.pendingOps[seq2], true
		}
	}
	return pendingOp{}, false
}

// retirePending drops a completed op (terminal frames: fill, position_close,
// position_close_rejected).
func (s *TCPServer) retirePending(seq uint64, signalID string) {
	s.pendingMu2.Lock()
	defer s.pendingMu2.Unlock()
	if seq != 0 {
		if op, ok := s.pendingOps[seq]; ok {
			delete(s.pendingBySig, op.signalID)
		}
		delete(s.pendingOps, seq)
		return
	}
	if signalID != "" {
		if seq2, ok := s.pendingBySig[signalID]; ok {
			delete(s.pendingOps, seq2)
			delete(s.pendingBySig, signalID)
		}
	}
}

// verifyInbound is the read-loop hook: resolve the pending op, run checkEcho, and on
// a MISMATCH freeze the owning trader (A4) + 🚨 and return process=false. Pre-v3
// (seq==0) is tolerated. On accept it returns true (caller processes + retires).
func (s *TCPServer) verifyInbound(kind string, echoSeq uint64, echoTraderID, echoAccount, signalID string) bool {
	pending, found := s.lookupPending(echoSeq, signalID)
	accept, mismatch, owner, reason := checkEcho(pending, found, echoSeq, echoTraderID, echoAccount)
	if mismatch {
		if discipline.FreezeTrader(owner, "echo mismatch on "+kind+": "+reason, time.Now().UnixMilli()) {
			s.logger.Error("🚨 A2 echo-verify MISMATCH — FREEZING trader; frame NOT processed",
				"kind", kind, "owner", owner, "reason", reason, "seq", echoSeq, "signal_id", signalID)
		}
		return false
	}
	if reason != "" { // tolerated-but-unverified (unknown/retired seq)
		s.logger.Warn("A2 echo-verify: "+reason, "kind", kind, "seq", echoSeq, "signal_id", signalID)
	}
	return accept
}
