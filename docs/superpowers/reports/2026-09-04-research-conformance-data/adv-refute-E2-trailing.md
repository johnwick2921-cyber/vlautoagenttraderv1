# ADVERSARIAL VERIFY — E2 "Trailing 2.0×ATR14 after breakeven"
Verdict: **CONFIRMED — could not refute.** Peer finding stands; evidence is STRONGER than they stated.
Files verified identical to deployed rev 70af663d: `git diff --stat 70af663d HEAD -- trader/auto_trader_trailing.go trader/auto_trader_pause.go trader/auto_trader_risk.go` → EMPTY.

## Resolved value NOW (A11 — boot line + resolver, never a file default)
- `/home/hoang/nofx/data/nofx_2026-09-04.log` 09-04 08:30:11 `nofx/main.go:335` → `🛑 exits: … BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)`
- resolver `trader/exit_mechs_suspend.go:35-43` → default `return true` (:42). `EXIT_MECHS_SUSPENDED` absent from `/home/hoang/nofx/.env` AND from `/proc/878451/environ` (0 hits each). → SUSPENDED. [A]
- Stored strategy `a5b7662e` ("MNQ", bound to running trader `hoang`) `ai_config.risk_control` keys present: `trailing_enabled=true`, `trailing_atr_period=14`. `trailing_atr_mult` and `trailing_arm` ABSENT. [A]

## The contradiction, printed by ONE process 0 seconds apart (peer did not cite this)
```
09-04 08:30:11 nofx/main.go:335            🛑 exits: … BE=off · trail=off …
09-04 08:30:11 trader/auto_trader.go:43    🧾 ledger boot: … trailing=2.0×ATR14 arm=after_breakeven (source: studio) …
```
`trader/auto_trader_pause.go:196-201` computes `trailing` from `trailingConfig()` ONLY — never calls `exitMechsSuspended()`. [A]
"(source: studio)" is false for 2 of 3 fields: mult 2.0 = `defaultTrailingATRMult` (`auto_trader_trailing.go:26`, taken at :45-47); arm after_breakeven = `TrailArmAfterBreakeven` (:30, taken at :53-57). Only period 14 is stored — and it equals the code default `defaultTrailingATRPeriod` (:27). [A]

## STRONGER THAN THE PEER SAID — the trail cannot even log its own refusal
`auto_trader.go:157` `exitMechSuspendedRefuse("auto-breakeven", …)` returns BEFORE `:161 at.breakevenDone[key] = true`.
→ under suspension `breakevenDone` is NEVER set → `beFired` permanently false
→ `trailArmed("after_breakeven", false, …)` false (`auto_trader_trailing.go:63-72`)
→ `maybeTrailStop` returns at **:156**, before the `📈 trailing_armed` WARN at :161 AND before its own suspension gate at :180.
The A9 promise at `exit_mechs_suspend.go:58-60` ("never a silent skip") is VIOLATED for `atr-trail`. [A]

Measured, n=5 positions since 0B (4657560b 2026-09-02 07:33:39 CT):
| id | side | entry | MFE | MAE | entry CT |
|---|---|---|---|---|---|
| 587 | LONG | 29079.25 | 25.75 | 33.00 | 2026-09-02 00:17:44 (pre-0B) |
| 588 | LONG | 29082.50 | 0.00 | 33.25 | 2026-09-02 07:41:05 |
| 589 | LONG | 29192.50 | 10.25 | 80.50 | 2026-09-02 09:41:04 |
| 590 | LONG | 29193.25 | 1.00 | 49.75 | 2026-09-02 10:37:17 |
| 591 | SHORT | 29285.00 | 43.50 | 75.00 | 2026-09-03 09:05:14 |

id 591 MFE 43.5 > breakeven_trigger_points 40 → the BE trigger DID fire, yet:
`grep -c` over `data/nofx_2026-09-0{2,3,4}.log`: `trailing_armed`=0, `trailing_moved`=0, `SUSPENDED`=0 (all three days).
`log_events` since 2026-09-02: `message LIKE '%trailing%'`=0, `LIKE '%SUSPENDED%'`=0, **total rows=8068** (sink demonstrably alive). [A]

