#!/usr/bin/env python3
"""r09 — kill-switch leg calibration with EXACT integer win counts (no float thresholds)."""
import csv, math
from collections import defaultdict
import numpy as np
SEED,B=20260905,5000
rng=np.random.default_rng(SEED)
rows=list(csv.DictReader(open('trade_sample_58.csv')))
days=defaultdict(list)
for r in rows: days[r['session_day_ct']].append(float(r['pnl_corrected']))
keys=sorted(days); D=len(keys)
def maxdd(p):
    eq=np.cumsum(p); pk=np.maximum.accumulate(np.concatenate(([0.0],eq)))[1:]; return float(np.max(pk-eq))
def path(h):
    seq=[]
    while len(seq)<h: seq.extend(days[keys[rng.integers(D)]])
    return np.array(seq[:h])
P=[path(100) for _ in range(B)]
def roll_dd(p,w=20): return max(maxdd(p[i:i+w]) for i in range(len(p)-w+1))
def min_wins30(p,w=30): return min(int((p[i:i+w]>0).sum()) for i in range(len(p)-w+1))
def fires8(p,k=8):
    cur=f=0
    for x in p:
        if x<0:
            cur+=1
            if cur==k: f+=1
        else: cur=0
    return f
rd=np.array([roll_dd(p) for p in P]); mw=np.array([min_wins30(p) for p in P]); f8=np.array([fires8(p) for p in P])
print(f"B={B} day-resampled 100-trade runs, compliant n=58")
print("rolling-20 max DD over 100 trades: " + " ".join(f"p{q}={np.percentile(rd,q):.0f}" for q in (50,90,95,99)))
for t in (1236,1400,1500,1600,1700):
    print(f"  P(any rolling-20 DD > ${t}) = {float((rd>t).mean()):.4f}")
print("min WINS in any rolling 30 (integer):")
for k in range(0,11):
    print(f"  P(some rolling-30 window has <= {k} wins of 30) = {float((mw<=k).mean()):.4f}"
          + ("   <- report's 0.229 bound = <=6 wins" if k==6 else ""))
print(f"  distribution of min-wins-of-30: p01={np.percentile(mw,1):.0f} p05={np.percentile(mw,5):.0f} p50={np.percentile(mw,50):.0f}")
print(f"8-streak: E[fires per 100]={f8.mean():.3f}  P(>=1)={float((f8>=1).mean()):.4f}")
asw=((rd>1236)|(mw<=6)); print(f"AS WRITTEN P(DD>1236 or <=6 wins of 30) = {float(asw.mean()):.4f}; with 8-streak leg {float((asw|(f8>=1)).mean()):.4f}")
for dt,wt in ((1600,3),(1700,3),(1600,4)):
    c=((rd>dt)|(mw<=wt)); print(f"RECAL P(DD>${dt} or <={wt} wins of 30) = {float(c.mean()):.4f}  (DD leg alone {float((rd>dt).mean()):.4f}, WR leg alone {float((mw<=wt).mean()):.4f})")
