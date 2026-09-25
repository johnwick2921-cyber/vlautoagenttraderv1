# Cheap Five — Knob Verdict Tables from Stored Data — 2026-08-30

Read-only dispatch (isolated worktree `nofx-cheap5` @ `79365622`, branch `docs/cheap-five`).
DB accessed as `sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro"` — zero writes.
Evidence tiers: **[A]** directly measured from DB/journal · **[B]** reconstructed with stated assumptions · **[C]** approximation.

Knob definitions per the provenance census (`docs/superpowers/reports/2026-08-30-knob-census.md`, commit `39a0481e`, branch `docs/knob-census`):
proximity `proximity_filter_atr=0.3` [O] (×DailyRangeProxy) · `FAST_MARKET_ATR=1.5` [I] (×ATR5m) ·
`LevelClusterTicks=12 = 3.0pt` [I] · `DefaultSideQuota=2` [C] (old rule 3) · killzones [O]:
asia 19:00→23:00, london 02:00→05:00, ny_am 08:30→11:00, ny_pm 13:00→14:45 CT.

---

## Executive summary (one line per knob)

| Knob | n | Verdict |
|------|---|---------|
| PROXIMITY 0.3 | 101 seated rows / 4 session-days | **MORE DATA — candidate pool never persisted; seat evidence doesn't indict 0.3 (70.3% touch rate)** |
| FAST_MARKET_ATR 1.5 | 4 fires (1 session-day) | **KEEP — tripwire fires correctly; wake cadence, not the multiplier, is the bottleneck (fired after the move completed)** |
| CLUSTER_TOL 3pt | 157 plans / 41 merge-pairs | **KEEP — zero grade-straddling merges; 80% of pairs ≤1.5pt anyway** |
| SIDE-QUOTA 2 vs 3 | 154 priced plans (106 before / 48 after) | **KEEP 2 — thin-map rate fell 30.2%→22.9%; the model is not quota-bound** |
| KILLZONE PREMIUM | 567 positions (285 in / 282 out) | **NO PREMIUM — current-era in-killzone avg −$14.5/fill vs −$0.5 out; historical "premium" is 100% pre-era** |

---

## 1. PROXIMITY 0.3 — is the band excluding actionable levels?

**Definition.** `proximity_filter_atr = 0.3` × DailyRangeProxy filters the candidate pool to a band
(±0.3×prior-day-range) around the write price before seating. "Excluded" = candidate level outside
the band that session; "seated" = level in the plan / `level_stats`.

**What is stored.** `level_stats` has 101 rows for exactly 4 session-days (2026-08-24 → 08-27;
`session_day` maps 1:1 to `plans.trade_date`, verified by full price/label overlap). The raw
excluded universe (machine candidate pool) is **not persisted anywhere** — planner prompts are not
stored in `decision_records` (executor cycles only). Proxy used for "excluded": levels seated in an
earlier plan version of the same session-day and **dropped** from the next version (n = 349), then
checked for a ±4pt touch after the drop in that session's window (bars) — same ±4pt band the
nightly `EvaluateLevelOutcome` uses.

**Table 1a — seated-but-never-touched (dead seats) [A]:**

| Session-day | Seated | Touched | Never-touched | Touch rate |
|-------------|-------:|--------:|--------------:|-----------:|
| 2026-08-24 | 28 | 13 | 15 | 46.4% |
| 2026-08-25 | 28 | 23 | 5 | 82.1% |
| 2026-08-26 | 18 | 10 | 8 | 55.6% |
| 2026-08-27 | 27 | 25 | 2 | 92.6% |
| **Total** | **101** | **71** | **30 (29.7%)** | **70.3%** |

Reacted (touch + ≥8pt continuation in 3 bars): 56/101 = **55.4%**.
Dead-seat examples (08-26): VWAP−1σ 29182.84, OR-L 29174.25, PDC 29228.50, RTH-L 29145.50, PDL 29095.50, ONH 29310.00 — all `touched=0`.

