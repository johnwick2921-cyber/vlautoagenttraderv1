import io,sys
P='/home/hoang/nofx-vet-05/docs/superpowers/reports/2026-09-05-vet-05-execution.md'
s=open(P,encoding='utf-8').read()
E=[]
def rep(old,new,tag):
    global s
    if s.count(old)!=1:
        print("!! FAILED (%d matches) %s"%(s.count(old),tag)); E.append(tag); return
    s=s.replace(old,new); print("ok",tag)

# ---------- SUMMARY ----------
rep("""2. **[A, S/BROKEN] The bound strategy has its daily-loss enforcement disabled.** Strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` has both master and daily-loss switches false. The reviewed dev revision adds an entry-block latch; it does not flatten an open position. A configured dollar amount is not an active cage. Source and fresh selected-field read: Q6 / q32.""",
"""2. **[A, S] The daily-loss switches are off by the owner's dated choice — but the wiring that would make a trip bite is not in the running binary.** Strategy `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` has master and daily-loss false. **That is not a defect and I withdraw the BROKEN label**: it is the owner's deliberate learning-mode setting, on the record since 2026-08-17 (`2026-08-17-cto-final-verification.md:13` "the owner's deliberate decision"; `2026-08-26-settings-census.md:25` "owner's dated choice"; `2026-08-19-controls-runtime-verify.md:84` runtime-verified). What survives is a revision gap: `git show 36648655:kernel/risk_limits.go | grep -c SetDailyForceFlat` = **0** against **2** on `2a66d91c`, so the #91 entry-block latch is on dev and **not in the running rev**. Switch the master on today and a trip would still only skip a decision cycle. The block-vs-flatten scope needs no owner ruling either — `kernel/risk_limits.go:164-167` already rules it deliberately.""","sum-p2")

rep("""3. **[A, A/BROKEN] Entry accounting lags the broker, while excursion telemetry is empty.** Arm 35 filled at 09:03:53, position 591 materialized at 09:05:14, and the arm-fill update was consumed at 09:05:35. The separate fill cache is not wholly blind, but the arm ledger lagged 102 seconds. `trade_excursions` still has zero rows (q31); stored position extrema cannot recover excursion timing.""",
"""3. **[A, A/BROKEN] Entry accounting lags the broker by 102 seconds.** Arm 35 filled at 09:03:53, position 591 materialized at 09:05:14, and the arm-fill update was consumed at 09:05:35. The separate fill cache is not wholly blind, but the arm ledger lagged 102 s. `trade_excursions` still has zero rows — **and that part is now explained, not open**: the writer (`44d4bbb7`, 2026-09-02 23:46:19 CT) is absent from every binary that ever held a position. Stored position extrema still cannot recover excursion timing.""","sum-p3")

rep("""3. **Repair measurement before changing exits.** The valid performance cohort contains 21 winners, 42 losers and two scratches; the MAE/MFE cohort loses one winner because its measurements are NULL. Keep these denominators separate. I would shadow a mechanical invalidation exit at one contract, retaining the existing stop/target/EOD baseline and SIM routing.""",
"""3. **Repair measurement before changing exits.** The compliant cohort is **18 winners, 38 losers, two scratches (n=58)**, and every eligible row carries `mae`/`mfe` — performance and excursion share one denominator. Two winners (**569, 584**, both `source='reconcile'`) carry `mae=0.0` written by the pre-E4 path that stored 0 where it could not compute; those are unmeasured, not unhurt. I would shadow a mechanical invalidation exit at one contract — but only with its cost side on the page: **5 of 18 winners (27.8%)** had MAE beyond the worked trigger.""","sum-o3")

# ---------- EVIDENCE CONTRACT ----------
rep("""Exclude test source, NULL corrected P&L, unresolved/unresolvable rows and correction notes. **71 era rows, 65 eligible**; excluded position ids **572–574, 576, 577, 579**. NULL excursion fields are excluded only from excursion calculations.""",
"""Exclude test source (`source='e7_farside_test'`), NULL corrected P&L, and the attribution sentinel `plan_id='UNRESOLVABLE'`. **71 era rows, 58 eligible**; excluded position ids **572, 573, 574** (test seam), **576, 577, 579** (`pnl_corrected` NULL), and **530, 539, 545, 546, 566, 571, 580** (`plan_id='UNRESOLVABLE'`, Σ **−$97.50**). A further **516** all-time rows are pre-era. **CORRECTION, my error [A].** The draft announced this exclusion and did not execute it: `q31_verified.py:19` filters `close_reason` and `pnl_correction_note` and never tests `plan_id`, so all seven sentinel rows sat inside the performance set and every primary figure ran on 65 rows where the rule gives 58. `SELECT id,plan_id,source,pnl_corrected FROM trader_positions WHERE entry_time>=1786770000000 AND plan_id='UNRESOLVABLE'` → 530(+13.0), 539(−84.5), 545(−29.5), 546(−88.5), 566(+97.0), 571(−44.5), 580(+39.5). The arithmetic was right; the population was not. Every figure below is re-cut on **58**; the 65-row cut appears only where labelled **sensitivity**. Re-derivation: `~/nofx-analysis/vet-05-0905/revise/r01_compliant.py` → `r01_compliant.out`. The seven sentinels exactly reconcile the two totals: −563.93 + 97.50 = −466.43.""","contract-pop")

