# Two-day audit — every loss, every refused opportunity, and why a fast tape produced one trade

**Window** 2026-09-02 00:00 CT → 2026-09-03 23:31 CT (with 08-26…09-01 as baseline)
**Dispatch** owner hoang, 2026-09-04 · READ-ONLY · SIM-only (Sim101), MNQ only
**Branch** `docs/two-day-audit-0904` · base `dfbfa660` (dev tip at accept) · worktree `~/nofx-2day04`
**Running rev at audit time** boot line `530009ff`; PID 594377 started 2026-09-03 23:12:54 CT
**Evidence classes** **[A]** directly verified · **[B]** inferred from strong evidence · **[C]** speculation

---

## 0. DANGEROUS RIGHT NOW — nothing

**[A] There is no open position and no resting arm.** Nothing needs the owner's hand tonight.

```sql
sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
 "SELECT * FROM trader_positions WHERE status='OPEN';"                        -- 0 rows

-- ⚠ SUPERSEDED — DO NOT COPY THE QUERY BELOW. It retyped 5 of the 7 terminal
-- states and invented a 6th ('done') that the code never sets. See class 99 in
-- docs/superpowers/AUDIT-CHECKLIST.md. Use the code's own predicate,
-- isTerminalArmState (trader/one_contract.go:285), or the exported SQL fragment
-- once it lands.
sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
 "SELECT * FROM armed_orders
   WHERE state NOT IN ('filled','cancelled','canceled','expired','done');"    -- 0 rows

-- ✅ THE CORRECT FORM — all seven states isTerminalArmState returns true for:
sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
 "SELECT * FROM armed_orders
   WHERE state NOT IN ('filled','cancelled','canceled','rejected',
                       'expired','superseded','shadowed');"                   -- 0 rows
```

**Both forms returned 0 that night, and that is the trap.** The retyped list was true when it
was written — the highest arm id in the table was 37, and the first row it would have
miscounted (id 92, `superseded`) was not created until 2026-09-04 10:44:01 CT. The defect
stayed latent for five days. On 2026-09-09 the same query counted **10** rows as resting arms
when every one of them was `superseded`, the true count was **0**, and that number went to the
owner as the reason to hold a cutover. Re-run against the same table at the same instant today,
the two forms return **10** and **0**. A hand-typed state list agrees with the code for exactly
as long as the table happens not to contain the states it omits — which is why no snapshot of
`select distinct state` can catch this. The wrong query is left standing beside the right one
deliberately. See class 99.

The last position (id 591) closed 2026-09-03 09:20:45 CT. The last arm (id 37) was cancelled
2026-09-03 12:15:01 CT. Nothing has been working at the broker since.

---

## 1. VERDICT FIRST

The two days were **not** a gate problem. In proportion, from the tables below:

| # | cause | share | the number that says so |
|---|---|---|---|
| **3** | **PLANNER / ENTRY SHAPE** | **~55%** | 09-03: **0 `open_long` proposed in 575 decisions** during a +483-pt rally; long scenarios were arm-enabled **4.3%** of the time vs **44.4%** for shorts |
| **5** | **NOT-A-RULE, NOT-THE-TAPE — host outage + post-boot blindness** | **~30%** | **the WSL2 host rebooted at 14:18** (`who -b`); silence 12:24:33→14:18:24 CT (**113 min 51 s**), then **~50 min blind** on FEED DOWN / `NT8 TCP link DOWN — NEW entries BLOCKED`. Together: the NY afternoon containing the day's high (29585) |
| **2** | NO SETUPS | ~10% | the day's only arm-enabled long (29481.05) **missed by 8.2 pts**; 09-02's four losses each stopped at their own authored stop |
| **1** | GATES TOO TIGHT | **~5%** | 61 refusal events over **44 distinct opportunities**, and the refused set **loses $860.64** on the actual tape. No leg's ledger says "review". A structural gate defect exists (`plan_mode=strict` closes the decision path) but it cost nothing in this window |
| **4** | CADENCE | **~0%** | class-47 cutoff/cooldown suppressed **nothing**: 09-02 ran WARN-first, 09-03 ran enforce and its single firing was fast-market-exempted |

**Can this system, as configured today, take a trade on a trending fast tape?**
**[A] Only by accident.** On 09-03 it correctly identified the trend — the machine regime read
`up/NORMAL` on all 17 planner reads, and from NY v3 (09:15 CT) the plan bias was `long` with
`day_type: trend` — and it still took exactly one trade: a **short**. The rule that prevents it is
not a gate. It is that **the planner grants resting arms almost exclusively to short scenarios**,
while long scenarios are left on the AI decision path, which proposed `open_long` zero times all
day. This is a **planner-shape answer**, not a gate answer.

---

## 2. C1 — THE TRADES

`pnl_corrected` only. **[A] Zero exclusions on both days**: 0 NULL-corrected, 0 `UNRESOLVABLE`,
0 `source='e7_farside_test'`, 0 pre-era (`DayPlanEraStart` = 2026-08-15 00:00 CT,
`store/attribution.go:131-158`).

| day (CT, by exit) | n | Σ pnl_corrected | excluded |
|---|---|---|---|
| 2026-09-01 (context) | 6 | **+212.00** | 0 / 0 / 0 |
| **2026-09-02** | **4** | **−381.50** | 0 / 0 / 0 |
| **2026-09-03** | **1** | **−140.00** | 0 / 0 / 0 |

### D1 — every closed position in the window

| id | sess | v | scen | side | entry (CT) | exit (CT) | entry→exit | pnl | authored SL | exit vs SL | path |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 587 | ASIA | 7 | S3 | LONG | 09-02 00:17:44 @29079.25 | 01:03:41 @29048.00 | −31.25 | **−62.50** | 29048.00 | **exact** | system |
| 588 | LONDON | 3 | S2 | LONG | 09-02 07:41:05 @29082.50 | 07:51:38 @29050.00 | −32.50 | **−65.00** | 29050.00 | **exact** | system |
| 589 | NY | 3 | S3 | LONG | 09-02 09:41:04 @29192.50 | 09:59:27 @29115.00 | −77.50 | **−155.00** | 29115.00 | **exact** | system |
| 590 | NY | 5 | S4 | LONG | 09-02 10:37:17 @29193.25 | 10:49:30 @29143.75 | −49.50 | **−99.00** | 29143.75 | **exact** | system |
| 591 | NY | 2 | S1 | SHORT | 09-03 09:05:14 @29285.00 | 09:20:45 @29355.00 | −70.00 | **−140.00** | 29351.63 | +3.37 slip | armed→reconcile |

**[A] All five were clean stop-outs at the position's own authored stop** — four to the tick, one
with 3.37 pts of slippage. There is **no premature-exit defect**: `close_reason='sync'` is the
broker-state sync recording a stop fill, not an early close. Multiplier confirmed empirically:
591 moved 70 pts for −140.00 → **$2/point** (MNQ).

```sql
-- the authored stop for each decision-path loss
SELECT json_extract(j.value,'$.action'), json_extract(j.value,'$.stop_loss'),
       json_extract(j.value,'$.confidence'), datetime(d.timestamp,'-5 hours')
FROM decision_records d, json_each(d.decision_json) j
WHERE json_valid(d.decision_json) AND json_extract(j.value,'$.action') LIKE 'open%';
-- 587 SL=29048 conf=62 · 588 SL=29050 conf=63 · 589 SL=29115 conf=65 · 590 SL=29143.75 conf=68
```

### The exits were right; the stops were where price turned

30 minutes after each exit, from 1m bars:

| id | side | exit px | best 30m **for** the trade | worst 30m **against** | reading |
|---|---|---|---|---|---|
| 587 | LONG | 29048.00 | +10.25 | −31.50 | exit correct |
| 588 | LONG | 29050.00 | +30.75 | −17.00 | marginal |
| 589 | LONG | 29115.00 | **+93.00** | −8.75 | **stopped at the low** |
| 590 | LONG | 29143.75 | **+54.25** | −8.00 | **stopped at the low** |
| 591 | SHORT | 29355.00 | **+82.50** | −9.75 | **stopped at the high** |

**[A] Three of five stopped within ~9 pts of the exact turning point** and then ran 54–93 pts in the
trade's favour. That is a stop-*placement* finding, not an exit-path finding: the stops were
honoured exactly as authored, and each sat where the market reversed from. 589 and 590 are
substantially the same idea 56 minutes apart (long 29192.50 then long 29193.25) — one thesis,
re-entered, stopped twice, for −254.00 of the day's −381.50.

---

## 2b. C2 — since the last fill, and C3 — the boots

### C2 — what happened after the one trade

The last fill is **arm 35 → position 591, filled 2026-09-03 09:05:14 CT**. Since that moment:

| | count |
|---|---|
| decision cycles evaluated | **314** |
| of those, an OPEN was proposed | **13** — every one `open_short` |
| arms authored | **2** (id 36, id 37) |
| arms actually placed at the broker | **1** (id 37; id 36 was cancelled marketable before placement) |
| arms filled | **0** |
| positions opened | **0** |

**[A] Not one long has been proposed in the 14+ hours since the last trade**, on a tape that rose
from 29285 to 29585 in that time.

### C3 — the boots, and the rule in force at each

**[A] 28 boots across the two days** — 17 on 09-02, 11 on 09-03 — from **17 distinct build
directories**. Every refusal in this report is attributed to the rule printed by the boot that
preceded it, not to a file default.

