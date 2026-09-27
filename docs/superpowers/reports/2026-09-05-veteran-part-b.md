# Veteran review — Part B: monitoring, execution, risk (sections 4–6)

**Sub-agent B · 2026-09-05 · READ-ONLY**
Role: thirty years in index futures, pit to screen, NQ since it listed; discretionary desks then my own automated books; LLMs inside the loop for the last stretch. I have blown up an account and I have watched a proven edge decay. Nothing below is polite about what loses money.

---

## EVIDENCE BASIS — read this before you read a number

This review ran against a **fresh clone of the repository**, not the owner's machine. Be precise about what that means:

**Could not reach (and did not fake):**

- **No running engine.** `/api/health`, `/api/expectancy`, `/api/config/resolved` are unreachable — nothing is listening. Where the dispatch asked me to call an endpoint, I read the handler and the resolvers instead and say so.
- **No SQLite store on disk.** Every query in the dispatch — `trader_positions`, `armed_orders`, `nt8_order_snapshots`, `bars`, `trade_excursions`, `decision_records` — is **BLOCKED — NO STORE IN THIS ENVIRONMENT**. I write the SQL I would have run and then answer from committed evidence, carrying that evidence's own n and interval.
- **No `~/nofx-analysis/`, no NT8 logs, no journal.**
- `docs/superpowers/plans/VL-MASTER-PLAN-v2.md` does not exist in this tree.

**Could reach, and used:**

- The full Go source, the C# AddOn (`ninjascript/`), and the whole web UI I was asked to critique.
- `docs/superpowers/SYSTEM-MAP.md` (250 lines, every knob with `file:line` and an evidence label), `AUDIT-CHECKLIST.md`.
- ~50 committed reports and — this is the part that saved the section — **committed CSV data directories**.
- **The 1E script IS committed**, at `docs/superpowers/reports/2026-09-03-mc-drawdown-data/mc_drawdown.py`, together with its input `trade_sample.csv`. **I ran it.** Section 6 carries real recomputed numbers, not quoted ones.

**Label law, kept exactly:** `[R]` researched (source named) · `[T]` measured on our own tape (n + interval + citation) · `[I]` my experience, untested here. I never dress `[I]` as fact.

**No secrets.** No `.env` value, key, token or account credential appears below. I name variables, never values.

---

## SUMMARY — the six things that matter

1. **The single largest killer of arms is the marketable guard, and it is invisible in every counter you can see.** Over 2026-09-01 → 2026-09-03, of **n=15** arms, **5 (33%) died `marketable-guard`, never reaching the wire at all** (`2026-09-04-two-day-audit-data/arms.csv`). Only **3 filled (20%)**. Seven of fifteen were never placed at the broker. The guard's cancel is a `logWarnf` with **no telemetry counter** (`trader/armed_executor.go:960-961`), so `GateBlocksPanel` cannot show it and `/api/risk/gate-blocks` never counts it.

2. **A daily-loss trip cannot flatten you, and cannot stop an arm.** `DailyGuardrails.Check()` returns `RiskForceFlat` (`kernel/risk_limits.go:245`) — and the **only** production caller throws the verdict away: `} else if _, gErr := g.Check(); gErr != nil {` (`kernel/engine_analysis.go:183`), then merely skips the decision cycle. `RiskForceFlat` has **zero non-test consumers** in the repository. Worse: the guardrail block lives only in the decision path. Neither `trader/entry_gate.go` nor `trader/armed_executor.go` contains the string `daily` or `guardrail`. **A resting arm fills through a tripped daily loss limit.**

3. **Slippage is measured by the broker and thrown in the bin.** The C# AddOn computes it — `slippageTicks = (e.AverageFillPrice - origEntry) / tick` (`ninjascript/VLTraderTCPClient.cs:1383`) — and ships it on every fill frame (`:1430`). Go declares the field (`provider/ninjatrader/tcp_framing.go:86`) and **never reads it**: no store column, no telemetry, no report. The only non-test writes set it to `0` in a mock.

4. **Every armed fill in this store printed the authorized price to the tick.** n=3 fills over the audit window, all three with `price_traded_through = YES`, all three filling at exactly the arm's `entry_px` (arms 24/28/35 at 29138.00 / 29082.75 / 29285.00 → positions 584/586/591 at the identical prices). That is the SIM's fill-on-touch model, not an execution result. It is the optimistic half of the lie.

5. **Updated 1E, run against the newest trade in the tree (n=65, adding 591):** mean **−$8.68**, sd **$101.16**, se **$12.55**, **95% CI [−$33.27, +$15.92]**, t = −0.691, **n required ≈ 1,067 trades**. Median 50-trade drawdown **$934**, p95 **$1,767**; 100-trade p95 **$2,771**. Post-0B is **n=3, all losses** (−155, −99, −140). The sample cannot distinguish −$8.68 from zero and will not for about a year.

6. **The Guide teaches a knob that no longer exists.** `ARM_MIN_RR` was deleted from production Go — only tests reference it, asserting it is dead (`trader/settings_r123_test.go:35-37`) — yet the Guide still lists **`ARM_MIN_RR = 2.0`** among the "9 env knobs" (`web/src/guide/content/settings.ts:671`) and teaches it as the governing floor in `guards.ts:90` and `faq.ts:76`. The owner is being told to turn a dial that is not connected.

---

# 4 · MONITORING — what the owner sees vs what a trader needs on screen

## 4.1 The expectancy table is the best-built object in this system, and it is showing you half of what it fetched

I want to be fair before I am hard. `web/src/components/plan/ExpectancyPanel.tsx` is better statistical hygiene than most funds run. It puts **n first, on purpose** — the file says so at `:207-208`, "the floor is a property of n, so the reader meets the sample before meeting any statistic". It renders `DESCRIPTIVE ONLY` below the floor rather than a greyed-out verdict (`:260-262`). It renders an absent statistic as an em dash and **never as `0`** (`:9-11`, `:80-83`) — the distinction between "measured, and it was zero" and "not measured" is the distinction most P&L dashboards get wrong and it is pinned by test here. It carries an exclusion ledger with counts by reason (`:292-303`). It holds no copy of `min_n` or the promotion rule — both arrive in the payload (`:302`) so the page cannot drift from the binary. I have paid consultants for less.

Now the problem. The payload's `Cell` interface declares **21 fields** (`:16-45`). The row renders **nine**: label, n, w/l, mean, mean 95%, win%, MAE, status, ids. Fetched over the wire, computed by the engine, and **dropped on the floor**:

`flats` · `sum_pnl_corrected` · `sd` · `wilson_lo` · `wilson_hi` · `t_stat` · `median_mfe` · `stop_hit_share` · `target_hit_share`

Nine fields. Four of them are the four a trader actually reads first.

