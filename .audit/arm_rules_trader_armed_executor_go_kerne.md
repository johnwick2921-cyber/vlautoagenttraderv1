names rewritten to vl on 2026-09-30 (VL rename)
# ARM RULES — research-conformance re-check at BOOT 8

**Binary under audit:** rev `70af663dcb6f` (boot 2026-09-04 08:30:11 CT, PID 878451). Source read at `/home/hoang/vl-conform` (dev tip `492d2067` + claim commit `fb50903f`). `70af663d` **is** the merge of `fix/arms-follow-bias` — every D1–D6 rule below is IN the running binary, verified by `git merge-base --is-ancestor 70af663d HEAD` = YES and by the boot line quoted in §3.1.

**Measurement window:** 2026-09-04 08:45–08:52 CT. DB read-only via `sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro"`. Live journal: `/home/hoang/vl/data/vl_2026-09-04.log` (last write 08:44:46; process still up).

**`git log -1` for every report cited** (SPEC-FRESHNESS LAW):

| report | `git log -1` |
|---|---|
| `docs/superpowers/reports/2026-09-02-belief-census.md` | `ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 …` |
| `docs/superpowers/reports/2026-09-04-two-day-audit.md` | `f3c640c3f9799e6fa80ce124ae87ee915cad63ed 2026-09-04 07:26:52 -0500 docs(two-day audit D3): why the blindness went unalerted` |
| `docs/superpowers/reports/2026-08-28-grand-audit-bcde-verdict.md` | `741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX` |
| `docs/superpowers/reports/2026-08-28-grand-audit-response-wave.md` | `233c44b0860f080df48e1b433a086f8c0387502d 2026-08-28 12:00:30 -0500 docs(grand-audit-response): post-click verification record` |
| `docs/superpowers/reports/2026-08-31-shadow-demotion.md` | `4be2c73db6d0309bbf4603736e069d55e58a3da4 2026-08-31 17:36:53 -0500 0C cutover record: SHIPPED 17:34:21 CT owner GO` |
| `docs/superpowers/reports/2026-08-25-1h-timeframe-research-wave.md` | `7ce2f77266fe5aa03b9d5e7262771737fc868bdb 2026-08-25 18:42:37 -0500 docs: 1h timeframe research wave — 20-agent parallel dispatch` |
| `docs/superpowers/reports/2026-08-29-weekend-audit-p2.md` | `741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 (archive commit)` |
| `docs/superpowers/reports/2026-08-30-knob-census.md` | `741bfc2a8c443feceaa0f31d30c015946b775633 2026-09-01 07:58:16 -0500 (archive commit)` |
| `docs/superpowers/research/INDEX.md` | `4e8e7e1ae069bc0285f677a316b4771437a39a06 2026-09-03 19:37:14 -0500 docs(index): the stranded-branch sweep` |
| `docs/superpowers/AUDIT-CHECKLIST.md` | `158743dba3751e7528c35b888a3e551d652c00f1 2026-09-04 08:11:57 -0500 (merged into fix/arms-follow-bias)` |

---

## 1. PREMISE CORRECTIONS (A17 — measured first)

1. **"a report named arms-follow-bias"** — there is no such report on dev. `arms-follow-bias` is the **WAVE / BRANCH name**, carried in the file headers (`kernel/arms_bias_coherent.go:1`, `kernel/arm_kind.go:1`, `trader/arm_kind_compose.go:1`, `trader/arm_far_counter.go:1`, `trader/arms_boot_line.go:1`) and in the merge subject of `70af663d`. The D1–D6 rules are documented **in `docs/superpowers/AUDIT-CHECKLIST.md:1786-1818` as class 78**, not in a report. That is the real name; there is no `docs/superpowers/reports/*arms*`. [A]
2. **"round 3: 88% of breaks resolve next candle"** — **could not be located.** No `88%` anywhere in `docs/superpowers/` relates to breaks (the three hits are: 88% of scenarios quality C, 88% of the excursion corpus unreadable, 88% of scenarios poisoned at write). See §4. [A]
3. **`armGateVerdict` is not the production gate.** The dispatch's implied single-arm gate `armGateVerdict` (`trader/armed_executor.go:1305`) has **0 production callers** — all 8 call sites are `trader/armed_executor_test.go:77…189`. Production calls `armGateVerdictFor` at `:430`. [A]
4. **The census's `armed_executor.go:33` anchor for "Arm R:R ≥ 2.0" is stale.** Line 33 is now a comment; the floor moved to `resolvedMinRR`/`armMinRRFor` at `:68`/`:78` and is sourced from `store.ResolveMinRiskReward` (owner ruling R1, 2026-09-03 — `ARM_MIN_RR` deleted, env and code default alike). [A]

---

## 2. THE RULE TABLE

