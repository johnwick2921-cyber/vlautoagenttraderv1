package trader

import (
	"context"
	"errors"
	"sync"
)

// ── W-ONE-BUTTON M2 — THE ENTRY BARRIER ─────────────────────────────────────
//
// One per process (installation-wide, not per trader). Every entry SEND holds
// a permit for the duration of the send — acquired immediately before the
// wire write and released after it returns — so "maintenance held" and "an
// entry is on the wire" can never both be true without Hold knowing.
//
//	Permit()  → (release, true) while released; (nil, false) once Hold began
//	Hold(ctx) → refuses new permits AT ONCE, then waits until every permit
//	            granted before it is released ("drained"); honours ctx so a
//	            wedged send cannot wedge the updater — a timed-out Hold STAYS
//	            held (fail closed) and reports not drained
//	Release() → grants permits again
//
// Design choice (dispatch said sync.RWMutex): a mutex + in-flight counter +
// drain signal gives the same ordering guarantee (a permit is either counted
// before Hold flips `held`, so Hold waits for it, or it observes `held` and is
// refused) AND a cancellable wait, which RWMutex.Lock cannot offer.
type EntryBarrier struct {
	mu       sync.Mutex
	held     bool
	inFlight int64
	zero     chan struct{} // closed when inFlight reaches 0 while held
}

// ErrBarrierReleased is returned by a Hold that was waiting when Release ran.
var ErrBarrierReleased = errors.New("entry barrier released while Hold was waiting")

// Permit grants one entry send, or refuses while held. The release func is
// idempotent.
func (b *EntryBarrier) Permit() (release func(), ok bool) {
	b.mu.Lock()
	if b.held {
		b.mu.Unlock()
		return nil, false
	}
	b.inFlight++
	b.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			b.inFlight--
			if b.inFlight == 0 && b.zero != nil {
				close(b.zero)
				b.zero = nil
			}
			b.mu.Unlock()
		})
	}, true
}

// Hold refuses new permits and waits for in-flight ones to finish.
func (b *EntryBarrier) Hold(ctx context.Context) error {
	for {
		b.mu.Lock()
		b.held = true
		if b.inFlight == 0 {
			b.mu.Unlock()
			return nil
		}
		if b.zero == nil {
			b.zero = make(chan struct{})
		}
		ch := b.zero
		b.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err() // stays held
		}
		b.mu.Lock()
		stillHeld := b.held
		b.mu.Unlock()
		if !stillHeld {
			return ErrBarrierReleased
		}
	}
}

// Release grants permits again.
func (b *EntryBarrier) Release() {
	b.mu.Lock()
	b.held = false
	if b.zero != nil { // wake any waiting Hold; it will see !held
		close(b.zero)
		b.zero = nil
	}
	b.mu.Unlock()
}

// Held reports whether new permits are refused.
func (b *EntryBarrier) Held() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held
}

// Drained reports held AND no permit outstanding.
func (b *EntryBarrier) Drained() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held && b.inFlight == 0
}

// InFlight is the number of entry sends currently holding a permit.
func (b *EntryBarrier) InFlight() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.inFlight
}