| # | boot (CT) | build tree | # | boot (CT) | build tree |
|---|---|---|---|---|---|
| 1 | 09-02 00:01:06 | clone-c39 | 15 | 09-02 22:37:35 | nofx |
| 2 | 09-02 00:10:20 | clonepnl ← `TRADING REFUSED` stale rev | 16 | 09-02 22:41:58 | nofx |
| 3 | 09-02 00:11:47 | clonepnl | 17 | 09-02 23:24:56 | nofx |
| 4 | 09-02 06:27:45 | c41clone | 18 | 09-03 10:28:29 | cleanclone |
| 5 | 09-02 06:57:49 | c33clone | 19 | 09-03 11:10:33 | cc2 ← **mid-NY-session** |
| 6 | 09-02 07:32:15 | rfclone | 20 | 09-03 14:18:24 | cc2 ← **host reboot** |
| 7 | 09-02 07:49:06 | clone0b2 | 21 | 09-03 14:59:03 | cc3 |
| 8 | 09-02 08:10:30 | rpclone | 22 | 09-03 15:02:18 | cc3 |
| 9 | 09-02 17:51:14 | bwclone | 23 | 09-03 18:08:56 | cleanbuild |
| 10 | 09-02 18:05:26 | nofx-deploy | 24 | 09-03 19:06:06 | cc6 |
| 11 | 09-02 18:27:17 | c46clone | 25 | 09-03 21:19:00 | nofx |
| 12 | 09-02 20:42:28 | nofx | 26 | 09-03 21:48:12 | nofx |
| 13 | 09-02 21:19:21 | nofx | 27 | 09-03 23:11:59 | nofx ← `TRADING REFUSED` stale rev |
| 14 | 09-02 21:32:51 | ncclone | 28 | 09-03 23:12:54 | cleanclone ← **the running PID 594377** |

**The rules changed inside the window.** The two that matter for attribution:

| behaviour | 09-02 | 09-03 (from boot 18) |
|---|---|---|
| class-47 wake cutoff/cooldown | `cutoff=25m cooldown=30m` — **WARN-first, refuses nothing** | `cutoff=25m(enforce) cooldown=30m(enforce, fast-market≥1.5×ATR exempt)` |
| min-SL multiplier | **1.0** until ~07:30, **1.5** after | **1.5** |
| 1B detector recorder | not present | call site landed **21:31 CT**, first ran **21:48:12** |
| trade-excursion writer | not present | first booted **10:28:29** — 67 min after the last trade closed |

**[A] The last boot verified itself:** `BOOT INTEGRITY OK — rev 530009ffd540 · built
2026-09-04T04:04:45Z · expected 530009ff · goldens PASS`. The worktree this audit reads
(`dfbfa660`) is **3 commits ahead** of the deployed `530009ff`; the diff is `deploy/RELEASE`,
`AUDIT-CHECKLIST.md` and docs only — no behavioural code.

**Three premise corrections on the running rev [A]:**

1. **PID 594377 *is* boot 7 running `530009ff`. There is no boot 8** — one `nofx-bin`,
   `ExecMainStartTimestamp = Thu 2026-09-03 23:12:54 CDT`. They were never two facts to reconcile.
2. **"~23:05 CT" is the build time, not the boot.** `530009ff` was committed/built at 23:04:45 CT
   (`vcs.time 2026-09-04T04:04:45Z`); the boot is **23:12:55 CT**, eight minutes later.
3. **The 23:11:59 boot is the failed half of that same cutover** (class 74 null cutover), not a
   separate deploy — the old binary `89673ccc` relaunched and self-refused because `deploy/RELEASE`
   already named the new rev. The two `TRADING REFUSED` lines in this window are one aborted
   cutover each, not two independent stale deploys.

**When each gate entered force** — this is what lets a refusal be attributed to the rule that
actually governed it:

| gate / behaviour | in force from |
|---|---|
| wake cutoff 25m + cooldown 30m, **WARN-only** | 09-02 21:19:21 CT |
| the same, **ENFORCING** | **09-03 10:28:29 CT** |
| cooldown fast-market exemption | 09-03 11:10:33 CT |
| **`plan_mode=strict` (EntryGate leg 0)** + invalidation (leg 3) | **09-03 11:10:33 CT** |
| one-open-position (leg 7) | **09-03 14:59:03 CT** |
| trade-excursion writer | 09-03 10:28:29 CT (67 min after the last trade closed) |
| 1B detector recorder | 09-03 21:48:12 CT |

**[A] None of the legs that entered force on 09-03 existed during any of the five losing trades** —
the last of which closed 09-03 09:20:45 CT, before the 10:28:29 boot. Every gate the dispatch
names as "new" post-dates every loss in the window.

---

## 3. C4 — THE TAPE

Session windows, read from the running process's own ledger boot line (not assumed):

```
🧾 ledger boot: sessions[ASIA 17:00→02:00 CT (last-entry 01:45, flat 02:00)
                       | LONDON 02:00→08:30 CT (last-entry 08:15, flat 08:30)
                       | NY    08:30→14:45 CT (last-entry 14:30, flat 14:45)]
```

| day | low | high | range | shape |
|---|---|---|---|---|
| 2026-09-02 | 28927.25 | 29255.00 | **327.75** | **chop / balance** — 40–90 pt swings, no sustained leg; open 29059 → close 29207.50 |
| 2026-09-03 | 29075.00 | **29601.00** | **526.00** | **strong uptrend** — 29101.75 (06:00 low) → 29585.00 (13:00 RTH high) = **+483.25 in 7 hours**; the 29601.00 print is the overnight extension at 23:5x CT |

### Per session (the dispatch's C4 table)

Session boundaries read from `kernel/session_registry.go:83-118`, the declared single source of
truth, and cross-checked against the persisted `system_config.session_registry`:
**ASIA 17:00→02:00 (read 16:30) · LONDON 02:00→08:30 (read 01:30) · NY 08:30→14:45 (read 08:00)**,
all CT. An ASIA session is labelled by the date it *starts*. (The registry's `enabled:false` on
ASIA/LONDON is only the default the resolver consults — `trader/auto_trader_session.go:105-107`
says it "must never VETO a session the resolver already said runs", and positions 587 (ASIA) and
588 (LONDON) prove it did not.)

ATR period verified as **14, Wilder, on 5m bars** — `kernel/structure.go:151` (hardcoded `n := 14`),
`kernel/plan_confirm.go:147`, `trader/auto_trader_planner.go:2808-2813`.

| day | session | open | high | low | close | range | net | ATR5m @ read | largest 15m | at (CT) |
|---|---|---|---|---|---|---|---|---|---|---|
| 09-01 | ASIA | 29137.25 | 29171.25 | 29016.50 | 29076.75 | 154.75 | −60.50 | 21.75 *[B]* | +49.25 | 01:30 |
| 09-02 | LONDON | 29076.75 | 29149.25 | 28927.25 | 29098.25 | 222.00 | +21.50 | 19.86 *[B]* | −58.25 | 03:45 |
| 09-02 | **NY** | 29098.25 | 29211.75 | 29017.25 | 29175.00 | **194.50** | +76.75 | **26.02** | **−81.50** | 09:45 |
| 09-02 | ASIA | 29175.75 | 29255.00 | 29075.00 | 29173.50 | 180.00 | −2.25 | 18.73 *[B]* | −59.25 | 00:00 |
| 09-03 | LONDON | 29173.25 | 29293.00 | 29101.75 | 29248.75 | 191.25 | +75.50 | **22.10** | +55.00 | 07:45 |
| **09-03** | **NY** | 29249.50 | **29585.00** | 29199.25 | 29524.00 | **385.75** | **+274.50** | **29.02** | **+101.50** | **10:00** |
| 09-03 | ASIA (open) | 29500.00 | 29601.00 | 29481.50 | 29599.75 | 119.50 | +99.75 | **12.74** | +18.00 | 19:15 |

**[A]** Bolded ATR values are the system's own recorded numbers (`planner_read_facts`, or the
09-02 arm-stop log line `1.5×ATR5m 26.02` at 08:03:06). **[B]** The three italic values are
reconstructions — `planner_read_facts` holds 17 rows, **all dated 09-03**, so no recorded ATR
exists for any 09-02 read. Measured against all 17 recorded rows, closed-bar reconstruction runs
**+0.01 to +2.89 pts** high (the live cache includes a partial 5m bar the DB never persists), so
treat them as ±1 pt in quiet tape and ±3 pts in fast tape. They are not exact and cannot be made so.

**The 09-03 NY session is the whole story in one row: +274.50 net on a 385.75 range, its largest
15-minute move +101.50 at 10:00 CT — and the system was short into it.**

**Snapshot caveat [A]:** the `bars` table is live and was still appending while this audit ran.
The RTH figures below are stable; the day's *extreme* moved from 29585.00 to **29601.00** between
the first and last measurement (1374 1m bars at 23:53 CT). Every figure in this report is stamped
with the reading that produced it.

09-03 hourly (CT), the two impulse hours in bold:

| hour | O | H | L | C | Δ |
|---|---|---|---|---|---|
| 06:00 | 29129.00 | 29187.25 | **29101.75** | 29105.25 | −23.75 |
| **07:00** | 29105.25 | 29260.00 | 29104.00 | 29244.50 | **+139.25** |
| 08:00 | 29244.75 | 29375.25 | 29194.50 | 29332.25 | +87.50 |
| 09:00 | 29332.25 | 29367.75 | 29241.50 | 29363.25 | +31.00 |
| **10:00** | 29363.50 | 29543.75 | 29362.00 | 29538.25 | **+174.75** |
| 13:00 | 29543.50 | **29585.00** | 29543.00 | 29568.50 | +25.00 |

The owner's "price moving crazy" is 09-03, and it is real: a 510-point day against a 327-point
chop day before it.

---

## 4. THE CENTRAL FINDING — the planner arms shorts and not longs

**[A] The planner correctly read the trend.** Machine regime was `up/NORMAL` on **all 17**
`planner_read_facts` rows for 09-03. From NY v3 (09:15 CT) the plan's own bias was `long`,
`day_type: trend`, and stayed so through v7 (11:27 CT). v4 authored **three long scenarios and
zero shorts**.

| 09-03 NY version | created (CT) | bias | conviction | day_type | longs | shorts |
|---|---|---|---|---|---|---|
| v1 | 08:05:19 | short | medium | balance | 1 | 2 |
| v2 | 08:45:05 | short | medium | balance | 1 | 2 |
| **v3** | **09:15:31** | **long** | medium | **trend** | 2 | 2 |
| **v4** | **09:44:33** | **long** | medium | **trend** | **3** | **0** |
| v5 | 10:14:45 | long | medium | trend | 1 | 1 |
| v6 | 10:49:57 | long | medium | trend | 2 | 1 |
| v7 | 11:27:53 | long | medium | trend | 2 | 1 |

