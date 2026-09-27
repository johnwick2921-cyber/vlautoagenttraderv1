# Pre-Live-Fire Verification — P2/P3/P4 (2026-08-30)

Sandboxed dispatch. Live system untouched (bot PID 482741 running throughout).
Worktree: `~/nofx-vf` @ `a9aa9a04` (branch `docs/pre-livefire-verify-0830`, the
deployed `23243670` lineage — boot-acked per the commit record). DB copy:
`/tmp/nofx-vf-db/data.db` (sandbox, rw). LIVE DB `/home/hoang/nofx/data/data.db`
opened ONLY `mode=ro` (+ `PRAGMA query_only=ON`). Harness:
`cmd/vfverify` in the worktree (imports the repo's `kernel`/`market`/`store`/
`calendar`/`mcp`/`crypto` packages; builds the planner input the way
`trader/auto_trader_planner.go assemblePlannerInputWithCtx` does). No Google
Drive tools called. Evidence tiers [A]/[B]/[C].

---

## P2 — GRADE SYSTEM SPOT-VERIFY

### 2.1 Weight-table zero-drift check — code vs documented truth

Documented truth (superseding chain: `docs/superpowers/reports/2026-08-24-level-grading-full-audit.md` §2/§4,
then `docs/superpowers/reports/2026-08-25-1h-wave-r2-r4-implementation.md` §1 —
the 1h wave raised the 1h tier and split the 15m/1h cap):

| component | documented | code (`kernel/levels_score.go`) | probe result [A] |
|---|---|---|---|
| zoneEvidence OB | 1m .40 · 15m .50 · 1h **.70** · 4h .72 | `zoneEvidenceByKind` :150-151 | .4000/.5500/.8400/.9360 (×tfmult) ✓ |
| zoneEvidence S/D/FVG/iFVG | 1m .35 · 15m .45 · 1h **.65** · 4h .65 | :152-156 | .3500/.4950/.7800/.8450 ✓ |
| reversal ×1.1 (RBD/DBR) | ×1.1 | `zoneReversalBonus` :163 | ratio 1.100 ✓ |
| zoneTFMult | 1.0/1.1/1.2/1.3 | :166 | (embedded in the scores above) ✓ |
| typeEvidence | structural 1.0 · VWAP/POC .90 · SWG/VWAP2σ .85 · VAH/VAL/SETT .80 · AS/LDN/OR/IB/EQ .70 · MID .60 · Round/Gap .55 · zone .30 · default .50 | :78-109 | PDH 1.00 · VWAP .90 · SWG-H .85 · VAH .80 · ONH .85 · MID-O .60 · AS-H .70 · RN .55 · GAP .55 · EQL .70 ✓ |
| freshness (anchors) | fresh 1.0 / b .8 / c .6 / done .5 | `freshMult` :355 | 1.2/0.96/0.72/0.60 (×HTF1.2) ✓ |
| freshness (zones) | fresh 1.0 / b .6 / c .3 / done .15 | `zoneFreshMult` :376 | 0.84/0.504/0.252/0.126 (OB·1h) ✓ |
| confluence | score ×(1+0.20·conf), distinct FAMILIES, cap 3 | :450-465 + `ConfluenceCap` | conf 0→1.00, 1→1.20, 2→1.40, 3→1.60, **4→1.60 (capped)** ✓ |
| HTF origin | ×1.2 | :471 | ratio 1.20 ✓ |
| grade bands | A≥1.0 · B≥0.70 · else C | `gradeFromScore` :633 | band edges observed ✓ |
| zoneSizeMult | ≤.30→1.25 ≤.60→1.10 ≤1.0→1.0 ≤1.5→.85 ≤2.5→.70 else .50 | :204-223 | 1.05/1.05/1.05/0.924/0.84/0.714/0.588/0.420 ✓ |
| floors/caps | 1m→C · 15m floor B cap B · 1h floor B cap A · 4h floor B, may A | :499-524 | with a Tier-1 anchor: 1m C, 15m B, 1h A(1.008), 4h A(1.123); 4h+conf A(1.310) ✓ |
| B2 gate | pattern above C only within 12 ticks of Tier-1 | :269 + `withinTier1Proximity` | no-anchor probes all C ✓ (documented amendment — bare probes sit at C, the floor is reachable only beside a Tier-1 row) |
| WEEKLY shadow knobs | band 0.25×ATR5m · mult 1.5 | `kernel/weekly_knobs.go:74,88` | 0.25 · 1.50 (env unset) ✓ |

