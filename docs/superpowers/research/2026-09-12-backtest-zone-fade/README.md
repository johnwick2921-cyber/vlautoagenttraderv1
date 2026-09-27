# Backtest 1 — does the fade at a zone pay?

**Branch:** `research/backtest-zone-fade` · claim `b994de5c` (ls-remote
`b994de5cce0d7a27b9ab34dfee945b128b7bf6fc`, session
`backtestzonefade-4abbb353/copilot[unlisted]`) · accepted dev tip `379713e3` ·
running rev measured `400ea26c12c8` (`/api/health` + `go version -m
/proc/3671783/exe`, `vcs.modified=false`) — **not restarted**.

## The headline

**No.** On 4.4 years of contract-stamped MNQ (2022-04-11 → 2026-09-11,
11,302 first touches of seated zones, 2,370 in-sample reads / 714 held-out
reads), the fade at a zone has **negative net expectancy after 2.0 points of
friction on every fill assumption**: −0.79 pts/trade filled-on-touch (fill a),
−4.93 pts/trade when the fill requires a one-tick-through print (fill b, the
honest minimum), and −0.54 pts/trade with one tick of adverse slippage (fill c).
It loses in-sample (−0.84, PF 0.70) and in the never-tuned held-out year (−0.62,
PF 0.86), and **0 of 81 cells** of the swept parameter surface is net-positive.
The gross edge (+1.21 pts/trade) is real but smaller than the friction; the
zone's high "hold" rate (73.3% vs a 50.6% coin flip on the bare anchor) is a
band-width artifact, not an edge — 13f's lesson, restated.

The fill sentence (C6): the result is **negative under (a), (b) and (c)** — it
does not merely fail to survive the honest fill assumption, it is negative
under the optimistic one. Under (b), the 31.7% of touches that DO print one tick
through the anchor lose −4.4 to −6.5 pts/trade — round 10's adverse selection,
measured on this tape.

---

## Basis — everything this measures against, pinned

Every cited file's `git log -1` receipt:

| File | Last commit |
|---|---|
| `kernel/level_zones.go` (the zone builder this backtest calls) | `5e81db95` 2026-09-11 13:09:08 -0500 |
| `kernel/detector_d1prime.go` (D1′ touch detector — the outcome mechanics) | `f00c8b87` 2026-09-03 18:43:16 -0500 |
| `kernel/levels_assemble.go` (detector universe assembly) | `285838b5` 2026-09-10 18:53:54 -0500 |
| `kernel/level_zone_inputs.go` (zone width/ATR inputs) | `5e81db95` 2026-09-11 13:09:08 -0500 |
| `kernel/min_sl.go` (stop floor 1.5×ATR5m, clearance 2 ticks) | `4657560b` 2026-09-02 07:33:39 -0500 |
| `trader/arm_stop_anchor.go` (composeArmStop — ported verbatim, see methods) | `4657560b` 2026-09-02 07:33:39 -0500 |
| `trader/auto_trader_planner.go` (the live read path this backtest replays) | `8cfc7394` 2026-09-11 12:51:47 -0500 |
| `store/bar_history.go` (per-contract reader semantics) | `0aea0c2e` 2026-09-11 21:39:47 -0500 |
| `kernel/session_registry.go` (session read/flat times) | `3f23d9bb` 2026-09-07 19:30:13 -0500 |
| `docs/superpowers/reports/2026-09-11-level-zones.md` (level-map basis) | `64d75cb9` 2026-09-11 14:01:02 -0500 |
| `docs/superpowers/reports/2026-09-11-forming-candle-test.md` (13f) | `53accd5e` 2026-09-11 13:41:59 -0500 |
| `docs/superpowers/reports/2026-09-11-historical-backfill.md` (wave 101) | `999ab793` 2026-09-12 00:25:07 -0500 |

- **The zone map** (level-zones report @ `64d75cb9`): a zone is a band whose
  width is `max(defining wick, k×ATR(tf))` (k=0.5 [I]); round numbers get a
  fixed 2-pt width; native detector bounds are retained when present. Zones
  merge when an anchor is within `m×ATR5m` (m=0.5 [I]) of an existing cluster
  **and** the merged envelope stays ≤ `1.0×ATR5m`; each source joins its ONE
  nearest compatible cluster, clusters never join each other (non-transitive),
  anchors never move, and every member's name is carried. Families are
  `swing-structure · volume-node · round-number · session/derived · imbalance`
  (`kernel/level_zones.go` `ZoneFamily`), counted capped at 3. `BuildLevelZones`
  is presentation-only: no `ScoredLevel`, score, or execution anchor changes.
