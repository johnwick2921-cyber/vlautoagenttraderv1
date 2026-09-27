# VOID PARITY — one resolver for the void scope

Branch `fix/void-parity-inputs` · worktree `/home/hoang/nofx-voidparity` ·
checklist class **51** (highest occupied at merge: 50).
Status: **BUILT, GREEN, STAGED — awaiting owner GO. Not deployed.**

## THE CLASS IN ONE SENTENCE

Two callers asked one predicate the same question and handed it different
inputs, and the test written to catch exactly that built both sides' inputs
itself — so it proved the two functions agree while production fed them
different tape.

---

## WHAT THE ORIGINAL DISPATCH ASSUMED, AND WHY IT WAS DROPPED

The wave was dispatched as WATERFALL-NAMES-LEVEL on my own incorrect report that
29141.25 was "not a seated level, only a candle low". **It is ONL, and it sits in
that very prompt's ranked list at 0.00 pts.** I had grepped the candle table and
the VOID list and never the ranked list. D1 (a new reject for off-list waterfall
references), D2 (the prompt constraint) and the must-cite-seated-level rule were
dropped by the owner on that measurement: there is no evidence of a waterfall
authored at an unseated price, and A24 forbids a reject gate for a defect that
has not occurred.

## D1 — THE REAL DIVERGENCE, FROM SOURCE

| | window | bars |
|---|---|---|
| prompt render, `trader/auto_trader_planner.go:2304` | `voidWindowStartMs` = CME session-day start | `bars1m` = **12,000** |
| write-site validator, `kernel/breakdown_continue.go:248` | **`sinceMs = 0`** | `bdBars` = `AISVPBarCount` = **2,000** |

`BreakdownContinueState` filters each bar on `b.OpenTime < sinceMs`, so the
window is load-bearing, not decorative. A level broken and reclaimed before the
17:00 CT boundary is void to the validator and invisible to the prompt.

**The 20:58 CT read:** eight seated levels listed VOID, ONL 29141.25 absent, read
rejected on ONL.

## WHY THE CLASS-45 PARITY FIXTURE PASSED THROUGH ALL OF IT

`TestClass45VoidListMatchesValidator` reported `parity: 40/40 checks agree across
20 tapes` while this was live. It fed **both sides the same `sinceMs`**. It
pinned the two FUNCTIONS; it never pinned the CALL SITES. A parity test that
constructs both sides' arguments can only prove self-consistency.

## THE FIX

**`kernel/void_scope.go`** — `VoidScope{Bars, SinceMs, Interval, BarCount, Source}`
and `ResolveVoidScope(symbol, now)`. Neither caller picks a window or a slice.
All three call sites resolve it: the prompt render, the write-site validator, and
the shadow A/B (`trader/rootfix_shadow_ab.go`). Both
`ComputeVoidBreakdownLevels` and `ValidateBreakdownContinueScenarios` now take
the scope. `voidWindowStartMs` is **deleted**.

**The scope VALUE is owner-ruled as the CME session day**, and this is a
behaviour change on the validator side: it narrowed from the whole slice to the
session-day window, so a level broken and reclaimed days ago no longer voids a
play today. That is **strictly fewer rejects**.

**The first cut was wrong in the other direction** and the owner caught it. It
made both sides read the whole slice, because that was the validator's history.
Replayed against the real tape that produced **20 void entries across 12 levels**
— nearly every level, both sides — a list that says "author no waterfall play
anywhere". The deleted render-side comment ("a level broken and reclaimed days
ago is not today's news") was correct; its only error was being applied to ONE
caller.

**Compact render** (owner ruling): one line per level, both sides folded in.

```
## VOID breakdown levels (a close came back across since the break, THIS session day — the write-site validator REFUSES a waterfall play at these)
- 29141.25 breakdown+breakup (reclaimed 2026-09-02 17:04 CT)
- 29193.38 breakdown (reclaimed 2026-09-02 17:21 CT)
- 29208.75 breakup (reclaimed 2026-09-02 18:48 CT)
- do NOT author breakdown_continue or breakup_continue at these prices. Any other condition is legal there.
```

No cap and no side filter — the read-facts rows measure the real list size first.

