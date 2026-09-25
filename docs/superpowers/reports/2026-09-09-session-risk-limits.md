# SESSION RISK LIMITS — dispatch 104 (W4)

Owner: hoang · agent `sessionrisk-554049f5/nofx-80[aa87f6]` · branch
`fix/session-risk-limits`, claimed at `9b578c06` off dev's tip `05125bd6`.
Worktree `/home/hoang/nofx-srl`; main tree untouched (A2b).

**Running rev, my own read:** `954f11b15f2e7615678f7d2b708c47895faebf1e`,
`vcs.modified=false`, pid 438, up since 2026-09-09 13:02:45 CT; `/api/health`
agrees. Matches the dispatch.

**Basis, pinned:** `docs/superpowers/research/2026-09-08-range-fade/report.md`
@ `f5927cdc` (requirement 6 and the "do not"); `2026-09-05-vet-06-risk.md`;
the two-day audit @ `f890ea60` D38.

---

# THE HEADLINE, IN PLAIN WORDS

**The $450 daily limit is decorative twice over.** Two switches are off, and
BOTH must be on before it enforces anything:

| knob (`ai_config.risk_control`, strategy `a5b7662e`) | value |
|---|---|
| `guardrails_enabled` — the master | **false** |
| `daily_loss_enabled` — this leg's own toggle | **false** |
| `daily_loss_limit_usd` | 450 `[O]` |
| `consecutive_loss_halt` | **absent** (0 = off) |

Confirmed on the live surface, firing every cycle at
`kernel/engine_analysis.go:173`:

```
⚠️ Strategy Studio: risk guardrails master OFF — daily loss/profit/trade limits
   + blackout NOT enforced this cycle
```

The gate itself is `kernel/risk_limits.go:309` —
`if g.DailyLossEnabled && g.DailyLossLimitUSD > 0 && ...` — so the leg's own
toggle gates it independently of the master. **Until both move, nothing in this
wave makes the daily limit bite.** The desk strip's DAY line and the boot line
now say exactly that, in those words.

**What DOES bite today:** the consecutive-loss breaker. Its own comment says it
is *"a per-strategy circuit breaker, NOT gated by the guardrails master switch"*,
and this wave wires it to the path that actually trades.

---

# SECTION C — THE EVIDENCE

## C1 — the daily limit's four holes

| hole (vet-06) | status at `954f11b1` | file:line |
|---|---|---|
| (a) up-to-4-minute arm window before the check | **reported, not changed** (leg-D logic, out of scope) | `trader/entry_gate.go:171` |
| (b) realized-only input | **reported, not changed** (leg-D logic) | `kernel/risk_limits.go:309` |
| (c) clears only at the first AI cycle after the roll | **WORSE THAN REPORTED — FIXED** (below) | `kernel/risk_limits.go` `MaybeResetDaily` |
| (d) no `daily_force_flat` case in `armRefusalClass` | **CONFIRMED — FIXED** | `trader/armed_executor.go` |

**(c) is not what vet-06 said, and the truth is worse.** The trip did not clear
on the CME roll, and it did not clear at the next AI cycle either: **it never
cleared automatically at all.** `MaybeResetDaily` rolled `lastDailyResetDate` and
left `forceFlatReason` untouched; `clearAllDailyForceFlat` had exactly ONE caller,
`ResetDailyPnLAt`, reachable only from the operator's `POST /api/risk/force-flat`.
A tripped desk stayed blocked across the roll, the next session and the one after,
until a human reset it or the process restarted.

The code has promised otherwise since #91, in the comment above the state:
*"Cleared by the CME session-day reset (ResetDailyPnLAt), so a trip lasts the
session-day and lifts with the daily window."* Nothing wired it. Unseen because
the master is OFF and the trip has never fired — and the latch is an in-memory
map a restart clears, so even a live occurrence would likely have been erased
before anyone correlated it. **Checklist class 96.**

## C2 — the breaker exists, is wired, and guards the wrong path

`consecutive_loss_halt` is registered `KnobLive`
(`store/knob_registry_table.go:30`) with reader `consecutiveLossHalted`
(`trader/auto_trader_orders.go:112`) and a production call site at `:250`, inside
`executeDecisionWithRecord`. Resolved value: **0 (OFF)** — the knob is absent
from the live config.

