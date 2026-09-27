# Weekend Deep Audit — Part 1/2: MACHINE LAYER (synthesis, 2026-08-28 → 08-29 CT)

- **Orchestration:** 5 parallel read-only agents (A1 wire · A2 bars · A3 detectors · A4 execution · A5 plan engine), each in an isolated worktree (`~/nofx-a1..a5`) at the DEPLOYED rev `8666db0b`. Main tree + live bot untouched. Market closed (Friday 16:00 CT) — "no live tape" is a valid state; nothing faked.
- **Tooling disclosure:** the agents' sessions shipped WITHOUT terminal/file-write tools, so DB/SQL/journal metrics were frequently UNVERIFIED (they used the on-disk `data/nofx_*.log` file logs, deployed-rev source, and live HTTP). The orchestrator independently re-confirmed the top BROKEN claims with terminal access (marked **[O]** below). Nothing marked UNVERIFIED was re-used as a finding.
- **Rules:** R1 fresh evidence this run · R2 independent math · R3 twin long/short · R4 file:line · R5 S/A/B/C · R6 PROVEN/EVENT-WAIT/BROKEN/UNVERIFIED · R7 pnl_corrected · R8 trader binding (strategy `a5b7662e`) · R9 1m-bar replays. Part 2/2 (strategy/money/calibration) is OUT of scope — no expectancy tables, no $-replays, no weight rulings here.

---

## S-LIST (ranked, cross-agent)

| # | Sev | Finding | R6 | Evidence |
|---|-----|---------|----|----------|
| S1 | **A** | **The waterfall play is parse-rejected: `scenarioConds` (`kernel/plan_doc.go:225`) has only 7 conditions and `ValidatePlanDocWithCaps` (`:484`) hard-rejects `breakdown_continue`/`breakup_continue` BEFORE the waterfall validator (`auto_trader_planner.go:1240`) runs.** The prompt mandates the 8th condition (`planner_prompt.go:514`), the test suite bypasses the schema gate (direct validator calls) — so the shipped play will fail-close its first live authoring (3×~500s retries → no-trade). **Orchestrator re-verified [O]: `plan_doc.go:225` = 7 conditions; `:484` hard-fails non-members.** | **BROKEN** (shipped-inert; EVENT-WAIT: first authoring, ASIA 16:55 CT read) | A5 |
| S2 | **A** | **"closes_dropped must be 0" violated once post-fix: 8 CLOSED bars dropped 08-28 11:20:06 CT** (`bar_persist.go` stall 2s→4s→6s, [ERRO] at `data/nofx_2026-08-28.log:10917`); healed by the 11:24:46 boot backfill (31,988 bars). Root cause = a ~2h08m GORM persist stall (09:12–11:20) that also pinned `peak_depth=4096/4096` ×7 with ~38k intrabar drops. Pre-fix era: 86,400 bars dropped over 12 events. **Orchestrator re-verified [O]: the 11:20:02/04/06 stall WARNs are in journald.** Fix the stall (GORM writer blocking), not the accounting. | **BROKEN** (one post-fix event; self-healed) | A2 |
| S3 | **B** | **Post-fill armed-ledger re-log spam + dead re-arm path:** terminal rows re-log `⚔️ armed` every cycle (69+ lines today: NY S1 10:53:45→12:05:01, NY S2 12:13→12:59) and a legit same-scenario re-arm is silently impossible (`store/armed_orders.go` UpsertArm keeps the terminal state). | PROVEN (journal) | A4 |
| S4 | **B** | **Planner latency at the ceiling: p95=600s with 2× context-deadline hits (12:41:05, 12:51:06)** — the NY v5→v6 rereads timed out. Max completion 35,126/65,536 tokens (54% headroom) — the cap fix holds, the 600s HTTP ceiling is the binding constraint. | PROVEN | A4/A5 |
| S5 | **B** | **Gate-change cancel missing (comment-vs-code):** `armed_executor.go` promises "gate input changes materially → cancel (1.3)" but no such pass exists — LONDON S4 stayed armed through repeated re-refusals (07:43:22→08:30 sweep). | PROVEN (code+journal) | A4 |
| S6 | **B** | **Fill-time lineage stamp race: all 4 live fills logged "no open position row to stamp"** (fill frame precedes position materialization); `RepairArmedLineage` stamped only 1 row at the 07:39 boot → the other armed-fill rows' lineage completeness is DB-UNVERIFIED. | PROVEN (race observed) / lineage UNVERIFIED | A4 |
| S7 | **B** | **`KindForLabel` (`kernel/levels_role.go:303-381`) has no SWG-H/SWG-L branch → unknown → `KindRound` → role `pivot`** for label-only lookups (live proof: 07:48:28 WARN `SS3 cites SWG-H·5m (pivot)`); `level_stats` family bucketing inherits the misbucket. Seated rendering unaffected (uses DetectedLevel→react_zone). **Orchestrator re-verified [O].** | PROVEN | A3 |
| S8 | **C** | **Clock drift −38.8s to −59.3s (Go behind NT8) today** — all past CLOCK_WARN (30s), none past tolerance (60s). Root fix: chrony in WSL (`pool time.windows.com iburst` + `makestep 1 -1`) or a root `hwclock -s --utc` timer; WSL2 mirrored mode syncs at boot only. | PROVEN (numbers) / fix = recommendation | A1 |
| S9 | **C** | **Hysteresis rearm half is unexercised: 0 rearms in 7d** (4 dormants all hysteresis-quoted; the mirror rearm never fired live). Coverage one-sided. | EVENT-WAIT (any rearm) | A5 |
| S10 | **C** | Storm-guard boundary rounding: `SKIPPED: 30m0s elapsed < wake_min_interval_min (30)` (cosmetic). | PROVEN | A5 |

