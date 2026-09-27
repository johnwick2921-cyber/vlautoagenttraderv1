# 1h Timeframe Research Wave — 20-Agent Parallel Dispatch (2026-08-25)

**Mandate (owner):** why do so many traders use 1h levels for day trading and we don't?
Scope: CME MNQ index futures intraday — NOT stocks. Hard rule: 20-minute research
minimum; 20 agents dispatched in parallel; every finding cited; code sites mapped
file:line for the implementation wave.

**Method:** R1–R20 dispatched simultaneously. R1–R12, R18–R20 external research
(futures/SMC/MTF sources); R13–R17 read-only codebase mapping. Dead URLs retried
with alternates; bot-blocked sources substituted.

---

## SYNTHESIS — the consensus answer

### 1. Why traders use 1h levels (and we don't)

**External consensus (R2, R3, R7, R18, R20):** the 1h is the *setup/context rung*
of the intraday ladder. Every published multi-timeframe stack puts it between the
4h (bias/zones) and the 15m (entry):

- Day trading: **4H trend → 1H setup → 15M entry** (brokeranalysis, quantum-algo, futureshive, nexusfi, atas).
- Scalping: 1H bias → 15M setup → 5M entry.
- Adjacent TFs spaced 4–6× apart; 4H/1H/15M is exactly 4× (Rayner, brokeranalysis).
- R1's golden-rule quote: *"The Daily controls the H4, the H4 controls the H1, and the H1 controls the M15."*
- ICTKillzone NQ stat: a perfect 5m setup **failed at the unseen 1h level 83% of
  the time** — the "invisible ceiling" (R10).

**Why our plans carry almost none:** the v3 grading design (owner-approved
2026-08-24) caps **15m/1h zones at B** while 4h can reach A. With only 8 table
seats and 2 guaranteed HTF seats, A-capable 4h rows outrank 1h every time
(R13: `kernel/levels_score.go:306-325`). Today's plans: 1h levels appear 3 times
in 8 plans, zero 1h S/D zones in ASIA/LONDON. The audit flagged exactly this as
item 4.2 and it sat undecided.

### 2. CME-specific case for 1h (R4, R19)

- RTH = 8:30–15:00 CT = **6–7 hourly candles**; each session phase (open drive,
  lunch lull, late trend) maps to ~1–2 distinct 1h candles. A 4h bar cannot
  separate session phases — an AI holding minutes-to-hours gets no phase signal.
- Proximity math: MNQ dATR ≈ 250 pts → band ±1.5×dATR = ±375. 1h bases ≈ 20–100
  pts, formed every few hours → **8–12 in-band 1h zones**; 4h bases ≈ 100–300 pts,
  multi-day structures → only 2–4 in band. The in-band table is effectively a
  1h/15m table with occasional 4h guests — but grading gives the 4h guests the A
  seats.

### 3. Academic honesty (R5)

- Osler (2000/2003, FRBNY): intraday S/R levels (FX) predict turn interruptions;
  take-profit/stop clustering is the mechanism. No peer-reviewed study exists for
  hourly-granularity S/R in CME index futures — treating 1h S/R as predictive is
  an extrapolation. Zapranis-Tsinaslanidis: algorithmically-identified S/R is
  weak and definition-dependent. Sullivan-Timmermann-White: data-snooping risk.
- Verdict: the 1h upgrade is justified by practitioner consensus + session
  structure, NOT by futures-specific empirical proof. Keep it as a
  user-adjustable knob, not a hardcoded law.

### 4. Recommended implementation (research-grounded, knobs only)

Numbers verified against the scoring formula (R12):
`score = evidence × freshness × (1+0.20·conf) × zoneTFMult; A ≥ 1.0, B ≥ 0.70`

1. **1h zones A-capable** — lift the 15m/1h cap only for 1h:
   - `zoneEvidenceByKind` 1h values: OB 0.60→**0.70**; Supply/Demand/FVG/iFVG 0.55→**0.65**.
   - Floor/cap switch (`levels_score.go:306-325`): 15m stays floor B cap B; **1h → floor B cap A**; 4h unchanged.
   - Resulting ladder: fresh 1h OB + 1 confluence → 1.008 = A; 1h S/D + 2 confluence → A; 1h S/D reversal + 1 confluence → A. Fresh-only (freshness 0.8 stays B). 4h stays senior via TFMult 1.3 vs 1.2.
