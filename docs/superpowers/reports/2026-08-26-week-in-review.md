# REPORT B — WEEK IN REVIEW (2026-08-19 → 2026-08-26, pnl_corrected)

Read-only analysis. Fresh DB + log reads only. Deployed rev 57b60b60 (1h wave +
R2/R4 landed 08-26 00:18 CT). Companion: `2026-08-26-settings-census.md`.
Feeds Phase 6 (same-window calibration) — cite the implementation report
`2026-08-25-1h-wave-r2-r4-implementation.md` §7B where numbers are shared.

---

## 1. P&L LEDGER (per day, pnl_corrected)

| Day | n | wins | win% | Σ | best | worst |
|---|---|---|---|---|---|---|
| 08-19 | 7 | 3 | 42.9% | +268.6 | +273.5 | −88.0 |
| 08-20 | 9 | 4 | 44.4% | +219.5 | +311.0 | −147.5 |
| 08-21 | 8 | 1 | 12.5% | −464.5 | +30.5 | −98.0 |
| 08-23 | 1 | 0 | 0% | −29.5 | — | −29.5 |
| 08-24 | 8 | 1 | 12.5% | −232.0 | +169.5 | −139.5 |
| 08-25 | 8 | 3 | 37.5% | +40.5 | +168.0 | −78.5 |
| **WEEK** | **41** | **12** | **29.3%** | **−197.4** | +311.0 | −147.5 |

Per session: ASIA 15 · 13.3% · **−477.4** (38min avg hold) · LONDON 16 · 31.2% ·
−250.5 (48min) · NY 10 · 50.0% · **+530.5** (49min).

## 2. EXPECTANCY TABLES

**By condition** (total): reject 14 · 35.7% · **+625.5** · acceptance 2 · 0% ·
−157.4 · sweep_reclaim 3 · 0% · −192.0 · breakout_retest 3 · 33.3% · +47.5 ·
off-plan 4 · 25% · −189.5 · unknown 6 · 50% · +93.5. **The split is the story:**
NY reject 4 · 75% · +665.5 vs ASIA reject 7 · 14.3% · −80.5.

**By quality**: A 15 · 40% · **+360.1** · B 8 · 12.5% · −292.0 · C 8 · 12.5% ·
−169.5 · unlinked 10 · 40% · −96.0.

**By cited level kind+grade**: OR-L/A +229.1 (10) · PWL/A +169.5 · ONH/A **−131.0**
(14, 21.4%) · EQL/A −116.0 · PDC/A −78.5 · PDH/A −62.0 · ONL/A −60.5 · PDC/B −52.0.

**By CT hour**: 11:00 +443 (WW) · 09:00 +156.5 · 21:00 +93 · 13:00 +81 · 04:00
+168 vs **07:00 −171.5 (LLL)** · **10:00 −150 (LL)** · 08:00 −134.5 · 19:00/20:00
−118/−153.9.

## 3. MAE/MFE AUTOPSY

- Losers: **15 STOPPED-TOO-TIGHT** (MFE ≥ 0.5×SL before stop-out) vs **12
  WRONG-FROM-START** — e.g. 08-25 02:09 MFE +58.0 → −62.0; 08-24 09:55 MFE +58.5
  → −54.5; 08-21 03:54 MFE +42.2 → −84.5.
- Winners survived real heat: 08-20 09:41 MAE −61.5 → +311.0; 08-20 05:26
  MAE −43.8 → +168.0; 08-25 04:19 MAE −22.5 → +168.0.
- **Verdict: BOTH, but the stop problem dominates** — 15/27 losers printed a
  tradable MFE first. Entry selection is secondarily wrong (12 wrong-from-start +
  21.4% win on ONH anchors).

## 4. PLAN QUALITY WEEK (plans table, Aug 19+)

- ASIA: 12 scheduled reads · **10 planner_fail_closed** · 6 owner_reset ·
  5 level_event wakes · **1 replans_exhausted**.
- LONDON: 12 scheduled · 1 fail_closed.
- NY: 11 scheduled · 1 structure_mss wake · 5 owner_reset.
- Avg lifespan between plan versions: **70.4 min** (n=46).
- Fail-closed rows: 11 · replans_exhausted: 1 (all pre-W6-D; the trap is closed
  — wakes are budget-free since 93412ce8, timeouts raised to 600s).

## 5. GATE / VETO LEDGER (log lines, incl. retry echoes)

| Type | 08-19 | 08-20 | 08-21 | 08-22 | 08-23 | 08-24 | 08-25 | 08-26 |
|---|---|---|---|---|---|---|---|---|
| htf_veto | 0 | 0 | 52 | 18 | 22 | 63 | 84 | 5 |
| transition | 0 | 0 | 0 | 0 | 0 | 5 | 0 | 0 |
| executor dead-plan | 0 | 0 | 0 | 0 | 0 | 0 | 3 | 0 |
| stale_reeval | 5 | 6 | 0 | 0 | 1 | 4 | 3 | 0 |
| clock_skew | 0 | 1 | 0 | 0 | 0 | 1 | 0 | 0 |
| guardrail would-trip (soft) | 301 | 756 | 76 | 0 | 0 | 202 | 271 | 0 |

