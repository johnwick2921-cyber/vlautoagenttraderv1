# Audit — NQ Trading via Databento + NinjaTrader Implementation Plan

**Audited:** 2026-05-22 (v4 re-audit)

## v4 status

v4's headline change is the new **Plan 1.5: NT8 AddOn migration** section — and it's a strong, well-researched answer to the deepest issue I raised (entry-only state sync / no round-trip). Replacing the CSV bridge with an in-process NinjaScript AddOn + loopback TCP server gives real balance, exit/PnL events, cancel/modify, and connection status, and it correctly keeps Plan 1 (CSV) shipping first for SIM validation, then migrates. The WSL2 networking, auth-token/loopback security, idempotency (UUIDv7 + seq), 5-stage migration, and the Tradovate/ProjectX escape ramp are all sensible.

Two caveats on Plan 1.5: (a) it's research-based — the NT8 API claims (`Account.CreateOrder/Submit/CancelAllOrders/Change`, `AccountItem.CashValue`, `ConnectionStatusUpdate`, ATM caveats, port 36973 = NT's ATI port, WSL mirrored mode requiring Win11 22H2+) are external to your repo, so I can't verify them here — validate against current NT8 docs and test on your machine before relying on them; (b) NT8 sockets are officially unsupported by NinjaTrader, as the section itself notes.

**But the two trade-blockers from the punch-list are still NOT fixed in the Plan 1 code:**
1. `GetBalance` still returns camelCase keys (`totalEquity`/`availableBalance`) → engine reads snake_case → **$0 equity** → no trades sized. Unchanged.
2. `placeEntry` still fabricates `entryRef := (sl + tp) / 2.0`, and Verified Facts still asserts "market orders" against the README's "limit orders." Unchanged — still the highest unresolved risk for Plan 1.

**Label nits still open:** Section A Group-3 ("17 implemented") and Section D ("17-method") still say 17 (should be 19); the per-surface section headers still read 55/120/49 while the recounted scorecard says 65/118/45; and S14 still cites `data.go:583` while S-M33 has the correct case-preserving fix — make them consistent.

So: Plan 1.5 is the right long-game architecture and a genuine improvement, but Plan 1 still won't trade correctly until #1 and #2 are applied.


**Plan file:** `docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md`
**Method:** Claims checked against the live working tree at `/home/hoang/nofx` and the upstream `J0shusmc/Claude-Trader-NinjaTrader` README.

---

## v3 verdict

Real progress. The #1 build-blocker is fixed and several issues closed. But **two concrete correctness bugs remain in the code**, and the biggest design risk (market-vs-limit) was not engaged at all. Closer to executable, not there yet.

### Fixed in v3 (verified)
1. **Task 7 now implements all 19 interface methods** — the 5 that were missing (`GetMarketPrice`, `CancelStopLossOrders`, `CancelTakeProfitOrders`, `CancelStopOrders`, `GetOpenOrders`) are present with correct signatures. The `var _ types.Trader` guard will now pass. This was the build-blocker — resolved.
2. **Symbol-case fix is now correct** — B1/S-M33 capture the raw symbol before `strings.ToUpper` and `return raw` for CME symbols, so `NQ.c.0` survives for Databento.
3. **Coverage scorecard now reconciles** — recounted to 65/96/118/45 = 324, and each row's categories sum to its total.
4. **MACDSignal phantom field removed** — `FuturesContext` no longer carries it; the prompt honestly says "MACD line only," and the test/smoke runner match.
5. **CSV hazards documented** — H1 (lost-signal race), H2 (dedup collision), H3 (DrvFs mtime), H4 (fill replay) are written up with fixes and scoped to Plan 1.5/pre-live. Disclosed, not coded — acceptable for a SIM paper slice.
6. **JWT** — refined with the "log line fires unconditionally" caveat and a recommended code-level warning.

### Still broken / unresolved