**But the wiring is on the DECISION path**, and `plan_mode=strict` — the live
setting — routes every entry through the ARM path: *"plan_mode=strict executes
plan scenarios on the ARM path only, and this is a %s-path market entry"*
(`trader/entry_gate.go:186`). The breaker guarded a door nobody walks through.

## C3 — REFUTED AS STATED, and the real holes are narrower and worse

The dispatch asked whether the 14:45 flatten cancels resting arms. **It already
did** — the S-LIST CLOSER (2026-08-27) cancels working arms synchronously,
ack-waited, before flattening, and `TestSListEODFlatCancelsArmsBeforeFlatten`
pins it. E1 as written ("MUST FAIL today, arms survive") would not have failed
for that reason.

Two real holes, both leaving a live order at the broker while the ledger reads
flat:

**(a) The early return.** `enforceEODFlatAt` read the open positions and
returned on `len(positions) == 0` **before** reaching the cancel. A session that
ended flat BY LUCK — nothing filled — left every resting arm alive past the
close, into the next session's tape. Nothing said so, because the position was
zero so the book "looked" flat. This is precisely the research's do-not:
*"Canceling remaining entries is part of being flat."*

**(b) `place_pending` settled without a send.** `cancelArmedOrdersSyncWith` read

```go
if r.State != "working" || r.SignalID == "" || cancelFn == nil || src == nil {
    _ = ledger.SetState(r.ID, "cancelled", reason)
    continue
}
```

`BeginPlacement` (`store/armed_orders.go`) sets `signal_id` **and**
`state = place_pending` in ONE update, **before** the order reaches the broker —
so a `place_pending` row ALWAYS carries a signal id and its order may already be
resting. This wrote `cancelled` **with no wire cancel at all**. Class 81 reached
by omitting the send.

**Count of arms non-terminal after a 14:45 flatten in the retained sample:** the
dispatch asked for this and it is not answerable from the record — `armed_orders`
is overwritten in place (no history table), so a row cancelled later is
indistinguishable from one cancelled at the close. Stated as UNKNOWN rather than
estimated. The holes are established from the code, not from a count.

## C4 — runs of losses on this tape

n = **73** usable. EXCLUDED and counted: **516** pre-era (< 2026-08-15 CT),
**3** `pnl_corrected` NULL, **3** `e7_farside_test`.

| measure | value |
|---|---|
| longest losing streak | **7** — ids **585, 586, 587, 588, 589, 590, 591** |
| session-days with a 3+ run | **7 of 15** (08-20 ×4, 08-21 ×3, 08-24 ×4, 08-25 ×3, 08-26 ×3, 09-02 ×4, 09-08 ×3) |
| losers | 47/73 = **64.4%** |
| worst single close | **−155.00** |
| 8 in a row ever reached | **NO** · 5 in a row: YES |

**So N = 8 `[I]` never fires on this tape** — it sits one above the observed
maximum, the same shape as the $450 that trips 0 of 12 days. That sentence is on
the boot line, not buried here. M = 5 `[I]` does fire.

## C5 — post-loss behaviour, and n is the finding

Stop-outs in the era: **4**. Minutes to the next arm: p50 **81.4**, p80 141.7,
min 0.0, max 141.7. Re-armed within 30 min: **2 of 4**. **Same level as the
stop: 0.**

The reason n is 4: of 47 losers, **42 close as `sync`**, 4 `stop`, 1 `manual`.
A cool-down keyed on `close_reason='stop'` would observe four events and read as
"this never happens" — so D3 triggers on a **losing close** (P&L sign), which
needs no attribution to work.

---

# THE EXIT-CAUSE FINDING — AND THE CORRECTION TO MY OWN CLAIM

The owner asked me to file the 89%-unattributed figure as a finding against Wave
A's claim that exit cause is "recorded, not inferred". **That framing is wrong,
and the correction is the finding.**

`ExitCauseFromBroker` (`store/position.go:138-152`) landed in `455dce5a`
(2026-09-05 21:37:17 CT), which **is** an ancestor of the running rev. Splitting
the record at that moment:

| window | n | `sync` | attributed |
|---|---|---|---|
| era ≥ 2026-08-15 | 73 | 66 (**90%**) | stop 4 · manual 2 · target 1 |
| **since Wave A** | 8 | 1 (**12%**) | stop 4 · manual 2 · target 1 |
| since Wave A, **losers only** | 5 | **0** | stop 4 · manual 1 |

**Every attributed close in the entire record happened after Wave A landed.** The
90% is pre-Wave-A history, when the string was a hardcoded literal. Wave A works;
the claim stands. What the figure actually measures is that only 8 closes have
happened since it shipped.

**A real gap remains, unfired.** The AddOn emits `exitReason = "limit"` for an
`-lx` limit exit (`ninjascript/VLTraderTCPClient.cs:1327`).
`ExitCauseFromBroker` maps sl/tp/manual and **has no `limit` case**, so such a
close lands in the `default` arm as `sync` — an attributed exit recorded as
unattributed. Zero occurrences so far. **Reported, not fixed: A31 puts the
recording path outside this wave's footprint.**

**Two further findings from the adversarial pass, also outside the footprint:**
post-Wave-A causes never reach the expectancy engine (5 of 8 post-Wave-A closes
have no `trade_excursions` row; `target` and `manual` have never appeared there),
and `hitOf` in `expectancy/aggregate.go` fabricates `false` rather than absent —
a canon class 49/53 shape. Both belong to whoever owns the record.

---

# SECTION D — WHAT SHIPPED

**D1 — flat means the book.** The cancel runs unconditionally and FIRST; positions
are read after it, so a fill that won the race is still flattened; a failed
position read logs `flatness UNVERIFIED` and never claims flat. The cancel guard
keys on the **signal id** — the only evidence anything could be at the broker —
so `place_pending` gets a wire cancel. A row that never got a signal id may go
terminal without asking; a duplicate cancel is idempotent where a missed one is a
live order we stopped watching.

**D2 — the breaker on both paths.** One resolution (`breakerHaltN`) shared by the
decision and arm paths: the owner's knob when set, else `8` `[I]`;
`BREAKER_HALT_N=0` is the explicit off switch. `M = 5` `[I]` WARNs without
refusing. Clears at the CME roll (`CMESessionDayStart`). Adjudicated ONCE per
cycle before the scenario loop — both are session-level facts, and a per-leg
re-query could answer the same question differently within one cycle.

**UNKNOWN never lengthens a run.** `CountConsecutiveLossesSince` excluded
unresolved rows in its WHERE and called them "never counted either way".
Excluding a row from the scan makes it **transparent**, not neutral: 3 losses, an
unresolvable close, 3 more counted as **six**. Bridging makes a halt MORE likely,
and blocking is this counter's destructive branch — the direction A24 forbids. An
unknown close now ENDS the run. The `e7` seam stays excluded: a synthetic row is
not a trade, so it neither counts nor breaks.

**The no-trade band on the arm path** (added mid-wave). `armed_executor.go` held
**zero** references to `InLunchNoTrade`, `InFirstNoTradeMinutes` or
`sessionEntryBlocked`; the sole enforcement was `auto_trader_orders.go:281`, on
the path strict forbids. An arm inside the band is refused; an arm already
resting when the band opens is **cancelled, not grandfathered**, through the same
seam the close uses. `sessionEntryBlocked` gains the A28 `At(now)` seam so both
paths read one band with one clock.

**The non-obvious half of the band: cancel, not grandfather.** Refusing only NEW
arms while an arm placed at 11:58 rests into 12:00 is a band that stops
AUTHORING and not ENTERING — the same class of defect as the one this wave
exists to fix, one layer in. The gap was found independently by the W5 lane,
whose filing would have stopped at refusing new arms; the owner's addition was
the cancel. An arm whose fill would land inside the band is not an arm we are
willing to own, however long it has been sitting there.

**D3 — post-loss counter, never a gate.** Triggered on a losing close; labels the
arm card and increments a per-(trader, session-day, session) counter.
K = `30` min `[I]`.

**D4 — (a)/(b) reported unchanged; (c) fixed (class 96); (d) `armRefusalClass`
gains `daily_force_flat`, plus `consecutive_loss` and `no_trade_band`.**

**D5 — boot line, Guide, SYSTEM-MAP** in the same commits.