- **`sum_pnl_corrected`** is *the dollars*. A row showing `mean −6.62` over n=64 does not tell the owner he is down $424. Mean is the statistician's summary; **the sum is the account's**. I have never met a working trader who checks the mean before the total.
- **`stop_hit_share` vs `target_hit_share`** is the exit-quality ratio, and it is the single most diagnostic pair on a level-based system. It answers "am I being stopped out of trades that would have paid, or am I paying at targets that are too near?" It is computed. It is not shown.
- **`median_mfe`** beside `median_mae` is how you tell a stop problem from a target problem. MAE alone tells you only that you were under water.
- **`sd`** is how the reader sizes. Without it the 95% interval is a black box.

**[I]** Ship `sum` and the stop/target share pair into the row. They cost you nothing — the engine already computed them and the wire already carries them.

**Second problem: the table is collapsed by default** (`:97 useState(false)`, comment `:131-134`). The as-of clock rides the closed toggle, which is a genuinely thoughtful touch — a reader deciding whether to open needs to know how stale it is first. But a system with **expectancy statistically indistinguishable from zero** should not require a click to see that. **[I]** The one number that must never be behind a disclosure triangle is the one that says you do not yet have an edge.

## 4.2 There is no drawdown on screen. Anywhere.

I grepped the whole of `web/src` for drawdown, daily P&L, unrealized P&L on the plan surface. What exists:

- `web/src/components/trader/PositionHistory.tsx:882-892` renders `max_drawdown_pct` — a **percentage**, from the crypto-era grid stats block (`web/src/types/trading.ts:215`, sibling of `GridConfigEditor.tsx:24 max_drawdown_pct: 15`). It is not the MNQ dollar drawdown and its thresholds (`≤10` green, `≤20` amber, `:886-890`) are percentage-of-equity bands inherited from a different instrument class.
- `TraderDashboardPage.tsx:912,1001` renders per-position `unrealized_pnl`.

That is it. **The plan card — the surface the owner actually watches during a session — carries no session-day realized P&L, no drawdown-from-peak, and no distance-to-limit.** `PlanCard.tsx:8-22` imports SessionTimelineStrip, SessionTabs, SessionPlanCard, Reread/Reset/Approve, AlertCenter, GateBlocksPanel, ExpectancyPanel, InstrumentsDrawer. Nothing that says how much money today has cost.

This matters more than it sounds. Section 6 shows a **median 50-trade drawdown of $934** and a **p95 of $1,767**. Those numbers live in a markdown file. **[I]** A trader who cannot see his drawdown against its expected distribution will interpret a perfectly ordinary $900 dip as breakage, intervene, and destroy the only thing that could ever have told him whether the system worked. That is not a hypothetical failure mode; it is the most common way a working automated book gets killed by its owner. The drawdown quantiles from 1E are **the** missing widget: a single strip reading "DD today $X · DD this 50 $Y · p50 $934 · p95 $1,767" would convert panic into arithmetic.

## 4.3 You cannot ask the running binary whether your risk limits are on

`GET /api/config/resolved` is described in its own header as the place a client asks *"what does this build actually honour?"* (`api/config_resolved.go:3-4`). `buildResolvedFields` (`:79-124`) narrates **exactly three** fields:

| Path | Line |
|---|---|
| `risk_control.min_risk_reward_ratio` | `:93` |
| `day_plan.plan_mode` | `:106` |
| `regime.htf_veto` | `:118` |

Not `guardrails_enabled`. Not `daily_loss_limit_usd`. Not `max_daily_trades`. Not `max_contracts_per_order`. Not the Stage-A contract cap. Not `MIN_SL_ATR_MULT`. Not the breakeven/trailing suspension. And by the A25 rule stated at `:18-19` — *"KnobEntry holds no values, and this payload adds none"* — the knob registry half of the response is classification only.

So the honest enumeration of your resolved risk knobs, read from code because the endpoint cannot be called:

| Knob | Resolved default | Source | Live? |
|---|---|---|---|
| guardrails master | `true` when unset | `kernel/engine_analysis.go:153` | **stored `false`** (report `2026-09-03-mc-drawdown.md:32`) |
| `daily_loss_enabled` | `true` when unset | `:157` | **stored `false`** (`:33`) |
| `daily_loss_limit_usd` | per-strategy, else env fallback | `:158` | 450 configured, unenforced |
| `daily_profit_enabled` | `false` when unset | `:160` | off |
| `max_daily_trades_enabled` | `false` when unset | `:163` | **off**; value 3 |
| `max_contracts_per_order` | 2, then clamped | `kernel/risk_limits.go:308-320` | clamp wins → **1** |
| `min_risk_reward_ratio` | safe default 3.0 | `store/strategy.go:76` | bound strategy carries **2.0** |
| `MIN_SL_ATR_MULT` | 1.5 | `kernel/min_sl.go` per SYSTEM-MAP:111 | live |
| `ARM_STOP_ANCHOR_MAX_ATR` | 3.0 | `trader/arm_stop_anchor.go:35` | live, **"CHOSEN DEFAULT, NOT AN OWNER RULING"** (`:33`) |
| breakeven / trailing | suspended | `trader/exit_mechs_suspend.go:35-43` | **off** |

The guardrail state does surface — in **one boot line**, buried. `trader/auto_trader_pause.go:202-206` emits `🧾 ledger boot: … · guardrails=%s · …` with **fourteen other fields on the same line**, and when the master is off the string is the bare `"master=OFF (soft-audit only)"` (`:185`). It **never states the values that would apply** if you flipped it back on, and it never says that each guardrail is *individually* disabled underneath the master — which is the actual configured state (`2026-09-03-mc-drawdown.md:32-38`). A reader who flips the master on expecting the $450 cage gets nothing, because `daily_loss_enabled` is also `false`.

**[I]** That is the boot line I would rewrite first. `guardrails=master=OFF · daily_loss=$450(OFF) · max_trades=3(OFF) · profit=$900(OFF) · size_clamp=1(ALWAYS ON)`. Ten seconds of work; it removes an entire class of "I thought that was on".

## 4.4 The gate-blocks panel does not know the names of your risk gates

`GateBlocksPanel.tsx:20-35` carries 14 human labels. Absent from the map: `strategy_studio_daily` (`kernel/engine_analysis.go:187`), `entry_gate` (`trader/entry_gate.go:478`), `task21_concurrent_cap` (`engine_analysis.go:139`), `pnl_integrity_mismatch` (SYSTEM-MAP:183). The fallback is honest — the comment at `:18-20` says an unknown gate shows its raw name rather than hiding, which is exactly right — but the owner reads `strategy_studio_daily` instead of "Daily loss limit". And the panel is scoped to counters that reset at 17:00 CT and on restart, which it also states honestly (`:6-9`).

The bigger gap is what never reaches the counter at all: **the marketable-guard cancel and the stale-arm expiry are log-only.** Gate refusals call `store.IncArmRefusal` (`trader/armed_executor.go:469`); the two cancel paths at `:941` and `:960` call nothing. Section 5 shows those two paths killed **8 of 15 arms** in the audit window. The most common thing that happens to your arms is the thing your dashboard cannot count.

