# WAVE B — the stop entry reaches the broker with a trigger, and the guard that judges it is the stop-side one

**Branch:** `fix/wave-b-stop-entry` · **base:** `a45cf551` (= `origin/dev` tip at accept) · **claim:** `cb23d9eb`
**Session:** waveb-stopentry-0905 · **worktree:** `/home/hoang/nofx-waveb`
**Status: MERGED TO DEV. GO HALF LIVE. C# HALF STAGED, NOT YET COMPILED.** The Go binary shipped in `f516da7c` — booted 2026-09-05 23:53:44 CT by the Wave A lane (`nofx-6b`) under an owner ruling that collapsed both waves into ONE boot; this lane stood down from its own F1 and handed over the head. F2 ran at 00:04 CT 2026-09-06: the corrected AddOn source is deployed and verified byte-identical, but NT8 is in its maintenance window and could not start, so **nothing has compiled and the C# half is NOT live**. `STOP_ENTRY_SEAM=off` throughout, by owner ruling.

**THIS REPORT COVERS TWO PASSES.** The build (`48e340c0` … `98c28dcd`) and the REPAIR after four adversarial reviews (`291a299c` … the report commit). The repair found **two BLOCKERs the build missed**, both live, both outside what the build had been looking at; they are class 77 and they are the reason this document has a VERIFICATION RECORD at the end naming every finding and its disposition.

**Commits:**

| sha | what |
|---|---|
| `cb23d9eb` | claim (PUSH-EMPTY-AT-ACCEPT) |
| `48e340c0` | D1-D5 + the pins |
| `d843b191` | Guide, SYSTEM-MAP, checklist class 76 |
| `98c28dcd` | this report, first pass |
| `291a299c` | **REPAIR** — canonical side, one value judged and sent, the caller-level pin |
| `95734ddd` | REPAIR — the AddOn's case-folded entry action |
| `90332dc1` | REPAIR — SYSTEM-MAP refs + a contract test for the region this wave owns |
| `7b1af151` | REPAIR — the Guide's gate ORDER, checklist class 77 |

**SPEC FRESHNESS (class 73).** Base `a45cf551` (2026-09-05 18:22:18 CT). `git log -1` for every file built from — nothing moved after the base, no rebase needed:

| file | last commit before base |
|---|---|
| `ninjascript/VLTraderTCPClient.cs` | `c84bd247` 2026-09-03 20:04:23 |
| `trader/armed_executor.go` | `51916172` 2026-09-04 09:48:03 |
| `provider/ninjatrader/tcp_framing.go` | `a1aa1eb6` 2026-09-03 19:41:44 |
| `provider/ninjatrader/order_snapshot.go` | `c84bd247` 2026-09-03 20:04:23 |
| `trader/ninjatrader/tcp_trader.go` | `2c0f005c` 2026-09-02 06:41:43 |
| `trader/class33_boot_sweep.go` | `c84bd247` 2026-09-03 20:04:23 |
| `trader/arms_boot_line.go` | `59d01948` 2026-09-04 08:11:25 |
| `trader/reaper_snapshot.go` | `e54d0ad7` 2026-09-04 09:51:48 |
| `docs/superpowers/SYSTEM-MAP.md` | `a96224dd` 2026-09-04 09:07:37 |
| `docs/superpowers/AUDIT-CHECKLIST.md` | `15340faa` 2026-09-04 13:22:07 |
| `web/src/guide/content/plays.ts` | `59d01948` 2026-09-04 08:11:25 |
| `web/src/guide/content/guards.ts` | `c84bd247` 2026-09-03 20:04:23 |

---

## THE ONE THING TO KNOW BEFORE ANYTHING ELSE

**The two defects masked each other, and shipping D1 without D2 is a REGRESSION, not a partial fix.**

A `StopMarket` whose `StopPrice` is 0 has a trigger of zero, so it rests inert forever. That is the only reason the inverted guard did no damage: on 2026-09-04 it admitted 21 sell stops while the market sat 50.00–102.75 points *below* their trigger — every one already-through — and all 21 sat harmless because the trigger was zero. Correct the C# slot alone and those same 21 placements become 21 orders NinjaTrader acts on the instant they land, roughly 75 points adverse (or 21 `Sell stop or sell stop limit orders can't be placed above the market` rejections — that native error already appears 14 times in the NT8 logs). Either outcome is wrong.

**D1 and D2 are in the same commit and must stay in the same binary.** If the wave is ever split, D2 alone is safe (it converts inert orders into cancels); D1 alone is not.

---

## SECTION G — C1 THROUGH C5

### C1 — CONFIRMED verbatim, at the corrected lines :972-979

The dispatch's original `:972-978` was off by one at each end; `:974-979` as the owner instructed. Read at base:

```
972  bool isLimit    = orderType == "limit" && limitPx > 0;
973  bool isStopEntry = orderType == "stop_entry" && stopPx > 0;
974  OrderType orderT = isLimit ? OrderType.Limit : (isStopEntry ? OrderType.StopMarket : OrderType.Market);
975  double orderPx  = isLimit ? limitPx : (isStopEntry ? stopPx : 0);
976  var entryOrder = submitAccount.CreateOrder(
977      instrument, entryAction, orderT, OrderEntry.Manual,
978      TimeInForce.Day, qty, orderPx, 0, string.Empty, signalId,
979      Core.Globals.MaxDate, null);
```

`Account.CreateOrder`'s positional signature after `quantity` is `(limitPrice, stopPrice, oco, name, gtd, customOrder)`. For a `StopMarket` the trigger sat in `limitPrice` and `stopPrice` received a literal `0`.

**Three independent witnesses, all agreeing, none of them our own logging:**
1. **NinjaTrader's own order log** — `Limit price=29590.5 Stop price=0 … Type='Stop Market'`, and none of the 21 ever reached `Working` or `Filled`.
2. **The broker's book on our wire** — re-read in the repair pass: `nt8_order_snapshots` **id 1664** holds nine entries, every one `{"order_id":…,"symbol":"MNQ","action":"sell","type":"stop","limit_price":29590.5,"quantity":1,"state":"Accepted"}` with `stop_price` **absent** (ids 1480/1483/1485 and 1604-1608 the same). `NT8Order.StopPrice` is `json:"stop_price,omitempty"` (`order_snapshot.go:35`), so absence there means **zero**, not unknown.
3. **The in-file control** — the bracket stop-loss at `:1822-1825` builds a `StopMarket` the other way round (`b.Qty, 0, b.Sl`) and is proven correct by a live fill (2026-09-03, `f2b1eb20-…-sl`, filled at 29355).

**Corrections to the dispatch's framing, both from the investigators and both re-verified here:**
- **The blast radius is wider than one session.** Stop-market *entries* exist on exactly three days. Re-measured in the repair pass, read-only, and stated with the denominators (A21): **23 lifetime submissions** — id 15 (2026-08-30 22:32:55 CT), id 16 (2026-08-31 00:09:35), and the 21 of 2026-09-04. Of those, **22 went to an E7-or-later AddOn and NONE filled**; the single lifetime fill is **id 15**, which filled at **29346.25 on a 28700 trigger** — 646 points away — because that submission hit the **PRE-E7** AddOn, which did not understand the frame and executed it as a **MARKET** order. So: 0 fills in 22 on the build family that parsed the frame, and the one fill that did happen is itself a second, older defect. Every quote of "22 of 22, 0 fills" in this report means the post-E7 family; the extra row is id 15 and it is named wherever the denominator matters.
- **`WHERE kind='stop_entry'` returns 21 and under-reports by one.** Re-verified read-only tonight: `armed_orders` kind census is `''=17, limit=29, stop_entry=21`; the 08-31 twin is **id 16** (`kind=''`, it predates the column, entry 28700, signal `1deb5e23-…`, created 2026-08-31 00:09:35). Any rate quoted off the kind filter has the wrong denominator.
- **The 08-31 order was the E7 capability's own PROOF.** It was accepted as a pass because it rested and cancelled — and a zero-trigger stop rests perfectly, forever. The proof that certified the feature was the bug.
- **No other `CreateOrder` call site is affected.** All four sites checked: `:976` (the defect), `:1137` limit close, `:1822` stop-loss, `:1826` take-profit. The last three are correct.

**Sample ids (A21), re-verified read-only at 21:48 CT 2026-09-05:** the 21 are `38, 62, 65, 67, 70, 73, 75, 77, 79, 81, 83, 85, 87, 89, 91, 93, 95, 97, 99, 101, 102` — all NY / S2 / SHORT / `entry_px` 29591.02, wire trigger 29590.50, **21 cancelled, 0 filled** (`SUM(fill_price>0) = 0`).

### C2 — CONFIRMED, and BOTH sides are inverted, not just the long side

`armed_executor.go:940` called `limitMarketableWrongSide(price, trigger, r.Side)` with the **trigger** in the `entry` argument. The predicate returns `price < entry` for long and `price > entry` for short. For a resting stop that is exactly the VALID condition, so all four cells are backwards:

| # | order | market vs trigger | correct | code did |
|---|---|---|---|---|
| 1 | BUY STOP | price **below** trigger — valid resting | place | **cancelled** |
| 2 | BUY STOP | price **at/above** trigger — already through | cancel | **placed** |
| 3 | SELL STOP | price **above** trigger — valid resting | place | **cancelled** |
| 4 | SELL STOP | price **at/below** trigger — already through | cancel | **placed** |

**Consequence, measured:** cell 4 fired 21 of 21 times on 2026-09-04. Cell 3 has **never been observed** (n=0) — not one tick of that window put price above the trigger — so the "cancels valid ones" half is proven by code reading only, and is stated as such. The stop-side guard has **never fired** in any log (`grep "already traded through"` → 0), while the limit-side guard fires normally (7 times).

**A secondary defect the dispatch did not name:** `throughWord` (`:999-1004`) returns "above" for short and "below" for long — correct for limits, **inverted for stops**. The cancel message at `:942` would have asserted the opposite of what happened. Fixed by giving the stop branch a message that prints the relation the guard actually evaluated (`>=` / `<=`) rather than a shared English word; `throughWord` itself is untouched because the limit branch at `:961` depends on it.

### C3 — CORRECTED. The direction claim holds; both counts do not.

- Reclaim-family **by condition** is **24 of 24 SHORT**, ids `38, 62-102, 103, 104, 105`. The dispatch's "21 of 21" is the **stop-entry subset**.
- **37 of 67 rows (ids 1-37) carry `condition=''`** — the column postdates them — so the family census rests on 30 classifiable rows. Any percentage must name that denominator.
- **`kind='stop_entry'` has never once been authored LONG.** The buy-stop path has **zero production exercise**: every line of the long branch this wave fixes is unexercised by live data, which is why the pins carry both sides and the exact-touch boundary explicitly.
- "The only stop-entry ever placed" is wrong as a *placement* count: **22 distinct broker orders** were placed (one logical arm re-placed 21 times on 09-04, plus id 16).

