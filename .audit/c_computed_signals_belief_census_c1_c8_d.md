names rewritten to vl on 2026-09-30 (VL rename)
## SUBSYSTEM C — COMPUTED SIGNALS · research-conformance re-check

Source tree `/home/hoang/vl-conform`. DB read-only `file:/home/hoang/vl/data/data.db?mode=ro`.
Logs `/home/hoang/vl/data/vl_2026-*.log`. Clock CT = UTC-5.
`/api/config/resolved` and `/api/risk/gate-blocks` both return **`{"error":"Missing Authorization header"}`** from this session — no API-resolved values are quoted below; every RESOLVED value comes from a boot line I read in the live log, or from the resolver code path plus the live `.env`.

**Live env, C-subsystem** (`/home/hoang/vl/.env`, key names only): the ONLY C-relevant knob set is `HTF_VETO_MODE=cross` (line 34). `HTF_VETO_TF`, `STRUCTURE_SWING_K`, `STRUCTURE_MIN_SWING_ATR`, `STRUCTURE_MSS_BODY_ATR`, `STALE_REEVAL_DRIFT_ATR`, `STALE_BAR_GRACE_S`, `STALE_DODGE`, `FAST_MARKET_ATR`, `TOUCH_BAND_TICKS`, `TOUCH_EPISODE_MAX_BARS`, `TOUCH_VOL_LOOKBACK`, `TOUCH_APPROACH_BARS`, `DETECTOR_K` are all **unset** → every one of them resolves to its code default. [A]

**Boot-8 lines read verbatim from `data/vl_2026-09-04.log`** (both present, 2 boots today):
```
🛡️ regime ledger: htf_veto=ON (Studio regime.htf_veto, default ON) · htf_veto_tf=1h (env HTF_VETO_TF)
🛡️ htf veto: mode=cross tf=1h (1h|cross|4h via HTF_VETO_MODE; cross = 1h AND 4h agree)
```

### `git log -1` for every report cited