## 4.5 What the plan card gets right — and it is a lot

I will not damn this by omission. `ArmedUnderBlock.tsx` is a component written by someone who got hurt and learned. Its header (`:1-12`) records the incident verbatim: on 2026-09-03 the card showed NY v3 S1 **long** while the account held a v2 S1 **short**, filled 09:03, stopped 09:20 for −$140; both scenarios were called "S1"; the owner read one and was holding the other. The fix states both provenances, armed-under first, and renders nothing when the versions agree (`:36`). A null version renders "version not recorded", **never "v0"** (`:40-45`).

That is the correct instinct applied correctly, and it is the instinct the rest of the monitoring surface needs. **[I]** Every number on a trading screen should answer "as of when, from how many, and could it be stale?" without being asked. The expectancy panel does. The instruments drawer does — `InstrumentsDrawer.tsx:10-12`: *"Every row names its SOURCE and its n… a number whose provenance is not on screen beside it is a number a reader has to trust rather than check."* The risk surface does not exist at all, so it does neither.

## 4.6 The Guide is teaching a deleted knob

`ARM_MIN_RR` is gone from production Go. The only references outside comments are tests asserting its death:
`trader/settings_r123_test.go:35-37` — *"ARM_MIN_RR is DELETED — it must not resurrect a second floor"*; `trader/armed_executor_test.go:86-88` — same.

The Guide still ships it as live:
- `web/src/guide/content/settings.ts:671` — a titled callout, **`ARM_MIN_RR = 2.0`**, inside the "9 env knobs" section.
- `guards.ts:90` — *"ARM floors (ARM_MIN_RR 2.0)"*.
- `faq.ts:76` and `settings.ts:385` — both teach that arms are refused "at ARM_MIN_RR 2.0".

The behaviour is currently identical because the bound strategy's `min_risk_reward_ratio` is also 2.0 — which is precisely the coincidence the code comment at `trader/armed_executor.go:65-68` warns about: *"They agreed by coincidence, so nothing looked wrong."* The code fixed the source and the documentation did not follow. A Studio save that moves the floor will move behaviour and leave the Guide asserting the old number.

Note also the drift the SYSTEM-MAP itself flags: the Guide's env callout is stale in the same place the map records `ARM_MIN_RR` as `DELETED [O]` (SYSTEM-MAP:125 vs SYSTEM-MAP:221). The map caught it; the Guide never got the commit. **[I]** The maintenance contract at SYSTEM-MAP:7 says every rule change updates the map in the same commit. Extend that contract to the Guide's env callout — it is the only owner-facing surface that lists env knobs at all.

---

# 5 · EXECUTION — arms, fills, and what NT8 SIM is hiding from you

## 5.1 The dispatch's queries, written out, and marked

The dispatch asked for three store computations. Here is the SQL I would have run, marked, and then answered as far as the tree permits.

```sql
-- BLOCKED — NO STORE IN THIS ENVIRONMENT
-- (1) entry vs the bar's range at fill time
SELECT p.id, p.side, p.entry_price, b.o, b.h, b.l, b.c,
       (p.entry_price BETWEEN b.l AND b.h) AS in_range
FROM trader_positions p
JOIN bars b ON b.symbol='MNQ' AND b.tf='1m'
           AND b.open_time_ms <= p.entry_time AND p.entry_time < b.open_time_ms+60000
WHERE p.status!='OPEN' AND p.pnl_corrected IS NOT NULL
  AND p.source IN ('system','armed_entry','reconcile');

-- (2) slippage proxy on the armed path
SELECT a.id, a.entry_px, a.fill_price, (a.fill_price - a.entry_px) AS slip
FROM armed_orders a WHERE a.state='filled';

-- (3) fills at the bar's extreme
SELECT COUNT(*) FILTER (WHERE p.entry_price IN (b.h, b.l)) AS at_extreme, COUNT(*) AS n
FROM trader_positions p JOIN bars b ON …;
```

I ran (1) on the committed tape instead. `docs/superpowers/reports/exports/2026-09-02-losses/bars_1m.csv` holds 509 MNQ 1m bars over the 09-01/09-02 window; `trades.csv` in the same directory holds positions 587–590.

```
$ cd docs/superpowers/reports/exports/2026-09-02-losses && python3 <<'PY'   # (script in scratch; joins entry_time to the containing 1m bar)
  id  side      entry        bar o/h/l/c          in range?  from adverse extreme  from favourable extreme
 587  LONG   29079.25   (no bar in the exported window)
 588  LONG   29082.50  29080.75/29083.25/29076.75/29080.50   True        5.75        0.75
 589  LONG   29192.50  29193.75/29197.50/29178.75/29182.00   True       13.75        5.00
 590  LONG   29193.25  29200.00/29200.00/29191.25/29193.50   True        2.00        6.75
```

**[T]** n=3 computable. All three fills sat **inside** the containing bar's range; **none printed at either extreme**. All three are `source=system` — the AI decision path, a market order — and mid-bar is exactly where a market order belongs. So the decision path's fills are not obviously fictional.

**The armed path is a different story, and it is the one that matters**, because arms are the design.

## 5.2 Every armed fill printed the authorized price to the tick

From `docs/superpowers/reports/2026-09-04-two-day-audit-data/arms.csv` (n=15 arm rows, 2026-09-01 → 2026-09-03) joined to `trades.csv` in the same directory:

| arm | side | `entry_px` | placed at broker | `price_traded_through` | position | recorded `entry_price` | slip |
|---|---|---|---|---|---|---|---|
| 24 | short | 29138.00 | YES @ 29138.00 | **YES** | 584 | 29138.00 | **0.00** |
| 28 | long | 29082.75 | YES @ 29082.75 | **YES** | 586 | 29082.75 | **0.00** |
| 35 | short | 29285.00 | YES @ 29285.00 | **YES** | 591 | 29285.00 | **0.00** |

**[T]** n=3, interval 2026-09-01 → 2026-09-03. Three of three. Zero ticks, three times, on limits the market had traded **through**.

Widen it. `verify-armed_fill-band-rows-alltime.csv` in the same directory lists **13 all-time `plan_band='armed_fill'` rows** (ids 567, 568, 569, 570, 572*, 575, 576, 577, 578, 579, 584, 586, 591; *572 is the `e7_farside_test` seam and is excluded per the exclusion law). Every recorded `entry_price` is a clean tick-grid value identical to the arm it came from: 29621.00, 29642.00, 29700.00, 29463.25, 29437.00 (×2), 29413.00 (×2), 29459.00, 29138.00, 29082.75, 29285.00.

And the audit's own tape check, which I did not have to redo: for **591**, *"29285.00 was NOT traded in the 09:05 bar (H 29283.00)"* — the recorded entry price does not appear in the bar the entry is timestamped in at all (`2026-09-04-two-day-audit-data/trades.csv`, `entry_time_caveat` column, id 591). Same for **584**: *"29138.00 was last traded in the 08:35 1m bar (H 29140.50); the 08:36 bar H is 29127.25"*.