Legend: effect ∈ REJECT / gate / cancel / WARN-only / advisory / prompt / label / **DEAD**.
`conforms` = does the RESOLVED live value match what the research/ruling that grounds it says.

| # | rule | file:line | RESOLVED now | label | grounding (report:line) | effect | conforms? | prod callers |
|---|---|---|---|---|---|---|---|---|
| 1 | **D2 bias-coherence warning** | `kernel/arms_bias_coherent.go:74` | boot line prints `bias-coherent=warn` | [O] | two-day-audit.md:283-284; AUDIT-CHECKLIST.md:1794 | **DEAD — nothing calls it** | **NO** — boot says `warn`, code emits nothing | **0** (tests only: `kernel/arms_bias_coherent_warn_test.go:52`) |
| 2 | D1 armable-vocabulary line in prompt | `kernel/arms_bias_coherent.go:41` | rendered live (§3.2) | [O] | AUDIT-CHECKLIST.md:1808-1811 | prompt | yes | 1 — `kernel/planner_prompt.go:731` |
| 3 | "ARMS FOLLOW THE BIAS … a long plan with no long arm is **invalid**" | `kernel/planner_prompt.go:730` | in prompt verbatim | [O] | AUDIT-CHECKLIST.md:1789-1791 | prompt ONLY | **NO** — prompt says *invalid*, no validator, not one of the 19 restrictions | 1 (prompt build) |
| 4 | Armable condition set | `kernel/armed.go:17` | `{fvg_entry, reject, breakdown_continue, breakup_continue, reclaim}` (+`sweep_reclaim` via the `ArmSpecValid` chain) | [T]/[O] | grand-audit-bcde-verdict.md:59,64 (breakout_retest −$ at every R floor, n=8); arm_kind.go:56-59 (owner ruling, reclaim) | REJECT at write | yes | 2 live — `kernel/plan_doc.go:162`, `kernel/arms_bias_coherent.go:45` |
| 5 | `reclaim` → **stop-entry** | `kernel/arm_kind.go:60-61` | `stop-entry=on(reclaim)` (boot 08:30:11) | [O] | AUDIT-CHECKLIST.md:1814-1815 | REFUSE contradiction / placement branch | yes — but **used 0 times**: 0 of 38 `armed_orders` rows carry `kind='stop_entry'` | 6 — incl. `trader/arm_kind_compose.go:27` |
| 6 | Waterfall = **pullback-primary limit** | `kernel/plan_doc.go:153-155` + `kernel/arm_kind.go:41-49` | `entry_mode=pullback` REQUIRED; kind = `limit`; stop-entry stays the E7 no-retest FALLBACK | [I] | grand-audit-response-wave.md:37-40 (F4 family); no report grounds pullback-vs-stop primacy | REJECT at write | **unknown** — no research names the primacy either way | 1 — `kernel/plan_doc.go:237` (`validateArmSpecs`) |
| 7 | `place_band=100t` | `trader/armed_executor.go:45` | **100 ticks = 25.00 pts** (`ARM_PLACE_TICKS` absent from `.env`) | [I] | knob-census (top-10 unvalidated, INDEX.md orphaned findings) | gate — armed→working trigger | unknown (never swept) | 2 — `:904`, `trader/auto_trader_dayplan.go:65` |
| 8 | `stale_working=15m` | `trader/armed_executor.go:122` | **15 min** (`ARM_WORKING_STALE_MIN` absent) | [I] | none found | cancel | unknown | 2 — `:977`, dayplan:65 |
| 9 | `arm_rr=2.0` | `trader/armed_executor.go:68,78` → `store.ResolveMinRiskReward` | **2.0**, source = Studio `min_risk_reward_ratio` on strategy `a5b7662e` | [T] | belief-census.md:48 (B8, n=18 +$994) | **REJECT** | yes (2.0 = 2.0) — but the census's file:line is stale (§1.4) | 5 — `:1352`, `:1353`, `trader/entry_gate.go:366`, `:412`, `trader/auto_trader_planner.go:1681` |
| 10 | min-SL floor at arm | `trader/armed_executor.go:1361` + `trader/entry_gate.go` leg 6 | **1.5×ATR5m + 2 ticks** (`kernel/min_sl.go:34`, env unset) | [R] | `kernel/min_sl.go:12-25` in-code grounding (15/27 losers stopped-too-tight; 1.5 = bottom of the 1.5–2.5 researched band) | REJECT | **NO vs the census** — census B6 (belief-census.md:46) still says **1.0×ATR5m [I/C]**; live is **1.5 [R]** | 3 gates |
| 11 | `re-arm-after-sweep=on` | `store/armed_orders.go:258-267`, `:378` | ON — only `boot_sweep`-prefixed terminal rows re-authorize under the SAME version | [O] | owner ruling 2026-09-02 quoted in code `:249-257` (position 587) | gate | yes | 5 `UpsertArm` sites |
| 12 | **marketable wrong-side guard** | `trader/armed_executor.go:985` | never place a limit/stop already traded through; cancel the arm | [R] | belief-census.md:49 (B9, 08-30 incident) | cancel (void) | yes — live proof `armed_orders` id 36, 09-03 09:20:47 CT, `"level accepted through — marketable, never placed"` | 2 — `:940` (stop trigger), `:955` (limit) |
| 13 | **class-39 arm normalizer** | `kernel/plan_doc.go:1210` | drop legs on non-`sweep_reclaim`, re-validate, keep single, ⚖ WARN + counter | [O] | owner ruling 2026-09-01 quoted `:1193-1209` | normalize + WARN | yes in shape — **has fired 0 times**: `system_config` key `arms_normalized_class39` **ABSENT** since deploy `aeb11179` 2026-09-02 00:01 CT | 1 — `kernel/plan_doc.go:495` |
| 14 | **UpsertArm re-authorize semantics** | `store/armed_orders.go:171-297` | (a) `working` row → **refuse to rewrite** (D5); (b) terminal **with** signal → new row at `placement_seq+1`; (c) terminal **without** signal → revive in place; (d) same version + terminal + not boot-sweep → **stays terminal** (manual-cancel-wins); (e) side UPPER-cased at the write (class 28) | [R] | belief-census.md:50 (B10, 08-30 incident); D5 in commit `59d01948` | gate | yes — live proof: today's row 38 side = `SHORT` (uppercase), older rows lowercase | 5 in `trader/armed_executor.go` |
| 15 | one-live-arm / **one open position** | `trader/armed_executor.go:640` + `trader/entry_gate.go` leg 7 | BOTH sides refused while a position is open; only `kind:"exit"` escapes | [O] | owner ruling 2026-09-03 quoted `:656-662` | REJECT + cancel | yes | 1 — `:483` (plus the EntryGate twin at `:510`) |
| 16 | 0C shadow demotion at the arm seam | `trader/armed_executor.go:335` | `shadow = {breakout_retest, fvg_entry}` (defaults; `SHADOW_CONDITIONS` absent from `.env`) | [O] | shadow-demotion.md:16,36; grand-audit-bcde-verdict.md:64 | authored + E8-scored, **never placed** | yes | 1 (+ EntryGate leg via `entry_gate.go:378`) |
| 17 | **D4 far-arm counter** | `trader/arm_far_counter.go:47` | threshold **3.0×ATR5m** (`ARM_FAR_ATR_MULT` absent) | [I] PROVISIONAL | two-day-audit.md:877 (D6: 34/48 = 71% authored where the tape never went) | **WARN-only**, nothing refused | yes — live proof 08:46:10 today (§3.4) | 1 — `:543`; **but its counter has 0 readers** |
| 18 | 0B arm stop anchoring | `trader/arm_stop_anchor.go:35,71` | `ARM_STOP_ANCHOR_MAX_ATR` = **3.0** (env absent); widest-wins, never tightens | [I] "CHOSEN DEFAULT, NOT AN OWNER RULING" (`:33`) | week-in-review evidence quoted `:15-18` (15/27 stopped-too-tight) | widens the stop before every downstream gate | **NO on review discipline** — `arm_stop_unanchored_0b` = **196** vs `StopUnanchoredReviewN = 30` (`store/zerob_counters.go:28`): 6.5× past its own review trigger, bound still [I] | 1 — `:390` |
| 19 | invalidation-wired arm gate | `trader/entry_gate.go:358` + EntryGate leg 3 | `invalidation-wired=on`; the evaluator's own `~invalidated` verdict REFUSES; unresolved = pass + a line | [O] | owner ruling 2026-09-03 (`invalidation-wired` report) | REJECT | yes — live proof today: `arm_refusals_0b:…:2026-09-04:LONDON:entry_gate:invalidated = 1` | 1 — `:510` |
| 20 | class-33 boot sweep **before** any arm work | `trader/armed_executor.go:201` | ON, positional guarantee (`:196-200`) | [O] | class 33 wave | cancel-at-boot | yes | 1 |
| 21 | class-47 stale-arm expiry | `trader/armed_executor.go:244` | ON | [O] | class 47 wave | retire never-placed superseded rows | **has fired 0 times** — `arm_superseded_unplaced_class47` **ABSENT** from `system_config` | 1 |
| 22 | split-leg capacity (class 27 FIX 5) | `trader/armed_executor.go:678,688` | capacity **1** (`max_contracts_per_order` unset) → every 2-leg split arm REFUSED at write | [O] | owner-added 2026-08-31, live proof quoted `:314-319` | REJECT | yes | 1 — `:320` |
| 23 | split-sibling stop-out cancel (E4) | `trader/armed_executor.go:758,790` | ON, ±2 ticks of the leg's stop | [I] | none found | cancel | unknown | 1 — `:623` |
| 24 | churn guard | `trader/armed_executor.go:1008` | modify only on ≥ 2 ticks of SL/TP move | [M] | mechanics | modify | n/a | 1 — `:596` |
| 25 | synchronous cancel before flatten | `trader/armed_executor.go:1449` | ack timeout 2000 ms, 1 retry | [O] | deep-verify-22 hole 11, quoted `:1400-1409` | cancel-first wire order | yes | 1 — `:228` |
| 26 | plan_mode=direction honored at the arm seam | `trader/armed_executor.go:1331` | per-SESSION mode (R2 ruling 2026-09-03) | [O] | owner ruling quoted `:1327-1330` | REJECT | yes | in `armGateVerdictFor` |
| 27 | HTF veto honors its own switch at the arm seam | `trader/armed_executor.go:1371` | `htf_veto=ON`, `tf=1h`, `mode=cross` (`.env HTF_VETO_MODE=cross`) | [O] | belief-census.md:56 (C1, P2 sweep KEEP) | gate | yes | in `armGateVerdictFor` |
| 28 | quality floor at arm | `trader/armed_executor.go:1338` | `min_scenario_quality = C` (boot line) → floor effectively inert | [I] | none | gate | vacuous at C | in `armGateVerdictFor` |
| 29 | E7 stop-entry seam | `kernel/entry_law.go:107,113,124` | `STOP_ENTRY_SEAM=on` (.env), offset **2 ticks**, `RETEST_WAIT_BARS=6` | [I] | none found | placement | yes | `trader/armed_executor.go:922,928,932` |
| 30 | legacy single-arm gate wrapper | `trader/armed_executor.go:1305` | — | [M] | — | **DEAD** | **0 production callers** — 8 test call sites | 0 |
| 31 | far-arm counter READER | `telemetry/far_arms.go:37` | — | [M] | class-35 law | **DEAD** | **0 callers** — write-only, in-memory, dies at restart | 0 |
| 32 | shadowed-arm-refusal counter READER | `telemetry/shadow_conditions.go:16` | — | [M] | class-35 law | **DEAD** | **0 callers** — write-only, in-memory | 0 |

