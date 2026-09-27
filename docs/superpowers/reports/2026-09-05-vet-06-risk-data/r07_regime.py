#!/usr/bin/env python3
"""r07 — regime history + ATR5m, re-measured."""
import sqlite3, datetime, zoneinfo, math
import numpy as np
ct=zoneinfo.ZoneInfo("America/Chicago")
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
d=con.execute("SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1d' ORDER BY open_time_ms").fetchall()
print("1d bars:", len(d), "first", datetime.datetime.fromtimestamp(d[0][0]/1000,ct).date(), "last", datetime.datetime.fromtimestamp(d[-1][0]/1000,ct).date())
rows=[]
prevc=None
trs=[]
for ot,o,h,l,c in d:
    day=datetime.datetime.fromtimestamp(ot/1000,ct).date().isoformat()
    tr = (h-l) if prevc is None else max(h-l, abs(h-prevc), abs(l-prevc))
    trs.append(tr); prevc=c
    atr14 = float(np.mean(trs[-14:])) if len(trs)>=14 else None
    rows.append(dict(day=day,o=o,h=h,l=l,c=c,rng=h-l,rngp=100*(h-l)/c,atr14=atr14,atrp=(100*atr14/c if atr14 else None)))
worst=max(rows,key=lambda r:r['rngp'])
print(f"WORST day by range%: {worst['day']} rng={worst['rng']:.2f} pts = {worst['rngp']:.2f}% (o={worst['o']} h={worst['h']} l={worst['l']} c={worst['c']})")
mx=max(rows,key=lambda r:r['rng']); print(f"WORST day by range pts: {mx['day']} {mx['rng']:.2f} pts = {mx['rngp']:.2f}%")
for chk in ('2026-04-08','2025-04-08'):
    r=[x for x in rows if x['day']==chk]
    print(f"  {chk}: " + (f"rng={r[0]['rng']:.2f} = {r[0]['rngp']:.2f}%" if r else "no bar"))
# sample days
sample=['2026-08-18','2026-08-19','2026-08-20','2026-08-23','2026-08-24','2026-08-25','2026-08-26','2026-08-27','2026-08-30','2026-08-31','2026-09-01','2026-09-02']
sm=[r for r in rows if r['day'] in sample]
print(f"SAMPLE (12 session-days): n={len(sm)}")
print(f"  range pts {min(r['rng'] for r in sm):.0f}-{max(r['rng'] for r in sm):.0f}  range% {min(r['rngp'] for r in sm):.2f}-{max(r['rngp'] for r in sm):.2f}")
print(f"  ATR14 pts {min(r['atr14'] for r in sm):.0f}-{max(r['atr14'] for r in sm):.0f} median {np.median([r['atr14'] for r in sm]):.0f}")
atrp=[r['atrp'] for r in sm]; print(f"  ATR14% {min(atrp):.3f}-{max(atrp):.3f} median {np.median(atrp):.3f}")
# 09-03 partial check
r3=[r for r in rows if r['day']=='2026-09-03']
if r3: print(f"  2026-09-03 (NOT a sample day, no trade): rng={r3[0]['rng']:.2f} = {r3[0]['rngp']:.2f}%  ATR14%={r3[0]['atrp']:.3f}")
n5=con.execute("SELECT COUNT(*) FROM bars WHERE symbol='MNQ' AND tf='5m' AND open_time_ms>=? AND open_time_ms<?",
    (int(datetime.datetime(2026,9,3,17,0,tzinfo=ct).timestamp()*1000), int(datetime.datetime(2026,9,4,16,0,tzinfo=ct).timestamp()*1000))).fetchone()[0]
print(f"  5m bars in session-day 2026-09-03: {n5} (a full CME session-day is 276)")
# history shares, using only COMPLETE days (exclude the last bar if partial)
hist=[r for r in rows if r['atrp'] is not None]
maxatr=max(atrp); maxrng=max(r['rngp'] for r in sm); medatr=float(np.median(atrp))
above_atr=[r for r in hist if r['atrp']>maxatr]; above_rng=[r for r in hist if r['rngp']>maxrng]
thr=2*medatr; above2=[r for r in hist if r['atrp']>thr]
print(f"HISTORY: {len(hist)} days with ATR14")
print(f"  days with ATR14% above the sample MAX ({maxatr:.3f}%): {len(above_atr)} = {100*len(above_atr)/len(hist):.1f}%")
print(f"  days with range% above the sample's widest ({maxrng:.2f}%): {len(above_rng)} = {100*len(above_rng)/len(hist):.1f}%")
print(f"  2x sample median ATR% = {thr:.2f}%: {len(above2)} days")
from collections import Counter
print("    by year:", dict(sorted(Counter(r['day'][:4] for r in above2).items())))
print("    last such day:", max(r['day'] for r in above2))
# by-year medians
byy={}
for r in hist: byy.setdefault(r['day'][:4],[]).append(r)
print("  by-year: year n medianRange maxRange% medianATR")
for y in sorted(byy):
    v=byy[y]; print(f"    {y} n={len(v):>4} medRng={np.median([x['rng'] for x in v]):>6.0f} maxRng%={max(x['rngp'] for x in v):>5.2f} medATR14={np.median([x['atr14'] for x in v]):>6.0f}")
# ATR5m RTH
b5=con.execute("SELECT open_time_ms,h,l,c FROM bars WHERE symbol='MNQ' AND tf='5m' ORDER BY open_time_ms").fetchall()
print(f"5m bars: {len(b5)}")
trs=[]; prevc=None; rth=[]
for ot,h,l,c in b5:
    tr=(h-l) if prevc is None else max(h-l,abs(h-prevc),abs(l-prevc)); trs.append(tr); prevc=c
    if len(trs)>=14:
        t=datetime.datetime.fromtimestamp(ot/1000,ct)
        if (t.hour>8 or (t.hour==8 and t.minute>=30)) and t.hour<15: rth.append(float(np.mean(trs[-14:])))
if rth:
    print(f"  RTH ATR5m(14): n={len(rth)} p50={np.percentile(rth,50):.2f} p90={np.percentile(rth,90):.2f} max={max(rth):.2f}")
    print(f"    1.5x floor: p50 -> {1.5*np.percentile(rth,50):.1f} pts (${1.5*np.percentile(rth,50)*2:.0f}), p90 -> {1.5*np.percentile(rth,90):.1f} pts (${1.5*np.percentile(rth,90)*2:.0f})")
    print(f"    share of RTH bars where 1.5xATR5m >= 75 pts (a $150 cap would refuse every arm): "
          f"{100*float(np.mean(np.array(rth)>=50)):.1f}%")
