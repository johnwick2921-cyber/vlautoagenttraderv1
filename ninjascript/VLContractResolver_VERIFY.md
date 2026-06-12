# VLContractResolver — Expected Output Verification Table

Companion verification artifact for `VLContractResolver.cs`. Because the NinjaScript SDK
does not resolve under WSL2, conventional C# unit tests cannot be run from this repo.
This document is the substitute: a hand-verifiable table of expected outputs that an
operator can eyeball after a future NT8 compile of `VLContractResolver`.

## What the helper does

`VLContractResolver.ResolveFrontMonthContractAt(symbol, now)` maps a generic futures
root (or a Databento continuous symbol) to NinjaTrader's fully-qualified front-month
contract name for a given `now`.

Examples (today is 2026-05-28, before the June roll):

- `MNQ`        -> `MNQ 06-26`
- `NQ.c.0`     -> `NQ 06-26`
- `nq.c.0`     -> `NQ 06-26`         (case-insensitive `.c.` strip; root uppercased)
- `MNQ 06-26`  -> `MNQ 06-26`        (idempotent — already qualified)
- `BTCUSDT`    -> `BTCUSDT`          (non-CME root — passthrough)

## Why it's date-derived (not a hardcoded contract)

A hardcoded string like `"MNQ 06-26"` goes stale every quarter. Eight days before each
quarterly expiry (the standard CME front-month roll convention) we need a new contract
name. Deriving from `now` means the AddOn keeps working through every roll without a
code change. No release ceremony, no missed roll, no manual edit.

## How to sanity-check after a future NT8 compile

When the bot starts and the Go side sends `bars_subscribe` for `MNQ`, the AddOn's NT8
log (Control Center -> New -> NinjaScript Output) should emit a line of the form:

```
resolved MNQ -> MNQ <MM-YY>
```

Look up today's date in the table below — the `<MM-YY>` should match. If it doesn't,
the helper is wrong. If it does, ship.

## Next roll note

This table shows the helper auto-rolling from `MNQ 06-26` to `MNQ 09-26` on
**2026-06-12** (one day after the inclusive June roll boundary 2026-06-11). The
operator should expect the log line to flip format on that date with no code change.
If the log still says `MNQ 06-26` on 2026-06-12 or later, the helper is broken.

## Spec the helper must satisfy

1. **Idempotency on already-qualified strings** — if input contains a space, return it
   unchanged. (`"MNQ 06-26"` -> `"MNQ 06-26"`)
2. **Strip Databento continuous suffix** — case-insensitive `.c.<digit>` is removed
   before whitelist lookup. (`"NQ.c.0"` -> root `"NQ"`; `"nq.c.0"` -> root `"NQ"`)
3. **Non-whitelist passthrough** — if the stripped root is not in
   `["MNQ", "NQ", "MES", "ES", "MYM", "YM", "M2K", "RTY"]`, return the **original**
   input unchanged.
4. **Quarterly front-month derivation**:
   - Quarterly months: H=3 (Mar), M=6 (Jun), U=9 (Sep), Z=12 (Dec)
   - Expiry = 3rd Friday of the month
   - Roll date = expiry − 8 days (`RollDaysBeforeExpiry`)
   - Front month = first quarterly where `rollDate >= now` (boundary inclusive)
   - Past December's roll → advance to next year's March
5. **Format** — `"<root> <MM-YY>"` via `CultureInfo.InvariantCulture`, both fields
   zero-padded 2-digit. Year = `year % 100`.

## Reference calendar (hand-verified against an actual calendar)

| Year | Quarter | 3rd Friday (expiry) | Roll date (−8 days) |
|------|---------|---------------------|---------------------|
| 2026 | Mar (H6) | 2026-03-20 (Fri) | 2026-03-12 (Thu) |
| 2026 | Jun (M6) | 2026-06-19 (Fri) | 2026-06-11 (Thu) |
| 2026 | Sep (U6) | 2026-09-18 (Fri) | 2026-09-10 (Thu) |
| 2026 | Dec (Z6) | 2026-12-18 (Fri) | 2026-12-10 (Thu) |
| 2027 | Mar (H7) | 2027-03-19 (Fri) | 2027-03-11 (Thu) |
| 2027 | Jun (M7) | 2027-06-18 (Fri) | 2027-06-10 (Thu) |
| 2027 | Sep (U7) | 2027-09-17 (Fri) | 2027-09-09 (Thu) |
| 2027 | Dec (Z7) | 2027-12-17 (Fri) | 2027-12-09 (Thu) |

All eight 3rd-Friday dates above were cross-checked against a Gregorian calendar and
verified to fall on a Friday. None disagree with the spec.

## Expected-value table

`now` is the value passed as the second argument to `ResolveFrontMonthContractAt`.
Treat all dates as wall-clock UTC; the helper uses only the `Date` portion so the
time-of-day is irrelevant.