- **The outcome definition** (`kernel/detector_d1prime.go`, D1′ — the live
  instrument): an episode OPENS on a genuine touch (bar range contains the
  level, previous bar's does not), the entry side is the side the previous bar
  closed on, and it CLOSES when a close crosses a barrier — exiting the entry
  side is a HOLD, the far side a BREAK, horizon exhaustion is
  `ambiguous_horizon`. Resolved live knobs: k=3.0, H=12×1m, exit_on=close;
  calibrated p(hold)=0.5067 on IID-shuffled tape. This backtest uses the same
  loop with the barriers at the **zone band edges** for C2 (the dispatch's
  hold/break definition) and the detector **verbatim on the anchor** for C3.
- **13f's lesson governs every figure here** (report @ `53accd5e`): a field
  whose name implies one thing and holds another produces a result that is the
  verdict restated. Its corrected live base rate: first-touch hold **48.09%,
  n=892, Wilson [44.8, 51.4]** — the C3 comparison target.
- **Round 10** (Lalor & Swishchuk 2024, passive NQ fills ~66% immediately
  precede an adverse move; Lo, MacKinlay & Zhang 2002 on fill-on-touch bias):
  the fill assumption is the biggest threat to this backtest's honesty — C6
  reports three assumptions side by side.
- **Mesfin 2026** (arXiv 2605.04004): 14 OHLCV signal families on 947 MNQ days,
  2-point friction, walk-forward — none passed. That is the prior; a positive
  result here must survive the same standard.
- **Playbook**: `docs/superpowers/AUDIT-CHECKLIST.md` (the standing 18+ bug
  classes; this wave's measured fill-bias class is appended there as
  **CLASS 125** in the same PR as this report, per the audit-playbook law).

## Methods — reproducible from this

The harness is committed beside this report in `harness/` (Go, package `main`,
run from the repo root: `go run ./docs/superpowers/research/2026-09-12-backtest-zone-fade/harness -db <copy> -out <dir> -perms 2000`).

1. **Data.** The live store is never opened. A byte-identical copy
   (`md5 62eb9e5929d99f992f70a67a7ae4ac68`, equal to the live `data/data.db` at
   copy time 2026-09-12 07:47 CT) lives in the worktree's `data/` (gitignored).
   Bars are read per contract: `SELECT … FROM bars WHERE symbol='MNQ' AND
   tf=? AND contract<>'' AND contract<>'unrecomputable:spans_roll' AND
   COALESCE(source,'') NOT IN ('mixed','replay:off-scale')` — exactly the
   exclusions `store.BarsBetweenOn` applies (`store/bar_history.go:476`), then
   grouped by `(contract, tf)` and kept ascending. `CloseTime = open + tfMillis − 1`
   (the live ring's convention; the store's +59_999 convention is 1m-only).
2. **The contract at each instant** is `store.ContractAt` semantics replayed in
   memory: the contract of the newest usable 1m bar at or before the read
   instant. Every window is then built from **that one contract's series** —
   no window ever concatenates two contracts (C10 assert, see below).
3. **The reads.** Three per CME day, at the `DefaultSessionRegistry` times:
   LONDON 01:30, NY 08:00, ASIA 16:30 CT; session windows LONDON 02:00→08:30,
   NY 08:30→14:45, ASIA 17:00→02:00 (+1d). The live store's registry
   (`system_config.session_registry`) equals the default (ASIA/LONDON
   disabled live; the dispatch fixes all three read times for this backtest,
   so all three are replayed). Reads without tape are skipped and counted.
4. **The live detection chain, called offline — nothing reimplemented:**
   `kernel.DetectHTFLevels(fetch, ["D","4h","1h","15m"], "MNQ", readTime)` with
   500-bar fetches per tf (canonical "D"→"1d"); 1m = last 2000 closed bars
   (`AISVPBarCount`); `zoneSeries` = 1m + `AggregateBars` 5m/15m, overwritten
   per tf by the real HTF fetch, exactly as
   `trader/auto_trader_planner.go:2168-2176` builds it; then
   `kernel.AssembleResearchLevels("backtest-zone-fade", bars1m,
   DefaultSessionRegistry(), "MNQ", 8, readTime, 1.5, "", extras…)`,
   `kernel.StaleConfirmATR5m(bars1m)`,
   `kernel.LevelZoneInputs(raw, zoneSeries, readTime)`,
   `kernel.BuildLevelZones(raw, price, atr5m, inputs, kernel.ResolveZoneOptions(8), readTime)`.
   Resolved defaults used: WidthK 0.5, MergeATR 0.5, MaxWidthATR 1.0, BroadATR
   1.0, RoundWidth 2, FamilyCap 3, ShortlistCap 8; max_levels 8, proximity 1.5,
   min_grade none. **Stated divergences from the live process:** owner levels
   (👤) are not fed (they did not exist across the replay era);
   `LevelStateProvider` is nil → all-fresh (byte-identical to the pre-W11b
   goldens, `kernel/level_state_provider.go:22-27`); store-fed nPOC extras
   (`kernel.NakedPOCs`) are fed only where the COPY's `session_profiles` has
   rows (the recent store era) — for 2022→2025 the in-kernel `VolumeLevels`
   POC family is present, the store nPOC rows are absent. The planner's LLM
   scenario choice is NOT replayed — see limitations.
5. **`composeArmStop` is a VERBATIM port** of
   `trader/arm_stop_anchor.go`'s pure function (unexported — a harness in
   another package cannot call it; precedent: the accepted vet-08 replay
   shipped the same verbatim copy). The harness asserts at startup that the
   port is byte-identical to the checkout's source and aborts otherwise
   (`verifyPort`, class-97 guard).