**Table 1b — excluded-proxy: version-dropped levels later touched [B]:**

| Session-day | Dropped (proxy) | Dropped→touched ±4pt | Rate |
|-------------|----------------:|---------------------:|-----:|
| 2026-08-24 | 48 | 23 | 47.9% |
| 2026-08-25 | 60 | 20 | 33.3% |
| 2026-08-26 | 105 | 66 | 62.9% |
| 2026-08-27 | 136 | 83 | 61.0% |
| **Total** | **349** | **192** | **55.0%** |

Examples of dropped-then-touched: RN 29600 (08-26 19:10), VWAP 29581.44 (19:13), ONL 29464.00 (20:15),
PDH 29283.50 (08-25 18:09), OB(bear)·4h 29154.38 (19:36) — all touched the same session after being dropped.

**Band sanity [B/C].** Reconstructed band = 0.3 × prior session-day range (17:00→17:00 CT; measured
532.8 / 403.3 / 341.8 / … pt — **wider** than the census's ~283–305pt proxy, so this is generous).
362/1139 (31.8%) of seated plan-level rows sit outside the reconstructed band of their plan's write
price — driven by HTF seats (1h/4h promotion), VWAP drift and version churn, not by band admission
changes (the band is constant within a session-day, so intra-day drops are model/price-drift driven).

**Verdict: MORE DATA — the question as posed cannot be answered from stored data.** The excluded
universe is never persisted, so "0.3-too-tight" cannot be measured directly. The observable
evidence does not indict 0.3: seats touch 70.3% of the time and only 29.7% are dead seats. The
55% dropped-then-touched proxy is dominated by model re-prioritization (16 ASIA versions on 08-25,
105 drops on 08-26) under a constant band, not by band exclusion. Cheapest real test remains the
census's #6: persist the pool (or replay `ScoreLevelsMinGradeFull` on stored bars) and diff
in-band vs out-of-band levels against touches.

---

## 2. FAST_MARKET_ATR 1.5 — does the fast-market seam fire early enough to matter?

**Definition.** Wake reads with drift ≥ `1.5 × ATR5m` downgrade planner reasoning to fast→low
(F3 seam, `trader/auto_trader_loop.go:86`). Journal coverage: **only from Aug 27 13:36 CT**
(08-26 wiped by the order_update flood suppression — retention gap, stated).

**Table 2 — every fast-market fire in the retained journal [A] (all four are today, 08-30):**

| Fire (CT) | Engine drift | ×ATR5m | Implied threshold (1.5×ATR5m) | Outcome |
|-----------|-------------:|-------:|------------------------------:|---------|
| 19:07:45 | 94.5 pt | 3.5× | 40.5 pt | 3 attempts rejected → wake failed (benign, plan kept). **0 writes.** |
| 20:10:01 | 68.0 pt | 2.4× | 42.5 pt | 3 attempts rejected → wake failed (benign). **0 writes.** |
| 20:40:01 | 54.8 pt | 1.9× | 43.3 pt | 3 attempts rejected → wake failed (benign). **0 writes.** |
| 21:40:01 | 88.5 pt | 3.3× | 40.2 pt | 3 attempts rejected → wake failed (benign). **0 writes.** |

`plans` confirms: 08-30 ASIA has exactly 2 versions, both written before 19:07 — every fast-market
wake today produced **rejections, zero plan writes**.

**Today's three fast-read rejections (the 19:07:45 fire's chain), verbatim [A]:**

> 08-30 19:12:42 — 📐 planner attempt 1/3 rejected: S1 breakdown_continue: a close came back across 29292.25 — the breakdown is void; author a reject/retest play instead
>
> 08-30 19:15:17 — 📐 planner attempt 2/3 rejected: S1 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (27.2 pts) — not a displacement move, author a normal reject/retest play instead
>
> 08-30 19:18:45 — 📐 planner attempt 3/3 parse/schema rejected: flip{price 29251.28} does not match any number in bias.flip_condition prose "5m close below 29273.50 ONL flips short"

