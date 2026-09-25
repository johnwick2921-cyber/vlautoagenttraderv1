import sqlite3, datetime, bisect, collections, statistics, json
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
CT=datetime.timezone(datetime.timedelta(hours=-5))
def ms(y,m,d,h,mi): return int(datetime.datetime(y,m,d,h,mi,tzinfo=CT).timestamp()*1000)
print('## (ss) 09-04 08:55–10:05 CT 5-min OHLC from 1m bars (MNQ)')
rows=con.execute("SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms BETWEEN ? AND ? ORDER BY open_time_ms",(ms(2026,9,4,8,55),ms(2026,9,4,10,5))).fetchall()
g=collections.OrderedDict()
for t,o,h,l,c in rows:
    k=datetime.datetime.fromtimestamp(t/1000,CT).strftime('%H:%M')[:4]+'x'
    if k not in g: g[k]=[o,h,l,c]
    else: g[k][1]=max(g[k][1],h); g[k][2]=min(g[k][2],l); g[k][3]=c
for k,v in g.items(): print('  ',k, v)
print('## (tt) LONDON 09-04 v1 S2 long reject arm 29579.5 / stop 29558.34 / target 29639.5 — refused by leg 3 at 02:00:46; cf 02:00→08:30')
bars=con.execute("SELECT open_time_ms,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall(); T=[b[0] for b in bars]
def cf(side, entry, stop, target, t0, t1):
    i0=bisect.bisect_left(T,t0); i1=bisect.bisect_right(T,t1); seg=bars[i0:i1]; touched=None
    for b in seg:
        if touched is None:
            if b[2]<=entry<=b[1]: touched=b[0]
            else: continue
        if side=='short':
            if b[1]>=stop: return datetime.datetime.fromtimestamp(touched/1000,CT).strftime('%H:%M'),'STOP',datetime.datetime.fromtimestamp(b[0]/1000,CT).strftime('%H:%M'),-(stop-entry)
            if b[2]<=target: return datetime.datetime.fromtimestamp(touched/1000,CT).strftime('%H:%M'),'TARGET',datetime.datetime.fromtimestamp(b[0]/1000,CT).strftime('%H:%M'),entry-target
        else:
            if b[2]<=stop: return datetime.datetime.fromtimestamp(touched/1000,CT).strftime('%H:%M'),'STOP',datetime.datetime.fromtimestamp(b[0]/1000,CT).strftime('%H:%M'),-(entry-stop)
            if b[1]>=target: return datetime.datetime.fromtimestamp(touched/1000,CT).strftime('%H:%M'),'TARGET',datetime.datetime.fromtimestamp(b[0]/1000,CT).strftime('%H:%M'),target-entry
    return (datetime.datetime.fromtimestamp(touched/1000,CT).strftime('%H:%M') if touched else None),'FLAT/NEVER',None,0
print('  ', cf('long',29579.5,29558.34,29639.5,ms(2026,9,4,2,0),ms(2026,9,4,8,30)))
print('   NY S1 recheck: ', cf('short',29657.38,29720.5,29503.38,ms(2026,9,4,9,2),ms(2026,9,4,14,45)))
print('## (uu) AI close actions since 08-19 vs close_reason')
n=0; byday=collections.Counter()
for (dj,ct) in con.execute("SELECT decision_json, date(timestamp,'-5 hours') FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-08-19' AND decision_json LIKE '%close_%'").fetchall():
    try: d=json.loads(dj)
    except Exception: continue
    items=d if isinstance(d,list) else d.get('decisions',[d]) if isinstance(d,dict) else []
    for x in items:
        if isinstance(x,dict) and str(x.get('action','')).startswith('close_'): n+=1; byday[ct]+=1
print('  close_* decisions since 08-19:', n, dict(byday))
print('  close_reason of era positions:', con.execute("SELECT close_reason, COUNT(*) FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' GROUP BY 1").fetchall())
print('## (vv) era holding minutes (pnl_corrected not null, excl test seam)')
d=[ (r[0]-r[1])/60000 for r in con.execute("SELECT exit_time, entry_time, pnl_corrected FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL AND exit_time>entry_time").fetchall()]
print('  n',len(d),'median',round(statistics.median(d),1),'p25',round(sorted(d)[len(d)//4],1),'p75',round(sorted(d)[3*len(d)//4],1),'max',round(max(d),1))