### C4 — NOT TOUCHED, as ruled

`stopEntryNeedsRetestWindow(r.Condition)` / `stopEntryFallbackDue(...)` and their comment ("a reclaim's buy stop IS the entry — waiting for a no-retest window would miss the reclaim it exists to catch") are byte-identical. `ArmKindFor`, the armable set, `composeArmStop`, the ATR floor, the reaper, prompts, validators, levels, wakes and bars are untouched — confirmed by the scope diff below. The waterfall pullback-limit design is not flipped.

### C5 — SPLIT: one half confirmed exactly, one half REFUTED

- **Arm 33's overshoot is 0.70 points — CONFIRMED exactly.** `29167.50 − 29166.80 = 0.70` (2.8 MNQ ticks), the smallest of the 7 marketable cancels on record. **"1.7" appears nowhere in the record.** The nearest lookalikes are arm 36's 7.70-point overshoot and arm 13's 1.7933 points of R:R headroom — both look like transcriptions that lost their column.
- **"Arm 35 at R:R 2.01 could not absorb a single adverse tick" — REFUTED.** Arm 35 measures **R:R 2.1087** (`140.50 / 66.6285`) with **9.66 ticks** of headroom to the 2.0 floor. "2.01" is not any arm's R:R; it is arm 36's headroom-to-floor **in points** (2.0104).
- **The underlying worry is real and lands elsewhere — REPORTED, NOT FIXED (A31).** The R:R gate (`armed_executor.go:1379-1387`) computes `rr` from the **authored entry**; the trigger is computed ~450 lines away and never re-gated. All 21 stop-entry arms passed the 2.0 floor at their authored entry (2.0058–2.0195) and were **already below it at the trigger the wire actually carried** (1.9775–1.9909) before a single tick of slippage — and a stop-market fills at or beyond its trigger *by construction*, so for this order type the slip is structural. Arm 38 had **0.43 ticks** of headroom. Stated; nothing proposed. No tick cap (separate owner ruling).

---

## THE CHANGE — exact before/after

### D1 · C# — one order, correct slots · `ninjascript/VLTraderTCPClient.cs`

**Before (dev `:975-979`)** → **After (`:1000-1005`)**:

```csharp
- double orderPx  = isLimit ? limitPx : (isStopEntry ? stopPx : 0);
- var entryOrder = submitAccount.CreateOrder(
-     instrument, entryAction, orderT, OrderEntry.Manual,
-     TimeInForce.Day, qty, orderPx, 0, string.Empty, signalId,
-     Core.Globals.MaxDate, null);
+ double limitArg = isLimit ? limitPx : 0;
+ double stopArg  = isStopEntry ? stopPx : 0;
+ var entryOrder = submitAccount.CreateOrder(
+     instrument, entryAction, orderT, OrderEntry.Manual,
+     TimeInForce.Day, qty, limitArg, stopArg, string.Empty, signalId,
+     Core.Globals.MaxDate, null);
```

`:983-985` (the type selection) and the parse at `:767-776` are untouched — both were already correct. Market orders keep `0, 0`. **StopMarket only; no stop-limit introduced.**

**REPAIR — the entry ACTION, three lines above the slots (`:975`, class 77).** The same block chose Buy vs SellShort with an ORDINAL ternary, and the ledger hands it an UPPERCASE side. See the class-77 section below; the fix is the in-file pattern already used four times elsewhere (`:1090`, `:1091`, `:1176`, `:2120`):

```csharp
- var entryAction = side == "long" ? OrderAction.Buy : OrderAction.SellShort;
- var exitAction  = side == "long" ? OrderAction.Sell : OrderAction.BuyToCover;
+ bool isLongSide = string.Equals(side, "long", StringComparison.OrdinalIgnoreCase);
+ var entryAction = isLongSide ? OrderAction.Buy : OrderAction.SellShort;
+ var exitAction  = isLongSide ? OrderAction.Sell : OrderAction.BuyToCover;
```

**A9 — the log now prints what was SENT, not what was parsed** (`:1027-1035`). The old line printed `stop@29590.5` on all 21 malformed submissions, which is precisely why a slot bug survived in a file that logs every placement:

```csharp
- + (isLimit ? (" limit@" + limitPx) : (isStopEntry ? (" stop@" + stopPx) : (" entry≈" + entry)))
+ + " action=" + entryAction + " type=" + orderT
+ + " limitPrice=" + limitArg + " stopPrice=" + stopArg
+ + (isLimit || isStopEntry ? "" : " entry~=" + entry)
```

**REPAIR — the last line was added back.** The first cut rewrote the ONE line shared by market, limit and stop entries, and a MARKET entry has 0 in both price slots by construction: the new line named no price at all on the path that carries essentially every live entry. The reference entry is printed for that case only, and is spelled so it cannot read as a `CreateOrder` argument.

`VL_BUILD_ID` (`:55`) `"2026-09-03-f12"` → **`"2026-09-05-g2"`**. Emission verified, not duplicated — three existing sites, one per frame: hello `:1456`, order_snapshot `:1580`, heartbeat `:2402`.

**Why `g2` and not `g1`.** The first pass minted `2026-09-05-g1`; the repair changed the `.cs`'s BEHAVIOUR (the entry action) afterwards. A build id must name exactly one source, so it moved again. This is safe and was verified rather than assumed: **no `g1` artifact exists anywhere** — the Documents AddOns folder still holds `f12` dated 2026-09-03 21:11 (`md5 b7d6700220f8cbd21ca73d4892400065`), and **100% of 5,963 `nt8_order_snapshots` rows carry `build_id=2026-09-03-f12`**. `ExpectedAddonBuild` and `MinAddonBuildStopSlot` moved in the same commit, and `TestAddonBuildIDMovesInLockstep` reads `VL_BUILD_ID` out of the `.cs` by regexp, so a half-bump reds (proved: mutation M10 below).

### D2/D3 · Go — the stop-side guard, chosen BY KIND · `trader/armed_executor.go`

**Before (dev `:940-943`)** → **After the repair (`:924`, `:952-953`)**:

```go
- offset  := float64(stopEntryOffsetTicks()) * tick
- trigger := r.EntryPx - offset
- if r.Side == "long" { trigger = r.EntryPx + offset }        // DEAD for a stored "LONG"
- if price > 0 && limitMarketableWrongSide(price, trigger, r.Side) {
-     _ = ledger.SetState(r.ID, "cancelled", "trigger already traded through — never placed")
-     at.logWarnf("✕ armed %s stop-entry cancelled — price %.2f already %s the trigger %.2f (never placed)", …throughWord(r.Side)…)
-     continue
- }
+ side := strings.ToLower(strings.TrimSpace(r.Side))          // :924 — ONE canonicalizer
+ …
+ d := decideStopEntry(side, r.EntryPx, float64(stopEntryOffsetTicks())*tick, tick, price)   // :952
+ at.placeOneStopEntry(nt, ledger, r, d, price, now)                                          // :953
```

**The first cut kept the trigger arithmetic and the switch inline.** The repair moved the WHOLE adjudication — canonical side, tick-rounded trigger, verdict, action — into one pure value (`decideStopEntry`) and the dispatch into one method behind two narrow interfaces (`placeOneStopEntry`), because nothing in the repository executed the inline version and three reviewers proved it by mutating it green. See the class-77 and E10 sections.

New, at `:1020-1240`:
- `stopGuardVerdict` — an iota enum whose **zero value is `stopGuardUnknown`**, so a forgotten verdict reads as the safe branch. `String()`'s `default` is `"unknown"`.
- `stopEntryMarketableWrongSide(side, trigger, price)` (`:1050`) — long `price >= trigger`, short `price <= trigger`. **Inclusive**, because a stop AT its trigger fires.
- `stopEntryGuardVerdict(side, trigger, price) (verdict, why)` (`:1066`) — pure, returns the reason in the words the log prints.
- **[repair] `stopEntryAction`** (`:1096`) — a second iota enum whose zero value is `stopEntryNoop`. The first cut's switch enumerated only the two refusals and let **anything unenumerated fall through to the wire**; placement is now reachable only by naming `stopEntryPlace` (`:1213`), with a `default:` that no-ops. Wrong-default in the one branch of this file that can open a position.
- **[repair] `stopEntryDecision` + `decideStopEntry`** (`:1119`, `:1152`) — one value carrying the side the wire will carry, the **tick-rounded** trigger it will carry, the verdict and the action. Three things it fixes beyond testability: the side is folded (class 77), the trigger is rounded BEFORE it is judged, and `EntryPx <= 0` is refused BEFORE the offset arithmetic can manufacture a positive trigger out of a missing entry.
- **[repair] `stopEntryPlacer` / `armStateWriter`** (`:1183`, `:1189`) — the two seams `placeOneStopEntry` (`:1197`) needs, satisfied in production by `*ninjatrader.TCPTrader` and `*store.ArmedOrderStore` (asserted at compile time in the test file). This is what makes A29 provable by execution rather than by grep.

**`limitMarketableWrongSide` is byte-identical and still called at `:956` with `r.EntryPx`.** It is correct for limits, has fired correctly 7 times in production, and `TestLimitMarketableWrongSide` is unedited. The boundary difference is the point of the split: **a limit at its price rests (strict `<`/`>`); a stop at its trigger fires (inclusive `>=`/`<=`).**

Refusal string, in the dispatch's exact shape — rendered live, not quoted from the spec:

```
accepted through (stop side): price 29515.25 <= trigger 29590.50
```

**D3 unknown path (`:1215-1221`):** no cancel, no placement, arm left exactly as it is for the next cycle, `⚠️ armed %s stop-entry NOT adjudicated [guard=stop-side] …` + `store.IncArmRefusal` class `stop_entry:guard_unknown` + `telemetry.IncGateBlock`, deduped by arm-spec via `armRefusalChanged`. Rendered live:

```
stop-side guard not evaluated: no price (trigger 29610.00) — nothing cancelled
```

**A9 on the success path too (`:1236-1237`):** the placement line now names the order type, the trigger, the price, the side and which guard cleared it — `📌 armed S2 → WORKING stop-entry [guard=stop-side verdict=rest action=place] LONG stop-market trigger=29610.50 price=29590.25 signal=… (rests (stop side): price 29590.25 < trigger 29610.50 · no retest in 6 bars, offset 2t)` — the repair added `verdict=` and `action=`, which also gives `stopGuardVerdict.String()` the production call site it lacked (A29).. Previously it named neither the price nor the guard, which is why the 21 admissions left no trace and the investigators had to reconstruct price from a `📏 arm far` line in a different function.

