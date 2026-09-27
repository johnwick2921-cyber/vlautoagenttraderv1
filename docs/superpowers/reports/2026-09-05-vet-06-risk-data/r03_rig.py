#!/usr/bin/env python3
"""r03 — full re-run of every primary figure on the COMPLIANT population (n=58),
with the broad n=65 cut printed beside it as a labelled sensitivity."""
import csv, math, statistics as st
from collections import defaultdict
import numpy as np
SEED, B = 20260905, 10_000

def wilson(k, n, z=1.96):
    if n == 0: return (0.0, 0.0)
    p = k/n; d = 1 + z*z/n
    c = (p + z*z/(2*n))/d; h = z*math.sqrt(p*(1-p)/n + z*z/(4*n*n))/d
    return (c-h, c+h)

def maxdd(path):
    eq = np.cumsum(path); peak = np.maximum.accumulate(np.concatenate(([0.0], eq)))[1:]
    return float(np.max(peak-eq))

def dd_span(path):
    eq = np.cumsum(path); peak = np.maximum.accumulate(np.concatenate(([0.0], eq)))[1:]
    dd = peak-eq; i = int(np.argmax(dd))
    # find the peak index that set this drawdown
    j = int(np.argmax(eq[:i+1])) if i > 0 else 0
    pk = float(eq[:i+1].max()) if i >= 0 else 0.0
    if pk <= 0: j = -1
    return float(dd[i]), j+1, i+1   # peak trade #, trough trade #

def max_streak(p):
    best=cur=0
    for x in p:
        cur = cur+1 if x < 0 else 0; best = max(best,cur)
    return best

def max_nonwin_streak(p):
    best=cur=0
    for x in p:
        cur = cur+1 if x <= 0 else 0; best = max(best,cur)
    return best