**1. HIGH — `GetBalance` still returns the wrong map keys (reads as $0, not $50k).**
Task 7's code (and the "honest disclosure" section) still ships:
```go
return map[string]interface{}{
    "totalEquity": 50000.0, "availableBalance": 50000.0,
}, nil
```
The engine reads **snake_case** keys — `balance["total_equity"]` / `["available_balance"]` (comma-ok), and the InitialBalance path's recognized list is `total_equity, totalWalletBalance, wallet_balance, totalEq, balance` (`auto_trader.go:327`). None match `totalEquity`/`availableBalance`. So the mock is silently read as **0 equity**, and since position sizing is `equity × ratio`, the AI is told its max position is **$0** and trades may all be rejected. The disclosure section discusses "$50k fantasy capital" but misses that the keys are simply wrong, so it's not $50k — it's zero. **Fix: use snake_case keys** (`total_equity`, `available_balance`, plus `wallet_balance`/`total_pnl`).

**2. HIGH — market-vs-limit contradiction was not addressed; the fabricated entry price is a live risk.**
v3 still asserts in Verified Facts: "Strategy places **market orders** (not limit, despite README claims)," and Task 7 still fabricates `entryRef := (sl + tp) / 2.0` on that basis. But the **upstream README (current, v1.3)** explicitly describes **limit orders**: "Limit Order Entry," `[SIGNAL] LONG LIMIT @ 25100.00`, order goes to "WORKING" then "FILLED," and the v1.1 changelog says "Changed to limit orders with CSV-based SL/TP." If it is limit orders, a fabricated midpoint entry price means the order rests at a nonsensical level and **may never fill**. The plan's own expected smoke-test log (`[SIGNAL] LONG MARKET ORDER`) also contradicts the README's `LONG LIMIT @`. This must be settled by reading the actual `src/claudetrader.cs` you'll deploy. Until then: **do not fabricate the entry — pass a real near-market price** (e.g., Databento's last close), so it works whether the order is market or limit.

**3. MEDIUM — entry-only state sync still undocumented.**
The new hazards section covers H1–H4 but not the bigger one: `trades_taken.csv` logs **entries only** (confirmed by the upstream README's format + example). NT prints exits and P/L to its Output window but writes nothing back. So `GetPositions` reports a **phantom open position forever** after the first SL/TP exit (it returns `lastFill` with `hasFill=true`, never cleared), and `GetClosedPnL` is always empty. For continuous running the bot's state desyncs from NT after the first close. Add this as H5 / a first-class scope limitation, and at minimum clear `hasFill` on some timeout or instruct the prompt that it cannot rely on position state.

### Minor nits
- Section A's Group-3 summary still says "**17 implemented**," and Section D still says "the 17-method Trader interface impl." Task 7's code and the self-review now correctly say 19 — update these two stragglers.
- The recounted scorecard table (65/118/45) doesn't match the per-surface **section headers** below it, which still read "55 items" / "120 items" / "49 items." Align the headers.

---

## Recommended fixes before execution (priority)
1. `GetBalance` → snake_case keys (or set `config.InitialBalance` and disable Go-side sizing for ninjatrader). — silent $0-equity bug
2. Resolve market-vs-limit by reading the real `src/claudetrader.cs`; meanwhile pass a real entry price, not `(SL+TP)/2`. — could break all fills
3. Document the entry-only state-sync limit (H5) and clear phantom positions.
4. Fix the "17" label stragglers (Section A, Section D) and the surface-header counts.

---

## Verification log (live tree + upstream README)

| Item | v3 status | Result |
|---|---|---|
| Trader interface — 19 methods implemented | Task 7 now has all 19 | ✓ FIXED (compiles) |
| "17 methods" label | still in Section A summary + Section D | ✗ stragglers remain |
| `GetBalance` map keys | still `totalEquity`/`availableBalance` | ✗ wrong; engine reads snake_case → $0 |
| Symbol-case preserve | B1/S-M33 capture raw before ToUpper | ✓ FIXED |
| Coverage scorecard | recounted 324, rows reconcile | ✓ FIXED (headers still old) |
| MACDSignal phantom field | removed; prompt honest | ✓ FIXED |
| CSV hazards H1–H4 | documented, scoped Plan 1.5 | ✓ disclosed |
| Market vs limit | still claims "market orders"; entry fabricated | ✗ contradicts upstream README; unresolved |
| Entry-only state sync (phantom position) | not documented | ✗ still open |
| JWT default secret | config.go:67-69 confirmed; warning recommended | ✓ accurate |
