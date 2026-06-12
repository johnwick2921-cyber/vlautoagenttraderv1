# Rename: ClaudeTrader → VLTrader (NinjaScript only)

**Date:** 2026-05-23
**Scope:** Cosmetic rename of the NinjaScript strategy file + class + display
name. Plan 1 CSV bridge protocol (`trade_signals.csv`, `trades_taken.csv`,
5-field signals, 3-field fills) is unchanged. Go side untouched.

## What changed

### NinjaScript source (in the external bridge clone)

`C:\NofxTrader\bridge\ninjascripts\claudetrader.cs` → `vltrader.cs`

Inside the file:

| Before | After |
|---|---|
| `public class ClaudeTrader : Strategy` | `public class VLTrader : Strategy` |
| `Name = "ClaudeTrader"` | `Name = "VLTrader"` |
| `Description = @"ClaudeTrader - Receives signals..."` | `Description = @"VLTrader - Receives signals..."` |
| `Print("ClaudeTrader Initialized ...")` | `Print("VLTrader Initialized ...")` |
| `EnterLong(0, contractQuantity, "CT_Long")` | `EnterLong(0, contractQuantity, "VL_Long")` |
| `EnterShort(0, contractQuantity, "CT_Short")` | `EnterShort(0, contractQuantity, "VL_Short")` |
| `GroupName="ClaudeTrader Parameters"` (×4 `[Display]`) | `GroupName="VLTrader Parameters"` |
| All `"CT_Long"` / `"CT_Short"` constants throughout exit-order placement | `"VL_Long"` / `"VL_Short"` |

17 sites rewritten in total; zero `Claude` / `CT_` references remain in the
renamed file.

### NinjaTrader install dir

`C:\Users\<you>\Documents\NinjaTrader 8\bin\Custom\Strategies\claudetrader.cs`
removed; `vltrader.cs` copied in.

## What stayed identical (Plan 1 bridge contract)

- CSV file names: `trade_signals.csv`, `trades_taken.csv`
- CSV schema: 5 fields signals (`DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit`),
  3 fields fills (`DateTime,Direction,Entry_Price`)
- Data directory: `C:\Users\<you>\NofxTrader\data\`
- File polling interval: 2 seconds
- Dedup behaviour: `DateTime + Direction` at 1-second resolution

Go side (`provider/ninjatrader/`, `trader/ninjatrader/`) never referenced
"Claude" — it uses the neutral "ninjatrader" package name throughout. No
Go-side changes were required.

## NT-side action John must take to activate the rename

1. NinjaTrader → Tools → Edit NinjaScript → Strategy. `ClaudeTrader` is gone;
   `VLTrader` should appear.
2. F5 to compile `VLTrader`. Expect `0 errors`.
3. On the MNQ chart: Strategies → remove the old `ClaudeTrader` instance →
   add `VLTrader` with the same parameters:
   - Signals File Path: `C:\Users\<you>\NofxTrader\data\trade_signals.csv`
   - Trades Log File Path: `C:\Users\<you>\NofxTrader\data\trades_taken.csv`
   - File Check Interval: `2`
   - Contract Quantity: `1`
4. Apply → Enable. Watch Output window for `VLTrader Initialized - Monitoring
   signals every 2 seconds`.

## Cross-reference

- Plan 1 plan document (`docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md`,
  on the `nq-databento-ninjatrader-plan` branch) still cites `claudetrader.cs`
  throughout. Those references are kept as historical record — the file was
  named `claudetrader.cs` at the time the plan was written and validated. When
  the Plan 1 PR (#1) merges, a follow-up commit on `main` should add a brief
  "renamed 2026-05-23 → vltrader.cs" note at the top of the verified-facts
  section in that doc.
