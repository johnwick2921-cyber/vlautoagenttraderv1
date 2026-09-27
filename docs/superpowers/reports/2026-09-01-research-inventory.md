# RESEARCH INVENTORY — everything commissioned 2026-08-17 → 2026-09-01

Date: 2026-09-01 ~01:10 CT · Read-only · No merges of stranded branches, no fixes.
Method: `git log --all` since 08-17 (docs paths), per-branch `git log dev..<branch>` file census, merge-status check for every local branch, worktree listing, repo memory, `docs/research/`.

## MASTER TABLE

| # | Artifact | Commissioned → delivered | Lives: branch · sha · path | Merged? | Verdict (1-2 lines) | Action taken | Contradicts/duplicates |
|---|---|---|---|---|---|---|---|
| 1 | reverse-trace audit | 08-17 → 08-17 | dev `283e75b5` reports/2026-08-17-reverse-trace-audit.md | dev | 22 values traced, 0 invented, 0 causeless | none (verification) | — |
| 2 | end-to-end audit | 08-17 → 08-17 | dev `f673ce5e` 2026-08-18-end-to-end-audit.md | dev | one decision traced bar→order, 17 hops, 1 mismatch | none | — |
| 3 | full verification | 08-17 → 08-17 | dev `a4025b97` 2026-08-18-full-verification.md | dev | 34 pass, 13 findings | partial fixes | — |
| 4 | total sweep (8 error types) | 08-17 → 08-17 | dev `7d98c619` 2026-08-18-total-sweep.md | dev | guardrail_skip causes autopsied, ASIA enabled | cleanup-train | — |
| 5 | consumed-without-touch audit | 08-17 → 08-17 | dev `448142f1` 2026-08-17-consumed-without-touch.md | dev | 0 of 7 burns justified | **shipped** touch-gate fix (08-17 `91333cfb`) | — |
| 6 | why-no-trades + aug-14 bisect + partner-vs-us + total root-cause + total e2e | 08-18 → 08-18 | dev `00ca0f32`, `5d58ca3f`, `7ec664c9`, `de5a54ba`, `63062e5e` | dev | pipeline exonerated; truncation zeroed decisions; clock-drift guard converting entries to wait | **shipped** `f6478923` + clock-drift fixes | one investigation chain, not dups |
| 7 | timegate audit | 08-18 → 08-18 | dev `8d8cf492` 2026-08-18-timegate-audit-ai-timeout.md | dev | 57 gates, 8 BUG rows, WSL skew caught | **shipped** same wave | — |
| 8 | 5-day zero-trade postmortem | 08-19 → 08-19 | **STRANDED** `docs/forensics-zerotrade-2026-08-19` `765ac11a` reports/2026-08-19-zerotrade-forensics.md | no | postmortem of the zero-trade stretch | partial (root causes fixed elsewhere) | overlaps #6 |
| 9 | strategy-controls census | 08-19 → 08-19 | **STRANDED** `docs/strategy-controls-census` `e12e3846` 2026-08-19-strategy-controls-census.md | no | 246 FE controls, 18 trader cols, 28 env knobs; 28/39 confirmed; 12-row DEAD/PARTIAL/SHADOWED register | partial (some dead controls fixed later) | — |
| 10 | controls runtime re-verification | 08-19 → 08-19 | **STRANDED** `docs/controls-runtime-verify` `1522cfa2` 2026-08-19-controls-runtime-verify.md | no | **min-conf enforced at 60 not 65; gate-65 vs futures-prompt-60 defect** | partially resolved (owner dated 60); defect status UNKNOWN | contradicts spec doc 65 |
| 11 | decision anatomy (canonical map) | 08-19 → 08-19 | **STRANDED** `docs/decision-anatomy` `2d4a706e` 2026-08-19-decision-anatomy.md | no | owner's canonical map: birth of a trade, 87-step gate order | none (reference) | — |
| 12 | brand census | 08-20 → 08-20 | **STRANDED** `docs/brand-census` `23582b2a` 2026-08-20-brand-census.md | no | docs branding census | none | — |
| 13 | shift-day loss forensics | 08-21 → 08-21 | **STRANDED** `docs/research-import-shift-forensics` `d070c932` 2026-08-21-shift-day-loss-forensics.md | no | shift-day loss root-cause | partial (fixes in later waves) | — |
| 14 | master audit v2 | 08-22 → 08-22 | dev `1b56436e` (merged) 2026-08-22-master-audit-v2.md | dev | 76 rows, 22 findings, 3 money-risk | fixed in waves | — |
| 15 | research conformance audit | 08-22 → 08-22 | dev `audit/research-conformance` (merged) 2026-08-22-research-conformance.md | dev | ATR WILDER mismatch fixed; calibration queue defined | **shipped** ATR fix + queue | — |
| 16 | research full readthrough | 08-22 → 08-22 | dev (merged) 2026-08-22-research-full-readthrough.md | dev | plan-card research readthrough | none | — |
| 17 | level-grading full audit | 08-24 → 08-24 | dev 2026-08-24-level-grading-full-audit.md | dev | zone-grading v3 basis | **shipped** v3 grading | — |
| 18 | deep-status full accounting + plan flip/death audit | 08-25 → 08-25 | dev | dev | flip/death chain accounting | **shipped** plan-lifecycle wave | — |
| 19 | 1h-timeframe research | 08-25 → 08-25 | dev 2026-08-25-1h-timeframe-research-wave.md | dev | 1h zones A-capable, TFmult ≈2.3× | **shipped** 1h-wave R2/R4 (rev 57b60b60) | — |
| 20 | fvg-entry model (external ~40k gaps study) | 08-26 → 08-26 | dev 2026-08-26-fvg-entry-model.md | dev | reaction real, NO tradeable edge after honest costs (intrabar look-ahead artifact) | **shipped** 0C shadow demotion (08-31) | later contradicted the FVG-demand prompt work (A2c) — 0C ruling settled it |
| 21 | volume-wave phase-0 stop (Pack B) | 08-26 → 08-26 | dev 2026-08-26-volume-wave-phase0-stop.md | dev | **NO-GO: no stored bar history anywhere** | queued bar persistence (shipped 08-28 bar-truth) | — |
| 22 | settings census + week-in-review | 08-26 → 08-26 | **STRANDED** `docs/settings-week-audit` `dda9777c` (2 files) | no | settings/week census | none | — |
| 23 | london drought (v1) | 08-26 → 08-26 | **STRANDED** `docs/london-drought` `607b8861` 2026-08-26-london-drought.md | no | flip-threshold vs chop; latency vs reeval_drift — "do NOT act" | none (read-only ruling) | **DUPLICATE of #24** |
| 24 | london drought (v2) | 08-27 → 08-27 | **STRANDED** `docs/london-drought-2026-08-27` `f962d648` 2026-08-27-london-drought.md | no | 08-26 last trade; 212 cycles/191 waits; no_trade can't self-revive | none | **DUPLICATE of #23** — answers agree |
| 25 | mega-research MNQ | 08-26 → 08-27 | **STRANDED** `docs/mega-research-mnq` `2cea2029` 2026-08-27-mega-research-mnq.md | no | S1 proximity ±530pt admits whole map; acceptance −1587; ASIA −1823; calibration queue | **partially shipped**: S-fix proximity 0.3; Sep-9 docket for the rest | — |
| 26 | master recheck | 08-27 → 08-27 | **STRANDED** `docs/master-recheck` `4b65eeeb` 2026-08-27-master-recheck.md | no | "cutover clean, scoreboard half-deployed"; pnl_corrected NULL era | fixed later (T7 pnl_corrected) | — |
| 27 | level-system verify | 08-27 → 08-27 | **STRANDED** `docs/level-system-verify` `d6aa9669` 2026-08-27-level-system-verify.md | no | zone weights off spec (iFVG .35 vs .30, volume weights), touch reaction inverted (grades not predictive) | none (documented-only) | partially superseded by v3 grading |
| 28 | deep-verify-22 | 08-27 → 08-27 | **STRANDED** `docs/deep-verify-22` `95049c0c` 2026-08-27-deep-verify-22.md | no | EOD armed race window; level_stats nightly skip; ingest cap hit; partner drift | partial (S-list closer wave) | — |
| 29 | refusal autopsy | 08-27 → 08-27 | **STRANDED** `docs/refusal-autopsy` `589f7865` 2026-08-27-refusal-autopsy.md | no | per-gate would-have-won/lost replay — **SCHEDULED Sep 3** | queued (Sep 3) | — |
| 30 | grand audit Parts A-E + verdict | 08-28 → 08-28 | **STRANDED** `docs/grand-audit` `104f0d3d` (2 files) | no | Part A verdict A zero-broken; S1 move_stop dead cell; S2 proximity doc mismatch; decline leak +$1974.5 | **shipped** GAR wave (F1-F6) | — |
| 31 | missed-200pt | 08-28 → 08-28 | **STRANDED** `docs/missed-200pt` `673e9240` 2026-08-28-missed-200pt.md | no | the 200pt missed move analysis | partial (wake fixes) | overlaps #32 |
| 32 | weekend audit part 1 (machine layer) | 08-28 → 08-29 | **STRANDED** `docs/weekend-audit-p1` `c18bd3a2` 2026-08-29-weekend-audit-p1.md | no | S1 scenarioConds 8th-condition BROKEN; S2 closes_dropped; S3 armed spam; S4 600s ceiling | **shipped** pre-reopen hotfix wave | — |
| 33 | weekend audit part 2 (money layer) | 08-28 → 08-29 | **STRANDED** `docs/weekend-audit-p2` `b964dc8e` 2026-08-29-weekend-audit-p2.md | no | week −$176.5 NY-only edge; gates net −$511.8 SAVING; zero knobs NOW; min-conf 60-64 band + proximity buckets → Sep-9 | **ruled on** (Sep-9 docket) | — |
| 34 | e2e verification | 08-28 → 08-28 | **STRANDED** `docs/e2e-verify` `03957e58` 2026-08-28-e2e-verification.md | no | **S-1 stored bars ≠ live truth (backpressure punctures); S-2 VWAP residual 1.74pt** | **shipped** bar-truth wave | VWAP question = dup with #35, agreed after fix |
| 35 | final verify sweep v2 | 08-28 → 08-28 | **STRANDED** `docs/final-verify` `91995ad4` 2026-08-28-final-verify.md | no | VWAP residual died (0.046pt — was misstamped bars); retention broken; ingestion cap overrun | **shipped** forensics-hygiene wave | resolves #34's S-2 |
| 36 | london forensics (first live armed fill) | 08-28 → 08-28 | **STRANDED** `docs/london-forensics` `a5595503` 2026-08-28-london-forensics.md | no | pos #567 armed-fill chain; planner truncation at 32768; 4 refusals every cycle | **shipped** london-fix wave | — |
| 37 | zone-math total verification | 08-29 → 08-29 | **STRANDED** `docs/zone-math-verification` `6919aa8b` 2026-08-29-zone-math-total-verification.md | no | 6 zone classes verified over 3 sessions | none | — |
| 38 | total audit (15 agents) | 08-29 → 08-29 | **STRANDED** `docs/total-audit-15` `35c3aad9` 2026-08-29-total-audit-15.md | no | S-list + 19-class table + conditional Sunday GO | consumed by Sunday cutover | overlaps #32/#33 |
| 39 | dryrun replay scope | 08-29 → 08-29 | **STRANDED** `docs/dryrun-replay` `4b0ba249` 2026-08-29-dryrun-replay-scope.md | no | **ruling-B smoke FAILED (libfaketime can't move Go time.Now vDSO); shadow cancelled; Sunday live-fire = integration test** | owner accepted | — |
| 40 | pre-livefire verify (v1) | 08-29 → 08-29 | **STRANDED** `docs/pre-livefire-verify` `85095811` 2026-08-29-pre-livefire-verify.md | no | 17/18 PASS; **S1 persist watchdog declared-but-unwired BROKEN** | **shipped** s1-watchdog-wire wave | **DUPLICATE of #41** |
| 41 | pre-livefire verify (0830) | 08-30 → 08-30 | **STRANDED** `docs/pre-livefire-verify-0830` `a290920e` 2026-08-30-pre-livefire-verify.md | no | P5 verdict table + owner-pending list | same | **DUPLICATE of #40** (superset) |
| 42 | dress rehearsal 0830 | 08-30 → 08-30 | **STRANDED** `docs/dress-rehearsal-0830` `b54a9bfc` 2026-08-30-pre-livefire-verify.md (same file, rehearsal branch) | no | session plan review + weekly r1-r6 clean + isolation proof | none | overlaps #41 |
| 43 | knob & constant provenance census | 08-30 → 08-30 | **STRANDED** `docs/knob-census` `39a0481e` 2026-08-30-knob-census.md | no | every money-path number labeled [R]/[D]/[O]/[C]/[I]; top-10 unvalidated = zoneSizeMult, decay ladders, FAST_MARKET_ATR 1.5, cluster 3pt | **NO ACTION** | — |
| 44 | cheap-five knob verdicts | 08-30 → 08-30 | **STRANDED** `docs/cheap-five` `9298f9d4` 2026-08-30-cheap-five-knob-verdicts.md | no | five cheapest knob validations with verdict tables | **NO ACTION** | — |
| 45 | confirm-cost forensics | 08-30 → 08-30 | **STRANDED** `docs/confirm-cost-0830` `8f09aa84` 2026-08-30-confirm-cost-forensics.md | no | close-confirms NET-COST ≈ −$681 over 30 MET replays | **NO ACTION** | — |
| 46 | massive-move audit | 08-30 → 08-30 | **STRANDED** `docs/massive-move-audit` (origin-only) `151ef42b` | no | −204pt ASIA sell-off minute-by-minute; 5 bug-class candidates (weekly render gap ×2, armed wrong-side, sub-60s ledger, flip latency) | **shipped** via F1/F3/F5 weekly + armed guards | — |
| 47 | planner latency autopsy | 08-31 → 08-31 | **STRANDED** `docs/planner-latency-autopsy` `168e5282` 2026-08-31-planner-latency-autopsy.md | no | ~99.5% provider generation (26.4k tokens @ 63-66 t/s ≈ 420s); pre/post sub-second; fix shortlist | **shipped** planner-speed wave (repair + streaming); T1/T2/T4 instrumentation gaps remain | — |
| 48 | EOD verification | 09-01 → 09-01 | dev `b8f68db1` 2026-08-31-eod-verification.md | dev | six waves verified live; +$164.00; class-34 live-fired | verification | — |

Static research docs (commissioned literature, on dev): `docs/research/PLAN-CARD-DESIGN-SYSTEM.md`, `VL-DAYPLAN-FULL-SPEC-(2).md`, `AI-Trader-Analysis-Report.md`, `Strategy-Studio-Complete-Plan.md`, `VL_Trading_System_*.md` + `docs/research/plan-card/` (7 files, the conformance audit's basis).

## A. Commissioned topics in date order

08-17 verification chain (reverse-trace, e2e, full-verify, sweep, consumed-touch) → 08-18 no-trade investigation chain (why-no-trades, bisect, partner A/B, root-cause, total-e2e) + timegate audit → 08-19 zero-trade postmortem + strategy-controls census + controls-runtime-verify + decision-anatomy → 08-20 brand census → 08-21 shift-day forensics → 08-22 master-audit-v2 + research conformance + full readthrough → 08-24 level-grading audit → 08-25 1h research + deep-status/flip-death → 08-26 fvg-entry model + volume-phase0 + settings week + london-drought v1 + mega-research → 08-27 london-drought v2 + level-system-verify + deep-verify-22 + refusal-autopsy + master-recheck → 08-28 grand-audit A-E + missed-200pt + weekend-audit p1/p2 + e2e-verify + final-verify + london-forensics → 08-29 zone-math + total-audit-15 + dryrun-replay + pre-livefire-verify → 08-30 pre-livefire-0830 + dress-rehearsal + knob-census + cheap-five + confirm-cost + massive-move → 08-31 planner-latency-autopsy → 09-01 EOD verification.

## B. Orphaned findings — delivered, never acted on

1. **knob-census (#43)**: top-10 unvalidated [I] knobs (zoneSizeMult ladder, decay ladders, FAST_MARKET_ATR=1.5, cluster 3pt) — validated-by-nothing, still live. NO action.
2. **cheap-five (#44)**: five ready-made knob verdicts. NO action.
3. **confirm-cost forensics (#45)**: close-confirms net-cost ≈ −$681. NO action, no ruling.
4. **mega-research calibration queue (#25)**: swing-k 2 vs 10-20, MSS-FVG on/off, HTF_VETO_TF, trail-mult 2.0 vs 1.5 — docketed Sep-9, not yet decided (loss-streak item MOOT — owner removed G6).
5. **weekend-audit-p2 (#33)**: min-conf 60-64 band and proximity distance buckets — docketed Sep-9 only.
6. **level-system-verify (#27)**: zone-weight spec divergences (iFVG .35 vs .30 flat; volume weights off) — documented, never ruled.
7. **strategy-controls-census (#9)**: the 12-row DEAD/PARTIAL/SHADOWED register — only partially swept.
8. **controls-runtime-verify (#10)**: the gate-65 vs futures-prompt-60 default mismatch defect — status UNKNOWN, never explicitly closed.
9. **london-drought v1/v2 (#23/#24)**: flip-threshold-vs-chop and latency-vs-reeval_drift open questions — explicitly "do NOT act", never re-opened.
10. **brand-census (#12)**: no action.
11. **settings-week-audit (#22)**: no action.
12. **planner-latency-autopsy (#47)**: T1 map-assembly, T2 prompt-render, T4 first-token instrumentation + completion-side diet — the diet was NOT shipped (only repair+streaming).
13. **grand-audit (#30)**: decline leak analysis delivered; no decline-specific ruling.

## C. Stranded — exists but a future session would not find it on dev

35 stranded branches, each holding a report that never merged to dev: `docs/brand-census`, `docs/cheap-five`, `docs/confirm-cost-0830`, `docs/controls-runtime-verify`, `docs/decision-anatomy`, `docs/deep-verify-22`, `docs/dress-rehearsal-0830`, `docs/dryrun-replay`, `docs/e2e-verify`, `docs/final-verify`, `docs/forensics-zerotrade-2026-08-19`, `docs/grand-audit`, `docs/knob-census`, `docs/level-system-verify`, `docs/london-drought`, `docs/london-drought-2026-08-27`, `docs/london-forensics`, `docs/master-recheck`, `docs/mega-research-mnq`, `docs/missed-200pt`, `docs/planner-latency-autopsy`, `docs/pre-livefire-verify`, `docs/pre-livefire-verify-0830`, `docs/refusal-autopsy`, `docs/research-import-shift-forensics`, `docs/settings-week-audit`, `docs/strategy-controls-census`, `docs/total-audit-15`, `docs/weekend-audit-p1`, `docs/weekend-audit-p2`, `docs/zone-math-verification`, `docs/massive-move-audit` (origin-only) + feature branches `fix/clock-hold`, `feat/weekly-bias`, `hotfix/breakeven-dead`, `fix/ledger-close-sep-risk` (their CONTENT merged via other commits, but the branch tips are stranded). Report paths and shas are in the master table. NOT moved (stop-line).

## D. Duplicate research

- London drought ×2 (#23 08-26 vs #24 08-27) — two branches, overlapping analysis; **answers agree** (drought causes consistent).
- Pre-livefire verify ×2 (#40 08-29 vs #41 08-30, plus rehearsal #42) — the 0830 is a superset; **agree**.
- VWAP residual asked in #34 (1.74pt) and re-asked in #35 (resolved to 0.046pt, root cause = misstamped bars) — **agree after fix**.
- Overlap (not full dup): total-audit-15 vs weekend-audit-p1/p2 vs grand-audit — same agent-audit style over overlapping scope; findings consistent.

## E. Contradictions

1. **fvg_entry**: the 08-26 external study (#20, "no tradeable edge, 5m/15m worst") vs the FVG-demand prompt contract shipped earlier (A2b/A2c, "SHOULD author an fvg_entry") — resolved by the 0C owner ruling (shadow; authoring allowed, arming refused). The prompt still demands FVG authorship; the arm seam refuses it. Documented tension, owner-ruled.
2. **min_confidence**: research-conformance contract 65 vs live 60 (#10) — resolved: owner dated 60 (spec updated `f7ff58e2`), 65-era never exercised.
3. **proximity**: research-era 1.5×dATR (±530pt, mega-research S1) vs S-fix 0.3 — resolved: owner clicked 0.3 (11:59 08-28), live.
4. **side quota**: original P0 ≥3-per-side vs quota-relax WARN — resolved: owner ruling 08-31 removed side counts entirely.
5. **VWAP residual**: 1.74pt vs 0.046pt — resolved as data bug, not a live disagreement.
6. **volume weights**: spec vs code (level-system-verify) — documented divergence, unrulled (live code wins).

## F. Knob/threshold/rule recommendations and their status

| Recommendation | From | Status |
|---|---|---|
| proximity_filter_atr 0.3 | mega-research S1, S-fix | **LIVE 0.3** |
| min_confidence 65 (band 60-64) | conformance, weekend-p2 | **LIVE 60** (owner); band **docketed Sep-9** |
| min_side_levels removal | owner | **LIVE removed** (08-31) |
| swing-k 10-20 (vs 2) | mega-research | **queued** (calibration) |
| MSS-FVG on/off | mega-research | **queued** |
| trail-mult 1.5 (vs 2.0) | mega-research | **queued** |
| loss-streak pause | G6 | **MOOT** (owner removed) |
| ARM_MIN_RR 2.0 keep | weekend-p2 B1 | **LIVE keep** |
| HTF_VETO_MODE cross | grand-audit S4 | **LIVE cross** |
| min_scenario_quality C knob | 1h-wave | **LIVE** |
| scenario_cap 5 / max_levels 12 | owner | **LIVE** |
| FAST_MARKET_ATR 1.5, zoneSizeMult, decay ladders, cluster 3pt | knob-census [I] | **LIVE UNVALIDATED — ignored** (the orphan list) |
| fvg_entry/breakout_retest shadow | fvg study + MNQ study | **LIVE shadow** (0C) |
| AI_PLAN_MAX_TOKENS 65536 + repair retry | latency autopsy | **LIVE** |
| BD_MIN_CLOSES 1 / BD_MIN_DISP_ATR 1.0 / ACCEPT_HOLD 10 / STOP_OFFSET 2 / RETEST_WAIT 6 | entry-mechanics | **LIVE** |
| level_stats nightly + bars persistence | deep-verify/e2e | **LIVE** |

## Unknowns

- The exact commissioned dates for several early items are reconstruction from commit dates (marked 08-17→08-17 style; commits are the best available record).
- #10's gate-65 defect closure status: UNKNOWN — would be resolved by grepping the current gate resolver (live: futures prompt uses 60).
- Partner-repo research copies: UNKNOWN without checking `~/vlautoagenttraderv1` (not this workspace).