rep("""- Performance row set **P** = `521–571, 575, 578, 580–591`. Each figure below names ids directly or an explicit keyed manifest.""",
"""- Performance row set **P** (n=58) = `521–529, 531–538, 540–544, 547–565, 567–570, 575, 578, 581–591`. Sensitivity set **P65** = `521–571, 575, 578, 580–591`. Each figure below names ids directly or an explicit keyed manifest.""","contract-P")

# ---------- Q1 ----------
rep("""`🛑 arm stop NY S1 leg 1 short: stop 29354.91 (authored 29340.00 WIDENED) · anchor ONH
29293.00 → beyond 29293.50`. A later-cycle floor observation (not the placement-time ATR): `📓 read facts: void=20 · floor=67.6 pts
(1.5×ATR5m 45.10)` (09:06:54). The placement stop implies 69.91 pts of risk; the later 67.6-pt floor is a different observation. Composition drifted between
29349.90 (09:00:54) and 29354.91 (09:02:54). Code: `kernel/min_sl.go:33` `MinSLATRMultDefault =
1.5`; `:39` `MinSLTickClearance = 2` (the "beyond 29293.50").""",
"""`🛑 arm stop NY S1 leg 1 short: stop 29354.91 (authored 29340.00 WIDENED) · anchor ONH
29293.00 → beyond 29293.50 · atr_floor 29354.91 (1.5×ATR5m 46.61) · bound=atr_floor`
(`nofx_2026-09-02.log:84978`). **CORRECTION, my error [A]:** the draft truncated this line
before `atr_floor` and then declared the placement-time ATR unavailable. It is on the record —
ATR5m **46.61**, floor 1.5 × 46.61 = **69.915 pts**, 29285 + 69.915 = 29354.915 = the composed
stop — and `bound=atr_floor` says the ATR leg, not the 2-tick anchor clearance, set it. The
09:00:54 line reads the same shape: `atr_floor 29349.90 (1.5×ATR5m 43.27) · bound=atr_floor`
(`:84939`). The 09:06:54 read-facts floor of 67.6 pts (1.5×ATR5m 45.10) is a later, separate
observation. Composition drifted between 29349.90 (09:00:54) and 29354.91 (09:02:54). Code:
`kernel/min_sl.go:34` `MinSLATRMultDefault = 1.5`; `:40` `MinSLTickClearance = 2` (the "beyond
29293.50") — the draft cited :33/:39, which are the comment lines above each const.""","q1-atr")

rep("""**order_update → Go [A].** `📡 armed order_update summary (1-line/min): frames=1
initialized=1` and `⚡ armed fill S1 @ 29285.00 (entry_class=armed_fill — stale_reeval NOT
applied)` at **09:05:35** — 102 s after NT8's Filled""",
"""**order_update → Go [A].** `⚡ armed fill S1 @ 29285.00 (entry_class=armed_fill — stale_reeval
NOT applied)` at **09:05:35** (`nofx_2026-09-02.log:85031`) — 102 s after NT8's Filled. That
line is emitted only inside `onArmedOrderUpdate`'s `"filled"` branch (`armed_executor.go:1216`),
so a filled frame was demonstrably consumed at that second. **CORRECTION, my error [A]:** the
draft also offered the `📡 armed order_update summary (1-line/min): frames=1 initialized=1`
line (`:85029`) as fill evidence; it reports no fill state. `logArmedOrderUpdateSummary`
(`33de2bef:trader/armed_executor.go:1048`, emit `:1065`) is throttled to 60 s and `Swap`s its
counters on emit, so it fires on the first frame of a drain batch; this order's `filled=1`
surfaces in the next summary at 09:20:47 (`:85624`, `frames=12 … filled=1`)""","q1-summary")

rep("""`trader/ninjatrader/reconcile.go:436` **09:05:14** `MATERIALIZED untracked NT8 position MNQ SHORT qty=1 @""",
"""`trader/ninjatrader/reconcile.go:445` **09:05:14** `MATERIALIZED untracked NT8 position MNQ SHORT qty=1 @""","q1-reconcile")