---

## 3. THE EVIDENCE

### 3.1 The boot line, READ from the live journal (A11)

```
09-04 08:30:11 [INFO] vl/main.go:430 🎯 arms: bias-coherent=warn · stop-entry=on(reclaim) · far-arm counter=on(3.0×ATR5m) · ledger append-only=on
09-04 08:30:11 [INFO] trader/auto_trader.go:43 ⚔️ armed_orders=on place_band=100t stale_working=15m test_seam=off arm_rr=2.0 (gate-at-arm only; market-entry floor 2.0 unchanged) (resting limits fill at the authorized price; stale_reeval NOT applied)
```

`/api/config/resolved` and `/api/risk/gate-blocks` both return `{"error":"Missing Authorization header"}` from this session — the resolved values above come from the journal and the resolver code path, never a file default presented as live.

**The `🎯 arms:` line is half-literal.** `trader/arms_boot_line.go:1-3` states its own law — *"READ from the same resolvers the code uses, never a literal: a boot line that restates its own source cannot report a change (A24)"* — and then hardcodes `bias-coherent=warn` inside the format string at `:21`. `stop-entry` and the far threshold ARE read (`:18`, `:22`); `bias-coherent=warn` is not, and it is reporting a rule that does not exist at runtime.

### 3.2 The D1 prompt line, rendered from the live resolvers [A]