2. **Guaranteed 1h seat** — `seatHTF` (`levels_score.go:490`, `maxHTFSeats=2` at :501): reserve one of the two HTF seats for an in-band **1h** S/D zone when one exists. Requires TF to survive into `ScoredLevel` (today only the `HTF` bool does — `isHTFSwingZone` :476).
3. **Planner prompt mandate** — extend the conditional MUST-include rule (`planner_prompt.go:216-221`, `hasHTFZones` arg :207): when a 1h S/D zone is present, `"…you MUST include the nearest 1h supply/demand zone row…"`. Data side: ensure a 1h S/D row survives the cap-4 at `auto_trader_planner.go:1193`.
4. **Knob (no hardcode)** — `DayPlanConfig` field `seat_1h_zone` (pointer-bool, default ON) + FE toggle, per the R16 catalog (7 touchpoints listed below).
5. **15m stays B** — R20 verdict: 15m is an *entry* TF, not a zone-defining TF; keeping it capped below A matches the sources.

### 5. Exact edit-site map (from R13/R14/R15/R16/R17)

- **Evidence table:** `kernel/levels_score.go:112-117`
- **Floor/cap:** `kernel/levels_score.go:306-325` (+ doc comment :108-109)
- **seatHTF:** `kernel/levels_score.go:490` (`maxHTFSeats` :501), call site :373
- **Planner mandate:** `kernel/planner_prompt.go:216-221` (arg :207, Rules line :234, section :144-155)
- **HTFZones population:** `trader/auto_trader_planner.go:1183-1197` (cap 4), attach :1327
- **Executor KEY LEVELS:** `kernel/engine_analysis.go:381-385` → `BuildKeyLevelsBlock` (`levels_assemble.go:20`) — 1h/4h already treated identically; mirror the guarantee
- **Tests to update:** `kernel/levels_htf_test.go:107/127` (`want B` → `want A`), `TestSeatHTFPromotesSwingLevels` :150, prompt tests `kernel/planner_prompt_test.go:70`
- **Knob touchpoints (R16):** Go struct `store/strategy.go:911-964`; defaults `:1199-1225`; accessor `:1236-1273`; TS `web/src/types/strategy.ts:81-111`; editor defaults `DayPlanEditor.tsx:29-52`; control :536-590; i18n `web/src/i18n/plan-translations.ts:420-454`
- **Docs to update:** this report + audit §4.2 + `levels_score.go` policy comment

---

## RAW AGENT REPORTS

### R1 — Why 1h timeframe matters intraday futures
- ICT top-down ladder: Daily→H4→H1→M15; H1 = short-term direction rung; explicit ES/NQ applicability. https://innercircletrader.net/tutorials/ict-top-down-analysis/
- Golden rule quote: "The Daily controls the H4, the H4 controls the H1, and the H1 controls the M15."
- Killzones (NY-local): Asia 19:00-22:00, London 02:00-05:00, NY 07:00-09:00/10:00, London close 10:00-12:00 ET. NY killzone best for NQ/ES. https://innercircletrader.net/tutorials/master-ict-kill-zones/
- Quantum Algo gates signals on 4H·1H·15m alignment; "Multi-Timeframe · 1H+". https://quantum-algo.com/
- MAK Trading School S/D zones: "fresher the zone, the better chances of it working." https://maktradingschool.com/best-price-action-trading-strategy-using-supply-and-demand-zones/
- Wikipedia S/R: multi-touch = significance; level flip on break. https://en.wikipedia.org/wiki/Support_and_resistance
- Implication: bot plans carry few H1 levels because the prompt ladder skips the H1 rung. No academic stat for "H1 zones hold X hours" — practitioner folklore.