All three are correct validator actions (the tape had already reclaimed), not fail-closed stubs.

**Timing vs the move [A].** ASIA v2 written 18:35:34 CT at 29372.00 (bars). Bar-close drift first
crossed 1.5× (≈29.7–40.5pt depending on ATR) at **18:45:00** (41.25pt). The move's low printed
**29291.25 @18:56** and the session low **29273.50 @19:01**. The first fast fire was **19:07:45 —
6¾ min after the move completed and ~22 min after the threshold first crossed**, because drift is
only checked at wake reads (today's reads: 18:43, 19:07:45, 19:45, 20:10, 20:40, 21:20, 21:40 —
20–40 min apart, event-driven), and each attempt adds ~5 min of model latency (fire 19:07:45 →
attempt 19:12:42). Non-fast reads, inferred bar-close drift vs 29372.00: 18:43 ≈ 17.8pt, 19:45 ≈
2.8pt, 21:20 ≈ 44.8pt (engine judged non-fast at 21:20 — its ref/ATR differ from the bar-close
proxy). Wake re-read failure counts: 08-27 ×3, 08-28 ×3, 08-30 ×7.

**Verdict: KEEP 1.5.** The tripwire itself fires and the reasoning downgrade works as designed; on
the one big move it did **not** fire early enough to matter — but the lateness comes from the
wake/read cadence and ~5-min attempt latency, not from the multiplier. Lowering 1.5 would only add
more fast reads that still arrive after the fact (and every fast read today was rejected on
merits). The multiplier's own calibration (n=4 fires, all one session) = MORE DATA.

---

## 3. CLUSTER_TOL 3pt — how tight are real near-duplicate pairs?

**Definition.** `LevelClusterTicks = 12` = 3.00pt: levels within tolerance of a stronger survivor
collapse to one seat (survivor = higher score → today-priority kind → nearer → lower price; zones
are exempt). Measured here on **all 161 non-WEEKLY plan docs** (157 with ≥2 levels): pairwise
`|price_i − price_j| ≤ 3.0` between seated levels — note these are post-collapse survivors, so
the histogram **undercounts** pre-merge density.

**Table 3 — merged-pair distance histogram [A]:**

| Distance bin | Pairs | % |
|--------------|------:|--:|
| (0, 0.5] | 4 | 9.8% |
| (0.5, 1.0] | 17 | 41.5% |
| (1.0, 1.5] | 12 | 29.3% |
| (1.5, 2.0] | 2 | 4.9% |
| (2.0, 2.5] | 3 | 7.3% |
| (2.5, 3.0] | 3 | 7.3% |
| **Total ≤3.0** | **41** | — |

- Pairs that would merge: 41 across 157 plans (≈0.26 pairs/plan).
- **Grade-straddling merges (A+B → A): 0.** Machine-grade-straddling pairs: **0.** All 41 pairs
  are same-grade/same-machine-grade references; 13 are same-label (e.g. EQH 30218.0 + EQH 30219.5
  @1.5pt, PDH 30148.25 + PDC 30147.50 @0.75pt).
- 33/41 (80.5%) are already ≤1.5pt — the tolerance's effective bite is in the 1.5–3.0pt tail (8 pairs).

**Verdict: KEEP 3pt.** The tolerance never crosses a grade boundary in stored plans, so widening
or narrowing only changes how many same-grade twins collapse. Halving to 1.5pt would leave 8
observed near-duplicate pairs standing for a noise-level gain. n=41 pairs is modest; no retune case.

---

## 4. SIDE-QUOTA 2 vs 3 — did relaxing MIN_SIDE_LEVELS change stored maps?

**Definition.** `DefaultSideQuota` 2 (old rule 3; the P0.1 ≥3-per-side hard rule born 08-18).
Per plan: levels strictly above vs below the **write-time price** — inferred from the closest 1m
MNQ bar close to `plans.created_at` (106 before / 48 after); 3 pre-bar plans used the nearest
`decision_records.input_prompt` `price = X` line within ±15 min. Split at 2026-08-26 per dispatch.
Caveats: the code 3→2 relax actually deployed **08-27 18:38 CT**, and the live DB knob is
`min_side_levels=4` (owner) — so "after" is quota-4-era stored maps with machine-caused-thin
tolerance, not pure quota-2.

