# 2026-08-26 · Touch Telemetry Addendum (T1–T5) — Pack B final pack

**PR:** #78 (same wave PR, touch commits folded in before merge) · **Branch:** `feat/volume-levels`
**Merged → dev:** `024cf6b0` · **Deployed:** rev `024cf6b0` (boot `goldens PASS`), flat cutover.

## Scope

Machine read of level reactions — **ADVISORY: zero gates, zero order authority, telemetry only.**
Every threshold env-tunable; rejection/absorption semantics per order-flow research (citation comments in `kernel/touch_telemetry.go`).

## T1 — touch detector

- Episode opens when price comes within `TOUCH_BAND_TICKS` (env, default 16 = **4.0 pts**) of a seated level; closes on band exit OR `TOUCH_EPISODE_MAX_BARS` (default 12).
- Per episode (1m bars + live volume): **penetration** (max pts through the level, wick vs body) · **close side** (1m and 5m: closed back on the approach side = REJECT, closed through = ACCEPT building) · **volume** (episode vol ÷ 20-bar pre-episode average = spike ratio, `TOUCH_VOL_LOOKBACK=20`) · **approach speed** (pts in the 5 bars before touch, in ATR multiples, `TOUCH_APPROACH_BARS=5`) · **touch number** (1st/2nd/3rd+, monotonic per level — freshness tie via level_state).
- **Dedup:** one episode per level, never one per bar (locked by test).
- Persisted to `touch_episodes` (additive, append-only) via `TouchEpisodeSink`; feeds `level_stats` through `EpisodeCountByLevel` (rejections/acceptances join for the 2-week verdict).

## T2 — prompt line (executor + watcher)

One line per active episode, max 2 nearest, facts only:
`TOUCH: PDC 29228.50 · 1st touch · through 6pt · 1m closed through above · vol 2.1×avg · fast approach 1.8×ATR — shape: acceptance`

Executor: appended under KEY LEVELS. Watcher: `## TOUCH` section in the observer prompt (in-position only).

## T3 — scenario tie-in

A scenario whose `confirm.ref_price` sits within 3 pts of an OPEN episode's level gets the live shape appended to the confirm advisory:
`confirm S2 NOT MET — touch active at PDC 29228.50: forming shape` — **no new gates**, the confirm machinery is unchanged.

## T4 — card chips

Seated level rows render a live touch chip: ○ approaching · ◐ touching · ✕ rejected · ▲ accepted (backend `touch_state` per level fact, FE `ZoneTable`).

## T5 — tests (golden fixtures)

`kernel/touch_telemetry_test.go`: episode open/close + dedup · penetration + close-side math on golden fixtures (wick-through vs body-through) · vol ratio + approach ATR · prompt render · card states · touch numbering. `go test ./...` EXIT=0; FE tsc clean; vitest 263 PASS.

## Boot line

```
🎯 touch telemetry: band=16t(4.0pt) max_bars=12 vol_lookback=20 approach=5 — advisory, zero gates
```

## Live episode quote

- **Live OPEN episode** (executor prompt, 14:47:55 UTC — price within the 4-pt band of PDC):
  `TOUCH: PDC 29228.50 · 1st touch · none · n/a · shape: forming`
- **Sim CLOSED episode** (T5 golden fixture `TestTouchPenetrationMath`): level 100, approach from below →
  `penetration 6.0pt (wick 6.0 / body 5.0) · close_1m accept · shape acceptance`.
  A live closed row will land in `touch_episodes` when price re-enters a seated level's band
  (post-fix market moved 13+ pts clear of every seated level; the sink fires on episode close).

## Live-caught bug fixed before the report closed

The first live episode (14:50:54) rendered `through 83pt` — impossible for a 4-pt band episode.
Root cause: penetration was measured over the rolling RING (which keeps pre-episode bars for the
vol-ratio/approach windows), so a pre-episode bar far beyond the level leaked in. Fixed (`0c8a3ecb`):
penetration now counts episode bars only (`OpenTime ≥ OpenedAtMs`); regression test
`TestTouchPenetrationEpisodeScoped` locks it. Redeployed flat, boot `goldens PASS`.