rep("""`cited_scenario_id=S1`, `plan_matched=1`, `adherence_grade=A`. The executor's own
`materializeArmedEntry` (`:1233`) ran at 09:05:35 and reported `position row not materialized
yet — stamp pending (reconcile completes it)` — 21 s after the reconcile row already existed
[B: the stamp's lookup did not find it; the final row is stamped anyway].""",
"""`cited_scenario_id=S1`, `plan_matched=1`, `adherence_grade=A`. **CORRECTION, my error [A]:**
the draft attributed the 09:05:35 `⚡ armed fill S1 @ 29285.00: position row not materialized
yet — stamp pending (reconcile completes it)` line to `materializeArmedEntry`. It cannot come
from there. `git grep -n "stamp pending" 2a66d91c -- '*.go'` returns exactly one hit,
`trader/armed_executor.go:1303`, inside `stampArmedFillLineage` (`:1292`);
`materializeArmedEntry` (`:1233`) hit its early return at `:1246-1248` — "already materialized
(reconcile won the race)" — and logged nothing. The real cause is sharper than "the lookup
missed": `materializeArmedEntry` upper-cases the side at `:1242`, while `stampArmedFillLineage`
passes `r.Side` **raw** (lowercase `short`) at `:1293` into a then case-sensitive
`GetOpenPositionBySymbol` (`33de2bef:store/position.go:614-618`, `side = ?`). The stamp could
never match. Fixed the same morning at **11:27:05 CT** by `664ab6b7` ("the side lookup was
case-sensitive — the fill-time stamp could never match"), 2 h 22 m after this event, and live in
the running rev.""","q1-stamp")

rep("""**Invalidation while open [A].** 09:15:01 `🎯 scenario S1 → ≈invalidated @ 29285.00 (price
accepted through the level against the trade — … display-only estimate, never exec`. The
09:10 5m bar (q34.trace_5m_bar, explicit bar id) closed 29301.50 above ONH 29293. Position open at ≈ −16 pts. Nothing acted.""",
"""**Invalidation while open [A].** 09:15:01 `🎯 scenario S1 → ≈invalidated @ 29285.00 (price
accepted through the level against the trade — … display-only estimate, never exec`. The
09:10 5m bar (`bars.rowid 440248`, o 29267.75 h 29303.75 l 29249.50 c 29301.50) closed 29301.50
above ONH 29293. Short from 29285.00, so the open excursion at that close is **−16.50 pts**.
Nothing acted. **This episode is already ruled and half-shipped.** It is the worked example of
**AUDIT-CHECKLIST class 59** ("A verdict the system published to itself and never read",
`AUDIT-CHECKLIST.md:1104-1110`, naming this arm, this fill and this −$140). The **entry** half
shipped as an owner ruling and is live in the running rev: `36648655:trader/entry_gate.go:199-206`
Leg 3, ARM PATH ONLY, wired at `:357-358`, booted `f478ed88` 2026-09-03 11:10:33 CT — two hours
after arm 35 filled. It works: the leg has since refused two arms in production
(`entry-gate REFUSED arm LONDON: … scenario S2 invalidated at 2026-09-04 02:00 CT` and
`… arm NY: … scenario S1 invalidated at 2026-09-04 09:00 CT`, with matching
`arm_refusals_0b:…:2026-09-04:{LONDON,NY}:entry_gate:invalidated` counters = 1 each). What
class 59 scoped out by design is the **exit** half — "nil = leg off, which is how the decision
path stays out of scope" — and that is the only part still open here. The draft named neither
the class nor the shipped fix.""","q1-invalidation")

rep("""**What the trace teaches.** Nine hops, four clocks (plan UTC, log CT, NT8 local, epoch-ms), one
real defect surfaced by the timing (arm ledger updated 102 s late), one by the prices (ledger stop ≠
broker stop, since fixed), one by the plan itself (invalidation with no exit).""",
"""**What the trace teaches.** Nine hops, four clocks (plan UTC, log CT, NT8 local, epoch-ms), one
real defect surfaced by the timing (arm ledger updated 102 s late), one by the prices (ledger stop ≠
broker stop, since fixed by `3b8d6cd6`), one by the case of a string (the fill-time stamp could
never match, since fixed by `664ab6b7`), and one by the plan itself — an invalidation the system
published and did not read, since ruled as class 59 and **wired on the entry path only**. Of the
four, exactly one is still open on this trade's evidence, and it is the exit.""","q1-teaches")

# ---------- Q2 ----------
rep("""There are 65 entries and 65 exits; each leg has 62 containing bars. Missing bars belong to positions **521–523**.""",
"""There are 58 entries and 58 exits; each leg has 55 containing bars. Missing bars belong to positions **521–523**.""","q2-coverage")

