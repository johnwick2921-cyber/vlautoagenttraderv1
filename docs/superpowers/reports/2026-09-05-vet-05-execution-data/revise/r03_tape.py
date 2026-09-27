"""Tape checks: penetration distribution, RTH bucket profile. READ-ONLY."""
import sqlite3,math,datetime
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON')
def pct(a,ps=(.5,.8,.9,.95)):
    a=sorted(a); out={}
    for p in ps:
        i=(len(a)-1)*p; j=int(i); out[p]=round(a[j]+(a[min(j+1,len(a)-1)]-a[j])*(i-j),3)
    return out
def wil(k,n):
    p=k/n; z=1.96; d=1+z*z/n; m=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (k,n,round(100*p,2),[round(100*max(0,m-h),2),round(100*min(1,m+h),2)])
print("=== touch_episodes penetration_pts, session_day >= 2026-08-15 ===")
v=[r[0] for r in c.execute("SELECT penetration_pts FROM touch_episodes WHERE session_day>='2026-08-15' AND penetration_pts IS NOT NULL")]
print(" n=%d  p50/p80/p90/p95 = %s"%(len(v),pct(v)))
for cap in (0.25,0.5,1.0,2.0,2.5,5.0):
    print("   within %.2f pts: %s"%(cap,wil(sum(1 for x in v if x<=cap),len(v))))
print(" all-time n=%d p50/p80/p95=%s"%(len([1 for _ in c.execute("SELECT 1 FROM touch_episodes")]),
      pct([r[0] for r in c.execute("SELECT penetration_pts FROM touch_episodes WHERE penetration_pts IS NOT NULL")])))
print("\n=== wick vs body penetration (era) ===")
for col in ('wick_pen_pts','body_pen_pts'):
    w=[r[0] for r in c.execute("SELECT %s FROM touch_episodes WHERE session_day>='2026-08-15' AND %s IS NOT NULL"%(col,col))]
    print("  %s n=%d p50/p80/p95=%s"%(col,len(w),pct(w) if w else None))
print("\n=== held-touch join (touch_episodes x touch_outcomes, +/-0.01 px, +/-10min) ===")
q="""SELECT te.penetration_pts p, tou.outcome o FROM touch_episodes te JOIN touch_outcomes tou
 ON abs(te.level_price-tou.level_price)<0.01 AND abs(te.opened_at_ms-tou.opened_at_ms)<600000
 WHERE te.session_day>='2026-08-15' AND te.penetration_pts IS NOT NULL"""
rr=[dict(r) for r in c.execute(q)]
print("  joined n=%d"%len(rr))
for o in sorted(set(r['o'] for r in rr)):
    g=[r['p'] for r in rr if r['o']==o]
    print("   %-18s n=%d p50/p80=%s within1.0: %s"%(o,len(g),pct(g,(.5,.8)),wil(sum(1 for x in g if x<=1.0),len(g))))
print("\n=== RTH 10-min volume buckets, MNQ 1m, Mon-Fri, >=2026-08-19 ===")
rows=[dict(r) for r in c.execute("""SELECT open_time_ms,v,h,l FROM bars WHERE symbol='MNQ' AND tf='1m'
   AND open_time_ms >= strftime('%s','2026-08-19')*1000""")]
buck={}
for r in rows:
    d=datetime.datetime.fromtimestamp(r['open_time_ms']/1000,datetime.timezone(datetime.timedelta(hours=-5)))
    if d.weekday()>4: continue
    key="%02d:%02d"%(d.hour,(d.minute//10)*10)
    buck.setdefault(key,[]).append((r['v'],r['h']-r['l']))
rth=sorted(k for k in buck if '08:30'<=k<='14:59')
stats=[(k,len(buck[k]),sum(x[0] for x in buck[k])/len(buck[k]),sum(x[1] for x in buck[k])/len(buck[k])) for k in rth]
med=sorted(s[2] for s in stats)[len(stats)//2]
print("  RTH buckets=%d  median avg-vol=%.1f"%(len(stats),med))
rank={k:i+1 for i,(k,_,_,_) in enumerate(sorted(stats,key=lambda s:-s[2]))}
for k in ('08:30','11:50','12:30','13:20','14:10','14:20','14:30','14:40','14:50'):
    s=[x for x in stats if x[0]==k]
    if s: k,n,av,ar=s[0]; print("   %s n=%3d avgvol=%9.2f avgrange=%6.4f  rank %d/%d  x median=%.2f"%(k,n,av,ar,rank[k],len(stats),av/med))
c.close()
