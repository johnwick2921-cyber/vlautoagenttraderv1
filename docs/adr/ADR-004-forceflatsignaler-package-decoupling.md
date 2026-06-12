# ADR-004: ForceFlatSignaler interface to decouple kernel from trader/ninjatrader

## Status

Accepted (2026-05-22). Shipped in Plan 3 (`v1.0-plan3`) and consumed by the Plan 4 API endpoint (`v1.0-plan4`).

## Context

`kernel/risk_limits.go` implements the daily-loss force-flat kill switch (Plan 3 Task 21). When the limit trips, the engine must emit a "flatten everything" signal that ultimately reaches the broker. For NinjaTrader that means writing a `FLAT` row through `provider/ninjatrader/csv_writer.go`.

The naive wiring is for `kernel/risk_limits.go` to `import "github.com/.../provider/ninjatrader"` and call `csv_writer.WriteFlat(...)` directly. That creates a problem:

- `kernel/` is the strategy engine — it should not know which broker is wired in.
- `kernel/` is imported by `trader/auto_trader.go`. If `kernel` imports `provider/ninjatrader`, and any `trader/*` broker (eventually) imports `kernel` for shared types, we are one refactor away from an import cycle.
- We may want a non-NT8 broker in the future. Hard-coding `provider/ninjatrader` in the risk path forecloses that.

## Decision

**Define an interface `ForceFlatSignaler` in `kernel/`** with the single method required to flatten:

```go
type ForceFlatSignaler interface {
    SignalForceFlat(reason string) error
}
```

The concrete implementation lives in `trader/ninjatrader/` (which already owns the CSV writer goroutine). The API endpoint shipped in Plan 4 Task 23 (`api/risk.go` — `POST /api/risk/force-flat`) is the wiring point: it pulls the active trader from the manager, type-asserts to `ForceFlatSignaler`, and invokes the method.

`kernel/risk_limits.go` declares the interface and uses it. It never imports `provider/ninjatrader` or `trader/ninjatrader`.

## Consequences

**Positive**

- Clean package boundaries: `kernel` → interface → `trader` → `provider`. Mirrors the rest of the trader stack (engine → 19-method `types.Trader` → broker impl).
- Easy to mock: a test can pass a `ForceFlatSignaler` stub instead of standing up a fake CSV writer.
- Future brokers (Plan 1.5 TCP AddOn, an eventual non-NT8 broker) implement the interface; nothing in `kernel/` changes.

**Negative**

- One layer of indirection at the call site. The wiring (in `api/risk.go`) reads slightly verbose because it must resolve the concrete trader and assert the interface.
- Anyone touching the force-flat path has to read two files (`kernel/risk_limits.go` for the interface, `trader/ninjatrader/...` for the impl).

**Neutral**

- The interface intentionally has one method. Future enrichment (e.g., a `Status() string` for the dashboard's emergency-flat indicator) can be added without breaking existing impls, provided the new method gets a default or sentinel for the existing implementer.

## Alternatives Considered

- **Direct import `provider/ninjatrader` from `kernel/risk_limits.go` (rejected)** — Concrete coupling. Closes the door on alternative brokers and risks an import cycle the next time `provider/ninjatrader` reaches back into `kernel` for a shared type.
- **Pub/sub channel of force-flat events (rejected)** — Overkill for a single emitter and a single consumer per active trader. Would also require a coordinator goroutine and shutdown semantics we don't need.
- **Inline the flatten logic in `risk_limits.go` (rejected)** — Means `kernel/` knows about CSV file paths. Strictly worse than the direct-import option.

See also ADR-001 (the CSV transport the implementation sits on top of) and ADR-007 (why `kernel/risk_limits.go` is on the Plan 1 critical file list).
