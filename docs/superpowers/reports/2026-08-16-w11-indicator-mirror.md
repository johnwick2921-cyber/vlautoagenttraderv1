# W11 — Planner Indicator Mirror + W11b flagged follow-ups

**LINE 1 — W11 + W11b DONE + GREEN.** Planner now sees the executor's EXACT
indicator state (zero new config), frozen per plan row; persisted level freshness +
overnight-gap now surface. 26 pkgs green, executor goldens byte-identical. Commits
`cbf12870` (W11) · `a7bdae50` (W11b). Owner deploys at the end.

## W11 — indicator mirror (owner override, spec §regime) [A]
- **Same renderer, not a re-derivation:** extracted `kernel.FormatIndicatorState`
  from the executor's `formatTimeframeSeriesData` (the method now delegates →
  executor output byte-identical). `kernel.RenderPlannerIndicatorBlock` renders
  per-TF EMA/MACD/RSI/ATR/BOLL through it; `trader.renderIndicatorMirror` computes
  `market.Data` via the SAME path (`market.GetWithTimeframes` → `FuturesBarsProvider`,
  offline) with the SAME ai_config toggles/periods/timeframes. Test proves the
  mirror body == executor renderer byte-for-byte.
- **Gated + dormant:** `BuildPlannerPrompt` emits `## Indicators` only when the block
  is non-empty → no toggle / no bars = byte-identical planner prompt.
- **Replay-frozen:** the rendered block is stamped onto the plan row at write time
  (`plans.indicators_block` + `plans.ai_config_hash`) — a later toggle change never
  rewrites what the planner saw. plans bypasses AutoMigrate → added to raw `plansDDL`
  (fresh DBs) + idempotent `ALTER TABLE` (existing sqlite). ai_config fingerprint =
  sha256 over a canonical projection of ONLY prompt-reaching fields (excludes the
  NofxOSAPIKey secret + crypto ranking); logged on every plan row.

## W11b — the two flagged follow-ups (rode the same regen) [A]
1. **Persisted freshness surfaces:** `AssembleScoredLevels`'s nil freshness callback
   → `kernel.LevelStateProvider` (trader installs it over `store.LevelStateStore`,
   SAME identity as W7's writer). KEY LEVELS drops a consumed level + labels
   B/tested; `RenderPlanStatus` (now takes `symbol`) annotates burned levels.
2. **Overnight gap:** `kernel.PriorCloseSessionOpen(daily)` feeds
   `RegimeInputs.PriorClose/SessionOpen` → the gap×ATR field (was inert). VIX n/a.

## GOLDEN REGEN — deliberate, both states, every diff [A]
Executor prompt goldens (`futures_mnq_{plan,empty,keylevels}.golden`): **byte-
identical, ZERO regenerated.** Every change is dormant in the disabled state:
- `FormatIndicatorState` extraction — pure refactor, executor output unchanged.
- W11b(1) freshness — nil `LevelStateProvider` in tests/goldens → all-fresh.
- `RenderPlanStatus` gains `symbol` — annotation appended ONLY when a provider
  returns non-fresh; nil provider → identical line.
- W11 planner block — the planner prompt is assertion-tested (not a byte golden);
  empty `IndicatorsBlock` → no `## Indicators` section.
**ENABLED state** proven by new tests (provider installed / block non-empty), not a
golden file, since it requires live level-state + bars.

## SAMPLE — real assembled INDICATORS block (real MNQ daily bars → real values)
```
## Indicators (executor mirror — your ai_config toggles, computed values)
### 1d
EMA20: [29071.9000]
RSI14: [58.9204, 57.8908, 56.6796, 58.5898, 61.3009, 60.7737]
ATR14: 595.8473
```
(ai_config fingerprint stamped alongside; 20 real daily bars 07-20..08-14.)

## Deploy
Go-only, no `.cs` → no NT8 F5. Applies at the next planner read. Owner handoff at
the end of the W12 report (single rebuild + `systemctl restart`).