rep("""| Inside range | 58/62 = 93.5%, Wilson 84.6–97.5% | 61/62 = 98.4%, Wilson 91.4–99.7% |
| Adverse extreme | 1/62 = 1.6%, Wilson 0.3–8.6%; position 537 at low | 1/62 = 1.6%, Wilson 0.3–8.6%; position 585 at high |
| Favorable extreme | 0/62 = 0.0%, Wilson 0.0–5.8% | 0/62 = 0.0%, Wilson 0.0–5.8% |""",
"""| Inside range | 54/55 = 98.2%, Wilson 90.4–99.7% | 54/55 = 98.2%, Wilson 90.4–99.7% |
| Adverse extreme | 1/55 = 1.8%, Wilson 0.3–9.6%; position 537 at low | 1/55 = 1.8%, Wilson 0.3–9.6%; position 585 at high |
| Favorable extreme | 0/55 = 0.0%, Wilson 0.0–6.5% | 0/55 = 0.0%, Wilson 0.0–6.5% |""","q2-table")

rep("""All numerator and denominator ids are in `q31.fill_summary` / `fill_rows`. Extreme tolerance is half a tick (0.126 pts to avoid floating-point boundary loss). The four outside-range entries are **566, 569, 571, 580**, all late materialization proxies. The outside-range exit is **578**, a reconstructed netting close.""",
"""All numerator and denominator ids are in `q31.fill_summary` / `fill_rows`; the compliant re-cut is `revise/r02_legs_floor.py` → `r02_legs_floor.out`. Extreme tolerance is half a tick (0.126 pts to avoid floating-point boundary loss). **The headline moves materially on the corrected population**: the draft's 93.5% rested on four outside-range entries (566, 569, 571, 580) of which **three — 566, 571, 580 — are `plan_id='UNRESOLVABLE'` rows the rule removes**. On the 58-row cut the single outside-range entry is **569** (29700.00 against a 29676.50–29696.50 bar), a late materialization proxy; the outside-range exit is **578**, a reconstructed netting close. Sensitivity, P65: entry 58/62 = 93.5% (84.6–97.5%), exit 61/62 = 98.4% (91.4–99.7%). The 93.5% was an artefact of excluded rows, not of fill quality.""","q2-outside")

rep("""The protective bracket is submitted only on full `Filled` (**1353–1355**).""",
"""The protective bracket is submitted only on full `Filled` — `SubmitBracketOnEntryFill(signalId)` at **`VLTraderTCPClient.cs:1352`**, inside the `if (e.OrderState == OrderState.Filled)` at `:1350` (the draft's "1353–1355" pointed at the `workingEntries.Remove` line and two comments).""","q2-bracket")

rep("""q33's **120 bars** in 14:40–14:49 average **2,394 volume / 9.91 pts range**, versus **120 opening-bucket bars**, **13,381 / 38.24**; exact bar ids are `q33.bar_buckets`. This is volume/range, not order-book depth or expected flat slippage.""",
"""**[T] The 14:45 flat has never fired in this cohort — n=0.** Zero of the 58 eligible positions exited between 14:40 and 15:10 CT; the only 14:xx exit in the whole era is **560 at 14:30:16 (+$81)**. Its realized cost is unmeasured, not adverse. **CORRECTION, my error [T]:** the draft's tape comparison was against a single bucket with no base rate. On MNQ 1m bars, Mon–Fri, from 2026-08-19 (n=120 bars per 10-minute bucket unless noted), 14:40–14:49 averages **2,393.66 contracts/min and 9.91 pts range** — which is **0.83× the RTH bucket median (2,886.2)** and ranks **#27 of 39** buckets from 08:30 to 14:59. It is *thicker* than every lunch bucket (12:30 → 2,125; 13:20 → 2,138; 14:10 → 1,951; 14:20 → 1,869, the day's thinnest), and it is immediately followed by 14:50–14:59 at **5,083.93**, 2.1× higher as the cash close approaches. The 08:30 bucket I compared against — **13,380.76 / 38.24 pts** — is the day's *maximum*, rank 1 of 39, 4.64× the median: the worst possible baseline. **I withdraw the "thin tape at 14:45" framing entirely.** Re-derivation `revise/r03_tape.py`; exact bar ids remain in `q33.bar_buckets`. This is volume/range, not order-book depth or expected flat slippage.""","q2-flat")