**And it took no long.** The mechanism is arm-enablement:

| day | side | scenarios authored | `arm.enabled` | **rate** |
|---|---|---|---|---|
| 2026-09-02 | long | 64 | 9 | **14.1%** |
| 2026-09-02 | short | 43 | 27 | **62.8%** |
| **2026-09-03** | **long** | **23** | **1** | **4.3%** |
| **2026-09-03** | **short** | **18** | **8** | **44.4%** |

```sql
SELECT p.trade_date, json_extract(s.value,'$.direction') dir,
  SUM(CASE WHEN json_extract(s.value,'$.arm.enabled')=1 THEN 1 ELSE 0 END) armed_enabled,
  COUNT(*) total
FROM plans p, json_each(json_extract(p.doc,'$.scenarios')) s
WHERE p.trade_date IN ('2026-09-02','2026-09-03') GROUP BY p.trade_date, dir;
```

A scenario without `arm.enabled` has no resting order; it can only become a trade if the AI
decision loop proposes an open. **[A] On 09-03 the decision loop proposed `open_long` zero times
in 575 decisions:**

| day | wait | open_long | open_short | decisions | open-intent |
|---|---|---|---|---|---|
| 2026-08-27 | 585 | 0 | 4 | 589 | 0.68% |
| 2026-08-28 | 403 | 0 | 0 | 405 | 0.00% |
| 2026-08-30 | 155 | 0 | 0 | 155 | 0.00% |
| 2026-08-31 | 463 | 0 | 0 | 463 | 0.00% |
| 2026-09-01 | 640 | 1 | 2 | 643 | 0.47% |
| **2026-09-02** | 623 | 5 | 5 | 633 | **1.58%** |
| **2026-09-03** | 561 | **0** | **14** | 575 | **2.43%** |

```sql
SELECT date(d.timestamp,'-5 hours') day, json_extract(j.value,'$.action') action, COUNT(*)
FROM decision_records d, json_each(d.decision_json) j
WHERE json_valid(d.decision_json) AND d.decision_json <> '' GROUP BY day, action;
```

**Two corrections to the dispatch's premises, both material:**

1. **The drought is not new and 09-03 was not the quiet day.** At 2.43%, 09-03 had the **highest**
   open-intent rate of the eight days measured. Four of the seven baseline days produced **zero**
   open intents. "Wait" at 91–100% is this system's normal state, not a new tightening. **[A]**
2. **The system placed one trade "since last night" is understated.** The last fill is position 591,
   entered **09-03 09:05:14 CT** — one trade in the whole of 09-03, from the NY morning, and
   nothing in the ~14 hours since. **[A]**

### The one arm-enabled long of 09-03 missed by 8.2 points

NY v6 S2, `long` @ **29481.05** (`arm.enabled: 1`), live 10:49:57 → 11:27:53 CT when v7 superseded it.

```sql
SELECT round(MIN(l),2) FROM bars WHERE symbol='MNQ' AND tf='1m'
 AND open_time_ms BETWEEN strftime('%s','2026-09-03 15:49:57')*1000
                      AND strftime('%s','2026-09-03 16:27:53')*1000;   -- 29489.25
```

**[A] Minimum low during its life: 29489.25 — 8.2 points above the entry.** It never became an
`armed_orders` row; the log only ever shows the display-only estimate
(`🎯 scenario S2 → ≈armed @ 29481.05 … never execution-wired`). Price did reach 29476.00 at
11:28 CT — one minute *after* v6 was superseded, and v7's two long scenarios had no arms.
Cause: **`never_reached`**, by 8.2 points and one minute.

---

## 5. THE TRADING AFTERNOON WAS LOST TO 113 MINUTES OF SILENCE, THEN TO A BLIND BOOT

**Two separate events, and the first one has no determinable cause.** This audit passed through
two wrong readings before arriving here — it first blamed deploy churn, then blamed the host
reboot for the whole window. Neither is right.

```
last log_events row before the silence   →  id 25754,  2026-09-03 12:24:33 CT
kernel boot   (uptime -s)                →  2026-09-03 14:02:25
utmp boot     (who -b)                   →  2026-09-03 14:18       ← utmp is written late on WSL2
bot log resumes                          →  2026-09-03 14:18:24
```

| segment | duration | what is known |
|---|---|---|
| 12:24:33 → **14:02:25** | **98 min** | **[A] CAUSE CANNOT BE DISTINGUISHED.** The process stopped emitting. There is **no panic, no fatal, no shutdown banner, no signal line** anywhere in 12:20–12:25, and journald retains **nothing** before 2026-09-03 18:21:48 CT, so the window is unrecoverable. The 548 bytes of NUL at log offset 779709 are an unclean-shutdown signature — consistent with the process dying or the host freezing, and it does not distinguish between them. |
| 14:02:25 → 14:18:24 | 16 min | host up (kernel boot), bot not yet running |
| 14:18:24 → ~14:58 | ~40 min | bot up and **blind** — see below |

**Corrected silence window: 2026-09-03 12:24:33 → 14:18:24 CT = 113 min 51 s**, of which only the
last 16 minutes are explained by the host boot. **The host reboot at 14:02:25 does not explain the
first 98 minutes**, and nothing in the record does.

What the tape did while nothing was running:

| | value |
|---|---|
| price at down / at up | 29484.50 → 29536.50 |
| low / high in the window | 29479.75 / **29585.00** ← the day's high |
| range traversed | **105.25 pts** |
| NY entry time consumed | 2h of the 08:30–14:30 last-entry window |

**[A] MNQ 1m bars are COMPLETE across the outage** (backfilled on reconnect), so an audit that
looked only at bar coverage would see no outage at all. The tape reconstruction in this report is
therefore sound; only the *live* decision path was absent.

### The genuine defect is what happened AFTER the boot

The process came up at 14:18:24 into a 27-minute remainder of the NY session and **could not
trade for any of it**:

```
09-03 14:18:27→14:58:28 CT:  21 decision cycles ran — ALL 21 SKIPPED, every one for
                             "no balance frame yet" (equity 0, no AddOn frame)
                             31 × 🚨 FEED DOWN (no NT8 bar, age reaching 40m0s, CME OPEN)
                              3 × 🚨 NT8 TCP link DOWN — NEW entries BLOCKED (dead-man watchdog)
```

The next decision cycle after 12:22:44 is **15:08:51** — a 166-minute decision drought, of which
**52 minutes had a fully alive, loudly-logging process that was simply blind**. The NY flat is
14:45. **[A] The whole NY afternoon — 12:24:33 to 14:45, 2h20m containing the day's high — ran
with either no host or no market data, and nothing alerted on the second half.**

**[B] The restart cadence is a separate finding, and it is not the cause of this gap either.** `nofx_2026-09-03.log`
alone carries **11 boot banners from at least 6 different build trees** (`cleanclone`, `cc2`,
`cc3`, `cleanbuild`, `cc6`, `nofx`) — audit and fix lanes cutting over repeatedly on a live
trading day, one of them at 11:10:33 CT landing *between* NY v6 (10:49:57) and v7 (11:27:53), so
the running code changed mid-session. Twice in the window the rev-guard caught a stale binary and
refused outright:

```
09-02 00:10:21 [ERRO] 🔐 TRADING REFUSED — binary is revision "23f56f49a536"
                       but the intended release is "aeb11179df5a" — a stale binary is running
09-03 23:11:59 [ERRO] 🔐 TRADING REFUSED — binary is revision "89673ccc5984"
                       but the intended release is "530009ff" — a stale binary is running
```

Both were boot-time and cleared on the next boot, so neither blocked trading for long — but each
is a cutover that shipped the wrong binary onto a live trading host.

---

## 6. D3 + D4 — the arms, and why arm 37 died

