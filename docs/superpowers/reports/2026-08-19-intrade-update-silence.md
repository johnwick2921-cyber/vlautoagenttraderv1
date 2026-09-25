# In-Position Update Silence — P0 Dispatch Report (2026-08-19)

Branch `fix/intrade-update-silence` · 3 fix commits + this report · preempts
`fix/ledger-close-sep-risk` (zero commits there — sequencing note in §6).

**Owner's symptom:** "UPDATES STOP WHILE POSITION OPEN." Confirmed live mid-fix
by the owner, twice: "ai call stop at Cycle #20149 8/19/2026, 7:17:32 AM after
get in trade" and "it resume back after stop loss." Both match the mechanism
below exactly. [A]

---

## 1. Layer table — what was alive vs. dead while holding

Evidence window: position #522 (SHORT 29558.25, 07:17:17→07:33:04 CT today,
−60.5 via stop) and #521 (SHORT, 03:22→04:43 CT, +138). [A]

| Layer | In-position state | Evidence |
|---|---|---|
| C# AddOn → TCP bar frames | **ALIVE, full rate** | `wire_liveness` during #522: 13–66k frames/min, `bar_age 1m=42–45s` throughout [A] |
| Go BarCache ingest | **ALIVE** | same liveness lines print newest-open ages advancing every 60s [A] |
| SSE / chart / klines API | **ALIVE** | chart drew live bars through both positions (owner-visible) [A] |
| Position state from NT8 | **ALIVE** | #522 exit recorded 07:33:04 with `close_reason="sync"` (position_close frame → reconcile) [A] |
| Bracket / auto-breakeven (60s risk loop) | **ALIVE** | separate goroutine (`startDrawdownMonitor`), independent of runCycle [A] |
| **Go decision cycle (runCycle body)** | **DEAD** | #521: 1 scan in 82 min; #522: 1 scan in 16 min (the entry itself) [A] |
| ↳ feed-down watch (`checkFeedDown`) | **DEAD** | sat at loop:247, below the skip return [A] |
| ↳ clock-health session-roll | **DEAD** | sat at loop:258, below the skip return [A] |
| ↳ equity snapshot (`saveEquitySnapshot`) | **DEAD** | sat at loop:299, below the skip → equity curve froze for the life of every position — the owner's visible freeze [A] |
| ↳ decision records | **DEAD** | no rows between entry and exit → decision feed frozen [A] |

## 2. First-silent-layer verdict + provenance

**First silent layer: the Go scan loop — suspect S2.** The guilty branch is
skip-while-open, which returned from `runCycle` at `trader/auto_trader_loop.go:219`
(gate defined `auto_trader_clock.go:34-40`) BEFORE context build, equity
snapshot, feed watch, and clock health. C# (S1) is eliminated by the liveness
receipts above; SSE/chart (S3) and position frames (S4) were alive. [A]

**Provenance:** commit `3bb7a730`, 2026-08-14 22:26 CT — "feat(dayplan): P2.2 —
skip-while-open gate," part of the day-plan campaign. Deliberate spend-saving
design ("calmer + cheaper than same-side refusal"); the defect was its
*placement*: everything observability- and guard-shaped sat downstream of the
early return. Every position since 08-14 22:26 froze the dashboard for its
entire life. [A]

## 3. The four questions (1.4)

While a position was open, was the bot…

- **(a) ingesting bars?** YES — full rate, receipts in §1. [A]
- **(b) running scans?** NO — one decision row per position (the entry), zero
  until the close. [A]
- **(c) running staleness/liveness/clock guards?** PARTIAL — the server-level
  60s `wire_liveness` reporter YES (it lives in tcp_server.go, outside the
  cycle); the in-cycle guards (feed-down alert, clock-health roll, B4
  evaluation) NO — all below the skip. **Worse: under bar-close cadence a dead
  feed stops the cycles themselves, so the in-cycle `checkFeedDown` could never
  fire during a real outage even when FLAT** (`tickOnce`,
  auto_trader_clock.go:466-470: no new closed bar → idle tick). The fix removes
  both blind spots at once. [A]
- **(d) able to see NT8 position state?** YES — close-sync/reconcile recorded
  the stop-loss exit within seconds. [A]

## 4. The fix (3 commits, per the IN-POSITION CONTRACT)

