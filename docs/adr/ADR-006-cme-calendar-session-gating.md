# ADR-006: CME calendar session gating before any AI inference

## Status

Accepted (2026-05-22). Shipped in Plan 2 (`v1.0-plan2`). Implementation in `kernel/cme_calendar.go`; wired at `kernel/engine_analysis.go:50` (the first gate in the decision pipeline).

## Context

CME Globex hours for index futures (NQ / MNQ / ES / MES / RTY / MRTY / YM / MYM):

- **Sunday 17:00 CT → Friday 16:00 CT**, with a daily **maintenance break 16:00–17:00 CT**.
- Plus US federal + CME-specific holidays — full or early close depending on the day.

The bot scans on a 60s interval. Without a session gate it will:

1. Spend AI inference budget on bars that are stale because the market is closed.
2. Potentially emit signals during the 16:00–17:00 CT maintenance gap, which lands as a rejected order at NT.
3. Emit signals on holidays when the user expects the system idle.

Constraints:

- AI inference costs real USD per call (DeepSeek is cheap, Claude is not). Wasting calls on closed markets is a direct cost.
- Holiday data has no first-party feed we trust. CME publishes a holiday calendar PDF; we have to encode it.
- The gate must run BEFORE the AI call, not after, to capture the cost saving.

## Decision

**`kernel/cme_calendar.go` exposes `IsCMEOpen(time.Time) bool`** (and supporting helpers like `IsHoliday`, the daily-break check, and weekend/Sunday-open logic). It is the **first gate** in `GetFullDecisionWithStrategy` at `engine_analysis.go:50` — before contract-roll check (Task 19), before risk limits (Task 21), before stale/drift (Task 22), and crucially **before** the AI call.

Holidays are encoded as **13 US federal + CME-specific dates** for the relevant years, hardcoded in `kernel/cme_calendar.go`. Each entry is `time.Date(year, month, day)`; the table is cross-referenced in `docs/operations/MONITORING.md` and `docs/operations/TRADER_MODE.md`.

When the gate trips:

- The engine logs the skip reason and returns a `hold` decision without invoking the AI.
- Monitoring picks up the 16:00–17:00 CT skip via the log line `🚫 CME closed — daily maintenance break` (see `docs/operations/MONITORING.md` §1).

## Consequences

**Positive**

- **Zero AI cost during closed markets.** Saves real USD on holidays and weekends.
- Deterministic skip — the engine state machine is the same every Saturday night, every December 25.
- Operators see a clean log line, not a sea of rejected-order errors from NT.
- The same calendar is reused for the contract-roll gate (Task 19) and the weekly checklist in `docs/operations/TRADER_MODE.md` §3.

**Negative**

- The 13-entry holiday table requires **annual maintenance**. When 2027 calendars publish, somebody updates the file. There is no test that fails on Jan 1 reminding us — this is a manual process documented in CONTRIBUTING.md.
- Hard-coded for CME US index futures. Adding non-US instruments (Eurex, ICE) would need a parallel calendar or a refactor to instrument-keyed calendars.
- Half-day sessions (Thanksgiving Friday, Christmas Eve) are encoded as full closes in the current implementation. The bot does not trade those half-days. A future task could relax this.

**Neutral**

- The CME-specific gap (16:00–17:00 CT) does not align with calendar-day boundaries in any timezone. The gate uses CT (`America/Chicago`) and is robust to DST transitions because Go's `time.LoadLocation` handles them.

## Alternatives Considered

- **Trust exchange responses (rejected)** — Lets NT8 / CME reject the order. Wastes the AI call. Floods logs with rejection errors. Loses the deterministic "engine knows market is closed" property.
- **24/7 trading with NT-side filter only (rejected)** — VLTrader could refuse signals outside RTH. But that pushes the burden into NinjaScript, which is harder to test, and still spends the AI budget upstream.
- **External holiday API (rejected)** — Adds a network dependency to the hot path. CME holidays change rarely; a hardcoded table with documented annual maintenance is simpler and outage-proof.
- **Run the gate AFTER the AI call (rejected)** — Captures none of the cost saving, which was the primary motivation.

See also ADR-002 (the broader gating philosophy) and ADR-007 (why `kernel/cme_calendar.go` is on the Plan 1 critical file list).