**Zero-drift verdict: [A] — every table value reproduces the documented truth exactly. No drift.**

### 2.2 Latest NY plan — machine-grade recompute

Row: `plans` trade_date 2026-08-28 session NY **version 7** (created
2026-08-28T18:08:51Z). Doc carries per level ONLY
`price/label/grade/instruction/machine_grade` — **score, distance, sweep,
evidence counts, freshness and confluence are NOT stored in the plan doc**, so
every row is **NOT-RECOMPUTABLE-FROM-DOC** for those fields (missing components
named). The independent recompute below instead re-runs the repo's
detector→scorer pipeline (`AssembleScoredLevelsFullMinGrade` +
`DetectHTFLevels` + naked-POC extras + `level_state` freshness) on the stored
1m bars at a cutoff = plan write instant (recomputed price 29504.00, dATR
367.2, proximity 0.30, maxLevels 12, minGrade B, TFs [D 4h 1h 15m 5m]):

| price | label | docG | machG | recomputed | verdict |
|---|---|---|---|---|---|
| 29424.00 | PDL | A | A | A | EXACT |
| 29437.00 | SWG-L·5m | A | A | A | EXACT |
| 29445.53 | VWAP−2σ | A | A | — | NO-STAMP (replay pool VWAP−2σ at 29444.25) |
| 29502.88 | OB(bear)·1h | C | C | C | EXACT |
| 29512.00 | EQL | A | A | — | NO-STAMP (replay EQL cluster at 29501.50/29509.50/… ) |
| 29516.00 | EQL·15m (HTF) | A | A | — | NO-STAMP (replay at 29516.25) |
| 29531.05 | VWAP−1σ | A | A | — | NO-STAMP (replay at 29529.98) |
| 29549.00 | SWG-H·5m | A | A | A | EXACT |
| 29573.96 | pdVWAP | A | A | — | NO-STAMP (absent from replay pool) |
| 29577.75 | ONL | C | C | C | EXACT (fresh=flipped, score 0.680) |
| 29592.50 | SWG-L·15m | A | A | A | EXACT |

**6/11 EXACT · 0 DELTA · 5 NO-STAMP.** The 5 no-stamps are a sandbox-replay
artifact, not a defect: volume-family (VWAP±σ, pdVWAP) and EQL-cluster prices
shift ~1 pt between the live BarCache at 13:08 CT and my stored-bars replay, so
the rounded-price stamp key misses [B]. Every stampable row matches the stored
machine grade.

**S-FINDING P2-S1 (label provenance on live plans):** the live v6/v7 docs carry
`"PDL" = 29424.00`, but the machine's own prior-calendar-day low (the code's
PDL definition, `kernel/levels_multiday.go:39,146`) is **29402.25** — Thursday
2026-08-27 06:12 CT bar, present in the stored bars [A]. 29424.00 is the prior
**RTH** low (the recomputed pool seats `RTH-L` at exactly 29424.00). The model
relabeled the RTH low as PDL and the P0.4-H mislabel gate
(`kernel/plan_doc.go:621 MislabeledStructuralLevels`) did not stop it — at the
write instant the machine label map presumably held no "RTH-L" entry at that
rounded key, or a non-structural label recorded first [B]. The plan's bias-tree
"dealing range 29424–29707.50" therefore uses the RTH low as the day floor, not
the true PDL — a phantom-PDL-class labeling issue on live plans. Recommend:
strengthen the mislabel gate to compare against the detector's *anchor-class
prices* (PDL family) even when the exact rounded key is unstamped.

### 2.3 Weekly-class shadow case

ATR5m = 38.22 (recomputed by the repo's own `StaleConfirmATR5m` — the plan's
`indicators_block` stores per-TF ATR14 only, **no ATR5m row exists**; named as
the missing stored component). Band = ±9.56 pts. Weekly refs: weekly_open
29392.25 · PWH 29688.50 · PWL 29202.50 (RefsOK) → 3 shadow refs (IPDA 20d+
insufficient; no unfilled NWOG in the 08-19→08-28 window).
**No seated doc level is within ±0.25×ATR5m of PWH/PWL/weekly_open — stated
explicitly.** Shadow math therefore not exercised by this plan; the g×1.5 rule
itself was verified separately (GradeRank A/B/C = 3/2/1 × 1.5 in §2.1 + the P4
dry-run render below).

