# W-PICTURE-HTF — Verification Trace (DS-102, 2026-09-19/20)

Inspection base (dispatch's historical context): `30db2e25ae9ce564198d37d629cb729884449bd5`.
**Acceptance base (fresh dev tip):** `origin/dev = d7ca3846` — the wave was REBASED onto it 2026-09-20 (branch `fix/picture-htf` @ `1aa1733d`); the earlier base diverged below the #177 merge and has been corrected. `git log -1 -- docs/superpowers/reports/2026-09-19-picture-htf-verification.md` at accept = the claim commit `ceb894b9` rebased onto d7ca3846.
Worktree `/home/hoang/nofx-102-picture` is `git worktree lock`-ed; the main-tree deployment lock is NOT held during implementation (released 2026-09-20; it will be re-acquired only for the owner-attended cutover).

## 1. Verification table

| Planned behavior | Existing production path (file:line) | Missing / change required | Test that will prove it |
|---|---|---|---|
| Native NT8 bars arrive | C# `VLBarsSubscriptionManager` → `VLTraderTCPClient.SendFrame("bar_update")` (VLTraderTCPClient.cs:2660-2667) → Go `BarUpdatePayload` (provider/ninjatrader/tcp_framing.go:593) → `BarCache.Upsert` (provider/ninjatrader/bar_cache.go:341) | wire `Bar` has NO emission time, NO final/completed marker, NO contract field (tcp_framing.go:555) | C# emits `emitted_at` + `final` + capability; Go pin proves receipt |
| Bar completion evidence | Forming-bar caveat only (bar_cache.go:140); completion inferred from a NEWER bar | **GAP (addendum #2):** wall-clock cannot prove the final OHLC; session-end bars close without a successor | C# `final:true` on the last update of each bar; session-end proof |
| Fade-only play authorization | `ArmableCondition` (kernel/arm_kind.go:34), arm refusal site (trader/armed_executor.go:514) | `picture_htf` mode must carry its OWN authorization path; keep arm rules untouched | Mode selector test + armability unchanged test |
| Plan modes | `DayPlanConfig.PlanMode` advisory\|direction\|strict (store/strategy.go:918-919), `PlanModeFor` (:1481) | add `picture_htf` as a named mode | Resolver test |
| Confirmation rules | `EntryLawFor` (kernel/entry_law.go:80), `ValidateEntryLaw` (:144), unknown conditions rejected upstream (:151) | add `h1_close_break` rule; unknown must reject explicitly | entry_law tests |
| Entry gate / risk | `EntryGate` (trader/entry_gate.go), account limits, risk_control | reuse unchanged; mode passes THROUGH the shared gate | gate test with a picture_htf decision |
| Market entry submission | `TCPTrader.OpenLong/OpenShort` (trader/ninjatrader/tcp_trader.go:260/264) → `placeEntry` (:324); `PlaceLimitEntry` (:437), `PlaceStopEntry` (:498) with `beforeSend` persistence callbacks | add a dedicated `MarketEntryWithProtection(side, qty, sl, tp, beforeSend)` on the CONCRETE type (not the 19-method interface — trader/types.Trader untouched) | wire + persistence test |
| Order frames → state | `TCPTrader.OrderUpdates()` channel (tcp_trader.go:605), `OrderUpdatePayload` | rejection reason absent on wire — add C# field | frame pin test |
| Protection (bracket) | `ModifyBracketPayload` (tcp_framing.go:145); C# `SubmitBracketOnEntryFill` | verify from RECEIVED frames, never infer from submit | bracket receipt pin |
| Dashboard decisions | `store.DecisionAction` + `/api/decisions/*` | add opportunity detail endpoint + cards | handler test |

## 2. Addendum resolutions — status table

DESIGN-RESOLVED = the rule is documented and implemented in code where it exists; IMPLEMENTED+VERIFIED = production-path tests prove it. Integration items stay OPEN until their evidence lands.

| Item | Status | Notes |
|---|---|---|
| 1 simultaneous H1/4H close | DESIGN-RESOLVED | snapshot over completed bars only; retirements applied before target selection, after confirmation |
| 2 native completion evidence | OPEN (needs C# wire) | requires `final` + `emitted_at` on bar frames; Go consumes only final bars |
| 3 freshness / clock skew | DESIGN-RESOLVED, OPEN (integration) | four stamps separated; skew bounds defined; recheck at send — evaluator wires it |
| 4 one execution owner | DESIGN-RESOLVED, OPEN (integration) | unique row claim + atomic confirmed→place_pending transition; must still be proven against BOTH executors |
| 5 swing / target semantics | DESIGN-RESOLVED, OPEN (integration) | strict comparisons + freeze; target re-evaluated at submit, nearer never skipped |
| 6 market fills & protection | OPEN | pre-submit vs actual R:R stored separately; protection timeout/recovery needs an owner ruling if the existing bracket path does not already define it |
| 7 activation & rollback | DESIGN-RESOLVED | strategy/account quoted at cutover; capability floor for the AddOn; config backup first |
| 8 verification outcomes | PARTIAL | replay/tests/controlled-SIM/natural categories kept separate |

Resolutions in detail:

1. **Simultaneous H1/4H close.** Resolved by ordering: eligibility of levels is frozen from a snapshot of COMPLETED 4H candles only (the 120-bar window excludes the forming 4H); the H1 evaluation uses that snapshot; 4H retirements (completed 4H closes through the far edge) are applied to the snapshot BEFORE target selection, AFTER confirmation evaluation. Order-independent because both read only completed bars and the snapshot is immutable per evaluation.

2. **Native completion evidence.** Requires the C# AddOn to mark each bar frame with `final` (emitted when NT8 closes the bar and no further update for that index will arrive) + `emitted_at`. Go-side evaluators consume only `final:true` bars for completed-candle rules. Session-end bars: NT8 closes them on session end; the AddOn must emit `final` then (same mechanism). Missing `final` ⇒ opportunity unavailable (not guessed).

3. **Freshness.** Four separate stamps: `bar_time` (candle's own close/open, market clock), `emitted_at` (C# wall clock), `received_at` (Go TCP read), `evaluated_at`/`submitted_at` (Go wall clock). Clock skew: reject when `|received_at − emitted_at| > 5s` (transit bound) or when `emitted_at` is in the future beyond tolerance. The 10s submission window and 2s data-age limit are re-checked at send time from `received_at` of the freshest frame.

4. **One execution owner.** Serialize admission through a per-(account,instrument) atomic claim in the store (single-writer SQLite connection; `place_pending` write is the claim). The existing executor checks `no position && no outstanding entries` via its own gate — the new path additionally requires the claim row to be uniquely inserted (opportunity key unique index) AND the entry gate re-check inside the same critical section. An ambiguous send stays `place_pending` with no signal ID → blocks re-entry until reconciled against NT8 orders (existing reconcile path reused).

5. **Swing/target semantics.** Strict wick comparisons (`<`/`>`, not `<=`/`>=`) for pivot detection; both confirming candles must be `final` by the H1 close; the stop is frozen in the opportunity row at confirmation. At submission, target/R:R are re-evaluated from the LATEST price; if a nearer eligible opposing zone exists, it becomes the target even if R:R then fails → refuse (never skip a nearer zone to fix R:R). Record any target change.

6. **Market fills & protection.** Pre-submit R:R stored as `r_r_estimate`; actual fill R:R stored separately on the fill event. Structural stop/target are NOT moved to repair slippage. Partial fills: existing `netting_fills.go` + reconcile handle quantity shortfalls; protection timeout and recovery: **OWNER RULING REQUIRED** before activation unless existing bracket behavior already defines it (to be confirmed during implementation).

7. **Activation & rollback.** Selected trader = owner's `hoang` SIM strategy (exact strategy id + account binding to be quoted at cutover). Flat-state + no outstanding entries verified from FRESH broker evidence (NT8 snapshots), not DB only. Config backup before change. Rollback = previous mode + prior binary (`nofx-bin.old.<rev>` slot) + AddOn compatibility floor (`MinAddonBuildPictureHtf` capability gate — old AddOn ⇒ mode prints unavailable, never active).

8. **Verification outcomes.** Four categories kept separate in the final report: (a) historical replay (Sept 17 labeled, feed receipt NOT claimed), (b) automated tests, (c) controlled SIM execution (synthetic opportunity seam), (d) natural market execution (pending; if none occurs: "active and ready; natural-entry proof pending").

## 3. Discrepancies found (plan vs code)

- The plan's "the entire candle body need not cross" and "one tick" close rule have no existing code — new `h1_close_break` rule required in entry_law.go (planned, not a contradiction).
- The plan assumes native NT8 H1/4H/5m bars: confirmed (subscription frame `timeframes` includes 1m..1w; tcp_framing.go BarsSubscribePayload). NT8-native bars are NOT resampled Go-side — used as-is. ✓
- The plan's "10s / 2s" defaults are engineering defaults, not researched — documented as such (guide will say the same).

## 4. Rule defaults (explicit, per dispatch)

- Pivot window: latest 120 COMPLETED 4H candles; 2+2 strict confirmation; no lookahead (pivot usable only after both trailing candles close).
- Sweep/reclaim: completed 4H trades below a known support and closes back above it (mirrored for resistance).
- H1 confirmation: prior completed H1 close at/below the boundary, new completed H1 close ≥ 1 tick above it (mirrored for short). Multiple crossings → the extreme level; others recorded as context.
- Entry: within 10s of the following 5m interval start; freshest frame ≤ 2s old (emission+receipt); late ⇒ `expired`, never chased.
- Stop: nearest 5m swing confirmed 2+2, searched over the preceding 24 completed 5m candles, 1 tick beyond the wick extreme; only swings completed by the H1 close; no swing ⇒ no trade.
- Target: nearest active 4H opposing-zone body edge; no eligible zone ⇒ no trade; configured min R:R enforced (estimate at submit, actual recorded at fill).
- Momentum stall: advisory only — computed per completed H1 close, shown in AI context + dashboard; never auto-trades.

Status: trace complete; implementation begins in following commits.