**Sound items worth recording (all PROVEN):** every Go→C# send path has 5s write deadline + nil check (13 sites) · deployed C# AddOns byte-identical to repo HEAD · dead-man cycles clean (today's 4 DOWNs = boot-order artifacts) · Friday ingest gap-free at exactly 2 bars/min · `pnl_corrected` recompute exact on the two newest closes ($63.00, $17.00) · proximity band flipped ±458→±85pt at exactly 11:59:00 CT (0.3 × proxy 283.33) · weight tables = README §2.3 zero drift · move_stop/BE resolver LIVE on #570 (BE 12:47:05 + 5 trail ratchets + SL exit +$17) — the GAR-F1 fix has a live wire proof · cancel-before-flatten live at the 08:30 boundary (`✕ armed S2 cancelled in NT8` → `2 order(s) disarmed`) · 0 `finish_reason=length` post-fix · C6 executor-plan refusals 0 since 08-26 · level_stats all three due nights landed (0→28→74→101 rows).

---

## Per-agent one-line verdicts

- **A1 Wire:** sound at code+artifact level (deadlines, identity echo-verify, AddOn/Go lockstep) — journal-only metrics UNVERIFIED (no terminal); no BROKEN.
- **A2 Bars:** structurally sound (1m-only, open-stamped, dups=0, gap-free, exact newest PnL, clean Friday close) — but S2 (closes_dropped=8 post-fix) and both queue generations hit cap during a ~2h GORM stall; fix the stall.
- **A3 Detectors:** 19/19 formulas EXACT at deployed rev, 0.3 retune live-verified end-to-end, weights zero drift — one S (SWG label gap); bars/DB recomputes UNVERIFIED.
- **A4 Execution:** core sound (legal state machine, 0 orphans, cancel-before-flatten live, move_stop on armed fills live) — journal spam/dead re-arm, missing gate-change cancel, and the 600s planner ceiling need a fix wave.
- **A5 Plan engine:** ONE BROKEN (S1 — 8th condition unparseable), rest PROVEN (validators, aliases, truncation fix, hysteresis dormants, wakes + storm guard, dormant-keeps-eyes, two-leg render, C6).

## Cross-agent contradictions (called out)

1. **Pool census vs C6 prediction (A3 vs my C6):** C6 predicted "pool ~5–6 at 0.3×" PER PLAN DOC; A3's live census = **12 seated (max_levels cap) of 51–192 in-band candidates** (the full generated universe). Not a data error — a measure-definition mismatch. Resolution: C6's number was the per-plan-doc in-band count; live seating always fills the 12 cap.
2. **Armed-row count:** A4 reconstructed ≥11 real rows from journal; the DB table has 9 (4 TEST-E2 + 5). DB read is UNVERIFIED for A4 — DB agents should settle at 9 rows with journal-reconstructed events as derived history.
3. **Newest stored planner prompt (A5-S2):** the newest `decision_records.system_prompt` is a PRE-waterfall render (NY v7, 13:08:51 CT, prompt `475f94b5c4f5`) — the 8-condition contract exists in CODE but not yet in any stored prompt. Any quote claiming otherwise is wrong.
4. **A1 dead-man nuance:** all 4 of today's DOWNs coincide with BOOT INTEGRITY lines (boot-order artifact), not live link drops — contradicts a naive "4 link failures today" reading.
5. **Token file:** `/tmp/token.txt` expired (exp=1787799515) — agents needing auth must re-mint via `/tmp/mint_token.py`.