## P3 — FULL PROMPT RENDER AUDIT (no AI, nothing stored)

Rendered with the worktree harness at 2026-08-30T10:29 CT, session=ASIA,
tradeDate=2026-08-30 (plannerTradeDateCT convention), from the LIVE DB ro.
**Prompt: 22,391 chars → 5,597 est. tokens (chars/4) = 34.2% of the 65,536
planner cap** (the ~12% expectation under-shoots reality — quote the real
number).

**3.1 Candles:** all 4 tables present — 15m×12 · 1h×12 · 4h×8 · daily×8,
oldest→latest. 3 rows per TF re-aggregated by hand from the raw 1m rows (epoch-
floor buckets; session-day 17:00 CT roll for daily) → **12/12 EXACT O/H/L/C/V**
(first/middle/last of each table) [A].

**3.2 Weekly Context:** renders exactly `## Weekly Context\nWEEKLY: none\n` —
no WEEKLY doc exists before today's 16:30 CT read [A].

**3.3 Calendar:** renders `(no filtered events)` — `calendar_slices` has **0
rows for 2026-08-30** (2026-08-31 carries German CPI T2; 2026-09-01 carries ISM
Manufacturing PMI T1 09:00 CT). **Monday ISM 09-01 correctly does NOT appear
for today's session** — the slice lookup is per trade_date, and `EventsForSession`
(currency filter only, `calendar/calendar.go:186`) is never even consulted
across days. Static T1 list has no 08-30 event [A].

**3.4:** ranked levels 12 rendered (max_levels 12 · min_grade B ·
seat_1h_zone on · proximity 0.30), all grades consistent with the score bands.
Structure: 5 lines (`D: unavailable` · `4h: unavailable` · `1h/15m/5m: RANGING`)
— D/4h "unavailable" is the honest H9 line: `kernel.StructureTFs = [5m,15m,1h]`
(`kernel/structure.go:34`) only; D/4h can never render (cosmetic S-note:
the prompt says 4h structure is unavailable while the 4h candle table sits
right above it). Regime rendered from all inputs; env HTF_VETO_MODE=cross (live
.env). FRESH FVGs 0 = source 0. HTF zones 4 rendered / 20 in the full source
universe. Consumed levels 10 (EQL/EQH cluster + PDL 29437 consumed per
`EvaluateLevelFacts`) — count matches source.

**3.5 Empty sections:** Calendar (weekend-legit, no slice) · FRESH FVGs
(weekend-legit, none fresh) · Auction story (by design — the assembler's return
literal never populates OvernightStory/PriorDayStory) · Owner note (none) ·
Prior-plan invalidation/levels (first read) · Warming line (profile store has
11 rows). Present: Consumed (10), HTF zones (4), digests (10 dailies).

**PlannerInput source list (T9 lesson — enumerated, no blind spots):** bars1m ·
session_registry · strategy day_plan+ai_config · session_profiles→NakedPOCs ·
DetectHTFLevels · AssembleScoredLevelsFullMinGrade+Seat1HZone+freshness ·
HTFZones/Full · owner_levels (0) · regime inputs · calendar_slices ·
day_plan_digests · StructureSnapshot · indicator mirror · consumed ·
ComputeBiasContext · candle tables · weekly ctx · warming · FreshFvgCandidates ·
ATR5m (recomputed).

## P4 — MANUAL PLANNER TEST (one real AI call, sandboxed)

Mechanics: sandbox DB (`/tmp/nofx-vf-db/data.db`, rw) → test-trader row
`trader-1` inserted there ONLY (sandbox traders count proof below); the rotated
DeepSeek key was decrypted from the sandbox `ai_models` row
(`8ef641a7…_deepseek`, enabled) via the repo's `crypto` package (ENC:v1
AES-GCM, `DATA_ENCRYPTION_KEY` + `RSA_PRIVATE_KEY` from the live `.env`,
read-only). One call: `mcp.NewAIClientByProvider("deepseek")` →
`CallWithMessages(plannerSystemPrompt, p3Prompt)` with `ApplyMaxTokens(65536)`.
Nothing was persisted anywhere except the sandbox trader row and /tmp artifacts.

<!-- P4-RESULTS -->

