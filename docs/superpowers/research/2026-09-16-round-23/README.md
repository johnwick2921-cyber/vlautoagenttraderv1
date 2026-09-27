# Round 23 — "HTF defines the map, LTF times the entry" (in progress)

Read/investigate lane (R101, DeepSeek). Dispatch received over the agent bridge
from the CTO 2026-09-16 20:57Z. This README records the method BEFORE the
numbers land (the report in `docs/superpowers/reports/` cites this file).

## The owner's hypothesis (2026-09-16)

The planner seats ~24 of ~400 in-band candidates with a flat timeframe model:
`zoneTFMult` 1.0/1.1/1.2/1.3 for 1m/15m/1h/4h, `HTFScoreMultiplier` ×1.2
(UNTESTED [I]), only 2 HTF seats guaranteed (`seatHTF` maxHTFSeats=2), and
reference lines never decay while zones decay 1.0/0.6/0.3/0.15. Plans carry
8–10 session references and 0–2 structure zones; the owner sees no 4h levels.
This is the research round for that belief: no market belief is designed
without a round.

## Instrument — the calibrated D1′ touch detector

`kernel.DetectorK()` = 3, Δ = trailing-5-day mean |1m close-to-close
increment| (≈5.34 pts → barriers ≈ ±16pt, the round-21 calibration),
H = `kernel.DetectorHorizonBars()` = 12×1m, exit_on = close.
IID null p(hold) = 0.5067. All outcomes come from `kernel.DetectTouchOutcomes`
— the ONE live detector, never reimplemented. `kernel.HoldRate` /
`kernel.WilsonInterval` for rates and intervals.

## Population (the sampling decision)

Every level in the raw detected universe of each replayed read
(`kernel.AssembleResearchLevels` raw return, already dedupeSameKind per read),
scanned in the referencing session's window
`[max(session start, FormedAtMs), session flat)`. A level referenced by several
sessions contributes one scan per session — a (plan session × level) use of the
map. D1′ episodes are anchored at the level's `Price` line (not the zone band).
Levels with `FormedAtMs` = 0 scan from session start (documented; mostly
pre-W-TF detections).

Reads are replayed at the `DefaultSessionRegistry` times (LONDON 01:30,
NY 08:00, ASIA 16:30 CT) for every CME day of the era, from the DB COPY
(`data/db.copy.db`, backup API snapshot of the live store taken 2026-09-16
~15:59 CT — the live store is never opened). The live detection chain is called
offline with the same call sites as the backtest-zone-fade harness
(`docs/superpowers/research/2026-09-12-backtest-zone-fade/README.md` methods,
including: per-contract bars via `store.BarsBetweenOn` exclusions, `ContractAt`
replay, `DetectHTFLevels(["D","4h","1h","15m"])`, nPOC extras from the COPY's
session_profiles only, no owner levels, no LevelStateProvider → all-fresh).

## Q1 — detection-TF × kind first-touch hold

Cells = TF {1m, 5m, 15m, 1h, 4h, 1d} × kind (43 `kernel.LevelKind` values) and
× family (`kernel.ZoneFamily`: swing-structure / volume-node / round-number /
session/derived / imbalance). Rate = hold/(hold+break); ambiguous episodes are
recorded and excluded from the rate, never dropped. Wilson CI + exact
two-sided binomial p vs the 0.5067 null. Cells below n=200 are reported
UNMEASURED. Output: `out/q1_cells.json`.

## Q4 — reference-line decay

Kinds PDH / PDL / ONH / ONL / OR-H / OR-L / VWAP: hold rate by touch ordinal
(1st, 2nd, 3rd+) per level-session scan. If reference-line hold decays with
touches, the anchors-no-decay rule is wrong. Output: `out/q4_ordinals.json`.

## Q5 — entry timing (LTF half), computed on every ordinal-1 episode

Three entries on the SAME tape, SAME barriers anchored at the level, each via
`localOutcomeFrom` — a line-for-line copy of the D1′ inner loop with the open
condition replaced by "start at index, entry side given" (a parity assertion at
every ordinal-1 episode requires the loop at the touch index to reproduce the
kernel episode's outcome/MFE/MAE exactly — class-97 guard):

- **touch** — the kernel D1′ episode itself.
- **1x5m confirm** — entry at the first 5m bucket CLOSE strictly beyond the
  level (either side) at/after the touch, within the episode's H-bar window;
  entry side = the side the previous 1m bar closed on. `no_confirm` when no
  5m close crosses within the window.
- **1m MSS** — entry at the first `kernel.EvaluateMSS` verdict (either side,
  production evaluator: k=2 fractal swing broken by a 1m close,
  disp ≥ 0.5×ATR5m) whose BreakTimeMs falls after the touch within the
  H-bar window; entry side = the side the previous 1m bar closed on.
  `no_mss` when none fires.

Hold/break from the variant entry instant under the same ±k·Δ barriers, H=12.
Output: `out/episodes.jsonl` (variant fields on ordinal-1 rows).

## Q2/Q3 — analyzed from episodes.jsonl (separate pass)

- Q2: HTF (TF ∈ {4h, 1d} ∪ prior-week refs) vs intraday hold/MFE, matched on
  distance-from-price at read (bucket) and freshness (age bucket).
- Q3: measured per-TF ladder vs the 1.0/1.1/1.2/1.3 `zoneTFMult` tier ladder
  and the ×1.2 HTF multiplier.
- Q6: seat-share replay (N ∈ {4, 8, 12} HTF seats over the last 30 sessions)
  — separate pass, needs plans/fills/refusals from the store.

## Run

```sh
go build -o /tmp/r23harness ./docs/superpowers/research/2026-09-16-round-23/harness
/tmp/r23harness -db data/db.copy.db -out out    # era 2022-04-11 → last bar day
```

## Evidence tiers / laws

[A] computed here (query stated) · [B] inferred · [C] guess. Every figure
carries n and its interval; no rate without n. Read-only lane: no edits to the
deploy tree, no builds there, DB copy only, no lock verbs except `check`.