# ---------- Q3 ----------
rep("""**My ruling [I]: retain cancellation for fades; shadow alternatives, no change now.** For a future continuation experiment I would compare caps **N=0, 2, 4 and 8 ticks**, with **4 ticks (1 point)** the provisional central case. This is a bounded experimental grid, not a research-derived MNQ threshold.""",
"""**My ruling [I]: retain cancellation for fades; shadow alternatives, no change now.**

**CORRECTION, my error.** Q3 asks for the N I would set on MNQ *with the reasoning*, and the draft gave a grid of 0/2/4/8 ticks with no reasoning at all — no tick size, no spread, no ATR, no measured through-distance — while the tape that answers the question sat unqueried in the store. Here it is.

**[T, n=1,315 `touch_episodes` since 2026-08-15]** MNQ level penetration: wick p50 **5.00** / p80 **19.75** / p90 **30.00** / p95 **38.33** pts. Only **485/1,315 = 36.9% (Wilson 34.3–39.5%)** of touches stay inside 1.00 pt. A 1-point cap therefore sits in the *bottom third* of the instrument's own penetration distribution and would forgo most touches.

**The measurement basis matters more than N, and that is the real finding.** Body penetration is p50 **0.00** / p80 13.50 / p95 31.75 against wick p50 5.00 — **the median touch's body does not penetrate at all; it is a wick.** So whether you measure the cap on the trigger bar's close or its extreme moves the accept rate further than any N in the proposed grid. That is exactly the ambiguity the draft found on arm 33 and failed to connect: **0.70 pts by close, 1.70 pts by extreme** — accepted by a 4-tick cap on one basis, refused on the other. Fix the basis before arguing the number.

**There is signal, and it is not locatable yet.** Joining `touch_episodes` to `touch_outcomes` (±0.01 price, ±10 min — an approximate join, not an FK; n=89): touches that **held** (n=46) penetrate p50 **2.375** / p80 22.25, while touches that **broke** (n=32) penetrate p50 **25.25**. A tenfold median separation says an informative cut probably exists well below 20 pts — but the overlap is heavy, n=46 cannot locate it, and `penetration_pts` is an episode maximum, not the price at the instant the guard fires.

**So my grid spans the distribution instead of sitting under it: N = 0, 4, 10, 20, 40 ticks (0, 1, 2.5, 5, 10 pts), measured on the trigger bar's CLOSE, with the extreme recorded beside it as the pessimistic bound. I name no central case, and I specifically refuse to centre it on the held-touch median:** a cap set at the median poke is a coin flip on being filled at the worst of it, which is the opposite of what a cap is for. **[I]** I hold no MNQ-specific edge for any particular value. What I would demand before setting one: the executor's actual decision price at cancellation — not a bar proxy — joined to the subsequent 30-minute path, for at least 40 cancellations. Until that exists no N is defensible and the guard stays exactly as it is. This is a bounded experimental grid, not a research-derived MNQ threshold.""","q3-ticks")

# ---------- Q4 ----------
rep("""The largest numerical loss in aligned opportunity count is **22 → 7**, but this does not identify a defective gate: some plan versions were superseded, scenarios can change across versions, and a version appearing in the table need not have been current for a placement pass.""",
"""The largest numerical loss in aligned opportunity count is **22 → 7** — **15 of 22 = 68.2% (Wilson 47.3–83.6%)**, and by opportunity count that is this section's dominant leak, against **1 of 22 = 4.5% (0.8–21.8%)** for the malformed stop-entry I rank first. I rank the stop-entry first on **hazard, not funnel share**: a malformed order type is an unbounded live-money cost independent of how often it recurs. But the 22→7 gap is a real 15-opportunity hole that the draft named, declared undiagnosable and then gave to nobody, and I have added it to the ranked list as **1b**. It does not identify a defective gate: some plan versions were superseded, scenarios can change across versions, and a version appearing in the table need not have been current for a placement pass. Partial attribution IS available and the draft did not fetch it: the `arm_refusals_0b` counters record **11 R:R refusals** across 09-02→09-04 (ASIA 2, NY 3, LONDON 1, NY 1, LONDON 1, NY 3) and **2 invalidation refusals** on 09-04; the decision path records nothing but `decision_records.execution_log`, which mentions `entry_gate` on **5** rows (09-02) and **14** rows (09-03) of 682 and 596 decisions. A zero from any one of those is a plausible zero, not a measured one.""","q4-leak")

# ---------- Q5 ----------
rep("""Stop composition uses the farther anchor clearance and ATR floor; `kernel/min_sl.go:33,39` defines **1.5×ATR5m** and **two ticks**.""",
"""Stop composition uses the farther anchor clearance and ATR floor; `kernel/min_sl.go:34,40` defines **1.5×ATR5m** and **two ticks**.""","q5-minsl")

rep("""| Winners with measurements, n=20 | 10.875 / 18.300 / 44.638 | 67.750 / 86.600 / 138.463 |
| Losers with measurements, n=42 | 40.375 / 50.800 / 75.000 | 16.625 / 28.400 / 47.775 |""",
"""| Winners, n=18 (all measured) | 11.375 / 20.400 / 46.412 | 70.250 / 88.800 / 140.387 |
| Losers, n=38 (all measured) | 40.375 / 50.750 / 75.825 | 16.875 / 28.300 / 49.500 |""","q5-exctable")