for LABEL, FN in (("COMPLIANT n=58 (PRIMARY)", "trade_sample_58.csv"), ("BROAD n=65 (sensitivity)", "trade_sample_65.csv")):
    rng = np.random.default_rng(SEED)
    rows = list(csv.DictReader(open(FN)))
    pnl = np.array([float(r['pnl_corrected']) for r in rows]); n = len(pnl)
    days = defaultdict(list)
    for r in rows: days[r['session_day_ct']].append(float(r['pnl_corrected']))
    keys = sorted(days); D = len(keys)
    day_pnl = np.array([sum(days[k]) for k in keys]); day_n = np.array([len(days[k]) for k in keys])
    print("="*84); print(LABEL, " ids", rows[0]['id'], "..", rows[-1]['id'])
    mean, sd = pnl.mean(), pnl.std(ddof=1); se = sd/math.sqrt(n)
    print(f"EXPECTANCY: n={n} sum={pnl.sum():+.2f} mean={mean:+.4f} sd={sd:.3f} se={se:.3f} "
          f"95% CI [{mean-1.96*se:+.2f}, {mean+1.96*se:+.2f}] t={mean/se:+.3f}")
    w = int((pnl>0).sum()); l = int((pnl<0).sum()); f0 = int((pnl==0).sum())
    lo1,hi1 = wilson(w, w+l); lo2,hi2 = wilson(w, n)
    print(f"  W/L/F={w}/{l}/{f0}  p(win) flats-excluded {w}/{w+l}={w/(w+l):.3f} Wilson [{lo1:.3f}, {hi1:.3f}] | "
          f"flats-in-denominator {w}/{n}={w/n:.3f} Wilson [{lo2:.3f}, {hi2:.3f}]")
    wins = pnl[pnl>0]; loss = pnl[pnl<0]
    print(f"  avg win {wins.mean():+.2f} (n={len(wins)})  avg loss {loss.mean():+.2f} (n={len(loss)})  payoff {abs(wins.mean()/loss.mean()):.3f}")
    # n_required, both conventions
    for tgt in (5,7,10,15,20):
        n_ci = (1.96*sd/tgt)**2; n_pw = ((1.96+0.8416)*sd/tgt)**2
        print(f"  edge ${tgt}/trade -> n(CI clears at observed mean)={n_ci:.0f}  n(80% power)={n_pw:.0f}")
    n_ci_cur = (1.96*sd/abs(mean))**2; n_pw_cur = ((1.96+0.8416)*sd/abs(mean))**2
    print(f"  at the CURRENT effect |{mean:.3f}|: n(CI)={n_ci_cur:.0f}  n(80% power)={n_pw_cur:.0f}")
    print("  Stage-A mean required (lower 95% CI > 0  =>  mean > 1.96*sd/sqrt(n)):")
    for nn in (30,50,58,65,100,200,400):
        print(f"    n={nn:>4}  ${1.96*sd/math.sqrt(nn):.2f}  ({1.96*sd/math.sqrt(nn)/2.0:.1f} MNQ pts)")
    # DAY level
    print(f"DAYS D={D}")
    for k in keys: print(f"    {k}: n={len(days[k]):>2} sum={sum(days[k]):+8.2f}")
    dse = day_pnl.std(ddof=1)/math.sqrt(D); tcrit = 2.201 if D==12 else 2.2
    print(f"  day mean={day_pnl.mean():+.2f} sd={day_pnl.std(ddof=1):.2f} median={np.median(day_pnl):+.2f} "
          f"min={day_pnl.min():+.2f}({keys[int(day_pnl.argmin())]}) max={day_pnl.max():+.2f}({keys[int(day_pnl.argmax())]}) "
          f"green={int((day_pnl>0).sum())} red={int((day_pnl<0).sum())} trades/day med={np.median(day_n):.0f}")
    print(f"  per-day 95% t-CI (df={D-1}) [{day_pnl.mean()-tcrit*dse:+.2f}, {day_pnl.mean()+tcrit*dse:+.2f}] t={day_pnl.mean()/dse:+.3f}")
    bm = np.array([rng.choice(day_pnl, size=D, replace=True).mean() for _ in range(B)])
    print(f"  day-bootstrap pct-CI [{np.percentile(bm,2.5):+.2f}, {np.percentile(bm,97.5):+.2f}] P(mean>0)={float((bm>0).mean()):.3f}")
    dn_ci = (1.96*day_pnl.std(ddof=1)/abs(day_pnl.mean()))**2; dn_pw = ((1.96+0.8416)*day_pnl.std(ddof=1)/abs(day_pnl.mean()))**2
    print(f"  days required at current day effect: CI={dn_ci:.0f}  80% power={dn_pw:.0f}")
    for tgt in (50,100):
        print(f"    at +${tgt}/day: CI={(1.96*day_pnl.std(ddof=1)/tgt)**2:.0f}  80% power={((1.96+0.8416)*day_pnl.std(ddof=1)/tgt)**2:.0f}")
    # ICC
    grand = pnl.mean()
    ssb = sum(len(days[k])*(np.mean(days[k])-grand)**2 for k in keys); ssw = sum(sum((x-np.mean(days[k]))**2 for x in days[k]) for k in keys)
    msb = ssb/(D-1); msw = ssw/(n-D); n0 = (n - sum(len(days[k])**2 for k in keys)/n)/(D-1)
    icc = max((msb-msw)/(msb+(n0-1)*msw), 0.0); deff = 1 + (n0-1)*icc
    print(f"  ICC={icc:.3f} design effect={deff:.2f} effective n(trades)={n/deff:.1f} effective DAYS={D}")
    # MC — trade level iid + block, day level
    def iid_paths(h,b=B): return rng.choice(pnl, size=(b,h), replace=True)
    def block_paths(h,b=B,mb=5):
        out = np.empty((b,h)); p = 1.0/mb
        for i in range(b):
            seq=[]; idx = rng.integers(n)
            while len(seq)<h:
                seq.append(pnl[idx % n])
                idx = rng.integers(n) if rng.random()<p else idx+1
            out[i]=seq[:h]
        return out
    def day_paths(h,b=B):
        out = np.empty((b,h))
        for i in range(b):
            seq=[]
            while len(seq)<h: seq.extend(days[keys[rng.integers(D)]])
            out[i]=seq[:h]
        return out
    print("  MAX DRAWDOWN ($, 1 lot):")
    print(f"    {'h':>4} {'method':>8} {'p50':>7} {'p90':>7} {'p95':>7} {'p99':>7} {'worst':>7}")
    ddstore = {}
    for h in (20,50,100):
        for nm, P in (("iid", iid_paths(h)), ("block5", block_paths(h)), ("day", day_paths(h))):
            dd = np.array([maxdd(p) for p in P]); ddstore[(h,nm)] = dd
            print(f"    {h:>4} {nm:>8} {np.percentile(dd,50):>7.0f} {np.percentile(dd,90):>7.0f} {np.percentile(dd,95):>7.0f} {np.percentile(dd,99):>7.0f} {dd.max():>7.0f}")
    for h in (53,65):
        dd = np.array([maxdd(p) for p in day_paths(h)]); ddstore[(h,"day")] = dd
        print(f"    {h:>4} {'day':>8} {np.percentile(dd,50):>7.0f} {np.percentile(dd,90):>7.0f} {np.percentile(dd,95):>7.0f} {np.percentile(dd,99):>7.0f} {dd.max():>7.0f}")
    print("  MAX DRAWDOWN over N session-DAYS:")
    for nd in (10,20,40):
        dd = np.array([maxdd(rng.choice(day_pnl, size=nd, replace=True)) for _ in range(B)])
        print(f"    {nd:>3}d p50={np.percentile(dd,50):.0f} p90={np.percentile(dd,90):.0f} p95={np.percentile(dd,95):.0f} p99={np.percentile(dd,99):.0f} worst={dd.max():.0f}")
    # streaks
    P50 = day_paths(50); s = np.array([max_streak(p) for p in P50])
    print("  P(losing streak>=k in 50, day-resampled): " + " ".join(f"k{k}={float((s>=k).mean()):.3f}" for k in (4,6,8,10)))
    P100 = day_paths(100); s8 = np.array([max_streak(p) for p in P100])
    # expected number of 8-streak STARTS per 100 trades
    def count_fires(p, k=8):
        cur=0; fires=0
        for x in p:
            if x<0:
                cur+=1
                if cur==k: fires+=1
            else: cur=0
        return fires
    f8 = np.array([count_fires(p,8) for p in P100]); f10 = np.array([count_fires(p,10) for p in P100])
    print(f"  8-loss breaker: E[fires per 100 trades]={f8.mean():.3f} P(>=1 in 100)={float((f8>=1).mean()):.3f} | 10-loss E={f10.mean():.3f}")
    # realised path
    rdd, pk_i, tr_i = dd_span(pnl)
    eq = np.cumsum(pnl)
    print(f"  REALISED: final={eq[-1]:+.2f} peak={eq.max():+.2f} at #{int(eq.argmax())+1} maxDD={rdd:.2f} "
          f"span trades #{pk_i}->#{tr_i} ({tr_i-pk_i} trades) losing-streak={max_streak(pnl)} nonwin-streak={max_nonwin_streak(pnl)}")
    span = tr_i - pk_i
    for h,nm in ((20,"day"),(50,"day"),(53,"day"),(65,"day")):
        dd = ddstore[(h,nm)]
        print(f"    realised {rdd:.0f} sits at percentile {float((dd<rdd).mean())*100:.1f} of the {h}-trade day-resampled DD distribution (p50={np.percentile(dd,50):.0f})")
    print(f"  REALISED by day: red-day streak={max_streak(day_pnl)} maxDD(days)={maxdd(day_pnl):.2f}")
    # kill-switch rolling window trip rates (research finding)
    def rolling_dd_max(p, w=20):
        best=0.0
        for i in range(len(p)-w+1):
            best = max(best, maxdd(p[i:i+w]))
        return best
    def min_roll_wr(p, w=30):
        best=1.0
        for i in range(len(p)-w+1):
            seg = p[i:i+w]; wr = float((seg>0).sum())/w
            best = min(best, wr)
        return best
    SUB = 2000
    rmax = np.array([rolling_dd_max(P100[i],20) for i in range(SUB)])
    rwr  = np.array([min_roll_wr(P100[i],30) for i in range(SUB)])
    print(f"  KILL-SWITCH calibration (B={SUB} day-resampled 100-trade runs):")
    for thr in (1236, 1400, 1574, 1800):
        print(f"    P(any rolling-20 DD > ${thr}) = {float((rmax>thr).mean()):.3f}")
    print(f"    rolling-20 DD max over 100 trades: p50={np.percentile(rmax,50):.0f} p90={np.percentile(rmax,90):.0f} p95={np.percentile(rmax,95):.0f} p99={np.percentile(rmax,99):.0f}")
    for thr in (0.229, 0.20, 0.167, 0.133):
        print(f"    P(any rolling-30 win rate < {thr:.3f}) = {float((rwr<thr).mean()):.3f}")
    print(f"    min rolling-30 win rate: p50={np.percentile(rwr,50):.3f} p05={np.percentile(rwr,5):.3f} p01={np.percentile(rwr,1):.3f}")
    both = ((rmax>1236) | (rwr<0.229)); print(f"    P(DD-leg OR WR-leg trips as WRITTEN in the report) = {float(both.mean()):.3f}")
    f8s = np.array([count_fires(P100[i],8)>=1 for i in range(SUB)])
    print(f"    P(any of the three legs incl. 8-streak) = {float((both|f8s).mean()):.3f}")
    # weekly loss probability by day-bootstrap of 5-day sums
    w5 = np.array([rng.choice(day_pnl, size=5, replace=True).sum() for _ in range(B)])
    w4 = np.array([rng.choice(day_pnl, size=4, replace=True).sum() for _ in range(B)])
    print(f"  WEEK: P(5-day sum <= -900)={float((w5<=-900).mean()):.3f}  P(4-day<=-900)={float((w4<=-900).mean()):.3f}  5-day p05={np.percentile(w5,5):.0f}")
    print()