Run in the worktree (probe test written, executed, then deleted):

```
ARMABLE + LIVE (these are the only conditions an arm can rest on): breakdown_continue→limit ·
breakup_continue→limit · reclaim→stop_entry · reject→limit · sweep_reclaim→limit.
armable but SHADOWED — do not arm these: fvg_entry.
NOT armable at all: acceptance, breakout_retest, hold.
```

So **yes**: the armable set includes `reclaim`, and `reclaim` is the one **stop-entry**. `breakout_retest` renders under *NOT armable at all* rather than under *SHADOWED* (it fails both `ArmableCondition` and `ArmKindFor`), which is the honest ordering — it was excluded by GAR-F4 first and shadowed second.

### 3.3 The bias-coherence rule is DEAD (A29 — loudly)

```
$ grep -rn "BiasArmWarning" --include=*.go .
kernel/arms_bias_coherent.go:74:func BiasArmWarning(...)          ← definition
kernel/arms_bias_coherent_warn_test.go:52,60,80,83,92             ← tests
kernel/arms_bias_coherent_test.go:82,117                          ← tests
```

Zero production callers. `BiasCoherentArmsHint` (`:31`) is likewise reachable only from inside `BiasArmWarning`, and its own doc comment calls it *"the class-34/38 hint registry entry that guards its tokens"* — **it is not in the registry**: `kernel/prompt_contract.go` holds exactly **19** `Site:` entries (matching the boot line `prompt/validator contract: 19 restrictions`) and none is `bias_requires_an_arm`.