**Table 4 — side-count distribution before vs after [B]:**

| Bucket | n (priced) | ≥3/≥3 | 2-2 | One-side ≤2 (thin) | Zero-side | One-sided-map rate |
|--------|-----------:|------:|----:|-------------------:|----------:|-------------------:|
| Before 08-26 | 106 | 74 (69.8%) | 0 | 32 | 4 | **30.2%** |
| After 08-26 | 48 | 37 (77.1%) | 0 | 11 | 1 | **22.9%** |

- **No 2-2 plan exists in either era** — the model seats 8–12 levels and lands on odd splits.
- Zero-side maps: 4 before (3.8%) → 1 after (2.1%); the single post-relax zero-side
  (`2026-08-27 08:07:37` LONDON, 0 above / 8 below) was **accepted** — the machine-caused-thin
  tolerance working as designed.
- Post-relax thin plans (n=11) persist via the same tolerance, e.g. `(6,2)`, `(0,8)`, `(11,1)`.

**Verdict: KEEP 2 (code default).** Quota 3→2 changes almost nothing observable in stored plans —
~70–77% clear ≥3 per side regardless, thin maps still occur at ~23% post-relax (machine-caused,
and correctly tolerated). The quota was never the binding constraint on map shape; the model's own
level selection is. The real fix that mattered was the machine-caused-thin tolerance, which
prevented the 08-27-style full-session fail-closed.

---

## 5. KILLZONE PREMIUM WINDOW — is there an edge inside the windows?

**Definition.** Killzone windows (census [O], `session_registry.go`): asia 19:00→23:00, london
02:00→05:00, ny_am 08:30→11:00, ny_pm 13:00→14:45 CT. A fill is in-killzone if its entry time
(CT) falls in any window. `calendar_static_t1.json` contains **zero T1 windows inside fill history**
(all September+) — T1-static classification is vacuous; session killzones are the operative set.
Entry times from `trader_positions.entry_time` (fills table has no entry_time); PnL =
`COALESCE(pnl_corrected, realized_pnl)` (pnl_corrected null for 354/567 rows). All 567 positions
are MNQ. id 520 (demo_seed) is absent from the table.

**Table 5a — in vs out of killzone, all history [A]:**

| Bucket | n | Wins | Win rate | Σ PnL | Avg / fill |
|--------|--:|-----:|---------:|------:|-----------:|
| In killzone | 285 | 94 | 33.0% | −$123.77 | −$0.43 |
| Out killzone | 282 | 93 | 33.0% | −$395.50 | −$1.40 |

**Table 5b — by window [A]:**

| Window (CT) | n | Win rate | Σ PnL | Avg / fill |
|-------------|--:|---------:|------:|-----------:|
| ASIA 19:00–23:00 | 112 | 35.7% | **+$2,278.27** | +$20.34 |
| LONDON 02:00–05:00 | 72 | 31.9% | −$499.54 | −$6.94 |
| NY AM 08:30–11:00 | 71 | 26.8% | −$1,820.50 | −$25.64 |
| NY PM 13:00–14:45 | 30 | 40.0% | −$82.00 | −$2.73 |

**Table 5c — era split (planner era = since 08-19) [A]:**

| Era | In-kz n / Σ / avg | Out-kz n / Σ / avg |
|-----|-------------------|--------------------|
| Since 08-19 | 28 / −$406.43 / **−$14.52** | 23 / −$12.00 / −$0.52 |
| Before 08-19 | 257 / +$282.66 / +$1.10 | 259 / −$383.50 / −$1.48 |

All 112 ASIA-killzone fills are **pre-08-19** (0 since) — the entire +$2,278 "ASIA premium" comes
from the old trading era.

