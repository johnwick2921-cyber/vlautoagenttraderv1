# Quarterly contract roll (CME equity futures: MNQ/NQ/MES/ES/…)

**The roll is whatever NT8's front month says. Nothing else.** Not a date
rule, not "expiry − 8 days", not a runbook that assumes those agree. Rewritten
2026-09-11 after the September roll put the bot on December while the
platform, the owner's chart and the Control Center were still on September —
for 5½ hours, one filled position (605) and every planner level.

## How the contract is resolved (since `c2eef211`, 2026-09-11)

`ninjascript/VLInstrumentLookup.cs` is the ONE door for bars, orders,
`close_position` and `place_protective_stop`:

1. `Instrument.GetInstrument("<root> ##-##")` — NT8's rolling front-month
   instrument.
2. `MasterInstrument.GetNextExpiry(DateTime.Now)` on it — NT8's own
   rollover table, the same one its charts follow.
3. `Instrument.GetInstrument("<root> MM-yy")` — the CONCRETE contract. That
   one has data and takes orders; the rolling instrument itself returns zero
   bars (proved 2026-09-11 00:33 CT).

Only if NT8 refuses the rolling name does the old date rule
(`VLContractResolver.DateRuleContract`, expiry − 8 days) run, and it logs
`NT8 refused the rolling name … falling back to the DATE RULE` every time. If
you see that line, the bot may be on a contract the platform is not on —
treat it as a stop.

The AddOn names the contract on the wire: the `subscribed` ACK's
`resolved_contract`, and (since `ed6bac8b`) every `bars_historical` /
`bar_update` frame's `contract`. Go keys roll detection on the ACK name
(`provider/ninjatrader/contract_roll.go`): a different name than last time
purges the ring and reseeds it from the store for the new contract only, and
raises one P0 with both names.

## What you confirm before any entry on roll week

In the bot log (`journalctl -u nofx`) — **all three must name the same
contract, and it must be the one NT8's Control Center shows:**

- `📐 NT8 instrument_info MNQ (MNQ MM-yy)`
- `📜 contract: current=MNQ MM-yy (source=subscribed@…)` on the boot line
- a `bar_update` fact in the research archive with that `contract`

and in the NT8 log (`Documents\NinjaTrader 8\log\log.<date>.en.txt`):

- `resolved MNQ -> MNQ ##-## => MNQ MM-yy (rolling->MNQ MM-yy)`

If the NT8 log says `(date-rule-fallback)` or the three do not agree: **no
entries** until they do.

## The procedure when NT8 rolls

NT8 rolls on its own rollover date; the AddOn follows it on the next
resolution — every `bars_subscribe` and every order resolve fresh, and a
reconnect rebuilds the BarsRequests. To make it happen deliberately:

1. **Go flat.** An old-contract position does not migrate.
2. **Restart NT8** (full close + reopen, data connected). The AddOn
   re-resolves; the `subscribed` ACK names the new contract.
3. **Watch Go, no restart needed:** `🚨 P0 — CONTRACT ROLLED MNQ 09-26 →
   MNQ 12-26` — the ring is purged and reseeded for the new contract only.
   A Go restart is fine but not required; the store keeps both contracts
   under their own labels and the readers filter to the current one.
4. **Confirm** the three lines above agree with the Control Center.
5. Repoint your manual NT8 charts (for your eyes; the bot does not need it).

## Expected artifacts (not bugs)

- The new contract's history is its own; it differs from the old one by the
  carry basis — **~275–290 MNQ points at a quarter's distance at 2026 rates**
  (Sep→Dec 2026 measured 290). A step that size across a seam is the basis,
  not a market event; the contract label on every bar is what keeps readers
  from seating levels across it.
- Thin early history on deep timeframes. Normal.

## What went wrong in September 2026 (so it is not repeated)

- The resolver (`0b8d8342`, 2026-05-28) computed the front month from
  `DateTime.UtcNow` (expiry − 8 days). It flipped to December at 00:00 UTC on
  2026-09-11 = 19:00 CT on the 10th. NT8 had not rolled.
- Bars re-resolved on the 21:15 CT reconnect (December); orders re-resolved
  immediately (order 152 at 19:11 CT on December, position 605 at 23:04 CT on
  December — NT8 log `Instrument='MNQ 12-26'`).
- The December request's history came back at September's prices under the
  December name (NT8 serves the prior contract's history before its own
  rollover); its live ticks were December's. That ~290-point step was read
  first as the roll, then as a replay/live scale defect. It was the basis
  between two contracts, only one of which the platform was on.
- The June 2026 version of this document prescribed "restart NT8 + restart
  the bot on/after the roll date (expiry − 8 days)". Its premise was the
  defect: it took the AddOn's date for the platform's roll. Followed on
  2026-09-11 it would have put the bot further onto December.
- `docs/superpowers/reports/2026-09-05-vet-06-risk.md:255` named the window
  ("19:00 CT 09-10 → 09-14: orders on DEC26, bars on SEP26") and the refusal
  to apply, five days early. It was filed and not converted into a gate.
- The May plan ruled `GetNextExpiry` "UNBOOTSTRAPPABLE on Tradovate" (no
  MasterInstrument without a continuous contract). The rolling name
  `MNQ ##-##` resolves on this build and its MasterInstrument answers
  `GetNextExpiry`; the June refutation closed the correct path.