**The desk strip, class 82 caught before it bit.** `deskGuardrail` checked only
the master while the gate requires both toggles. The moment the owner turned the
master ON and left `daily_loss_enabled` off, the strip would have reported the
$450 limit **ENFORCED** while the gate ignored it. Absent stays true, matching
`engine_analysis.go`'s `boolOrDefault(..., true)`, so no desk that never set the
field is silently disarmed.

---

# SECTION E — RED, GREEN, MUTATION

**E1(a)** — flatten with no position, one working arm:
```
RED   wire=[]  ·  1 arm(s) still non-terminal after the close with no position open
GREEN after D1
MUT   restore `if len(ps)==0 { return false }` → --- FAIL   (restored → PASS)
```

**E1(b)** — flatten with a `place_pending` arm:
```
RED   wire=[close_long:MNQ cancel_stops:MNQ]   ← position flattened, no cancel:sig-pending
GREEN after D1
MUT   restore `r.State != "working"` → --- FAIL   (restored → PASS)
```

**E2** — breaker table (8 cases), threshold resolution, roll clearing, UNKNOWN:
```
RED   run = 6, want 3 — an UNRESOLVABLE close bridged two runs of 3 into one of 6
GREEN after the position_query.go change
MUT   halt at `> n` instead of `>= n`  → FAIL at_the_halt
MUT   band no longer precedes the record → FAIL inside_the_no-trade_band, band_wins_over_a_halt
MUT   remove `at.sessionRiskGateAt(now)` from the arm path → FAIL TestArmPathConsultsTheSessionRiskGate
```

**E3** cool-down counted, never refused · **E4** `daily_force_flat` has its own
class · **E5** shortened-day pull-in + the flatten's cancel-before-read ordering
· **D4(c)** roll lifts the trip (RED: *"the trip SURVIVED the CME roll"*),
race-checked.

**A pin of mine that did not bite, and was fixed.** The DAY-line test called
`deskDailyLimitText` directly, so reverting `deskDay` to its old literal left it
GREEN. Built is not wired; it now asserts the call site and fails on that
mutation.

**A contract test updated, not deleted.** `TestCountConsecutiveLossesSince`
asserted that an unknown-P&L close at the tail was excluded so an earlier loss
still governed. Under the owner's UNKNOWN ruling it now ends the run. Updated in
place with the reasoning, plus a new assertion that a later loss starts a fresh
run — so the break is a break, not a permanent mute.

**Suite at the branch head:** Go **20 packages ok, 0 FAIL**.

---

# SECTION A15 — WHAT THE OWNER WILL STILL SEE WRONG

- **The daily limit still enforces nothing.** Two switches, both off. This wave
  makes the state legible and the refusal countable; it does not turn the limit
  on — that is the owner's knob (A31).
- **The breaker will not fire either, on this tape.** N=8 is one above the
  observed maximum. It is ON by default now, and honest about that on the boot
  line. M=5 will fire and WARN.
- **`place_pending` cancels are ack-waited, not book-confirmed.** D1 sends the
  wire cancel the old code omitted; the settlement pass (`confirmPendingCancels`)
  is what confirms against a fresh snapshot. The "flat" claim is therefore as
  strong as the ack plus the next settlement pass, not stronger.
- **The `-lx` → `limit` mapping gap** and the two expectancy findings are
  reported and unfixed, outside the footprint.
- **The count of arms surviving past a historical 14:45** is UNKNOWN and stated
  so — `armed_orders` has no history table.
- **75/76/77 remain duplicated** across the checklist's two numbering formats,
  from waves before this one.

---

# ROLLBACK

Single Go boot, no AddOn change. Preserve the running binary as
`nofx-bin.old.954f11b1` (verified with `go version -m`), restore
`deploy/RELEASE`, `mv` it back, owner runs `kill -9`; systemd relaunches.

Everything here is additive except three behaviour changes, each independently
revertable: the flatten's ordering (`auto_trader_clock.go`), the cancel guard's
key (`armed_executor.go`), and the loss-run's UNKNOWN semantics
(`store/position_query.go`). `BREAKER_HALT_N=0` disables the breaker at runtime
without a rebuild.


---

# THE ADVERSARIAL PASS FOUND A DEFECT I HAD JUST INTRODUCED