6. **Population (C1).** A "seated zone" = a zone whose `Shortlisted` flag is
   true in the live map (rank < ShortlistCap=8 — the map's own seat concept).
   Primary population: every FIRST touch of a seated zone during its session
   (touch = bar range intersects the band, previous bar's does not, the D1′
   open condition on the band). Sensitivity: every complete-width zone
   (seated or not). A separate LINE population re-evaluates the same anchors
   with a bare 3.00-pt band (±1.5 — the pre-map `clusterToleranceFor` =
   `LevelClusterTicks × 0.25` = 12 × 0.25 = 3.00 pts, level-zones report C3).
7. **Outcomes (C2).** The D1′ loop with barriers at the band edges, close
   exits: hold = close exits the entry edge, break = the far edge, otherwise
   `ambiguous_horizon` (close mode cannot produce `ambiguous_span`). H=12 on
   1m (the live horizon), plus 10 and 20 on 5m (`AggregateBars` of the session
   tape, with the pre-session 5m bucket as touch context).
8. **The trade (C4).** On the first touch: a resting limit at the **zone
   anchor**, fading the approach (from below → short, from above → long).
   Stop = `composeArmStop(side, anchor, authored, atr5m, tick=0.25, levels,
   mult, clearTicks=2, maxAnchorATR=3.0)` with `authored` = the zone far edge
   ± 2 ticks and `levels` = the OTHER zones' anchors — i.e. stop = beyond the
   zone, floored at `mult×ATR5m` (live default 1.5), anchored wider by a
   nearer risk-side zone within 3×ATR5m, widest wins (the verbatim live rule).
   Target = the nearest opposing zone's anchor strictly in the trade
   direction (the "first opposing obstacle" of the scenario-economics
   contract, `kernel/planner_prompt.go:782`, realized deterministically).
   Exit = target, stop, or the session's flat time (last session bar close).
   A bar hitting both stop and target is counted as stop-first (conservative;
   stated).
9. **Friction (C5).** 2.0 points round-turn (Mesfin's figure) subtracted from
   every result; gross shown beside net only for scale. MNQ specifics stated
   separately: tick 0.25 ($0.50/tick), point value $2, typical retail
   commission ≈ $0.35–$0.75/side, slippage 1–2 ticks/side — the reader can
   vary them.
10. **Fills (C6).** (a) filled on touch at the anchor; (b) filled only if the
    touch bar trades ONE TICK THROUGH the anchor (short: high ≥ anchor+0.25;
    long: low ≤ anchor−0.25), entry at anchor; (c) filled on touch, entered at
    anchor ∓ 1 tick against you.
11. **Splits (C7).** Zone (volatility band) vs line (3.00-pt band) on the
    same anchor set; family count 1 vs 2 vs 3+ by DISTINCT families
    (`len(z.Families)`, uncapped — a 1d and a 4h swing at one price are ONE
    source, not two). Differences with Wilson intervals and n per group;
    round-21's kill condition applied as stated (interval includes 0 with
    upper bound below +4pp).
12. **Held out (C8).** In-sample = reads before 2025-09-12 00:00 CT; held-out
    = 2025-09-12 → 2026-09-11. Everything computed on both, nothing tuned on
    the held-out year.
13. **The surface (C9).** WidthK {0.25, 0.5, 1.0} × MergeATR {0.25, 0.5, 1.0}
    × FamilyCap {2, 3, 5} × StopMult {1.0, 1.5, 2.0} (expectancy) and the same
    27 maps × horizon {12@1m, 10@5m, 20@5m} (hold rate). The WHOLE surface is
    reported; cells are counted, the median/best/worst named. Multiple
    comparisons: Westfall–Young max-T via permutation (2,000 perms, sign flips
    for expectancy, label permutations for hold rates, in-sample only), plus
    the Bonferroni bound per cell. Held-out surface is a readout, never tuned.
14. **Clocks.** The backtest's clock is the bar's `open_time_ms`; read/flat
    times are CT wall times converted to unix ms once (`A28`).

## C1 — the population

3,084 reads built (1,760 skipped: sessions with no tape), 2,370 in-sample +
714 held-out. **11,302 first touches of seated (shortlisted) zones** — 8,639
in-sample, 2,663 held-out. Every cell below clears n≥30 except `tf=1h` (n=1,
descriptive only). All fill/trade columns are fill (a) at the live stop floor
1.5×ATR5m; expectancy is NET points after 2.0 pts friction; PF is on net points;
win rates are both shown (gross = exit before friction, net = after).

| group | n touches | decided | hold% | Wilson | n<30 | fillA n / win-net% / win-gross% / exp net pts / PF | fillB fill-rate / exp | fillC exp | MAE med | maxDD | streak |
|---|---|---|---|---|---|---|---|---|---|---|---|
| approach=above | 5430 | 5277 (amb 153) | 72.6% | [71.4, 73.8] |  | 5430 / win net 46.2% / win gross 92.0% / -0.04 / PF 0.98 | 24.6% / -4.61 | +0.21 | 0.0 | 1793.8 | 32 |
| approach=below | 5872 | 5700 (amb 172) | 73.9% | [72.7, 75.0] |  | 5872 / win net 47.4% / win gross 87.9% / -1.49 / PF 0.62 | 38.3% / -5.11 | -1.24 | 0.0 | 8875.9 | 19 |

| held_out | 2663 | 2607 (amb 56) | 71.3% | [69.6, 73.0] |  | 2663 / win net 57.8% / win gross 89.0% / -0.61 / PF 0.86 | 32.9% / -6.54 | -0.36 | 0.0 | 1870.7 | 25 |
| in_sample | 8639 | 8370 (amb 269) | 73.8% | [72.9, 74.8] |  | 8639 / win net 43.4% / win gross 90.1% / -0.84 / PF 0.70 | 31.4% / -4.40 | -0.59 | 0.0 | 7451.9 | 22 |

| families=1 | 3327 | 3230 (amb 97) | 73.9% | [72.4, 75.4] |  | 3327 / win net 36.1% / win gross 89.8% / -1.63 / PF 0.44 | 29.5% / -5.13 | -1.38 | 0.0 | 5508.3 | 23 |
| families=2 | 5922 | 5753 (amb 169) | 72.7% | [71.5, 73.8] |  | 5922 / win net 47.7% / win gross 89.9% / -0.71 / PF 0.77 | 32.6% / -4.59 | -0.46 | 0.0 | 4560.2 | 17 |
| families=3+ | 2053 | 1994 (amb 59) | 73.9% | [71.9, 75.8] |  | 2053 / win net 61.7% / win gross 89.8% / +0.33 / PF 1.09 | 32.8% / -5.60 | +0.58 | 0.0 | 527.4 | 13 |

| session=ASIA | 3708 | 3428 (amb 280) | 82.8% | [81.5, 84.0] |  | 3708 / win net 47.7% / win gross 93.5% / -0.24 / PF 0.91 | 21.3% / -5.41 | +0.01 | 0.0 | 1384.0 | 23 |
| session=LONDON | 3981 | 3943 (amb 38) | 73.3% | [71.9, 74.7] |  | 3981 / win net 36.8% / win gross 92.7% / -0.61 / PF 0.69 | 26.0% / -3.02 | -0.36 | 0.0 | 3402.2 | 36 |
| session=NY | 3613 | 3606 (amb 7) | 64.1% | [62.5, 65.6] |  | 3613 / win net 57.0% / win gross 83.1% / -1.56 / PF 0.70 | 48.7% / -5.83 | -1.31 | -3.8 | 5906.6 | 16 |

| tf= | 10112 | 9809 (amb 303) | 73.8% | [72.9, 74.6] |  | 10112 / win net 45.6% / win gross 90.2% / -0.83 / PF 0.74 | 31.5% / -4.91 | -0.58 | 0.0 | 8770.3 | 24 |
| tf=15m | 551 | 548 (amb 3) | 70.4% | [66.5, 74.1] |  | 551 / win net 52.8% / win gross 91.5% / -0.47 / PF 0.83 | 31.6% / -3.82 | -0.22 | 0.0 | 372.0 | 7 |
| tf=1h | 1 | 1 (amb 0) | 0.0% | [-0.0, 79.3] | ⚠️ | 1 / win net 100.0% / win gross 100.0% / +3.38 / PF 0.00 | 0.0% / +0.00 | +3.62 | 0.0 | 0.0 | 0 |
| tf=5m | 638 | 619 (amb 19) | 67.9% | [64.1, 71.4] |  | 638 / win net 60.0% / win gross 83.5% / -0.36 / PF 0.91 | 35.6% / -6.05 | -0.11 | -0.8 | 552.5 | 7 |

| width=0.25–0.5×ATR | 224 | 222 (amb 2) | 61.3% | [54.7, 67.4] |  | 224 / win net 61.2% / win gross 77.7% / +0.15 / PF 1.02 | 59.4% / -2.50 | +0.40 | -4.6 | 285.8 | 7 |
| width=0.5–0.75×ATR | 1284 | 1268 (amb 16) | 68.4% | [65.8, 70.9] |  | 1284 / win net 62.9% / win gross 84.2% / -0.51 / PF 0.91 | 44.3% / -4.54 | -0.26 | -1.9 | 817.1 | 8 |
| width=0.75–1.0×ATR | 9662 | 9355 (amb 307) | 74.5% | [73.7, 75.4] |  | 9662 / win net 44.2% / win gross 91.2% / -0.81 / PF 0.70 | 28.8% / -5.13 | -0.56 | 0.0 | 8146.9 | 30 |
| width=<0.25×ATR | 132 | 132 (amb 0) | 48.5% | [40.1, 56.9] |  | 132 / win net 59.1% / win gross 66.7% / -3.30 / PF 0.69 | 80.3% / -4.67 | -3.05 | -14.5 | 691.6 | 6 |

| year=2022 | 1960 | 1900 (amb 60) | 73.8% | [71.8, 75.7] |  | 1960 / win net 50.3% / win gross 90.7% / -0.68 / PF 0.77 | 32.3% / -4.67 | -0.43 | 0.0 | 1548.3 | 16 |
| year=2023 | 2540 | 2451 (amb 89) | 74.6% | [72.9, 76.3] |  | 2540 / win net 34.1% / win gross 90.0% / -1.21 / PF 0.49 | 30.6% / -3.48 | -0.96 | 0.0 | 3072.4 | 22 |
| year=2024 | 2478 | 2407 (amb 71) | 74.8% | [73.1, 76.5] |  | 2478 / win net 41.0% / win gross 90.8% / -0.86 / PF 0.69 | 29.5% / -4.27 | -0.61 | 0.0 | 2324.3 | 19 |
| year=2025 | 2401 | 2342 (amb 59) | 70.5% | [68.6, 72.3] |  | 2401 / win net 52.0% / win gross 88.4% / -0.72 / PF 0.80 | 34.4% / -5.58 | -0.47 | 0.0 | 2538.0 | 25 |
| year=2026 | 1923 | 1877 (amb 46) | 72.4% | [70.3, 74.4] |  | 1923 / win net 61.0% / win gross 89.5% / -0.33 / PF 0.93 | 32.0% / -6.92 | -0.08 | 0.0 | 1485.1 | 11 |

Sensitivity — the full map (every complete-width zone, seated or not):
279,110 touches, zone-band hold 80.7% (n=260,282 decided), fillA −1.17 pts net
(PF 0.51), fillB fill-rate 19.3% at −5.95 pts, fillC −0.92. Same sign everywhere.
## C3 — the baseline vs the live 48.1%

The live detector VERBATIM (`kernel.DetectTouchOutcomes`, k=3.0, H=12×1m,
exit_on=close) on the historical first touches of seated zone anchors:
**p(hold)=50.58%, Wilson [49.5, 51.7], n=8,232 decided (3,070 ambiguous
excluded)** — a coin flip, as the detector's IID calibration says (0.5067).
13f's corrected live figure is 48.09%, n=892, [44.8, 51.4]. The long history
**agrees**: the historical point estimate sits inside the live interval's
neighbourhood and the intervals overlap; the live sample is the noisier of the
two. The all-zones population says the same thing (50.85%, n=189,359). The
"zones hold ~73%" number the map reports is the ZONE-BAND outcome below — the
band is wider than the detector's barriers, so a hold is mechanically easier.
That uplift does not pay (C4).
## C4 — the trade, net, three fills side by side

**The anatomy** (seated zones, fill a, stop floor 1.5×ATR5m): the fade wins
small and loses big, by construction of the target rule. The "first opposing
zone" is on average **4.35 pts** away; the composed stop is on average
**23.52 pts** away (1.5×ATR5m ≈ 23 pts on this tape, wider when a nearer
risk-side zone anchors it). That is a ~1:0.18 reward:risk trade: **89.9% gross
win rate** (10,157 of 11,302 exit at target), avg gross win **+4.12 pts**, avg
gross loss **−24.57 pts** (1,145 stops + 18 flats). Gross expectancy +1.21 pts;
2.0 pts round-turn friction flips it to **−0.79 pts net** (−$1.58/trade, PF
0.75, avg R-multiple 0.047, max drawdown 9,290 pts cumulative, longest losing
streak 25). Net win rate 46.8%.

Hold rates do not pay here: 73.3% of touches HOLD by the band outcome, but only
46.8% of the resulting trades are net winners — a hold is a 12-bar rejection,
not a completed trade against the stop/target geometry.

| fill assumption | n trades | net win rate | expectancy pts net | profit factor |
|---|---|---|---|---|
| (a) filled on touch, entry at anchor | 11,302 | 46.8% | **−0.79** | 0.75 |
| (b) filled only on a 1-tick-through print | 3,586 (fill rate 31.7%) | 41.2% | **−4.93** | 0.33 |
| (c) filled on touch, entry ±1 tick against | 11,302 | — | **−0.54** | — |

Era split, fill (a): in-sample −0.84 (PF 0.70, win-net 43.4%); held-out −0.62
(PF 0.86, win-net 57.8%). Fill (b): in-sample −4.40, held-out −6.54. The loss is
not an artifact of any one year (C1 table: every year negative, −0.33 to −1.21).

Outcome horizons (C2): zone-band hold 73.3% at H=12×1m; 65.2% at H=10×5m
(n=10,573) and 65.0% at H=20×5m (n=10,670) — the same horizon erosion 13f
measured. Ambiguous excluded from every rate and counted separately
(H12: 325; H10: 131; H20: 34).
## C7 — round 21's question, answered in the same pass

**ZONE vs LINE.** Same anchors, two presentations: the volatility-scaled zone
band vs the pre-map bare 3.00-pt tolerance (±1.5). The zone band does NOT just
repackage the same touches — it changes all three measured quantities:

| quantity | zone (vol band) | line (3.00-pt band) | difference [CI] |
|---|---|---|---|
| hold rate (H12) | 73.25% [72.4, 74.1] n=10,977 | 59.33% [58.4, 60.2] n=11,005 | **+13.9pp [12.7, 15.2]** |
| fill-b rate (through-tick) | 31.7% | 65.7% | −34.0pp |
| expectancy pts net (fill a) | −0.79 | −3.07 | +2.28 pts |

Round-21's kill condition (interval includes 0 with upper bound < +4pp) does
**not** trigger for zone-vs-line: the zone presentation measurably changes the
hold rate (+13.9pp) AND the expectancy (+2.3 pts/trade) — but both expectancy
values are negative, so the zone is a less-bad fade anchor, not a profitable
one.

**MULTI-SOURCE vs SINGLE** (distinct families, uncapped: 1 vs 2 vs 3+). The
null for round 21 is no difference in hold:

| families | n touches | decided | hold rate [Wilson] | fillA exp pts net | era split (in / out) |
|---|---|---|---|---|---|
| 1 | 3,327 | 3,230 | 73.9% [72.4, 75.4] | −1.63 | −1.60 / −1.77 |
| 2 | 5,922 | 5,753 | 72.7% [71.5, 73.8] | −0.71 | −0.75 / −0.57 |
| 3+ | 2,053 | 1,994 | 73.9% [71.9, 75.8] | **+0.33** | +0.34 / **+0.32** |

fam3+ vs fam1 hold-rate difference: **−0.03pp [−2.5, +2.4], n=1,994 vs 3,230**.
The interval includes 0 with an upper bound below +4pp — by round 21's stated
threshold, **multi-source confluence does not improve the hold rate** (the
feature does not clear the bar on the question it was asked).

But the same split moves EXPECTANCY, not via holds: fam3+ is +0.34 pts net in
sample AND +0.32 held out (PF 1.11 / 1.06) — the only net-positive cell in the
entire study, at n=2,053 trades, on a split that was fixed by the dispatch (not
swept, not tuned). It is reported as measured; no recommendation follows (D6).
The direction of the effect is the trade's geometry (fam3+ zones sit at real
multi-family confluence, so the nearest opposing target is farther / the stop
closer), not the hold rate.
## C8 — held out

