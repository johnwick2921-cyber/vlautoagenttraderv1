#!/usr/bin/env python3
"""q06 — DAY-LEVEL Monte Carlo (the dispatch's ask: by session-day, not trade).
Unit of resampling = a realised session-day (its full trade sequence, in order). Seeded, B=10000.
Reports: expectancy per day CI (t and day-bootstrap), drawdown quantiles at horizons of 20/50/100 TRADES
reached by concatenating resampled days, streak probabilities under day resampling, effective sample in days."""
import csv, math, statistics as st
from collections import defaultdict
import numpy as np
SEED, B = 20260905, 10_000
rng = np.random.default_rng(SEED)
rows = list(csv.DictReader(open('trade_sample.csv')))
days = defaultdict(list)
for r in rows: days[r['session_day_ct']].append(float(r['pnl_corrected']))
keys = sorted(days); D = len(keys)
day_pnl = np.array([sum(days[k]) for k in keys]); day_n = np.array([len(days[k]) for k in keys])
pnl = np.array([float(r['pnl_corrected']) for r in rows]); n = len(pnl)
print("="*78)
print(f"DAYS D={D}  trades n={n}  ids {rows[0]['id']}..{rows[-1]['id']}")
for k in keys: print(f"  {k}: n={len(days[k]):>2}  sum={sum(days[k]):+8.2f}  seq={[f'{x:+.0f}' for x in days[k]]}")
print(f"\nDAY P&L: mean={day_pnl.mean():+.2f}  sd={day_pnl.std(ddof=1):.2f}  median={np.median(day_pnl):+.2f}  min={day_pnl.min():+.2f} ({keys[int(day_pnl.argmin())]})  max={day_pnl.max():+.2f} ({keys[int(day_pnl.argmax())]})")
print(f"  green days={int((day_pnl>0).sum())}  red days={int((day_pnl<0).sum())}  trades/day median={np.median(day_n):.0f} mean={day_n.mean():.2f}")
se = day_pnl.std(ddof=1)/math.sqrt(D); tcrit = {11:2.201,12:2.179,13:2.160,14:2.145}.get(D-1, 2.2)
print(f"  per-day expectancy 95% t-CI (df={D-1}, t={tcrit}): [{day_pnl.mean()-tcrit*se:+.2f}, {day_pnl.mean()+tcrit*se:+.2f}]  t={day_pnl.mean()/se:+.3f}")
# day bootstrap of the per-day mean
bm = np.array([rng.choice(day_pnl, size=D, replace=True).mean() for _ in range(B)])
print(f"  per-day expectancy day-bootstrap 95% pct-CI: [{np.percentile(bm,2.5):+.2f}, {np.percentile(bm,97.5):+.2f}]  P(mean>0)={float((bm>0).mean()):.3f}")
# per-TRADE expectancy via day-cluster bootstrap (respects within-day dependence)
bt = []
for _ in range(B):
    pick = rng.integers(D, size=D); tot = sum(day_pnl[i] for i in pick); cnt = sum(day_n[i] for i in pick)
    bt.append(tot/cnt)
bt = np.array(bt)
print(f"  per-TRADE expectancy, day-cluster bootstrap: mean={pnl.mean():+.3f}  95% pct-CI [{np.percentile(bt,2.5):+.2f}, {np.percentile(bt,97.5):+.2f}]  P(>0)={float((bt>0).mean()):.3f}")
# effective sample size: design effect from intra-day correlation (ICC, ANOVA estimator)
grand = pnl.mean()
ssb = sum(len(days[k])*(np.mean(days[k])-grand)**2 for k in keys); ssw = sum(sum((x-np.mean(days[k]))**2 for x in days[k]) for k in keys)
msb = ssb/(D-1); msw = ssw/(n-D); n0 = (n - sum(len(days[k])**2 for k in keys)/n)/(D-1)
icc = max((msb-msw)/(msb+(n0-1)*msw), 0.0); deff = 1 + (n0-1)*icc
print(f"  intra-day ICC={icc:.3f}  design effect={deff:.2f}  effective n (trades)={n/deff:.1f}  effective sample in DAYS={D}")
def maxdd(path):
    eq = np.cumsum(path); peak = np.maximum.accumulate(np.concatenate(([0.0], eq)))[1:]; return float(np.max(peak-eq))
def day_paths(h, b=B):
    out = np.empty((b,h))
    for i in range(b):
        seq = []
        while len(seq) < h: seq.extend(days[keys[rng.integers(D)]])
        out[i] = seq[:h]
    return out
def max_streak(p):
    best=cur=0
    for x in p:
        cur = cur+1 if x<0 else 0; best=max(best,cur)
    return best
print("\nMAX DRAWDOWN by DAY resampling ($, 1 contract):")
print(f"{'horizon':>8} {'p50':>7} {'p90':>7} {'p95':>7} {'p99':>7} {'worst':>7}")
res = []
for h in (20,50,100):
    P = day_paths(h); dd = np.array([maxdd(p) for p in P])
    print(f"{h:>8} {np.percentile(dd,50):>7.0f} {np.percentile(dd,90):>7.0f} {np.percentile(dd,95):>7.0f} {np.percentile(dd,99):>7.0f} {dd.max():>7.0f}")
    res.append(dict(horizon=h, method='day', p50=np.percentile(dd,50), p90=np.percentile(dd,90), p95=np.percentile(dd,95), p99=np.percentile(dd,99), worst=dd.max()))
    if h == 50:
        s = np.array([max_streak(p) for p in P])
        print("  P(losing streak >= k in 50 trades, day-resampled): " + "  ".join(f"k{k}={float((s>=k).mean()):.3f}" for k in (4,6,8,10)))
# drawdown in DAYS: over 10/20/40 session-days
print("\nMAX DRAWDOWN over N session-DAYS (day resampling):")
for nd in (10,20,40):
    dd = np.array([maxdd(rng.choice(day_pnl, size=nd, replace=True)) for _ in range(B)])
    print(f"  {nd:>3} days: p50={np.percentile(dd,50):.0f} p90={np.percentile(dd,90):.0f} p95={np.percentile(dd,95):.0f} p99={np.percentile(dd,99):.0f} worst={dd.max():.0f}")
    lr = np.array([max_streak(rng.choice(day_pnl, size=nd, replace=True)) for _ in range(B)])
    print(f"           P(>=3 red days in a row)={float((lr>=3).mean()):.3f}  P(>=5)={float((lr>=5).mean()):.3f}")
# realised path
eq = np.cumsum(pnl); print(f"\nREALISED: final={eq[-1]:+.2f}  peak={eq.max():+.2f} at trade #{int(eq.argmax())+1}  max DD={maxdd(pnl):.2f}  realised max losing streak={max_streak(pnl)}")
deq = np.cumsum(day_pnl); print(f"REALISED by day: peak={deq.max():+.2f}  max DD (days)={maxdd(day_pnl):.2f}  red-day streak={max_streak(day_pnl)}")
with open('day_mc_drawdown.csv','w',newline='') as f:
    w=csv.DictWriter(f, fieldnames=list(res[0])); w.writeheader(); w.writerows(res)
print("="*78)