**This is the NT8 SIM fill model, not an execution result.** A simulator that fills a resting limit the instant the market prints at the price gives you a perfect fill every time, with no queue, no size, and no partials. **[I]** In a live MNQ book, a limit resting at a level that the market runs through by 80–240 points fills *some* of the time and, when it does, you were the liquidity that got run over. The trades where your limit fills cleanly and the market keeps going are precisely the trades adverse selection was invented to describe. Three of three of your armed fills have `price_traded_through = YES`.

**[R]** This is not folklore. Adverse selection on passive limit orders is the standard result in the microstructure literature — Glosten & Milgrom (1985, *JFE*), "Bid, Ask and Transaction Prices in a Specialist Market with Heterogeneously Informed Traders", is the canonical statement: the passive side is filled preferentially by the informed side. A fill-on-touch simulator prices that at zero.

**[T]** Combined with §5.4: the honest reading is that **your armed-path fill statistics contain no information about live fill quality whatsoever**. Not "optimistic by a tick" — *no information*. n=3 realized armed fills is also, separately, far too few to say anything even if the model were right.

## 5.3 The broker measures your slippage and Go throws it away

This is the finding I would put in front of the owner first, because it is free to fix.

The C# AddOn already does the work:

```csharp
// ninjascript/VLTraderTCPClient.cs:1376-1384
double slippageTicks = 0.0;
lock (signalMapLock)
{
    if (signalEntryByOco.TryGetValue(signalId, out var origEntry)
        && signalTickSizeByOco.TryGetValue(signalId, out var tick)
        && tick > 0)
    {
        slippageTicks = (e.AverageFillPrice - origEntry) / tick;
    }
}
```

and ships it on every fill frame — `["slippage_ticks"] = slippageTicks` (`:1430`). The comment at `:67` says the intent out loud: *"Track signal_id → original entry price for slippage calculation."*

Go declares the field:

```go
// provider/ninjatrader/tcp_framing.go:86
SlippageTicks float64 `json:"slippage_ticks"`
```

and then, across the entire repository, **never reads it**. `grep -rn "SlippageTicks" --include=*.go .` returns exactly three hits: that declaration, `provider/ninjatrader/tcp_client_mock.go:221` setting it to `0` with the comment *"no slippage modelled in default mock"*, and `tcp_framing_test.go:51` setting it to `1.0`. There is no store column (`grep -rn "slippage" store/*.go` finds only `grid.go:68 SlippageTolerPct`, a crypto-grid field), no telemetry counter, no expectancy input, no UI.

**[T]** The one instrument in the whole stack that would let you compare SIM fill quality against reality on the day you go live already exists, already runs, and is discarded at the parse boundary. Persist it. One column on `trader_positions`, one on `armed_orders`. **[I]** On the day the account changes from the SIM account to a funded one, the difference between the two distributions of `slippage_ticks` *is* the go/no-go, and you will have zero baseline to compare against unless you start recording now, while it is still zero.

## 5.4 The stop side: the SIM fills at the stop price no matter how far the market blew through

The audit already refuted the "zero slippage" claim on the exit side and the refutation is sound (`2026-09-04-two-day-audit-data/verify-arm35-stop-slippage.md:42-49`): position 591's exit printed exactly 29355.00 while the 09:20 1m bar ran **h 29360.00** — the market traded **5.00 points / 20 ticks through** the stop and the SIM printed the stop. The file's own conclusion is the right one and I will not improve on it: *"under a fill-at-stop-price engine exit == working stop identically for ANY stop value, so 'exit == stop' is not a slippage measurement."*

What I can add is the distribution. From `2026-09-04-two-day-audit-data/trades.csv`, the `stop_overshoot_pts` column on the **8 stop-hit trades** (`nt8_exit_reason='sl'`), which is `mae − risk_pts_at_fill` — how far past the stop the market ran:

```
[2.00, 0.00, 13.00, 1.75, 2.00, 3.75, 2.75, 8.37]
mean 4.20 pts   median 2.375 pts   max 13.00 pts
```

**[T]** n=8, 2026-09-01 → 2026-09-03. **Seven of eight stop-hits (87.5%) had strictly positive overshoot** — the market did not touch the stop and reverse, it went through and kept going. Only 585 stopped at exactly the stop.

I want to be careful here, because the loose version of this argument is wrong. Overshoot is **not** slippage: a stop-market fills at the first print through, so how far the market travels afterwards does not set your fill price. Overshoot is an **upper bound** and a **conditioner**. What it tells you is that 7 of 8 stop exits occurred inside a *continuing* move rather than at a turn — and a continuing move is the exact regime in which a real stop-market slips.

**[I]** My charge for a stop-market on MNQ in a normal book is 1 tick; in the kind of move where price runs 8–13 points past your level in the same minute, 2–4 ticks. Applied to this sample: mean $1.00–$2.00 per stop-hit trade, more on the two worst. That is my experience, untested here, and I have labelled it as such — but note it is a *one-sided* correction. Every one of these adjustments moves the mean down and none moves it up.

**[T]** Note also the size of the recorded exit path in dollars for calibration: the stop-hit trades' `risk_pts_at_fill` runs **24.00 to 77.50 points**, median 34.25 (n=11 including winners) — **$48.00 to $155.00 at $2/pt, qty 1**, a **3.23×** spread between the tightest and widest risk you took.

## 5.5 The marketable guard is killing a third of your arms, and it is killing them permanently

This is the execution finding.

`trader/armed_executor.go:955-962`:

```go
if price > 0 && limitMarketableWrongSide(price, r.EntryPx, r.Side) {
    _ = ledger.SetState(r.ID, "cancelled", "level accepted through — marketable, never placed")
    at.logWarnf("✕ armed %s cancelled — price %.2f already %s entry %.2f (marketable, never placed)", …)
    continue
}
```

The *intent* is right and the incident it came from was real — the header at `:952-955` names it: the S2 re-place loop, fill → stop-out → re-arm → fill. Placing a limit the market has already traded through fills instantly at a worse price. Fine. But look at the three properties of the implementation together:

**(a) It runs on the scan cadence, not on the tape.** `maybeManageArmedOrders` is called once per cycle from `runCycle` (`trader/auto_trader_loop.go:432`; `armed_executor.go:149` — *"runs every cycle"*), and the cycle ticker is `at.config.ScanInterval` (`trader/auto_trader.go:936`), whose default and **minimum** is **3 minutes** (`store/trader.go:29 scan_interval_minutes;default:3`; SYSTEM-MAP:211 records min 3).

**(b) It judges on a bar close, not a quote.** `price = bars[len(bars)-1].Close` (`armed_executor.go:898`).

**(c) The cancel is terminal.** `SetState(…, "cancelled", …)` writes a terminal state, and `UpsertArm` re-authorizes a terminal row **only on a plan-version change** (`store/armed_orders.go:244-250`, MANUAL-CANCEL-WINS, per SYSTEM-MAP:127).

