# C1 — THE HTF VETO IS A DEAD GATE (evidence pack)

All queries read-only: `sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro"`; logs read from
`/home/hoang/nofx/data/nofx_2026-*.log`. Clock CT = UTC-5.

## 1. The mechanism

- `HTF_VETO_MODE=cross` — `/home/hoang/nofx/.env:34` (the ONLY C-subsystem env knob set).
- `kernel/htf_veto.go:92-101` — cross requires **both** `snap["1h"]` and `snap["4h"]`:
  `if !ok1 || !ok4 || ... { return false, "" }`.
- `kernel/structure.go:34` — `var StructureTFs = []string{"5m", "15m", "1h"}`.
  `StructureSnapshot` (`kernel/structure.go:392-407`) only ever writes keys from that slice.
- Therefore `ok4` is **always false** → `HTFVetoVerdict` returns `(false, "")` on every call.

## 2. The tape confirms it

```sql
SELECT COUNT(*) total,
       SUM(structure_json LIKE '%"4h"%') has_4h,
       SUM(structure_json LIKE '%"1h"%') has_1h
FROM decision_records WHERE structure_json NOT IN ('', 'null');
-- 6036 | 0 | 6019      (rows span 2026-08-23 22:13 .. 2026-09-04 13:44)
```

Log fires of `🛡️ HTF VETO`, per day file:

| 08-23 | 08-24 | 08-25 | 08-26 | 08-27 | **08-28** | 08-29 | 08-30 | 08-31 | 09-01 | 09-02 | 09-03 | 09-04 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 6 | 3 | 9 | 4 | 5 | **0** | 0 | 0 | 0 | 0 | 0 | 0 | 0 |

`cross` landed in `84543213` (2026-08-28 10:43:25 CT, "F3 HTF veto cross mode").
The gate has not fired once in the 7 days since. `2026-08-26-week-in-review.md:67`
counted 52/18/22/63/84/5 htf_veto lines on 08-21..08-26 (incl. retry echoes).

## 3. What it cost — [T], own tape, n named

Entry decisions since the cross cutover whose action opposes the **confirmed 1h trend**
(exactly what mode=1h would have refused):

```sql
WITH e AS (SELECT d.id, datetime(d.timestamp,'-5 hours') ct,
                  json_extract(d.structure_json,'$."1h".trend') t1h,
                  json_extract(j.value,'$.action') act, d.risk_check_passed rp
           FROM decision_records d, json_each(d.decisions) j
           WHERE d.structure_json<>'' AND datetime(d.timestamp,'-5 hours')>='2026-08-28 10:43'
             AND json_extract(j.value,'$.action') IN ('open_long','open_short'))
SELECT t1h, act, rp, COUNT(*) FROM e
WHERE (t1h='TRENDING_UP' AND act='open_short') OR (t1h='TRENDING_DOWN' AND act='open_long')
GROUP BY 1,2,3;
-- TRENDING_DOWN | open_long  | 0 |  1
-- TRENDING_DOWN | open_long  | 1 |  4
-- TRENDING_UP   | open_short | 0 | 14
```

**n = 19** counter-1h-trend entry decisions. 15 were stopped by some *other* gate.
**4 passed every gate** (`risk_check_passed=1`) — all four `open_long` against a
confirmed 1h `TRENDING_DOWN`:

| decision ct (CT) | position | side | entry | pnl_corrected | mae | mfe |
|---|---|---|---|---|---|---|
| 2026-09-02 00:17:45 | 587 | LONG | 29079.25 | **-62.50** | 33.00 | 25.75 |
| 2026-09-02 07:41:06 | 588 | LONG | 29082.50 | **-65.00** | 33.25 | 0.00 |
| 2026-09-02 09:41:05 | 589 | LONG | 29192.50 | **-155.00** | 80.50 | 10.25 |
| 2026-09-02 10:37:18 | 590 | LONG | 29193.25 | **-99.00** | 49.75 | 1.00 |
| | | | | **-381.50** | | |

(position timestamps from `trader_positions.entry_time`, within 1s of each decision row.)

**-$381.50** is exactly the loss the no-chase wave was built to study
(`trader/no_chase.go:16-21`: "four longs, four stops, -$381.50 ... Their MFE was 10.25
and 1.00"). The no-chase leg is WARN-first and refuses nothing
(`trader/entry_gate.go:288-294` returns `("", false)`), while a gate that WOULD have
refused all four was silently unsatisfiable.

## 4. Where the research said otherwise

- `2026-08-28-grand-audit-bcde-verdict.md:17` (S4): "4h was RANGING at all 7 veto
  timestamps → a 4h-cross-check would have allowed all 7 (hypothetical net +$833)".
  Correct as a replay — and it is also the reason cross can never veto: the 4h leg
  never opposes because it is never present in production at all.
- `2026-08-29-weekend-audit-p2.md:42` claims "HTF-veto-cross **n=9** -$114.0 SAVING".
  That n=9 is a replay of 1h-era refusals relabelled cross; the live gate has n=0
  since the switch. `:33` and `:48` KEEP cross on that basis.
- `2026-09-01-full-system-audit.md:621` already recorded "LIVE (unfired since 08-30)"
  without diagnosing why.
