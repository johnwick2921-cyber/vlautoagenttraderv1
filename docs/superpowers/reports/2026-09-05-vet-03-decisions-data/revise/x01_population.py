import sqlite3, math, statistics as st
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
c=sqlite3.connect(DB,uri=True); c.row_factory=sqlite3.Row
ERA=1786770000000  # 2026-08-15 00:00 CT
rows=[dict(r) for r in c.execute(
 "SELECT id,source,plan_id,pnl_corrected,cited_scenario_id,plan_session,plan_band,side,close_reason,entry_time,exit_time,mae,mfe,entry_price,exit_price "
 "FROM trader_positions WHERE entry_time>=? ORDER BY id",(ERA,))]
print("era rows (entry_time>=2026-08-15 00:00 CT):",len(rows))
seam=[r['id'] for r in rows if r['source']=='e7_farside_test']
nullp=[r['id'] for r in rows if r['pnl_corrected'] is None]
unres=[r['id'] for r in rows if (r['plan_id'] or '')=='UNRESOLVABLE']
noplan=[r['id'] for r in rows if not (r['plan_id'] or '')]
print("e7 seam:",seam)
print("pnl_corrected NULL (excluded, COUNT shown):",len(nullp),nullp)
print("plan_id=UNRESOLVABLE:",len(unres),unres)
print("plan_id empty/NULL:",len(noplan),noplan)
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d
    hw=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (100*(ctr-hw),100*(ctr+hw))
def stats(name,rs):
    v=[r['pnl_corrected'] for r in rs]; n=len(v)
    if n==0: print(f"{name}: n=0"); return
    s=sum(v); m=s/n
    sd=st.stdev(v) if n>1 else 0.0; se=sd/math.sqrt(n) if n else 0
    W=[x for x in v if x>0]; L=[x for x in v if x<0]; F=[x for x in v if x==0]
    aw=sum(W)/len(W) if W else 0; al=sum(L)/len(L) if L else 0
    payoff=(aw/abs(al)) if L else float('nan')
    lo,hi=wilson(len(W),len(W)+len(L))
    be=100/(1+payoff) if L and payoff==payoff else float('nan')
    print(f"{name}: n={n} sum={s:+.2f} mean={m:+.2f} sd={sd:.2f} se={se:.2f} t={(m/se if se else 0):+.2f} "
          f"W/L/F={len(W)}/{len(L)}/{len(F)} win={len(W)}/{len(W)+len(L)}={100*len(W)/max(1,len(W)+len(L)):.1f}% [{lo:.1f}%,{hi:.1f}%] "
          f"avgW={aw:.2f} avgL={al:.2f} payoff={payoff:.2f} BE={be:.1f}%")
    print("   ids:",[r['id'] for r in rs])
base=[r for r in rows if r['source']!='e7_farside_test' and r['pnl_corrected'] is not None]
broad=[r for r in base]
comp=[r for r in base if (r['plan_id'] or '')!='UNRESOLVABLE']
print("\n===== BROAD (test-seam + NULL excluded only)")
stats("all",broad)
print("\n===== COMPLIANT (also excludes plan_id=UNRESOLVABLE)")
stats("all",comp)
for nm,f in [("decision path source=system",lambda r:r['source']=='system'),
             ("armed_fill band",lambda r:r['source']=='reconcile' and r['plan_band']=='armed_fill'),
             ("armed lineage any",lambda r:r['source'] in ('reconcile','armed_entry'))]:
    stats("  "+nm,[r for r in comp if f(r)])
for s in ['S1','S2','S3','S4']:
    stats("  slot "+s,[r for r in comp if r['cited_scenario_id']==s])
stats("  S2-S4",[r for r in comp if r['cited_scenario_id'] in ('S2','S3','S4')])
for s in ['ASIA','LONDON','NY']:
    stats("  session "+s,[r for r in comp if r['plan_session']==s])
stats("  LONDON+NY",[r for r in comp if r['plan_session'] in ('LONDON','NY')])
stats("  LONG",[r for r in comp if (r['side'] or '').upper()=='LONG'])
stats("  SHORT",[r for r in comp if (r['side'] or '').upper()=='SHORT'])
# session-days
import datetime as dt
def sd(ms):
    t=dt.datetime.utcfromtimestamp(ms/1000)-dt.timedelta(hours=5)
    return (t-dt.timedelta(hours=17)).date()
print("  session-days:",sorted({sd(r['entry_time']) for r in comp}))
hold=sorted(((r['exit_time']-r['entry_time'])/60000) for r in comp if r['exit_time'])
print(f"  holding min n={len(hold)} median={st.median(hold):.1f} p25={hold[len(hold)//4]:.1f} p75={hold[3*len(hold)//4]:.1f} max={max(hold):.1f}")
print("  mae/mfe non-null in compliant:",sum(1 for r in comp if r['mae'] is not None and r['mfe'] is not None))
print("  UNRESOLVABLE sum:",sum(r['pnl_corrected'] for r in base if (r['plan_id'] or '')=='UNRESOLVABLE'))