Put those together and the mechanism is this: a level gets touched, rejected, and left behind inside a three-minute window; the arm was still `armed` because the cycle had not come round; the next cycle sees a close on the wrong side and **kills the arm forever**. The trade the system exists to take is the one the guard is most likely to cancel, because a clean touch-and-reject at a level *is* a wick through followed by a close back — and the guard samples the close.

Now the measured cost. `2026-09-04-two-day-audit-data/arms.csv`, every arm row in the window:

| outcome | n | share |
|---|---|---|
| **`marketable-guard`** | **5** | **33.3%** |
| `filled` | 3 | 20.0% |
| `stale-arm-expiry` | 3 | 20.0% |
| `session-EOD` | 2 | 13.3% |
| `invalidation` | 1 | 6.7% |
| `cancelled-in-NT8` | 1 | 6.7% |
| **total** | **15** | |

**[T]** n=15 arms, 2026-09-01 → 2026-09-03. **The marketable guard is the single largest cause of arm death, larger than fills.** All five carry `price_traded_through = YES` and all five are `placed_at_broker = NO — never on the wire`. Seven of fifteen arms (46.7%) never reached the broker at all.

Look at how *close* some of them were. `nearest_approach_pts` on the five guard-cancels: **−1.70** (arm 33), −8.71 (25), −8.95 (36), −12.50 (27), −12.50 (34). Arm 33 was cancelled because price closed **1.70 points — under seven ticks — on the wrong side of its level.** On MNQ that is noise. That is not "the market accepted through the level"; that is a normal probe of a level.

And the stale-arm expiries are the same story from the other end: arm **31** was cancelled by `ARM_WORKING_STALE_MIN` (default 15 min, `armed_executor.go:127-133`) with a `nearest_approach_pts` of **0.70** — **under three ticks from filling**.

**[I]** Three changes, in order of how much money I think they are worth:

1. **The guard should not cancel; it should stand down.** Mark the arm `held_marketable` and re-evaluate next cycle. The reason to refuse placement is that the limit would fill badly *right now*. That is a statement about this instant, not about the rest of the session. Converting an instantaneous condition into a permanent verdict is the error, and it is the same error class the reaper was rewritten to remove — `trader/reaper_snapshot.go:29-30` gets it exactly right for orders (*"no book, or a book too old to believe. WARN, never cancel"*) and the marketable guard gets it exactly wrong for arms. The reaper's own header says the principle: **silence is not death.** A close on the wrong side is not death either.
2. **Judge the guard on the tape, not the close.** `price` should be the last *trade*, or at minimum the guard should compare the arm's level against the bar's high/low rather than its close, so a wick through does not read as acceptance. You already have OHLC in `bars` at the call site.
3. **Count it.** `armed_executor.go:960` needs an `IncArmRefusal`-equivalent so §4's panel can show the owner that a third of his arms die here.

## 5.6 The R:R gate is judged before the fill, and half the fills came in worse than the floor

`verify-rr-snapshot-vs-fill.csv` (`2026-09-04-two-day-audit-data/`) is the cleanest execution evidence in the tree:

| pos | side | snapshot entry | fill entry | slip pts | R:R at gate | floor | **R:R at fill** | sub-floor |
|---|---|---|---|---|---|---|---|---|
| 581 | SHORT | 29470.25 | 29466.00 | **+4.25** | 3.00 | 3.00 | 2.504 | **YES** |
| 583 | LONG | 29216.75 | 29213.25 | −3.50 | 3.07 | 3.00 | 3.412 | no |
| 587 | LONG | 29069.50 | 29079.25 | **+9.75** | 2.03 | 2.00 | **1.088** | **YES** |
| 588 | LONG | 29081.25 | 29082.50 | +1.25 | 2.68 | 2.00 | 2.538 | no |
| 589 | LONG | 29182.00 | 29192.50 | **+10.50** | 2.02 | 2.00 | **1.610** | **YES** |
| 590 | LONG | 29196.50 | 29193.25 | −3.25 | 2.29 | 2.00 | 2.505 | no |

**[T]** n=6 decision-path entries, 2026-09-01 → 2026-09-02. Mean entry slip **+3.17 points = +$6.33**; worst **+10.50 pts = +$21.00**. **Three of six (50%) transacted below the R:R floor the gate had just cleared them at.** Position 587 was authorized at 2.03R and taken at **1.088R** — barely half the mandated floor.

Credit where due: this was found and partly fixed. `trader/entry_gate.go:237-239` is explicit — *"Leg 5 — R:R at the REAL entry price (the fix for 587/589): the floor is judged at the price the order will transact at, not the prompt snapshot"* — and `entryGateForDecision` now passes `Entry: livePrice` (`:420`, with `:385-386` documenting livePrice as "the execution-time market price"). That closes the *stale-snapshot* half of the gap, which was the larger half.

It does not close the *wire* half. A market order gated at the live price still fills at whatever the book gives it milliseconds later, and **nothing records or alerts on the realized R:R after the fill.** `rr_at_fill` exists as a *column in the auditor's CSV*, computed by a human during a forensic wave. It is not a column the engine writes and not a number the owner ever sees.

**[I]** Post-fill R:R belongs on `trader_positions` and in the expectancy row. A system whose gate is an R:R floor and which cannot report the R:R it actually got is grading its own homework on the answer sheet.

## 5.7 What is built well, so it does not get changed by accident

**The reaper.** `trader/reaper_snapshot.go` is the best-reasoned file in the execution path. Three verdicts — `reaperUnknown`, `reaperAlive`, `reaperGone` — with **unknown as the zero value** (`:29-30`), so the failure mode of every unhandled path is "do nothing" rather than "cancel". A row with no signal id returns unknown and is checked **first**, with the reason stated (`:60-63`): *"there is no identifier to look up… every order in it will fail to match and the loop below would fall through to GONE — cancelling on ignorance."* A book older than `2×interval` is evidence in **neither** direction (`:78-82`). And the note at `:93-101` records that broker states Working/Accepted/Initialized/Submitted/CancelPending/CancelSubmitted all read ALIVE, *asserted by test rather than inherited from absence*, because — the file's words — *"safety by accident is not safety."* That is how you write an order-management safety net. Leave it alone.

**F12.** The order-snapshot frame (`2026-09-03-f12-order-snapshot.md`) fixed a genuinely dangerous hole: flat-gate leg 4 was asking `armed_orders`, **our own ledger** (`:12-20`), so it could not detect the one failure a flat gate exists to detect. The frame is account-scoped, carries `build_id`, and an empty book is `orders: []` — *"an explicit empty, never silence"* (`:49-50`). Correct in every respect.

**One open position.** Enforced twice and deliberately kept in step: `oneLiveArmGuard` (`armed_executor.go:640-671`) and EntryGate leg 7 (`entry_gate.go:277-281`), with the comment at `:653-658` recording that the arm guard used to `continue` on a same-side match and that this is how a new plan version could add to a live position. Both sides refused now. At one contract this is the correct posture and it is the reason there is no gross-exposure question in section 6.

