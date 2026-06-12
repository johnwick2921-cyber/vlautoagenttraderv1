# Quarterly contract roll (CME equity futures: MNQ/NQ/MES/ES/…)

Every **March, June, September, December**, the front-month contract expires
(3rd Friday) and volume moves to the next quarter about **8 days before
expiry** (the roll date). The bot's resolver
(`ninjascript/VLContractResolver.cs` — `ResolveFrontMonthContractAt`) picks
the first quarterly whose roll date (expiry − 8 days) is still ahead, so any
**fresh resolution on/after the roll date automatically lands on the new
contract**. Resolution happens on every `bars_subscribe`
(`VLBarsSubscriptionManager.cs` `HandleBarsSubscribe`) and on every order
submit (`VLTraderTCPClient.cs` `SubmitSignalOrder`) — it is dynamic, **not**
cached at AddOn startup.

## Symptom of a missed roll

Around the roll date, every chart/timeframe goes thin at once: sparse 1m
bars, collapsing volume, stale-looking prices — the old contract is dying
while the bot still streams it. Decisions then run on dying-contract prices.

## The procedure (once per quarter, on/after the roll date)

1. **Go flat** (or wait until flat). Never roll with an open position — the
   old-contract position does not migrate; close it on the old contract
   first.
2. **Restart NinjaTrader 8** (full close + reopen, data connected). The
   AddOn's BarsRequests are recreated → the resolver picks the new quarter
   (the `subscribed` ack in the bot log shows it, e.g.
   `subscription ACK symbol=MNQ contract="MNQ 09-26"`).
3. **Restart the bot**: `sudo systemctl restart nofx` (or kill the PID — the
   service auto-restarts). This wipes the in-memory bar cache so it re-seeds
   **purely from the new contract** (~500 bars per timeframe).
   *Why this matters:* without the bot restart, the cache still holds
   old-contract bars; the new contract trades at a different basis
   (~100–250 NQ points at a quarter's distance), so every chart shows a
   price **cliff** at the switch moment — that IS the "all charts buggy"
   symptom, in the cache, not the chart code.
4. **Repoint your manual NT8 chart** windows to the new contract (the bot
   does not need this; it's for your eyes).
5. **Verify** (bot log / `journalctl -u nofx`):
   - `subscription ACK … contract="<ROOT> <new MM-YY>"`
   - `instrument_info … matches table ✓`
   - dense `bar_update` flow; 1m chart fresh on every timeframe
   - the next decision cycle saves normally.

## Expected artifacts (not bugs)

- **Price gap vs the old contract:** the new contract's history is its own —
  prices differ from the old contract by the carry basis. Indicators that
  span long windows (EMA200, deep ATR) read the *new* contract's history
  after the re-seed, so they are consistent; only a no-bot-restart roll
  leaves mixed-contract series.
- **Thin early history on deep timeframes:** the new contract's daily bars
  from months ago are low-volume (it barely traded back then). Normal.

## Roll dates cheat-sheet

Roll = 3rd Friday of Mar/Jun/Sep/Dec **minus 8 days** (usually the
Wednesday-Thursday of the prior week). When in doubt: if today ≥ roll date,
a fresh NT8 + bot restart lands on the new quarter automatically.