Net effect at boot 8: the prompt tells the model *"A long plan with no long arm is **invalid**"* (`kernel/planner_prompt.go:730`) and **nothing checks it — not a reject, not a warn, not a counter**. `AUDIT-CHECKLIST.md:1816` records the fix as shipped *("`BiasArmWarning` + per-side counts")*; only the per-side counts shipped wired.

**Today this was not academic.** NY v2 (08:44:46 CT) is bias `long`, `S1 long sweep_reclaim` armed — and `S1` was refused at the gate 83 seconds later:

```
09-04 08:46:09 ⚔️ arm REFUSED NY S1 leg 1: R:R 0.84 below arm min 2.00 (studio min_risk_reward_ratio) · rr refusals this session: 1
```

leaving a long-biased plan with only its **short** arm live (row 38). That is exactly the class-78 condition, and the journal carries no bias-coherence line because none can be emitted.

### 3.4 D4 far-arm counter — first live fire, and a write-only counter

```
09-04 08:46:10 📏 arm far: S2 short entry 29720.00 is 125.50 pts / 3.5×ATR5m from price 29594.50 (counted, not refused)
```

WARN-only as designed. But `telemetry.FarArmCounts()` (`telemetry/far_arms.go:37`) has **no caller** — no API, no boot line, no log — and the counters are `atomic.Int64` in memory, so they reset at every restart. The same repo states the opposing law 20 lines away in a file shipped two days earlier: `trader/no_chase.go:156-157` — *"RECORDS the outcome per path (class-35 lesson — a log-only tally evaporates at the next boot)"* — and persists via `store.IncSystemCounter`. `telemetry.ShadowedArmRefusalCount()` (0C) is the same shape: incremented at `trader/armed_executor.go:351`, read nowhere.

### 3.5 Live counters that DO persist (`system_config`, read 08:50 CT)

```
arm_refusals_0b:<hoang>:2026-09-02:ASIA:rr                  2
arm_refusals_0b:<hoang>:2026-09-02:NY:rr                    3
arm_refusals_0b:<hoang>:2026-09-03:LONDON:rr                1
arm_refusals_0b:<hoang>:2026-09-03:NY:rr                    1
arm_refusals_0b:<hoang>:2026-09-04:LONDON:entry_gate:invalidated  1
arm_refusals_0b:<hoang>:2026-09-04:LONDON:rr                1
arm_refusals_0b:<hoang>:2026-09-04:NY:rr                    1
arm_stop_unanchored_0b                                    196   (review N = 30)
nochase_arm_ok                                             42
nochase_arm_run_null                                       41
arms_normalized_class39                                   ABSENT   → class 39 has NEVER fired
arm_superseded_unplaced_class47                           ABSENT   → class 47 has NEVER fired
```

Two readings worth flagging:
- **`nochase_arm_run_null` ≈ `nochase_arm_ok` (41 of 42).** The no-chase *run* leg has been unmeasurable on essentially every arm it has ever judged — the arm path judges on `dist` alone. No `nochase_arm_would_refuse` / `nochase_arm_dist_fired` key exists at all → **the no-chase leg has refused zero arms, ever.** Live line today: `09-04 08:46:10 🚫 no-chase: arm S2 has no recorded touch — run stored NULL, judged on dist alone (0.00×ATR)`.
- **`arm_stop_unanchored_0b = 196`** against the owner's own review trigger of 30. Every one of those 196 arms had *no seated level within 3.0×ATR5m on the risk side* and fell to the ATR floor. Today's S2: `🛑 arm stop NY S2 leg 1 short: stop 29773.61 (authored 29770.00 WIDENED) · anchor none (stop_unanchored) · atr_floor 29773.61 (1.5×ATR5m 35.74) · bound=atr_floor`. The provisional bound is 6.5× past the count at which it was to be ruled on.

### 3.6 Ledger census (`armed_orders`, n = 38 rows, all time)

| dimension | value |
|---|---|
| states | `cancelled` 27 · `filled` 10 · `armed` 1 |
| `kind` | `limit` 21 · `''` (legacy) 17 · **`stop_entry` 0** |
| `condition` | `sweep_reclaim` 1 (row 38, written today) · `''` 37 (column added after the fact) |