### R2 — ICT/SMC 1h rules for futures
- MTF pairing (quantum-algo.com): day = 4H bias → 1H setup → 15M entry; scalping = 1H bias → 15M setup → 5M entry; swing = Daily → 4H → 1H entry. TFs spaced 4–6×.
- Kill zones (ictkillzone.com): London 2–5 AM, NY Open 8:30–11 AM ET; setups must form inside a kill zone or be skipped.
- 1h stats: FVGs "fill ~70–80% of the time on the 1H+"; cleanest setups when "daily markup, 4H clean OB, 1H confirms with CHoCH".
- Ladder: monthly → weekly → daily → 4H → 1H → 15m → 5m → 1m. Conflict rule: higher TF always wins.
- Trade unmitigated HTF (4H/1H) OBs/FVGs — first touch highest probability; 2–3 touches = mitigated, skip.
- URLs: ictkillzone.com/ict-kill-zones, /ict-macro-times, /ict-daily-bias; quantum-algo.com/glossary/multi-timeframe-analysis/, /academy/multi-timeframe-mastery/, /glossary/fair-value-gap/.

### R3 — MTF confluence role of 1h
- Universal three-tier: HTF = trend, MTF = setup/context, LTF = entry. BrokerAnalysis: Day Trading = 4H → 1H → 15M. FuturesHive (futures-specific): ES/NQ = Daily+4H+15M; MNQ = Daily+4H+5M; "4H is the critical bridge".
- No source publishes quantified win-rate deltas for skipping the 1h; all qualitative ("lower win rates, larger drawdowns").
- 4:1 ratio confirmed: adjacent TFs 4–6× apart (Rayner: "factor of 4 to 6… avoid getting lost in the noise").
- Quotes: BrokerAnalysis "Use timeframes 4-6x apart. E.g., 4H, 1H, 15M"; United-Daytraders "The exact timeframe pairing matters less than having enough separation".
- URLs: brokeranalysis.com/blog/multi-timeframe-analysis-complete-guide/, tradingwithrayner.com/multi-timeframe-analysis/, futureshive.com/blog/multi-timeframe-analysis-futures-trading-2026, united-daytraders.com/blog/multi-timeframe-analysis-precise-entries.

### R4 — CME session structure vs 1h bars
- CME Globex equity index futures: Sun 17:00 CT open → Fri 16:00 CT close; daily maintenance halt 16:00–17:00 CT. RTH = 8:30–15:00 CT; ETH = overnight Globex.
- Session phases (CT): open drive 8:30–10:00; mid-morning 10:00–11:30; lunch dead zone 11:30–13:00; late-day trend 13:00–15:00 (MOC flows into 15:00 cash close).
- RTH = 6–7 hourly candles → each phase = 1–2 candles; a 4h bar cannot separate open/lunch/close. 1h matches a 1–6h holding horizon.
- URLs: cmegroup.com/trading-hours.html; cmegroup.com E-mini specs pages; en.wikipedia.org/wiki/Extended_hours_trading.

### R5 — Academic evidence (honesty section)
- Osler 2000 (FRBNY EP Review 6(2)): FX S/R levels predict intraday trend interruptions; varies by market/firm.
- Osler 2003 (JF 58(5):1791): take-profits cluster at round numbers; stops cluster just beyond.
- Chung & Chiang 2006 (J. Futures Markets): price clustering ubiquitous in index futures (tick-level, not predictive S/R).
- Donaldson & Kim 1993 (JFQA): DJIA restrained at 100-multiples.
- Zapranis & Tsinaslanidis 2012: algorithmic S/R weak + definition-dependent. Sullivan et al. 1999: data-snooping.
- No peer-reviewed hourly-S/R study for CME index futures exists. Treat 1h S/R as extrapolation from Osler.
- URLs/DOIs in report body.

### R6 — Published 1h S/D zone rules
- Drawing (forextradelab): base 1–5 small-bodied candles; zone = base only, never the impulse; departure ≥3 strong candles, low wicks.
- JustMarkets (Seiden 4 models): RBD/DBR reversal stronger than RBR/DBD; "base = accumulation zone"; cancel zone if price passes the middle/POC of the base.
- Freshness: each touch consumes orders; 2–3 touches = depleted. Seiden: faster exit, less base time, farther departure, fresher = stronger.
- Time decay (PriceActionNinja): "1m/5m/15m – 1 day; 30m/1h/4h – 20 days; Daily – 3 months". Seiden's method "from M30 and above" → 1h qualifies.
- URLs: forextradelab.com/blog/supply-demand-zones-forex-trading-guide/; justmarkets.com/trading-articles/learning/4-models-of-supply-and-demand; priceactionninja.com/sam-seiden-supply-and-demand.