**0B stop composition.** `trader/arm_stop_anchor.go:20-25` — beyond the nearest seated level on the risk side plus clearance, floored at `MIN_SL_ATR_MULT×ATR5m`, widest wins, **never tighter than authored**. The reasoning at `:13-18` is the correct diagnosis: *"a wider stop in a dead zone is still a stop in a dead zone: width alone is not the fix, ANCHORING is."* **[T]** backed by the week-in-review count it cites (15 of 27 losers stopped too tight; 0 of 5 of the biggest losers had a stop on a seated level). This is the right idea. See §6.3 for the cost it imposes that nothing currently bounds.

**Exit suspension.** `trader/exit_mechs_suspend.go:14-28` suspends breakeven and the ATR trail — not because the direction is known wrong, but because *"unmeasured mechanisms are moving live stops"* (`:21-22`) — and proves it at the **wire boundary**: both mechanisms reach the broker only through the `moveStopWire` variable (`:49-51`) so fixtures assert zero calls. That is the correct standard of proof and it is rare.

---

# 6 · RISK — one contract, no cage, and the honest go/no-go

## 6.1 I ran the 1E rig. It reproduces bit-for-bit.

The dispatch said run it. **The script is committed** at `docs/superpowers/reports/2026-09-03-mc-drawdown-data/mc_drawdown.py`, with its input `trade_sample.csv` (n=64, ids 521–590) alongside. I copied both to a scratch directory, installed numpy, and ran it unmodified:

```
$ cd $SCRATCH/mc && cp docs/superpowers/reports/2026-09-03-mc-drawdown-data/{mc_drawdown.py,trade_sample.csv} . && python3 mc_drawdown.py
SAMPLE  n=64  ids 521..590  sum=-423.93  mean=-6.624  sd=100.589
        wins=21 losses=41 flat=2  p(win)=0.3387
Q4 — mean=-6.624  sd=100.589  se=12.574  95% CI [-31.268, +18.020]  t=-0.527
     n_required = ((1.960+0.842)*100.59/6.624)^2 = 1,810 trades
```

**Every figure matches the published report exactly** (`2026-09-03-mc-drawdown.md:122-124`, `:174-176`, and the Q1/Q2/Q3/Q5/M6 tables). Seeded at 20260903, B=10,000, reproducible. The report is honest and the rig works.

## 6.2 Updated numbers — n=65, adding position 591

The store is unreachable, but **position 591 is the newest trade anywhere in this tree** (I checked: nothing above 591 appears in any report or CSV) and it is fully specified in `2026-09-04-two-day-audit-data/trades.csv`: NY, S1, `reconcile`/`armed_fill`, entry 2026-09-03 09:05:14 CT at 29285.00, exit 29355.00, **`pnl_corrected = −140.00`**. Session-day 2026-09-02 (the CME day rolls 17:00 CT; this is the same convention rows 587–590 use, where a 09-02 09:41 entry carries `session_day_ct 2026-09-01`). Its `created_ms` = 1788444314000 ≥ `CUT_MS` 1788353340000, so it is **post-0B**.

I appended it and re-ran the identical script:

```
$ python3 mc_drawdown_65.py          # sed 's/trade_sample.csv/trade_sample_65.csv/' only
SAMPLE  n=65  ids 521..591  sum=-563.93  mean=-8.676  sd=101.162
        wins=21 losses=42 flat=2  p(win)=0.3333
```

**Q1 — max drawdown ($, 1 contract, $2/pt), n=65, B=10,000**

| horizon | method | p50 | p90 | p95 | p99 | worst |
|---|---|---|---|---|---|---|
| 20 | IID | **506** | 866 | **975** | 1,181 | 1,663 |
| 20 | block(5) | 496 | 858 | 970 | 1,197 | 1,581 |
| 50 | IID | **934** | 1,575 | **1,767** | 2,148 | 3,067 |
| 50 | block(5) | 920 | 1,552 | 1,749 | 2,129 | 2,940 |
| 100 | IID | **1,495** | 2,478 | **2,771** | 3,304 | 4,350 |
| 100 | block(5) | 1,478 | 2,428 | 2,711 | 3,264 | 4,521 |

The two bootstraps still agree to within ~3% at every quantile — no detectable streakiness beyond IID. Every drawdown quantile is **6–10% worse** than the published n=64 table, which is what one extra $140 loser does to a 64-trade sample.

**Q2 — P(losing streak ≥ k in 50 trades) at p(win)=0.3333:** k=4 → **0.9937**; k=6 → **0.8213**; k=8 → **0.4808**; k=10 → **0.2313** (exact recursion; bootstrap 0.9909 / 0.7689 / 0.4178 / 0.1882).

**Q3 — guardrail counterfactual, 12 session-days, B=10,000:**

| rule | trips on | kept/day | forfeited/day | net effect |
|---|---|---|---|---|
| daily loss $450 | **8.3%** of days | −49.12 | +0.00 | **+0.00** |
| max_daily_trades 3 | **75.1%** of days | −72.51 | +23.09 | **−23.09** |
| both | 75.7% | −71.17 | +23.75 | −23.75 |

**Q4 — expectancy, n=65:**

```
mean = -8.676   sd = 101.162   se = 101.162/sqrt(65) = 12.548
95% CI = -8.676 ± 1.96 × 12.548 = [-33.269, +15.917]
t = -8.676 / 12.548 = -0.691
n_required = ((1.960 + 0.842) × 101.162 / 8.676)^2 = 1,067 trades
```

**Q5 — era split (0B cutover 2026-09-02 07:49 CT):** pre-0B **n=62**, mean −2.741, sum −169.93. Post-0B **n=3 — ids 589, 590, 591, values −155, −99, −140. Three trades, three losses, −$394.** No distribution claim is possible and I make none.

**M6 — sensitivity:** drop the largest win (532, +311) and the largest loss (589, −155): n=63, mean **−11.427** (worse), maxDD@20 p50 500 / p95 954 vs full 506 / 984. **The picture does not flip.** The drawdown result is not an artifact of two extreme trades.

## 6.3 A professional's risk framework at this size

The dispatch asks what a professional would run at one MNQ contract. Here it is, and most of it is already right.

**What is correct today.** Size clamped to 1 by `ClampStageAContracts` (`kernel/risk_limits.go:318`), and the reasoning at `:299-306` is the reasoning I would have written: *"ONE contract until n≥30 closed trades with a positive lower-CI expectancy. Kelly and optimal-f are undefined without an edge estimate, so size does not move on intuition."* **[R]** That is correct as mathematics, not just as caution: the Kelly fraction f\* = μ/σ² requires μ, and at CI [−$33.27, +$15.92] you do not have μ — you have a sign you cannot determine. Any position-sizing formula fed this μ returns garbage with a decimal point. Note also `:302-304` gets the interaction right: 0B raised the stop floor from 1.0 to 1.5×ATR5m, which **raises dollar risk per trade by ~50% at constant size** — which is exactly why size must not move at the same time. Two changes at once and you learn nothing from either.