**`44808dc0` — guards outlive the position.** `checkFeedDown` moves from
runCycle to `monitorTick` on the 60-second wall-clock drawdown-monitor ticker
(`auto_trader_risk.go`) — it now beats regardless of position state AND
regardless of bar-close cadence (fixing the latent flat-outage hole in §3c).
The clock-health session-roll hoists above the skip. Nil-trader guard added to
`checkPositionDrawdown` (shutdown race / harness safety).

**`37b3b6bd` — louder in-position, never quieter.** While any OPEN position row
exists (NOT gated on day_plan), the feed-down threshold tightens from 10 min to
`INTRADE_FEED_ALERT_S` (default **120s**; documented in `.env.example`). The P0
banner becomes **"⚠ NO DATA WHILE POSITION OPEN — check NT8"** with an
in-position body naming the bracket protection and the blindness. Same alert
kind (`feed-down`) → rides the existing AlertCenter P0 persistent-banner path,
no FE change.

**`c335f897` — AI-skip only AFTER snapshot+broadcast.** Skip-while-open
relocates BELOW `buildTradingContext` + `saveEquitySnapshot`. In-position, each
bar-close cycle now: builds the full context → records the equity snapshot →
writes a heartbeat decision row (`skip-while-open: holding …` with account
state) → THEN skips the AI call as the documented spend-saving branch. Equity
curve and decision feed move for the life of the trade; AI spend unchanged
(still zero calls while holding). A source-order contract test
(`trader/intrade_contract_test.go`) pins snapshot-before-skip and
feed-watch-outside-runCycle so a refactor can't silently reintroduce the freeze.

**No C# change required** — S1 was eliminated by evidence, so the owner
lockstep-deploy question never arises.

**Dashboard (2.3), no FE commits:** positions panel polls `/api/positions`
every 5s (`REFRESH_POSITIONS_MS=5000`, AppRoutes.tsx:347) — already better than
"each bar"; PnL is backend-computed from NT8 state. The P0 banner is the
existing AlertCenter behavior (banner + toast until ack).

## 5. Verification V1–V5

- **V1/V2 (live in-position continuity):** deploy landed in a flat window
  (07:46 CT, pre-NY). Proof point is the next organic entry: expect per-bar-close
  heartbeat decision rows, a moving equity curve, and 5s positions-panel
  updates while holding. Pre-fix baseline for contrast: #521/#522 (1 scan in
  82/16 min). [B until the next entry, then A]
- **V3 (feed loss while holding, simulated):** unit tests —
  `TestFeedWatchBeatsWhileHoldingAPosition` (monitor beat fires the P0 while
  holding, 11-min gap), `TestInPositionFeedAlertTightens` (3-min gap: quiet
  flat, LOUD holding, dispatch banner title asserted),
  `TestInPositionFeedAlertEnvOverride` (INTRADE_FEED_ALERT_S=300 respected).
  A real 11-minute NT8 kill remains undrivable from WSL (AddOn auto-reconnects,
  iptables needs sudo) — same simulation precedent as #48/T7. [A for the code
  paths, simulated]
- **V4 (regression):** `go build ./...`, `go vet ./trader/`,
  **full `go test ./...` — zero failures**. Web untouched → no FE build needed. [A]
- **V5 (flat path unchanged):** flat cycles execute the same steps in the same
  order (clock-health kept its once-per-roll dedup; skip branch not taken when
  flat; candidate/AI flow byte-identical). Only differences when flat: the feed
  watch now checks every 60s from the monitor ticker instead of once per cycle
  (same per-gap dedup → same alert count), and its log line adds
  `holding=false`. All pre-existing trader tests pass unmodified. [A]

## 6. Sequencing note (ledger-close preemption)

Per the dispatch's sequencing rule: `fix/ledger-close-sep-risk` had **zero
commits** (Phase-1 recon only) when this P0 preempted it, so this branch was
cut from the deployed head (`00a4bef6`) and ledger-close will restart REBASED
on top of this fix after it lands. No commits were interleaved. The `stopUntil`
dead consumer (loop:~245) was deliberately NOT touched here — it belongs to
ledger-close P2.

Env knob shipped: `INTRADE_FEED_ALERT_S` (seconds, default 120, in-position
only). Normal flat threshold stays 600s.