rep("""Exact winner/loser ids are `q31.excursions`; percentiles use linear interpolation. Missing winner **566** has corrected profit **$97** and NULL extrema. It belongs in performance: **21/63 wins excluding scratches = 33.3% (Wilson 22.9–45.6%)**, average win **$114.67**, average loss **−$70.76**, payoff **1.6205**, sum **−$563.93**, mean **−$8.68** across all **65** ids P. These are corrected row P&Ls, not a new all-in commission model. q14's 20/62 win rate and 1.63 payoff improperly dropped 566.""",
"""Exact winner/loser ids are `q31.excursions` and `revise/r01_compliant.out`; percentiles use linear interpolation. **On the compliant population every eligible row has non-NULL `mae`/`mfe`, so the performance and excursion cohorts are the identical 58 rows** — the draft's instruction to "keep these denominators separate" is retired along with the row that caused it. Primary figures: **18/56 wins excluding scratches = 32.1% (Wilson 21.4–45.2%)**, average win **$125.47**, average loss **−$71.71**, payoff **1.7498**, sum **−$466.43**, mean **−$8.04** across all **58** ids P. These are corrected row P&Ls, not a new all-in commission model. *Sensitivity, P65 (not a compliant estimate):* 21/63 = 33.3% (22.9–45.6%), avg win $114.67, avg loss −$70.76, payoff 1.6205, sum −$563.93, mean −$8.68 — the seven sentinel rows are the whole difference.""","q5-perf")

rep("""`q31.exit_reasons` lists every numerator id: **stop 44/65 = 67.7% (Wilson 55.6–77.8%)**, **target 13/65 = 20.0% (12.1–31.3%)**; four manual, three limit-close and **578 unknown** account for the remainder. Stops include profitable exits **557,570** and scratches **542,551**, so “stop-hit losers” is an incorrect label for all 44.""",
"""`q31.exit_reasons` and `revise/r02_legs_floor.out` list every numerator id: **stop 40/58 = 69.0% (Wilson 56.2–79.4%)**, **target 11/58 = 19.0% (10.9–30.9%)**; three manual (**526, 538, 575**), three limit-close (**558, 560, 569**) and **578 unknown** account for the remainder. Stops include profitable exits **557, 570** and scratches **542, 551**, so “stop-hit losers” is an incorrect label for all 40. *Sensitivity, P65:* stop 44/65 = 67.7%, target 13/65 = 20.0%.""","q5-exitreasons")

rep("""Winner ids **555,557,560,569,570,575,578,580,581,582,584** have MAE/floor p50/p80/p95 **0.124 / 0.457 / 0.671**; **0/11** exceed the proxy floor (**Wilson 0.0–25.9%**). Each contributing bar id and freshness value is in `q31.floor_rows`.""",
"""On the compliant cohort the winner ids are **555, 557, 560, 569, 570, 575, 578, 581, 582, 584** (580 drops with the sentinels) — MAE/floor p50/p80/p95 **0.161 / 0.481 / 0.680**, **0/10** exceeding the proxy floor (**Wilson 0.0–27.8%**).

**Two of those ten ratios are sentinels, not measurements [A].** `569` and `584` carry `mae=0.0`, and both are `source='reconcile'` rows written by the pre-E4 path that stored 0 where it could not compute — the fix commit says so in its own words: *"an uncomputed excursion is not a zero, and writing 0 here is what made 517 closed rows unreadable"* (`trader/auto_trader_clock.go:757-759`, E4, committed 2026-09-02 23:57 CT, **after** both rows were written). A zero MAE here means UNMEASURED, not "never went adverse". Dropping them leaves **n=8** with p50/p80/p95 **0.293 / 0.527 / 0.699** and **0/8** exceeding (Wilson 0.0–32.4%). **The draft's p50 of 0.124 was not a measured median.** The qualitative verdict survives either cut.

**The loser base rate is what makes 0/10 interpretable, and the draft omitted it.** Same proxy, losers n=22: MAE/floor p50 **0.996**, p80 1.350, p95 1.536, with **11/22 = 50.0% (30.7–69.3%)** exceeding 1.0. That asymmetry is **not** evidence for tightening: loser MAE is censored *at* the stop that caused the exit, winner MAE is uncensored. The winners' 0/10 is a genuine but survivor-selected statement; the losers' clustering at 1.0 is mechanical. Neither moves the floor without the excursion timestamps the Q5 gate demands. Each contributing bar id and freshness value is in `q31.floor_rows`.""","q5-floor")

