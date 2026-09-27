# CONTROL RE-VERIFICATION — RUNTIME EVIDENCE (2026-08-19, read-only)

**Branch:** `docs/controls-runtime-verify` · report is the only artifact.
**Trigger:** census (#53) claimed Min-Confidence gates at 65; owner observed a live entry at confidence 62. **The owner is right. The enforced number is 60.**
**Evidence classes:** **[RUNTIME]** = a decision/position/log row demonstrating the behavior · **[DB]** = stored value queried now · **[CODE]** = reading only → per the dispatch standard the verdict stays UNVERIFIED and is marked so.

---

## PHASE 1 — MIN-CONFIDENCE, SETTLED

### 1.2–1.3 The 62 entry and the stored truth

- **Stored value NOW [DB]:** `strategies.config → ai_config.risk_control.min_confidence = 60` (strategy `a5b7662e` "MNQ", the active trader's strategy; row `updated_at` 2026-08-18 01:00:38 UTC = Aug 17 ~20:00 CT). No env var exists for min-confidence.
- **The 62 [RUNTIME]:** position **#523** MNQ SHORT @ 29466.5, entered 09:06:52 CT today, `entry_confidence=62`, from decision record **30084** (14:06:53 UTC, `open_short`, conf 62, success=1). 62 ≥ 60 → legitimately accepted. Siblings today: #521 conf 64, #522 conf 63, #524 conf 65 — all pass at 60, and **#521–#523 would all have been REFUSED if 65 were enforced**. Zero `confidence too low` refusals exist in 4 days of complete file logs and 14 days of decision records.
- **The live prompt agrees [RUNTIME]:** stored `system_prompt` of row 30098 (today 10:23 CT): *"Min confidence to open: 60. Below that, use hold or wait."* and *"`confidence`: 0-100 (open only when ≥ 60)"*.

### 1.1 ALL comparison/consumer sites (the complete chain)

| Site | Number it uses | Source | Role |
|---|---|---|---|
| **kernel/engine_position.go:190-191** `if minConfidence > 0 && d.Confidence < minConfidence` → *"confidence too low (%d), must be ≥%d"* | **60** today | param ← engine_analysis.go:437 `riskConfig.MinConfidence` ← strategy JSON, after ClampLimits (engine_analysis.go:233) | **THE gate — only enforcement site, both venues** |
| store/strategy.go:209-216 `ClampLimits` | 0→**65** (`SafeDefaultMinConfidence`, :75) · <50→50 (:212, const `MinConfidence=50` :58) · >max→max | shipped in `7e5932fc` 2026-08-16 14:56 | mutates the config each cycle; **no-ops on stored 60** |
| **kernel/engine_prompt_futures.go:62-64** `minConf := rc.MinConfidence; if minConf <= 0 { minConf = 60 }` | **60** (own default!) | strategy JSON, pre-clamp copy | futures prompt TEXT — the second threshold constant |
| kernel/engine_prompt.go:93,122,127,164 | raw value | strategy JSON | crypto prompt text |
| kernel/golden_selfcheck.go:39 (75) | 75 | fixture | self-check only, not live |

### 1.4 Behavioral ground truth — every open proposal conf ≤65, 14 days [RUNTIME]

| When (UTC) | conf | Outcome | Explains |
|---|---|---|---|
| Aug 5–6 (9 rows: 50×1, 55×4, 60×4) | 50–60 | **ALL EXECUTED** — decision→position matches within ~3s (e.g. conf-50 rec 24020 02:01:09 → position #357 SHORT 29515.75 02:01:06; conf-55 rec 23808 → #353) | **Era 1:** stored 0 + pre-clamp binary (clamp shipped Aug 16 14:56) → `0 = disabled`, gate off |
| Aug 10–11 (62, 60) | 60–62 | executed | still Era 1 |
| Aug 16 14:56 → Aug 18 01:00 UTC | — | **zero open proposals in the window** | **Era 2 (the "65" era): never exercised.** Clamp live + stored 0 → gate 65 while futures prompt said 60 — a real divergence that produced no observable event |
| Aug 18 20:28–21:33 CT (4 rows conf 62–63) | 62–63 | refused — **`last_entry_cutoff`**, not confidence | different gate; refusal recording works |
| Aug 19 (62,63,64,65 + 67/68/70 band) | 62–65 | **executed** → positions #521–524 | **Era 3: stored 60 (owner's Aug-17 save), gate = 60** |

Confidence-band distribution (all proposals 55–70, 14d): 55×4 · 60×5 · 62×4 · 63×3 · 64×1 · 65×36 · 67×1 · 68×5 · 70×15. Confidence refusals in the whole window: **0**.

### 1.5 VERDICT

- **Enforced number today: 60 [RUNTIME]**, one gate site, both venues. 65 exists only as the clamp default for *unset* strategies and has never gated a live decision.
- **Census error modes (calibration):** (a) **wrong JSON path** — the original probe read top-level `risk_control` (empty) instead of `ai_config.risk_control`, concluded "unset"; (b) **stale DB assumption** — the adversarial verify checked the clamp *code* faithfully but never re-queried the stored value; (c) **twin site missed** — the futures-prompt default (60) was never listed as a threshold site.
- **Real defect found (described, NOT made):** for unset strategies the gate default (65) and the futures-prompt default (60) **disagree** — the AI is told ≥60 and then judged at ≥65, silently discarding 60–64 setups. Fix: align both constants to one number (one-line each side) + Studio helper text "0 → uses default N". **Size S.**

---

## PHASE 2 — REGISTER RE-VERIFIED (census verdict → runtime verdict)

| # | Census row | Runtime verdict | Evidence |
|---|---|---|---|
| 1 | `max_contracts_enabled` toggle DEAD | **AGREE** | [DB] stored `false`; zero readers [CODE]. Caveat: strategy `updated_at` is Aug 18 01:00 UTC — **the owner's "toggled today" has no DB save trace today**; both toggles have been false since that save. Post-OFF behavior unchanged (entries #521–524 all qty 1, same clamp path), but no >2-contract proposal ever probed the clamp → clamp-still-enforces stays [CODE/UNVERIFIED-runtime] |
| 2 | `notional_cap_enabled` toggle DEAD | **AGREE** | same as #1; notional today ≈$59k vs cap ≈$1.05M — cap unprobeable live |
| 3 | Cadence saves on create only (FE drops it on edit) | **AGREE** | [DB] `traders.cadence_mode=""` (pre-P10 create, never persisted); [RUNTIME] boot line `cadence=interval 2m0s` + 2-min cycle spacing = the default path; the FE drop itself re-read by hand (AITradersPage.tsx:257-265) [CODE] |
| 4 | Min-Confidence slider at 0 gates at 65 | **DISAGREE — replaced by Phase 1** | enforced 60 [RUNTIME]; the 65-claim survives only as the unset-strategy clamp default; new defect = 65-vs-60 default mismatch |
| 5 | Daily-loss 0 → env $500 silent fallback | AGREE as code-claim, **moot at runtime** | [DB] stored 450 + `daily_loss_enabled=false` + master OFF → the whole branch is off; fallback untestable live → [CODE/UNVERIFIED-runtime] |
| 6 | Blackout+consistency silently dead under master OFF | **AGREE — UPGRADED to [RUNTIME]** | 69 live `🔍 guardrail WOULD have tripped (master OFF, not enforced)` lines today — **all** from the loss/profit/trades trio (`max daily trades would trip (today=4, max=3)`), zero ever for blackout/consistency. Also proves master OFF end-to-end: the 4th entry today exceeded `max_daily_trades=3` and was NOT blocked |
| 7 | `min_position_size` shadow literals 12/60 | AGREE | [CODE/UNVERIFIED-runtime] — futures sizes ≈$59k never approach the literals |
| 8 | Stale "10-contract" comments (real default 2) | AGREE | [CODE/A] — comment-only |
| 9 | 7 dead `traders` columns | **AGREE, runtime-corroborated** | [DB] hoang's row: `btc_eth_leverage=10` stored while the strategy stores 5 and futures uses neither (prompt: leverage always 1); `system_prompt_template="default"` stored while the getter derives from strategy; `custom_prompt`/`trading_symbols`/`cadence_mode` empty. No stored value surfaces anywhere in today's prompts/behavior |
| 10 | `trading_symbols` update path skips format check | AGREE | [CODE]; column empty [DB] |
| 11 | `NINJATRADER_DATA_DIR` dead + misleading error text | AGREE | [RUNTIME] live transport is TCP (hello handshake, subscription ACKs in today's journal) — the CSV data-dir path isn't even in use |
| 12 | Dead env reads (`RISK_MAX_NOTIONAL_USD`, `RISK_MAX_CONTRACTS_PER_ORDER`, `DATABENTO_DATASET`) | AGREE | [CODE] — nothing at runtime contradicts; unprobeable without env edits (forbidden here) |

**Score: 11 AGREE / 1 DISAGREE (#4).** The disagreement is exactly the class the census could not catch by reading: a **DB-state claim asserted without a fresh DB query**. Its pure code-structure claims went 11/11.

### 2.2 Bundle rulings under runtime evidence

| Ruling | Status | Runtime evidence |
|---|---|---|
| `position_mode` = SAME (wire into `skipWhileOpen`) | **STANDS, strengthened** | [RUNTIME] today's `🧘 skip-while-open: holding 1 open (MNQ SHORT) — AI decision skipped for cycle #33…#42+` lines; `day_plan.plan_enabled=true` [DB]. **New input:** `hold_discipline=true` [DB] yet **zero 🔒 HOLD-LOCK firings in 4 days of logs** — under `bracket_only` the AI is never consulted in-position, so hold-lock is LATENT. `ai_watch` mode is precisely what would make it meaningful; the bundle's help text must say so |
| Trailing stop = CLEAR/RELATED | STANDS | only stop amendments ever = the two `move_stop` breakeven fires (NT8 log, PR #52) [RUNTIME] |
| `POST_EXIT_RESCAN` = RELATED (B7 veto) | STANDS | B7 stored absent → 0=off [DB]; today's same-direction re-entries (07:17 → 09:06 after a loss) show no cooldown, consistent |
| `STALE_DODGE` = RELATED (pre-execution stage) | STANDS, **motivation now measured** | [RUNTIME] AI call durations today 59–166s (rows 30097–30099) — the decision executes up to ~2.5min after its snapshot; nothing re-checks in that window |
| cadence = SAME (fix FE drop only) | STANDS | boot line + 125–131s cycle spacing [RUNTIME] |

### 2.3 Spot-checks of LIVE verdicts (config value ⇄ observed behavior)

1. `scan_interval_minutes=2` [DB] ⇄ `cadence=interval 2m0s` boot + ~125s cycle gaps **[RUNTIME]** ✓
2. `min_confidence=60` [DB] ⇄ prompt "Min confidence to open: 60" + entries 62/63/64 accepted **[RUNTIME]** ✓
3. `ema_periods=[50,200]` [DB] ⇄ prompt "EMA indicators (periods: [50 200])" **[RUNTIME]** ✓
4. `rsi_periods=[14]` [DB] ⇄ prompt "(periods: [14])" **[RUNTIME]** ✓
5. `enable_svp=true` [DB] ⇄ live SVP line "POC 29540.62 VAH 29628.75 VAL 29462.50" in row 30098 **[RUNTIME]** ✓
6. `max_positions=3` [DB] ⇄ futures prompt says "One position at a time" — the census's *prompt-shadowed-on-futures* nuance observed live **[RUNTIME]** ✓
7. `guardrails_enabled=false` [DB] ⇄ 69 would-trip WARNs while the 4th trade executed **[RUNTIME]** ✓
8. `day_plan.plan_enabled=true` [DB] ⇄ skip-while-open lines + plan bias/scenario block in the prompt **[RUNTIME]** ✓
9. `breakeven_enabled=true, trigger 50` [DB] ⇄ two `move_stop` fires today (04:27, 11:36) **[RUNTIME]** ✓ (PR #52)
10. `account=Sim101` [DB] ⇄ today's entries/closes filled on the bound account, no fallback refusals **[RUNTIME]** ✓

No spot-check disagreed. The census's LIVE verdicts hold; its one wrong claim was a DB-state assertion, not a wiring one.

---

## PHASE 3 — PRIORITIZED WIRE-LIST (described, NOT implemented)

| P | Item | Root cause | Fix | Size |
|---|---|---|---|---|
| 1 | **Gate/prompt default mismatch 65 vs 60** for unset strategies | two constants added independently (`SafeDefaultMinConfidence=65` in 7e5932fc vs futures-prompt literal 60) | align to one constant used by both; Studio helper "0 → default N" | S |
| 2 | Cadence edit-PUT drop | FE request body omits the field | add `cadence_mode` to `handleSaveEditTrader` | S (1 line) |
| 3 | CheckSoft blind to blackout/consistency under master OFF | soft-check covers only the trio | extend `CheckSoft` — now runtime-motivated (69 asymmetric would-trip lines today) | S–M |
| 4 | Dead toggles `max_contracts_enabled`/`notional_cap_enabled` | toggles shipped, clamps deliberately always-on | remove toggles or render fixed "always on" | S |
| 5 | Silent $500 env fallback (when master ON + value 0) | `firstPositive` substitution unlogged | one boot log line + helper text | S |
| 6 | Stale "10-contract" comments | comments not updated with the researched 2 | fix 3 comment sites | S |
| 7 | `entry_confidence=0` on 67 of 71 recent positions | column populated only since the P5 wiring | backfill from decision JSON or accept; document | S |
| 8 | Dead trader columns + `NINJATRADER_DATA_DIR` error text | legacy | deprecation sweep | M |

## PR

Report-only PR on `docs/controls-runtime-verify`; number parsed from the `gh pr create` output URL, stated in chat delivery.