In-sample (2022-04-11 → 2025-09-11): 8,639 touches, fillA **−0.84** pts net
(PF 0.70). Held-out (2025-09-12 → 2026-09-11): 2,663 touches, fillA **−0.62**
pts net (PF 0.86). Both negative; nothing was tuned on the held-out year; the
held-out year's win rate (57.8% net) is higher than in-sample's (43.4%) and its
expectancy is still negative. The fam3+ cell's held-out readout is stated in C7.
Per-session held-out split: ASIA −0.46, LONDON +0.84, NY −2.38 — no session is
reliably positive across eras (in-sample: ASIA −0.17, LONDON −1.05, NY −1.31).

## C9 — the whole surface

Grid: WidthK {0.25, 0.5, 1.0} × MergeATR {0.25, 0.5, 1.0} × FamilyCap {2, 3, 5}
× StopMult {1.0, 1.5, 2.0} = 81 expectancy cells; the same 27 maps × horizon
{12@1m, 10@5m, 20@5m} = 81 hold-rate cells. Corrected with Westfall–Young max-T
(2,000 permutations, in-sample) and Bonferroni per cell; the held-out surface is
a readout only.

**Expectancy surface (fill a, net pts): 81/81 cells negative.** Median −1.07,
best **−0.64** (k=0.5, m=1.0, cap=2, stopMult=2.0, n=8,254), worst −1.76
(k=1.0, m=0.25, cap=2, stopMult=1.0, n=8,125). Every cell's loss is significant
against the sign-flip null (best-cell max-T p<0.001, Bonferroni p<0.001). No
configuration of the swept knobs makes the fade pay — the 900,000-trade
random-entry study's shape, reproduced: a single good cell would be noise; here
there is not even one.