### D5 · the AddOn build floor · `provider/ninjatrader/tcp_framing.go`, `trader/ninjatrader/tcp_trader.go`

No new machinery — `FarSideProven` is reused unchanged. One new constant plus a sentinel:

```go
const MinAddonBuildStopSlot = "2026-09-05-g2"          // tcp_framing.go:241
var   ErrAddonBuildTooOld = errors.New("addon build predates the stop-slot fix")  // :246
```

`PlaceStopEntry`'s first check (`tcp_trader.go:502`) now gates on `MinAddonBuildStopSlot`:

```
ninjatrader/tcp: refusing stop-entry short MNQ trigger=29590.50 qty=1 [guard=far_side_build] —
addon build predates the stop-slot fix (build_id=2026-09-03-f12, need ≥ 2026-09-05-g2):
does not prove stop_entry support; F5-compile + restart the new AddOn
```

The old substring `does not prove stop_entry support` is retained so the existing refusal fixture stays byte-identical. `%q` on the build id is gone — an unheard-from AddOn now renders as `none` via a single shared `BuildIDForLog` (`order_snapshot.go`), used by both the refusal and `AddonBuildLine`, so the two cannot disagree about what "we have not heard from NT8" looks like (A24).

**`ExpectedAddonBuild` bumped in lockstep** to `"2026-09-05-g2"` (`order_snapshot.go:214`), and a test reads `VL_BUILD_ID` **out of the .cs file** rather than restating it, so a half-bump cannot go green.

**[repair] The sentinel is now pinned, not just the prose.** `E6` asserted three substrings of `err.Error()` and never `errors.Is`. Changing the wrap's `%w` to `%v` left the entire wave-B suite green while silently deleting the counting path: `armed_executor.go:1230` falls through to the generic branch, so no `IncArmRefusal`, no `IncGateBlock`, no dedupe — one WARN per armed leg **per cycle** for the whole Go-first window, with `/api/risk/gate-blocks` reading zero refusals while every stop entry is refused. `TestStopEntryRefusedOnPreStopSlotBuild` now asserts the sentinel (RED quoted as M6).

**THE LEXICAL TRAP, and why the date moved.** `FarSideProven` is a bytewise string compare and suffixes are not zero-padded: `"2026-09-03-f9" >= "2026-09-03-f12"` is **TRUE**. A same-date suffix bump would silently open the gate on the next two-digit build. The new floor advances the **ISO date**, and the pin asserts `MinAddonBuildStopSlot[:10] > "2026-09-03"` so a future suffix-only bump fails the suite.

**`FarSideBuildE7` is RETAINED and now gates nothing** — a deliberate, recorded decision, not an oversight. Deleting it would strand the prose comment at `provider/ninjatrader/tcp_server.go:80`, and that file is outside this wave's footprint (A31). Its doc comment now says plainly that it proved the AddOn *parsed* a stop_entry frame and nothing more, and it earns a real ongoing role as the **negative fixture** in `TestStopEntryRefusedOnPreStopSlotBuild`. Flagged here rather than hidden: **this is one constant with zero production call sites**, and the next lane to touch `tcp_server.go` should delete it and the comment together.

### D4 · the boot line · `StopEntryBootLine` (`armed_executor.go:1296`)

Every field READ from the enforcing code, none a literal (A11). Rendered live, all three states:

```
🎯 stop-entry: seam=on · slots=stop_price · guard=stop-side · unknown=no-op · addon build_id=2026-09-05-g2 expected=2026-09-05-g2 match=yes
🎯 stop-entry: seam=on · slots=unproven(addon build) · guard=stop-side · unknown=no-op · addon build_id=2026-09-03-f12 expected=2026-09-05-g2 match=NO
🎯 stop-entry: seam=on · slots=unproven(addon build) · guard=stop-side · unknown=no-op · addon build_id=none expected=2026-09-05-g2 match=NO
```

**THE LINE THIS BOOT WILL ACTUALLY PRINT (owner ruling, 2026-09-05).** The cutover runs with
`STOP_ENTRY_SEAM=off`, so the seam field leads and says what it MEANS, not merely what it is:

```
🎯 stop-entry: seam=OFF — NO stop entry is placed (owner ruling 2026-09-05: cancel-confirmation wave owed; broker-side stacking) · slots=unproven(addon build) · guard=stop-side · unknown=no-op · addon build_id=2026-09-03-f12 expected=2026-09-05-g2 match=NO
```

- `seam` — resolved from the SAME predicate the placement branch consults (`stopEntrySeamOn`, `:935`), stated FIRST because it is the only field that decides whether the others can matter: with the seam off the branch returns **before** the guard, the build floor or the wire is reached. A reader who sees `guard=stop-side` and stops must not conclude that this binary places stop entries. **It does not place any.** The owner's reason is on the line itself: `nt.CancelOrder` reports success on a SEND, not on a confirmation, and the broker was once observed holding **nine** working stop orders for one arm slot (`nt8_order_snapshots` id 1664) on an account capped at two contracts. Until a cancel-confirmation wave lands, the AddOn build floor is not a brake the owner will rely on. Pinned by `TestStopEntryBootLineStatesTheSeam` (mutation-checked: a literal `seam` fails it).

- `slots` — resolved from the **same gate** `PlaceStopEntry` uses. The slot order lives in the C# and this process cannot read it, so the only honest Go-side claim is "the AddOn that answered proves the fix". An unproven build reads `unproven`, never `stop_price`.
- `guard` — resolved by asking `stopEntryGuardVerdict` on the canonical already-through and resting cases. An inverted guard renders `MISROUTED`.
- `unknown` — resolved from the verdict enum on an unevaluable input. A guard that cancelled on ignorance would render `CANCELS`.
- `build_id` — `at.farSideBuildID()`, **the last received frame**, never the source constant. The build half is rendered by `AddonBuildLine`, the one renderer that decides what `match` means.

**Emission site, and why not the obvious one.** It is emitted from the first armed cycle with a bound NT8 trader (`logStopEntryBootLine`, `:1337`, called at `:896`) — **not** hung off `class33_boot_sweep.go:121`. That path latches and is skipped entirely when the sweep defers, when the ledger read fails, or when any cancel failed — three ways for the line to silently not exist (3 of 8 observed 🔌 lines already read `build_id=none` for exactly that reason).

**[repair] AND IT NO LONGER LATCHES ON AN UNKNOWN BUILD.** Three lenses caught this and they were right: the first cut used `LoadOrStore` on "have I emitted", while `farSideBuildID()` is `""` until a hello or heartbeat lands and `armedTrader()` is non-nil with **no far-side connection at all**. A Go restart mid-session, with a plan already in the store, therefore reached `:896` before the AddOn reconnected and pinned `slots=unproven … build_id=none … match=NO` **for the life of the process** — the exact failure the move off the class-33 sweep was supposed to escape, reproduced one layer down. Worse, it made **F2 proof #1 — the owner's stated acceptance criterion — unobtainable without a restart at a lucky moment.** The latch is now keyed on the RENDERED LINE, so the none→proven transition is recorded exactly once and a steady state still prints once. Pinned by `TestStopEntryBootLineReEmitsWhenTheBuildArrives`; RED quoted as M7.

**It is also not, strictly, a boot line, and the Guide now says so:** it needs a plan with arms AND a bound NT8 trader, so a quiet day may not print it at all.

---

## THE REPAIR PASS — what four adversarial lenses found, and the two BLOCKERs they were right about

Four reviews came back: SCOPE/FOOTPRINT, GUARD LOGIC AND UNKNOWN-SAFETY, THE C# WIRE AND THE FAR-SIDE CONTRACT, and TEST QUALITY. Two said boot it; two said no. **The two that said no were right, and about the same thing.** Everything below was re-measured by me before it was acted on — a reviewer can be wrong, and two of them were, on separate points recorded in the VERIFICATION RECORD.

### CLASS 77 — the canonicalizer landed at the write; two consumers stayed ordinal

`store.UpsertArm` uppercases `armed_orders.side` at the write (`store/armed_orders.go:181`, class 28, owner ruling 2026-09-03, commit `a05c6ca1`, **an ancestor of the deployed rev `36648655`** — so this is live, not hypothetical). Two consumers still compared that column to the lowercase literal `"long"`:

```go
trader/armed_executor.go:936 (dev)   if r.Side == "long" { trigger = r.EntryPx + offset }
```
```csharp
ninjascript/VLTraderTCPClient.cs:964 (dev)   var entryAction = side == "long" ? OrderAction.Buy : OrderAction.SellShort;
```

**Two consequences, both silent, both directional.**

1. A LONG stop entry was built at `entry − offset` — a BUY stop **below** the level it exists to break above — and the new stop-side guard then adjudicated that mis-signed trigger with long semantics. Depending on where price sat it either placed a buy stop on the wrong side of the authored level, or **cancelled an arm that should have rested**. And the destructive half runs at `:945` (dev), **ahead of the D5 build refusal**, so the Go binary alone — old AddOn still loaded, no order ever attempted — could permanently cancel long stop-entry arms.
2. `PlaceStopEntry`/`PlaceLimitEntry` copy the side into the frame verbatim (`tcp_trader.go:457`, `:529` — `Side: side`, no fold; the sibling market path at `:388` even carries the comment `// lowercase per spec L4390`), and the AddOn's ternary answers **SellShort to every side it does not recognise**. `"LONG" != "long"`, so **a LONG entry — limit OR stop — would have been submitted to NinjaTrader as a live SELL.**

**Why nobody had seen it, and the reason is the trap.** Census, read-only, `sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro"`:

```
side |kind       |n |first CT             |last CT
long |           |10|2026-08-27 11:31:36  |2026-09-02 10:30:30
short|           | 7|2026-08-27 23:52:52  |2026-08-31 00:09:35
short|limit      |11|2026-08-30 19:50:01  |2026-09-03 11:58:33
long |limit      | 9|2026-08-30 23:10:36  |2026-09-02 00:16:32
SHORT|limit      | 9|2026-09-04 09:10:08  |2026-09-04 12:11:02
SHORT|stop_entry |21|2026-09-04 10:05:00  |2026-09-04 10:53:11
```

**Every row written since the canonicalizer is SHORT, and for SHORT the wrong branch is accidentally the right answer.** Zero `LONG` rows have ever existed. A census of the column reads "no problem" precisely because the half that breaks has no rows — and `store/armed_orders.go:36` still comments the field `// long | short`, which is true of the type and false of the data.

**And D1 is what makes it reachable.** The reclaim buy stop that C4's protected comment exists for ("a reclaim's buy stop IS the entry") is LONG by construction. The first LONG stop entry this wave makes real is the first one that hits both halves.

**THE FIX — one canonicalizer, where the value enters the placement path.**