**Verdict: NO PREMIUM — the killzone hypothesis is unsupported by the fill record.** Win rates are
identical in vs out (33.0%), and in the current era killzone entries are the *worse* bucket (−$14.5
vs −$0.5 per fill, n=28/23). The only positive window (ASIA 19:00–23:00) is a pre-planner-era
artifact. Keep the windows as the [O] owner contract, but treat the "premium window" advisory
weighting as unproven — the premium is in window identity, not killzone-ness.

---

## Data gaps (what blocked or weakened each table)

1. **PROXIMITY:** the excluded universe (machine candidate pool with distances) is never persisted —
   planner reads leave no trace in `decision_records` (executor cycles only). `level_stats` covers
   only 4 session-days (08-24→27; 08-28/30 not yet evaluated by the nightly). 08-29 has no plans
   (Saturday). Band reconstruction uses a 17:00→17:00 CT prior-day range that is wider than the
   engine's ~283–305pt proxy, so the 31.8% out-of-band seated figure is approximate. The drop-churn
   proxy conflates band exclusion, VWAP re-computation and model re-prioritization.
2. **FAST_MARKET_ATR:** journald retention starts 08-27 13:36 CT (08-26 wiped by the order_update
   flood suppression). Non-fast read drift is inferred from bar closes vs the v2 write price; the
   engine's exact drift ref/ATR differ (21:20 read: bar-close 44.8pt yet engine judged non-fast).
   n=4 fires, all in one session.
3. **CLUSTER_TOL:** plan-doc pairs are post-collapse survivors — the histogram undercounts raw
   pre-merge near-duplicate density (the scorer's collapse happens before seating). Machine-grade
   straddle counts are only meaningful where machine grades differ, which was never observed.
4. **SIDE-QUOTA:** 3 pre-bar plans needed the `input_prompt` price fallback; the 08-26 split date
   ≠ the actual code deploy (08-27 18:38 CT); the live DB knob is 4 (owner), so "after" is
   quota-4 + machine-tolerance, not pure quota-2. No 2-2 plan exists in either era, so that cell
   is 0 by construction.
5. **KILLZONE:** `calendar_static_t1.json` has no T1 windows in fill history (all Sep+). Fills
   lack entry_time (positions used instead; 387/412 fills matched to entries via entry_order_id,
   the rest are exit fills). `pnl_corrected` is null for 354/567 rows (realized_pnl fallback).
   Sub-60s round-trips are known-missing from the ledger. Killzone windows are [O] static walls;
   no T1 dynamic slices could be tested.

## Evidence appendix (verbatim journal lines)

- `08-30 19:07:45 … 🧠 planner mode: fast-market (drift 94.5 pts = 3.5×ATR5m) — reasoning downgraded to fast→low for this read (F3)` (also 20:10:01 68.0pt=2.4×, 20:40:01 54.8pt=1.9×, 21:40:01 88.5pt=3.3×)
- `08-30 19:12:42 … 📐 planner attempt 1/3 rejected: S1 breakdown_continue: a close came back across 29292.25 — the breakdown is void; author a reject/retest play instead`
- `08-30 19:15:17 … 📐 planner attempt 2/3 rejected: S1 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (27.2 pts) — not a displacement move, author a normal reject/retest play instead`
- `08-30 19:18:45 … 📐 planner attempt 3/3 parse/schema rejected: flip{price 29251.28} does not match any number in bias.flip_condition prose "5m close below 29273.50 ONL flips short"`
- `08-30 18:45:26 … 📐 planner attempt 1/3 parse/schema rejected: scenario[0].confirm.rule "1x5m_close" — sweep_leg1_requires_touch …` (first post-v2 wake, non-fast)
- `08-27 19:33:27 … 🗓️ wake re-read failed for 2026-08-27 ASIA (benign — active plan kept): only 3 levels above price 29652.50 but the machine table offered 10 — the plan must carry ≥4 on EACH side (AI dropped levels the map supplied)`