**Hold-rate surface: 81/81 cells above the calibrated coin flip.** Zone-band
hold rates 63.4%–74.6% across every swept map and horizon, all far above the
D1′ IID calibration 0.5067 (min max-T p=0.0015 vs pooled). The zone band
mechanically inflates "hold" everywhere, and C4 shows the inflation does not
pay. A hold rate is not an edge — this table is why.

## C10 — the seam check

`seam.json` records the assertions, all PASS:

- **D5 match**: the DB copy's `source='historical_import'` census equals the
  wave-101 `import-results.jsonl` totals per (contract, tf) — the only delta is
  the standalone MNQ 06-22 import (66,450 rows, run before the jsonl loop), and
  it is exactly 66,450. `D5 matches=true`.
- **Single-contract series**: every loaded series carries only its own
  contract's rows (asserted in code over all loaded rows). Every read's window
  is fetched from ONE contract's series — the harness has no code path that
  concatenates two contracts.
- **No window spans a roll**: the contract is re-resolved per read instant via
  `ContractAt` semantics (newest 1m bar ≤ read time); session windows stay on
  that contract, exactly as the live ring does across a roll day.

## C10 assertion, quoted

> assert: every bar of every window was fetched from ONE contract's series; a
> window is built by lastClosed(contract, tf, n, readTime) and
> sessionBars1m(contract, …) which read only that contract's rows — no window
> ever concatenates two contracts' bars.