```go
// trader/armed_executor.go:924, inside `case "armed":`
side := strings.ToLower(strings.TrimSpace(r.Side))
```

and that one value feeds the trigger, the guard, the log, `PlaceStopEntry` **and** `PlaceLimitEntry`. `decideStopEntry` folds again, defensively, because it is a pure function that a test drives directly. The AddOn folds on arrival too (`:975`), because the far side is deployed separately and will at some point be running the version that does not — and because a ternary whose unrecognised branch opens a position in the OPPOSITE direction must not be the last line of defence.

**The limit path was fixed in the same line, deliberately, and here is the reason.** The footprint grants "the placement branch choosing limit vs stop-entry", and the two branches share one row loop; splitting the fold would have left `PlaceLimitEntry` handing raw `"LONG"` to the old AddOn during **precisely the Go-first window this wave designs for**. For SHORT nothing changes (SellShort either way). For LONG the limit path stops submitting a live SELL. **This is a behaviour change on a live path and it is stated here rather than buried:** it is a fix to a pre-existing defect, it has never been exercised (no LONG row post-canonicalizer), and it is fail-*correct* rather than fail-closed. If the owner would rather ship the stop half alone, the limit half is `git revert` of one token.

**Checklist class 77 is appended in this PR** (dev's highest is 75; this branch added 76, and 77 is next — no other lane's entry renumbered, A16).

### THE OTHER BLOCKER-CLASS FINDING: nothing executed the decision

Three lenses independently mutated the call site and got a green suite. I reproduced it and agree: `grep -rn runArmedPlacement --include=*_test.go` finds only `class33_boot_sweep_test.go`, which is a **source grep**. `TestStopEntryGuardHasAProductionCallSite` asserted `strings.Contains` over `armed_executor.go`'s TEXT — it proves the call is spelled, not that the verdict governs anything. Keeping the call and overwriting its answer, or deleting the unknown arm, both passed.

**The wave's own thesis is that a defect survived because every layer reported what it INTENDED and nothing read back what was done. Its test set reproduced that shape one level up.** That is the finding I care about most and it is fixed by construction, not by another grep: `trader/wave_b_placement_test.go` drives `decideStopEntry` **and** `placeOneStopEntry` over the casing the store returns, with a fake placer and a fake ledger, and asserts what reached each. Twelve mutations are quoted RED below; the first three are exactly the ones the reviewers ran.

### E10 — the mutation battery (A8, RED before GREEN, every one re-run by me at this head)

| # | mutation | red |
|---|---|---|
| M1 | `decideStopEntry` always returns REST (keep the call, discard the answer — the 2026-09-04 defect, restored) | `stored LONG already through: action place, want cancel` · `stored SHORT already through (09-04 id 38 price): action place, want cancel` · `an already-through stop reached the wire: [{symbol:MNQ side:short qty:1 trigger:29590.5 …}]` |
| M2 | the `default:` (no-op) arm of `placeOneStopEntry` deleted | `no price: an unadjudicated arm reached the wire: [{… trigger:29610.5 …}]` · `no price: an unadjudicated arm was written to (cancel-on-ignorance): [{id:300 state:working reason:}]` (×3 inputs) |
| M3 | `decideStopEntry` stops folding the side (**the blocker, exactly as it stood at 98c28dcd**) | `stored LONG rests below trigger: wire side "LONG", want "long" — an uppercase side reaches a C# ternary that reads \`side == "long" ? Buy : SellShort\`` · `trigger 0.0000, want 29610.5000` · `verdict unknown, want rest` |
| M4 | the trigger is judged UNROUNDED | `stored SHORT rests above trigger: trigger 29590.5200, want 29590.5000` · `the judged trigger 29591.5200 is not on a tick boundary — the wire will round it and judge≠sent` |
| M5 | placement refusal key back to 0-based, un-namespaced | `placement key "2026-09-05:NY:t1:3:S2:leg1" is byte-identical to the arm-gate key for leg 0` |
| M6 | `%w` → `%v` in the build refusal | `the refusal does not wrap ErrAddonBuildTooOld — the caller cannot count it: ninjatrader/tcp: refusing stop-entry short MNQ trigger=29590.50 …` |
| M7 | boot line latches on HAVING EMITTED again (`LoadOrStore`) | `the boot line latches on HAVING EMITTED again — an unknown build would pin match=NO for the life of the process` |
| M8 | C# entry `CreateOrder` back to dev's single `orderPx` | `entry CreateOrder passes a literal 0 into the stopPrice slot (limit="orderPx" stop="0") — a StopMarket built this way has a ZERO trigger and rests inert forever` |
| M9 | C# entry action back to the ordinal ternary | `the AddOn still decides the order DIRECTION with an ordinal \`side == "long"\` — an uppercase side submits a live SellShort` |
| M10 | half-bump: `VL_BUILD_ID` moves, the Go constants do not | `VL_BUILD_ID="2026-09-06-h1" but ExpectedAddonBuild="2026-09-05-g2" — every boot would print match=NO` · `… the stop-slot gate needs >= "2026-09-05-g2" — the fixed AddOn would refuse itself` |
| M11 | the placement branch hands the RAW ledger side to the wire again | `the placement branch hands the RAW ledger side to the wire — an uppercase LONG submits as a live SellShort` |
| M12 | the 8-case guard table is truncated | `the four cells plus both boundaries plus both case-folds = 8 cases, got 7` |
| M13 | SYSTEM-MAP cites a stale line for `decideStopEntry` | `SYSTEM-MAP says decideStopEntry is at armed_executor.go:1063, but that line reads: …` |
| M14 | the MAPCHECK region is deleted | `the MAPCHECK region is gone — the map section this wave owns is no longer checked` |

Every mutation was applied in this worktree, run, and reverted; `git status --porcelain` was empty afterwards each time. **One caveat I own:** during M13 I reverted with `git checkout -- docs/superpowers/SYSTEM-MAP.md` while the map's corrections were still uncommitted, and lost them. They were redone from the same script and re-verified by the contract test. This is the class-45 shape (a revert that silently discards work) and the lesson is the standing one: commit before you mutate.

### E11 — the class-75 contract test that did not exist

Three lenses found that the SYSTEM-MAP text this wave wrote shipped **six wrong line references, including the one it claimed to have corrected**. I re-derived every one and they were right — and found more: Wave B had ALSO shifted the entire gate-leg block by 219 lines (`armGateVerdictFor` cited `:1316`, actually `:1701`, plus seven sub-refs), and the knob line still quoted `"no order_update within stale window (reconnect/reconcile)"`, a cancel string the reaper wave **deleted from the code**.

Nothing caught any of it, because class 75 has no enforcing test. It has one now, for the region this wave owns: `MAPCHECK` markers delimit the two paragraphs, and `TestSystemMapStopEntryRefsResolve` **reads the numbers out of the map** (so the map stays the single source and no third copy exists to drift) and resolves each backticked symbol against `armed_executor.go` at the cited line. **It found a bad reference on its first run.** It is deliberately scoped to this wave's region — a repo-wide version would red the suite on other lanes' drift, which is not mine to gate.

## THE TESTS — every pin RED before GREEN (A8)

Baseline at `cb23d9eb`: `go build ./...` rc=0, full suite green. Any red below is this wave's.

### E3 — the four cells · `trader/wave_b_stop_guard_test.go`

RED, run by **replacing the new predicate's body with the production expression as it stood** (`limitMarketableWrongSide(price, trigger, side)`) — **all 8 cases inverted, zero exceptions.** A reviewer correctly noted this is a **mutation-red on new code**, not a pre-fix red (the shipped test calls a function that does not exist at `cb23d9eb`, so reverting production yields a compile error). Labelled here exactly as E5's identical weakness was. The semantics it pins are real and the red is reproducible on demand:

```
--- FAIL: TestStopEntryMarketableWrongSide (0.00s)
    buy stop rests below: side="long" trigger=29610.00 price=29590.25: got through=true want false
    buy stop through above: side="long" trigger=29610.00 price=29612.25: got through=false want true
    buy stop exact touch: side="long" trigger=29610.00 price=29610.00: got through=false want true
    sell stop rests above: side="short" trigger=29590.50 price=29650.00: got through=true want false
    sell stop through below (09-04 id 38): side="short" trigger=29590.50 price=29515.25: got through=false want true
    sell stop exact touch: side="short" trigger=29590.50 price=29590.50: got through=false want true
    case folded long / case folded short: … got through=false want true
```
**GREEN** after D2. The two exact-touch cases are the ones a strict mirror of the limit predicate would still get wrong. **[repair]** The table had no length assertion — it would have passed when emptied — while its sibling replay pinned its own 21. Fixed; RED quoted as M12.

### E5 — UNKNOWN never cancels

Compile-red before D3 (`undefined: stopEntryGuardVerdict`), **GREEN** after: five unevaluable inputs (no price, no trigger, neither, unknown side, empty side) all return `stopGuardUnknown` with a non-empty reason, the enum's zero value is UNKNOWN, and `String()` says "unknown".

### E2 — the C# slot · `trader/wave_b_addon_slot_test.go`

**There is no C# harness in this repo** — no `.csproj`, no `.sln`, no `*Test*.cs`; NinjaScript compiles only inside NT8. **I chose the source-grep pin and say so plainly:** it proves the shipped `.cs` text, and D5's build gate proves the DLL NT8 actually loaded. Neither alone is sufficient; together they are.

RED:
```
--- FAIL: TestAddonEntryOrderPassesTheTriggerInTheStopSlot (0.00s)
    entry CreateOrder passes a literal 0 into the stopPrice slot (limit="orderPx" stop="0")
    — a StopMarket built this way has a ZERO trigger and rests inert forever
--- FAIL: TestAddonSubmissionLogNamesTheFourValuesItSent (0.00s)
    the submission log does not name action= / type= / limitPrice= / stopPrice= …
```
**GREEN** after D1. The pin also asserts each slot is selected by its own branch (`isLimit` / `isStopEntry`), not merely that the two names differ, and that the in-file bracket-SL control still has its shape.

### E6 — the build floor · `trader/ninjatrader/stop_entry_wire_test.go`

RED:
```
--- FAIL: TestStopEntryRefusedOnPreStopSlotBuild (0.17s)
    a stop entry was sent to an AddOn that predates the stop-slot fix — it would go out with a ZERO trigger
```
**GREEN** after D5. The existing `TestPlaceStopEntryFrameOnLoopback` was reseeded from `FarSideBuildE7` to `MinAddonBuildStopSlot` **by import** (A24 — a fixture must not hold its own copy of a constant).

### E1 — the payload · **honestly not a red→green pin**