rep("""Position **591** supplies a test case: authored invalidation is a 5m close above 29293 and a later close is 29301.50. Executing there would be hypothetical: a display invalidation is not a proven executable exit at that close. I do not claim the draft's $108 improvement, scale half a one-contract position, or increase to two contracts to enable scaling.""",
"""Position **591** supplies a test case: authored invalidation is a 5m close above 29293 and a later close is 29301.50, **16.50 pts** adverse against the 29285.00 short, versus the realised stop at 29355 (−70 pts, −$140). Executing there would be hypothetical: a display invalidation is not a proven executable exit at that close. I do not claim the draft's $108 improvement, scale half a one-contract position, or increase to two contracts to enable scaling.

**And the cost side, which the draft owed and did not write [T, n=18 winners].** The same table bounds it: **5 of 18 winners had MAE beyond 591's 16.50-pt trigger — 27.8% (Wilson 12.5–50.9%), ids 529, 532, 555, 557, 578 — carrying $785.50 of $2,258.50 = 34.8% of all winner dollars.** Winner MAE p80 (20.40 pts) sits *above* that trigger. An invalidation exit at this scale is an upper bound of roughly a quarter of winners cut, not a free saving. Upper bound, because invalidation levels are per-plan rather than a fixed 16.50 pts, the trigger is a 5m **close** rather than a touch, and `mae` is the boundary-contaminated bar proxy above. **Any shadow of this must report winners-cut beside losers-saved or it is not a measurement.**""","q5-invalidation")

rep("""**Regime and day-count boundary [A].** Independent mode=ro q34 groups the same **65 positions over 12 CME session-days**, versus **14 CT calendar dates**, for **−$563.93**.""",
"""**Regime and day-count boundary [A].** On the compliant cut, **58 positions over 12 CME session-days**, and — the sentinels removed — **12 CT calendar dates**, for **−$466.43**. *Sensitivity, P65:* 65 positions, 12 CME session-days, 14 CT calendar dates, −$563.93.""","q5-regime")

# ---------- Q6 ----------
rep("""3. **[A, S/BROKEN] Daily loss.** Fresh q32 reads the bound strategy's master=false, daily=false, limit=450. `kernel/risk_limits.go:305-306` returns allow when the master is off. At `36648655`, analysis discarded the returned decision and could hold the AI cycle; at `2a66d91c`, `kernel/engine_analysis.go:183-199` records `RiskForceFlat` in the entry-block latch. `kernel/risk_limits.go:151-175` explicitly describes blocking entries, not closing positions. No revision activates false strategy switches. Owner policy must distinguish refusing new risk from flattening existing risk; a variable name is not protection.""",
"""3. **[A, S] Daily loss — a revision gap, not a defect.** Fresh q32 reads the bound strategy's master=false, daily=false, limit=450. **CORRECTION:** the draft called this S/BROKEN; it is the owner's dated, deliberate learning-mode choice, recorded across five prior reports on dev since 2026-08-17 (`2026-08-17-cto-final-verification.md:13,:162`; `2026-08-26-settings-census.md:25,:96,:127`; `2026-08-19-strategy-controls-census.md:27`; `2026-08-19-controls-runtime-verify.md:84`; `2026-09-01-full-system-audit.md:451`). Nothing is broken by the flags being false, and no owner ruling is outstanding on scope either — `kernel/risk_limits.go:164-167` already states it in the code the draft itself cited: *"SCOPE, deliberately: this blocks NEW ENTRIES only. Open positions are NOT closed. Closing them stays operator-initiated via POST /api/risk/force-flat … that policy is unchanged by this wave."* `kernel/risk_limits.go:305-307` returns allow when the master is off (func at :305, guard :306, return :307). **What is genuinely open is the revision split, and it is the thing to check before anyone trusts the $450:** `git show 36648655:kernel/risk_limits.go | grep -c SetDailyForceFlat` → **0**; the same grep on `2a66d91c` → **2**. The #91 wiring (`kernel/engine_analysis.go:183-199` → `SetDailyForceFlat` → `EntryGate`, closing both order paths) is on dev and **not in the running binary**. Section 09 asks for the same verification independently (`2026-09-05-vet-09-top-ten.md:109`, rank 3). Turning the master on today would restore the pre-#91 behaviour: skip a decision cycle, arm seam unaffected.""","q6-daily")

