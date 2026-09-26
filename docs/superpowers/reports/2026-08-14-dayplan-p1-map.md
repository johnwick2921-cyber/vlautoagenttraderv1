# DAY-PLAN CAMPAIGN — P1 · THE MAP (checkpoint report, 4/8 items)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `d8e2f88c` (P0 head) → `9436ea79` · 5 commits
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**P1 is PARTIAL and pushed — the pure, deterministic MAP backbone is built (4 of
8 items).** Multi-day + intraday + round-number detectors, the confluence scorer
(graded TOP-8) with its KEY LEVELS renderer, and the regime block all landed with
fixture tests; the full suite + `-race` (kernel, store) are green; **goldens and
the frontend are byte-untouched** (every item is a new file) and the running bot
(rev 3624a2a4, PID 363618) is untouched. Checkpointed at a clean item boundary —
the four remaining items couple to the trader loop / live prompt / network and
belong to a fresh session (never start an item I can't finish).

## STEP 0 gate
PASS — HEAD `d8e2f88c` (P0 head) · tree clean · bot PID 363618 untouched · cycling.

## Items shipped

### P1.1 — session-tagged bars + multi-day extractor — `14adc47e`
`kernel/levels.go` (shared `DetectedLevel` + `LevelKind` consts + cached tz) and
`kernel/levels_multiday.go` `ExtractMultiDayLevels`. Pure: tags each bar by CME
session (P0.3 registry), groups by CT calendar day / futures session-day per the
contract's naming (PDH = calendar-day only). Derives PDH/PDL/PDC, RTH-H/L (prior
day's NY), AS/LDN/ON (current overnight, ON = composite), PW/PM. Only levels WITH
data are emitted — warms forward, no backfill. 4 tests incl. a 15-level fixture.

### P1.2 — round numbers + gap tracker + OR/IB — `e1cd5993`
`kernel/levels_intraday.go`: `RoundNumberLevels` (100/50/25 within 1.5×dATR,
strongest-granularity dedup), `GapLevels` (unfilled gaps ≥ minGapATR×atr, fill
target + fill-state, up & down), `OpeningRangeLevels` (OR first 5m + IB first 60m
+ 1.5×/2× extensions, nil pre-open). 6 tests.

### P1.5 — confluence scorer → graded TOP-8 — `3058e131`
`kernel/levels_score.go`: deterministic (LLM never sorts). type-evidence ×
freshness × confluence × HTF → A/B/C; day-trade lock (±1.5×dATR); today's
structural kinds get first seats; cap = max_levels (8); S/D+FVG/OB confluence-only
(capped C, excluded standalone). `RenderKeyLevelsBlock` → the prompt block; empty
→ "" (B9-style dormant). 3 tests (filter, consume, grades, cap+priority, order,
render).

### P1.6 — regime block (7 fields) — `15e55520`
`kernel/regime.go` `ComputeRegime`: trend_daily (px vs EMA200-daily), trend_1h (px
vs EMA50-1h), atr_regime (daily ATR14 percentile LOW/NORMAL/HIGH/EXTREME),
realized_vol_pct (5m RV √288-annualized; vs-20d when a baseline exists, else
WARMING), vix (no feed → unavailable), expected_range_pts (VIX-implied or ATR
cross-check), overnight_gap ×ATR. Every field degrades honestly. `Render()` = one
line. 4 tests.

### sample — end-to-end pipeline block — `9436ea79`
`kernel/dayplan_sample_test.go`: assembles detectors → scorer → renderer on a
realistic MNQ fixture.

## SAMPLE KEY LEVELS block (real pipeline output — the exact text P1.7 will inject)
Fixture: price 15600, dATR 120; multi-day + round + OR/IB levels; scored, cap 8:
```
KEY LEVELS (map, nearest-first; price 15600.00):
  15600.00  PDC            A  fresh·x3  +0.0
  15590.00  OR-L           A  fresh·x4  -10.0
  15620.00  PDH            A  fresh·x4  +20.0
  15580.00  RTH-H          A  fresh·x3  -20.0
  15620.00  ONH            A  fresh·x4  +20.0
  15630.00  OR-H           A  fresh·x4  +30.0
  15450.00  PDL            A  fresh·x3  -150.0
  15450.00  RTH-L          A  fresh·x3  -150.0
Anchor: react AT these levels (grade A>B>C); do not chase price between them.
```
(`fresh·xN` = confluence count; all A here because the fixture clusters densely
around 15600. With live level-state, burned levels decay to tested/B/C.)

Sample regime line (from the regime fixtures):
```
REGIME: trend D=up 1h=up · ATR14=40.9 (LOW p0) · RV=…%(warming) · VIX=n/a · exp-range≈41pts · o/n-gap=+0.24×ATR
```

## EXIT BAR (partial)
- `go build ./...` ✓ · `go vet ./kernel ./store` ✓ · `go test ./...` ✓ ·
  `go test -race ./kernel ./store` ✓.
- Goldens: **untouched** (`git diff d8e2f88c..HEAD -- kernel/testdata web` empty) —
  the KEY LEVELS block is not yet wired into the live prompt (P1.7), so no golden
  changed. tsc/npm N/A (zero FE files).

## What remains (next session — clean item boundaries)
- **P1.3** durable session-profile store + 17:00-CT snapshot writer (RECON #2,
  MANDATORY). New store table + a roll-edge writer in the trader loop; warms
  forward, WARMING n/10 persisted. Couples to `auto_trader` — its own session.
- **P1.4** EQH/EQL + S/D zones + FVG/OB (C/confluence-only). Pure detectors.
- **P1.7** KEY LEVELS block into the LIVE executor prompt. Scorer + renderer are
  READY; this wires `SetKeyLevelsContext` (mirror `SetSVPContext`) into
  `engine_prompt_futures.go` + computes/sets it in `engine_analysis.go`, and
  regenerates the futures goldens DELIBERATELY (list every diff). B9-style empty
  log. Recon anchors for the injection point are captured
  (`/tmp/.../wkxs9h0i4.output`).
- **P1.8** calendar fetcher (ForexFactory weekly JSON + static T1 fallback,
  per-day slice stored with the trade date, replay-frozen).

All shipped code is ADDITIVE + DORMANT — nothing runs until ★ OWNER RESTART 1
(after P2). vlauto: DEFERRED to the next propagation train.