## D3 — THE READ IS RECORDED WHETHER OR NOT IT FAILS

`store/planner_read_facts.go` — one `planner_read_facts` row per read, ACCEPTED
OR REJECTED: the rendered void list verbatim, `void_count`, the stop floor with
its ATR and multiplier, the bias/regime labels, prompt hash, and the resolved
scope (`since_ms`, bar count, interval). Cap **500**, oldest trimmed.

Until now a rendered prompt was persisted only in `planner_rejected_prompts`, on
failure. Class 45's sections could be proven live only because a read happened to
be rejected: *"does the fix work"* and *"is the fix present"* were the same query,
and a perfectly working fix erased its own evidence. An empty list encodes `[]`
(computed and empty); not-computed is `""` — the distinction is deliberate (A24).

Journal line per read:
`📓 read facts: void=N · floor=X pts (1.5×ATR5m Y) · scope=1m×2000 since=Z (cap 500)`

## BOOT LINE

`📜 void scope: session-day window · 1m×2000 · one resolver for prompt AND validator (parity)`
— every field read from its resolver.

## TESTS

| id | pins | result |
|---|---|---|
| `TestVoidParityPreSessionReclaimIsVoidForNeither` | **owner-requested fixture:** a reclaim BEFORE the session-day start is void for NEITHER caller | PASS |
| `TestVoidParityInSessionReclaimIsVoidForBoth` | the positive half — an in-session reclaim voids for BOTH | PASS |
| `TestVoidParityWindowIsTheMechanism` | whole-slice=2 entries vs ruled session-day=0 on the same tape | PASS |
| `TestVoidParityCallSitesUseTheResolver` | **source pin:** no call site may pick its own window/slice; `voidWindowStartMs` cannot return; the ruled window IS `CMESessionDayStart`; boot-line fields | PASS |
| `TestReadFactsPersistOnEveryReadAndCap` | an ACCEPTED read records its facts; void list round-trips verbatim; `[]` ≠ `""`; cap trims oldest, newest survives | PASS |
| `TestReadFactsNilStoreIsSafe` | telemetry never fails a read (A10) | PASS |

The RED, before the resolver existed:
```
void_parity_test.go: PARITY BROKEN: the validator voids 29141.25 but the prompt's
VOID list omits it (0 entries) — the model is told nothing and authors straight
into the reject
```

**Suites:** Go **27 ok / 0 FAIL**, goldens PASS, `go vet` clean, `tsc --noEmit`
clean, **vitest 38 files / 298 tests**.

## A15 — WHAT IS STILL NOT PROVEN, AND WHAT THE OWNER WILL SEE

- **The 20:58 omission is NOT reproduced.** A replay of that ranked list against
  the `bars` table returns 20 entries under BOTH the old and the whole-slice
  scope, with ONL present in both — the live prompt had 8 and no ONL. The `bars`
  table is not the live BarCache, so the replay cannot stand as proof that the
  window explains that specific omission. The divergence is real **in the
  source**; which of the two differences (window or 12,000-vs-2,000 bars)
  produced that omission is **not established**. The read-facts row answers it on
  the next occurrence — that is what D3 is for.
- **The validator now rejects less.** A pre-session-day reclaim no longer voids a
  waterfall play. If a stale level turns out to matter, this is the knob to
  revisit, and the read-facts rows are the evidence base.
- **Two provider fetches per read remain** (render and validate resolve
  separately, seconds apart). Both now use the identical resolver, so the class
  is closed; a level that becomes void in those seconds could still miss the
  prompt. Narrow, and inherent without threading the scope through the read.
- **`GUIDE_BUILT_REV` is not bumped here** — that belongs to the marker commit.

## ROLLBACK

Additive plus two signature changes; no data migration beyond an AutoMigrate that
creates `planner_read_facts` (dropping the table is safe — nothing reads it on a
hot path). `git revert` the wave commits, rebuild, restart.
Binary rollback: `nofx-bin.prev.boot`.

## CHECKLIST

Class **51** appended, with the probe: *for every predicate with more than one
caller, diff the ARGUMENTS at each call site, not the function; any test that
constructs both sides' inputs proves only self-consistency; and the instrument
that proves a fix works must not fire only when the fix fails.*
