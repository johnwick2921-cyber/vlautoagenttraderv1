import sqlite3, csv, re, math, datetime
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); c=con.cursor(); c.execute('PRAGMA query_only=ON')
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (round(100*(ctr-h),1), round(100*(ctr+h),1))
def pct(xs,q):
    xs=sorted(xs);
    if not xs: return None
    i=(len(xs)-1)*q; lo=int(i); hi=min(lo+1,len(xs)-1); return round(xs[lo]+(xs[hi]-xs[lo])*(i-lo),2)
era=1786770000000
rows=c.execute("""SELECT id, side, entry_price, exit_price, entry_time, exit_time, pnl_corrected, mae, mfe, source, close_reason, plan_band, cited_scenario_id FROM trader_positions
 WHERE entry_time>=? AND entry_time<1788584400000 AND upper(trim(coalesce(plan_id,'')))<>'UNRESOLVABLE' AND pnl_corrected IS NOT NULL AND source<>'e7_farside_test' AND close_reason NOT IN ('reconcile_flat','unresolved','e7_farside_test') ORDER BY id""",(era,)).fetchall()
excluded=c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time>=? AND entry_time<1788584400000 AND upper(trim(coalesce(plan_id,'')))<>'UNRESOLVABLE' AND (pnl_corrected IS NULL OR source='e7_farside_test' OR close_reason IN ('reconcile_flat','unresolved','e7_farside_test'))",(era,)).fetchone()[0]
# exit reasons from log lines
reasons=[]
for line in open("log_nt_closes.out"):
    m=re.search(r'^(\d\d-\d\d \d\d:\d\d:\d\d).*MNQ (LONG|SHORT) qty=[\d.]+ exit=([\d.]+) reason=(\w+) pnl=(-?[\d.]+)', line)
    if m: reasons.append((m.group(1),m.group(2),float(m.group(3)),m.group(4),float(m.group(5))))
def find_reason(side, exitpx, exit_ms, pnl):
    best=None
    for t,s,ex,rs,pn in reasons:
        if s!=side.upper() or abs(ex-exitpx)>0.01: continue
        tm=datetime.datetime.strptime("2026-"+t,"%Y-%m-%d %H:%M:%S").replace(tzinfo=datetime.timezone(datetime.timedelta(hours=-5)))
        dt=abs(tm.timestamp()*1000-exit_ms)
        if dt<5*60000 and (best is None or dt<best[0]): best=(dt,rs)
    return best[1] if best else 'unknown'
out=[]
for pid,side,ent,ex,et,xt,pnl,mae,mfe,src,cr,band,scen in rows:
 out.append(dict(id=pid,reason=find_reason(side,ex,xt,pnl),side=side,exit_price=ex,exit_time=xt,provenance='nearest log side/exit-price within 5 minutes; not execution-id proof'))
with open('q14_mae_mfe.csv','w') as f:
 w=csv.DictWriter(f,fieldnames=list(out[0])); w.writeheader();w.writerows(out)
print('Eligible exit-label audit',len(out),'no initial-risk or ATR inference')