### R7 — Platform conventions 1h vs 15m/4h
- NexusFi "Standard Stack": Weekly regime → Daily direction → 4H zones → 1H timing → 5m trigger; ES/NQ day trade = Daily→4H→15m→5m.
- ATAS pairing: 1h trend ↔ 15m entry; 4h trend ↔ 15–30m signal; daily trend ↔ 1h entry.
- 15m = standard ES/NQ execution/trigger frame (structural shift at 4H zone, stops under 4H must-hold).
- URLs: nexusfi.com/a/strategies/multi-timeframe-analysis-futures; atas.net/blog/multiple-time-frames-how-to-use-them-in-your-trading/; bookmap.com/blog/how-to-use-multi-time-frame-context-in-order-flow-trading; tradingview.com/support/solutions/43000591555-leveraging-multi-timeframe-analysis/.

### R8 — 1h order block reliability
- LiquidityScan pairing: Daily/Weekly→1H/15m (swing); 4H→15m/5m (day); 1H→5m/1m (intraday); 15m→1m (scalper). 1h = mid-tier zone TF.
- ICTKillzone 220 NQ OBs (2023–2025): T1 hit 74% first test, 58% second, 41% third — "tested twice = invalid".
- Unmitigated = live until a candle BODY closes through (wicks don't mitigate). No calendar expiry; touch-count freshness.
- Timing matters: 9:45 AM OB ≫ 2:00 PM dead-zone OB.
- URLs: ictkillzone.com/ict-order-block; liquidityscan.io working slugs listed in body.

### R9 — 1h FVG/iFVG rules
- Converged grading: 4H/Daily FVGs = bias/magnet; **1H/30m/15m = "actionable zones"** (DailyPriceAction quote); 5m/1m = execution.
- ICTKillzone 280 NQ FVGs (Silver Bullet 2024–2025): body ratio >70% → avg 2.1R; 20–60 pt gaps fill at 50% CE most often (sweet spot 20–80 pts).
- Tradeify: FVGs do NOT always fill. FluxCharts: higher TFs more consistent; 1h fills slower but respects more.
- iFVG requires candle-body close through (wick ≠ inversion). "ICT introduced IFVG on US index futures — NQ… cleanest instrument." 1h iFVG = bias/confluence; 15m = trigger.
- URLs: ictkillzone.com/ict-fair-value-gap; tradeify.co/post/fair-value-gap-trading-futures-traders; dailypriceaction.com/blog/fair-value-gap/; innercircletrader.net/tutorials/ict-inversion-fair-value-gap/; tradezella.com/strategies/ifvg-trading-model.

### R10 — HTF-only risk + proximity
- ICTKillzone: perfect 5m bullish setups failed at unseen 1h resistance 83% of the time — "invisible ceiling".
- Range/timeframe mismatch: weekly/daily range for a 2-hour trade = meaningless premium/discount; "stale ranges produce stale analysis".
- TradingStats (6,142 ES/NQ days): 30m ORB consumes 41–46% of daily range; 74% of breaks resolve within 15 min; median sessions retrace 100–206% of local structure.
- FuturesHive: "Three timeframes are sufficient". ICT checklist D→4H→**1H**→5/15m: "1-hour MSS = look for the entry on the lower timeframe".
- URLs: ictkillzone.com/ict-market-structure, /ict-premium-discount; futureshive.com; united-daytraders.com; tradingstats.net/orb-breakout-strategy-guide/; investopedia.com/terms/r/reversal.asp.

### R11 — Level-map size conventions
- The Forex Guy (Dale Woods): "anything over 3 lines… too busy"; 4h chart "just two levels"; "mark one or two intraday levels and move them as price progresses".
- Investopedia: significance grows with touches, move size, volume; S/R are zones, not lines.
- Budget implied: ~2/side from HTF, ~1–2/side intraday near price; cap 6–8 total, truncate by proximity.
- URLs: theforexguy.com/how-to-draw-support-and-resistance/, /support-and-resistance-mistakes/; investopedia.com/trading/support-and-resistance-basics/.

### R12 — Numeric 1h evidence proposal
- No source publishes cardinal weights; all ordinal. Freshness decay 1.0/0.78/0.55 (ictkillzone) matches our 1.0/0.8/0.6/0.5 ladder.
- Key finding: a fresh 1h OB at conf=3 already computes to 1.152 (A) — the ONLY blocker is the 15m/1h A→B clamp (`levels_score.go:311-322`).
- Proposal: 1h OB 0.60→**0.70** (A at conf≥1); 1h FVG/S/D 0.55→**0.65** (A at conf≥2, or conf=1 with reversal); strict variant 0.70 all kinds. 4h stays senior via TFMult 1.3 vs 1.2. Optionally nudge 15m for ladder monotonicity.
- URLs: ictkillzone.com/ict-order-block; liquidityscan.io scanner article; brokeranalysis.com; forextradelab.com; snappchart.app confluence article; quantum-algo.com/glossary/displacement.

### R13 — Code map: grading/seating edit sites
- (a) `zoneEvidenceByKind` kernel/levels_score.go:114-118 (1h: OB 0.60, FVG/SD/IFVG 0.55).
- (b) Floor/cap inline switch `ScoreLevels` kernel/levels_score.go:306-325 (`case "15m","1h": C→B, A→B`; `case "4h": C→B`; default 1m→C). Policy comment :108-109.
- (c) `seatHTF` :490, `maxHTFSeats=2` :501, call :373; `isHTFSwingZone` :475-488 (HTF bool only — no TF distinction).
- (d) `gradeFromScore` :387-395 (A≥1.0, B≥0.70); rank map :51; FE grade chips are tier-agnostic.
- (e) Tests: `levels_htf_test.go:107/127` (1h want B), `levels_score_test.go:198` zoneTierFor; no golden pins 1h grades.
- (f) No other tier switches; `isHTFDetectionTF` already includes 1h.
- (g) Docs: audit §4.2 lines 52-56, 88.

### R14 — Planner prompt HTF section map
- Section template `kernel/planner_prompt.go:144-155` ("## HTF zones (nearest first — confluence references, NEVER standalone triggers)").
- MUST-include mandate :216-221 (`plannerOutputContract`), arg `hasHTFZones` at :207, Rules line :234.
- `input.HTFZones` populated `trader/auto_trader_planner.go:1183-1197`: zone kinds only, `ScoreLevels(zones, price, dATR, nil, 4, prox)` → ±band, nearest-first, cap 4; attached :1327.
- New 1h-mandate slot: `planner_prompt.go:220` (needs `has1hSDZone` signal) + ensure 1h S/D survives cap-4 at :1193.
- Tests: `planner_prompt_test.go:70` (section + mandate), :38, :56; `levels_score_test.go:166`.

### R15 — Tests/goldens affected
- ONE hard breaker: `kernel/levels_htf_test.go:107` `TestHTFZoneGradesB` line 127 (`want B (floor/cap)` → `want A`).
- `TestZoneReversalBonus` :140 relative-only — breaks only on asymmetric raise.
- `TestScoreLevels` :47 (1m tier C) safe; `TestZoneTierForUnknownFallsBackTo1m` :196 safe.
- No golden contains a 1h zone row. Non-breakers with hardcoded input grades: planner_prompt_test.go:73, plan_overlay_roundtrip_test.go:25, wake_levels_test.go:189, w4_overlay_executor_test.go:22, ask_planner_test.go:12.
- Also update comment blocks: levels_htf_test.go:107, levels_score.go:109, 308-309.

### R16 — Settings-knob patterns
- Pointer-bool default-ON: `DayPlanConfig` W6 knobs (store/strategy.go:911-964), accessors :1236-1273, `wakeBoolPtr` :1228, defaults :1199-1225. OFF-default plain bool: `WakeOnHTFOB`.
- Env knob: `TransitionMaxMin` (transition.go:24-34), `ConfluenceCap` (levels_score.go:154-162).
- New field touchpoints (7): Go struct :911-964; defaults :1199-1225; accessor :1236-1273; TS type strategy.ts:81-111; editor DEFAULT DayPlanEditor.tsx:29-52; control :536-590; i18n plan-translations.ts:420-454.
- FE ranges: NumberField clamps; proximity 0.5–3.0, max_levels 3–12, scenario_cap 1–5, replan_cap 0–4, wake interval 5–120.

### R17 — Executor KEY LEVELS map
- `BuildKeyLevelsBlock` kernel/levels_assemble.go:20 → `AssembleScoredLevels` :35 (detector chain) → `RenderKeyLevelsBlock`.
- Executor injection: engine_analysis.go:381-385 adds `DetectHTFLevels(..., ["1h","4h"], ...)` as extra → 1h and 4h treated identically; seating = `ScoreLevels` :369-374 (seatHTF + seatBothSides + top-N).
- Audit §4.6 lives in `seatBothSides` :565 (early-return no-op when nothing cut; `MinSideLevels=3` :467).
- 1h-guarantee slot: extend `seatHTF` :490 (TF must survive into ScoredLevel — today only HTF bool does).
- Tests: levels_assemble_test.go:11, levels_htf_test.go:150 `TestSeatHTFPromotesSwingLevels`, levels_score_test.go:105.

### R18 — Broad practitioner consensus sweep
- Consensus-ranked reasons for 1h: 1) HTF context/bias filter; 2) confirmation screen for 15m/5m entries (FXOpen: 15m primary + 1h secondary); 3) fewer cleaner signals; 4) 4–6× natural pairing; 5) fits hour-length holds.
- Investopedia/Elder Triple Screen: day traders <1h holds → 10-min intermediate, **1h long-term**.
- Contradiction: one FXOpen article calls 15m/30m "swing" frames; scalping content ignores 1h.
- URLs: investopedia.com/articles/trading/03/040903.asp; fxopen.com blog x4; blueberrymarkets.com/academy/the-best-time-frame-for-forex-trading/; babypips.com/learn/forex/high-school.

