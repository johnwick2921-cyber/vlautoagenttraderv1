# F0 — Calendar Ignition Fix (surgical)

**LINE 1: FIXED — root cause was gate ORDER: the W3 calendar producer sat BELOW the CME session gate ([trader/auto_trader_loop.go:68-70] `cmeSessionClosedSkip() → return nil`), so the owner's Saturday 12:40 restart skipped every cycle before reaching it — zero fetches, zero slices, all weekend. 4 commits, all additive, 26 pkgs green.**

| # | Commit | What |
|---|--------|------|
| F0.1 | `2985e50f` | Producer HOISTED above the session gate (needs no market/NT8/account; weekend boots now ignite). Plain logs: `📅 fetched N events — stored M day slice(s)` / `📅 fallback static — stored …` / `📅 skip-fresh` (once per trade date). Injectable fetch seam |
| F0.2 | `3d91e574` | FF 404/outage → static T1 fallback **stores rows** (`source=static`): owner-editable `calendar_static_t1.json` (env `NOFX_CALENDAR_STATIC` overrides), template ships with the confirmed week red — FOMC Minutes Wed 08-19 13:00 CT. Blackout coverage no longer depends on feed availability |
| F0.3 | `5578966a` | Freshness: skip-fresh honors only `source=forexfactory`; static/none slices are stale → re-fetch (1h throttle) + `UpsertSliceUpgrade` replaces them on feed recovery; live rows stay FROZEN (replay rule unchanged). New trade dates re-fetch by absence → Sunday week-roll covered |
| F0.4 | `c886b359` | Test matrix, all asserting storage: `TestF0FetchOKStores` · `TestF0Fallback404StoresStatic` · `TestF0SkipFreshNoRefetch` (fatal-if-called stub) · `TestF0StaticUpgradesToLive` (+frozen re-check) — PASS |

STEP 0 receipts: HEAD `cf66b016` ≥ train head `0f79fb4f` ✓ · landed train = W1–W10 marked FINAL in DAYPLAN-IN-PROGRESS.md (no W11/W12 markers exist anywhere in tree — dispatch naming assumed complete per that FINAL block) ✓ · `git diff HEAD` empty (5 "M" files were stat-only) ✓ · live bot untouched ✓.

## ⏫ Deploy handoff (owner, before MON 08:00 CT)
```
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
sudo systemctl restart nofx
```
Receipts after restart (works even Sat/Sun now): `grep 📅 data/nofx_$(date +%F).log` → expect `fetched N events` (or `fallback static` while FF still serves the old week — it upgrades automatically after Sunday's roll); then `calendar_slices` rows for the new week, Wed 08-19 carrying the FOMC T1. Sunday-evening re-check recommended.
