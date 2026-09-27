package safe

// panic-net-complete (2026-09-26): the FULL net for long-lived goroutines.
// A panic in any bare goroutine tears down the entire process — GoNet is the
// single wrapper that guarantees a panic becomes ERROR + stack + counter
// (+ trader freeze + optional alert) instead of a process death.

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"nofx/discipline"
	"nofx/logger"
	"nofx/telemetry"
)

// TestPanicNextGoNet is a test seam (default false): when true, the NEXT GoNet
// call panics before running fn, so a test can pin that a specific wrapped
// goroutine's panic is recovered — the process survives and the net logs and
// counts. A mutant that removes the recover from GoNet dies inside this test.
var TestPanicNextGoNet atomic.Bool

// GoNet launches fn in a new goroutine under the full panic net (P1-F
// extension). On panic it:
//   - logs ERROR + stack (the goroutine name and owner are on the line),
//   - records a telemetry counter ("goroutine_panic", owner key),
//   - FREEZES the owner trader when owner != "" (entries refuse, exits keep
//     working — the #246 freeze contract),
//   - runs the optional onPanic callbacks (e.g. an in-app P0 alert) guarded by
//     their own recover,
//   - then the goroutine exits cleanly.
//
// Server-level goroutines (owner == "") get ERROR + counter + clean exit; a
// caller that can restart its goroutine correctly may do so in onPanic.
func GoNet(name, owner string, fn func(), onPanic ...func(recovered interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Errorf("🔥 [%s] goroutine panic recovered (owner=%s): %v\n%s", name, owner, r, stack)
				telemetry.RecordError(owner, "goroutine_panic", fmt.Sprintf("%s: %v", name, r), telemetry.CostNone)
				if owner != "" {
					discipline.FreezeTrader(owner, fmt.Sprintf("panic in %s: %v", name, r), time.Now().UnixMilli())
				}
				for _, cb := range onPanic {
					if cb == nil {
						continue
					}
					func() {
						defer func() {
							if r2 := recover(); r2 != nil {
								logger.Errorf("🔥 [%s] onPanic callback itself panicked: %v", name, r2)
							}
						}()
						cb(r)
					}()
				}
			}
		}()
		if TestPanicNextGoNet.CompareAndSwap(true, false) {
			panic(fmt.Sprintf("test-injected panic in %s", name))
		}
		fn()
	}()
}