The clamp is also **master-independent**: `ResolveMaxContracts`'s header (`:290-296`) states that the guardrails master and the max-contracts toggle govern only daily limits, *never* this clamp, and *"NEVER returns 0 — a futures order can never be left unclamped."* Correct. That is the one control that is genuinely protecting the account today.

**What is missing, in order.**

**(1) There is no dollar risk cap per trade, and 0B only widens stops.** `composeArmStop` (`trader/arm_stop_anchor.go:71-125`) takes the *widest* of authored / anchor+clearance / ATR floor. Nothing bounds the result in dollars. The dead-zone bound `ARM_STOP_ANCHOR_MAX_ATR = 3.0` (`:35`) caps the *anchor* leg only, and its own comment concedes `:33` **"CHOSEN DEFAULT, NOT AN OWNER RULING"**. **[T]** The realized spread: `risk_pts_at_fill` over n=11 trades ran **24.00 → 77.50 points = $48 → $155**, a 3.23× range; and the arms census in the 1E pre-registration (`2026-09-03-mc-drawdown.md:19`, n=34 arms) records median **26.75** pts, p75 40.00, and **max 150.00 points = $300 on one contract**. Your worst-case single-trade risk is six times your best-case, and it is set by whatever the ATR happens to be. **[I]** At one contract this is survivable. It is also the thing that breaks first when size moves, because at 3 contracts that 150-point stop is $900 on one trade — twice the daily loss limit you have configured and disabled. Bound it now, while it costs nothing: refuse or resize any arm whose composed stop exceeds a stated dollar figure, and log the refusal.

**(2) The daily loss limit cannot flatten and cannot stop an arm.** Two defects, both quotable.

*It cannot flatten.* `DailyGuardrails.Check()` returns `RiskForceFlat` on a daily-loss trip (`kernel/risk_limits.go:245`) and the doc comment two lines up (`:243`) promises *"Daily-loss → ForceFlat"*. The only production caller is:

```go
// kernel/engine_analysis.go:183
} else if _, gErr := g.Check(); gErr != nil {
    logger.Warnf("⚠️ Strategy Studio daily guardrail tripped: %v — skipping decision cycle (HOLD)", gErr)
    …
    return holdCycle("daily_guardrail"), nil
}
```

The verdict is discarded into `_`. `grep -rn "RiskForceFlat"` over the repo returns the two producers (`:92`, `:245`) and **three test files. Nothing in production consumes it.** `RiskLimits.Classify` — the other producer, whose whole purpose is to return that class — has **no non-test callers at all**. So a tripped daily-loss limit does not close your open position; it declines to open a new one. On a day where the limit trips *while you are in a trade*, the trade runs to its stop, its target, or EOD flat.

*It cannot stop an arm.* The guardrail block exists only inside `RunAnalysis`. `grep -n "daily\|guardrail" trader/entry_gate.go trader/armed_executor.go` returns **nothing** — neither the canonical gate (whose seven legs are inventoried at SYSTEM-MAP:141-149) nor the arm placement loop consults it. `runArmedPlacement` (`armed_executor.go:886-975`) will place a working limit and NT8 will fill it regardless of the session's realized P&L.

**[I]** For a system whose whole safety story is "SIM-only, one contract", this is tolerable. On the day it is not SIM-only it is the difference between a bad day and a stopped-out account. A daily-loss trip must (a) return a verdict somebody acts on, (b) cancel every working arm, and (c) be a leg of `EntryGate` so both seams are held to one standard — exactly the reasoning already applied to `one_open_position` at `armed_executor.go:653-658`, *"kept in step deliberately so an arm can never be held to the weaker of two standards."* Apply the same sentence to the daily limit.

**(3) Every guardrail is off, individually as well as at the master.** `2026-09-03-mc-drawdown.md:32-38` records the bound strategy: `guardrails_enabled=False`, `daily_loss_enabled=False`, `daily_profit_enabled=False`, `max_daily_trades_enabled=False`, `max_contracts_enabled=False`, with values 450 / 900 / 3 / 2 sitting behind them. The code defaults would have given you `daily_loss_enabled=true` when unset (`engine_analysis.go:157`) — the stored `false` is an explicit choice, not an omission. And the boot line reports only `"master=OFF (soft-audit only)"` (`auto_trader_pause.go:185`), so flipping the master back on restores **nothing**, and nothing on screen would tell you.

**(4) Commissions are zero in every recorded row.** `docs/superpowers/reports/exports/2026-09-02-losses/trades.csv` — the `fee` column reads `0.0` on every exported row. `pnl_corrected` is gross. See §6.4.

**(5) What I would run at this size, if you asked.** **[I]** — my experience, untested here, stated as opinion:
- Size 1 until the lower CI bound on expectancy clears zero. Already correct, already enforced.
- A **per-trade dollar risk cap** — a hard number, refuse above it. Today's implicit cap is $300 and nobody chose it.
- A **daily loss limit that actually flattens and cancels arms**, set at roughly 2× the median 20-trade drawdown so it catches disasters and not ordinary noise. From §6.2, p50 maxDD@20 is $506 — so **$450 is arguably too tight**, and the Q3 counterfactual agrees for a different reason: it trips on 8.3% of days.
- **No trade-count cap.** Q3 says `max_daily_trades=3` trips on **75.1%** of days and forfeits **$23.09/day** of realized P&L against a median 4-trade day. It is a rule that costs money and buys nothing measurable here.
- A **consecutive-loss circuit breaker** instead. §6.2 Q2 says 6 losers in a row is p=0.82 and 8 in a row p=0.48 inside 50 trades, so a breaker below ~8 fires on noise — but a breaker at 8–10 that pauses for a session gives you a forced look at the tape at the point where a regime change and a bad run become indistinguishable. **[I]**
- **Record slippage now** (§5.3). It is free and it is the only thing that will let you price the SIM→live transition.

## 6.4 Friction, which is one-sided

Every recorded row carries `fee = 0.0`, and NT8 SIM fills limits on touch and stops at the stop price. So the recorded distribution is **gross of commission and gross of all execution cost**. Applying realistic friction — commission plus adverse slippage on both sides, MNQ tick = $0.50:

| friction/trade | mean | 95% CI | t | n required |
|---|---|---|---|---|
| $0.00 (recorded) | −8.676 | [−33.27, +15.92] | −0.691 | 1,067 |
| $1.24 (comm only) | −9.916 | [−34.51, +14.68] | −0.790 | 817 |
| $2.24 (comm + 1 tick/side) | −10.916 | [−35.51, +13.68] | −0.870 | 674 |
| $3.24 (comm + 2 ticks/side) | −11.916 | [−36.51, +12.68] | −0.950 | 566 |
| $4.48 (comm ×2 + 2 ticks/side) | −13.156 | [−37.75, +11.44] | −1.048 | 464 |