API `/api/risk/gate-blocks` shows the CURRENT session-day (post-22:00Z roll):
"no gate blocks recorded". HTF-veto lines run ~3 per refusal (error fed back on
retries). Entries that SHOULD have been refused under current config but
predate a gate: none material — the min-conf 60 entries that the 65-rule would
have refused are in §7B of the implementation report (08-24: 5/5 positions).

## 6. INCIDENT RECAP (one line each, fix commit, verified)

1. **v9 ASIA fail-closed (21:36 CT)** — DeepSeek 300s timeout ×2 on the
   post-reset read. Fix: env `AI_HTTP_TIMEOUT_SECONDS=600` + `AI_MAX_RETRIES=2`
   (no code); re-reset produced v10. VERIFIED: boot line 21:57:44 + v10/v11 active.
2. **Replans-exhausted session death (19:35 CT)** — wake re-plans consumed the
   whole budget. Fix `93412ce8` (W6-D): wakes unlimited/budget-free + G5
   consumed-at-birth removed. VERIFIED live: 00:04 CT wake → v11 active.
3. **Wake fail-closed killed healthy ASIA (17:29 CT)** — fix `cf5d9d96` (W6-C):
   wakes async + non-fatal. VERIFIED: 00:20:56 CT wake on new rev kept v11.
4. **Manual-trade reconcile gap (−87.50 invisible)** — fix `1a5708d3`
   (untracked NT8 position materialization). VERIFIED: test locks the recovery.
5. **C1 IDOR / C2 owner-levels / C3 fallback / C4 klines** — fixed
   `7ef01f5a`/`fc9b06a6`/`514e0e86`/`ad915e08`; re-verified mounted 2026-08-25.
6. **C12 cursor persistence broke replay** — REVERTED `fa546f28` (1-candle charts
   gone). VERIFIED.
7. **Grading bugs (zero-score unknown TF, stamp collision, phantom labels,
   flip-bias)** — `c1cf4fdb`, `a5f42da3`, `5f620cac`, `5e1390e6`. VERIFIED.

## 7. LEVEL-TABLE REALITY CHECK (5 biggest losers, seated map at entry)

Stop-vs-nearest-seated-level distance (tolerance 3.0 pts):

| Entry (UTC) | Side | P&L | Stop | nearest seated level | gap |
|---|---|---|---|---|---|
| 08-20 13:41 | LONG | −147.5 | 29370.00 | RTH-L 29375.75 (B·x32) | 5.75 |
| 08-24 14:09 | SHORT | −139.5 | 29038.50 | OR-L 29082.50 (A) | **44.0** |
| 08-21 10:04 | LONG | −98.0 | 29460.00 | RTH-H 29470.25 (flipped) | 10.25 |
| 08-20 08:26 | SHORT | −97.5 | 29552.00 | EQH 29557.50 (A) | 5.5 |
| 08-21 15:12 | SHORT | −88.5 | 29400.00 | OR-L 29357.75 (A) | **42.25** |

**Finding: 0 of 5 stops sat ON a seated level; 2 of 5 sat in dead zones 40+ pts
from any seated row.** Stops are chosen off-map (below/above intermediate
swings the top-12 didn't seat) — the market ran the stop through empty space.
Supports the volume-detector hypothesis: real turning points ≠ the seated map;
the table needs the levels price actually reacts to, or stops must be anchored
to seated rows (or widened per the too-tight autopsy).

## 8. SYNTHESIS — TOP 5 EVIDENCE-BACKED CHANGES (ranked by projected Σ delta)

1. **`min_scenario_quality = A`** (knob exists, Report A §2) — projected
   **Δ ≈ +461.5** retroactive (keep A-quality entries +360.1, avoid B+C −461.5).
2. **Disable ASIA** (`sessions[ASIA].enable=false`) — **Δ ≈ +477.4** (ASIA 13.3%
   win, −477.4; NY alone +530.5). OR keep ASIA but quality-gate it.
3. **`min_confidence = 65`** — **Δ ≈ +326.4** over the week (strongly positive
   recent days; negative 08-19/20).
4. **Stop anchoring + sizing** — 15/27 losers stopped-too-tight AND 5/5 biggest
   loser stops off-map: anchor stops to seated levels or widen 1.5× → unpriced
   retroactively, but largest behavioral EV.
5. **Guardrails master ON** — 1,606 would-trips ignored this week (daily-trades
   1039 + daily-loss 266); enabling `max_daily_trades=3` alone would have blocked
   the overtrading days.

**3 things working (don't touch):** NY `reject` setups (+665.5, 75%),
quality-A scenarios (+360.1, 40%), the 600s timeout + budget-free wake machinery
(ASIA self-healed at 00:04 CT).

**Still missing to settle the calibration queue:** swing-k sweep + MSS-FVG need
historical bar persistence (08-22 has zero decision rows, BarCache in-memory);
HTF_VETO_TF=4h needs 4h in structure_json; trail-mult deltas need trail config
persisted per position. All listed in the Phase 6 report §7.
