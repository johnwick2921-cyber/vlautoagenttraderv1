#!/usr/bin/env python3
"""r04 — daily/weekly limit calibration, trade-cap counterfactual, per-bucket CIs,
long/short by day direction, on BOTH populations."""
import csv, math, sqlite3, datetime, zoneinfo
from collections import defaultdict
import numpy as np
ct = zoneinfo.ZoneInfo("America/Chicago")
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; c=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (c-h,c+h)
def tci(v):
    v=np.array(v,dtype=float); n=len(v)
    if n<2: return (float('nan'),float('nan'),float('nan'))
    se=v.std(ddof=1)/math.sqrt(n)
    tc={1:12.706,2:4.303,3:3.182,4:2.776,5:2.571,6:2.447,7:2.365,8:2.306,9:2.262,10:2.228,
        11:2.201,12:2.179,13:2.160,14:2.145,15:2.131,16:2.120,17:2.110,19:2.093,20:2.086,
        21:2.080,22:2.074,23:2.069,25:2.060,29:2.045,31:2.040,37:2.026,41:2.020}.get(n-1,2.0)
    return (v.mean()-tc*se, v.mean()+tc*se, v.mean()/se if se else float('nan'))

for LABEL, FN in (("COMPLIANT n=58", "trade_sample_58.csv"), ("BROAD n=65", "trade_sample_65.csv")):
    rows=list(csv.DictReader(open(FN)))
    for r in rows: r['pnl']=float(r['pnl_corrected'])
    days=defaultdict(list)
    for r in rows: days[r['session_day_ct']].append(r)
    keys=sorted(days)
    print("="*80); print(LABEL)
    # within-day running minimum + daily limit trips
    print("  per-day: n, total, running-min (intraday worst realised running sum)")
    runmins={}
    for k in keys:
        seq=[r['pnl'] for r in days[k]]; cum=np.cumsum(seq); runmins[k]=float(cum.min())
        print(f"    {k}: n={len(seq):>2} tot={sum(seq):+8.2f} runmin={cum.min():+8.2f}")
    print("  DAILY-LIMIT calibration (trip = running realised sum <= -L at any close in the day):")
    for L in (225,300,450,600,900):
        trips=[k for k in keys if runmins[k]<=-L]
        # forfeited = P&L of trades after the first trip within the day
        forf=0.0
        for k in trips:
            seq=[r['pnl'] for r in days[k]]; cum=np.cumsum(seq)
            i=int(np.argmax(cum<=-L)); forf+=sum(seq[i+1:])
        print(f"    ${L:>4}: trips {len(trips)}/{len(keys)} days {trips} forfeited P&L after trip = {forf:+.2f}")
    # 2-lot equivalent: limit L applies to 2x P&L  => per-lot threshold L/2
    print("  at 2 LOTS the $450 limit is $225/lot: " +
          f"{sum(1 for k in keys if runmins[k]<=-225)}/{len(keys)} days trip")
    # trade cap 3/day
    forf=0.0; hit=0
    for k in keys:
        seq=[r['pnl'] for r in days[k]]
        if len(seq)>3: hit+=1; forf+=sum(seq[3:])
    print(f"  MAX_DAILY_TRADES=3: exceeds on {hit}/{len(keys)} days ({hit/len(keys)*100:.1f}%); forfeited after 3rd = {forf:+.2f} "
          f"({forf/len(keys):+.2f}/day); kept mean/day = {(sum(r['pnl'] for r in rows)-forf)/len(keys):+.2f}")
    # weekly, two conventions
    for conv in ("ISO","SUNDAY17"):
        wk=defaultdict(float)
        for k in keys:
            d=datetime.date.fromisoformat(k)
            if conv=="ISO": key=f"{d.isocalendar()[0]}-W{d.isocalendar()[1]:02d}"
            else:
                # session-day k is labelled by its 17:00 CT open; the week opens Sunday 17:00 CT
                # => week key = the Sunday on/before k
                key=(d - datetime.timedelta(days=(d.weekday()+1)%7)).isoformat()
            wk[key]+=sum(r['pnl'] for r in days[k])
        ser=[(k,wk[k]) for k in sorted(wk)]
        runmin=[]; 
        for k in sorted(wk):
            dd=[kk for kk in keys if ((datetime.date.fromisoformat(kk)-datetime.timedelta(days=(datetime.date.fromisoformat(kk).weekday()+1)%7)).isoformat()==k) or (conv=="ISO" and f"{datetime.date.fromisoformat(kk).isocalendar()[0]}-W{datetime.date.fromisoformat(kk).isocalendar()[1]:02d}"==k)]
            cum=np.cumsum([sum(r['pnl'] for r in days[x]) for x in sorted(dd)])
            runmin.append((k, float(cum.min()) if len(cum) else 0.0))
        print(f"  WEEKS ({conv}): " + " · ".join(f"{k} {v:+.2f} (runmin {dict(runmin)[k]:+.2f})" for k,v in ser))
    # buckets
    print("  BUCKETS (mean, 95% t-CI, p(win) Wilson with both denominators):")
    for field in ("session","condition"):
        g=defaultdict(list)
        for r in rows: g[r[field] or "(empty)"].append(r['pnl'])
        for k in sorted(g):
            v=g[k]; lo,hi,t=tci(v); w=sum(1 for x in v if x>0); l=sum(1 for x in v if x<0)
            wl,wh=wilson(w,w+l); al,ah=wilson(w,len(v))
            print(f"    {field}={k:<22} n={len(v):>2} mean={np.mean(v):+8.2f} CI[{lo:+8.2f},{hi:+8.2f}] t={t:+.2f} "
                  f"p(win) {w}/{w+l}={w/(w+l) if w+l else 0:.3f}[{wl:.3f},{wh:.3f}] {w}/{len(v)}=[{al:.3f},{ah:.3f}]")
    g=defaultdict(list)
    for r in rows: g[r['side'] or "?"].append(r['pnl'])
    for k in sorted(g):
        v=g[k]; lo,hi,t=tci(v); w=sum(1 for x in v if x>0); l=sum(1 for x in v if x<0)
        wl,wh=wilson(w,w+l)
        print(f"    side={k:<22} n={len(v):>2} mean={np.mean(v):+8.2f} CI[{lo:+8.2f},{hi:+8.2f}] t={t:+.2f} p(win) {w}/{w+l} [{wl:.3f},{wh:.3f}]")
    print()

# long/short by 1d bar direction of the session-day
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
bars={}
for sym,ot,o,c in con.execute("SELECT symbol,open_time_ms,o,c FROM bars WHERE symbol='MNQ' AND tf='1d'"):
    d=(datetime.datetime.fromtimestamp(ot/1000, ct)).date().isoformat()
    bars[d]=(o,c)
for LABEL,FN in (("COMPLIANT n=58","trade_sample_58.csv"),):
    rows=list(csv.DictReader(open(FN)))
    g=defaultdict(list)
    for r in rows:
        d=r['session_day_ct']; b=bars.get(d)
        if not b: continue
        direc = "UP" if b[1]>b[0] else "DOWN"
        g[(r['side'],direc)].append(float(r['pnl_corrected']))
    print("LONG/SHORT by 1d bar direction of the session-day (", LABEL, "):")
    for k in sorted(g):
        v=g[k]; w=sum(1 for x in v if x>0)
        print(f"    {k[0]:<6} on {k[1]:<4} days: n={len(v):>2} mean={np.mean(v):+8.2f} sum={sum(v):+8.2f} wins={w}")