# ---------- RECOMMENDATIONS ----------
rep("""| 2. Resolve daily-loss policy and enforce the chosen behavior | [A] fresh false flags and block-only latch; flatten choice [I] | Owner ruling, then scoped config/code work | Triggered-loss event blocks all entry paths; existing-position behavior matches ruling |""",
"""| 1b. Diagnose authored→armed attrition (22 → 7) | [T] 15 of 22 enabled-arm opportunities since 09-02 never reach the ledger — 68.2% (47.3–83.6%), the section's dominant leak by count; cause unattributed | Per-opportunity trace of supersession vs gate refusal vs no-touch, using `decision_records.execution_log` beside the `arm_refusals_0b` counters (the decision path logs nothing else) | Opportunities lost per cause, with ids; the 11 R:R and 2 invalidation refusals already counted are the start, not the answer |
| 2. Confirm the #91 daily-loss wiring is live before trusting the $450 | [A] `SetDailyForceFlat` present on `2a66d91c`, **absent from running `36648655`**; the flags-off state is the owner's dated ruling, not a defect, and the block-vs-flatten scope is already ruled at `risk_limits.go:164-167` | Read the running rev after the next cutover; no new ruling needed. Whether the master goes ON is the owner's, and only, open call | On a trip with the master on: both order paths refuse — arm seam included — and open positions are untouched, per the stated scope |""","rec-2")

rep("""| 4. Make excursion and slippage records complete | [A] empty table, unread field; measurement before policy | Diagnose existing hooks, persist signed/side-normalized slippage and initial broker prices | Missing eligible rows and unmatched ids; 30-close instrumentation gate from Q5 |""",
"""| 4. Get one trade under the excursion writer; close the reconcile hole; persist slippage | [A] the empty table needs no diagnosis (writer post-dates every stored position); the unread `slippage_ticks` field still does | Trade once under a binary carrying `44d4bbb7`, then fix the forward gap: `excursionOnOpen` is called only from `armed_executor.go:1287`, **below** the `:1246-1248` reconcile-won-the-race early return, and never from `trader/ninjatrader/reconcile.go` — so reconcile-sourced entries (3 of the last 8 positions, 591 among them) would still write no entry half | Eligible rows with a missing entry half, by `source`; unmatched ids; the 30-close instrumentation gate from Q5 |""","rec-4")

rep("""| 6. Shadow closed-bar invalidation exits at one contract | [T] 591's authored invalidation; policy [I] | Explicit mechanical spec, telemetry, owner ruling before any change | Q5 gate; net incremental P&L confidence bound and loss breaches |""",
"""| 6. Shadow closed-bar invalidation **exits** at one contract — the missing half of class 59 | [T] 591's authored invalidation; the **entry** half is already ruled and live (`entry_gate.go:199-206`, two production refusals on 09-04), and class 59 scoped the exit out by design; policy [I] | Explicit mechanical spec, telemetry, owner ruling before any change. This is the exit complement of a shipped wave, not a new gap | Q5 gate; **winners cut beside losers saved** — 5 of 18 winners (27.8%) had MAE past 591's 16.50-pt trigger, 34.8% of winner dollars; net incremental P&L confidence bound and loss breaches |""","rec-6")

# ---------- SURPRISES ----------
rep("""- **566** is profitable but lacks extrema. The inherited draft incorrectly excluded it from win rate and payoff while including its dollars in total P&L.""",
"""- **566 was correctly excluded all along, and my draft reinstated it.** It is `plan_id='UNRESOLVABLE'`, `source='reconcile'`, +$97 — barred by the governing rule, not a wrongly dropped winner. The draft built a headline reconciliation, this surprise and a standing "keep these denominators separate" instruction on a row that should never have been in the set. With it out, `SELECT id … WHERE <compliant> AND (mae IS NULL OR mfe IS NULL)` returns **zero rows**: performance and excursion share one 58-row denominator. q14's exclusion of 566 was right; its 20/62 and 1.63 were wrong only in the other rows they carried.""","surp-566")

rep("""- Excursion hooks already exist (`trader/trade_excursion_hook.go:40,75`; `trader/armed_executor.go:1287`), but the table remains empty. “Add a writer” is premature until the missing path is diagnosed; a function's presence is not proof it ran.""",
"""- **`trade_excursions` is empty for a determined reason, and the draft called it undiagnosed when the answer was one command away.** The hooks (`trader/trade_excursion_hook.go:41,77` — the draft cited :40,:75, both comment lines) landed in `44d4bbb7` at **2026-09-02 23:46:19 CT**. The last stored position, 591, opened 09-03 09:05:14 under rev `33de2bef`, where `git show 33de2bef900d:trader/trade_excursion_hook.go` is **ABSENT** and `git merge-base --is-ancestor 44d4bbb7 33de2bef900d` is **false**. No trade has been opened since. The writer has never been in a binary that held a position; the table needs one trade, not a diagnosis. A function's presence is still not proof it ran — I simply had the commit dates to settle it and did not use them.""","surp-hooks")

open(P,'w',encoding='utf-8').write(s)
print("\nFAILED TAGS:",E if E else "none")