A read-only census over every flatten path landed after this report was first
written. Two of its findings were acted on; the rest are recorded below.

## FIXED — my own D1 fix was half a fix

`cancelArmedOrdersSyncWith` read:

```go
if r.SignalID == "" || cancelFn == nil || src == nil {
    _ = ledger.SetState(r.ID, "cancelled", reason); n++; continue
}
```

I replaced `r.State != "working"` with the signal-id test **and left the rest of
the disjunction standing.** So a row that HAS a signal id — by definition an
order at the broker — was still written `cancelled` whenever the cancel function
or the ack stream was missing. The same class-81 shape, surviving in the half of
the condition nobody re-read. Worse: the comment I wrote above it claimed *"every
other row gets a cancel on the wire"*, which the code did not do. A comment
asserting the fix, over code that only half performs it.

`one_contract.go` had already learned this exact lesson — *"an unreachable AddOn
sent the row down the terminal branch below — writing 'cancelled' on an order the
broker still holds, which is class 81 exactly"*. A missing wire records the
INTENT, never the outcome. The row now goes `cancel_pending` (non-terminal, so
the slot stays taken and the settlement pass reconciles it) and is counted
UNACKED. Pinned RED (*"row 1 (signal sig-nowire) was written 'cancelled' with NO
wire available"*), mutation-tested.

## FIXED — the same A24 hole in the sibling path

`enforceT1ForceFlatAt` — the red-news force-flat — read
`if err != nil || len(positions) == 0 { return … }`. A DB read failure two
minutes before FOMC silently became "no position to flatten" and the position
rode the print. The EOD path closed this on 2026-09-09; its sibling two functions
away did not get the fix. Now logs `flatness UNVERIFIED going into the red-news
window`. Pinned RED, mutation-tested.

## REPORTED, NOT FIXED

- **`cancelArmedOrders` (the no-link fallback)** writes `cancelled` on every
  non-terminal row with no wire contact of any kind, and its count feeds the
  operator-facing flat claim. Same shape as the above, different function; it is
  the branch taken when `armedTrader()` is nil.
- **The ack-timeout branch** promotes a row straight to `cancelled`, bypassing
  the `RequestCancel → cancel_pending → ConfirmCancel` lifecycle the 2026-09-06
  wave built. **`ConfirmCancel` has never fired in production: 0 of 69 cancelled
  rows carry a `cancel_settled_snapshot_id`.**
- **"Acked" means "the row left the non-terminal set", not "cancelled".** A FILL
  arriving during the drain window retires the row and is counted and logged as a
  successful cancel.
- **Every flatten decides "no open position" from the local store, never the
  broker** — the same store this codebase documents as lagging real exits by
  ~80s. The cutover gate deliberately splits store from broker for exactly this
  reason; the flatten has only the store leg.
- **The flatten is the only cancel site with no `cancelSafetyFor` guard.** That
  is the owner's ruling of 2026-09-07 ("a guard that refuses without a book,
  applied to the EOD flatten and the news-halt sweep, leaves arms live and
  re-opens class 33 — worse than the defect it guards") and it stands. The census
  notes the harm is currently bounded by the C# side, which no longer walks
  `placedBrackets` — a bound that lives in the AddOn, not in Go.

None of these are regressions from this wave; all predate it.

## OWED AT MERGE — three sentences this boot makes false

The W5 lane (`nofx-6b`) is booting a docs wave that writes the no-trade-band gap
into the Guide and SYSTEM-MAP in plain words, correctly, because at rev
`954f11b1` it is TRUE. **This wave falsifies all three**, and the GUIDE CONTENT
LAW puts that on the wave that changes the gate:

| file | the sentence that goes false |
|---|---|
| `web/src/guide/content/plays.ts` | *"refuses AI-decision entries only … by NOTHING on the arm path … with plan_mode=strict it refuses nothing"* |
| `web/src/guide/content/planCard.ts` | *"Nothing on the ARM path reads them … neither band can refuse an entry"* |
| `docs/superpowers/SYSTEM-MAP.md` | *"WHICH PATH EACH BLACKOUT BINDS (W5, measured at rev 954f11b1) … They do not bind the ARM path … grep = 0"* |

They are corrected **after** the merge that brings them in, and **rewritten, not
deleted** — the SYSTEM-MAP row is deliberately rev-stamped, and a reader who
loses it loses the fact that this was ever a gap. Sequenced this way because
none of the three text exists on this branch until W5 merges.

---

# OWNED BY THE NEXT WAVE (owner-ruled 2026-09-09) — NOT THIS ONE

## 1 · ConfirmCancel has NEVER fired in production

**`0` of `69`** cancelled rows carry a `cancel_settled_snapshot_id`:

```
sqlite> SELECT (cancel_settled_snapshot_id>0) AS settled, COUNT(*)
        FROM armed_orders WHERE state='cancelled' GROUP BY 1;
0|69
```

`ConfirmCancel` (`store/armed_orders.go:466`) is the ONLY writer that records
evidence — it requires the id of the snapshot whose book no longer listed the
order — and it has exactly one production caller,
`trader/cancel_confirm.go:381`.

**The bypass is `trader/armed_executor.go:2198`:**

```go
_ = ledger.SetState(r.ID, "cancelled", reason+" (ack timeout — flatten proceeds)")
```

That promotes a row straight to `cancelled` on a TIMEOUT, skipping the whole
`RequestCancel → cancel_pending → ConfirmCancel` lifecycle the 2026-09-06 wave
was built to enforce. `cancelled` is the word that frees the slot
(`UpsertArm`) and the word cutover leg 4 counts, so a row promoted on a timeout
can be replaced while its order still rests.

**This is built ≠ wired, on the wave that was ABOUT settlement.** The lifecycle
exists, is correct, and is tested; the path that actually retires rows at a
close does not use it. Class 95's shape (the guard is on the path that does not
run) reaching the settlement machinery itself.