**The one real call** (2026-08-30 10:30:29 CT → 10:37:05 CT, 396.3s):
model `deepseek-v4-pro`, default base URL, `sk-ed…4681` (decrypted from the
sandbox row). Client telemetry `mcp/client.go:402`: **completion=22616
prompt=8304 finish_reason=stop** — [A] `finish_reason != length` (no
truncation; the repo's `LastFinishReason` exposes it, `mcp/client.go:551`).
**Real prompt size: 8,304 tokens = 12.7% of the 65,536 planner cap** (chars
22,391; the chars/4 heuristic gives 5,597 — the API usage number is the honest
quote).

**Validator chain (the write path):**
- `ParsePlanDocCapped(raw, 12, 5)` → **ACCEPTED** (11 levels ≤ 12 · 3 scenarios;
  the 9-condition enum + caps schema gate at `kernel/plan_doc.go:336,448`)
- `CollapsePlanLevels` → no merges
- `MislabeledStructuralLevels` → **clean**
- `ValidatePlanDocWithFactsMachine(…, sideQuota=4, 12, 5)` → **ACCEPTED**,
  thin-side notes []
- `ArmFeasibilityWarnings` (ATR5m=18.31) → none fired
- FVG / breakdown validators → n/a (no such scenarios)
- `bias-tree` branch: **CITED ✓** — "bias-tree: branch 3 inside-day flat LOW;
  branch 5 discount (price 19% of dealing range) disallows shorts…"
- **"per candles" citation: ABSENT (0 lines) — reported honestly** (the
  candle-ground-truth law has not been picked up by the model yet).

**The dress-rehearsal plan (never acted on):**

| id | condition | direction | entry/stop/target (arm) | R:R | verdict |
|---|---|---|---|---|---|
| S1 | sweep_reclaim | long | no arm (AI path) | — | confirm 1x5m_close above 29502.88 |
| S2 | sweep_reclaim | long | 29494.75 / 29475.00 / 29539.38 (wait_confirm) | 2.26 | well-formed long (stop<entry<target), stopDist 19.75 ≥ 18.31 = 1×ATR5m → gate-feasible |
| S3 | breakout_retest | long | no arm (AI path) | — | confirm 2x5m_close above 29520 |

bias neutral/low · day_type balance · death 29437 below 15m_close · flip 29520
above 2x5m→long · no_trade [first 5m, lunch].

**WEEKLY dry-run render (NO AI, NO execution)** — exactly 2 completed weeks
(bars start 08-19), `thin_history=true`:
```
## Weekly candles (12 completed weeks, oldest → latest)
Time(CT)  Open  High  Low  Close  Volume  Struct
2026-08-17  29623.25  29688.50  29202.50  29370.00  5655158  first
2026-08-24  29392.25  29811.75  28947.75  29509.50  12366930  outside
## Weekly references
weekly_open 0.00 · PWH 29811.75 · PWL 28947.75 · PWC 29509.50
## NWOG (last 5 weekend gaps, oldest → latest)
born  hi  lo  CE  filled
2026-08-24  29392.25  29370.00  29381.12  yes
## IPDA (trailing dealing ranges)
20d: insufficient history
40d: insufficient history
60d: insufficient history
## Prior week recap (facts only)
range 864.0 pts · close at 65% of range
```
(weekly_open 0.00 = current week has no first print yet — Sunday morning;
RefsOK=true, WeeklyRefSet=2 = {PWH, PWL}: IPDA insufficient, the one NWOG is
filled.)

## ISOLATION PROOF

| metric (LIVE, ro) | before P4 | after P4 |
|---|---|---|
| plans COUNT | 159 | **159 — UNCHANGED** |
| decision_records MAX(rowid) | 34740 | **34740 — no new rows** |
| armed_orders COUNT | 11 | **11 — UNCHANGED** |
| traders COUNT | 1 | **1 — UNCHANGED** (trader-1 exists ONLY in the sandbox DB: sandbox traders 1→2) |
| journal tail | 10:30:08 positions snapshot | bot lines only (wire_liveness/ack/GIN) — **zero harness lines** |

Bot PID 482741 alive throughout (48:59 → 57:18 uptime). The harness's own
stderr carried the AI-call logs; the live journal never saw them.

## VERDICT

- **P2 PASS** — weight table zero-drift vs documented truth (every value [A]-probed);
  6/11 machine grades EXACT recompute, 0 DELTA, 5 NO-STAMP (replay approximation [B],
  not a defect); shadow band empty (none within ±0.25×ATR5m) stated explicitly.
  **S-FINDING P2-S1:** live NY v6/v7 docs label 29424.00 "PDL" but the machine's
  PDL is 29402.25; 29424.00 is the prior RTH low — a phantom-PDL relabel that the
  P0.4-H gate (`kernel/plan_doc.go:621`) did not catch (price-keyed gate missed an
  anchor that wasn't in the machine label map at write) [A facts, B mechanism].
- **P3 PASS** — all sections render; candles 12/12/8/8 EXACT by hand aggregation;
  "WEEKLY: none" exact; ISM 09-01 correctly absent; empty sections all
  weekend-legit or by-design. S-note: `D`/`4h` structure lines always render
  "unavailable" (`kernel/structure.go:34` StructureTFs=[5m,15m,1h]) while 4h
  candles render right above them.
- **P4 PASS** — one real call: finish_reason=stop [A], schema gate + facts
  validator ACCEPTED, bias-tree cited, 1 gate-feasible arm (R:R 2.26, stop
  19.75 ≥ 1×ATR5m), "per candles" ABSENT (honest). Weekly dry-run exact
  (2 weeks, thin_history, insufficient IPDA 20/40/60d). ISOLATION PROVEN.


## P5 · VERDICT (orchestrator)

| Area | Verdict | Evidence |
|---|---|---|
| Connections (P1) | READY | NT8 wire 13 fr/min age 12s · 0 reconnects · AddOn md5 ×3 MATCH · DB integrity ok / WAL / last bar 08-28 15:59 CT (weekend-expected) · systemd active (NRestarts 90 incl. 3 today's cutovers) · chrony 0.12s + rtc_vs_go 0s · **DeepSeek rotated key HTTP 200** (sk-e…4681) · FF true-count "fetched 13 events" · static NFP/CPI armed · env truth: proximity 0.3 · veto cross · ARM_MIN_RR 2.0 · PLANNER_CANDLES on · WEEKLY_READ_CT sun 16:30 |
| Grades (P2) | READY (1 soft S) | weight table zero-drift; machine-path 6 EXACT / 0 DELTA; **P2-S1 phantom-PDL label** (29424.00 RTH-low vs machine prior-day 29402.25) — advisory label drift, fix spec ≤3 lines (label source or gate extension), owner-ruled post-NFP |
| Prompt integrity (P3) | READY | 8,304 real tokens = 12.7% of 65,536 · candles 12/12 exact · "WEEKLY: none" exact · Calendar/FRESH FVGs empty = weekend-legit · **S-note**: Structure D/4h "unavailable" by construction while 4h candles render (cosmetic) |
| AI round-trip (P4) | READY | finish_reason=stop · 9-condition gate ACCEPT · bias-tree cited · arm S2 well-formed (R:R 2.26, stop 19.75 ≥ 1×ATR5m) · "per candles" ABSENT (first-evidence pending, telemetry counts it) |
| Isolation | READY | live plans 159→159 · decisions max-rowid 34740→34740 · armed 11→11 · zero harness lines in the live journal |
| Weekly dry-run | READY | 2 completed weeks · thin_history=true · NWOG 08-24 filled · IPDA 40/60d insufficient — exactly what tonight's 16:30 read should author |

**OWNER-PENDING (no deploy, no blocker):** (1) DeepSeek 2 row (15m trader) still holds the OLD key `sk-2…054c` — still HTTP 200, rotation incomplete; owner click in Settings. (2) P2-S1 phantom-PDL. (3) Structure D/4h note.

## WATCHLIST (re-confirmed, expected lines)
- **16:30 CT** — `📅 WEEKLY READ starting for week 2026-08-31` → `📅 WEEKLY READ written 2026-08-31 v1 bias=… conviction=low … thin=true` (depth-guard proof).
- **16:55 CT** — `🛑 planner_preflight_refused … stale_bars` ×n (known, benign) → passes at first forming bar.
- **17:00–17:04 CT** — `🗓️ PLAN written 2026-08-30 ASIA v1`; prompt carries `## Candles` + `## Weekly Context: WEEKLY: <bias>/low …`.
- **17:05 CT** — `📊 level_stats: 2026-08-29 … giving up after 4 attempts` (known; real eval Monday 17:05).
- Watchdog: silence throughout (0 🔕); persist summaries `closes_dropped=0` through the reopen flood.
