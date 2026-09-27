# q13: (gg) 1m bar coverage per hour 09-03/09-04; (ii) leg-3 refusal counterfactuals; (mm) broker book during 09-04 churn; (qq) plan levels vs seated candidates; (rr) planned arm R:R bins
import sqlite3, json, datetime, collections, bisect, math
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
CT=datetime.timezone(datetime.timedelta(hours=-5))
print('## (gg) 1m MNQ bars per CT hour, 09-03 and 09-04')
for d in ('2026-09-03','2026-09-04'):
    rows=con.execute("SELECT strftime('%H', open_time_ms/1000,'unixepoch','-5 hours') h, COUNT(*) FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours')=? GROUP BY 1 ORDER BY 1",(d,)).fetchall()
    print(d, ' '.join(f'{h}:{n}' for h,n in rows))
    mx=con.execute("SELECT datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours') FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours')=?",(d,)).fetchone()[0]
    print('   last 1m bar', mx)
bars=con.execute("SELECT open_time_ms,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall(); T=[b[0] for b in bars]
def cf(side, entry, stop, target, t0, t1):
    i0=bisect.bisect_left(T,t0); i1=bisect.bisect_right(T,t1); seg=bars[i0:i1]
    touched=None; out='NEVER_FILLED'
    for b in seg:
        if touched is None:
            if b[2]<=entry<=b[1]: touched=b[0]
            else: continue
        if side=='short':
            if b[1]>=stop: return touched,'STOP',b[0],-(stop-entry)
            if b[2]<=target: return touched,'TARGET',b[0],entry-target
        else:
            if b[2]<=stop: return touched,'STOP',b[0],-(entry-stop)
            if b[1]>=target: return touched,'TARGET',b[0],target-entry
    if touched is not None:
        last=seg[-1][3] if seg else entry
        return touched,'FLAT_AT_HORIZON',None,(entry-last if side=='short' else last-entry)
    return None,'NEVER_FILLED',None,0.0
print('## (ii) leg-3 (invalidation) refusal counterfactuals on 09-04')
# NY S1 v3: short reject 29657.38 / 29720.5 / 29503.38 (arm id 39), refused 09:02:09 CT; horizon NY flat 14:45 (bars end 12:19)
t0=int(datetime.datetime(2026,9,4,9,2,tzinfo=CT).timestamp()*1000); t1=int(datetime.datetime(2026,9,4,14,45,tzinfo=CT).timestamp()*1000)
print(' NY S1 short @29657.38 stop 29720.5 tgt 29503.38 →', cf('short',29657.38,29720.5,29503.38,t0,t1), '(bars end 12:19 CT — horizon truncated)')
# LONDON v2 S2 — read its doc
r=con.execute("SELECT version, doc FROM plans WHERE trade_date='2026-09-04' AND session='LONDON' ORDER BY version").fetchall()
for v,doc in r:
    d=json.loads(doc)
    for s in d.get('scenarios',[]):
        if s.get('id')=='S2':
            a=s.get('arm') or {}
            print(f' LONDON v{v} S2 {s.get("condition")} {s.get("direction")} confirm={s.get("confirm")} arm={a}')
            if v==2 and a.get('entry'):
                t0=int(datetime.datetime(2026,9,4,2,0,tzinfo=CT).timestamp()*1000); t1=int(datetime.datetime(2026,9,4,8,30,tzinfo=CT).timestamp()*1000)
                print('   cf →', cf(s.get('direction'),float(a['entry']),float(a['stop']),float(a['target']),t0,t1))
print('## (mm) nt8_order_snapshots 09-04 10:00–11:00 CT: snapshots, with working orders')
rows=con.execute("SELECT COUNT(*), SUM(working_count>0), MAX(working_count), MIN(datetime(emitted_at_ms/1000,'unixepoch','-5 hours')), MAX(datetime(emitted_at_ms/1000,'unixepoch','-5 hours')) FROM nt8_order_snapshots WHERE emitted_at_ms BETWEEN ? AND ?",(int(datetime.datetime(2026,9,4,10,0,tzinfo=CT).timestamp()*1000),int(datetime.datetime(2026,9,4,11,0,tzinfo=CT).timestamp()*1000))).fetchone()
print('  snapshots', rows)
ex=con.execute("SELECT datetime(emitted_at_ms/1000,'unixepoch','-5 hours'), working_count, substr(orders_json,1,160) FROM nt8_order_snapshots WHERE emitted_at_ms BETWEEN ? AND ? AND working_count>0 ORDER BY emitted_at_ms LIMIT 3",(int(datetime.datetime(2026,9,4,10,0,tzinfo=CT).timestamp()*1000),int(datetime.datetime(2026,9,4,11,0,tzinfo=CT).timestamp()*1000))).fetchall()
for e in ex: print('  ', e)
print('## (qq) plan levels vs seated candidate_pool (same plan_id+version), since 09-01')
plans=con.execute("SELECT plan_id, version, doc FROM plans WHERE trade_date>='2026-09-01' AND session IN ('ASIA','LONDON','NY')").fetchall()
tot=match=0; nplans=0
for pid,ver,doc in plans:
    seated=[x[0] for x in con.execute("SELECT level_price FROM candidate_pool WHERE plan_id=? AND plan_version=? AND seated=1",(pid,ver)).fetchall()]
    if not seated: continue
    nplans+=1
    for l in json.loads(doc).get('levels',[]):
        tot+=1
        if any(abs(float(l['price'])-float(s))<=1.0 for s in seated): match+=1
print(f'  plans with a seated pool: {nplans}; plan levels {tot}; matching a seated candidate within 1pt: {match} = {match/tot:.3f}' if tot else '  no pool rows')
print('## (rr) planned arm R:R bins (all versions, arm-enabled, n=132)')
rows=con.execute("SELECT doc FROM plans WHERE session IN ('ASIA','LONDON','NY')").fetchall()
rr=[]
for (doc,) in rows:
    for s in json.loads(doc).get('scenarios',[]):
        a=s.get('arm') or {}
        if a.get('enabled') and a.get('entry') and a.get('stop') and a.get('target'):
            e,st,t=float(a['entry']),float(a['stop']),float(a['target'])
            if s.get('direction')=='long' and e>st: rr.append((t-e)/(e-st))
            elif s.get('direction')=='short' and st>e: rr.append((e-t)/(st-e))
bins=collections.Counter()
for x in rr:
    b='<2.0' if x<2 else '[2.0,2.1)' if x<2.1 else '[2.1,2.3)' if x<2.3 else '[2.3,3.0)' if x<3 else '>=3.0'
    bins[b]+=1
print('  n', len(rr), dict(bins), 'share in [2.0,2.3):', round(sum(1 for x in rr if 2<=x<2.3)/len(rr),3))