**[I]** The commission figures are typical retail all-in MNQ round turns; the tick charges are my §5.4 estimate. **[T]** The arithmetic is mine, on the n=65 sample, and I show it.

Two things to take from that table. First, **the interval still spans zero everywhere** — friction does not turn "unknown" into "proven negative", and I will not claim it does. Second, and this is the part that matters: **friction is one-sided.** Every correction for something the SIM is not modelling moves the mean down. There is no plausible adjustment that moves it up. The recorded −$8.68 is therefore a **ceiling** on the true expectancy, not an estimate of it. The upper bound of the CI falls from +$15.92 to +$11.44 under a realistic charge.

## 6.5 The go/no-go for live money, in numbers

**NO-GO. Not close. Here is the arithmetic rather than the opinion.**

**The edge is not measured, and it is not nearly measured.**
mean −$8.676, se $12.548, **t = −0.691**. To reject "expectancy = 0" at α=0.05 two-sided you need |t| ≥ 1.96; you have 0.691, about a third of the way. The 95% interval is **$49.19 wide** — [−$33.27, +$15.92] — against a point estimate of −$8.68. **The interval is 5.7× the estimate.** You cannot tell a system that loses $33 a trade from one that makes $16 a trade. Both are inside your data.

**How long until you can.**
n_required = ((z_α + z_β)·sd/|μ|)² = ((1.960 + 0.842) × 101.162 / 8.676)² = **1,067 trades** at power 0.80. At the realized rate — 65 trades over 12 session-days ≈ 5.4/day — that is **~197 session-days, roughly ten months of continuous trading**. And that figure is *optimistic*: it assumes the current effect size is real. If the true expectancy is nearer zero, n_required rises as 1/μ² and goes to infinity at μ=0.

**What it costs to find out.**
Trading 1,067 trades at −$8.68 expected costs **$9,257**, plus commission of roughly **$1,300–2,600** at $1.24–2.48 a round turn. Call it **$10,500–11,900 to buy the answer**, if the answer is the current point estimate.

**What you must survive while you find out.**
From §6.2, n=65: 100-trade **p95 drawdown $2,771**, **p99 $3,304**, worst simulated path **$4,350**. Over 1,067 trades you would see roughly ten independent 100-trade blocks, so a p95-per-block event is close to certain within the sample. **[I]** My rule for capitalizing a strategy in evaluation is 3× the p95 drawdown at the horizon you intend to run, so the risk capital is **≈ $8,300**, and I would not fund it below **$6,600** (2× p99). A $2,000 or $3,000 account trading this is not undercapitalized by a little; it is mathematically guaranteed to be stopped out by ordinary noise before the sample resolves.

**What a normal bad stretch looks like, so you do not misread one.**
Inside any 50 trades at p(win)=0.333: **4 losers in a row is essentially certain (p=0.994)**, **6 in a row is the base case (p=0.821)**, **8 in a row happens about half the time (p=0.481)**, **10 in a row roughly one time in four (p=0.231)**. None of that is breakage. It is what a 33%-win-rate distribution with sd ≈ $101 does. The distinguishing signal is never drawdown *depth* — it is drawdown **without those statistics**: trades stopping, losses clustering in one condition or one session, or the win rate itself moving. **[I]** That is the alarm to build, and §4.2 is where it belongs.

**And the post-0B evidence you actually have.**
**n=3. All losses. −$155, −$99, −$140. Sum −$394.** The current configuration — 1.5×ATR stop floor, anchored stops, breakeven off, trail off, size 1 — has three trades behind it. That is not a sample; it is an anecdote. Every drawdown and expectancy number above is computed on a distribution that is **95% pre-0B**, generated by a materially different exit regime. **[T]** The pre-0B mean (−2.741, n=62) is *better* than the full-sample mean precisely because the three post-0B losses are dragging it — which is a warning not to read the full-sample figure as descriptive of the system as it runs today.

**Add §5's execution finding on top, and the go/no-go gets worse, not better.** The 1E distribution is built from trades whose armed entries filled at the authorized price with zero slippage, three times out of three, on limits the market ran through by 80 to 240 points (§5.2), and whose stop exits printed at the stop price while the market traded up to 13 points past it (§5.4). **[I]** The live distribution of this system is not the SIM distribution shifted by a constant; it is the SIM distribution with the *good* fills selectively removed — the passive fills you get in a live book are disproportionately the ones you did not want. Until `slippage_ticks` is persisted (§5.3) you have no way to size that gap.

**The single sentence.** You do not have an edge; you have a sample too small to say whether you have one, generated by a fill model that flatters the entries, on a configuration with three trades of history, protected by a daily limit that cannot flatten you and cannot stop an arm. **[I]** Fix the four cheap things — persist slippage, stand the marketable guard down instead of cancelling, make the daily limit a leg of `EntryGate` and act on `RiskForceFlat`, put drawdown-vs-distribution on the screen — keep size at 1, and keep running SIM until the lower CI bound clears zero. That is not caution. At t = −0.691 it is the only conclusion the arithmetic supports.

---

## Surprises — recorded, not acted on

1. **`RiskLimits.Classify` has no callers at all.** Not one, outside tests. An entire risk-classification function, documented (`kernel/risk_limits.go:83-86`) and tested, that nothing calls.
2. **The bracket-modify branch is dead code** — two independent always-false conditions, verified by the prior wave with a reproduction (`2026-09-04-two-day-audit-data/verify-arm35-stop-slippage.md:24-35`), and `grep -c "bracket modify"` returns 0 across all 18 log files. The churn guard cannot fire. I did not re-verify against logs (none here) but the code path at `armed_executor.go:515-565` reads exactly as the wave describes.
3. **`max_contracts_per_order = 2` in config while the Stage-A clamp caps at 1.** The clamp governs, so behaviour is right — but the stored value and the effective value disagree and only the clamp's own doc comment records it (`2026-09-03-mc-drawdown.md:44-45`).
4. **`trade_excursions` has zero rows all-time** (`2026-09-04-two-day-audit-data/trades.csv`, `trade_excursions_row` column: "ABSENT (table has 0 rows all-time)" on all 11 rows). Every MAE/MFE figure in this review comes from `trader_positions.mae/mfe` or the auditor's 1m reconstruction, not from the excursion recorder — which is also why `median_mfe`, `stop_hit_share` and `target_hit_share` would render as em dashes even if §4.1's fix shipped today.
5. **Arm 29 was placed at a different price than its ledger row carries** — `entry_px 29044.00`, `placed_at_broker … @ 29035.25` (`arms.csv`). Consistent with the documented re-arm overwrite on the `plan_id+scenario+leg_index` unique index, but worth a look on its own.

## Scope note

Everything above is read-only. I changed no code, config, DB, knob, prompt, env or unit; cancelled, restarted and reset nothing; ran no git command. The only file I wrote in the tree is this report. All computation ran in a scratch directory on CSVs committed to this repository, and every command is shown.
