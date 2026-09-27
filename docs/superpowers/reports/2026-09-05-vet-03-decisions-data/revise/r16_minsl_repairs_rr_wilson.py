import sqlite3, json, math, csv, statistics as st, collections
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
c=sqlite3.connect(DB, uri=True)
def wilson(k,n,z=1.96):
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d; return (ctr-h, ctr+h)
def W(lbl,k,n):
    lo,hi=wilson(k,n); print(f"  {lbl}: {k}/{n} = {100*k/n:.1f}% [{100*lo:.1f}%, {100*hi:.1f}%]")
print("### B1 min-SL validateDecision refusals (audit refusals.csv) — was the same cycle's final decision a taken open_*?")
rows=list(csv.DictReader(open('/home/hoang/nofx-vet-03/docs/superpowers/reports/2026-09-04-two-day-audit-data/refusals.csv')))
ms=[r for r in rows if r['leg'].startswith('min_sl (validateDecision')]
print("  min_sl validateDecision rows:", len(ms))
taken=[]; notaken=[]; nocycle=[]
for r in ms:
    cyc=r['decision_cycle'].strip()
    if not cyc: nocycle.append(r['ts_ct']); continue
    d=c.execute("SELECT id, datetime(timestamp,'-5 hours'), execution_log FROM decision_records WHERE cycle_number=? ORDER BY id",(int(cyc),)).fetchall()
    ok=[x for x in d if x[2] and ('open_long succeeded' in x[2] or 'open_short succeeded' in x[2])]
    if ok: taken.append((r['ts_ct'], cyc, ok[0][0], ok[0][1], r['cf_usd']))
    else: notaken.append((r['ts_ct'], cyc, r['cf_usd'], (d[0][2][:80] if d and d[0][2] else 'no record')))
print("  taken in same cycle:", len(taken)); [print("    ",t) for t in taken]
print("  NOT taken:", len(notaken)); [print("    ",t) for t in notaken]
print("  no cycle id:", len(nocycle), nocycle)
print("  cf sum of taken rows:", sum(float(t[4]) for t in taken), "; cf sum of not-taken:", sum(float(t[2]) for t in notaken))
# which positions correspond to the taken cycles
for t in taken:
    p=c.execute("SELECT id, entry_price, exit_price, pnl_corrected, upper(side), datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE abs(entry_time - (strftime('%s',?)+5*3600)*1000) < 180000",(t[3],)).fetchall()
    print("    position for cycle",t[1],":",p)

print("### B2 like-for-like planned R:R — decision path (executor's own SL/TP), compliant system rows")
pos=c.execute("SELECT id, entry_time, upper(side), entry_price, pnl_corrected, mfe FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND plan_id<>'UNRESOLVABLE' AND source='system' AND pnl_corrected IS NOT NULL ORDER BY entry_time").fetchall()
decs=c.execute("SELECT id, strftime('%s',timestamp)*1000, decision_json FROM decision_records WHERE decision_json LIKE '%open_%' AND date(timestamp,'-5 hours')>='2026-08-19'").fetchall()
rr=[]; reached=0; nn=0
for pid,et,side,ep,pnl,mfe in pos:
    best=None
    for did,dtm,dj in decs:
        if abs(dtm-et)<=180000:
            try: d=json.loads(dj)
            except: continue
            items=d if isinstance(d,list) else [d]
            for x in items:
                if isinstance(x,dict) and str(x.get('action','')).endswith(side.lower()) and x.get('stop_loss') and x.get('take_profit'):
                    sl=float(x['stop_loss']); tp=float(x['take_profit'])
                    risk=abs(ep-sl); rew=abs(tp-ep)
                    if risk>0 and (best is None or abs(dtm-et)<best[0]): best=(abs(dtm-et), rew/risk, rew, did)
    if best:
        rr.append(best[1]); nn+=1
        if mfe is not None and mfe>=best[2]: reached+=1
print(f"  n={nn} planned R:R median={st.median(rr):.2f} mean={st.mean(rr):.2f} min={min(rr):.2f} max={max(rr):.2f}; MFE reached own target: {reached}/{nn}")
bins=collections.Counter('<2.0' if r<2 else '[2.0,2.1)' if r<2.1 else '[2.1,2.3)' if r<2.3 else '[2.3,3.0)' if r<3 else '>=3.0' for r in rr); print("  bins:", dict(bins))
print("  decision-path realized: avgW/avgL from r10: 151.54/72.10 = 2.10 (compliant n=47); broad 140.88/71.72 = 1.96 (n=51)")
print("  armed_fill 9: avgW", st.mean([63,17,32.5,92,106]), "avgL", st.mean([42,32,54,140]), "payoff", st.mean([63,17,32.5,92,106])/st.mean([42,32,54,140]))
armsf=c.execute("SELECT id, entry_px, stop_px, target_px FROM armed_orders WHERE state='filled' AND session NOT LIKE 'TEST%'").fetchall()
arr=[abs(t-e)/abs(e-s) for _,e,s,t in armsf if abs(e-s)>0]; print("  filled arms planned R:R:", [round(x,2) for x in arr], "median", round(st.median(arr),2), "n", len(arr))

print("### B3 missing Wilson intervals")
W("short arm-enabled all versions",96,395); W("long arm-enabled all versions",36,343); W("arms within 0.3 of floor",47,132); W("open intents refused",19,31); W("plan levels matching seated",54,105)
for k,lbl in ((6,'transport'),(5,'fade_requires_touch'),(5,'entry_mode=pullback'),(5,'arm legs'),(7,'confirm-rule vocab incl not-allowed'),(2,'gap-up'),(2,'retest+levelcap'),(32,'continuation stated')): W(lbl,k,64)
W("decision-path win compliant",12,45); W("armed lineage win compliant",6,11); W("LONG win compliant",4,19); W("SHORT win compliant",14,37)
W("13 of 13 strict",13,13); W("MFE>=2R compliant",13,54); W("MFE>=1R compliant",21,54)
W("S1 win",10,27); W("S2-S4 win",8,29); W("ASIA win",2,15); W("system long win compliant",2,16)
