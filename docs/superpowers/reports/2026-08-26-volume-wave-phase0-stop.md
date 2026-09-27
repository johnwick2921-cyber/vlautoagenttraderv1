# VOLUME-LEVELS WAVE — PHASE 0 STOP REPORT (2026-08-26)

Dispatch: volume-levels wave (VWAP · POC/VAH/VAL · naked POC + tier restructure
+ noise filter). Branch `feat/volume-levels` NOT created — Phase 0 killed the
wave before building, per the dispatch's own rule ("this phase can kill the
wave"). Read-only; no code changes.

---

## SEQUENCING GATE — PASSED

- (a) Sync-chain dispatch completed: PR #75 merged → `dev` (`2dc8886c` = deploy
  RELEASE marker `eeaffe83`), aligned redeploy done — running PID 2010886,
  boot: `🔐 BOOT INTEGRITY OK — rev eeaffe83cd7a +dirty · built 2026-08-26T06:03:18Z · expected eeaffe83 · goldens PASS`.
- (b) winrate-pack content (diagnostic pack §7B + calibration) shipped inside
  PR #75 → merged and deployed at the same rev. ✓

## 0.1 — VOLUME TRUTH: LIVE YES, STORED NO

**Live path — volume IS real.** The NT8 TCP bar frame carries per-bar volume:
- `provider/ninjatrader/tcp_framing.go:364` — `V float64 json:"v"` ("volume can
  be tick-volume (fractional)").
- C# source: `ninjascript/VLBarsSubscriptionManager.cs:459,511` —
  `bars.GetVolume(i)` (NT8 tick-volume per bar — real per-bar data, not zeros,
  not synthetic). The dispatch's assumption "volume already arrives on existing
  bar frames" holds for LIVE bars.

**Stored path — there are no stored bars at all.** Fresh checks:
- SQLite (`data/data.db`): no `bars`/`klines`/`candles`/`market` tables.
- Disk: no bar caches (`data/` holds only DBs + logs; no jsonl/csv bar files
  anywhere under the repo or `~`).
- REST: `/api/klines?symbol=MNQ` returns `"Get klines from CoinAnk failed"`
  (crypto legacy path — futures bars never served over REST).
- The only bar store is the in-memory **BarCache**, seeded by
  `bars_historical` TCP frames at boot (`tcp_framing.go:155`) — ≈33h of 1m
  backfill per TF, nothing beyond, and nothing persisted across restarts.

## 0.2 / 0.3 — VALIDATION REPLAY: CANNOT RUN

The test requires **Aug 19→26 stored 1m bars** (session VWAP, prior-day volume
profile, naked-POC tracking, and the 5-biggest-loser + turning-point replay).
That history does not exist anywhere. Without it there is no way to produce the
verdict table, so the GO/NO-GO decision cannot be made — and the dispatch's
NO-GO branch says STOP before spending a week of building.

## VERDICT: STOP — the wave needs a FEED FIX first (bar persistence)

This is exactly the dispatch's 0.1 stop condition ("If volume is absent/
unreliable → STOP, report: the wave needs a feed fix first") — here the blocker
is one level deeper: bars themselves are not persisted.

### Concrete unblock (small, additive — recommend as its own dispatch)

1. **Bar-history writer (Go, additive):** a tiny subscriber on the existing TCP
   bar path persists every closed 1m (and optionally 15m/1h) bar into a new
   `bar_history` SQLite table `(symbol, tf, open_time, o,h,l,c, tick_vol)`,
   UPSERT on `(symbol,tf,open_time)`. No C# change (frames already carry
   volume). Then Phase 0 replay + per-kind hit-rate log (Phase 4) both get
   real data.
2. **Alternative (no code):** export Aug 19→26 1m MNQ history from NT8 (Data →
   Export → tick/minute) and load it into a scratch DB for the Phase-0 script
   run only. Faster for the validation, but doesn't give the ongoing
   level_stats loop.
3. Keep the live-only parts of the wave (VWAP/VAH/VAL/nPOC detectors) queued —
   they work on live bars today, but the dispatch's own gate says don't build
   before validation.

### What remains buildable without stored bars (for planning only)
- Nothing in Phase 0. Phases 1–5 depend on the GO verdict.

## Recommendations for the owner
1. Approve the bar-persistence feed fix (option 1 above, ~small) OR provide an
   NT8 history export (option 2), then re-dispatch the wave.
2. The week-in-review's "data still missing" list already flagged bar
   persistence as the blocker for swing-k/MSS-FVG calibration — this wave hits
   the same wall. One feed fix unlocks three queued workstreams.