**The stop-entry path has never placed an order.** `STOP_ENTRY_SEAM=on` in `.env`, `PlaceStopEntry` is wired, `ArmKindFor("reclaim")` now returns it, the boot line advertises it — and 0 of 38 rows carry it. This is precisely the "built ≠ wired ≠ used" pattern the wave was written to close (`AUDIT-CHECKLIST.md:1804-1808`), still open one boot later. The two `reclaim` longs on today's LONDON v1 were authored 01:41 CT, ~7 h before the capability existed.

---

## 4. D7(a) — the "88% of breaks resolve next candle" claim

**Verdict: the source could not be located.** [A]

- `grep -rn "88%\|0\.88" docs/superpowers/` returns three hits, none about breaks: `AUDIT-CHECKLIST.md:1011` (88% of the excursion corpus unreadable), `2026-08-25-plan-flip-death-audit.md:47` (88% of scenarios quality C), `2026-09-03-trade-excursions.md:107` (the same corpus figure).
- `grep -rn "breaks resolve\|next candle\|next bar"` over `docs/superpowers/reports/*.md` returns exactly **one** quantified break-resolution claim, and it is a different number, a different window and a different source:

> `docs/superpowers/reports/2026-08-25-1h-timeframe-research-wave.md:157` — *"TradingStats (6,142 ES/NQ days): 30m ORB consumes 41–46% of daily range; **74% of breaks resolve within 15 min**; median sessions retrace 100–206% of local structure."*

That is third-party literature (TradingStats), about **30m opening-range breaks resolving within 15 minutes** — not "88%", not "next candle", and not own tape. There is no `docs/superpowers/research/` corpus of rounds to check it against: that directory holds only `INDEX.md`. I do not accept the 88% premise; if it exists it is outside this repo.

**Is `breakout_retest` in the SHADOW set consistent with the claim, or contradicted by it?**

Answer: **consistent in direction, but the shadow decision does not rest on it.** [B]

- *Consistent*: a break that resolves fast (74% within 15 min) is a break whose retest often never comes. A play whose entry is a resting limit AT the broken level is, under that statistic, a play that mostly does not fill — and when it does fill, it fills on the breaks that failed. Both live decisions push the same way: `breakout_retest` is un-armable (`kernel/armed.go:12-14`) *and* shadowed (`kernel/condition_status.go:26-29`). The stop-entry FALLBACK (`trader/armed_executor.go:916-931`, fires only after `RETEST_WAIT_BARS=6` bars with no touch) is the machine's own concession to the same statistic.
- *But the live grounding is own tape, not literature.* `kernel/armed.go:12-14` cites GAR-F4 verbatim, and GAR-F4's numbers are:
  - `2026-08-28-grand-audit-bcde-verdict.md:59` — `breakout_retest` R-floor sweep: **n=8 −$42 · n=7 −$54 · n=7 −$54 · n=7 −$54 · n=6 −$14**;
  - `:64` — *"breakout_retest is negative at EVERY floor → exclude from arm authoring"*;
  - `:112(c)` — the owner-facing recommendation;
  - `2026-08-29-weekend-audit-p2.md:45` — `breakout_retest LONDON 0/2 −$148`.
  - Shadow status separately: `2026-08-31-shadow-demotion.md:16,36` (`{fvg_entry: shadow, breakout_retest: shadow}`, defaults).
- **A24 floor:** n=8 → n=6 across the sweep is far below any decision floor. The rule is labelled [T] on that basis, but it is [T]-weak; nothing at n=6–8 distinguishes −$14 from zero. The literature claim (74%, n=6,142 days) is the *stronger* evidence and it is **not** what the code cites.
- **Second-order consequence, measured:** the two-day audit found 18 of the stranded long plans leaning on `breakout_retest` (`AUDIT-CHECKLIST.md:1798-1801`) — un-armable AND shadowed, and until 2026-09-04 08:11 the prompt named neither fact. So the exclusion, correct or not, was invisible to the only party that could act on it.

---

## 5. D7(b) — RE-MEASURE of the arm-enablement asymmetry, 2026-09-04