## EVENT-WAIT register (exact events)

| Item | Event that completes it |
|---|---|
| F6 refusal-dedup live proof | next live arm refusal on `8666db0b` (market closed) |
| FIX-1 session-end cancel-first with a WORKING arm | a working arm at a session/EOD boundary (14:45 had armed-only S3) |
| A5-S1 failure (or fix) | ASIA 16:55 CT read / first waterfall authoring — WILL fail-close unless `scenarioConds` is fixed |
| 3-way bars arbiter (NT8 leg) | authenticated `/api/nt/bar-arbiter` diff when NT8 is live (Sun 17:00) |
| move_stop DB-fallback path (path 3) | a materialized non-signal position firing BE/trail (paths 1–2 proven; #570 used path 1) |
| Hysteresis rearm half | any `⚡ REARMED` line |
| DB recomputes closed by terminal (orchestrator/next run) | VWAP residual @0.046pt · missed-turns @0.3 · per-row grade recompute · stamp-gap count · freshness transitions 7d · 100-row open-stamp · newest-20 pnl recompute · touch_episodes 50-sample · journald bytes/day + integrity_check |

## MACHINE-LAYER SOUND? — **NO (2 BROKEN), but one line each**

- **S1 (waterfall unparseable) is the machine-layer verdict-breaker:** a shipped play that cannot pass its own schema gate is the worst class — fail-close is silent-by-design here (3 retries then no-trade), so it would burn a session's read budget at the worst moment (the exact −347pt scenario). One-line fix: add the two conditions to `scenarioConds` (`kernel/plan_doc.go:225`).
- **S2 (closes_dropped=8 + GORM stall) is an infrastructure reliability break:** self-healing worked (backfill), but the ~2h writer stall is the root cause and unowned.

**Evidence chain for YES/NO:** S1 [O] re-verified at `plan_doc.go:225/484` + prompt `planner_prompt.go:514` + write path `auto_trader_planner.go:1139→1240` · S2 [O] journald stall WARNs 11:20:02/04/06 + A2's [ERRO] drop line + 31,988-bar backfill. Everything else PROVEN-by-fixture or live, nothing else BROKEN.

**Recommendation:** a small hotfix wave (scenarioConds + armed-Upsert state reset + gate-change cancel + GORM stall diagnosis) before Sunday's reopen; Part 2/2 (strategy/money/calibration) unchanged by this report.

---

## Agent sections (full, as written)

### A1 — WIRE & TRANSPORT

S-list: S1 every Go→C# send path has 5s write deadline + nil-check (13 sites: `provider/ninjatrader/tcp_server.go:762,848,1044,1062,1079,1096,1113,1421,1444,1683,1748,2005,2080`) — PROVEN. S2 deployed C# AddOns byte-identical to repo HEAD 8666db0b (all 3 files) — PROVEN. S3 running binary = 8666db0b (boot 15:22:35, goldens PASS) — PROVEN. S4 dead-man DOWN→UP→RESUMED clean; entry REFUSED during gap (08-26 01:25:47) — PROVEN; today's 4 DOWNs coincide with BOOT INTEGRITY lines (boot-order artifact, not link drops). S5 backup timers + 05:00/17:30 gz artifacts 7d — PROVEN. S6 zero `database is locked|busy` and zero echo-mismatch/freeze in 7d file logs — PROVEN. Frame census: 26 types enumerated (`tcp_framing.go`: Go→C# signal/hello/ack/heartbeat/cancel_order/modify_bracket/bars_subscribe/bars_unsubscribe/account_select/close_position/account_register/move_stop; C#→Go fill/order_update/subscribed/unsubscribed/subscribe_error/bars_historical/bar_update/account_balance/accounts_list/position_close/position_close_rejected/feed_status/instrument_info/positions) — 7d counts+freshest-age UNVERIFIED (tcp_server logs via slog→journal only). Clock drift: −37755/−58943/−59295/−38713ms @07:39/08:30/11:59/14:45 CT (WARN-class, all <60s tolerance); clock-guard state ntp_offset=+139ms, resync=unavailable-no-root. Fix rec (sized): chrony in WSL (~3MB, `pool time.windows.com iburst` + `makestep 1 -1`) or a root `hwclock -s --utc` 10-min timer. systemd: Restart=on-failure, RestartSec=5, StartLimitIntervalSec=0; 7d restart counts UNVERIFIED; boot census = owner cutovers (4 today, 4 different revs), not crashloops. journald: Storage=persistent 2G cap, RateLimitBurst 200000/30s; bar_update DEBUG-sampled 1/500 post-T8; 08-27 file log reached 10,007,650 lines; bytes/day+duration UNVERIFIED. DB: WAL (data.db-wal present); 0 busy/locked in file logs; integrity_check + size growth UNVERIFIED. **Verdict: sound at code+artifact level; no BROKEN.** (Full A1 text preserved in the agents' output record.)

### A2 — BARS & DATA TRUTH

S-list: S1 **BROKEN** post-fix `closes_dropped=8` @08-28 11:20:06 ([ERRO] persist queue stalled 6s+; healed by 31,988-bar boot backfill). S2 pre-fix 86,400 bars over 12 events. S3 peak_depth hit cap both eras (1024/1024 08-27; 4096/4096 ×7 during the 09:12–11:20 GORM stall, ~38k intrabar drops, self-healing). S4 Friday ingest gap-free (2 bars/min across all 5 boots; `bars integrity OK: dups=0 tfs=1m` every boot; rows 14622→15570). S5 Friday 16:00 close clean at cache level (16:00:39 CME-close log, then no_new_data; DB phantom census UNVERIFIED). S6 pnl_corrected exact on both newest closes ($63.00, $17.00; NULL rule = 317 sync + 37 reconcile_flat legacy exclusions, `store/position_query.go:39`). S7 level_stats 0→28→74→101 rows, all 3 due 17:05 runs landed (tonight 27 seated / 176 episodes / 67 levels). S8 [B] 2nd symbol×tf pair in bars unidentified (no SQL). Open-stamp convention PROVEN in code+tests (`open_stamp_test.go`, close-stamp→open-stamp at ingest, persist closed bars only); 100-row sample UNVERIFIED. 5m/15m aggregates: none stored BY DESIGN (1m-only + derive-on-read); independent rebuild UNVERIFIED. Volume column `v` exists; values UNVERIFIED. touch_episodes schema sound; 50-sample UNVERIFIED. 3-way arbiter legs EVENT-WAIT (auth token expired; NT8 leg needs Sunday). **Verdict: structurally sound; one post-fix violation + GORM stall to fix.**

### A3 — DETECTORS & MAP

S-list: S1 [B] `KindForLabel` lacks SWG-H/SWG-L → `KindRound`→`pivot` on label-only paths (live 07:48:28 WARN `SS3 cites SWG-H·5m (pivot)`); seated roles unaffected. Detectors: 19/19 formulas EXACT at 8666db0b (PDH/PDL/PDC, RTH-H/L, PWH/PWL/PMH/PML, ONH/ONL, OR/IB±ext, round numbers, gaps, EQH/EQL, SETT, MID-O, VWAP±1σ/±2σ (volume-weighted population σ, session-day window), eVWAP 15:00 anchor, pdVWAP, pdPOC/VAH/VAL 120-bin 70%VA, nPOC retire-on-touch, SWG-H/L 5m+15m k=2, S/D base≤6 body≤0.5ATR departure≥1.5ATR, FVG gap≥2pt + session guard, iFVG, OB disp≥1.5ATR lookback 8, touch band 16t=4pt, fvg_entry disp≥1.5ATR5m, breakdown BD_MIN_DISP_ATR=1.0 max-pullback 0.4×leg 2 closes) — none MISSING. Proximity @0.3 PROVEN live: band flipped ±458→±85pt at exactly 11:59:00 CT (0.3 × proxy 283.33); pool census 12 seated (cap) of 51–192 in-band (11:59 12/170 · 12:05 12/86 NY v5 · 12:51 12/192). Weight tables = README §2.3 zero drift. Role validator WARNs behaving correctly (6 live). VWAP residual + missed-turns @0.3 + grade recompute + freshness transitions + stamp-gap + consumed census: UNVERIFIED (no terminal). **Verdict: formula-sound, retune live-verified; one S (SWG label gap); no BROKEN.**

### A4 — EXECUTION & ORDERS

S-list: S1 [B] post-fill `⚔️ armed` re-log spam + dead re-arm (`store/armed_orders.go` UpsertArm never resets state; 69+ lines today). S2 [B] planner p95=600s, 2× timeout (12:41:05, 12:51:06). S3 [B] gate-change cancel missing (comment 1.3 vs code). S4 [B] fill-time lineage race (all 4 fills "no open position row to stamp"; repair stamped 1). S5 F6 dedup has no live proof yet (EVENT-WAIT). PROVEN: state machine legal (armed→working→filled/cancelled only; `expired` has no writer), 0 orphans; gate chain order = ArmSpecValid→direction→plan_mode→quality→R:R→min-SL→HTF-veto (`armed_executor.go:523`); placement band ±100t exact (#567 Δ0.01t, #569 exact, #570 67t favorable in-band); fill chains re-quoted (#567 f14ea5dd 01:21:03→01:57:47 −42.00; #569 57cff804 08:43:28→08:45:28 +63.00; #570 LONG 29463.25 +$17); stale_reeval bypass structural for armed_fill; **move_stop/BE LIVE on #570 (BE 12:47:05 `auto_trader.go:183` + 5 trail ratchets 13:20–13:45 + SL exit 13:48:34 @29471.75)** — GAR-F1 wire-proven; cancel-before-flatten LIVE at 08:30 boundary (`✕ armed S2 cancelled in NT8` → `2 order(s) disarmed`); 14:45:21 `session ended (EOD flat) — 1 order(s) disarmed` (no WORKING arm — FIX-1 wire proof still EVENT-WAIT); ClassifyCitation fixtures 8-case (7 in `plan_citation_test.go` + w9) — live strict instance 0; min-SL/R:R/veto one live each; quality/direction zero live (EVENT-WAIT); FAST TAPE 0 firings (EVENT-WAIT). Exec latency n=28 p50≈37.0s p95≈88.2s; planner n=18 p50≈401.9s. **Verdict: execution core sound — but journal spam/dead re-arm, missing gate-change cancel, and the 600s planner ceiling need a fix wave.**

### A5 — PLAN ENGINE & LIFECYCLE

S-list: S1 **BROKEN** `scenarioConds` 7-only → schema gate rejects the 8th/9th conditions at parse before the waterfall validator (verified [O]); S2 newest stored prompt is pre-waterfall (NY v7 13:08:51, `475f94b5c4f5`); S3 storm-guard exact-30m rounding (cosmetic). PROVEN: full prompt renderer enumerated (all section headers + FRESH FVGs demand + 8-condition schema + ARMED ORDERS mandate + FEASIBILITY CONTRACT + WATERFALL PLAY + two-leg confirm2 wording) at `planner_prompt.go:276,324,514,536,554`; validator chain enumerated (parse→collapse→flip-bias→labels→facts+side-quota→arm-feasibility WARN→FVG re-verify→waterfall re-verify) with live rejection quotes for 6 classes; alias normalization both enums (`plan_doc.go:369-396`) + pre-fix rejection evidence; truncation: 0 finish_reason=length post-fix (4 pre-fix at 32768 on 08-27), max completion 35,126/65,536 (54% headroom); lifecycle: 4 dormants all hysteresis-quoted (buffer 0.5×ATR14, 2 closes), **0 rearms in 7d**, deaths/session trend table (pre-fix DIED-with-budget → post-fix dormants), replans_exhausted 0 post-fix; wake census: level_event 7/33+/20/15 (08-25→28), MSS 2, owner resets 3, post-exit 7, storm guard 30m with 60+ skips; dormant-keeps-eyes live (NY v3 dormant 09:31:35 → wake 09:43:37 → 3 rejects → "benign — active plan kept" 10:05:47); two-leg render fixture PASS (`waterfall_twoleg_test.go`); C6 3 refusals on 08-26, 0 since; counters (decline_fresh_met vs arm_authored) UNVERIFIED (no token). **Verdict: one BROKEN (S1), rest PROVEN.**

---

*Part 2/2 (strategy/money/calibration layer) is a separate dispatch — not covered here.*