## Limitations — stated plainly

1. **The fill is the honest core** (round 10). A resting limit at the anchor
   is assumed filled on touch under (a); (b) requires a through-tick print and
   (c) prices adverse slippage. If the result survives only under (a), that is
   the first sentence of this report (see headline).
2. **This backtest cannot know that the zones it marks are the zones the live
   planner would have marked.** The planner is an LLM; its scenario choice,
   direction judgement and authored stops/targets are not replayed. What IS
   replayed is the machine's zone map at each read instant with the live
   detection chain, and the fade is the DETERMINISTIC rule the dispatch
   specifies on top of it. The result measures the rule, not the model.
3. **A backtest's fills are assumed** — (b) and (c) bound how much that
   assumption costs, but no assumption reproduces queue position, size, or
   adverse selection magnitude (round 10's ~66%).
4. **Era fidelity:** store-fed nPOC rows and owner levels exist only in the
   recent store era and are fed/omitted as stated in methods; the 2022–2025
   tape has no `session_profiles`/owner rows to feed — the live process in
   those years is reconstructed from its bars alone.
5. **Session enablement:** the live registry has ASIA/LONDON disabled; the
   dispatch fixes all three read times, so all three are replayed. Killzone
   and other runtime gates are NOT replayed (touches are evaluated, not
   gated).
6. **Entry/exit price realism:** intrabar stop-first on double touches is the
   conservative choice; exits fill exactly at stop/target prices (no
   stop-slippage beyond C5/C6; state it as an assumption).
7. **The prior.** Mesfin found nothing on MNQ survived 2 points. If this
   backtest finds an edge, the burden is on showing it is not the same
   overfit in different clothing — which is why the surface is reported whole
   (C9) and the last year held out (C8).

## Surprises (A23 — included, not acted on)

1. **The zone inflates the hold rate and does not change the verdict.** Bare
   anchor: 50.6% (coin flip). Zone band: 73.3%. Expectancy negative either way.
   The headline "levels hold ~73%" is a band-width artifact — 13f's lesson at
   population scale.
2. **fam3+ is the only net-positive cell in the study** — +0.34 in-sample AND
   +0.32 held-out — while its hold rate is identical to fam1's. The
   multi-source effect (if real) lives in the trade geometry, not the holds.
   Reported, not recommended.
3. **Adverse selection is measurable**: fills that require a through-tick print
   lose −4.4 to −6.5 pts/trade vs −0.79 for touch fills (round 10, quantified).
4. **The trade's shape, not the levels**: 89.9% of fade trades exit at target
   because the first opposing zone averages 4.35 pts away, against a ~23.5-pt
   stop. The fade as specified is a 1:0.18 R:R lottery that friction turns
   negative.
5. **Session asymmetry**: NY is the worst session (−1.56 pts/trade, in-sample
   and held-out), ASIA the least bad (−0.24). Short fades (approach from below)
   lose −1.49; long fades (approach from above) −0.04.
6. **A hold is not a win**: 73.3% hold but 46.8% net win rate under the
   stop/target geometry.
7. **Days are labelled by CME session-day** (`CMESessionDayKey` = the 17:00 CT
   roll date), so a Monday 01:30 CT read carries Sunday's date — convention,
   stated, not a bug.

## Artifacts

- `harness/` — the replay harness + extraction scripts (committed; nothing in
  it ships; it imports only `nofx/kernel`, `nofx/market` and a read-only SQLite
  view of the DB COPY).
- `artifacts/` — the small aggregates this report quotes (`seam.json`,
  `summary.json`, `c1c4_spine{,_all_zones}.json`, `c3_baseline.json`,
  `c7_splits{,_all_zones}.json`, `c8_era_splits{,_all_zones}.json`,
  `c9_surface_{expectancy,holdrate}.csv`, `c9_surface_summary.json`).
  Worktree-only (gitignored): `backtest.db` (the 1.2 GB DB COPY, md5
  `62eb9e5929d99f992f70a67a7ae4ac68`), `run/` (`events.jsonl` — every event
  with its id and every field; `run.log`). Regenerate the worktree artifacts
  with the harness command in Methods.
