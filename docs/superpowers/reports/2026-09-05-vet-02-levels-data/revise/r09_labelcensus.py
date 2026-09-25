import sqlite3, json, collections, re
db=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); db.row_factory=sqlite3.Row
rows=list(db.execute("SELECT plan_id, version, trade_date, session, doc FROM plans WHERE session<>'WEEKLY'"))
sess=set(); cnt=collections.Counter(); sessions=collections.defaultdict(set)
for r in rows:
    try: j=json.loads(r['doc'])
    except Exception: continue
    sess.add((r['trade_date'], r['session']))
    for lv in (j.get('levels') or []):
        lab=str(lv.get('label') or '')
        base=lab.split('·')[0].strip()
        cnt[base]+=1
        sessions[base].add((r['trade_date'], r['session']))
print("non-WEEKLY plan-sessions:", len(sess), " plan rows:", len(rows))
for k in ['VWAP+2σ','VWAP-2σ','VWAP±2σ','eVWAP','VWAP','SETT','MID-O','VAH','VAL','GAP','POC','RTH-L','RTH-H','PDL','PDH','PDC','ONH','ONL','EQH','EQL','OB','SWG-H','SWG-L','PWH','PWL']:
    if k in cnt: print(f"  {k}: rows={cnt[k]} plan-sessions={len(sessions[k])}")
two=[k for k in cnt if '2σ' in k]
print("2σ labels:", {k:(cnt[k],len(sessions[k])) for k in two}, " union plan-sessions:", len(set().union(*[sessions[k] for k in two])) if two else 0)