### R19 — Proximity-band math for MNQ
- NQ daily range: normal regime 200–450 pts; daily ATR(14) 250–400; measured RTH avg 254.4 pts (n=1,670 sessions); first-hour IB 144.5 pts ≈ 57% of day.
- Zone sizes: 1h ATR ≈ 51–98 pts → 1h bases 20–100 pts; 4h bases ≈ 100–300 pts.
- Example: price 29,200, dATR 250 → band ±375 (28,825–29,575): **8–12 1h zones inside** vs **2–4 4h zones**. The in-band table is effectively a 1h/15m table with occasional 4h guests.
- URLs: cmegroup.com micro/e-mini pages; steady-turtle.com/statistics/initial-balance-break-probability; volatilitybox.com/research/nq-futures-volatility/.

### R20 — Re-verification of audit item 4.2 sources
- forextradelab: zones on Daily/4H; entries on 4H **or 1H**; quality scoring TF-agnostic; 15m not a zone TF.
- justmarkets (Seiden): method "from M30 and above"; no TF grading.
- liquidityscan: original cited URL 404 (superseded); "Identify on 4H and 1H; refine on 15m"; A-grade = 4/4 quality gates, TF-agnostic; table 4H→15m/5m (day), 1H→5m/1m (intraday).
- ictkillzone: 4 validity criteria, no TF grades; 15m OBs + 5m entries under daily bias.
- brokeranalysis: day trading = 4H/1H/15M (trend/setup/entry).
- **Verdict:** making 1H A-capable is defensible; keeping 15m below A MATCHES the sources (15m = entry TF, not zone-defining). Caveat: forex/crypto education sources, not CME empirical data.

---

## Next steps (pending owner go)

Implement items 1–5 of the synthesis (1h A-cap + 1h seat guarantee + prompt
mandate + `seat_1h_zone` knob + tests), then: build → targeted tests → full
suite EXIT=0 → FE build → deploy ritual (rev marker + BOOT INTEGRITY + goldens).
