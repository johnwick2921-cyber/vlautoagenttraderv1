#!/usr/bin/env python3
"""q11 — MNQ 1d regime history (bars tf='1d', 2019-05..2026-09): daily range, ATR(14), percentiles;
where the 12 sample session-days sit; the share of history in vol regimes the sample never saw."""
import sqlite3, datetime, zoneinfo, statistics as st, math
ct = zoneinfo.ZoneInfo("America/Chicago")
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
rows = con.execute("SELECT open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf='1d' ORDER BY open_time_ms").fetchall()
print("1d rows:", len(rows), "first", datetime.datetime.fromtimestamp(rows[0][0]/1000, ct).date(), "last", datetime.datetime.fromtimestamp(rows[-1][0]/1000, ct).date())
# gaps check
ds=[datetime.datetime.fromtimestamp(r[0]/1000, ct).date() for r in rows]
gaps=[(ds[i-1],ds[i],(ds[i]-ds[i-1]).days) for i in range(1,len(ds)) if (ds[i]-ds[i-1]).days>5]
print("gaps >5 days:", gaps[:10], "count", len(gaps))
def q(a,p):
    a=sorted(a); i=(len(a)-1)*p; lo=math.floor(i); hi=math.ceil(i); return a[lo]+(a[hi]-a[lo])*(i-lo)
prev=None; tr=[]; atr=[]; rng=[]; rngpct=[]; dates=[]
for ms,o,h,l,c,v in rows:
    t = h-l if prev is None else max(h-l, abs(h-prev), abs(l-prev))
    tr.append(t); rng.append(h-l); rngpct.append(100*(h-l)/c); dates.append(datetime.datetime.fromtimestamp(ms/1000, ct).date())
    atr.append(sum(tr[-14:])/min(len(tr),14)); prev=c
print(f"daily RANGE pts (all): n={len(rng)} p10={q(rng,.1):.0f} p50={q(rng,.5):.0f} p90={q(rng,.9):.0f} p99={q(rng,.99):.0f} max={max(rng):.0f} ({dates[rng.index(max(rng))]})")
print(f"daily RANGE % of close (all): p10={q(rngpct,.1):.2f} p50={q(rngpct,.5):.2f} p90={q(rngpct,.9):.2f} p99={q(rngpct,.99):.2f} max={max(rngpct):.2f}")
print(f"ATR14 daily pts (all): p10={q(atr,.1):.0f} p50={q(atr,.5):.0f} p90={q(atr,.9):.0f} p99={q(atr,.99):.0f} max={max(atr):.0f}")
# by year
print("--- by year: n, median range pts, median range %, max range %, median ATR14 ---")
years={}
for d,r,rp,a in zip(dates,rng,rngpct,atr): years.setdefault(d.year,[]).append((r,rp,a))
for y in sorted(years):
    v=years[y]; print(f"{y}: n={len(v):3d} range p50={st.median([x[0] for x in v]):5.0f} p90={q([x[0] for x in v],.9):5.0f}  range% p50={st.median([x[1] for x in v]):.2f} max={max(x[1] for x in v):.2f}  ATR14 p50={st.median([x[2] for x in v]):5.0f}")
# the sample window
smp = [i for i,d in enumerate(dates) if datetime.date(2026,8,18) <= d <= datetime.date(2026,9,3)]
srng=[rng[i] for i in smp]; satr=[atr[i] for i in smp]; srp=[rngpct[i] for i in smp]
print(f"\nSAMPLE WINDOW 2026-08-18..09-03: n={len(smp)} days  range p50={st.median(srng):.0f} min={min(srng):.0f} max={max(srng):.0f}  range% p50={st.median(srp):.2f}  ATR14 p50={st.median(satr):.0f} min={min(satr):.0f} max={max(satr):.0f}")
lo,hi=min(srp),max(srp)
inside=sum(1 for x in rngpct if lo<=x<=hi); above=sum(1 for x in rngpct if x>hi); below=sum(1 for x in rngpct if x<lo)
print(f"history days with range% inside the sample's [{lo:.2f},{hi:.2f}]: {inside}/{len(rngpct)} = {100*inside/len(rngpct):.1f}%  ABOVE (never seen): {above} = {100*above/len(rngpct):.1f}%  below: {below}")
alo,ahi=min(satr),max(satr)
# ATR relative to price
atrp=[100*a/c for a,(ms,o,h,l,c,v) in zip(atr,rows)]; salo,sahi=min(atrp[i] for i in smp),max(atrp[i] for i in smp)
ab=sum(1 for x in atrp if x>sahi)
print(f"ATR14 as % of price: sample [{salo:.2f},{sahi:.2f}]  history days ABOVE the sample max: {ab}/{len(atrp)} = {100*ab/len(atrp):.1f}%  (p99 of history = {q(atrp,.99):.2f}%, max {max(atrp):.2f}% on {dates[atrp.index(max(atrp))]})")
# episodes above 2x sample-median ATR%
thr=2*st.median(atrp[i] for i in smp); ep=[dates[i] for i,x in enumerate(atrp) if x>thr]
print(f"days with ATR14% > 2x sample median ({thr:.2f}%): {len(ep)}  first/last: {ep[0] if ep else None} .. {ep[-1] if ep else None}")
yrs={}
for d in ep: yrs[d.year]=yrs.get(d.year,0)+1
print("  by year:", yrs)
# last 30 rows for the current level
print("\n--- last 15 days: date, range, range%, ATR14 ---")
for i in range(len(rows)-15,len(rows)): print(f"  {dates[i]} range={rng[i]:6.0f} {rngpct[i]:.2f}% ATR14={atr[i]:.0f}")