All `armed_orders` rows in the window. **Timestamp warning (defect #4 below): `created_at` is
stored with a CT offset and `updated_at` with `+00:00` UTC.** Both are normalised to CT here.

| id | day | sess | scen | side | entry | state | alive | reason | cause class |
|---|---|---|---|---|---|---|---|---|---|
| 29 | 09-01 | ASIA | S1 | long | 29044.00 | cancelled | 70.8m | no order_update within stale window (reconnect) | **defect (restart)** |
| 30 | 09-01 | ASIA | S1 | long | 29035.25 | cancelled | 18.1m | " | **defect (restart)** |
| 31 | 09-02 | ASIA | S3 | long | 29068.05 | cancelled | 15.3m | " | **defect (restart)** |
| 32 | 09-02 | NY | S3 | long | 29070.00 | cancelled | 254.5m | session ended (EOD flat) | never_reached |
| 33 | 09-02 | NY | S3 | short | 29166.80 | cancelled | 0.0m | level accepted through — marketable, never placed | marketable_guard |
| 34 | 09-02 | ASIA | S1 | short | 29199.50 | cancelled | 0.0m | " | marketable_guard |
| **35** | **09-03** | **NY** | **S1** | **short** | **29285.00** | **filled** | 85.6m | → position 591 | **the one trade** |
| 36 | 09-03 | NY | S2 | short | 29351.05 | cancelled | 0.0m | " (price 29358.75 already above 29351.05) | marketable_guard |
| 37 | 09-03 | NY | S3 | short | 29543.75 | cancelled | 16.5m | cancelled in NT8 | **parked(death)** |

**The arms table under-counts the arms [A].** `armed_orders` is **not append-only**: `UpsertArm`
(`store/armed_orders.go:155,175-196`) rewrites the `(plan_id, scenario, leg_index)` slot in place.
The 15 rows below therefore under-represent **at least 11 distinct broker placements**; three
signal ids (`0c77307d`, `fd71b48f`, `18a7cc55`) survive nowhere in the table and **two of them
filled into real positions** (582 +129.50, 585). Worse, the rewrite happens even while `state =
'working'` — under a live broker order, with no cancel or replace sent: **arm 29's ledger says
entry 29044.00, but the order actually resting at NT8 was a buy limit at 29035.25**, repriced at
23:58:16 while live. Treat the entry prices below as the ledger's last word, not the broker's.

**Arms 29/30/31** were cancelled by the stale-order-update reconciler at 00:07–00:31 CT on 09-02 —
inside the window of three boots in eleven minutes (00:01:06, 00:10:20, 00:11:47). **[B]** The
restart resets the NT8 order-update stream; the reaper then sees no update inside the 15-minute
stale window and cancels. **The mechanism is genuinely broken:** `onArmedOrderUpdate`
(`armed_executor.go:1139-1156`) switches only on `filled/partfilled/rejected/cancelled` — there is
no default branch and no `Touch` for `submitted/accepted/working`, so a **perfectly healthy**
resting limit is reaped ~15 minutes after placement simply because non-terminal frames write
nothing to the ledger. Three long arms in ASIA died to deploy churn, not to the market.

**And they would have filled [A].** Price traded through all three entries within 45 minutes of
the cancel (arm 31 at 00:54, arms 29/30 at 01:17 CT). **But the reaper was lucky** — carried to
their own stops and targets on the real tape, all three would have been **stopped out**:

| arm | side | entry | stop | target | outcome | pts |
|---|---|---|---|---|---|---|
| 29 | long | 29035.25 | 29025.00 | 29099.85 | STOP 01:18 | −10.25 |
| 30 | long | 29035.25 | 29018.00 | 29071.41 | STOP 01:33 | −17.25 |
| 31 | long | 29068.05 | 29044.00 | 29131.66 | STOP 01:04 | −24.05 |
| | | | | | **net** | **−51.55 pts = −$103** |

So the reaper cost three real fills and **saved $103** doing it. The defect is the mechanism, not
this outcome.

### Arm 37 — the case that looks like a broker defect and is not

Arm 37 was a **real working order**, not a display estimate — `nt.PlaceLimitEntry` returned a
signal id (`armed_executor.go:927-935`):

```
09-03 11:58:33 ⚔️ armed NY S3 leg 1 short limit 29543.75 SL 29592.50 TP 29397.11
09-03 11:58:33 📌 armed S3 → WORKING limit 29543.75 signal=43512e1d-… (band ±100t)
```

Price later traded at or above 29543.75 on **83 one-minute bars**, up to 29585.00. It never filled.
That looks like a broker defect. **It is not** — by 12:51 the order was already gone:

```
09-03 12:15:00 😴 plan 2026-09-03 NY v7 DORMANT — death-condition: 2x5m close below 29502.25
09-03 12:15:01 📡 armed order_update summary: frames=4 accepted=1 working=1 cancelpending=1 submitted=1
09-03 12:15:01 ✕ armed S3 cancelled in NT8
09-03 12:15:01 🔒 armed cancel: no active plan — 1 order(s) disarmed
```

**[A] The plan's death condition fired at 12:15:00, the disarm-on-no-active-plan rule cancelled the
arm, and NT8 acked it.** "cancelled in NT8" is our own cancel being acknowledged, not an external
one — the string is written by `onArmedOrderUpdate` when the broker reports the cancel, so the
ledger **launders our own action into something that reads like a broker event**. That is a defect
in its own right: an operator reading `state_reason` alone would investigate NT8.

**A correction, and an honest CANNOT-DISTINGUISH.** An earlier pass of this report said the tape
came no closer than **9.5 pts** during the arm's life. That figure silently excluded the arm's own
creation minute. The 11:58 bar is `o 29539.25 · h 29548.00 · l 29530.50 · c 29532.00` — **its high
is 4.25 pts THROUGH the 29543.75 short entry**, corroborated at 3m/5m/15m (all `h=29548.00`).

The arm was created at **11:58:33.110 CT, inside that bar**. **[A] The finest granularity stored
for MNQ is 1m** (verified across every timeframe present), so whether 29548.00 printed in
11:58:00–11:58:33 (before the arm existed) or in 11:58:33–11:58:59 (while it was live)
**cannot be distinguished from the store**. The post-creation segment high is bounded only to
[29532.00, 29548.00].

Nor does the `frames=4 accepted=1 working=1 cancelpending=1 submitted=1` line date the transition:
it is a global, cumulative, untimestamped counter drained on print
(`armed_executor.go:1078-1105`), so it establishes only that NT8 reported *Working* at some instant
in [11:58:33, 12:00:33] — it cannot rule out that the order was not yet live during the 11:58 spike.

What is certain: **after** the cancel, the first 1m bar whose high reaches 29543.75 opens
**12:51:00 CT — 36 minutes after the arm was gone.**

**Counterfactual, had the arm survived** (filled 12:51 @29543.75, stop 29592.50, target 29397.11):

- stop **never hit** — session max was 29585.00, **7.5 pts short of the stop**
- target **never hit** — session min after fill was 29481.50
- outcome: still open at the NY flat, closed ≈29505 → **≈ +38.75 pts ≈ +$77.50**

One trade's worth of profit, forgone to a death condition that fired 36 minutes early. Note the
death rule and the arm disagreed by construction: v7's bias was **long**, its death was a *close
below* 29502.25, yet the arm it carried was a **short** at 29543.75.

### After the death, nothing replaced it

**[A] NY v7 (11:27:53 CT) is the last NY plan version of the day.** It went dormant at 12:15:00.
The next plan of any session is ASIA v1 at **16:37:36 CT**. Between them the process was down
113 min 51 s (host reboot) and the NY session ended (flat 14:45). No NY plan was on the desk for the final
2h30m of the session, which contained the day's high.

---

## 7. D2 — every gate refusal

### A correction this audit owes: the entry gate DID refuse — silently

An earlier pass of this report concluded the entry gate refused nothing, on the strength of
`grep -c "entry_gate:"` returning **0/0** over both logs and no `entry_gate:*` counter existing in
`system_config`. **That conclusion was wrong, and the reason it was wrong is itself a defect [A].**

There are two refusal recorders and only one of them is visible:

| path | function | logs? | counter? |
|---|---|---|---|
| **arm** | `recordEntryGateRefusal` (`trader/entry_gate.go:461-474`) | ✅ `🚦 entry-gate REFUSED arm …` | ✅ `arm_refusals_0b:…:entry_gate:<class>` |
| **decision** | `entryGateDecisionTelemetry` (`trader/entry_gate.go:477-486`) | ❌ **nothing** | ❌ **nothing** — only `actionRecord.Error` |

The decision-path refusal is a **WARN-less code path**, so it is absent from the file log *and*
from `log_events` (which carries only WARN+). It exists in exactly one place:

```sql
SELECT COUNT(*) FROM decision_records
WHERE date(timestamp,'-5 hours') BETWEEN '2026-09-02' AND '2026-09-03'
  AND execution_log LIKE '%entry_gate%';                                  -- 19
```

**[A] 19 decision-path entry-gate refusals in the window:**

| leg | n | window (CT) |
|---|---|---|
| **strict (leg 0)** | **13** | 09-03 20:35:06 → 21:12:40 |
| min_sl (leg 6) | 3 | 09-02 18:48:49 → 18:52:33 |
| rr_at_fill (leg 5) | 3 | 09-02 19:00:23 → 09-03 02:12:56 |

The **arm**-path entry gate refused **zero** — that half of the original finding stands, and it is
counter-proven: the only `arm_refusals_0b` keys in the store are four `rr`-class rows totalling 7.

### The full refusal inventory

**[A] 61 refusal EVENTS across 44 distinct opportunities**, 09-02 00:00 → 09-03 23:34 CT:

| gate | seam | events | code |
|---|---|---|---|
| min-SL | `validateDecision` | **34** | `kernel/engine_position.go:229` → `kernel/min_sl.go:56` |
| **strict** | EntryGate leg 0 | **13** | `trader/entry_gate.go:162-172` |
| R:R at arm | `armGateVerdictFor` | 7 | `trader/armed_executor.go:415` |
| min-SL | EntryGate leg 6 | 3 | `trader/entry_gate.go:268` |
| rr_at_fill | EntryGate leg 5 | 3 | `trader/entry_gate.go:257` |
| other | — | 1 | — |
| marketable guard | `armed_executor.go:918-924` | 5 (ids 25,27,33,34,36) | wrong-side guard, no knob |
| stale-arm reaper | `armed_executor.go:1019-1034` | 3 (ids 29,30,31) | `armedWorkingStaleMin` = **15m** (resolved, printed) |
| plan-death disarm | `cancelArmedOrdersSync` | 1 (id 37) | — |

**[A] Fourteen wired gate legs never fired once**: EntryGate legs 1, 2, 3, 4, 7 (direction,
class-48 mismatch, invalidation, shadow, one_open_position); the arm-legacy `ArmSpecValid`,
not-armable, plan-bias, quality, min-SL and HTF-veto legs; `oneLiveArmGuard`;
`split_leg_capacity`; `condition_shadowed`; and on the decision side `feed_down` and the dead-man
legs. **The named "new legs" the dispatch asked about — one-open-position, invalidation-wired,
shadow map, direction — are among them. They refused nothing.**

### A29 — a gate with zero production callers

**[A] `armGateVerdict` (`trader/armed_executor.go:1268`) has EIGHT call sites and every one is a
test** (`armed_executor_test.go:77,82,87,94,100,180,185,189`). Production calls the sibling
`armGateVerdictFor` at `:415`. Per A29 a gate with 0 production callers is not a gate — this one is
dead code that a reader would reasonably mistake for the live path.

### Knobs, resolved (A11)

**[A] Every knob below was established from the running process's own boot lines, the saved
strategy row, or the refusal message that printed the value it applied — never from a file default
presented as live.** `/api/config/resolved` and `/api/risk/gate-blocks` both return
`{"error":"Missing Authorization header"}` from this session, so neither was used. Full table with
sources in `docs/…-two-day-audit-data/knobs.csv`.

| knob | live value | source |
|---|---|---|
| **`day_plan.plan_mode`** | **`strict`** — a *saved* strategy value (shipped default is `advisory`) | strategy `MNQ` `a5b7662e`; `store/resolve_source.go:47` |
| `min_risk_reward_ratio` | **2.0** — *saved*, both seams | strategy `MNQ`; store default `SafeDefaultMinRiskReward` = **3.0** |
| EntryGate leg-5 fallback floor | 3.0 (only when MinRR ≤ 0) | hardcoded, `entry_gate.go:~253` |
| `MIN_SL_ATR_MULT` | **1.0** until the 09-02 07:49:06 boot, **1.5** after | env UNSET → `kernel/min_sl.go:34`; boot line `atr_mult=1.5` |
| min-SL level clearance | 2 ticks | `kernel/min_sl.go:40`; boot line |
| exit stop composition (0B) | `max(anchor+clr, 1.5×ATR5m)`, `anchor_max=3.0×ATR5m` | boot line |
| **breakeven** | **OFF** — *despite the strategy carrying `breakeven_enabled: true`, `trigger=40pts`* | boot line `BE=off` |
| **trailing stop** | **OFF** — *despite `trailing_enabled: true`* | boot line `trail=off` |
| position size | 1 contract (fixed by 0B) | boot line `size=1` |
| no-chase | `max_dist=1.00×ATR`, `max_run=1.5×ATR5m`, **`mode=warn`** | env UNSET → `trader/no_chase.go:77,146` |
| shadow map | shadow = `[breakout_retest, fvg_entry]`; 7 conditions live | `kernel/condition_status.go:75`; env UNSET |
| wake cutoff / cooldown | 25m / 30m — **WARN on 09-02, ENFORCE from 09-03** | env UNSET → `class47_wake_cadence.go:52,59` |
| `armedWorkingStaleMin` | 15 min | printed resolved in `no order_update for 15m` |
| `min_confidence` | 60 (saved; coincides with the default) | strategy `MNQ`; `store/strategy.go:83` |
| one_open_position | **ON, hardcoded** — no knob | `entry_gate.go:272`, `armed_executor.go:623` |
| invalidation (leg 3) | ON, **arm path only**; unresolved verdict = leg PASSES | boot line `invalidation-wired=on` |
| `htf_veto` | ON, `mode=cross` (needs 1h **and** 4h to agree) | `HTF_VETO_MODE=cross` in `.env` |
| no-trade band | first 5m + lunch 12:00–13:30 CT, **code constants, not owner-configurable** | `kernel/no_trade_band.go:37,42` |
| `max_daily_trades` | **3 — NOT ENFORCED** (guardrails master off) | strategy value; master switch |

**[A] Two owner-set behaviours are silently overridden:** the strategy carries
`breakeven_enabled: true` (trigger 40 pts) and `trailing_enabled: true`, and the 0B wave suspends
both. Worse, **two boot lines disagree with each other** — `🛑 exits:` prints `BE=off · trail=off`
while the `🧾 ledger boot` line in the same boot still narrates
`trailing=2.0×ATR14 arm=after_breakeven (source: studio)`. An operator reading the ledger line
would believe trailing is live.

### Three gate defects

**[A] `plan_mode=strict` + EntryGate leg 0 refuses EVERY decision-path market entry, regardless of
citation.** Strict executes plan scenarios on the ARM path only; a market entry is by construction
a "market-path" entry, so leg 0 refuses it before any other test. All 13 strict refusals cite a
real scenario (S1) and are refused anyway. Since 09-03 10:43 CT (commit `c8c90dcc`) the decision
path is effectively closed to market entries — **and it says so nowhere.**

**[A] On 09-02 evening the EntryGate min-SL leg was fed the DAILY ATR.** It rendered
`1.5×ATR5m = 450.56` from a daily ATR of 300.37 while the arm seam in the same minutes used
ATR5m 12.78–14.12 — a **32×** threshold. Fixed by `609067ec` on 09-03 08:23 CT. The three
09-02 18:48–18:52 min_sl(leg6) refusals are that bug.

**[A] The no-chase leg is structurally incapable of firing on the arm path.** 40 evaluations,
every one with `dist = 0.00×ATR` and `run = NULL`, 0 `would_refuse`. `citedLevelFor` returns
`sc.Confirm.RefPrice`, which equals the arm entry in every plan in the window, so the distance is
always zero by construction. Its 27 log lines are noise.

### The reporting defect that inverts a correct refusal

`kernel/min_sl.go:62` prints the raw ATR where the threshold belongs:

```go
if dist < mult*atr {
    return true, fmt.Sprintf("sl_too_tight: %.1f < %.1f×ATR (%.1f) — widen or skip", dist, mult, atr)
}
```

`sl_too_tight: 40.2 < 1.5×ATR (27.3)` reads as "40.2 < 27.3", which is false; the threshold is
1.5 × 27.3 = 40.95 and 40.2 < 40.95 is correct. **[A] All 34 are arithmetically correct.** This
auditor initially recorded it as a gate defect and had to withdraw that.

---

## 8. SECTION F — THE COUNTERFACTUAL LEDGER

The direct test of "gates too tight". $2/pt, verified from position 591 (70 pts → −140.00).

### The whole refused set — 44 distinct opportunities

| outcome | n |
|---|---|
| TARGET first | **7** |
| STOP first | **31** |
| flat at horizon | 5 |
| never filled | 1 |
| **net, session-flat horizon** | **−$860.64** |
| net, CME-day horizon | −$1,036.52 |

**[A] The refused set loses money.** Taking every refused trade at its authored stop and target on
the actual tape produces a loss of about **$861**. **No leg's ledger says "review".** The gates
were not what cost these two days; on balance they saved roughly a day's worth of losses.

### Per leg

**min-SL — n=34 events, 11 distinct clusters.** A min-SL-rejected decision is not persisted with
its prices (the retry overwrites the record with the final `wait` — D24), so the exact stop/target
counterfactual is unrecoverable. The honest substitute — each refusal's own too-tight stop against
realised excursion:

| outcome | n | points |
|---|---|---|
| would have hit its **own** too-tight stop within 30 min | **18** | **−431.2 pts = −$862** |
| still alive at 30 min | 16 | ΣMFE30 = +715.5 pts = +$1,431 |
| **upper bound** (every survivor exits at its exact 30-min high — unattainable) | 34 | +284.3 pts = +$569 |

Per side: short n=25 (12 stopped, −248.9 pts) · long n=9 (6 stopped, −182.3 pts).
**Verdict: min_sl is not a leg to review.** Positive only under perfect exits; over half the
refused trades were stopped on their own stop inside half an hour.

**R:R-at-arm — n=7**, full stop/target counterfactual (prices recovered from the plan doc in force):

| time (CT) | sess | side | entry | stop | target | R:R | outcome | pts |
|---|---|---|---|---|---|---|---|---|
| 09-02 09:55:01 | NY | long | 29135.65 | 29099.00 | 29209.25 | 0.95 | **TARGET** 15:06 | **+73.60** |
| 09-02 10:25:01 | NY | long | 29058.75 | 29020.00 | 29143.97 | 1.74 | never_reached | 0.00 |
| 09-02 13:06:29 | NY | long | 29163.99 | 29138.75 | 29246.25 | 1.32 | STOP 13:12 | −25.24 |
| 09-02 18:47:17 | ASIA | short | 29209.25 | 29223.00 | 29181.64 | 1.30 | **TARGET** 19:07 | **+27.61** |
| 09-02 20:07:16 | ASIA | short | 29175.66 | 29194.30 | 29136.25 | 1.75 | STOP 20:16 | −18.64 |
| 09-03 08:18:54 | LONDON | short | 29219.97 | 29255.00 | 29144.50 | 1.68 | STOP 08:30 | −35.03 |
| 09-03 08:30:54 | NY | short | 29259.67 | 29289.00 | 29178.73 | 1.68 | STOP 08:38 | −29.33 |
| | | | | | | | **net** | **−7.03 pts = −$14** |

**n=7 is below the n=10 floor, so no verdict is stated** — the number is reported plainly.
**[A] All seven were authored at R:R ≥ 2.0 and pushed below the 2.0 floor by `composeArmStop`
widening the stop.** The plan cleared the gate; the stop composer then broke it.

**stale-arm reaper — n=3.** All three would have filled after the cancel and all three would have
been stopped: **−51.55 pts = −$103**. The reaper's ledger is *positive*; its mechanism is still
wrong (D10).

**strict (leg 0) — n=13.** All 09-03 20:35–21:12 CT, all ASIA, all after the trading day. They cost
nothing on 09-02/09-03, but they demonstrate a decision path that is structurally closed.

**The one winning refusal cluster.** 09-02 18:47:17–19:00:23 ASIA: 13 refusal events / **7 distinct
opportunities, all 7 would have hit target** — **+$644.22** treated independently, **+$55.22** as
one sequential position. This is the only cluster where refusing cost money, and the sequential
figure is the honest one: they are the same short re-proposed, not seven trades.

---

## 9. C5 + D5 — CADENCE suppressed nothing

**C3 — the rule in force matters, and it changed mid-window.** The wake mode is printed at every
boot, and it is not the same on the two days:

| boots | wake mode printed |
|---|---|
| 09-02 21:19:21, 21:32:52, 22:37:38, 22:41:58, 23:24:56 | `cutoff=25m cooldown=30m` — **WARN-first** |
| 09-03 10:28:29 onward (10 boots) | `cutoff=25m(enforce) cooldown=30m(enforce, fast-market≥1.5×ATR exempt)` |

So class-47 was advisory for all of 09-02 and enforcing for 09-03. **[A] It suppressed nothing on
either day.**

| event | 09-02 file (09-02 00:01→09-03 10:28) | 09-03 file (10:28→23:31) |
|---|---|---|
| wakes **fired** (`waking the planner`) | 43 | 8 |
| skipped on `wake_min_interval_min` | 446 | 47 |
| class-47 cutoff/cooldown lines | 8 | 1 |
| of those, actually **suppressed** | **0** | **0** |

All eight class-47 lines on 09-02 read `would_skip … WARN-first: the wake PROCEEDS`. The single
09-03 line reads:

```
09-03 19:50:05 ⏱ wake cooldown bypassed: fast market 2.4×ATR (cooldown 21 min …, cooldown 30m)
```

— the fast-market exemption working as designed. **The 446/47 `wake_min_interval_min` skips are
the designed cadence ceiling, not a class-47 refusal**, and 43 and 8 wakes still fired through it.

**D5 — the big moves and the plan in force.** The two impulse hours on 09-03 (07:00 +139.25,
10:00 +174.75) both occurred with the process **up** and a plan on the desk (NY v3–v6 were
authored at 09:15, 09:44, 10:14, 10:49 — four re-plans inside the rally). Cadence delivered
plans during the move. What did not follow was an entry, for the arm-enablement reason in §4.

The one genuine cadence hole is **not** class-47: after NY v7 died at 12:15:00 no NY re-plan was
authored, and 9 minutes later the host went down for 113 min 51 s. **[B]** Whether a re-plan
would have fired in that window cannot be determined — the process was not running to fire one.

---

## 10. C6 — touch_outcomes and candidate_pool: zero rows, and it is NOT a defect

```sql
SELECT COUNT(*) FROM touch_outcomes;   -- 0
SELECT COUNT(*) FROM candidate_pool;   -- 0
```

**Premise corrected.** The dispatch framed this as a defect candidate; it is not one **[A]**. The
1B recorder's **production call site landed 2026-09-03 21:31:00 CT** (commit `f1d7cf51`) and
**first ran in a live process at 21:48:12 CT**. The last planner read of the day completed at
**20:24:03 CT**. The hook has therefore never had an opportunity to fire. The tables themselves
were created by AutoMigrate at ~19:06:08 CT on 09-03.

The zero is **(a) the code only just entered the running binary**, not (b) a flag or (c) a wired
hook writing nothing. It settles on the LONDON read scheduled 01:30 CT 2026-09-04 — look for
`🔬 detector recorded: N new episode(s)`.

**But a real defect was found in the new code [A]:** `touch_outcomes.candidate_seated` is
degenerate — the only production writer stamps the literal `true`
(`trader/detector_record.go:66`), so the column can never distinguish a seated candidate from an
unseated one, and every analysis keyed on it will be vacuous from the first row written.

**Related: `trade_excursions` has ZERO rows all-time [A]**, and all 11 positions in the window
lack an excursion row. **This too is expected, not a runtime failure** — the excursion writer
merged to dev 09-03 10:22:16 CT and first booted at 10:28:29 CT, **67 minutes after position 591,
the last trade, had already closed**. `excursionOnClose()` deliberately early-returns for
pre-wave positions.

**Correction to a stronger claim this audit nearly made:** MAE/MFE are **not** missing. They are
present for all 11 positions in `trader_positions.mae`/`mfe` (0 NULLs) — e.g. 589 `mae=80.5
mfe=10.25`, 591 `mae=75.0 mfe=43.5` — written by the same `kernel.ComputePathExcursion` path the
excursion table uses. Only the excursion-table *copy* is absent. Expectancy degrades gracefully:
`loadExcursions()` returns an empty map and `BuildAt` still built `cells=41` at every boot.

The 30-minute after-exit excursions in §2 were computed by this audit from 1m bars and are
labelled as such; they answer a different question (what happened *after* the stop) than the
stored MAE/MFE (what happened *during* the trade).

---

## 11. E — ATTRIBUTION, and the baseline it is judged against

Cause vocabulary per the dispatch. Counts are **opportunities**, not log lines.

### 2026-09-02

| cause | n | detail |
|---|---|---|
| planner_shape (no arm authored on the bias side) | 55 | long scenarios authored without `arm.enabled` |
| gate_min_sl (validateDecision) | 6 clusters (16 events) | all arithmetically correct; ledger negative |
| gate_min_sl (EntryGate leg 6) | 3 | **fed the DAILY ATR — threshold 32× too large (D35)** |
| gate_rr_at_fill (EntryGate leg 5) | 2 | silent — `decision_records` only |
| gate_rr_at_arm | 5 | net −36.3 pts across the five |
| marketable_guard | 2 | ids 33, 34 |
| defect(restart→stale arm) | 1 | id 31 (+ ids 29/30 authored 09-01) |
| never_reached | 1 | id 32, EOD flat after 254.5 min |
| defect(double entry path) | 1 | arm 31 live at NT8 while the decision path opened the same scenario at market → position 587 |
| cadence_suppressed | **0** | class-47 in WARN mode |

### 2026-09-03

| cause | n | detail |
|---|---|---|
| **planner_shape** | **22** | long scenarios authored without `arm.enabled` during a `trend`/`long` plan |
| **silence (cause undetermined) + post-boot blindness** | **1 window** | 113m51s silent (12:24:33→14:18:24; only the last 16 min explained by the 14:02:25 kernel boot) + ~40 min blind after; covers the day's high and the last 2h of NY entry time |
| **gate_strict (EntryGate leg 0)** | **13** | 20:35→21:12 CT ASIA; refuses *every* decision-path market entry (D33). Cost nothing this day — after the trading session |
| parked(death) | 1 | arm 37; counterfactual **+38.75 pts** |
| never_reached | 1 | the only arm-enabled long, missed by **8.2 pts** |
| marketable_guard | 1 | id 36 |
| gate_min_sl | 5 clusters (18 events) | ledger negative |
| gate_rr_at_fill | 1 | silent |
| gate_rr_at_arm | 2 | net −64.4 pts |
| cadence_suppressed | **0** | one cooldown, fast-market-exempted |

### Baseline — the 7 days before 09-02

| day | positions | Σ pnl_corrected | arms authored | filled | fill rate | open-intent |
|---|---|---|---|---|---|---|
| 08-27 | 3 | (1 UNRESOLVABLE excluded) | 5 | 1 | 20% | 0.68% |
| 08-28 | 4 | — | 6 | 3 | 50% | 0.00% |
| 08-30 | 4 | (3 seam + 3 unresolvable excluded) | 4 | 1 | 25% | 0.00% |
| 08-31 | 6 | (3 NULL-corrected excluded) | 7 | 2 | 29% | 0.00% |
| 09-01 | 6 | +212.00 | 8 | 2 | 25% | 0.47% |
| **09-02** | **4** | **−381.50** | **4** | **0** | **0%** | 1.58% |
| **09-03** | **1** | **−140.00** | **3** | **1** | **33%** | 2.43% |

**[A] The fill rate on the two audited days (1 of 7 = 14%) is low but inside the historical band
(20–50%). What actually collapsed is arm SUPPLY: 5–8 arms/day across the baseline, 4 and 3 on the
audited days.** Combined with the arm-enablement asymmetry in §4, the constraint is upstream of
every gate — the planner is authoring fewer resting orders, and granting them almost only to
shorts.

---

## 12. THE DEFECT REGISTER

Every defect found, with the code path. None was acted on — this audit is read-only.

### Blocking the two days

| # | defect | evidence | path |
|---|---|---|---|
| **D1** | **113 min 51 s of silence — the first 98 min have NO determinable cause** | last act `log_events` 25754 @12:24:33 CT; kernel boot `uptime -s` **14:02:25**; no panic/fatal/shutdown banner in 12:20–12:25; journald retains nothing before 18:21:48 CT; NUL hole at log offset 779709 | **cannot distinguish** — process death vs host freeze |
| **D2** | **NT8 feed did not reconnect after the boot** | 46 `🚨 FEED DOWN` lines on 09-03; episode 1 runs 14:28:39→14:58:28, newest-bar age reaching **40m0s** | `trader/auto_trader.go:53` |
| **D3** | **~50 min of post-boot blindness with no alert** — 3 × `NT8 TCP link DOWN — NEW entries BLOCKED`, 3 × `no_balance_frame` (equity 0), next decision cycle not until **15:08:51** vs 12:22:44 | the real defect in the outage story; the NY flat is 14:45 | dead-man watchdog + balance-frame wait | **NOTE (liveness lane, 2026-09-04 — recorded, not built; the wave was dropped):** the reason this went unalerted is structural, not an oversight. The in-app alert path (`trader/auto_trader_alerts.go:15 emitAlert` → `store.Alert().Emit` → `day_plan_alerts` → the UI feed) is IN-PROCESS and DB-backed, so **a dead or silent process cannot write its own alert row** — the 98-minute silence was structurally unreportable by the bot, and the 40-minute blindness was reportable only to a UI nobody was watching on a weekday afternoon. Reporting silence requires a channel that does not depend on the thing being watched. AlertCenter can never serve that role; only a bot-independent path (Telegram, or an external watcher writing its own log) can.
| **D4** | **Planner arms shorts, not longs** — 4.3% vs 44.4% arm-enablement on 09-03 while the plan bias was `long`/`trend` | §4 | planner output; `plans.doc.scenarios[].arm.enabled` |
| **D5** | **09-03 NY v7 authored a LONG and a SHORT on the identical level 29543.75; both confirms became true at 11:58 CT; only the short carried an arm** | agent D4 section | the same |
| **D6** | **71% of arm-enabled scenarios (34 of 48) were authored at a price the tape never reached during that version's own life** | agent D4 section | the same |
| **D7** | **On 09-02 NY, six versions each carried a short arm at 29317.25 while the whole calendar day's high was 29255.00 — 62.25 pts below.** Unreachable by construction | agent D4 section | the same |
| **D8** | **Wake storm** — on 09-02 NY, 11 of 12 plan versions were woken by the identical never-clearing condition (`seated OB(bull)·4h invalidated: close X below 29246.25`) | agent C5 section | `auto_trader_wake_levels.go` |
| **D9** | **Structural 105-min wake blackout every weekday** — NY `WindowEndCT`=14:45 and ASIA `ReadCT`=16:30, so `inSessionReadWindow` is false for all of 14:45–16:30 | agent C5 section | `auto_trader_planner.go` |
| **D10** | **The 15-min stale-working reaper is a hard time-stop on every placed arm** regardless of link health — it killed arms 29/30/31 across a restart | `armed_executor.go:1021-1032` | — |

### Stop and R:R construction

| # | defect | evidence |
|---|---|---|
| **D11** | **591's stop was 8.3× its own invalidation distance** — stop 29351.63 (66.63 pts from entry) while the scenario's own invalid is "a 5m close above 29293.00 ONH" (8.00 pts). **The ATR floor is not the main cause: the planner itself authored `stop: 29340` = 55.00 pts = 6.875× the invalidation.** The floor added the last 11.63 pts; the planner supplied the other 47. A scenario that says "dead at 29293" carrying a stop at 29351.63 is authored incoherence, not a gate artefact |
| **D12** | **591's arm was placed 11.63 pts wider than even the plan's own wide stop** — plan `arm.stop = 29340`; ledger `stop_px = 29351.6284728996`, recomputed from live ATR at placement (`stop_floor_pts` 67.65 on the 09:06:54 read, `atr5m` 45.1) rather than taken from the plan |
| **D13** | **The R:R gate evaluates against `current_price_snapshot`, not the fill** (`kernel/engine_position.go:140,187`). Positions 587 and 589 filled 9.75 and 10.x pts away from the price the gate scored, so the R:R that passed is not the R:R that traded |

### Data integrity and observability

| # | defect | evidence |
|---|---|---|
| **D13b** | **`BackfillExcursions` has no automatic trigger** — it is reachable only from `cmd/excursions/main.go` (manual CLI) and one test, never from boot. Eleven boots since 09-03 10:28 all printed `backfilled=0`, so the 11 pre-wave positions stay unrecorded until an operator runs it | coverage gap |
| **D14** | `armed_orders.created_at` is stored with a **CT offset** and `updated_at` with **`+00:00` UTC** — same table, same subsystem, two clocks. Reading the raw column makes every arm look ~5h longer-lived than it was |
| **D15** | `plans.created_at` is **UTC** while `plan_lifecycle_log.at` is **CT** — sibling tables of the same subsystem disagree on the wall clock |
| **D16** | `planner_read_facts.bias_ai` and `bias_tree` are **empty in all 17 rows**, and `plan_id=''`/`version=0` in all 17 (`persistReadFacts`, `auto_trader_planner.go:2420-2453`). The AI and tree bias labels the dispatch asked for **do not exist in the store** |
| **D17** | `planner_read_facts` holds **17 rows, all `trade_date='2026-09-03'`** (ids 1–17 contiguous, 00:00:56→20:24:03 CT) and **zero for 09-02** — the table was created with the wave. *Measurement confirmed; the "defect" reading is not supported* — this is a feature that had not shipped on 09-02, so 09-02's regime is simply unrecorded, not lost |
| **D18** | `plan_lifecycle_log` holds **2 rows all-time** (09-03 ASIA v2 @19:25:00, ASIA v5 @21:21:00 CT); the fix that creates it landed 09-03 17:40:18 CT, so it covers **8 of 51** versions in the window (15.7%, n=51). 09-03 LONDON v1's full history — PLAN 01:34:53 → DORMANT 02:40:00 → REARMED 04:30:55 → DORMANT 08:00:54 → superseded 08:15:44 — is recoverable only from the log file, not the lifecycle table |
| **D19** | **38 of 51 plan versions** in the window record their trigger as the bare token `level_event` — *which* level woke the planner is never persisted (n=51 versions created 09-02 00:00 CT onward) |
| **D20** | Until commit `4e901261` (09-03 17:40:18 CT), `UpdatePlanLifecycle` **overwrote `plans.trigger_reason`** with the dormant/flip marker, destroying the authoring trigger. This is why §6's version table shows death markers in the trigger column |
| **D21** | **`data/nofx_<date>.log` rotates on PROCESS START, not on date.** Position 591's close (09-03 09:20:45 CT) is at line 85591 of **`nofx_2026-09-02.log`**. Any grep scoped by filename silently misses events |
| **D22** | **`nofx_2026-08-27.log` is 2,049,637,081 bytes** (2.0 GB) against 0.6–13 MB for every other day — 180,139 lines in the first 40 MB are empty-payload `📡 armed order_update frame` spam |
| **D23** | **The min-SL refusal line prints the raw ATR where the threshold belongs** (`kernel/min_sl.go:62`), so a correct refusal reads as an arithmetic error. All 34 were correct |
| **D24** | **A min-SL-rejected decision is not persisted with its prices** — the retry overwrites the record with the final `wait`, so the refused set's exact counterfactual is unrecoverable from the store |
| **D25** | `armed_orders` rows for positions **582 and 585 were overwritten in place** by later re-arms (`UNIQUE INDEX idx_armed_orders_plan_scenario`), so their brackets are gone |
| **D26** | `source='reconcile'` rows (584, 586, **591**) carry `entry_time` = **materialization time, not fill time** (`trader/ninjatrader/reconcile.go:412` sets `EntryTime=nowMs` after a 60s grace) |
| **D27** | Entry fills exist in `trader_fills` **only for `source='system'`** positions; `armed_entry` and `reconcile` positions have exit fills only |
| **D28** | `WakeCadenceBootLine` (`trader/class47_wake_cadence.go:222`) claims the cutoffs govern `LEVEL_EVENT`/`structure_mss` wakes only, but `maybeWakePlannerOnMSSAt` does not honour that — the boot line misdescribes the running rule |
| **D29** | `collectLevelWakeCandidates` emits **no log line** when it returns zero candidates (`auto_trader_wake_levels.go:263-265`) and none when the bars provider is nil — a silent wake path |
| **D30** | Boot-time rev-mismatch `🔐 TRADING REFUSED` fired **twice** in the window (09-02 00:10:21, 09-03 23:11:59) — two cutovers shipped the wrong binary |
| **D31** | **8 of 21 big-move windows (|15m open→close| ≥ 40 pts) drew no wake line at all.** *The "190–370 min stale" band attached to six of them did not survive verification and is withdrawn* — the 8-of-21 count reproduces independently; the staleness figure does not |

### Gate defects (added after the D2 pass corrected this audit's first reading)

| # | defect | evidence |
|---|---|---|
| **D32** | **The decision-path entry-gate refusal is invisible** — `entryGateDecisionTelemetry` (`trader/entry_gate.go:477-486`) writes no log line and no counter, only `actionRecord.Error`. 19 refusals in the window produced **0** log lines and **0** `log_events` rows. This audit's first pass therefore concluded the gate refused nothing |
| **D33** | **`plan_mode=strict` + EntryGate leg 0 refuses EVERY decision-path market entry regardless of citation** — 13 refusals 09-03 20:35:06→21:12:40, all citing a real scenario. Since commit `c8c90dcc` (09-03 10:43 CT) the decision path is closed to market entries and nothing says so |
| **D34** | **A29 DEAD GATE — `armGateVerdict` (`trader/armed_executor.go:1268`) has ZERO production callers**; all 8 call sites are in `armed_executor_test.go`. Production uses `armGateVerdictFor` at `:415` |
| **D35** | **EntryGate min-SL leg was fed the DAILY ATR on 09-02 evening** — threshold rendered `1.5×ATR5m = 450.56` from a daily ATR of 300.37, while the arm seam used ATR5m 12.78–14.12 in the same minutes (**32×**). Fixed by `609067ec`, 09-03 08:23 CT |
| **D36** | **The no-chase leg cannot fire on the arm path** — 40 evaluations, all `dist = 0.00×ATR`, `run = NULL`, 0 `would_refuse`, because `citedLevelFor` returns `sc.Confirm.RefPrice` which equals the arm entry in every plan in the window. Its 27 log lines are noise |
| **D37** | **All 7 R:R-at-arm refusals were authored at R:R ≥ 2.0** and pushed below the 2.0 floor by `composeArmStop` widening the stop. The plan cleared the gate; the stop composer broke it |
| **D38** | **`max_daily_trades=3` is not enforced** because the guardrails master switch is off. It would have blocked every entry from 09-02 09:59:29 CT onward; position **590** (−99.00) opened straight through it |
| **D39** | **`UpsertArm` rewrites `entry_px`/`stop_px`/`target_px` on a row whose state is `working`** — i.e. under a LIVE broker order — leaving `signal_id` and `state` intact and sending no cancel or replace. Arm 29's ledger says 29044.00; the order actually resting at NT8 was a buy limit at **29035.25** |
| **D40** | **`armed_orders` is not append-only** — the `(plan_id, scenario, leg_index)` slot is overwritten, so 15 rows under-represent **at least 11 distinct broker placements**; three signal ids (`0c77307d`, `fd71b48f`, `18a7cc55`) survive nowhere, and two of them **filled into real positions** (582 +129.5, 585) |
| **D41** | **No independent broker-side record exists for any arm** — `trader_orders` holds **1 row all-time** (2026-06-02) and the NT8 armed path writes nothing there; `nt8_order_snapshots` holds 228 rows all with `symbol=''` (blank). There is no way to reconcile the ledger against the broker |
| **D42** | **Double entry path** — arm 31 (ASIA S3 long limit 29068.05, signal `dc8405f1`) was live at NT8 from 00:16:32 while the **decision** path opened the SAME plan+scenario at market 29079.25 at 00:17:44 (position 587). Both live together for 14m04s |
| **D44** | **Two owner-set knobs are silently off** — strategy `MNQ` carries `breakeven_enabled: true` (trigger 40 pts) and `trailing_enabled: true`; the 0B wave suspends both, and **two boot lines in the same boot disagree**: `🛑 exits: … BE=off · trail=off` vs `🧾 ledger boot: … trailing=2.0×ATR14 arm=after_breakeven (source: studio)` |
| **D43** | **Zero arms authored under a 09-02-dated plan ever reached the broker** (ids 32, 33, 34). The single 09-02 broker placement (00:16:32, arm 31) belonged to the 09-01 ASIA plan. All four 09-02 positions entered via the decision path |

---

## 13. D6 — THE ONE TRADE, in full

**Arm 35 → position 591.** NY S1, short, entry 29285.00, stop 29351.63, target 29144.50.
Armed 09-03 09:02:54.66 CT, exited 09:20:45 @29355.00, **−140.00**, grade A.
**[A] The stored `entry_time` 09:05:14.627 is the reconcile *materialization* instant, ~81 s after
the actual fill** (`trader/ninjatrader/reconcile.go:412` sets `EntryTime=nowMs` after a 60 s
grace) — the true fill is ≈09:03:53. Every `source='reconcile'` row carries this offset (D26).

**Why it was authored.** Plan `2026-09-03:NY` v2 (08:45:05 CT), bias `short`/medium,
`day_type: "balance"`. Scenario S1 in the plan's own words:

> "A retest of 29285.00 OR-H stalls after the 08:30 failure; short the touch."
> `invalid`: "A 5m close above 29293.00 ONH voids the fade."
> `arm`: `{enabled: true, entry: 29285, stop: 29340, target: 29144.5, wait_confirm: true}`

The plan's reasoning names the premise: *"I lean short fades unless 29285 is reclaimed… S1 carries
a resting short at 29285 because R:R clears the min-stop gate."* The machine regime on that same
read was **`up/NORMAL`**, and the day was called **"balance"** on a day that ran 510 points.

**Why it filled.** Price genuinely traded 29285.00 at 09:05 CT. The fill is correct.

**What killed it.** A stop-out at the arm's own stop with 3.37 pts of slippage
(29351.63 → filled 29355.00). Not a market close, not a sync artefact.

**Where the stop came from — and the correction this audit owes.** The obvious reading is that the
ATR floor blew the stop out. **[A] It did not, mostly.** The plan's own authored stop was
`29340` = **55.00 pts**, already **6.875×** the scenario's stated invalidation distance
(29293.00 is 8.00 pts from the entry). The ledger's **29351.6284728996** is the live-ATR floor
(`stop_floor_pts` 67.65 against `atr5m` 45.1 on the 09:06:54 read) adding a further **11.63 pts**.
So of the 66.63-pt stop, the planner authored 55 and the floor added 11.63. **The scenario declared
itself dead at 29293 and then carried a stop 58 points beyond that.** Price closed above 29293 and
the trade kept running to 29355 — losing 140 on a thesis its own document had already voided.

**What refused everything else that session.**

| time (CT) | event |
|---|---|
| 09:20:47 | arm 36 (short 29351.05) cancelled — price 29358.75 already above entry → **marketable_guard** |
| 09:15:31 | plan flips **long** (v3), `day_type: trend` — and authors long scenarios **without arms** |
| 10:49:57 | the day's only arm-enabled long (29481.05) goes live; **misses by 8.2 pts**; superseded 11:27:53 |
| 11:58:33 | arm 37 (short 29543.75) placed — the *short* half of v7's two scenarios on that level |
| 12:15:00 | v7 dies (2×5m close below 29502.25) → arm 37 disarmed, 36 min before price reached it |
| 12:24:33 | **silence begins — 113 min 51 s; first 98 min have no determinable cause**; kernel boot 14:02:25 |
| 14:18:24 | host+process boot; **FEED DOWN ×31**, `NT8 TCP link DOWN — NEW entries BLOCKED`, `no_balance_frame` — blind for the remaining 27 min of NY |
| 14:45 | NY flat |

**The tape after the exit.** From 29355.00 the market went to **29585.00** (+230). Had 591 been
held it would have been −460 at the extreme; the stop-out was correct. Had the system flipped long
at 29355.00 it would have had **+230 points (+$460)** available on the day's continuation. It
proposed `open_long` zero times.

---

## 14. G — VERDICT

**[A]** Two days that closed −521.50 combined (09-02 −381.50 on n=4, 09-03 −140.00 on n=1, zero
exclusions) did not lose because the gates were too tight. Sixty-one refusal events fired across
**44 distinct opportunities**, and taking every one of them at its authored stop and target on the
actual tape **loses $860.64** — the gates saved roughly a day's worth of losses rather than costing
one. **No leg's ledger says "review":** min-SL at n=34 events / 11 clusters would have been stopped
on its own too-tight stops 18 times for −431.2 pts and is positive only under an unattainable
perfect-exit assumption; R:R-at-arm at n=7 nets −7.03 pts and sits below the n=10 floor, so no
verdict is stated for it; the one winning cluster (09-02 18:47–19:00 ASIA, 7 opportunities all
reaching target) is worth **+$55.22** as the single sequential position it actually was. Four of
the named new legs — one-open-position, invalidation-wired, shadow map, direction — are among
**fourteen wired legs that never fired once**, and, decisively, **none of the legs the dispatch
calls "new" existed during any of the five losing trades**: strict and invalidation entered force
at 09-03 11:10:33 CT and one-open-position at 14:59:03, while the last loss closed at
**09-03 09:20:45**. A rule that post-dates every loss cannot have caused one. **Gates account for roughly 5%**, and the real gate
findings are structural rather than costly: `plan_mode=strict` refuses **every** decision-path
market entry regardless of citation and announces this nowhere; the decision-path refusal recorder
writes no log line and no counter, so 19 refusals were invisible until `decision_records` was read
directly (this audit's own first pass concluded "the gate refused nothing" and had to withdraw it);
`armGateVerdict` has **zero production callers**; the no-chase leg is arithmetically incapable of
firing; and on 09-02 evening the EntryGate min-SL leg was fed the **daily** ATR, rendering a
threshold 32× too large until `609067ec` fixed it on 09-03. Cadence accounts for **~0%**: class-47
ran WARN-first for all of 09-02 and enforcing on 09-03, and across both days it suppressed **zero**
wakes — its one enforcing firing was fast-market-exempted, and 43 and 8 wakes fired through the
`wake_min_interval_min` ceiling. What is left is planner shape and a defect. **Planner shape is the
majority, ~55%**: on 09-03 the system read the trend correctly — `up/NORMAL` on all 17 planner
reads, plan bias `long` with `day_type: trend` from 09:15 CT — and then authored its long scenarios
**without resting arms**, 1 of 23 (4.3%) against 8 of 18 shorts (44.4%), leaving the longs to a
decision loop that proposed `open_long` **zero times in 575 decisions**; v7 put a long and a short
on the *identical* level 29543.75 with both confirms true at 11:58 CT and armed only the short;
71% of all arm-enabled scenarios were authored at prices their own version never saw, and on 09-02
six versions carried a short arm 62.25 points above the day's high. **Something that is neither a rule nor the tape accounts for
~30%**, and it is two things, not one: the process fell silent at **12:24:33 CT** and stayed silent
for **113 min 51 s**, of which only the last 16 minutes are explained by the host's kernel boot at
14:02:25 — **the first 98 minutes have no determinable cause** (no panic, fatal or shutdown banner,
and journald retains nothing before 18:21:48 CT, so this is a CANNOT-DISTINGUISH, not an
infrastructure finding dressed up as one) — and then the boot came up **blind**, with 31 `FEED DOWN` lines, the
dead-man watchdog logging `NT8 TCP link DOWN — NEW entries BLOCKED`, and `no_balance_frame` at
equity 0, so the next decision cycle after 12:22:44 was **15:08:51**; the NY flat is 14:45, so the
entire afternoon containing the day's high of 29585 had either no host or no data, and **the
second half of that — ~50 minutes of a live process unable to enter — raised no alert.** The
restart cadence (11 boot banners from 6 build trees on 09-03 alone, one landing mid-session
between NY v6 and v7; two rev-guard `TRADING REFUSED` cutovers) is a real hazard but is **not** the
cause of this gap. **No-setup accounts for ~10%**: the single
arm-enabled long of 09-03 missed by 8.2 points and one minute, and the five losses each stopped at
their own authored stop, three of them within ~9 points of the exact turning point before the
market ran 54–93 points their way.

**Gates whose ledger says "review": none.** min_sl at n=34 is doing its job; rr_at_arm is at n=7,
below the floor. **Every other leg has n=0 and cannot be assessed.**

**Can the system take a trade on a trending fast tape, as configured today? [A] No — not
reliably.** The rule that prevents it is **not a gate**. It is that the planner grants resting arms
almost exclusively to short scenarios, and the decision path — the only route a long has — did not
propose a single long during a 483-point rally. A fade-only *arm* set inside a correctly-identified
*long* plan is a planner-shape answer, and it is the answer here.

---

## 15. CSVs

All under `docs/superpowers/reports/2026-09-04-two-day-audit-data/`:

| file | contents |
|---|---|
| `trades_window.csv` | every closed position 09-01→now with plan lineage |
| `refusals_minsl_counterfactual.csv` | all 34 min-SL refusals with threshold check + 30/60-min excursions |
| `refusals_rr_counterfactual.csv` | all 7 R:R-at-arm refusals with full stop/target outcome |
| `minsl_rejects_raw.txt` | the raw refusal lines |
| `arms.csv` | every `armed_orders` row with cause class |
| `plans.csv` / `d6_plan_0903NY_v2.json` | plan versions, scenarios, condition-truth |
| `cadence.csv` | wake event ledger |
| `tape.csv`, `tape_moves_0903.csv`, `d6_bars_1m_0903.csv` | the tape |
| `baseline.csv`, `decision_intent_by_day.csv` | the 7-day baseline |
| `c6-*.csv` | 1B writer inventory and merge-vs-boot timeline |

## 16. Limits of this audit

- **[A]** `GET /api/config/resolved` and `/api/risk/gate-blocks` require an Authorization header
  this session does not have. Live knob values are taken from the process's own boot lines and
  from refusal messages that print the multiplier they applied — never from a file default
  presented as live. Knobs not established that way are named, not guessed.
- **[A]** `bias_ai` and `bias_tree` are empty in every `planner_read_facts` row (D16), and the
  table has no 09-02 rows at all (D17). The AI/tree bias-accuracy table the dispatch asked for
  **cannot be built** — only `bias_regime` and the plan doc's own `bias.direction` exist.
- **[A]** `trade_excursions` is empty (0 rows all-time) because the writer shipped 67 minutes
  after the last trade closed. MAE/MFE **do** exist for all 11 positions in
  `trader_positions.mae`/`mfe`; the 30-min after-exit excursions in §2 are a different measurement,
  computed by this audit from 1m bars and labelled as such.
- **[A]** The exact entry/stop/target of a min-SL-refused decision is unrecoverable (D24); §8's
  min-SL ledger therefore uses each refusal's own stop distance against realised excursion, which
  is stated in place rather than dressed up as a stop/target counterfactual.
- **[B]** Whether a re-plan would have fired between 12:15 and 14:18 CT on 09-03 cannot be
  determined — the process was not running to fire one.
