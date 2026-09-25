import sqlite3, math, statistics as st
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
c=sqlite3.connect(DB, uri=True)
rows=c.execute("""SELECT id, source, plan_id, cited_scenario_id, plan_session, plan_band, pnl_corrected, side,
  datetime(entry_time/1000,'unixepoch','-5 hours') ct, close_reason, mae, mfe, entry_price, exit_price, exit_time
  FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' ORDER BY id""").fetchall()
print("plan-linked rows:", len(rows))
e7=[r for r in rows if r[1]=='e7_farside_test']
print("e7 seam:", [r[0] for r in e7])
nul=[r for r in rows if r[1]!='e7_farside_test' and r[6] is None]
print("NULL pnl_corrected (excluded, count shown):", [r[0] for r in nul])
unres=[r for r in rows if r[2]=='UNRESOLVABLE']
print("plan_id=UNRESOLVABLE:", [(r[0],r[1],r[6],r[3],r[4]) for r in unres], "sum", sum(r[6] or 0 for r in unres))
broad=[r for r in rows if r[1]!='e7_farside_test' and r[6] is not None]
comp=[r for r in broad if r[2]!='UNRESOLVABLE']
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (ctr-h, ctr+h)
def summ(name, rs):
    n=len(rs); s=sum(r[6] for r in rs)
    W=[r[6] for r in rs if r[6]>0]; L=[r[6] for r in rs if r[6]<0]; F=[r for r in rs if r[6]==0]
    mean=s/n if n else 0
    sd=st.stdev([r[6] for r in rs]) if n>1 else 0
    se=sd/math.sqrt(n) if n else 0
    t=mean/se if se else 0
    aw=sum(W)/len(W) if W else 0; al=sum(L)/len(L) if L else 0
    dec=len(W)+len(L)
    lo,hi=wilson(len(W),dec) if dec else (0,0)
    payoff=aw/abs(al) if al else 0
    be=1/(1+payoff) if payoff else 0
    print(f"{name}: n={n} sum={s:.2f} mean={mean:.2f} sd={sd:.2f} se={se:.2f} t={t:.2f} W/L/F={len(W)}/{len(L)}/{len(F)} win={len(W)}/{dec}={100*len(W)/dec if dec else 0:.1f}% [{100*lo:.1f}%,{100*hi:.1f}%] avgW={aw:.2f} avgL={al:.2f} payoff={payoff:.2f} BE={100*be:.1f}%")
    print("   ids:", [r[0] for r in rs])
for label,pop in (("BROAD65",broad),("COMPLIANT58",comp)):
    print("=====",label)
    summ("all",pop)
    summ("decision path (source=system)",[r for r in pop if r[1]=='system'])
    summ("armed lineage (reconcile|armed_entry)",[r for r in pop if r[1] in ('reconcile','armed_entry')])
    for slot in ('S1','S2','S3','S4'):
        summ(f"slot {slot}",[r for r in pop if (r[3] or '')==slot])
    summ("slot other/uncited",[r for r in pop if (r[3] or '') not in ('S1','S2','S3','S4')])
    print("  cited distinct:", sorted(set((r[3] or 'NULL') for r in pop)))
    summ("S2-S4",[r for r in pop if (r[3] or '') in ('S2','S3','S4')])
    for sess in ('ASIA','LONDON','NY',None):
        summ(f"session {sess}",[r for r in pop if r[4]==sess])
    summ("LONDON+NY",[r for r in pop if r[4] in ('LONDON','NY')])
    summ("long",[r for r in pop if r[7]=='long'])
    summ("short",[r for r in pop if r[7]=='short'])
    summ("S1 system",[r for r in pop if (r[3] or '')=='S1' and r[1]=='system'])
    summ("S1 armed",[r for r in pop if (r[3] or '')=='S1' and r[1]!='system'])
    summ("S2 armed",[r for r in pop if (r[3] or '')=='S2' and r[1]!='system'])
    summ("S2-S4 armed",[r for r in pop if (r[3] or '') in ('S2','S3','S4') and r[1]!='system'])
    summ("S1 system on/before 08-20",[r for r in pop if (r[3] or '')=='S1' and r[1]=='system' and r[8]<'2026-08-21'])
    summ("S1 system after 08-20",[r for r in pop if (r[3] or '')=='S1' and r[1]=='system' and r[8]>='2026-08-21'])
    summ("S1 any after 08-20",[r for r in pop if (r[3] or '')=='S1' and r[8]>='2026-08-21'])
    summ("ASIA system",[r for r in pop if r[4]=='ASIA' and r[1]=='system'])
    summ("ASIA armed",[r for r in pop if r[4]=='ASIA' and r[1]!='system'])
    summ("long system",[r for r in pop if r[7]=='long' and r[1]=='system'])
    summ("long armed",[r for r in pop if r[7]=='long' and r[1]!='system'])
    summ("short armed",[r for r in pop if r[7]=='short' and r[1]!='system'])
    # session-days at 17:00 CT
    days=set()
    for r in pop:
        # r[8] is CT datetime; session-day rolls at 17:00
        import datetime as dt
        d=dt.datetime.strptime(r[8],'%Y-%m-%d %H:%M:%S')
        if d.hour>=17: d=d+dt.timedelta(days=1)
        days.add(d.date())
    print("  session-days (17:00 CT roll):", len(days), sorted(days))
    # holding minutes
    hm=[]
    for r in pop:
        ent=c.execute("SELECT entry_time, exit_time FROM trader_positions WHERE id=?",(r[0],)).fetchone()
        if ent[0] and ent[1]:
            hm.append((ent[1]-ent[0])/60000)
    hm.sort()
    def pct(a,p):
        k=(len(a)-1)*p; f=math.floor(k); cc=math.ceil(k)
        return a[f] if f==cc else a[f]+(a[cc]-a[f])*(k-f)
    print(f"  holding min n={len(hm)} median={pct(hm,.5):.1f} p25={pct(hm,.25):.1f} p75={pct(hm,.75):.1f} max={max(hm):.1f}")
    # mae/mfe populated
    print("  mae/mfe non-null:", sum(1 for r in pop if r[10] is not None and r[11] is not None))
# also: which of the 65 broad are UNRESOLVABLE and their slot/session
print("UNRESOLVABLE rows detail:")
for r in unres: print("  ",r)
# entry_time typeof & era count checks
print(c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time >= 1786770000000").fetchone(), "rows entry_time >= 2026-08-15 00:00 CT")
print(c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-09 05:00:00')*1000").fetchone(), "rows since 08-09 00:00 CT")
print(c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-09 00:00:00')*1000").fetchone(), "rows since 08-09 00:00 UTC")
print(c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-01 05:00:00')*1000").fetchone(), "rows since 08-01 CT")
