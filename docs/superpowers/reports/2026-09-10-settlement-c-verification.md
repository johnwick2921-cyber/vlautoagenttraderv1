# Section C re-verification at dev — settlement and flat truth

**Spec:** `reports/2026-09-09-next-wave-basis-settlement-and-flat-truth.md`,
`git log -1` → **`1718cf75` 2026-09-09 14:28:22 -0500** (SPEC-FRESHNESS LAW).
**Measured at:** `954f11b1`. **Re-verified at:** dev `7f6ee3d3`.

Between those two the arm-state wave landed in exactly this area
(`armed_executor.go`, `armed_orders.go`, `one_contract.go`, `f12_leg4.go`), so
every line reference was re-read rather than trusted.

| # | claim | at dev | note |
|---|---|---|---|
| C1 | ConfirmCancel bypass at `armed_executor.go:2198` | **HOLDS**, line exact | `_ = ledger.SetState(r.ID, "cancelled", reason+" (ack timeout — flatten proceeds)")` |
| C2 | five flatten reads from the local store | **HOLDS**, all five lines exact | `auto_trader_clock.go` 80 · 158 · 533 · 648 · 721 |
| C3 | no-link fallback writes `cancelled` with zero broker contact | **HOLDS** | `:2095` → `cancelArmedOrders` at `:2015` |
| C4 | `acked` means "left the non-terminal set", so a FILL counts as a cancel | **HOLDS** | `:2190`, `armedRowStillActive` at `:2207` |
| C5 | the flatten is the last unguarded cancel site | **HOLDS** | `cancelArmedOrdersSyncWith` contains **0** guard calls |

## Two corrections to the spec's own detail, neither changing a finding

**C2 — the count.** `grep -c GetOpenPositions trader/auto_trader_clock.go` now
returns **6**, not 5. The sixth is a COMMENT at `:688` ("GetOpenPositions only
returns open ones, so retries are naturally…"). Five real call sites, exactly as
specified.

**C5 — the seventh guarded site is guarded through a WRAPPER.** The spec lists
seven per-arm sites carrying `cancelSafetyFor`, including
`trader/position_desync.go:109`. A raw grep for `cancelSafetyFor` finds only six
and does NOT match that line, because it calls
`at.cancelSignalIfSafeWith(nt.CancelOrder, …)` — which adjudicates internally via
`adjudicateArmCancelWith(store.StateWorking, signalID, book, have, pos)`, the
same adjudicator. **The site is guarded; the grep was wrong, not the spec.**

Recorded because I nearly filed it as "the spec has drifted" on the strength of a
grep count, which is the failure this repo has been filing all day: a
characterisation repeated without opening the ref. One `awk` over the wrapper
settled it.

## What ties C1–C5 together, restated from the spec

Four of the five are one sentence: **a word in our ledger is written from
something other than the broker's answer.** `cancelled` from a timeout (C1), from
a missing link (C3), from a fill (C4); `flat` from a local table (C2). C5 is the
guard that would have asked the broker, deliberately absent from the one path
that matters most.
