import sqlite3, datetime as dt, collections, json
c=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
def ct(s):
    s=str(s)
    if s.endswith('+00:00'): return (dt.datetime.strptime(s[:19].replace('T',' '),'%Y-%m-%d %H:%M:%S')-dt.timedelta(hours=5))
    return dt.datetime.strptime(s[:19].replace('T',' '),'%Y-%m-%d %H:%M:%S')
rows=c.execute("SELECT id, placement_seq, state, state_reason, signal_id, created_at, updated_at, stop_px, target_px FROM armed_orders WHERE id BETWEEN 38 AND 102 AND abs(entry_px-29591.02)<0.01 ORDER BY id").fetchall()
print("id seq state reason sig created_ct updated_ct stop target")
for r in rows: print(f"  {r[0]:3d} {r[1]:2d} {r[2]:9s} {r[3][:28]:28s} sig={'Y' if r[4] else '-'}({str(r[4])[:8]}) {ct(r[5]).strftime('%H:%M:%S')} -> {ct(r[6]).strftime('%H:%M:%S')} stop={r[7]} tgt={r[8]}")
print("with signal id:", sum(1 for r in rows if r[4]), "distinct signal ids:", len(set(r[4] for r in rows if r[4])))
# overlap: how many rows were live (created <= t < updated) at each creation time
live_at_create=[]
for r in rows:
    t=ct(r[5]); n=sum(1 for q in rows if q[0]!=r[0] and ct(q[5])<=t<ct(q[6]))
    live_at_create.append((r[0],n))
print("rows already live (per ledger created..updated) when each was created:", live_at_create)
print("max simultaneous per ledger:", max(n for _,n in live_at_create)+1)
# broker snapshots 10:00-11:00
snaps=c.execute("SELECT id, datetime(emitted_at_ms/1000,'unixepoch','-5 hours'), working_count, order_count, orders_json FROM nt8_order_snapshots WHERE emitted_at_ms BETWEEN strftime('%s','2026-09-04 15:00:00')*1000 AND strftime('%s','2026-09-04 16:00:00')*1000 ORDER BY emitted_at_ms").fetchall()
print("snapshots 10:00-11:00 CT:", len(snaps), "with working>0:", sum(1 for s in snaps if s[2]>0), "max working:", max(s[2] for s in snaps))
wc=collections.Counter(s[2] for s in snaps); print("  working_count histogram:", sorted(wc.items()))
# per 5-min bucket max working
b=collections.defaultdict(int)
for s in snaps: b[s[1][11:15]+'x']=max(b[s[1][11:15]+'x'], s[2])
print("  max working per 10-min:", dict(sorted(b.items())))
# distinct broker order ids at 29591.02 across the hour
ids=set(); ex=None
for s in snaps:
    try: oj=json.loads(s[4])
    except: continue
    items=oj if isinstance(oj,list) else oj.get('orders',[])
    for o in items:
        if isinstance(o,dict):
            px=[v for k,v in o.items() if isinstance(v,(int,float)) and abs(v-29591.02)<0.05]
            if px:
                ids.add(o.get('order_id') or o.get('id') or o.get('signal_id') or o.get('name') or json.dumps(o)[:40]); ex=ex or o
print("  distinct broker order records at 29591.02 during 10:00-11:00:", len(ids)); print("  example order record keys:", list(ex.keys()) if ex else None)
# S1 29657.38 rows count (the regression commit's 24 rows)
s1=c.execute("SELECT COUNT(*), MIN(id), MAX(id), SUM(signal_id<>''), GROUP_CONCAT(DISTINCT state_reason) FROM armed_orders WHERE abs(entry_px-29657.38)<0.01").fetchone(); print("S1 @29657.38 rows:", s1)