## 2 · Every flatten trusts the ledger over the broker

All five position reads in the flatten paths are local:
`trader/auto_trader_clock.go:80`, `:158`, `:533` (EOD), `:648`, `:721` (T1) —
every one `at.store.Position().GetOpenPositions(at.id)`.

The codebase documents that store as lagging real exits:
`trader/position_desync.go:18` — *"for up to ~80s after a real exit, the store
row is still OPEN"* — and `:78` records the opposite skew, the store showing
OPEN while the broker reports FLAT.

**The broker's own answer is already in the process.** `at.liveBook(now)`
(`trader/cancel_confirm.go:236`) and `at.brokerBook()` (`trader/f12_leg4.go:182`)
serve the F12 order snapshot that cutover leg 4 answers from — and the cutover
gate deliberately splits leg 1 (sqlite) from legs 2/3 (broker) so a routing
fault cannot hide behind one number. **The flatten has only the sqlite leg.**

A flatten that trusts the ledger over the broker is the 2026-09-06 naked-stop
shape arriving from the exit side: our record says flat, the broker says
otherwise, and the close is the one moment where the two must agree.

## 3 · The no-link fallback writes `cancelled` with zero broker contact

`trader/armed_executor.go:2095` — `return at.cancelArmedOrders(reason), 0` —
taken exactly when `armedTrader()` is nil, i.e. when NT8 is unreachable, which
is precisely when resting orders are most likely to outlive us.
`cancelArmedOrders` (`:2015`) walks `ListNonTerminal` and calls
`SetState(r.ID, "cancelled", reason)` with no wire, no book, and no state test.
Its count then feeds the operator-facing flat claim.

## 4 · "Acked" means "left the non-terminal set", so a FILL counts as a cancel

`trader/armed_executor.go:2190` — `acked = !at.armedRowStillActive(ledger, r.ID)`
(`armedRowStillActive` at `:2207`). A fill arriving during the drain window
writes state `filled`, which is terminal, which removes the row from
`ListNonTerminal` — and is therefore counted and logged as a successful cancel.
A limit that filled two seconds before the close is reported as an order we
cancelled.

## 5 · The flatten is the last unguarded cancel site

`cancelArmedOrdersSyncWith` contains **0** `cancelSafetyFor` calls while seven
per-arm sites have one. **This is the owner's ruling of 2026-09-07 and it
stands** — a guard that refuses without a book, applied to the EOD flatten and
the news-halt sweep, leaves arms live and re-opens class 33. Recorded because the
harm is currently bounded by the C# side no longer walking `placedBrackets`, and
that bound lives in the AddOn rather than in Go.
