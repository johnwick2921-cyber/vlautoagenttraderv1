# ADR-001: CSV file bridge over TCP socket for Go → NinjaTrader 8

## Status

Accepted (2026-05-22). Shipped in Plan 1 (`v1.0-plan1`). A TCP AddOn migration is designed in Plan 1.5 of `docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md` and remains deferred.

## Context

The Go trading bot runs in WSL2 (Linux). NinjaTrader 8 runs on the Windows host. We need a bidirectional integration:

- **Go → NT8**: emit a signal (action / symbol / entry / stop / take) so NT places a market order.
- **NT8 → Go**: report fills (timestamp / direction / fill price) so the engine can reason about the next decision.

Three viable transports were on the table:

1. **CSV file polling** — Go writes `trade_signals.csv` via atomic temp+rename; NT polls every ~2s with a NinjaScript strategy (`vltrader.cs`) and appends fills to `trades_taken.csv`.
2. **TCP socket via custom NT8 AddOn** — long-lived TCP server inside NT8, framed JSON or protobuf messages, single round-trip per signal.
3. **COM / .NET interop** — drive NT8 through its in-proc .NET API.

Constraints:

- The first NQ live-SIM milestone needed to land in days, not weeks.
- The bot must remain debuggable from either side without compiling instrumentation in.
- WSL2 default networking is NAT; reaching Windows-side `127.0.0.1` only works in "mirrored" mode (Win 11 22H2+). A transport that survives NAT-mode hosts is preferable.
- The user already runs NT8 with a fully licensed copy; touching NT is allowed.

## Decision

**Plan 1 ships CSV polling**, with the writer in `provider/ninjatrader/csv_writer.go` (atomic temp+rename) and the tailer in `provider/ninjatrader/csv_tailer.go`. NT polls the signal file every 2s; the bot tails the fills file from WSL via `/mnt/c/...`.

**Plan 1.5 documents a TCP migration path** in the implementation plan, sketching a NinjaScript AddOn TCP server with a length-prefixed JSON protocol. It is **not** scheduled — SIM trading on the CSV bridge is acceptable for current goals.

The 5-field signal protocol and 3-field fill protocol are pinned in `CLAUDE.md` ("CSV protocol for `vltrader.cs` — 5-field signals, 3-field fills") and verified by `provider/ninjatrader/csv_writer_test.go` plus `csv_tailer_test.go`.

## Consequences

**Positive**

- Robust against network glitches — a missed poll just means a 2s retry, not a dropped TCP connection.
- Trivially debuggable from either side: `cat trade_signals.csv` shows the last command, `tail -f trades_taken.csv` shows fills in real time. No protocol decoder required.
- Survives WSL2 NAT mode — `/mnt/c/...` is a filesystem mount, not a network socket.
- Atomic temp+rename guarantees NT never reads a partial signal row.
- Implementation hit the validated SIM-fill milestone on 2026-05-22 (live SIM fills on SIM101 via NT Playback).

**Negative**

- 2-second polling floor on latency. Acceptable for the algorithm cadence (which acts on 1m/5m/15m bars) but unsuitable for high-frequency strategies.
- Windows Defender's real-time scanner can transiently hold file locks during `os.Rename`, surfacing as recoverable `rename: Access is denied` errors. Mitigated by adding the data dir to Defender exclusions — see `docs/operations/STARTUP.md` §6.
- The bridge has no built-in heartbeat. Tailer staleness must be detected by the engine via `market/data_freshness.go` (Plan 3 Task 22), not by the transport.

**Neutral**

- Files are append-only fills + atomic-replace signals — no schema versioning. A future protocol change requires a coordinated Go + NinjaScript update; documented in `docs/operations/ROLLBACK.md` §3.

## Alternatives Considered

- **TCP AddOn (rejected for Plan 1; designed in Plan 1.5)** — Significant NinjaScript engineering (AddOn class, threading, JSON parser), much harder to debug, and only meaningfully wins on latency we don't need. Worth the effort only once the algorithm itself demands sub-second round-trips.
- **COM / .NET interop (rejected)** — Tightly couples the Go binary to a specific NT8 build, fragile across NT updates, and requires Mono or a Windows host for the Go side. Vendor lock-in with no offsetting benefit at this latency tier.
- **Shared SQLite via `/mnt/c/...` (rejected)** — Considered briefly. NT8 NinjaScript has no native SQLite driver, and concurrent writers across the WSL/Win boundary deadlock under Defender pressure.

See also ADR-007 for why the Plan 1 critical files must remain byte-stable across subsequent plans.
