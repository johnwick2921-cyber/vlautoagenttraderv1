# Post-boot observation kit (BOOT-DAY, DS-101 · docs/postboot-kit-0926)

Read-only checks to run after the weekend boot and again at **Sunday 17:00 CT
market open**. Every check names the exact log line or SQL that proves it;
`deploy/postboot-check.sh` runs all of them and exits with the FAIL count.

- Run 1 (right after the boot-line sweep): `bash deploy/postboot-check.sh`
- Run 2 (Sunday 17:00 CT open): same script — the open-critical subset is
  planner-max, fast-market max, salvage counters, no-ERROR (checks 5, 8, 9, 10).

Defaults: log = newest `data/nofx_*.log`, DB `data/data.db` (read-only via
`sqlite3 -readonly "file:…?mode=ro"`), bin `nofx-bin`, env `.env`.
`--selfcheck` proves every expected string still exists in the sources.

## The checks, and the proof each one reads

| # | Check | Proof (log line / SQL) |
|---|-------|------------------------|
| 1 | stop-entry (attempted-entry) guard | `🎯 stop-entry: seam=… · slots=… · guard=…` — `guard=MISROUTED` = REFUSED; the line is rendered from the SAME predicate the placement branch calls (`trader/armed_executor.go StopEntryBootLine`, emitted on the first armed cycle with a bound NT8 trader, re-emitted only when it changes). |
| 2 | retention off | `🧹 retention: decision_records=off(keep all, rows=N) · equity_snapshots=… · nt8_order_snapshots=… · level_stats=…` — every table must read `off(keep all, rows=N)` and N must EQUAL `SELECT COUNT(*) FROM <table>` (live rows, never literal). Also `🧹 log retention: off (keep all)` for the log-prune knob. |
| 3 | JWT | `🔑 JWT secret configured` fires UNCONDITIONALLY — the real proof is `.env`: `JWT_SECRET` ≥ 24 chars and ≠ the shipped default. |
| 4 | transport | per-trader `🏦 [name] Using NinjaTrader (transport via NT_TRANSPORT env, CME futures via SIM)` AND the absence of `⚠️ transport: … UNSET — using the deprecated CSV transport`; `.env` must hold `NT_TRANSPORT=tcp`. |
| 5 | fast-market reasoning matches .env | `.env` `FAST_MARKET_REASONING` (default `max` when unset — the owner's MAX rule). Every `planner mode: fast-market … reasoning downgraded to …` line must carry THAT level; any other level is a mismatch FAIL. (The owner's HIGH A/B from this weekend boot passes when .env says high.) |
| 6 | death-reread retry | `🧬 death→reread=on(default)` / `=on(saved)` (never `=off`); per-strategy knob read from SQL: `SELECT config FROM strategies` → `.day_plan.death_reread_retry_min` (nil = wake_min_interval_min; the owner-decided value). |
| 7 | busy_timeout | the shipped binary embeds the per-connection DSN pragma (`#246` P1-B): `strings nofx-bin` must contain `_pragma=busy_timeout(5000)` or `_busy_timeout=5000`. A `PRAGMA busy_timeout` on the CLI's own connection proves nothing — the pool opens its own. |
| 8 | first planner read at the .env level | the expected level is READ from `.env` `AI_PLAN_REASONING` (default `max` when unset). The FIRST `🧠 planner call (reasoning=… wire=… cap=…)` after boot must carry THAT level; PASS = match, FAIL only on a mismatch. (Owner A2 update + timing correction 07:3x CT 2026-09-26: the owner's HIGH A/B runs from THIS weekend boot via the owner's `.env` — a `reasoning=high` reading is the test in force, not a boot defect.) 8b: duplicate `AI_PLAN_REASONING` / `FAST_MARKET_REASONING` / `AI_EXEC_REASONING` keys in `.env` WARN (the bot reads the LAST occurrence) — remove duplicates. |
| 9 | born-dead salvage counters | `plan liveness: … · born-dead dropped=N` must EQUAL `SELECT COUNT(*) FROM system_config WHERE key LIKE 'plan_liveness_event:born_dead_dropped:%'` (counters record, never infer). |
| 10 | no ERROR lines | `grep -c '\[ERRO\]' <log>` must be 0; any hit is printed (a human must look). |
| 11 | no `_ =` regressions | `git grep -E '_ = [a-zA-Z_]' -- '*.go' ':!*_test.go'` counted hits must be ≤ the documented baseline **432 at dev `35a53d29`** (19 of them sit in the changed files of the fix-planner wave and are pre-existing — see PR #248's ERROR TRACEABILITY). Bump this number deliberately, here, when a reviewed wave adds a justified hit. |

## Sunday 17:00 CT open — what the subset means

At the open, re-run checks 5, 8, 9, 10: the first planner read of the day must
be at max (8), any fast-market read must not downgrade (5), the salvage
counter must still match the live event count (9) and the day must be clean of
`[ERRO]` (10). A failed open-check means the bot is trading on something other
than the booted configuration — stop and report before touching anything.

## Anti-rot

`deploy/postboot-check.sh --selfcheck` greps the repo sources for every needle
the kit depends on (the emoji-prefixed boot lines, the transport WARN, the DSN
pragma). A wave that renames a boot line breaks the selfcheck, not the boot.
