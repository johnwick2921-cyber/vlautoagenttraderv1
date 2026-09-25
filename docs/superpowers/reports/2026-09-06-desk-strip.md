# THE DESK STRIP — one endpoint, one row per fact, every number dated

**Branch:** `fix/desk-strip` · **session:** `desk-strip-554049f5` ·
**base:** `e28b604d` (dev tip at accept) · **date:** 2026-09-06

**Running rev, measured not quoted:** `/api/health` → `ea3f33fe9ccc`;
`go version -m /proc/2118306/exe` → `vcs.revision=ea3f33fe9cccfb2555181ee6e71493e9f92c1da2`,
`vcs.modified=false`, built 2026-09-06T11:48:37Z.

**Spec freshness (A1):** vet-04-monitoring and vet-05-execution last written at
`a45cf551`; SYSTEM-MAP and AUDIT-CHECKLIST at `ea3f33fe` — all ancestors of my
base. Nothing moved under me.

**A31:** this wave reads and renders. It writes nothing to the trading store,
changes no gate, no order, no knob, no plan and no exit.

---

## 1 · THE PREMISES

### C1 — the facts exist and nothing renders them · **CONFIRMED** [A]

The plan card polls `/api/plan/today`, `/api/plan/versions` and
`/api/plan/alerts`. None carries open risk, distance to stop, the day against
its limit, resting arms, the broker book, feed age, planner state or the day's
range. Every one of those is in the store or on the F12 frame today.

### C2 — the green header answers a narrower question · **CONFIRMED** [A]

`SYSTEM_STATUS::ONLINE` rendered green whenever `/api/health` returned
`status:"ok"` (`TraderDashboardPage.tsx:651-656`). That endpoint reports that the
HTTP process answered — nothing about the feed, the link, or any control.