| report | pinning commit |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 …` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX` |
| `docs/superpowers/reports/2026-08-29-weekend-audit-p2.md` | `741bfc2a 2026-09-01 07:58:16 -0500` (same archive commit) |
| `docs/superpowers/reports/2026-08-28-grand-audit-bcde-verdict.md` | `741bfc2a 2026-09-01 07:58:16 -0500` |
| `docs/superpowers/reports/2026-08-26-week-in-review.md` | `741bfc2a 2026-09-01 07:58:16 -0500` |
| `docs/superpowers/reports/2026-08-19-total-e2e-investigation.md` | `ec78b214 2026-08-18 14:49:02 -0500` |
| `docs/superpowers/reports/2026-09-01-full-system-audit.md` | `8f57f845 2026-09-03 11:41:54 -0500` |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1a 2026-09-03 19:37:14 -0500 docs(index): the stranded-branch sweep …` |
| `docs/superpowers/reports/2026-09-03-trade-excursions.md` | `0c1a808c 2026-09-03 00:05:11 -0500` |
| `docs/superpowers/reports/2026-09-02-detector-redesign.md` | `0465a10b 2026-09-02 07:58:10 -0500` |
| `docs/superpowers/reports/2026-09-04-two-day-audit.md` | `f3c640c3 2026-09-04 07:26:52 -0500` |
| `docs/superpowers/reports/2026-09-03-expectancy-1d.md` | `38a63a9b 2026-09-03 15:26:02 -0500` |
| `docs/superpowers/reports/2026-09-02-bias-calibration.md` | `2deab3c8 2026-09-02 20:53:20 -0500` |
| `docs/superpowers/reports/2026-09-02-level-kind-replay.md` | `3961f873 2026-09-02 19:03:10 -0500` |
| `docs/superpowers/reports/2026-09-02-live-bias-replay.md` | `53498adb 2026-09-02 21:02:58 -0500` |
| `docs/superpowers/reports/2026-09-03-mc-drawdown.md` | `77e1cdfc 2026-09-03 00:39:25 -0500` |

---

## THE RULE TABLE

| # | rule | file:line | resolved value NOW | label | report:line grounding | live effect | CONFORMS? | production callers |
|---|---|---|---|---|---|---|---|---|
| C1a | htf_veto **MODE** | `kernel/htf_veto.go:40-51` | **cross** (`.env:34`; boot `mode=cross`) | [O] | `2026-08-28-grand-audit-bcde-verdict.md:17` (S4) + `2026-08-29-weekend-audit-p2.md:33,48` | gate — **UNSATISFIABLE** | **NO** — research: "cross = 1h AND 4h agree → blocks counter-trend". live: **0 blocks are possible** | 2 — `kernel/engine_position.go:262`, `trader/armed_executor.go:1373` |
| C1b | htf_veto **TF** | `kernel/htf_veto.go:21-29` | **1h** (`HTF_VETO_TF` unset → `DefaultHTFVetoTF`) | [O] | `2026-09-01-full-system-audit.md:47` (boot ledger quoted) | gate input — **never read while mode=cross** (`htf_veto.go:92-101` returns before the `snap[tf]` branch) | **NO** — boot prints `tf=1h` beside a mode that ignores it | 2 — `trader/auto_trader_loop.go:418`, `trader/armed_executor.go:1373` |
| C1c | htf_veto **ENABLED** toggle | `store/resolve_source.go:38-44` | **true** (strategy `a5b7662e` has no `regime` block → `SourceShippedDefault`) | [O] | `2026-09-02-belief-census.md:56` | gate toggle — ON over a dead verdict | yes | 2 — `trader/auto_trader_loop.go:415-421`, `trader/armed_executor.go:1372` |
| C2 | VIX regime bands | `kernel/regime.go:164-175` (`vixBucket`); guard `:94-98` | `<15 / 15-20 / 20-30 / >30` — **never evaluated** | [I] | `belief-census.md:57` | advisory — contributes only a permanently-dark field to the dark-regime WARN | **NO** — census effect "advisory (dark-regime WARN)"; live it is **0 evaluations**, not an advisory | **0 REACHABLE — DEAD.** Only call site `kernel/regime.go:97`, behind `if in.VIX > 0`; the one production `ComputeRegime` call (`trader/auto_trader_planner.go:2208-2211`) never sets `VIX` |
| C3 | ATR regime buckets | `kernel/regime.go:152-163` (`atrBucket`), percentile `:74-77` | `LOW <25 · NORMAL <75 · HIGH <90 · EXTREME ≥90` (percentile of rolling daily ATR14) | [I] | `belief-census.md:58` | advisory — string into the planner prompt as `BiasRegime` | yes | 1 — `trader/auto_trader_planner.go:2208` → `:2443` |
| C4a | swing **k** | `kernel/structure.go:57-64` | **2** | [T] | `2026-08-29-weekend-audit-p2.md:48` "swing-k **KEEP 2** (best 7/9 days)"; contra `2026-08-28-grand-audit-bcde-verdict.md:70` "k=2→3 … k=3 halves churn for free" | detector input — seats `SWG-H/L·5m/15m` levels; also feeds the (dead) HTF trend and the transition stand-down | yes | 6 — `kernel/levels_swing.go:45,65,80`, `kernel/structure.go:201`, `kernel/levels_assemble.go:81,135,184` |
| C4b | min-swing ×ATR | `kernel/structure.go:66-74` | **0.25** | [I] | `2026-08-30-knob-census.md:65` labels the trio **[C]**; `belief-census.md:59` says [I] | detector input | yes | 2 — `kernel/structure.go:238`, `kernel/levels_swing.go:120` |
| C4c | MSS body ×ATR | `kernel/structure.go:76-84` | **1.5** | [I] | `2026-08-30-knob-census.md:65` | detector input → MSS events → transition stand-down gate | yes | 3 — `kernel/structure.go:334,357`, `kernel/regime_ledger.go:14` |
| C5 | stale_reeval drift | `trader/discard_burn.go:38` + `:50-57` | **0.25 × ATR(14, 5m)** | [T] | `2026-08-29-weekend-audit-p2.md:42` "stale_reeval n=5 **−$372.5 SAVING** (hero)" | **REJECT** (clean skip, `stale_reeval_refused`) | yes | 2 — `trader/auto_trader_loop.go:796`, `:805` |
| C6a | stale-bar grace | `kernel/stale_data.go:46`, `:49-57` | **15 s** | [R] | `2026-08-19-total-e2e-investigation.md:2` | **REJECT** — entry neutralized to `wait` | yes | 1 — `kernel/engine_analysis.go:601` |
| C6b | C2 clock-drift tolerance | `kernel/clock_drift.go:29` | **60 000 ms** | [R] | `2026-08-19-total-e2e-investigation.md:2`; `2026-08-19-zerotrade-forensics.md:111` "C2 is log-only now (signals feed-stamped)" | **WARN-only** — logs + `IncClockSkewObserved`; never mutates `d.Action` (`clock_drift.go:62-88`) | **NO** — `belief-census.md:61` labels the effect "gate" | 1 — `kernel/engine_analysis.go:613` |
| C7 | fast-market threshold | `trader/auto_trader_loop.go:78-87` | **1.5 × ATR5m** since last plan write | [T] | `2026-08-29-weekend-audit-p2.md:33,48` "FAST_MARKET_ATR 1.5 (**0 live triggers**) … KEEP, EVENT-WAIT" | **not a wake trigger** — (a) reasoning-wire downgrade + `FAST TAPE` prompt line, (b) class-47 **cooldown exemption** | **NO** — `belief-census.md:62` calls it a "wake trigger"; and the research's `n=0` is **stale** | 3 — `trader/auto_trader_planner.go:1243`, `trader/auto_trader_wake_levels.go:292`, `trader/class47_wake_cadence.go:223` |
| C8a | touch band | `kernel/touch_telemetry.go:28-36` | **16 ticks = 4.00 pts** | [I] | `belief-census.md:63` | advisory | yes | 3 — `kernel/touch_telemetry.go:195,521`; `TouchUpdate` ← `kernel/engine_analysis.go:426`, `trader/auto_trader_watcher.go:326` |
| C8b | episode max bars | `kernel/touch_telemetry.go:38-46` | **12** | [I] | `belief-census.md:63` | advisory | yes | 1 — `kernel/touch_telemetry.go:196` |
| C8c | vol lookback | `kernel/touch_telemetry.go:48-56` | **20** | [I] | `belief-census.md:63` (boot line `vol_lookback=20`) | advisory | yes | 2 — `kernel/touch_telemetry.go:197`, `kernel/levels_volume_boot.go:16` |
| C8d | approach bars | `kernel/touch_telemetry.go:58-66` | **5** | [I] | `belief-census.md:63` | advisory | yes | 2 — `kernel/touch_telemetry.go:198`, `kernel/levels_volume_boot.go:16` |
| C8e | **"advisory, zero gates"** claim | `kernel/touch_telemetry.go:16-17` | **TRUE — verified on all 7 production consumers** | [M] | `belief-census.md:63` | advisory / label / one WARN-only input | **yes** | 7, none gates (enumerated below) |
| C8f | `ResolvedTouchBandPoints` (the "ONE band") | `kernel/touch_telemetry.go:89-97` | `k×Δ = 3.0 × MeanAbsIncrement` (`DETECTOR_K` unset) | [M] | `kernel/level_stats_calc.go:23` asserts "measurement consumers use `kernel.ResolvedTouchBandPoints`" | **none** | **NO** | **0 — DEAD.** Only `kernel/detector_retirement_test.go:58,59,72` |

---

## C1 — THE HTF VETO IS A DEAD GATE. Both knobs printed. [A]

**mode and tf are two separate env knobs**, and they are not alternatives — the mode decides whether the tf is read at all.

- `HTFVetoTF()` — `kernel/htf_veto.go:24-29`, env `HTF_VETO_TF`, default `1h`. **Resolved: `1h`** (unset).
- `HTFVetoMode()` — `kernel/htf_veto.go:40-51`, env `HTF_VETO_MODE`, default `"1h"`. **Resolved: `cross`** (`.env:34`, and the live boot line prints `mode=cross`).

Under `cross`, `HTFVetoVerdict` never reaches the `snap[tf]` branch:

```go
// kernel/htf_veto.go:91-101
if mode == "cross" {
    st1, ok1 := snap["1h"]
    st4, ok4 := snap["4h"]
    if !ok1 || !ok4 || !vetoOpposes(side, st1.Trend) || !vetoOpposes(side, st4.Trend) {
        return false, ""
    }
```

`snap` comes from `kernel.StructureSnapshot` at exactly two production sites (`trader/auto_trader_loop.go:404`, `trader/armed_executor.go:188→430`), and that function only ever writes keys drawn from

```go
// kernel/structure.go:34
var StructureTFs = []string{"5m", "15m", "1h"}
```

**There is no 4h detector anywhere in the process.** `ok4` is always false. **`HTFVetoVerdict` returns `(false, "")` on every production call.** [A]

### The tape agrees, three ways

1. **Persisted snapshots.** 6 036 `decision_records` rows carry a structure snapshot (2026-08-23 22:13 CT → 2026-09-04 13:44). `has_1h = 6019`. **`has_4h = 0`.** [A]
2. **Log fires of `🛡️ HTF VETO`** per day file — the cross mode landed in `84543213` (2026-08-28 10:43:25 CT, "F3 HTF veto cross mode"):

| 08-23 | 08-24 | 08-25 | 08-26 | 08-27 | **08-28** | 08-29→09-04 |
|---|---|---|---|---|---|---|
| 6 | 3 | 9 | 4 | 5 | **0** | **0 every day** |

   `2026-08-26-week-in-review.md:67` counted 52/18/22/63/84/5 `htf_veto` lines on 08-21…08-26 (incl. retry echoes). Zero in the seven days since the switch. [A]
3. **Counter-trend entries now pass.** Since the cutover, **n = 19** entry decisions oppose the confirmed 1h trend (the exact class `mode=1h` refuses). 15 died on other gates — 13 `plan_mode=strict`, 1 `last_entry_cutoff`, 1 `R:R 1.76 below floor 2.00`; **none on `htf_veto`**. The other **4 passed every gate** (`risk_check_passed=1`), all `open_long` against a confirmed 1h `TRENDING_DOWN`: [T]

| decision (CT) | position | side | entry | `pnl_corrected` | MAE | MFE |
|---|---|---|---|---|---|---|
| 2026-09-02 00:17:45 | 587 | LONG | 29079.25 | −62.50 | 33.00 | 25.75 |
| 2026-09-02 07:41:06 | 588 | LONG | 29082.50 | −65.00 | 33.25 | 0.00 |
| 2026-09-02 09:41:05 | 589 | LONG | 29192.50 | −155.00 | 80.50 | 10.25 |
| 2026-09-02 10:37:18 | 590 | LONG | 29193.25 | −99.00 | 49.75 | 1.00 |
| | | | | **−381.50** | | |

That −$381.50 is, to the cent and to the MFE, the loss the **no-chase wave** was built to study (`trader/no_chase.go:16-21`: *"four longs, four stops, −$381.50 … Their MFE was 10.25 and 1.00"*). The no-chase leg refuses nothing by design (`trader/entry_gate.go:284-296` — runs last, returns `("", false)`), while a gate that **would** have refused all four had been silently unsatisfiable for five days.

### What the research said, and why it does not rescue this

- `2026-08-28-grand-audit-bcde-verdict.md:17` (S4): *"4h was RANGING at all 7 veto timestamps → a 4h-cross-check would have allowed all 7."* Correct as a replay — and it is the same fact that makes the live gate dead: the 4h leg never opposes, because in production it is not computed at all.
- `2026-08-29-weekend-audit-p2.md:42` reports **"HTF-veto-cross n=9 −$114.0 SAVING (flipped)"**, and `:33`/`:48` KEEP cross on that basis. That n=9 is a relabelled replay of 1h-era refusals; the live cross gate has **n=0 since the switch**. The KEEP ruling rests on a population the running binary cannot produce.
- `2026-09-01-full-system-audit.md:621` already recorded *"LIVE (unfired since 08-30)"* — noticed, never diagnosed.

### The boot line is honest and still misleads

`🛡️ htf veto: mode=cross tf=1h (1h|cross|4h via HTF_VETO_MODE; cross = 1h AND 4h agree)` reads every field from its resolver, yet prints a `tf` the mode never consults and describes an agreement the process has no 4h series to test. Fix owner: **code** (either seat a 4h detector in `StructureTFs`, or make `cross` refuse to arm and log `WARN: cross requested but no 4h snapshot — falling back to 1h`), then **ruling** (re-adjudicate KEEP-cross on a population that exists).

---

## C2/C3 — regime bands

`ComputeRegime` has **one** production call site, `trader/auto_trader_planner.go:2208-2211`, and its `RegimeInputs` literal sets `Price, DailyBars, Hour1Bars, Min5Bars, RVBaseline20d, PriorClose, SessionOpen`. **`VIX` is never set** — the comment on `:2203` says so out loud ("VIX stays honest n/a (no feed)"). `kernel/regime.go:94` guards `vixBucket` behind `if in.VIX > 0`, so **`vixBucket` has zero reachable production evaluations**. [A]

Live corroboration: the dark-regime alert body (`kernel/regime_dark.go:80-90`) fires only when something is dark, and across `vl_2026-09-03.log` + `vl_2026-09-04.log` **16 of 16 alert lines read `1/7 regime fields unavailable (vix_level)`** — `vix_level` is the only dark field, every time. [A]

C3's `atrBucket` is live and reaches the planner prompt as `BiasRegime = "<trend_daily>/<atr_regime>"` (`auto_trader_planner.go:2443`, field declared `store/planner_read_facts.go:40`). Advisory, no gate. Conforms.

---

## C4 — swing structure

All three knobs resolve to their defaults (env unset): **k=2, min-swing 0.25×ATR, MSS body 1.5×ATR** — byte-identical to the boot ledger line `structure engine TFs=[5m 15m 1h] (5m/15m/1h, swing k=2, min-swing 0.25×ATR, MSS body 1.5×ATR)` (`kernel/regime_ledger.go:14`). Conforms.

Two notes for the owner, neither a conformance failure:

- `2026-08-30-knob-census.md:65` labels the whole trio **[C]** (speculation) while `belief-census.md:59` gives swing-k **[T]**. The [T] rests on `weekend-audit-p2.md:48` ("KEEP 2, best 7/9 days"); `grand-audit-bcde-verdict.md:70` measured the opposite direction — *"k=2→3 cuts swing count 72→46 (5m) and 29→18 (15m) with missed-turns ~93% unchanged → k=3 halves churn for free"*. The two live side by side, unreconciled.
- The **other** consumer of these knobs, the transition stand-down gate (`kernel/engine_position.go:275`), has fired **0 times in every `data/vl_2026-*.log` file** (`TRANSITION STAND-DOWN`, all-time count = 0), while the boot ledger reports `transition_standdown=ON cap=45min`. Not in my C-list; flagging it because it shares C4's inputs and looks like a second never-fires gate.

---

## C5 — stale_reeval

Resolved **0.25 × ATR(14, 5m)** (`STALE_REEVAL_DRIFT_ATR` unset → `defaultReevalDriftATR`, `trader/discard_burn.go:38`). Real gate with teeth: the refusal path stamps `stale_reeval_refused` and returns without executing (`trader/auto_trader_loop.go:796-801`).

Live, all log files: **`stale_reeval outcome=pass` = 19, `outcome=refused` = 15 (n=34)**. [T] Conforms — and it is the only C-rule whose live n is large enough to argue about.

Note the semantics the boot line already states: `armed fill … (entry_class=armed_fill — stale_reeval NOT applied)` — resting-limit fills bypass this gate by contract (`belief-census.md:56` A10).

---

## C6 — grace and tolerance are **two different effect classes**, and the census flattens them

- **C6a, stale-bar grace = 15 s** (`kernel/stale_data.go:46`). Genuine **REJECT**: `applyStaleDataBlock` rewrites `d.Action = "wait"` and stamps `stale_data` (`stale_data.go:157-162`). Live fires 09-01…09-04: **0**. Conforms.
- **C6b, C2 tolerance = 60 000 ms** (`kernel/clock_drift.go:29`). **NOT a gate.** `applyClockDriftBlock` (`clock_drift.go:62-88`) logs `⚠️ clock-drift DETECTED (no entry block)` and calls `telemetry.IncClockSkewObserved`; it never touches `d.Action`. The function's own doc comment says so: *"Since 2026-08-18 outgoing NT8 signals are stamped with the FEED clock … The guard no longer converts entries to wait."*

  Live: **25 `clock-drift DETECTED` lines** on 09-01 (3), 09-02 (10), 09-03 (12) — **0 entries blocked**. [A]

  `belief-census.md:61` labels the row "gate". Its own cited research is the reason it is not one (`2026-08-19-total-e2e-investigation.md:2` — the C2 guard *was* converting every entry to `wait`, which is exactly what the feed-stamp fix removed). **Fix owner: ruling** (correct the census row to WARN-only). The escalation layer that *does* gate is a different rule — F6 clock-hold, which defers **plan authoring**, not entries (`kernel/clock_drift.go:88-107`, consumed at `trader/auto_trader_planner.go:877,1028`).

---

## C7 — fast-market is not a wake trigger, and its "0 live triggers" is stale

`fastMarketATR()` resolves **1.5** (`trader/auto_trader_loop.go:78-87`, `FAST_MARKET_ATR` unset). `fastMarketDrift` (`auto_trader_planner.go:1232-1247`) measures |price − last plan-write price| against `1.5 × ATR5m`. Its three production consumers:

1. `auto_trader_planner.go:900-905` — sets `input.FastTape` + `FastTapeNote` and **downgrades the reasoning wire** (`fastMarketReasoningWire`, default `fast`). Prompt effect.
2. `auto_trader_wake_levels.go:292,299-301` — supplies `FastMarketATR` to the class-47 cadence decision, where it **exempts the cooldown** (`class47_wake_cadence.go:164-165,183-186`). The cutoff is explicitly *not* exempted.
3. `class47_wake_cadence.go:223` — boot line only.

Nothing in that set *starts* a wake. **`belief-census.md:62`'s effect column ("wake trigger") is wrong**: it is a wake **modifier**. Fix owner: ruling/census text.

**Re-measurement (A17).** `weekend-audit-p2.md:33` and `:48` KEEP 1.5 on the grounds of **"0 live triggers … EVENT-WAIT"**. That is no longer true: **35 `🧠 planner mode: fast-market` lines** across 08-30 (4), 08-31 (2), 09-01 (10), 09-02 (17), 09-03 (2), and **1** `cooldown bypassed: fast market` line. [T] The event the docket was waiting for has happened; the KEEP is now decidable on real n.

---

## C8 — "advisory, zero gates" **VERIFIED TRUE**, plus one dead function

All four thresholds resolve to their defaults and match the boot line `touch telemetry: band=16t(4.0pt) max_bars=12 vol_lookback=20 approach=5`.

I enumerated **every production consumer** of touch telemetry (`TouchUpdate`, `ActiveTouchEpisodes`, `TouchBandPoints`, `store.TouchEpisodes()`). All 7, and what each does:

| # | consumer | what it does | gates? |
|---|---|---|---|
| 1 | `kernel/engine_analysis.go:426` | `TouchUpdate(...)` → closed episodes to the sink | no |
| 2 | `kernel/engine_analysis.go:431` | `RenderTouchLines` → executor prompt text | no |
| 3 | `kernel/engine_analysis.go:502` | `RenderScenarioTouchTies` → prompt text (its own comment: *"No gates — the confirm machinery is unchanged"*) | no |
| 4 | `trader/auto_trader_watcher.go:326,331` | `TouchUpdate` + `RenderTouchLines` → watcher prompt | no |
| 5 | `trader/auto_trader.go:688-700` | installs the DB sink (append-only insert) | no |
| 6 | `trader/ninjatrader/level_stats_wire.go:163` | `EpisodeCountByLevel` → NT8 chart level-stats display | no |
| 7 | `trader/no_chase.go:229-238` `lastTouchFor` → `trader/entry_gate.go:349,444` → `:288-293` | feeds `NoChaseInputs.LastTouchPx/HasTouch`; the no-chase leg runs **last** and the function then `return "", false` | **no** — WARN-first by construction |

**Verdict: the claim holds.** Zero touch-telemetry value reaches a refusal. [A] The single *near*-gate is no-chase (`mode=warn`, boot line, `trader/no_chase.go:146`), which has produced **0 `no-chase WOULD_REFUSE` lines** in 09-01…09-04 logs.

**Dead function found (A29).** `kernel.ResolvedTouchBandPoints` — declared as *"the ONE band: k×Δ … so every consumer asks the same question of the same tape"* (`touch_telemetry.go:85-97`) and advertised by `kernel/level_stats_calc.go:23` (*"measurement consumers use `kernel.ResolvedTouchBandPoints`"*) — has **ZERO production callers**. Its only references are `kernel/detector_retirement_test.go:58,59,72`. Every live measurement still uses the legacy fixed `TouchBandPoints()` (16t = 4.00 pts) the D5 comment calls *"wrong on its face"*. The three-geometries-collapse-to-one ruling (D5, 2026-09-03) is documented but **not wired**. Fix owner: code.

---

## Commands used (reproducible, read-only)

```bash
# resolved env (names only, no values printed except the C-knob)
grep -oE '^[A-Z_0-9]+=' /home/hoang/vl/.env | tr -d '='
grep -nE '^HTF_VETO' /home/hoang/vl/.env

# boot lines actually read
grep -h "htf veto: mode=\|regime ledger: htf_veto" /home/hoang/vl/data/vl_2026-09-04.log

# 4h absence across every persisted snapshot
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
SELECT COUNT(*), SUM(structure_json LIKE '%\"4h\"%'), SUM(structure_json LIKE '%\"1h\"%')
FROM decision_records WHERE structure_json NOT IN ('','null');"

# counter-1h-trend entries since the cross cutover
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
WITH e AS (SELECT d.id, datetime(d.timestamp,'-5 hours') ct,
                  json_extract(d.structure_json,'\$.\"1h\".trend') t1h,
                  json_extract(j.value,'\$.action') act, d.risk_check_passed rp, d.risk_check_error re
           FROM decision_records d, json_each(d.decisions) j
           WHERE d.structure_json<>'' AND datetime(d.timestamp,'-5 hours')>='2026-08-28 10:43'
             AND json_extract(j.value,'\$.action') IN ('open_long','open_short'))
SELECT t1h, act, rp, COUNT(*) FROM e
WHERE (t1h='TRENDING_UP' AND act='open_short') OR (t1h='TRENDING_DOWN' AND act='open_long')
GROUP BY 1,2,3;"

# live fire counts
for f in /home/hoang/vl/data/vl_2026-0*.log; do grep -c 'HTF VETO' $f; done
grep -ho "stale_reeval outcome=[a-z]*" /home/hoang/vl/data/vl_2026-0*.log | sort | uniq -c
grep -h "planner mode: fast-market" /home/hoang/vl/data/vl_2026-0*.log | wc -l
grep -ho "regime fields unavailable ([^)]*)" /home/hoang/vl/data/vl_2026-09-0[34].log | sort | uniq -c
```
