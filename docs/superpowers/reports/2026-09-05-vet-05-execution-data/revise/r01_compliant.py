"""Independent compliant re-derivation. READ-ONLY."""
import sqlite3,math,json,datetime,csv
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON')
ERA=1786770000000; END=1788584400000
def ct(ms): return datetime.datetime.fromtimestamp(ms/1000,datetime.timezone(datetime.timedelta(hours=-5))).strftime('%Y-%m-%d %H:%M:%S')
print("ERA bound CT:",ct(ERA)," END bound CT:",ct(END))
def rate(k,n):
    if not n: return (k,n,None,None)
    p=k/n; z=1.96; d=1+z*z/n; m=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (k,n,round(100*p,2),[round(100*max(0,m-h),2),round(100*min(1,m+h),2)])
def pct(a):
    a=sorted(a); out={}
    for p in (.5,.8,.95):
        i=(len(a)-1)*p; j=int(i); out[p]=round(a[j]+(a[min(j+1,len(a)-1)]-a[j])*(i-j),3)
    return out
base="FROM trader_positions WHERE entry_time>=? AND entry_time<? AND pnl_corrected IS NOT NULL AND coalesce(source,'')<>'e7_farside_test'"
# REPORT's executed filter (close_reason + note only)
rep=base+" AND lower(coalesce(close_reason,'')) NOT IN ('unresolved','unresolvable','reconcile_flat','e7_farside_test') AND upper(coalesce(pnl_correction_note,'')) NOT LIKE '%UNRESOLVABLE%'"
# COMPLIANT filter (adds plan_id)
comp=rep+" AND coalesce(plan_id,'')<>'UNRESOLVABLE'"
for label,sql in (("REPORT P65",rep),("COMPLIANT P58",comp)):
    ps=[dict(r) for r in c.execute("SELECT * "+sql+" ORDER BY id",(ERA,END))]
    ids=[r['id'] for r in ps]
    win=[r for r in ps if r['pnl_corrected']>0]; lose=[r for r in ps if r['pnl_corrected']<0]; scr=[r for r in ps if r['pnl_corrected']==0]
    aw=sum(r['pnl_corrected'] for r in win)/len(win); al=sum(r['pnl_corrected'] for r in lose)/len(lose)
    print("\n==== %s ===="%label)
    print(" n=%d  W=%d L=%d scratch=%d"%(len(ps),len(win),len(lose),len(scr)))
    print(" sum=%.4f  mean=%.6f  avgWin=%.4f  avgLoss=%.4f  payoff=%.4f"%(sum(r['pnl_corrected'] for r in ps),sum(r['pnl_corrected'] for r in ps)/len(ps),aw,al,aw/abs(al)))
    print(" win rate ex-scratch:",rate(len(win),len(win)+len(lose)))
    print(" ids:",ids)
    print(" winner ids:",[r['id'] for r in win]); print(" scratch ids:",[r['id'] for r in scr])
    for lab,g in (("winners",win),("losers",lose)):
        for k in ('mae','mfe'):
            v=[r[k] for r in g if r[k] is not None]
            print("  %s %s n=%d  p50/p80/p95=%s  (missing ids=%s)"%(lab,k,len(v),pct(v) if v else None,[r['id'] for r in g if r[k] is None]))
    # zero-valued extrema sentinels
    print("  mae==0 rows:",[(r['id'],r['source'],r['pnl_corrected']) for r in ps if r['mae']==0])
    # CME session days (roll 17:00 CT) and CT calendar dates
    days=set(); cal=set()
    for r in ps:
        d=datetime.datetime.fromtimestamp(r['entry_time']/1000,datetime.timezone(datetime.timedelta(hours=-5)))
        cal.add(d.date())
        days.add((d+datetime.timedelta(hours=7)).date())  # 17:00 CT roll -> add 7h
    print("  CME session-days=%d  CT calendar dates=%d"%(len(days),len(cal)))
c.close()