The Go payload was never the defect. `PlaceStopEntry` already rounds the trigger and sends `OrderType:"stop_entry", StopPrice: entry`, and `tcp_trader.go` / `SignalPayload` needed **no change** — I looked for a reason to override the owner's ruling and found none. Rather than write a pin that cannot fail, I **strengthened the existing loopback test** with the assertion it lacked: on both sides, a stop entry's `limit_price` must be **0**, so a trigger leaking into the limit slot would rebuild the 2026-09-04 defect from the Go end. It is a regression guard and is **not counted as a red→green pin**.

### E7 — the replay · `trader/wave_b_replay_test.go` · **CORRECTS THE DISPATCH**

The dispatch expects "ONE well-formed submission". Replayed against the **investigators' measured** 21 cycle prices (their number, not the dispatch's framing of one arm) the fixed path yields **ZERO submissions**: at every one of the 21 cycles the market was 50.00–102.75 points below a sell-stop trigger, so the correct answer is 21 cancels and no order at all. A well-formed submission appears only in the valid-side replay (price 29650.00 above the trigger → rests → one placement). Arm 35's LIMIT case replays byte-identically, including that a limit **at** its own price still rests while a stop **at** its trigger does not.

Proven to bite by restoring the inverted semantics — 21 real failures, first three:
```
--- FAIL: TestReplayNY0904S2ThroughTheFixedGuard (0.00s)
    cycle 0: price 29515.25 is 75.25 pts BELOW a sell-stop trigger 29590.50 and must never be placed
    cycle 1: price 29524.75 is 65.75 pts BELOW a sell-stop trigger 29590.50 and must never be placed
    cycle 2: price 29540.50 is 50.00 pts BELOW a sell-stop trigger 29590.50 and must never be placed
```

### E8 — A29, built ≠ wired · **RETAINED AS A TRIPWIRE, DEMOTED AS EVIDENCE**

`TestStopEntryGuardHasAProductionCallSite` reads `armed_executor.go` and requires the stop branch to call `decideStopEntry` and dispatch through `placeOneStopEntry`, to **not** call the limit predicate with the trigger, to keep `limitMarketableWrongSide(price, r.EntryPx, side)` on the limit branch, to distinguish `ErrAddonBuildTooOld`, and — added in the repair — to **never hand `r.Side` raw to either wire call**. Proven to bite by reverting the call site, and by M11 above.

**But it is a source grep, and the repair pass demoted it accordingly.** It is brittle to a harmless rename and blind to semantic breakage that preserves the string; two reviewer mutations preserved every asserted substring and left the suite green. The A29 guarantee now rests on **E10, executed**, with this as a cheap tripwire beside it. The report's first pass over-claimed here and the claim is withdrawn.

### E10-pin — the caller-level test the first cut lacked · `trader/wave_b_placement_test.go`

Five behavioural sub-tests over `placeOneStopEntry`, plus a 13-row decision table over the STORED casing:

- **through → cancel, and nothing reaches the wire.** Row 38's real numbers: exactly one `SetState(38, "cancelled", …)` whose reason contains `accepted through (stop side)` and `never placed`; zero `PlaceStopEntry` calls.
- **rest → exactly one placement**, with `side == "long"` (not `"LONG"`), `trigger == 29610.50` (a BUY stop **above** the level), `trigger == d.Trigger` (judged == sent), qty/SL/TP forwarded intact, `SetSignal` recorded, exactly one `working` transition.
- **unknown → the arm is untouched**, for all three unevaluable inputs: zero wire calls AND **zero ledger writes**. This is the assertion that makes D3 a property rather than a comment.
- **a build refusal never touches the ledger** — the call happens, the error is `errors.Is`-matched upstream, and the row stays armed for the next cycle.
- **a transport failure also leaves the arm alone.**
- **`TestPlacementRefusalKeyCannotAliasTheArmGate`** — for leg indices 0, 1 and 2, the placement key is keyed and is never byte-identical to the arm-gate key for any leg of the same plan/version/scenario.
- **`TestTickSourcesAgree`** — `market.FuturesTickSize` and `ninjatrader.InstrumentTickSize` must agree for NQ/MNQ/ES/MES/RTY/M2K/YM/MYM, because judge-equals-sent holds only while they do.

Compile-time proof that the seams are the production types' own shape:
```go
var _ stopEntryPlacer = (*ntTrader.TCPTrader)(nil)
var _ armStateWriter  = (*store.ArmedOrderStore)(nil)
```

### D4 boot-line pin — proven to bite by replacing the resolved `slots` with a literal:
```
--- FAIL: TestStopEntryBootLineIsRead (0.00s)
    a pre-stop-slot build must not claim slots=stop_price: 🎯 stop-entry: slots=stop_price … build_id=2026-09-03-f12 … match=NO
    an unheard-from AddOn cannot prove the slot: 🎯 stop-entry: slots=stop_price … build_id=none … match=NO
```

### E4 — limits unchanged
`TestLimitMarketableWrongSide` is **unedited and passing**. `limitMarketableWrongSide` is byte-identical. **There is no placement golden in `trader/` or `provider/`** — every golden in this repo is a kernel *prompt* golden — so E4's "golden diff must be EMPTY" is satisfied as: `git diff origin/dev...HEAD -- kernel/testdata` returns **0 files**, and `go test ./kernel/` is ok. No golden was regenerated; this wave touches no prompt.

### E9 — the full suite, at this head

Re-run in full at the REPAIR head:

| gate | command | result |
|---|---|---|
| build | `go build ./...` | rc=0 |
| vet | `go vet ./...` | rc=0 |
| Go suite | `go test ./...` | **`grep -c '^FAIL'` = 0 · `grep -c '^ok'` = 28** · `ok nofx/trader 54.440s` · `ok nofx/trader/ninjatrader 8.603s` · `ok nofx/provider/ninjatrader 18.617s` · `ok nofx/store 23.598s` · `ok nofx/kernel 1.770s` |
| goldens | `go run ./cmd/nq_smoke prompt` | `OK prompt: system=1530 bytes, user=504 bytes` rc=0 |
| goldens | `go run ./cmd/nq_smoke roundtrip` | `OK roundtrip: signal->fill in 400.893174ms (direction=LONG entry=21500.00)` rc=0 |
| goldens | `git diff --name-only origin/dev...HEAD -- kernel/testdata` | **0 files** |
| tsc | `cd web && npx tsc --noEmit -p tsconfig.json` | rc=0 |
| vite | `cd web && npm run build` | `✓ built in 4.09s` |
| vitest | `cd web && npm test` | **Test Files 44 passed (44) · Tests 345 passed (345)** |

(`web/node_modules` was absent in this worktree; `npm ci` was run **in the worktree only** — the main tree was never touched.)

---

## F2 — ATTEMPTED 2026-09-06 00:04 CT. SOURCE DEPLOYED, COMPILE DEFERRED BY NT8 MAINTENANCE.

**Owner GO received 2026-09-06 ~00:02 CT** ("run F2 now — window open, book empty, backups verified"),
with the standing condition restated: **the seam stays OFF regardless — F2 proves the order SHAPE, it
does not enable stop entries.**

**Window and flat gate, quoted (A7).** Sunday 2026-09-06 00:01 CT, CME closed. Open positions **0**;
non-terminal arms **0**; broker book empty across the last ten `nt8_order_snapshots`
(`0 orders / 0 working`), freshest id 6152 `Sim101` build `2026-09-03-f12` at 00:00:55 CT.

**A13 — BOTH HALVES BACKED UP BEFORE ANYTHING WAS COPIED IN**, md5-verified against the live files
(the AddOn has no git revert; this is the only rollback that exists):

| file | md5 | build it HOLDS |
|---|---|---|
| `~/nofx-backups/nt8-addon/VLTraderTCPClient.2026-09-03-f12.cs` | `b7d6700220f8cbd21ca73d4892400065` | 2026-09-03-f12 |
| `~/nofx-backups/nt8-addon/NinjaTrader.Custom.2026-09-03-f12.dll` | `7c2789ff35d96beb73dd740a29b913f1` | 2026-09-03-f12 |

**What was actually done, and it is only the first half of F2:**

1. `ninjascript/VLTraderTCPClient.cs` from the deployed head copied to
   `…/Documents/NinjaTrader 8/bin/Custom/AddOns/VLTraderTCPClient.cs`.
   **Verified byte-identical after the copy:** repo `34efc3f85d0a775247f6c2f2ea576224` ==
   live `34efc3f85d0a775247f6c2f2ea576224`. Build id on the far side moved
   `2026-09-03-f12` → **`2026-09-05-g2`** in source.
2. NinjaTrader stopped (pid 14964) and relaunched (pid 45436, 00:04:56 CT).
3. **It never compiled.** The new process parked at its **`Welcome`** window and stayed there for
   nine minutes, responsive, with a 61-byte log containing one `Session Break` line and zero
   `VLTrader` activity. `NinjaTrader.Custom.dll` is **unchanged at `7c2789ff…`** — which is exactly
   why A13 requires the DLL's md5: it is the only thing that distinguishes "the F5 succeeded" from
   "NT8 silently kept the old binary". Here it says, correctly, that no compile happened.
   **Owner: NT8 is in its maintenance window and cannot start.**

**ATTRIBUTION, CORRECTED ON THE RECORD.** I first told the owner the cause was "plainly my
force-kill". That was wrong, and the logs say so. The NT8 feed was **already dead before I touched
anything**: at 00:00:23 CT — four minutes BEFORE the stop — the AddOn's own watchdog logged
`most-stale MNQ|1M bar age 398s; dead-subscriptions=28`, and `There was a problem authenticating
account Google Simulation` had been repeating every ~30 s since 00:00:14. The identical pattern
appears at the identical time on 2026-09-05 (first line 00:00:14:032), i.e. it is the nightly
maintenance window, not this wave. My restart did not cause the outage; it removed the one thing
masking it — a process that was already connected and could not have re-authenticated either.

**THE COMPILE IS DEFERRED, NOT SKIPPED.** NinjaTrader compiles NinjaScript on startup, and the
corrected source is in place and verified. **The next time NT8 starts after maintenance it will
compile `2026-09-05-g2` with no further action.** Nothing was restored: the restore rule is for a
COMPILE ERROR, and there was no compile error — there was no compile. Restoring would have put the
defect back for no benefit, since the AddOn is not loaded either way.

**RISK WHILE THE BRIDGE IS DOWN: bounded, and the Go half is the reason.** Market closed until
17:00 CT Sunday. Zero positions, zero arms, book empty. `STOP_ENTRY_SEAM=off` is in force in the
running process (`🎛 entry law: … stop_entry_seam=off`), so no stop entry is placed at all; and even
with the seam on, `PlaceStopEntry` refuses any build below `MinAddonBuildStopSlot`. Limits are
unaffected by both. **The one thing the owner must ensure is that NT8 is up and the AddOn connected
well before 17:00 CT Sunday** — without it there are no bars and no execution path, which is a
data/feed outage, not a Wave B regression.

