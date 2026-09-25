# DEEP STATUS — FULL DETAIL PER ITEM (2026-08-25)

Chat report saved to file per owner request. No code changes.

Sources: repo memory (verified facts), `git log` (54 commits 08-23→08-25), DB read-only queries, journald, docs.
Caveat: journald rotated before 08-25 08:00 CT — 08-24 log-level evidence is DB-derived only.

---

## 1. OWNER DECISIONS LOG (08-23 → now)

| # | What was asked / raised | Owner's ruling | Shipped as | Still open? |
|---|---|---|---|---|
| 1 | G6 loss-streak pause (4/60) — research said 3/30 | **"remove completely"** | `5126e57c` (08-23 08:42 CT): deleted `kernel/loss_streak.go`, `trader/auto_trader_lossstreak.go`+test, engine Context fields, engine_position gate, `regime.loss_streak_n`+resolver, env docs; runbook strike `a620b794` | No |
| 2 | 4.3 EOD limit-then-market (C# wire + env) | **"enable"** | `428d19d3`+`f4a8f83d` (C# fixes), PR #74 `121a917c`+`12e729cf`; **ENABLED 08-23 08:31 CT**: `.env EOD_FLAT_LIMIT_TICKS=2 / EOD_FLAT_MARKET_AFTER_SEC=10`, owner F5+restart NT8, PID 564250 | No |
| 3 | Studio day_plan config | owner set values | `max_levels=12`, `scenario_cap=5`, `proximity_filter_atr=2`, `min_grade=B`, `planner_timeframes=[D,4h,1h,15m]`; **ASIA+LONDON enabled** (`enable:true, min_grade:B, max_trades 7/10` — deviates from research A/1·A; flagged, owner accepted) | No (config live) |
| 4 | C-quality scenario acceptance | owner reported no-plan bug | `b2753f2d`: C accepted in `scenarioQualities` (machine-only; prompt schema stays A+|A|B) | Superseded by #10 |
| 5 | C-gate "B-and-up scenarios only" (`86d1c943`) | **"remove the hardcode only"** | `6549ff00` (08-24 13:47 CT): hardcoded gate removed; engine byte-identical to pre-gate. Owner believed a `min_scenario_quality` setting existed — it does not. No knob ships yet | Yes → pending §6.1 |
| 6 | v3 zone evidence tiers | **"approve v3"** | `00090003` (08-24): `DetectedLevel.TF`+`ZonePattern`; evidence OB .40/.50/.60/.72 · FVG/SD .35/.45/.55/.65 (1m/15m/1h/4h); TFmult 1.0/1.1/1.2/1.3; reversal ×1.1; floors 1m=C, 15m/1h=B cap B, 4h=B→A | No; 1h A-cap upgrade now researched, awaiting go |
| 7 | Grading audit 4 bugs | owner approved fixes | `c1cf4fdb`: unknown-TF zero-score, stamp collision, stale doc, HTF mandate | No |
| 8 | 14-plan flip/death/label audit | — | `e2087c6b` report; `5e620cac` phantom-label rejection; `5e1390e6` flip-bias consistency + prior-level continuity | No |
| 9 | "go 1" Phase-1 machine-grade display | (approved, ran) | shipped inside waves | No |
| 10 | W6 wake wave (iFVG + level wakes + knobs) | **"go"** after 20-min research | `a0678bda` (08-25) | — |
| 11 | Wake fail-closed bug | reported | `cf5d9d96`: wake re-reads non-fatal + async | No |
| 12 | Wake re-plan budget counting | **"unlimited wake session it should not count"** | `93412ce8`: both wake paths budget-free; G5 consumed-at-birth removed; interval 10→30 | No |
| 13 | 1h timeframe upgrade | owner raised "1h important for day trade" | Research only (20 agents, report `e5c9d569`) | Yes → pending |
| 14 | Replays/calibration (swing k, MSS-FVG, HTF_VETO_TF, min-conf, trail) | queued — never run | nothing | Yes → §8 |
| 15 | EOD-flat 14:30 vs 14:45 | never ruled | `EODFlatCT="14:45"`, `EODFlatOffsetMin=15` | Yes → §6.3 |
| 16 | DeepSeek timeouts (300s) | reported twice | env fix proposed (AI_TIMEOUT_SECONDS=600) | Yes → §6.4 |

---

## 2. C1–C14 DETAIL (from `docs/superpowers/plans/2026-08-25-research-findings-implementation.md`)

- **C1** Cross-user IDOR on `/api/plan/*` — mega-research S1 finding (any authed user could read/mutate any trader's plan).
  Fix: `requireTraderOwnership` + 404-not-403 + mounted middleware (`fc9b06a6`, re-verified `cc607909+`).
  Tests: `handler_plan_idor_test.go`. **LIVE.**
- **C2** Owner-level cross-user leak — `owner_levels` had no `user_id`.
  Fix: column + backfill + `ListActiveForUser`/`UpdateNoteTag(userID)`/`DeleteForUser` (`7ef01f5a`). **LIVE.**
- **C3** `getTraderFromQuery` global-trader fallback — could serve another user's trader when none matched.
  Fix: 404. **LIVE** (plus `514e0e86`: public endpoints get explicit no-JWT path after the regression).
- **C4** `/klines` + `/klines/svp` unauthenticated.
  Fix: moved into protected group (`ad915e08` — first attempt missed the route move, caught by verify agents). **LIVE.**
- **C5** MSS wake cap + same-cycle double-append.
  Fix: death-check-first ordering, budget via `MayReplanFrom`, overlay warn+carry on MSS wake (`cc607909`, `e3e9fb0f`). **LIVE** — later superseded by W6-D (wakes now budget-free).
- **C6** Executor-side dead-plan gate.
  Fix: `ExecutorPlanDead` set pre-AI in loop; `ExecutorPlanDeadVerdict` blocks `open_*` (`cc607909`). **LIVE.**
- **C7** Reset baseline trader-scoped.
  Fix: `dayplan_reset:<traderID>:<date>:<session>`; all 7 call sites pass `at.id`. **LIVE.**
- **C8** Entry-rejection reconciliation visibility.
  Fix: NT8 reject branch clears `pending/pendingAt/lastFill/lastEntrySignalID` + alerts. **LIVE**
  (sweep does not clear `lastEntrySignalID` on silently-dropped entries — accepted residual).
- **C9** Re-read races + carryOwnerEdits.
  Fix: honest claim outcomes; carry after reread; budget re-verified against latest row (`85282f8c`, `6f274669`). **LIVE** (TOCTOU narrowed, not eliminated — accepted).
- **C10** Calendar past-date freeze.
  Fix: trade_date guard in `UpdateLiveSliceIfChanged` (`85282f8c`); D2 hardened with caller-side today-guard on the fetch clock (`004d11ce`). **LIVE.**
- **C11** BarCache timeframeMs table.
  Fix: 6h/8h/12h/3d/1w + later "2m" parity (`004d11ce`). Test `TestTimeframeMsAllAutoBarsTFs`. **LIVE.**
- **C12** AddOn cursor persistence — **REVERTED** (`fa546f28`).
  What it was: persist the dedup cursor across Go reconnects so NT8 didn't re-send 2000×14 bars.
  **What broke:** Go restart + persisted cursor → NT8 skipped history → BarCache starved → 1-candle charts after every restart.
  **What's lost with the revert:** nothing functional — the pre-C12 design (full re-seed on reconnect, union-merge by bar time) is correct; the cost is a heavier seed burst (~11.5k frames) and a ~1–3 min cold-cache window after each Go restart (paid twice on 08-25: ASIA plan delay 17:00, and the bar replay after 18:02).
- **C13** Zone-size axis.
  Fix: `zoneSizeMult` bands 0.5–1.25 ATR units (`85282f8c`). Test `levels_score_axis_test.go`. **LIVE.**
- **C14** Confluence cap.
  Fix: `ConfluenceCap()` env `CONFLUENCE_CAP` default 3; `effConf` in both formulas (`85282f8c`). **LIVE.**

---

## 3. D-TRIAGE DETAIL (the 20-agent verification, `004d11ce`)

Full findings list (fixed vs open, sized):

- **D1** orphaned protective bracket after limit-exit close (C# fresh uuid never matched entry bracket) — **FIXED** `122e1b3c` (`CancelBracketsFor` account+instrument fallback); owner did NT8 F5+restart. S.
- **D2** calendar past-date freeze latent (MNQ-only clock issue) — **FIXED** (`004d11ce`, today-guard).
  Related latent: `tick_rounding` vs `futures_symbol` CL/GC divergence — **OPEN**, MNQ-only, S.
- **D3** dead-plan line not in executor prompt — **FIXED** (`004d11ce`, line injected pre-AI after `RenderPlanStatus`).
- **D4** `ExecutorPlanDead` fail-open on empty bars — **OPEN (accepted posture)**, documented, S.
- **D5** reread TOCTOU — narrowed in C9/D6; residual accepted. S.
- **D6** reread budget re-check before claimed read — **FIXED** (`004d11ce`).
- **D7** `/risk/freezes` + gate-blocks + errors visible to any authed user — **OPEN (accepted)**, S.
- **D8** `CleanOldRecords` never wired (decision/equity tables unbounded) — **OPEN**, M (§7).
- **D9** "binary 2 commits stale" — **FALSE ALARM** (C#-only commits don't bump `deploy/RELEASE`).
- **D10** bar_cache "2m" parity — **FIXED** (`004d11ce`).
- **D11** FE gate labels (`approval_required`, `clock_skew_observed`) — **FIXED** (`004d11ce`).
- **D12** comment rot in C# (~line 110) — **FIXED** (`004d11ce`).
- **D13** owner NT8 F5+restart pending — **COMPLETED** by owner ("restart done").

---

## 4. HTF WAVE DETAIL (08-24)

**What changed:** before, every detector ran on the 1m slice only — no real 1h/4h levels existed.
Now `DetectHTFLevels` (G2/G3/G6, `7aa4e3c6`) runs EQH/EQL + S/D + FVG + OB per configured TF (≥15m, skips D),
tags `HTF=true` + `·tf` labels; `seatHTF` promotes ≤2 HTF swing/zones into the top-N; `a0751c94` added the
planner's `## HTF zones` section with the contract "MUST include at least ONE HTF zone row"
(`planner_prompt.go:144-155,216-221`).

**Real observed output** (17:02:26 CT read): 311 levels —
`Demand·1h:7 Demand·4h:12 EQH·1h:16 EQH·4h:16 EQL·1h:16 EQL·4h:14 FVG·1h:4 FVG·4h:2 OB(bear)·1h:15
OB(bear)·4h:22 OB(bull)·1h:15 OB(bull)·4h:26 Supply·1h:6 Supply·4h:8 iFVG(bear)·15m:5 …`
(W6 iFVG tiers included post-deploy). The prompt section template (planner sees this, nearest-first, ≤4):

```
## HTF zones (nearest first — confluence references, NEVER standalone triggers)
  29209.25  Supply·1h     grade B  +42.5 (HTF zone)
  29154.38  OB(bear)·4h   grade A  −12.4 (HTF zone)
```

(executor KEY LEVELS separately gets 1h+4h detection every cycle.)

**Grading fixes (one line each):**
- near-dup collapse (`40d917e9` — 2.13-pt duplicates killed ASIA v2);
- zero-price distance (`a5f42da3`);
- phantom-label rejection (`5f620cac` — LONDON v1 labeled a zone "PDH");
- flip-bias enforcement (`5e1390e6` — ASIA v4 wrote short after a fired "→long" flip).

---

## 5. LIVE REGIME EVIDENCE (08-24/25)

### 08-24 (journald rotated; DB-derived)

- Chains: ASIA v1→v2 fail-closed 17:37 CT→v3 owner_reset 18:02→v4 22:39→v5 01:38 CT.
  LONDON v1 fail-closed 02:07 CT.
  NY v1 08:31 → **v2 `structure_mss` 08:54 CT** (MSS wake fired, correct) → v3 09:52 →
  **v4–v8 five owner resets 10:38–11:19 CT**.
- Trades (7 entries): 5 recorded positions, **day −142.5** (`pnl_corrected` all NULL):
  S2 −139.5, S1 −54.5, S2 +169.5, off-plan −29.5/−88.5.
  Vetoes/standdowns: log evidence lost to rotation.
- Known regime finding (08-24 evening, from owner session): 6 straight red closes incl. Friday, −462;
  two were adherence-A shorts on C/consumed scenarios while price reversed +157 pts → seeded the quality-C decision.

### 08-25

- 18:10:04 CT **HTF VETO fired** on a real entry:
  `🛡️ HTF VETO MNQ open_short: htf_veto: short vs 1h TRENDING_UP (HL 29145.50 @10:00 CT)`
  → decision re-validated (retry 1/3 fed the error back). Correct per regime config (`htf_veto=ON, htf_veto_tf=1h`).
- 17:15 level wake (seated OB invalidation) → read timed out → 17:29 fail-closed (bug, fixed `cf5d9d96`).
  18:03 owner reset → v3; wakes v4–v7 (18:15/18:22/18:45/18:52).
  **19:35:01 v7 DIED** — `flip-condition: 2x5m close below 29199.75 → bias short` → budget exhausted →
  v8 `replans_exhausted` (bug, fixed `93412ce8`; session dark since).
- Gate blocks (session-day 08-25, 15 total): `level_burned_retouch`×7 (alert-only, never blocks — verified code),
  `clock_skew_observed`×3 (real gate), `stale_reeval_refused`×2 (drift `|5.50|≥3.75` and `|12.75|≥4.00` = 0.25×ATR),
  `htf_veto`×1, `plan_matched`×1, `superseded_wait`×1.
- Trades (12 entries, all 12 recorded): **day −78.5**.
  S2×5 (−84.5 / 0 / −7.5 / −27 / −62), S1×3 (−40 / +46.5 / −22.5), S4×1 (−78.5),
  S3×1 (**+81**), S2×1 (**+168**), S1×1 (−52 at 19:11 CT — the sweep-reclaim long stopped out).
  No `pnl_corrected` applied anywhere (all NULL); the 08-25 morning manual-trade reconcile case
  (−87.50, `1a5708d3`) is the one known P&L-recovery event, separate from these rows.
- Refusals quoted: the HTF veto line above; lunch-window waits were the AI honoring the plan's own
  12:00–13:30 CT no-trade (correct).

---

## 6. PENDING OWNER DECISIONS — FULL DETAIL

### 6.1 Quality-C execution treatment

Question: should the executor refuse entries on C-quality/consumed scenarios?
Options:
- (a) keep advisory (current, post-W6-D no write-time demotion),
- (b) ship a settings knob `min_scenario_quality` (A|B|C, default C = no restriction, per-session override like `min_grade`),
- (c) hard gate — already rejected ("remove the hardcode only").

**My rec: (b).** Evidence both ways: C scenarios lost −139.5/−54.5 on 08-24 but +169.5/+81 on 08-25;
no clean edge — knob, not law.

### 6.2 The 8 grading items (audit `2026-08-24-level-grading-full-audit.md` §4)

1. TFmult double-counts tier (effective 4h:1m ≈2.3×) — remove one layer or document?
   My rec: remove `zoneTFMult`.
2. 15m/1h cap B stricter than consensus — now partly settled by the 1h wave decision (1h→A-capable);
   15m stays B per sources. Confirm.
3. Confluence unbounded — **DONE** (C14, cap 3).
4. FVG needs 1×ATR gap — kills 20–80pt sweet spot; my rec: `gap > max(2×tick, noise)` + size weighting.
5. OrderBlocks unbounded lookback — my rec: bound scan to ~8 bars.
6. `seatBothSides` no-op → one-sided executor table — my rec: fill both sides.
7. `minGrade` planner-only — my rec: apply to executor/status/fail-closed maps.
8. Prompt quality string `A+|A|B` hides C — my rec: add C definition.

### 6.3 EOD-flat offset

Current: `EODFlatCT="14:45"` + per-session `EODFlatOffsetMin=15` (NY 14:45 CT).
Question: 14:30 or 14:45?
Evidence: CME cash close 15:00 CT; maintenance halt 15:15 CT; 14:45 = 15 min before cash close,
inside the 14:30–15:00 MOC window. Keeping 14:45 captures late-trend; 14:30 avoids MOC drift.
**My rec: keep 14:45.**

### 6.4 DeepSeek 300s timeouts

Twice on 08-25 (17:15/17:29, ~21:0x). Fix options: env `AI_TIMEOUT_SECONDS=600` + `AI_MAX_RETRIES=2` (no code)
or planner-only longer client (code). **My rec: env bump now + planner-only client as hardening.**

### 6.5 1h wave go/no-go

Research done (§4 + 20-agent report `2026-08-25-1h-timeframe-research-wave.md`):
1h A-cap (OB .60→.70, S/D/FVG/iFVG .55→.65), 1h seat guarantee, prompt mandate,
`seat_1h_zone` knob (default ON), 15m stays B.

---

## 7. OPEN / UNWIRED

- **`CleanOldRecords` unwired** (`store/equity.go:148`, `store/decision.go:351`): decision_records grows
  ~633 rows/day (~19k/month, ~230k/year), equity snapshots + fills 398 total. At current SQLite
  single-writer setup it's benign for months; matters at ~1-year scale (DB size + slow scans).
  Size M. Wire as a daily timer with TTL knob.
- **Exit-fill lineage:** `trader_fills` has entries but no `nt8-exit-*` rows today — 4.2's `recordClose`
  exit writes are not landing on this NT8 path; exits live only in `trader_positions.exit_price`.
  Needs verification. Size S–M.
- **Others:** tick-size table CL/GC vs treasuries divergence (S); backup `integrity_check` wiring (S);
  partner repo vlauto sync (S); `e4-soak.timer` missing — needs `systemd-run --user --on-calendar`
  re-arm (S); demo_seed position id 520 contaminates stats (S); T1 calendar fail-open + B3/B4 bypass
  fixtures absent (S); min_confidence 60 vs 65 (queued calibration).

---

## 8. REPLAY / CALIBRATION STATE

**Inputs still needed:**
1. Owner ruling on swing-window k (research 10–20 vs current 2), MSS-FVG toggle,
   `HTF_VETO_TF` (currently 1h), min-conf (60 vs 65), trail mult (2.0 vs 1.5) —
   loss-streak item **MOOT** (G6 removed).
2. The 08-21/08-22 decision-row replays through the current gate pipeline.
3. 08-24/25 live analysis inputs are all in hand now (this report).

**Can I run now, directly?**
- **Yes for the offline parts:** `nq_smoke prompt` + `roundtrip` (offline-safe, ~seconds),
  kernel goldens (~seconds), and DB-derived 08-24/25 live analysis (already done above).
- **No for the soak harness:** `e4-soak.timer` is missing — I can re-arm it via
  `systemd-run --user --on-calendar` (no sudo, linger is on) and it runs Sun 16:55 CT;
  the 08-21/08-22 calibration replays are script+analysis work I can execute now as a session task.
- Estimated runtime: smokes+goldens minutes; the full calibration matrix
  (swing k sweep × replay evaluation) is a multi-hour agent pass, not minutes.

---

## NEXT ACTIONS (awaiting owner)

1. Settle pending decisions (§6.1 quality-C knob, §6.2 grading items, §6.3 EOD offset, §6.4 timeout bump).
2. Revive ASIA: owner reset (`POST /api/plan/reset`) — with `93412ce8` live the replans_exhausted trap cannot recur.
3. Go/no-go on the 1h A-cap wave.
4. Re-arm `e4-soak.timer` + run calibration replays.