**The part worth keeping:** an earlier wave already fixed this once, from a
*static* string to a real poll (`:158`, "the debug strip's SYSTEM_STATUS was a
static lie"). The poll was honest and the WORD was still wrong. That is why it
felt fixed, and why it was not.

### C3 — a ledger price is not the broker's · **CONFIRMED** [A]

Arm 35: ledger `29351.6284728996`, accepted `29355` — 3.371527 pts. PROTECTION
renders the accepted stop or UNKNOWN; it has no path to the ledger's number.

### C4 — **CORRECTED. MAE/MFE can now render, and it is proven live.** [A]

The drawer was already written correctly and data-driven: "no excursion rows
yet" *only* when there are none, otherwise medians **with n** and its source
named. Wave A's backfill (587 `trade_excursions` rows, 68 with `mae_pts`) made it
start working. Live, from `GET /api/expectancy`:

| condition | median MAE | median MFE | n |
|---|---|---|---|
| reject | 21.5 | 39.5 | **31** |
| breakout_retest | 42.25 | 25.75 | 9 |
| acceptance | 49.5 | 25.0 | 6 |
| sweep_reclaim | 35.0 | 20.5 | 6 |
| reclaim | 44.0 | 16.25 | 5 |
| hold | 22.5 | 92.0 | 1 |

**D6(c) required no code change.** The honest fix was already in place; what was
missing was data.

### C5 — the alert backlog is an attention problem · **CONFIRMED to the decimal** [A]

`day_plan_alerts`: **P0 152 total, 62 acked = 40.8 %**; P1 500/160 = 32.0 %; P2
3/3 = 100 %. The strip therefore has no unread count and nothing to acknowledge.

### C6 — **WRONG.** [A]

`SessionPlanCard.tsx:431` is a **code comment** (`{/* W7 … the WEEKLY chip
(advisory view). */}`), not a rendered label. `WeeklyChip` renders `WEEKLY refs —
PWH x · PWL y` or `WEEKLY none`, and its tooltip reads "no directional call
(class 50)". The chip was fixed by the refs-only wave on 2026-09-02. The only
residue was the comment — which is precisely what sent this dispatch hunting a
bug that no longer existed. Corrected.

---

## 2 · THE TWELVE LINES AND THEIR SOURCES

| # | line | source |
|---|---|---|
| 1 | MODE | `planModeFor` · `SessionRegistry.ActiveSession` · bars · `FeedStatus` |
| 2 | POSITION | `at.GetPositions()` · `market.FuturesPointValue` |
| 3 | PROTECTION | `accepted_risk.accepted_stop_px` (Wave A) — **never** `armed_orders.stop_px` |
| 4 | DRIFT | `accepted_risk` vs `armed_orders`, rendered only when they differ |
| 5 | TARGET | `accepted_risk.accepted_target_px` |
| 6 | DAY | `trader_positions.pnl_corrected` (A22) · resolved guardrail |
| 7 | ARMS | `armed_orders.ListNonTerminal` (incl. `cancel_pending`) |
| 8 | BOOK | `CutoverGateStatus()` leg 4 — **reused, not re-derived** |
| 9 | FEED | bars · `FeedStatus` · AddOn `build_id` from the F12 frame |
| 10 | PLANNER | `AnyPlannerReadInFlight()` |
| 11 | RANGE | bars (session-day) · `plannerATR5m` |
| 12 | LAST FILL | `trader_fills` — slippage renders **UNKNOWN**, see §5 |

---

## 3 · RED → GREEN

**E1's RED is the base commit itself:** at `e28b604d` there is no
`api/handler_desk.go` and no `trader/desk_facts.go`, so none of these assertions
could compile. `git show e28b604d:trader/desk_facts.go` → *path does not exist*.

| pin | result |
|---|---|
| E2 accepted-vs-ledger | `PROTECTION: accepted stop 29355.00 · 55.00 pts (220 ticks) away` / `DRIFT: ledger 29351.628473 vs accepted 29355.00 → +3.371527 pts`; `29351` never appears in PROTECTION |
| E2b no record | UNKNOWN with `no accepted-risk record for this signal` — **never** the ledger's stop |
| E4 flat | PROTECTION/TARGET/DRIFT/POSITION all read FLAT, not UNKNOWN and not 0 |
| E1 dated | 12 lines; each `ok`/`flat` has an as-of, each `unknown`/`stale` has a reason |
| E6 cadence | flat → 15000 ms; a resting arm → 5000 ms |
| E8 broken store | `with the store closed: 5 unknown of 12`, no panic, 12 rows still render |
| FE ×6 | dated · UNKNOWN-with-reason · accepted-not-ledger · amber-stale · unreachable-stated · header counts |

### THREE MORE, FOUND MID-CUTOVER — the lock was held and the merge was done

The verification fan-out returned while the clean-clone build was running. It
found three defects in **this wave's own code**, and one of them was the kind
that must never ship:

1. **The `SIM ·` label was a LITERAL** (`desk_facts.go:252`). On a trading
   dashboard, the word separating simulated money from real money is the last
   thing that may be asserted: it would have printed SIM on an account the AddOn
   never reported as a simulation. It is now READ from `AccountInfo.IsSim` on
   the accounts frame, renders `*** LIVE ACCOUNT ***` unsoftened if NT8 ever says
   so, and renders **UNKNOWN** — never "SIM", never "live" — when the frame has
   not arrived. **This is the exact class of defect the wave was commissioned to
   remove, committed by the wave itself.**
2. **`kernel.DefaultSessionRegistry()` bypassed the admin registry** — the
   shipped fallback instead of `system_config`, which is the dead wire W8 exists
   to close. It agreed with the stored registry today, which is how a bypass
   survives review. Now `at.sessionRegistry(now)`.
3. **`ActiveSession` names the WINDOW, not the market.** It ignores `Enabled`
   and the weekday, so it answered "NY" at 14:22 on a Sunday with CME shut. Row 1
   now states the market separately via `CMEClosedReason` — `CME CLOSED
   (weekend)` beside `session=NY`.

And a fourth the fan-out proved about the data rather than the code: **567 of
587 `trader_positions` rows carry the literal five-character string `"<nil>"`**
in `entry_order_id` — a formatted nil pointer persisted as text. It passes
`IS NULL`, it passes `= ''`, and it joins to nothing, so a naive read looks like
it worked. `deskJoinKey` now treats it as absent, which sends PROTECTION to
UNKNOWN instead of to a false join.

**The cutover was stopped for these** (A23) after the merge and before the
binary swap, and the clean-clone build was redone at the corrected head.

### Three defects the pins found in my own code

1. **`deskPosition`/`deskBook` ran outside the containment loop**, so a nil
   broker link panicked past every `deskSafe` below them. A10 says a read
   surface may never reach the trading loop; it could have.
2. **Cadence read 5000 ms with nothing live** — it matched the substring
   `"resting"`, and `"none resting"` contains `"resting"`. It now decides on the
   row's STATE.
3. **Four of six FE tests passed against test one's payload** — a shared SWR
   key. Each case now has its own.

A fourth was mine and not the code's: an assertion forbade *any* em-dash, when
the rule is "no dash standing in for a value".

---

## 4 · THE FOUR TRUTH FIXES

- **(a)** No code change: the chip was already right (C6). Comment corrected.
- **(b)** `SYSTEM_STATUS::ONLINE` → **`PROCESS::RESPONDING`**.
- **(c)** No code change: already correct, and now fed (C4).
- **(d)** `/api/risk/status` gains `not_enforced[]`. The pre-prompt gate calls
  `CheckPreTrade(0, positions, 0, 0)` (`engine_analysis.go:118-136`) — zero for
  both notional args and zero for pnl — so it enforces only the concurrent cap.
  `max_notional_usd` is the futures-unaware **crypto** cap that one MNQ contract
  (~$61k) exceeds; `kill_switch_armed` is `limit > 0`, not an armed state;
  `daily_loss_limit_usd` is the env value a Studio guardrail supersedes. The
  fields stay — readers exist — and now say what they are.

---

## 5 · A15 — WHAT THE OWNER WILL STILL SEE WRONG

- **PROTECTION, DRIFT and TARGET will read UNKNOWN until the first order
  acceptance writes an `accepted_risk` row. That is the strip being honest, not
  a gap** (owner's words, and the correct reading). `accepted_risk` has **0 rows**
  today because its only writer shipped 2026-09-05 21:59 CT and the last arm was
  created 2026-09-04 12:11 CT — the path has never had an opportunity to fire.
- **AND THEY MAY STAY UNKNOWN EVEN AFTER THE NEXT ARM.** `applyBrokerTerms` keys
  on order names ending `-sl`/`-tp` and on `o.StopPrice`, and across all stored
  `nt8_order_snapshots` there are **zero** orders with either: every observed
  order carries a bare signal-id name and puts its price in `limit_price`,
  including `type:"stop"` orders. So `accepted_stop_px` will likely be written
  NULL. The strip will say UNKNOWN with its reason and will not invent a number —
  but the underlying recorder needs its own wave, and this report is where that
  starts.
- **LAST FILL's slippage is UNKNOWN and will stay UNKNOWN.** The intended price
  is not stored beside the fill, and the AddOn's `slippage_ticks` has no
  production consumer. The row says so rather than computing a number from the
  wrong inputs.
- **FEED will read stale outside RTH.** At the time of writing the newest bar is
  ~46 h old (CME closed). Amber is the correct render; it is not a fault.
- **The DAY row shows the resolved ENFORCED limit** — which may differ from the
  number `/api/risk/status` publishes. That difference is the point of D6(d), and
  a reader comparing the two surfaces will see it.
- **The strip cannot see what no table records.** Wakes-suppressed-today (D2
  line 10's second half) has no counter, so it is not rendered rather than
  guessed.
- **`SYSTEM_STATUS` still appears in the guide's own older table** further down
  `status.ts`; the header itself is renamed and the new text explains it, but the
  legacy rows below were left as another lane's words.

---

## 6 · PROOF — THE FIRST LIVE STRIP

Boot **2026-09-06 15:12:38 CT**, PID 2364404, rev `2a66bf5d`. Dump taken
15:13:01 CT — **12 lines, 1 UNKNOWN, 1 stale, cadence 15000 ms** (idle, resolved
from what is live):

```
 1 MODE       [ok     ] SIM · plan_mode=strict · session=none · CME CLOSED (weekend) ·
                        15:13:01 CT · process responding · feed 47h14m2s · link  · book …
 2 POSITION   [flat   ] FLAT — no open position
 3 PROTECTION [flat   ] FLAT — nothing to protect
 4 DRIFT      [flat   ] FLAT
 5 TARGET     [flat   ] FLAT
 6 DAY        [ok     ] realized +0.00 USD on 0 trade(s) · no enforced daily limit
                        (guardrails master OFF — soft-audit only)
 7 ARMS       [flat   ] none resting
 8 BOOK       [ok     ] 0 working order(s) at the broker (ledger agrees: 0)
                        src: broker — NT8 order_snapshot frame (age 13s, build 2026-09-03-f12)
 9 FEED       [stale  ] last bar 47h14m2s ago · link  · AddOn build 2026-09-03-f12
                        reason: the newest 1m bar is older than 2 minutes
10 PLANNER    [ok     ] idle — no planner read claimed
11 RANGE      [unknown] UNKNOWN
                        reason: no bars inside the current CME session-day
12 LAST FILL  [ok     ] BUY MNQ 29355.00 qty 1.00 · slippage UNKNOWN (…)
```

**What the dump proves.** MODE reads `SIM` **from the accounts frame**, not from
a literal — the defect caught mid-cutover. It states `CME CLOSED (weekend)`
separately from `session=none`, so a closed market cannot read as a live one.
BOOK is cutover leg 4 rendered continuously. RANGE is UNKNOWN **with its
reason**, not a zero. FEED is amber at 47h — correct, CME has been shut since
Friday.

**PROTECTION/DRIFT/TARGET read FLAT, not UNKNOWN**, because there is no position
to protect. They become UNKNOWN-with-reason the moment a position exists without
an `accepted_risk` row — which is the case the owner flagged and which the E2b
pin covers.

**The second dump — with a position or an arm — has NOT been taken.** It needs a
session; CME opens 17:00 CT. Until then this wave is proven for the flat case
only, and a green suite proves neither.

### A defect this dump itself exposed

Row 9 renders `link ` — an **empty string** where UNKNOWN belongs. `FeedStatus()`
returned `""` and the strip printed it verbatim. It is not misleading about
safety (the row is already amber and states its reason), but a blank where a
value should be is precisely the class this strip exists to remove, and it is
mine. Fix: render `link UNKNOWN (no status reported)` when `FeedStatus()` is
empty. Filed here rather than hot-fixed, so the fix arrives with its own pin
rather than as an unpinned edit to a just-booted binary.

---

## 7 · ROLLBACK

```
git -C ~/nofx checkout dev && git -C ~/nofx reset --hard <prior-dev-sha>
mv ~/nofx/nofx-bin.old.ea3f33fe ~/nofx/nofx-bin      # named for the rev it HOLDS
echo ea3f33fe > ~/nofx/deploy/RELEASE
kill -9 $(pgrep -f nofx-bin)
```

No data migration to reverse: this wave writes nothing. `web/dist` must be
rebuilt on rollback, because Go serves it from disk and the strip's assets would
otherwise outlive the binary that answers them.