## THE FOUR LIVE PROOF LINES: **NOT YET RECEIVED — the C# half is NOT live (A20/class 6)**

None of these can exist until the owner copies `ninjascript/VLTraderTCPClient.cs` to the Documents AddOns folder, F5-compiles, and **fully restarts NT8** — and then a CME session opens. All four are owed:

1. **NOT YET RECEIVED — the C# half is NOT live (A20/class 6)** — the line reading `🎯 stop-entry: slots=stop_price · guard=stop-side · unknown=no-op · addon build_id=2026-09-05-g2 expected=2026-09-05-g2 match=yes`. Until the recompile it will correctly read `slots=unproven(addon build) … build_id=2026-09-03-f12 … match=NO`. **[repair]** It is now obtainable without a lucky restart: the line re-emits when the build id arrives, so a Go boot that beats the AddOn's first frame prints the unproven form and then the proven one, instead of latching the first forever.
2. **NOT YET RECEIVED — the C# half is NOT live (A20/class 6)** — an NT8 order-log line for a stop ENTRY reading `Limit price=0 Stop price=<trigger> … Type='Stop Market'`. **This is the acceptance criterion, and it must be the price slot — not "it rested and cancelled".** That was the 2026-08-31 criterion and the order that passed it carried `Stop price=0`.
3. **NOT YET RECEIVED — the C# half is NOT live (A20/class 6)** — that order reaching NT8 state **`Working`**. No stop entry has ever reached `Working` in the system's history (0 of the 22 post-E7 submissions).
4. **NOT YET RECEIVED — the C# half is NOT live (A20/class 6)** — an `order_snapshot` frame carrying `build_id=2026-09-05-g2` with the entry's **non-zero `stop_price`** parsed back off the wire. Assert the parsed struct field or `limit_price == 0` beside it: `stop_price` is `omitempty`, so a zero **vanishes** from the persisted JSON and an assertion written as "stop_price is absent" would pass on the bug.

Read-only, re-measured in the repair pass: **only `2026-09-03-f12` has ever been received** — `SELECT MIN(build_id), MAX(build_id), COUNT(*) FROM nt8_order_snapshots` → `2026-09-03-f12 | 2026-09-03-f12 | 5963`, i.e. 100% of every frame ever persisted.

**A FIFTH PROOF IS NOW OWED, and it is the one class 77 costs.** **NOT YET RECEIVED — the C# half is NOT live (A20/class 6):** an AddOn submission log line for a **LONG** entry reading `action=Buy`. Every armed order the system has placed since the canonicalizer landed has been SHORT, so `OrderAction.Buy` on the armed path has **never been observed** — not once, either before or after this wave. Until a long arm is submitted and logged, the class-77 fix is proven by source and by the Go-side table test, and by nothing on the far side. The AddOn currently deployed is byte-identical to the repo at base, so NT8 genuinely holds the defect right now, and from this commit the repo and the deployed DLL diverge until the recompile.

---

## A15 — LIVE-SURFACE TRUTH

- **Nothing is deployed.** The running binary is rev `36648655`, booted 2026-09-04 13:25:47 CT. It has **none** of this. Today is Saturday 2026-09-05, CME closed, engine idle.
- **`STOP_ENTRY_SEAM=on` in `/home/hoang/nofx/.env`.** This is a LIVE path, not a dormant one — that is how 21 malformed orders reached a broker.
- **The cutover state is clean.** No working stop entry exists anywhere. The only non-terminal rows are ids 92/94/96/98 (`superseded`) and **104, 105** (`armed`), all `kind='limit'`, none carrying a `signal_id` — so nothing is resting at the broker.
- **After a Go-side deploy but before the NT8 recompile — EXACTLY WHAT THE OWNER WILL STILL SEE WRONG.** This is the honest list, not the reassuring one:
  1. **Every stop entry is REFUSED**, counted as `stop_entry:addon_build`, and logged `📌 armed <S> stop-entry REFUSED [guard=far_side_build verdict=rest] …`. Correct and intended, but it means **zero stop entries trade** until the F5. If the owner reads "refused" as a bug, it is not.
  2. **The 🎯 line reads `slots=unproven(addon build) … build_id=2026-09-03-f12 … match=NO`**, and the 🔌 class-33 line reads `match=NO` too, because `ExpectedAddonBuild` moved with the source. **Both are supposed to say NO.** They will keep saying NO until NT8 is recompiled and restarted — a compile alone is not enough (AddOns do not hot-reload).
  3. **An already-through stop arm is CANCELLED, not held.** The wrong-way guard runs BEFORE the build floor, so during this window a stop arm the market has run past is terminal-cancelled rather than preserved by the refusal. Refused = still armed, retried next cycle. Cancelled = gone. The Guide now says this; it is the one asymmetry of the go-first order.
  4. **`GUIDE_BUILT_REV` still reads `36648655`** until the owner bumps it at the cutover, so the Guide banner will report drift against the new binary. That is the boot-5 rule working, not a defect.
  5. **The AddOn's own submission log is unchanged** — it still prints `stop@29590.5` (the parsed variable) rather than the four arguments, because that line lives in the DLL. Do not read the running AddOn's log as evidence about the slots until after the F5.
  6. **`FarSideBuildE7` now gates nothing**, and `provider/ninjatrader/tcp_server.go:80` still tells the next reader that frame types are gated on it. That comment is now FALSE. It is a four-word prose fix in a file outside this wave's footprint, so it is **left alone and recorded here** — the next lane to touch `tcp_server.go` should delete the constant and the comment together.
  7. **A LONG limit arm now submits as a Buy where before it would have submitted a SellShort.** No such arm has existed since 2026-09-03, so nothing changes on today's data — but it is a live behaviour change and the owner should know it shipped in this wave rather than discover it.
- **`GUIDE_BUILT_REV` is NOT bumped** (`web/src/guide/types.ts:6`, still `36648655…`). It must equal the **deployed** rev, which does not exist yet; setting it to a sha that never ships would make the drift banner lie in the other direction. **Owner-side at cutover: bump it and rebuild `web/dist` BEFORE the boot** (boot-5 rule), not in the marker commit. The Guide's *content* is updated in this PR, satisfying the substantive half of the GUIDE CONTENT LAW. Every log string quoted in the new Guide prose was checked verbatim against the shipped format strings.

---

## WHAT IS NOW POSSIBLE THAT WAS NOT

1. **A stop entry can actually work.** Lifetime fill rate was 0/22 — the feature has never once placed an order NinjaTrader could trigger. After the recompile it can.
2. **A valid stop entry can be placed at all.** The inverted guard cancelled 100% of the valid cases; cell 3 never occurred live only because the market never offered one.
3. **An already-through stop is refused instead of admitted.** The 21 admissions of 2026-09-04 would now be 21 cancels naming the relation: `accepted through (stop side): price 29515.25 <= trigger 29590.50`.
4. **A guard that cannot answer does nothing, out loud, and is counted.** Previously an unevaluable state fell through to placement.
5. **A stale AddOn is refused rather than sent something it will mis-execute.** The old floor could not refuse the malformed build — `"2026-09-03-f12" >= "2026-08-30-e7"` is true.
6. **The AddOn's own log can now witness the defect class.** It prints the arguments handed to `CreateOrder`, so the AddOn log and NT8's order log must agree or disagree visibly.
7. **The posture is stated at boot from resolved values**, so an inverted guard, an unproven build or a cancel-on-unknown each change the line rather than leaving it lying — and it now RE-states it when the far side finally answers, instead of pinning the first thing it saw.
8. **[repair] A LONG entry can reach the broker as a BUY.** This is the headline of the repair pass and it is not a refinement of 1-7: **a long continuation entry — the reclaim buy stop this feature exists for — would have been submitted to NinjaTrader as a live SELL**, and so would a long LIMIT arm, because the ledger canonicalizes `side` to uppercase and two consumers compared it ordinally to `"long"`. Nothing had caught it because every armed row written since the canonicalizer landed is SHORT, for which the wrong branch is accidentally right. Both consumers now fold. That is the thing that is possible today and was not: **the long half of this feature can execute at all, in the direction the plan authored.**
9. **[repair] The number judged is the number sent.** The trigger is tick-rounded before the guard reads it, so the guard can no longer call "resting" an order the broker receives at the market.
10. **[repair] The placement decision is executable by a test**, which is the only reason any of 1-9 can be believed after the next edit. The first cut's wiring guarantee was a source grep, and three reviewers independently mutated it green.

---

## ROLLBACK — both halves, independently

**Go half (safe to revert alone).** `git revert` the code commit, or roll back the binary and `deploy/RELEASE`. Reverting Go alone with the NEW AddOn compiled in NT8 restores the inverted guard and the old build floor — the AddOn would then build correct orders that the inverted guard admits at the wrong moments. **Do not do this while a plan can arm a reclaim; disable `STOP_ENTRY_SEAM` first** (`STOP_ENTRY_SEAM=off` in `.env` + restart) — limits are unaffected.

**C# half — AND THERE IS NO `git revert` FOR IT.** The artifact that executes is `NinjaTrader.Custom.dll`, compiled inside NT8 from a copy of the `.cs` in the Documents folder. Neither is in git; both are overwritten in place by the deploy. **Back both up, BY BUILD ID, WITH md5s, BEFORE the F5** — this is a precondition of F2, not a nicety:

```
# BEFORE copying anything in (run from WSL; read-only until the cp):
NT="/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom"
mkdir -p ~/nt8-addon-backups/2026-09-03-f12
cp "$NT/AddOns/VLTraderTCPClient.cs" ~/nt8-addon-backups/2026-09-03-f12/
cp "$NT/NinjaTrader.Custom.dll"      ~/nt8-addon-backups/2026-09-03-f12/
md5sum ~/nt8-addon-backups/2026-09-03-f12/* | tee ~/nt8-addon-backups/2026-09-03-f12/MD5SUMS
```

**The md5s to expect, measured read-only 2026-09-05 22:5x CT (nothing was written):**

| artifact | build | mtime | md5 |
|---|---|---|---|
| `AddOns/VLTraderTCPClient.cs` (deployed, to be backed up) | `2026-09-03-f12` | Sep 3 21:11 | `b7d6700220f8cbd21ca73d4892400065` |
| `NinjaTrader.Custom.dll` (deployed, to be backed up) | compiled from f12 | Sep 3 21:21 | `7c2789ff35d96beb73dd740a29b913f1` |
| `ninjascript/VLTraderTCPClient.cs` (this branch, to be deployed) | `2026-09-05-g2` | — | `34efc3f85d0a775247f6c2f2ea576224` |

