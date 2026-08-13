package ninjatrader

import (
	"fmt"
	"sync"
)

// B3 — duplicate-order guard + rate limiter. Two structural protections at the
// order-submission chokepoint:
//   - DUPE GUARD: an identical order action (idempotence key: account|side|symbol|
//     qty) repeated within ~one bar is DROPPED — a replayed / double-fired signal
//     can never place a second order in the same bar.
//   - RATE LIMITER: if more than orderRateMax order-actions fire within
//     orderRateWindow, a breaker trips and halts new actions (🚨) until the window
//     drains — a runaway loop can't machine-gun the broker.

const (
	orderDedupWindowMs = 55_000 // drop an identical order within ~1 (1-minute) bar
	orderRateWindowMs  = 60_000 // rate-limit sliding window
	orderRateMax       = 10     // max order-actions per window before the breaker trips
)

type orderGuard struct {
	mu          sync.Mutex
	lastSeen    map[string]int64 // idempotence key → last-admitted ms
	actionTimes []int64          // admitted-action timestamps within the rate window
}

func newOrderGuard() *orderGuard {
	return &orderGuard{lastSeen: make(map[string]int64)}
}

// admit reports whether an order action with the given idempotence key may be sent
// at nowMs. ok==false → the action MUST be dropped; reason explains why (duplicate
// or rate breaker). On admit, the key + timestamp are recorded.
func (g *orderGuard) admit(key string, nowMs int64) (reason string, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if last, seen := g.lastSeen[key]; seen && nowMs-last < orderDedupWindowMs {
		return fmt.Sprintf("duplicate order dropped — same key within %dms [%s]", nowMs-last, key), false
	}

	// Prune the rate window, then count.
	cut := nowMs - orderRateWindowMs
	kept := g.actionTimes[:0]
	for _, ts := range g.actionTimes {
		if ts >= cut {
			kept = append(kept, ts)
		}
	}
	g.actionTimes = kept
	if len(g.actionTimes) >= orderRateMax {
		return fmt.Sprintf("rate breaker TRIPPED — %d order-actions within %ds; halting new actions", len(g.actionTimes), orderRateWindowMs/1000), false
	}

	g.lastSeen[key] = nowMs
	g.actionTimes = append(g.actionTimes, nowMs)
	return "", true
}