## Production callers — peer said 1, actual E2 surface is 4 (method-level search, `_test.go` excluded)
| symbol | prod callers | site |
|---|---|---|
| `maybeTrailStop` | 1 | `trader/auto_trader_risk.go:104` |
| `trailingConfig` | 2 | `auto_trader_trailing.go:119` + **`auto_trader_pause.go:198` (the lying boot line)** |
| `pruneTrailStates` | 1 | `trader/auto_trader_risk.go:158` |
| `currentTrailLevel` | 1 | `trader/auto_trader_watcher.go:302` → `:342` → `kernel/engine_prompt_observer.go:80-81` ("· trail armed @ %.2f") |
All 4 inert while suspended (`LastEmitted` is set only at :195, downstream of the :180 gate → `currentTrailLevel` always 0). No alternate ATR-trail exists: repo-wide `grep -i trail` over non-test .go finds no second implementation. [A]

## Grounding audit
`git log -1 -- docs/superpowers/reports/2026-09-02-belief-census.md`
→ `ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 — every market belief labeled [R]/[X]/[T]/[I]/[O] with live effect + demotion queue (read-only)`
census:85 verbatim: `| E2 | Trailing 2.0×ATR14 after breakeven | boot ledger (trailing=2.0×ATR14, studio) | [O] | exit |`
Citation ACCURATE. But **circular**: the census's own "Where" column IS the boot ledger — it sourced E2 from the line that never consults `exitMechsSuspended()`. Census pinned **77 minutes AFTER** the suspension shipped (07:33:39 → 08:50:38 CT, same day) and still records effect=`exit`. [A]
Nit (A21): census:85 does not contain the word "ON"; effect=`exit` is the peer's paraphrase. Fair, but it is a paraphrase.

## The [X] rationale is UNSOURCED
`exit_mechs_suspend.go:17-19` and `web/src/guide/content/settings.ts:417` both cite "Round-7 research / a 567,000-backtest study ranks ATR/Chandelier trails in the worst group of 15 exit families".
`grep -rn "567,000\|567000\|Chandelier\|worst group of 15" docs/ --include=*.md` → **0 hits**. No rounds corpus exists on dev (`docs/superpowers/research/` holds only INDEX.md — `4e8e7e1ae069bc0285f677a316b4771437a39a06 2026-09-03 19:37:14 -0500`). Nearest is `2026-09-03-trade-excursions.md:107` (`0c1a808ca1ca90dee9dad84a9d5403f11211406b 2026-09-03 00:05:11 -0500`), which itself only asserts "round 7 ruled…" with no citation. Label is **[O] with an unsourced [X] rationale** — not [R]. [A]

## Third lying surface the peer missed
`store/knob_registry_table.go:153-157` — all five `trailing_*` knobs `Status: KnobLive`, `Note: ""`, none using the registry's own `KnobIneffective`. Under 0B none can move a stop. [A]

## The one surface that CONFORMS
`web/src/guide/content/settings.ts:415-427` — "Trailing stop — SUSPENDED (0B) … the ratchet still computes a level, but NO move_stop frame is sent — the boot line reads trail=off". `status.ts:67` quotes `trail=off`. Guide is HONEST; ledger boot line, knob registry and belief census are not. [A]

## ROW
E2 trailing · `trader/auto_trader_trailing.go:115-200` (gate `:180`), cfg `:42-60` · resolved NOW = **OFF/suspended** (boot 08:30:11 `trail=off`; `EXIT_MECHS_SUSPENDED` unset → `exit_mechs_suspend.go:42` default true) · **[O]** (unsourced [X] rationale) · census:85 · live effect = **NONE on the wire**; WARN-only/label damage via `auto_trader_pause.go:202` + `knob_registry_table.go:153-157` · **CONFORMS? NO — owner/census value = live "exit" 2.0×ATR14 after breakeven (census:85) vs resolved OFF** · production callers = 4 (`trader/auto_trader_risk.go:104` +3)