**A13 applies: name the backup for the rev it HOLDS (`2026-09-03-f12`), not for the rev it is being taken before.** After the F5, take the same pair again into `~/nt8-addon-backups/2026-09-05-g2/` — the DLL is the only artifact that proves what NT8 actually compiled, and its md5 is the only thing that will distinguish "the F5 succeeded" from "NT8 silently kept the old binary" if the boot line disagrees.

**To roll back:** restore the `2026-09-03-f12` pair over the Documents folder, F5-compile, **fully restart NT8**. The reverted AddOn reports `2026-09-03-f12`, which the new Go floor **refuses** — so with the Go half deployed, rolling back the AddOn fails CLOSED: no stop entries at all, loudly, counted, with limits untouched. That is the designed behaviour and needs no coordination. **One caveat class 77 adds:** the reverted AddOn also loses the case fold, so it is again a ternary that answers SellShort to an unrecognised side. The Go half folds before it sends, so this is safe as long as the Go half is the NEW one — which is exactly the deploy order below.

**The dangerous combination is exactly one: the NEW AddOn with the OLD Go binary.** The build floor cannot stop it (the old floor is permissive), and the old inverted guard would then hand correctly-formed orders to the market at already-through prices. **Deploy Go FIRST, recompile the AddOn SECOND** — Go-first is fail-closed at every intermediate moment. If a rollback must cross both halves, roll the AddOn back first, then the binary.

**Fastest kill switch, needing neither rollback:** `STOP_ENTRY_SEAM=off` + restart. The stop-entry branch short-circuits before any of this code runs, and the limit path is untouched.

---

## SURPRISES — recorded, not acted on (A23)

1. **The two bugs masked each other.** Not in the dispatch. C1 alone is a regression. This drove the whole sequencing and is why both are in one commit.
2. **The stop-entry path has never worked, ever.** 22 lifetime submissions across two days, `Stop price=0` on every one, 0 ever reaching `Working`. Wider than the dispatch's single-session framing.
3. **NT8 accepted a zero-trigger stop silently.** No reject, no error, no counter — and none on our side either. A silent-refusal path in both directions.
4. **The capability's own acceptance proof was the bug.** The 2026-08-31 E7 far-side proof order went out as `Limit price=28700 Stop price=0` and was recorded as a PASS, because the criterion was "it rests and cancels" and a zero-trigger stop rests perfectly.
5. **Up to NINE stop entries rested concurrently for a SINGLE arm slot** (`nt8_order_snapshots` 1604-1606 and 1664, `order_count=9`, 8 Accepted + 1 Initialized, none cancel-pending). Between 10:22 and 10:40 the re-arm loop placed 8 new orders with no cancels. Harmless only because all nine were inert. **With D1 fixed and this cancel path unchanged, one arm could take nine positions on an account capped at `maxFuturesContracts=2.0`** whose one-open-position rule is enforced Go-side at arm time and cannot reach orders already resting at the broker. The cancel path is outside this wave's footprint (A31) — **this needs an owner ruling before the seam runs again on a live session.**
6. **A row whose `kind` contradicts its `condition`.** Id 38 carries `kind='stop_entry'` with `condition='sweep_reclaim'`, but `ArmKindFor("sweep_reclaim")` returns `ArmKindLimit` — consistent with `UpsertArm` updating `condition` in place while leaving `kind` stale. `kind` and `condition` can disagree on a real row.
7. **`TestStopEntryKnobDefaults` (`trader/split_entry_test.go:267`) reads the AMBIENT environment** with no `t.Setenv`, asserting `STOP_ENTRY_SEAM` defaults off. It passes in a clean shell and would FAIL in one that has sourced `/home/hoang/nofx/.env`. All runs in this report were from a shell that has **not** sourced `.env`. Outside the footprint; not touched.
8. **Guide drift already on dev, unrelated to this wave.** `plays.ts` lists `STOP_ENTRY_SEAM (off)` in the *defaults* line while `.env` has it **on** and every boot line since 09-01 reads `stop_entry_seam=ON`. It is defensible as a default list, but it reads as the live value on the very path this wave fixes. **Left alone — it needs a ruling**, and rewriting a defaults list into resolved values is a different change.
9. **AUDIT-CHECKLIST class 75's contract test did not exist — and the first pass then shipped six stale refs, proving the point.** `grep -rln 'SYSTEM-MAP' --include=*.go .` returned nothing at the first pass. **[repair] It returns `trader/wave_b_replay_test.go` now**, for the delimited region this wave owns (E11). The rest of the map is still unenforced and the class-75 discipline elsewhere in this PR must not be reported as machine-verified.

11. **[repair] THE CANONICALIZER SURPRISE — class 77, and it is the largest thing in this document.** A class-28 fix applied correctly at the write on 2026-09-03 left two ordinal consumers downstream, one of which would submit a LONG entry to NinjaTrader as a live SELL. It has never fired only because every row written since is SHORT. Neither the wave's build pass nor its own census saw it; two adversarial lenses did, by asking what `r.Side` **contains at runtime** rather than what the struct comment says. Recorded as checklist class 77.

