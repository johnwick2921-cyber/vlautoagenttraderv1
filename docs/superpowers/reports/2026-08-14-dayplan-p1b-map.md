# DAY-PLAN CAMPAIGN — P1 · THE MAP COMPLETE (final 4 items)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `2edee42a` (P1 checkpoint) → `09579af3` · 4 feature commits
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**P1 · THE MAP is COMPLETE (8/8) and pushed.** The final four — zones (P1.4),
the live-prompt wire (P1.7), the durable session-profile store + snapshot writer
+ nPOC (P1.3), and the calendar fetcher (P1.8) — all landed with tests; the full
suite + `-race` (kernel, store, trader, calendar) are green; the ONLY golden
change is the new enabled-state `futures_mnq_keylevels.golden` (every existing
golden is byte-identical); every change is additive + dormant (the running bot is
untouched — the map activates only when the owner enables day_plan at ★ RESTART 1).

## STEP 0 gate
PASS — HEAD `2edee42a` · tree clean · bot PID 363618 untouched · both cycling.

## Items shipped

### P1.4 — EQH/EQL + S/D zones + FVG + OB — `f138b7f3`
`kernel/levels_zones.go` (+ shared `closedBars`/`zoneLevel`). Four pure detectors:
EqualHighsLows (k=2 strict pivots clustered within tol), SupplyDemandZones
(small-bodied base + ≥1.5×ATR departure → demand/supply), FairValueGaps
(3-candle imbalance, unfilled only), OrderBlocks (last opposing candle before a
≥1.5×ATR displacement). All enter the scorer as C/confluence-only — a test proves
a standalone zone is excluded by `ScoreLevels`.

### P1.7 — KEY LEVELS wired into the live futures prompt — `c42a629c`
Mirrors the SVP seam. `engine.go`: `keyLevelsContextLine` + `SetKeyLevelsContext`.
`engine_prompt_futures.go`: gated consumption right after the SVP block (before
Entry Standards) — renders ONLY when `day_plan.plan_enabled` AND the block is
non-empty. `engine_analysis.go`: once-per-cycle compute/set via
`BuildKeyLevelsBlock` (+ nPOC via the provider); B9-style INFO log when empty.
`kernel/levels_assemble.go`: `BuildKeyLevelsBlock` (all detectors → scorer →
renderer) + `DailyATRProxy`.

**GOLDENS (deliberate — every diff listed):**
- `kernel/testdata/futures_mnq_empty.golden` — **UNCHANGED** (disabled state; the
  block is gated off by default). Proven by `TestFuturesPromptEmptyBoxesByteIdentical`
  + SubModes + SVP tests still green.
- `kernel/testdata/golden/*.txt` (aggressive/conservative/user) — **UNCHANGED**.
- `kernel/testdata/futures_mnq_keylevels.golden` — **NEW** (enabled state). Its
  only difference vs the empty golden is the KEY LEVELS block inserted after the
  Available-Data/SVP section:
  ```
  Use confluence across timeframes. Confidence ≥ 75 required to open.

  KEY LEVELS (map, nearest-first; price 21500.00):
    21500.00  PDC            A  fresh     +0.0
    21520.00  PDH            A  fresh·x2  +20.0
    21480.00  RTH-L          B  fresh     -20.0
  Anchor: react AT these levels (grade A>B>C); do not chase price between them.

  # Decision Process
  ```
Config-truth: renders only when day_plan enabled — both states tested
(`TestFuturesKeyLevelsInjection`: on/off/disabled/empty).

### P1.3 — durable session-profile store + snapshot writer + nPOC — `a450adfc`
RECON #2 (MANDATORY). `store/session_profiles` (PK symbol,session_date; POC/VAH/
VAL + range + profile JSON), append-only `SaveIfAbsent` — restart-safe (reload
replays, no loss/dupes; proven). `kernel/naked_poc.go`: `NakedPOCs` pure detector
(a prior POC is naked until a LATER session brackets it; daily<weekly HTF bump) +
the `NakedPOCProvider` seam (kernel stays DB-blind). `trader/auto_trader_dayplan.go`:
`snapshotSessionProfiles` hooks the trader loop's roll region, GATED (futures +
day_plan) → DORMANT; persists newly-frozen sessions (warms forward) + installs the
nPOC provider once. `engine_analysis.go` now threads nPOC into the block.

### P1.8 — calendar fetcher — `09579af3`
New `calendar/` package: `FetchWeek(fetch, static)` — injectable fetch. Parses
ForexFactory weekly JSON, maps High→T1 / Medium→T2 (Low/Holiday ignored), filters
to USD/EUR/GBP/JPY/CNY, groups by CT date. Outage / parse error → static T1
fallback + a Warning (never errors, never panics — blackouts never silently
vanish). `SessionCurrencies`/`EventsForSession` apply the per-session filter.
`store/calendar_slices`: per-trade-date slice stored ONCE (`SaveSliceIfAbsent` =
replay-frozen) + source tag.

**Stored calendar-slice sample** (`FetchWeek(sample)` → what gets stored):
```
trade_date=2026-08-13  source=forexfactory  events_json=[
  {"time":"2026-08-13T05:00:00Z","currency":"JPY","title":"BOJ Rate","impact":"T1"},
  {"time":"2026-08-13T12:30:00Z","currency":"USD","title":"CPI m/m","impact":"T2"},
  {"time":"2026-08-13T18:00:00Z","currency":"USD","title":"FOMC Statement","impact":"T1"}
]
```
(Low-impact + off-list currencies dropped; a network outage would instead store
`source=static` from the fallback file, with a Warning surfaced.)

## Sample KEY LEVELS block — REAL assembled pipeline (BuildKeyLevelsBlock)
Fixture price 15600, dATR 120, cap 8 (detectors → scorer → renderer):
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

## EXIT BAR
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./kernel ./store ./trader ./calendar` ✓.
- Goldens: only the NEW `futures_mnq_keylevels.golden` (+69); ALL existing
  goldens byte-identical (`git diff 2edee42a..HEAD -- kernel/testdata/…` empty for
  the existing files).
- config-truth: `day_plan` gates the KEY LEVELS block (both states tested); no new
  StrategyConfig scalar field beyond P0.1's `day_plan`.
- tsc/npm: N/A — zero `web/` files touched (the Plan Card is P4).

## Dormancy / deploy
Nothing runs until the owner enables day_plan and restarts (★ RESTART 1, after
P2): the prompt block, the snapshot writer, and the nPOC provider are all gated on
`plan_enabled` (default false). New tables (`session_profiles`, `calendar_slices`)
are created by AutoMigrate on the next binary start; additive + empty.

## What's next — P2 · THE CLOCK
True bar-close cadence (replace the scan timer), BUILD the any-position→skip-cycle
gate (RECON #5: does not exist), last_entry 14:00 ET + eod_flat 15:45 ET armed
(session-flat routes through the trader close path — RECON #10), MAE/MFE +
confidence logging. Ends at ★ OWNER RESTART 1 (map + clock live). vlauto: DEFERRED.
