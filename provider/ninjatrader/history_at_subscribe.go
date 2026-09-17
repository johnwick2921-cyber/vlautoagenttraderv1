// ── 101 E1 — what NT8 actually delivered at subscribe, per timeframe, READ ──
//
// The 09-16 diagnosis took hours because the only delivery count on the Go
// side (📼 historical=139) was a STORE census misread as a delivery count,
// while the AddOn's own `emitted bars_historical … bars=2000` lines sat in the
// NT8 log on the Windows side. This records, per (symbol, tf), the size of the
// LAST bars_historical frame received against the bars_back that was asked,
// so one boot line answers "did the history arrive" without leaving Go.
package ninjatrader

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type historyAtSubscribe struct {
	mu       sync.Mutex
	received map[string]int // key symbol|tf -> bars in the last historical frame
	frames   map[string]int // key -> historical frames seen
}

func (h *historyAtSubscribe) note(symbol, tf string, n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.received == nil {
		h.received = map[string]int{}
		h.frames = map[string]int{}
	}
	k := barKey(symbol, tf)
	h.received[k] = n
	h.frames[k]++
}

// HistoryAtSubscribe returns, per symbol×tf, bars in the last historical frame
// and how many historical frames have been seen (a reconnect that delivered
// bars=0 is a frame too, and shows as such).
func (s *TCPServer) HistoryAtSubscribe() (received map[string]int, frames map[string]int) {
	s.histAtSub.mu.Lock()
	defer s.histAtSub.mu.Unlock()
	received = make(map[string]int, len(s.histAtSub.received))
	frames = make(map[string]int, len(s.histAtSub.frames))
	for k, v := range s.histAtSub.received {
		received[k] = v
	}
	for k, v := range s.histAtSub.frames {
		frames[k] = v
	}
	return
}

// BarsBackAsked is the bars_back the subscribe frame carried (A11: read).
func (s *TCPServer) BarsBackAsked() int { return s.barsSubscribe.BarsBack }

// HistoryAtSubscribeLine renders the E1 summary for one symbol: every
// subscribed tf, received/asked, in ladder order. A tf with no frame yet reads
// n/a — never 0 (A24: a zero is a measurement; an absence is not).
func HistoryAtSubscribeLine(symbol string, asked int, received, frames map[string]int, tfs []string) string {
	parts := make([]string, 0, len(tfs))
	zero := 0
	for _, tf := range tfs {
		k := barKey(symbol, tf)
		n, ok := received[k]
		switch {
		case !ok:
			parts = append(parts, fmt.Sprintf("%s=n/a/%d", tf, asked))
		default:
			parts = append(parts, fmt.Sprintf("%s=%d/%d", tf, n, asked))
			if n == 0 {
				zero++
			}
		}
	}
	tail := ""
	if zero > 0 {
		tail = fmt.Sprintf(" · %d tf(s) received ZERO on the last frame — a BarsRequest that ran while the feed was down (09-16 shape)", zero)
	}
	return fmt.Sprintf("🧯 nt8 history at subscribe: %s %s%s", symbol, strings.Join(parts, " "), tail)
}

// ladderOrder sorts timeframes by their minute span for a stable line.
func ladderOrder(tfs []string) []string {
	out := append([]string(nil), tfs...)
	sort.SliceStable(out, func(i, j int) bool { return timeframeMs(out[i]) < timeframeMs(out[j]) })
	return out
}

// SubscribedTimeframes is the subscribe frame's timeframe list, in ladder
// order (A11: read from the frame, not retyped).
func (s *TCPServer) SubscribedTimeframes() []string {
	return ladderOrder(s.barsSubscribe.Timeframes)
}

// HistoryAtSubscribeLineFor renders E1 for one symbol from this server's own
// records. THE PRODUCTION CALL PATH is the contract boot block in
// trader/ninjatrader/bar_persist_wire.go.
func (s *TCPServer) HistoryAtSubscribeLineFor(symbol string) string {
	received, frames := s.HistoryAtSubscribe()
	return HistoryAtSubscribeLine(symbol, s.BarsBackAsked(), received, frames, s.SubscribedTimeframes())
}