12. **[repair] A `git checkout --` cost me the SYSTEM-MAP corrections mid-mutation** (see E10's caveat). Uncommitted work in a worktree is one careless revert from gone, with no reflog trace — the class-45 shape, from a different direction. Redone and re-verified; recorded because the standing lesson (commit before you mutate) earned itself again.
10. **100 broker rejections on the SIM account** reading `Your maximum position limit has been met… Scope: all Rule #<id>`, across eight dates, plus 4 `User-initiated trading lockout is currently active`. Entirely outside this footprint.

---

## SCOPE (A31)

`git diff --name-only origin/dev...HEAD` — **15 files** after the repair (the report, plus the new caller-level test), every one inside the footprint:

```
docs/superpowers/AUDIT-CHECKLIST.md          docs/superpowers/SYSTEM-MAP.md
docs/superpowers/reports/2026-09-05-wave-b-stop-entry.md
ninjascript/VLTraderTCPClient.cs             provider/ninjatrader/order_snapshot.go
provider/ninjatrader/tcp_framing.go          trader/armed_executor.go
trader/ninjatrader/stop_entry_wire_test.go   trader/ninjatrader/tcp_trader.go
trader/wave_b_addon_slot_test.go             trader/wave_b_placement_test.go
trader/wave_b_replay_test.go                 trader/wave_b_stop_guard_test.go
web/src/guide/content/guards.ts              web/src/guide/content/plays.ts
```

**Nothing was reverted for scope**, because nothing was out of it: all four lenses agreed on the footprint, and I re-derived it. The two judgement calls the repair makes, both stated where they are made rather than here: **the LIMIT branch's side fold** (same row loop, same granted region, closes a live directional inversion during the window this wave creates — one token to revert if the owner would rather not) and **the AddOn's `OrdinalIgnoreCase` fold** (the `.cs` footprint is "the stop-entry order construction and VL_BUILD_ID"; the entry ACTION is three lines above the slots in the same `try` block, it is what makes the constructed order a Buy or a Sell, and it is the same class of positional/ternary defect as C1).

The forbidden-name grep over the ADDED CODE lines (`git diff origin/dev...HEAD -- ':!docs' ':!*.md' | grep '^+' | grep -E 'entry_gate|ArmKindFor|composeArmStop|arm_stop_anchor|reaperVerdict|detector_record|trade_excursions|close_sync'`) returns **one hit, and it is prose**: the word `entry_gate` inside a comment explaining that the refusal-counter keys share a namespace with the rr / entry_gate counters. No code touches any of them. No gate leg, no armable set, no stop composition, no R:R floor, no tick cap, no flip of the waterfall design. `trader/ninjatrader/tcp_trader.go` was edited for **one reason, quoted**: the D5 build gate at `:502`. Its `SignalPayload` construction is **untouched** — the owner's ruling that the wire already carries the TRIGGER correctly is confirmed end to end (`PlaceStopEntry` at `:489` takes `stopPx` as the trigger, rounds it, and sends `OrderType:"stop_entry", StopPrice: entry`; `tcp_framing.go:64-66` carries `order_type`/`limit_price`/`stop_price`). **The class-77 side fold was deliberately NOT put here**, even though `Side: side` at `:457`/`:529` is where the uppercase leaks onto the wire: the owner's ruling was "do not change the Go payload unless you can quote a reason", and folding at the executor's row loop is the class-28-correct place anyway — one canonicalizer, where the value enters the path. Both wire calls receive the folded value from a single source, and `TestStopEntryGuardHasAProductionCallSite` fails if either is ever handed `r.Side` raw again (M11).

---

## VERIFICATION RECORD — every finding from the four reviews, its disposition, and the evidence

Lenses: **S** = scope/footprint · **G** = guard logic & unknown-safety · **W** = the C# wire & far-side contract · **T** = test quality. Where two lenses raised the same defect the row names both. **I re-measured every one before acting; three are REFUTED and one is REFUTED-IN-PART, with the command and output that refutes it.**

| # | lens(es) | severity as filed | finding | disposition | evidence |
|---|---|---|---|---|---|
| 1 | G, T | BLOCKER | A LONG armed entry is submitted to NinjaTrader as a SHORT — the store uppercases `side`, the AddOn's `side == "long" ? Buy : SellShort` is ordinal | **ACCEPTED — FIXED** | Confirmed: `store/armed_orders.go:181` uppercases (commit `a05c6ca1`, `git merge-base --is-ancestor a05c6ca1 36648655` → ancestor of the DEPLOYED rev); `tcp_trader.go:457/:529` copy `Side: side` verbatim; `VLTraderTCPClient.cs:964` ordinal. Census: zero `LONG` rows post-canonicalizer. Fixed by folding once at `armed_executor.go:924` **and** `string.Equals(…, OrdinalIgnoreCase)` at `.cs:975`. RED: M3, M9, M11. Class 77 appended. |
| 2 | G, T | BLOCKER | `armed_executor.go:936` picks the stop trigger with `r.Side == "long"` — every LONG stop entry gets the SHORT trigger, and the new guard then cancels it | **ACCEPTED — FIXED** | Same root cause; same one-line canonicalizer. Verified the mis-signed trigger by driving `decideStopEntry("LONG", 29610.00, 0.50, 0.25, …)`: unfolded → trigger `0` + verdict `unknown`; folded → `29610.50`. RED: M3. Pinned by `TestDecideStopEntryOverStoredCasing` (13 rows, both casings). |
| 3 | G, W | BLOCKER (G) / owner-ruling (W) | Nine simultaneous working stop orders at the broker for ONE arm slot; D1 converts them from inert to live | **CONFIRMED — NOT FIXED, OUT OF FOOTPRINT, ESCALATED** | Re-derived independently: `nt8_order_snapshots` id 1664 holds **9** entries, `order_count=working_count=9`, names matching `armed_orders` signal_ids of ids 85/87/89/91/93/95/97/99/101 — all nine marked **cancelled** in the ledger while the broker still showed them Accepted. `MAX(working_count)` over 5,963 rows = **9**. The cancel-at-the-broker path (`nt.CancelOrder` returning success on a SEND, not on a confirmation) is outside the footprint (A31: this wave fixes how a stop entry reaches the broker and which guard judges it). **In `blockers[]`: this needs an owner ruling before `STOP_ENTRY_SEAM` sees a live session.** It is not a merge blocker; it is a cutover precondition. |
| 4 | G, W, T | WEAK-TEST (all three, independently mutated) | D3's "unknown never cancels" is pinned only at the pure-function level; nothing executes the placement decision, so cancel-on-ignorance passes green | **ACCEPTED — FIXED** | Confirmed: `grep -rn runArmedPlacement --include=*_test.go` finds only a source grep. Fixed by extracting `decideStopEntry` + `placeOneStopEntry` behind `stopEntryPlacer`/`armStateWriter` and adding `trader/wave_b_placement_test.go`. RED: M1, M2 — both the reviewers' own mutations, now red. |
| 5 | S, W, T | WRONG | The D4 boot line latches on an unknown build and never re-emits, making F2 proof #1 unobtainable | **ACCEPTED — FIXED** | Confirmed by reading `:1191` (`LoadOrStore` on a bare struct) against `f12_leg4.go:229-238` (`""` until a frame lands) and `armedTrader()` (non-nil with no connection). Now dedupes on the RENDERED LINE. RED: M7. |
| 6 | S, G, W, T | MINOR/WRONG (all four) | `armKey` is 0-based while every arm-gate key in the same map is 1-based — collides on a split arm | **ACCEPTED — FIXED** | Confirmed: `:942` used `Itoa(r.LegIndex)` vs `:455/:498/:525/:579` `Itoa(li+1)`. Now `Itoa(r.LegIndex+1) + ":place"` — 1-based **and** namespaced, so the keyspaces cannot alias at all. RED: M5. Latent, not live: no `stop_entry` row has ever carried `leg_index > 0`. |
| 7 | G | WRONG | The guard judges an unrounded trigger the wire rounds — judge ≠ send | **ACCEPTED — FIXED** | Confirmed with the live non-tick-aligned `entry_px` 29591.02: a BUY stop judged at 29591.52 reaches the broker at 29591.50, and with the market at 29591.50 the unrounded judgement says REST while the order fires on arrival. `decideStopEntry` rounds once, before judging. RED: M4. Plus `TestTickSourcesAgree`, because judge==send holds only while the two tick functions agree. |
| 8 | G | MINOR | The guard's `trigger <= 0` check sits behind the caller's offset arithmetic — entry 0 manufactures a positive trigger | **ACCEPTED — FIXED** | Confirmed: with the casing fixed, `entry=0, LONG` yields `trigger = +offset` and the guard adjudicates it as THROUGH → cancel on data that could not be adjudicated. `decideStopEntry` now refuses `entryPx <= 0` **before** the arithmetic. Pinned as the `no authored entry` row (decision) and sub-test (dispatch). |
| 9 | G | MINOR | Anything not enumerated in the verdict switch falls through to PLACEMENT | **ACCEPTED — FIXED** | Confirmed by reading the switch. Placement is now reachable only by naming `stopEntryPlace`; `default:` no-ops. RED: M2. |
| 10 | S, G | MINOR | `telemetry.IncGateBlock(at.id, "stop_entry_"+…)` renders `stop_entry_stop_entry_guard_unknown` | **ACCEPTED — FIXED** | Confirmed by reading `:1137` — the class already carries the prefix. Now one token: `stop_entry_guard_unknown`. |
| 11 | S | MINOR | `stopGuardVerdict.String()` has zero production call sites (A29) | **ACCEPTED — FIXED (wired, not deleted)** | Confirmed: only test callers. Now printed as `verdict=%s` on all four placement log lines, which also satisfies A9's "which guard evaluated it" more literally. |
| 12 | S | MINOR | The A9 submission-log rewrite deleted the only price a MARKET entry logged | **ACCEPTED — FIXED** | Confirmed: for a market order `limitArg == stopArg == 0` by construction and `entry` was printed nowhere. Restored for that case only, spelled `entry~=` so it cannot read as a `CreateOrder` argument. |
| 13 | W, T | WEAK-TEST | Nothing pins the `%w` wrap of `ErrAddonBuildTooOld`; `%v` kills the counting path silently | **ACCEPTED — FIXED** | Confirmed by reading E6 (three substring assertions, no `errors.Is`). Added the sentinel assertion. RED: M6 — and the red text shows the message survives `%v` intact, which is exactly why substrings could not catch it. |
| 14 | S, W, T | DOC-DRIFT | SYSTEM-MAP ships six wrong line refs, including the one it claims to correct | **ACCEPTED — FIXED, AND WIDENED** | All six confirmed by `grep -n`. Found more: Wave B had shifted the whole gate-leg block by 219 lines (`armGateVerdictFor` `:1316` → `:1701` + 7 sub-refs), and the knob line quoted a cancel string the reaper wave **deleted**. Added `TestSystemMapStopEntryRefsResolve` (E11), which found a bad ref on its first run. RED: M13, M14. |
| 15 | T | DOC-DRIFT | The Guide lists the three stop-entry gates in an order production does not use | **ACCEPTED — FIXED** | Confirmed: production is seam → retest window → stop-side guard → `PlaceStopEntry` (build floor inside it, LAST). The Guide said floor second. Corrected, **and** the consequence spelled out: in the go-first window an already-through arm is cancelled, not held. |
| 16 | T | WEAK-TEST | `stopGuardCases` has no length assertion; E3's RED is a mutation-red presented as a pre-fix red | **ACCEPTED — FIXED (both halves)** | Table length pinned (RED: M12) and the E3 section now labels its red as a mutation-red exactly as E5's was. |
| 17 | S, W | DOC-DRIFT / open item | `tcp_server.go:80`'s comment now misstates the live gate (`FarSideBuildE7` gates nothing) | **CONFIRMED — DELIBERATELY NOT FIXED (footprint)** | Confirmed: `grep -rn FarSideBuildE7 --include=*.go` → definition, that comment, and two test files. `provider/ninjatrader/tcp_server.go` is **not** in this wave's footprint (which grants `tcp_framing.go` and `order_snapshot.go` "the far-side build constants ONLY"). Recorded in A15 item 6 as an owned open item for the next lane. |
| 18 | G | MINOR | The destructive THROUGH branch fires on a price with no freshness check | **REFUTED AS IN-SCOPE — recorded** | The reviewer states it themselves: "Same shape as the pre-existing limit branch, so not new." A staleness gate needs a bar timestamp threaded into the branch and a knob to govern it, which is a change to WHEN arms may be placed — A31 scope, not this wave. Recorded as an open item; nothing built. |
| 19 | W | MINOR | GUIDE CONTENT LAW asks for the `GUIDE_BUILT_REV` bump in the same PR | **REFUTED — the current state is correct** | `GUIDE_BUILT_REV` must equal the **deployed** rev. This wave is not deployed; setting it to a sha that never ships makes the banner lie in the other direction. Boot-5 rule: bump it and rebuild `web/dist` **in the main tree, before the boot**, not in the marker. The Guide's *content* is updated here, which is the substantive half. Carried on the cutover checklist (A15). |
| 20 | W, T | MINOR | The A29 pin is a literal source-substring match — brittle and semantically blind | **ACCEPTED — the CLAIM is withdrawn, the pin is retained as a tripwire** | The guarantee now rests on E10 (executed). The report's first pass over-claimed that the D4 boot line's `unknown=` field "renders CANCELS if the verdict ever stops being no-op" — **that was false**, `StopEntryBootLine` probes the pure function and cannot observe the caller. Withdrawn in the D4 and E8 sections. |
| 21 | S | SURPRISE | A second process was running `go test` inside this worktree at 22:01:34 | **REFUTED as still-live — recorded** | `ps aux | grep -E 'nofx-waveb\|go test'` at 22:2x CT: no such process. The worktree list shows `nofx-waveb` held only by `fix/wave-b-stop-entry`. Most likely a reviewer's own read-only run overlapping the review window. No writes reached the branch: `git status --porcelain` was empty at every checkpoint and `git ls-remote` matched HEAD throughout. |
| 22 | S | (assessment) | "NO SCOPE VIOLATION FOUND — and no deploy happened" | **CONFIRMED INDEPENDENTLY** | Re-verified by me, not taken on trust: main tree HEAD `b2d3826e`, porcelain **clean**; `deploy/RELEASE` = `36648655`; `nofx-bin` still Sep 4 13:25 with PID 1137991 from that boot; Documents AddOns `.cs` still Sep 3 21:11 / `f12` / md5 `b7d670…`; `~/nofx-main.lock.d` **does not exist**; no binary in the worktree. Nothing was copied, killed, swapped or restarted (A3). |

**Counts: 22 findings adjudicated — 16 ACCEPTED-FIXED, 2 CONFIRMED-NOT-FIXED (footprint, both recorded and one escalated), 3 REFUTED, 1 CONFIRMED-INDEPENDENTLY.**

---

## OPEN ITEMS — recorded, not built (the next lane's, or the owner's)

1. **Broker-side stacking of stop-entry orders (finding 3).** Nine working orders for one arm slot, with a `CancelOrder` that reports success on a send rather than a confirmation. **Owner ruling needed before `STOP_ENTRY_SEAM` runs on a live session.** With D1 fixed and this path unchanged, one arm could take up to nine positions on an account capped at `maxFuturesContracts=2.0`, whose one-open-position rule is enforced Go-side at arm time and cannot reach orders already resting at the broker.
2. **`FarSideBuildE7` + `tcp_server.go:80` (finding 17)** — delete the constant and the comment together.
3. **Price freshness on the destructive branch (finding 18)** — both the stop and the limit guard treat `price > 0` as "price is knowable".
4. **C5's real finding, unchanged from the first pass:** the R:R gate computes `rr` from the AUTHORED entry and the trigger is never re-gated. All 21 arms passed the 2.0 floor at their entry (2.0058–2.0195) and were already below it at the trigger the wire carried (1.9775–1.9909). Reported, not fixed (A31, and no tick cap by separate owner ruling).
5. **`TestStopEntryKnobDefaults` reads the ambient environment** (`trader/split_entry_test.go:267`, no `t.Setenv`) and would fail in a shell that has sourced `.env`. Every run in this report was from a shell that has not.
6. **Guide drift already on dev**: `plays.ts` lists `STOP_ENTRY_SEAM (off)` in the defaults line while `.env` has it **on**. Needs a ruling on whether that list states defaults or resolved values.
