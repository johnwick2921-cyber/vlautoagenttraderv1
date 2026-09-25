# SIGNAL CLOCK — the wire timestamp is a creation time, not a market fact

**Wave:** signal-clock · **Branch:** `fix/signal-clock`
**Lane:** `signalclock-ee7f9468/nofx-db[ca9c60]` · claimed 2026-09-07T23:54:39-05:00
**Base:** dev `33c4bf78`

---

## 0. CORRECTION OF RECORD — the 180-second delta was NOT REPRODUCED

The dispatch for this wave was opened on a finding I reported: that Go and the
NT8 AddOn disagreed by **exactly 180 s** on the age of the same signal, across
four independent rejections, and that a constant offset means one wrong term
rather than drift.

**Measured against both logs, the two clocks AGREE.**

| rejection (CT) | Go's own age (`🚨 FEED DOWN`) | NT8's age (`stale signal`) | Δ |
|---|---|---|---|
| 22:28:55 | 11m26s = 686 s | 715.3 s | 29.3 s |
| 22:47:24 | 30m24s = **1824 s** | **1824.5 s** | **0.5 s** |
| 23:05:23 | 48m24s = **2904 s** | **2903.6 s** | **−0.4 s** |
| 23:23:20 | 1h6m21s = **3981 s** | **3981.0 s** | **0.0 s** |

Both sides imply the same frozen reference instant, **22:16:59–22:17:00 CT**:

```
NT8  sigTime = reject_time − age  →  22:16:59.7 · .5 · .4 · .0
Go   newestStamp = alert_time − age  →  22:16:59 · 22:16:59 · 22:17:00
```

**The artifact.** `checkFeedDown` fires on a **1-minute monitor tick**
(`trader/auto_trader_feedwatch.go:51-61`, `age = now − (newestBar.OpenTime +
60_000)`); NT8's age is computed **at signal arrival**
(`ninjascript/VLTraderTCPClient.cs:814`, `ageSec = DateTime.UtcNow − sigTime`).
Pairing an alert sampled up to three minutes earlier with a rejection sampled at
arrival manufactures a constant-looking offset out of **two sampling cadences**.

Three of the four pairs agree to ≤0.5 s. The first differs by 29.3 s — the
alert-cadence granularity, not a term. And the audit's quoted "Go 535 s" appears
**nowhere in the log**: the minimum Go age recorded is 626 s (`10m26s`).

The hosts' clocks were never in question — `clock-guard [boot] rtc_vs_go=0s`
already said so, and this measurement confirms it.

**What was actually wrong was never two clocks. It was one frozen bar cache**,
and the fix is the wall-clock stamp on every outgoing command — three paths by
the placement-truth lane, the fourth here.

**Lesson recorded:** every individual number in the false finding was true; the
subtraction was wrong because the operands came from different sampling
cadences. That is harder to catch than a fabricated number, and it is why A17
requires a premise to be measured before it is built on.

## 1. What shipped

`MoveStopPayload.Timestamp` (`trader/ninjatrader/tcp_trader.go:677`) now stamps
`time.Now().UTC()`. It was the **last** outgoing command still using
`feedNowUTC` — the last bar's close.

**Inert today, fixed anyway.** The AddOn parses `timestamp` in exactly one place
(`HandleSignal`); `HandleMoveStop` reads only `signal_id` and `new_stop_loss`.
Two reasons it does not wait:

1. A freshness guard on `move_stop` is the obvious hardening after the
   placement-truth wave, and would inherit the bug **fully formed** — auto-
   breakeven would then fail during feed gaps, which is when a runner most needs
   its stop moved.
2. The internal inconsistency taught the wrong convention: the sibling
   `PlaceProtectiveStopPayload` already stamped `time.Now().UTC()`, so one file
   said two different things about what `timestamp` means.

`feedNowUTC` keeps its definition — the placement-truth lane's pin
(`placement_truth_test.go:31`) reads it as the bar-clock reference — and now has
**zero production call sites**, which is the intended end state.

## 2. The audit that preceded the change

`feedNowUTC` has **4 production call sites**, all in one file. Classification:
**32 class-A** (latency/staleness → wall clock), **3 class-B**, **3 class-C**,
and **zero class B among the `feedNowUTC` sites** — it was used *exclusively*
for wire timestamps, never for a market fact, so there was no risk of the
inverted bug (flipping a market fact onto the wall clock).

Three of the four were already fixed on dev by lane
`placement-truth-96604090/root[unlisted]`, claimed **25 minutes before this
wave** — a collision two honest claims could not detect, because the wave names
(`placement-truth`, `signal-clock`) share no words while touching the same lines.

## 3. Pins — mutation-tested, and the first mutation was a no-op

`TestMoveStopStampIsWallClockNotBarClock` · `TestOutboundCommandStampsAgreeOnTheirClock`

The fixture is its own (`moveStopServer`): `stopEntryServer`'s reader **discards
every non-signal frame**, so reusing it would have silently dropped the frame
under test.

**The first mutation attempt passed and proved nothing.** The search string used
`Timestamp:   time.Now()` (three spaces) where gofmt had aligned
`Timestamp: time.Now()` (one). It matched **zero** occurrences, mutated nothing,
and the suite went green — which was briefly read as "the pins bite".

Corrected, asserting the match count first:

```
occurrences of 'Timestamp: time.Now().UTC().Format(time.RFC3339),' = 1
MUTATION APPLIED (line 693, move_stop back to the bar clock)
--- FAIL: TestMoveStopStampIsWallClockNotBarClock
    move_stop stamp is 31m1s old with a 31-minute-stale bar cache — it is on the BAR clock
--- FAIL: TestOutboundCommandStampsAgreeOnTheirClock
    an outgoing command is still stamped from the BAR clock
```

Filed as the second pin of **class 89**: a mutation that passes is not evidence;
the wave must show the mutation actually changed the source.

## 4. Class 89 — a wave verified by a toolchain that cannot see half of it

My own place-confirmation wave shipped `ninjascript/VLTraderTCPClient.cs` to dev
with **two compile errors** while reporting *"Suite: 28/28, 0 FAIL"*. That was
true **of the Go half**: `go build`/`vet`/`test` cannot compile NinjaScript, so
every check that passed was blind to the broken file. It did no runtime damage
only because NinjaScript needs copy → F5 → restart — the damage would have
landed on whoever ran that next, with no reason to suspect it. Another lane
repaired it.

**Standing pin:** any wave touching `.cs` states plainly that NinjaScript is
outside the Go toolchain and names what was and was not compiled.

**Applied to this wave:** `git diff --name-only` → `AUDIT-CHECKLIST.md`,
`move_stop_clock_test.go`, `tcp_trader.go`. **No `.cs`.** There is no unverified
half.

## 5. A15 — what the owner will still see wrong

- **`backoff=UNKNOWN`** on the session-calendar boot line — the publish sits
  inside `backoffWhileClosed`, which only runs when the market is closed, so at
  boot it has never been set. Truthful, still wrong seam. Outstanding.
- **The AddOn's `reason` field is not live** until the next copy/F5/restart;
  rejections record the honest fallback until then.
- **`feedNowUTC` still exists** with zero production call sites, kept alive by
  another lane's pin. Intentional, but it is a loaded gun for anyone who reaches
  for a "now" in that file.
