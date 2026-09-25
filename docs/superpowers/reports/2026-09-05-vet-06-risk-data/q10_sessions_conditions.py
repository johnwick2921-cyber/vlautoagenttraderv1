#!/usr/bin/env python3
"""q10 — per-session / per-condition P&L from trade_sample.csv with Wilson (z=1.96) on win rate and t-CI on mean.
Also: the rows with an EMPTY condition (the API's by=condition view drops them; this sample keeps them)."""
import csv, math, statistics as st
from collections import defaultdict
rows = list(csv.DictReader(open('trade_sample.csv')))
def wilson(k, n, z=1.96):
    if n == 0: return (0.0, 0.0)
    p = k/n; d = 1 + z*z/n; c = p + z*z/(2*n); h = z*math.sqrt(p*(1-p)/n + z*z/(4*n*n))
    return ((c-h)/d, (c+h)/d)
def summ(label, xs, ids):
    n=len(xs); w=sum(1 for x in xs if x>0); l=sum(1 for x in xs if x<0); f=n-w-l
    mu=st.mean(xs); sd=st.stdev(xs) if n>1 else 0.0; se=sd/math.sqrt(n) if n>1 else 0.0
    lo,hi=wilson(w, w+l) if w+l>0 else (0,0)
    print(f"{label:>22}: n={n:2d} W/L/F={w}/{l}/{f} p(win)={w/max(w+l,1):.3f} Wilson[{lo:.3f},{hi:.3f}] sum={sum(xs):+8.2f} mean={mu:+7.2f} sd={sd:6.2f} 95%CI[{mu-1.96*se:+7.2f},{mu+1.96*se:+7.2f}] t={mu/se if se else 0:+.2f} ids={ids}")
bys=defaultdict(list); byc=defaultdict(list); ids_s=defaultdict(list); ids_c=defaultdict(list)
for r in rows:
    x=float(r['pnl_corrected']); s=r['session'] or 'NULL'; c=r['condition'] or 'EMPTY'
    bys[s].append(x); ids_s[s].append(r['id']); byc[c].append(x); ids_c[c].append(r['id'])
print("== by plan_session ==")
for s in sorted(bys): summ(s, bys[s], ids_s[s])
print("== by cited_scenario_id (S1..S5 = scenario slot, not condition name) ==")
for c in sorted(byc): summ(c, byc[c], ids_c[c])
print("== rows with EMPTY condition ==")
for r in rows:
    if not r['condition']: print("  ", r['id'], r['session'] or 'NULL', r['source'], r['pnl_corrected'], r['created_ct'])
summ("ALL", [float(r['pnl_corrected']) for r in rows], "521..591")
# long vs short
for side in ('LONG','SHORT'):
    xs=[float(r['pnl_corrected']) for r in rows if r['side'].upper().startswith(side[:4]) or r['side'].upper()==side]
    ids=[r['id'] for r in rows if r['side'].upper().startswith(side[:4])]
    if xs: summ(side, xs, f"n={len(ids)}")
