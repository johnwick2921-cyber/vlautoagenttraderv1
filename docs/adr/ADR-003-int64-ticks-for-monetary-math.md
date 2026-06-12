# ADR-003: int64-ticks for all monetary math (no float64)

## Status

Accepted (2026-05-22). Shipped in Plan 2 (`v1.0-plan2`). Implementation in `market/decimal_safe.go`, drift verified by `market/decimal_safe_test.go`.

## Context

The bot accumulates monetary quantities across many operations: position sizing, P&L summation, daily-loss tracking, R/R calculations, SL/TP placement. NQ has a tick size of **0.25 index points** with a tick value of **$5** (MNQ: $0.50/tick). Cumulative float64 arithmetic drifts because most decimal values that look exact in code (0.25, 0.05, 0.1) are not representable exactly in binary IEEE-754.

Concrete drift was confirmed in `market/decimal_safe_test.go`:

```
Accumulating 3000 operations of 0.25 in float64 drifts by ~2.84e-13.
```

This is small per-step, but:

- Realized P&L sums across a year of trades can land just outside a risk limit by a hundredth of a cent.
- Tick rounding of an SL/TP that derives from a float computation can land on the wrong tick.
- Equality checks (`if price == stopLoss`) become unreliable.

Three options:

1. **`shopspring/decimal`** — arbitrary-precision decimal arithmetic, well-known Go library.
2. **int64 ticks** — store all prices and money as integer multiples of the instrument's minimum unit.
3. **Raw float64** — accept the drift, hope it stays in the noise.

## Decision

**All monetary math uses int64 ticks.** A "tick" is the smallest price unit of the instrument: 0.25 for NQ/MNQ. Helpers in `market/decimal_safe.go` convert between human-readable prices and tick counts, perform addition/subtraction/multiplication in int64, and round-to-tick on output.

Floats remain the boundary representation:

- Databento returns float OHLCV (after dividing by its 1e9 fixed-point divisor) — bars enter the system as floats.
- The Decision JSON shape (`entry`, `stop_loss`, `take_profit`) is written and read as floats over HTTP.
- The web UI reads/writes floats.

The conversion happens at the boundary. Once a price is inside the gating / position-sizing / P&L pipeline, it is a tick count. `trader/ninjatrader/tick_rounding.go` (Plan 2 Task 17) ensures every order leaving the system is on a valid tick.

## Consequences

**Positive**

- Zero drift across arbitrary chains of operations — int64 addition is exact.
- Equality and ordering checks are reliable.
- Tick-rounded outputs (SL/TP) are guaranteed valid for CME match — no broker rejections from sub-tick prices.
- The drift test in `market/decimal_safe_test.go` is a regression guard for any future refactor that tries to "simplify" by switching back to floats.

**Negative**

- Conversion ceremony at JSON boundaries. Every decision write does `price → ticks` on read and `ticks → price` on write. Tested in `decimal_safe_test.go`.
- The tick size is per-instrument. Today the system is NQ/MNQ-only, so 0.25 is hard-wired. Adding ES (0.25) or RTY (0.10) requires plumbing instrument metadata through the tick helpers.
- Slightly higher cognitive overhead reading the code — a number labelled `ticks` is not directly human-readable.

**Neutral**

- int64 gives ±9.2e18 of headroom — more than enough for any plausible price or position size in cents-of-USD or ticks-of-NQ.

## Alternatives Considered

- **`shopspring/decimal` (rejected)** — Adds a third-party dependency to the monetary path. Decimal arithmetic costs ~10–50× a native int op for our workloads. Worth it for arbitrary scale; overkill for a system where every price is a fixed multiple of a known tick.
- **Raw float64 (rejected)** — `market/decimal_safe_test.go` documents 2.84e-13 drift in 3000 ops. That drift compounds. It also forces every comparison to use `math.Abs(a-b) < epsilon`, which becomes its own source of bugs.
- **int32 ticks (rejected)** — Saves no real memory (Go pads in structs), gives only ~2.1e9 of headroom, breaks the moment we want sub-tick accounting (e.g., commission in cents per share for a future equity expansion).

See also ADR-002 (the decision JSON shape that defines the boundary) and ADR-007 (why `market/decimal_safe.go` is on the Plan 1 critical file list).
