# VLTrader TCP Client — NinjaScript AddOn

Plan 1.5 alternative to the SIM-validated CSV bridge. Connects NT8 to the VL Trader Go bot over local TCP for sub-second signal/fill latency.

## When to use

By default the Go bot ships with `NT_TRANSPORT=csv` (Plan 1, validated SIM fill at NQ 29807). Switch to `NT_TRANSPORT=tcp` if any of the following triggers fire (see `docs/adr/ADR-001-csv-bridge-vs-tcp.md`):

- CSV polling latency becomes a measurable problem in live trading
- Windows Defender file-lock contention causes signal misses
- Operator-reported CSV bridge failures under load
- Algorithm requires sub-second cadence

The CSV bridge remains the canonical production path until one of these triggers fires. TCP is purely additive.

## Install

1. Copy `VLTraderTCPClient.cs` from this directory into NT8's custom AddOns folder:

   ```
   C:\Users\<you>\Documents\NinjaTrader 8\bin\Custom\AddOns\
   ```

2. Open NT8.
3. Open the NinjaScript editor: **New → NinjaScript Editor**.
4. Navigate to **AddOns → VLTraderTCPClient.cs** in the tree.
5. Press **F5** to compile. Watch the bottom Output window for compile errors.
6. If compilation succeeds, restart NT8.
7. Verify the AddOn is loaded: **Control Center → Output** should show:

   ```
   VLTraderTCPClient: AddOn Active, connecting to 127.0.0.1:36974
   ```

8. Start the VL Trader Go bot with `NT_TRANSPORT=tcp`:

   ```bash
   NT_TRANSPORT=tcp ./nofx-bin
   ```

## Verify end-to-end

1. Open a SIM trader in the VL dashboard.
2. Trigger a test signal (e.g. via the smoke matrix: `go run ./cmd/nq_smoke tcp`).
3. NT8's Output window should show: `VLTraderTCPClient: connected`.
4. NT8 Account Performance should show the OCO bracket: entry market + stop + limit.
5. When the fill occurs, the VL bot's position panel should populate.

## Troubleshooting

- **Compile errors**: NT8 API version drift can cause minor type mismatches between releases. Check the error message; usually a `using` directive or a method signature needs a small tweak (e.g. `NinjaScript.Log` vs `NinjaScript.NinjaScript.Log`). The wire protocol shape is the contract; the C# API surface is allowed to drift.
- **"Connection refused"**: Go bot is not running or `NT_TRANSPORT=tcp` is not set. Check with `NT_TRANSPORT=tcp ./nofx-bin > /tmp/nofx.log 2>&1` and verify `listening on 127.0.0.1:36974` appears in the log.
- **Port collision (36974 already in use)**: another process is holding the port. Verify with `ss -tlnp | grep 36974` on Linux or `netstat -ano | findstr 36974` on Windows. Note: 36974 is intentionally NOT NinjaTrader's ATI port 36973 — do not change it.
- **OCO bracket not placed**: check NT8 Output for `VLTraderTCPClient: stale signal` or `no account` warnings. Stale signals (>60s old) are rejected by design.
- **Multiple AddOn instances**: only one `VLTraderTCPClient` should be Active at a time. NT8 will load every `.cs` in the AddOns folder, so do not leave older copies behind.
- **WSL2 networking**: requires "mirrored" networking mode (Win 11 22H2+) for `127.0.0.1` from WSL to reach the Windows-side NT8 process. Plain NAT mode requires firewall rules + a host-IP rewrite — not supported by this AddOn.

## Uninstall

1. Stop the Go bot or set `NT_TRANSPORT=csv` and restart.
2. Delete `VLTraderTCPClient.cs` from the AddOns folder.
3. Restart NT8.

The CSV bridge remains unchanged and will resume operation immediately.

## Wire protocol reference

See `vltrader_tcp_PROTOCOL.md` for the full wire-format spec. The Go-side implementation at `provider/ninjatrader/tcp_server.go` is authoritative — any drift between this AddOn and the Go side is a bug to be filed and fixed, not a documentation problem.
