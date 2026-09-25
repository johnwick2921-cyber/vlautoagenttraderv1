#!/usr/bin/env python3
"""q14 — weekly P&L (ISO week of session-day), within-day peak-to-trough (realised intraday drawdown),
and how a $450 daily / weekly limit would have interacted with the realised sequences."""
import csv, datetime, statistics as st
from collections import defaultdict
rows = list(csv.DictReader(open('trade_sample.csv')))
days = defaultdict(list)
for r in rows: days[r['session_day_ct']].append(float(r['pnl_corrected']))
keys = sorted(days)
print("== per session-day: seq, sum, running-min (worst point in the day), peak-to-trough within day ==")
worst_pts=[]; p2t=[]
for k in keys:
    seq=days[k]; run=0; mn=0; pk=0; dd=0
    for x in seq:
        run+=x; mn=min(mn,run); pk=max(pk,run); dd=max(dd,pk-run)
    worst_pts.append(mn); p2t.append(dd)
    print(f"  {k}: n={len(seq):2d} sum={sum(seq):+8.2f} worst_running={mn:+8.2f} peak_to_trough={dd:8.2f}")
print(f"worst running point within a day: min={min(worst_pts):+.2f} median={st.median(worst_pts):+.2f}; days below -450: {sum(1 for w in worst_pts if w<=-450)}/{len(keys)}; below -300: {sum(1 for w in worst_pts if w<=-300)}")
print(f"peak-to-trough within day: max={max(p2t):.2f} median={st.median(p2t):.2f}")
print("\n== weekly (ISO week of session-day) ==")
wk=defaultdict(list); wkdays=defaultdict(set)
for k in keys:
    d=datetime.date.fromisoformat(k); y,w,_=d.isocalendar(); wk[(y,w)].extend(days[k]); wkdays[(y,w)].add(k)
for (y,w) in sorted(wk):
    seq=wk[(y,w)]; run=0; mn=0
    for x in seq: run+=x; mn=min(mn,run)
    print(f"  {y}-W{w:02d}: days={len(wkdays[(y,w)])} n={len(seq):2d} sum={sum(seq):+8.2f} worst_running={mn:+8.2f}")
print("\n== daily-limit calibration: with a $450 realized trip, which trades would NOT have happened? ==")
for L in (300, 450, 600):
    saved=0; trips=0
    for k in keys:
        run=0; tripped=False
        for x in days[k]:
            if tripped: saved+=x; continue
            run+=x
            if run<=-L: tripped=True; trips+=1
    print(f"  limit ${L}: trips on {trips}/{len(keys)} days; P&L of trades after the trip (forfeited, + means we would have lost that profit): {saved:+.2f}")