**Method (reproduces the audit's, verified):** `plans` × `json_each(doc->'$.scenarios')`, numerator `arm.enabled ∈ (1,'true')`, denominator = every scenario of that direction. My query reproduces the audit's 09-02 numbers **exactly** (long 9/64, short 27/43 — `two-day-audit.md:1016`), so the method matches.

| scope | date | long armed/n | rate | short armed/n | rate |
|---|---|---|---|---|---|
| `trade_date` | 2026-09-02 | 9/64 | 14.1% | 27/43 | 62.8% |
| `trade_date` | 2026-09-03 | **2/28** | 7.1% | **8/20** | 40.0% |
| `trade_date` | **2026-09-04 (so far)** | **2/6** | **33.3%** | **2/6** | **33.3%** |
| CT-created-day | 2026-09-04 (so far) | 3/11 | 27.3% | 2/8 | 25.0% |
| report claim | 2026-09-03 | 1/23 (4.3%) | — | 8/18 (44.4%) | — |

**Premise correction on the 09-03 figures (A17).** The audit reports long **1/23** and short **8/18** (`two-day-audit.md:283-284`). Re-measured against the same table today I get **2/28** and **8/20** — the shorts numerator agrees, the two denominators and the long numerator do not. Under CT-created-day scoping I get long **1/24** and short **10/23**, which reproduces the audit's long numerator but not its short. The audit's exact scoping (which plan versions were counted) is not recoverable from the report. **Both scopings preserve the finding's direction and both are far outside its stated precision**, so the asymmetry claim stands while the specific ratios `1/23` / `8/18` do not reproduce and should not be re-quoted as exact.

**Is the asymmetry still present today? NOT AT THIS n — and no verdict is available (A24).**

- 2/6 vs 2/6 is perfectly symmetric, and 33.3% long is nominally 8× the 09-03 long rate. But n = 6 per side. A 95% Wilson interval on 2/6 spans roughly **[0.10, 0.70]** — it contains 14.1% (09-02) and 40.0% (09-03 shorts) alike. **Six scenarios cannot separate any two of these rates.** No verdict below its floor.
- **Attribution is worse than the n suggests.** Of the 12 scenarios on 2026-09-04, **10 were authored before boot 8** (08:30:11 CT). Exactly **one plan version — NY v2 at 08:44:46 CT — was written by the new binary and the new prompt**: bias `long`, `S1 long sweep_reclaim` ARMED, `S2 short sweep_reclaim` ARMED. So today's symmetry rests on **n=1 plan version, 2 scenarios**. Whatever the wave did or did not change, this measurement cannot see it.
- **And the one post-wave plan armed neither of the newly-armable plays.** No `reclaim` arm, no stop-entry; both arms are `sweep_reclaim` limits, and both were flagged infeasible at write:
  ```
  08:44:46 ⚔️ arm feasibility: S1 arm R:R 0.84 below min_risk_reward_ratio 2.00 (Studio) …
  08:44:46 ⚔️ arm feasibility: S2 arm stop 29770.00 too close (50.00 < 52.92 = 1.5×ATR5m) …
  08:46:09 ⚔️ arm REFUSED NY S1 leg 1: R:R 0.84 below arm min 2.00 …
  ```
  The long (the bias direction) was refused; only the short reached the ledger as row 38. **Arm-enablement rose to 33% on both sides and the number of arms that survived the gate in the bias direction is zero.** Counting `arm.enabled` in the doc measures what the planner *authored*, not what can *trade* — the gate is downstream of the metric, and on the single post-wave plan the two disagree completely.

Per-scenario detail is written to `d7-0904-scenarios.csv`; a sibling agent's independent measurement (`arm_enablement_by_side.csv`, same directory) agrees row-for-row on 09-02/09-03/09-04.

---

## 6. DEAD RULES (A29 — 0 production callers)

1. **`kernel/arms_bias_coherent.go:74 BiasArmWarning`** — the D2 bias-coherence rule the boot line advertises as `bias-coherent=warn`. Test-only. **The single most consequential finding here.**
2. **`kernel/arms_bias_coherent.go:31 BiasCoherentArmsHint`** — reachable only from (1); claims registry membership it does not have.
3. **`trader/armed_executor.go:1305 armGateVerdict`** — legacy single-arm gate wrapper; 8 test call sites, 0 production. Class-53 note: the arm-gate suite drives this wrapper, not the production call site at `:430`, so it exercises the legacy leg shape rather than the split-leg shape production passes.
4. **`telemetry/far_arms.go:37 FarArmCounts`** — the D4 counter's only reader. Nothing reads it; the counts are in-memory and die at restart.
5. **`telemetry/shadow_conditions.go:16 ShadowedArmRefusalCount`** — same shape, 0C's counter.

Not dead but **never fired in production** (distinct from dead — wired, reachable, zero events):
- class-39 arm normalizer (`arms_normalized_class39` absent since 2026-09-02 00:01 CT);
- class-47 stale-arm expiry (`arm_superseded_unplaced_class47` absent since 2026-09-02 21:19 CT);
- the E7/D3 stop-entry placement branch (0 of 38 ledger rows);
- the no-chase leg on the arm path (0 `would_refuse` in 42 judgements).

---

## 7. DRIFT REGISTER

| # | rule | research / ruling says | live says | owner of the fix |
|---|---|---|---|---|
| 1 | bias-coherence | `AUDIT-CHECKLIST.md:1816` records `BiasArmWarning` as shipped; boot line prints `bias-coherent=warn` | 0 production callers — nothing warns | **code** |
| 2 | `🎯 arms:` boot line | `trader/arms_boot_line.go:1-3`: "READ … never a literal" | `bias-coherent=warn` is a literal in the format string `:21` | **code** |
| 3 | min-SL at the arm | belief-census.md:46 (B6): **1.0×ATR5m**, label [I/C] | **1.5×ATR5m** (`kernel/min_sl.go:34`), now [R]-grounded at `:12-25` | **ruling** (re-label the census row) |
| 4 | min-SL in the planner prompt | live gate = 1.5×ATR5m | `kernel/planner_prompt.go:733` still tells the model **"≥ 1.0× the current 5m ATR"** — the model is authoring to a floor 50% below the one that judges it (3 `arm feasibility … too close` WARNs today) | **prompt** |
| 5 | R:R knob name in the prompt | R1 ruling 2026-09-03 **deleted** `ARM_MIN_RR`, env and code default alike (`trader/armed_executor.go:59-67`) | `kernel/planner_prompt.go:733` still says **"must be ≥ 2.0 (ARM_MIN_RR)"** — names a knob that no longer exists | **prompt** |
| 6 | "ARMS FOLLOW THE BIAS … invalid" | prompt states a MUST | no validator, no warn, not among the 19 restrictions | **code or prompt** (a MUST with no check is the class-38 shape) |
| 7 | B8 arm-R:R anchor | belief-census.md:48 cites `trader/armed_executor.go:33` | floor now at `:68`/`:78` via `store.ResolveMinRiskReward` | **ruling** (census line stale) |
| 8 | 0B dead-zone bound | `trader/arm_stop_anchor.go:33` "reviewed at n≥30" | `arm_stop_unanchored_0b = 196` — 6.5× past the trigger, bound still [I] | **ruling** |
| 9 | far-arm + shadow counters | class-35 law: "counters RECORD; log-only tallies evaporate at the next boot" (`trader/no_chase.go:156-157`) | both are in-memory atomics with zero readers | **code** |
| 10 | in-code comment | `kernel/min_sl.go:43` "default 1.0; 0 = gate off" | constant at `:34` is **1.5** | **code** (comment) |
| 11 | 09-03 arm-enablement ratios | two-day-audit.md:283-284 → 1/23 and 8/18 | re-measured 2/28 and 8/20 (`trade_date`), or 1/24 and 10/23 (CT-created-day) | **ruling** (direction holds; exact ratios do not reproduce) |

---

## 8. UNMEASURABLE / NOT ANSWERED

- `/api/config/resolved` and `/api/risk/gate-blocks` — `{"error":"Missing Authorization header"}` from this session. Every RESOLVED value above therefore comes from the boot journal or the resolver code path.
- Whether the D2 warning was *intended* to be wired and was dropped, or was deliberately left as a library for a later ruling — the commit message (`59d01948`) claims D6 wires the boot line but never claims D2 is wired; `AUDIT-CHECKLIST.md:1816` claims the fix included it. Only the owner can rule which.
- Whether the boot-8 wave moved arm-enablement: **n=1 post-wave plan version**. Not answerable before several sessions have run on `70af663d`.
- The 09-03 audit's exact plan-version scoping (which produced 1/23 and 8/18) — not recoverable from the report text.
- `place_band=100t`, `stale_working=15m`, `RETEST_WAIT_BARS=6`, `STOP_ENTRY_OFFSET_TICKS=2`, split-sibling 2-tick tolerance — all [I], no report grounds any of them and none has been swept.
- Whether `breakout_retest`'s exclusion is *right* — n=6–8 is below any floor; the only large-n evidence (74%/6,142 days) is literature the code does not cite.

## 9. COMMANDS THAT PRODUCED THE NUMBERS

```bash
sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "
SELECT trade_date, lower(trim(json_extract(s.value,'\$.direction'))) dir,
       COUNT(*) n, SUM(CASE WHEN json_extract(s.value,'\$.arm.enabled') IN (1,'true') THEN 1 ELSE 0 END) armed
FROM plans p, json_each(json_extract(p.doc,'\$.scenarios')) s
WHERE trade_date IN ('2026-09-02','2026-09-03','2026-09-04') GROUP BY trade_date, dir;"

sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "SELECT key,value FROM system_config
  WHERE key LIKE '%arm%' OR key LIKE '%nochase%';"

sqlite3 "file:/home/hoang/vl/data/data.db?mode=ro" "SELECT kind,COUNT(*) FROM armed_orders GROUP BY kind;"

grep -h "🎯 arms:\|⚔️ arm\|📏 arm far\|🛑 arm stop\|🚫 no-chase" /home/hoang/vl/data/vl_2026-09-04.log

# the rendered D1 prompt line (probe test written into the worktree, run, then removed):
go test ./kernel/ -run TestPrintArmableLine -v
```