| `now` (UTC) | input `symbol` | expected output | rationale |
|-------------|----------------|-----------------|-----------|
| 2026-05-28  | `MNQ`          | `MNQ 06-26`     | May is before June's 2026-06-11 roll → front is June 2026 |
| 2026-06-10  | `MNQ`          | `MNQ 06-26`     | One day before the June roll → still on June |
| 2026-06-11  | `MNQ`          | `MNQ 06-26`     | Exactly on the June roll date; rule is `rollDate >= now`, so June still qualifies (boundary inclusive) |
| 2026-06-12  | `MNQ`          | `MNQ 09-26`     | Past the June roll → September |
| 2026-06-19  | `MNQ`          | `MNQ 09-26`     | June's expiry day itself — long past June's roll → September |
| 2026-09-09  | `MNQ`          | `MNQ 09-26`     | Before Sept roll → September |
| 2026-09-10  | `MNQ`          | `MNQ 09-26`     | Exactly on Sept roll (boundary inclusive) |
| 2026-09-11  | `MNQ`          | `MNQ 12-26`     | Past Sept roll → December |
| 2026-12-09  | `MNQ`          | `MNQ 12-26`     | Before December roll → December |
| 2026-12-10  | `MNQ`          | `MNQ 12-26`     | Exactly on December roll (boundary inclusive) |
| 2026-12-11  | `MNQ`          | `MNQ 03-27`     | Past December roll → next year's March |
| 2026-12-25  | `MNQ`          | `MNQ 03-27`     | Holiday — past December roll → next year's March |
| 2026-12-31  | `MNQ`          | `MNQ 03-27`     | Year-end boundary — still rolls into March 2027 |
| 2027-01-01  | `MNQ`          | `MNQ 03-27`     | New-year boundary — March 2027 front-month confirmed across year flip |
| 2027-03-10  | `MNQ`          | `MNQ 03-27`     | Before March 2027 roll |
| 2027-03-11  | `MNQ`          | `MNQ 03-27`     | Exactly on March 2027 roll (boundary inclusive) |
| 2027-03-12  | `MNQ`          | `MNQ 06-27`     | Past March 2027 roll → June 2027 |
| any         | `MNQ 06-26`    | `MNQ 06-26`     | Idempotent — contains a space, already qualified |
| any         | `MNQ 03-27`    | `MNQ 03-27`     | Idempotent |
| any         | `NQ 12-26`     | `NQ 12-26`      | Idempotent — non-front-month qualified contract passes through |
| any         | `MNQ  06-26`   | `MNQ  06-26`    | Idempotent — even with double-space; helper checks for any space, doesn't reformat |
| 2026-05-28  | `NQ.c.0`       | `NQ 06-26`      | Continuous suffix stripped → root `NQ` → June 2026 |
| 2026-05-28  | `MNQ.c.0`      | `MNQ 06-26`     | Same with Micro |
| 2026-05-28  | `nq.c.0`       | `NQ 06-26`      | Case-insensitive `.c.` strip; root uppercased |
| 2026-05-28  | `MNQ.C.0`      | `MNQ 06-26`     | Upper-case `.C.` also stripped (case-insensitive) |
| 2026-05-28  | `NQ.c.1`       | `NQ 06-26`      | Suffix is `.c.<digit>` — non-zero index also stripped to root `NQ` |
| 2026-05-28  | `ES.c.0`       | `ES 06-26`      | ES is whitelisted; continuous suffix strips to root `ES` |
| 2026-05-28  | `ES`           | `ES 06-26`      | Whitelisted root |
| 2026-05-28  | `MES`          | `MES 06-26`     | Whitelisted root |
| 2026-05-28  | `YM`           | `YM 06-26`      | Whitelisted root |
| 2026-05-28  | `MYM`          | `MYM 06-26`     | Whitelisted root |
| 2026-05-28  | `RTY`          | `RTY 06-26`     | Whitelisted root |
| 2026-05-28  | `M2K`          | `M2K 06-26`     | Whitelisted root |
| any         | `BTCUSDT`      | `BTCUSDT`       | Not a CME root — passthrough unchanged |
| any         | `ETH`          | `ETH`           | Not a CME root — passthrough unchanged |
| any         | `CL`           | `CL`            | Crude oil — not in whitelist — passthrough unchanged |
| any         | `GC`           | `GC`            | Gold — not in whitelist — passthrough unchanged |
| any         | `NQX`          | `NQX`           | Not a whitelisted root (root != `NQ` after no strip) — passthrough |
| any         | `` (empty)     | `` (empty)      | Defensive — empty stays empty (helper should handle gracefully) |

## Boundary edge cases worth flagging

- **Roll-date inclusivity**: the rule is `rollDate >= now`, NOT `rollDate > now`.
  That means on the exact roll date the front-month remains the **current** quarter,
  not the next one. Rows 2026-06-11, 2026-09-10, 2026-12-10, 2027-03-11 above all
  exercise this. If the helper uses `>`, every one of those four rows will be wrong.
- **Year flip past Z**: when `now` is in Q4 after the December roll, the front month
  is the next year's March. Rows 2026-12-11, 2026-12-25, 2026-12-31, 2027-01-01
  exercise this — they all expect `MNQ 03-27`.
- **Case-insensitive `.c.` strip**: both `.c.0` and `.C.0` must strip identically.
  Rows `nq.c.0` and `MNQ.C.0` exercise this.
- **Continuous-suffix index variation**: Databento's continuous form is `.c.<N>`
  where `N` is the depth. `.c.0` and `.c.1` should both strip to the same root.
- **Non-whitelist roots starting with whitelist prefix** (e.g. `NQX`): the helper
  must match the **whole** root, not a prefix. If it prefix-matches, `NQX` would
  incorrectly resolve to `NQ <MM-YY>`.
- **Whitespace already present**: any input containing a space is treated as
  already-qualified and returned verbatim — including pathological forms like
  `MNQ  06-26` (double space). The helper is not expected to normalise whitespace.
- **Empty input**: should not throw. The whitelist test will simply fail and the
  empty string passes back through.

## Operator quick-check

For **today (2026-05-27)**, the expected log line is:

```
resolved MNQ -> MNQ 06-26
```

If the AddOn logs anything other than `MNQ 06-26`, the helper is broken — compare
against the May/June rows above.
