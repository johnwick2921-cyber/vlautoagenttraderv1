package ninjatrader

import "strings"

// W-EXEC-TRUTH W0 (d) — CLASS 160: entry evidence keyed by SIGNAL.
//
// The AI path confirmed its entries through GetOrderStatus, which correlates
// on the single shared lastEntrySignalID — every entry function overwrites it,
// so an armed or Picture fill landing inside the AI's poll became the AI's
// fill, and a broker REJECT was never visible at all (the reject branch clears
// the slot and drops the fill). The recent-fill ring (RecentFillFor) already
// answers "did THIS signal fill"; this ring answers "was THIS signal
// rejected". Both are process-local and bounded; absence is "no evidence yet",
// never a fill.

const recentRejectCap = 32

type recentReject struct {
	SignalID string
	Reason   string
}

// recordRecentReject retains a broker rejection for its signal. Guarded by mu.
func (t *TCPTrader) recordRecentReject(signalID, reason string) {
	if strings.TrimSpace(signalID) == "" {
		return
	}
	t.recentRejects = append(t.recentRejects, recentReject{SignalID: signalID, Reason: reason})
	if len(t.recentRejects) > recentRejectCap {
		t.recentRejects = t.recentRejects[len(t.recentRejects)-recentRejectCap:]
	}
}

// RecentRejectFor reports whether NT8 rejected the entry sent under signalID
// since this process booted, with the broker's reason ("" when the AddOn sent
// none). ok=false means no rejection was received for that signal.
func (t *TCPTrader) RecentRejectFor(signalID string) (reason string, ok bool) {
	if strings.TrimSpace(signalID) == "" {
		return "", false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := len(t.recentRejects) - 1; i >= 0; i-- {
		if t.recentRejects[i].SignalID == signalID {
			return t.recentRejects[i].Reason, true
		}
	}
	return "", false
}
