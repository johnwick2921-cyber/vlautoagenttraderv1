import sqlite3, json, math, collections
db=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True); db.row_factory=sqlite3.Row
ERA=1786856400000  # 2026-08-15 00:00 CT
plans={}
for r in db.execute("SELECT plan_id, version, doc FROM plans"):
    plans[(r['plan_id'], r['version'])]=r['doc']
def scen(doc, sid):
    try: j=json.loads(doc)
    except Exception: return None
    for s in (j.get('scenarios') or []):
        if str(s.get('id'))==str(sid): return s
    return None
rows=list(db.execute("""SELECT id, source, plan_id, plan_version, cited_scenario_id, pnl_corrected
  FROM trader_positions WHERE entry_time>=? AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' ORDER BY id""",(ERA,)))
print("cited era rows:", len(rows))
def cells(rs, label):
    byc=collections.defaultdict(lambda: dict(ids=[], pnl=0.0, wins=0, nulls=[]))
    for r in rs:
        doc=plans.get((r['plan_id'], r['plan_version']))
        s=scen(doc, r['cited_scenario_id']) if doc else None
        cond=(s.get('condition') if s else 'NO-SCEN') or 'NO-COND'
        d=byc[cond]
        if r['pnl_corrected'] is None: d['nulls'].append(r['id'])
        else:
            d['ids'].append(r['id']); d['pnl']+=r['pnl_corrected']
            if r['pnl_corrected']>0: d['wins']+=1
    print("==", label)
    tot=0.0; totn=0
    for k,v in sorted(byc.items(), key=lambda kv:-len(kv[1]['ids'])):
        n=len(v['ids']); tot+=v['pnl']; totn+=n
        m=v['pnl']/n if n else 0
        sd=(sum((db.execute("SELECT pnl_corrected FROM trader_positions WHERE id=?",(i,)).fetchone()[0]-m)**2 for i in v['ids'])/(n-1))**0.5 if n>1 else 0
        t=(m/(sd/ n**0.5)) if n>1 and sd>0 else float('nan')
        print(f"  {k}: n={n} sum={round(v['pnl'],2)} mean={round(m,2)} wins={v['wins']} t={round(t,2)} nulls={v['nulls']} ids={v['ids']}")
    print(f"  TOTAL usable n={totn} sum={round(tot,2)}")
compliant=[r for r in rows if r['source']!='e7_farside_test' and r['plan_id']!='UNRESOLVABLE']
cells(compliant, "COMPLIANT (excl e7 572, excl UNRESOLVABLE 530/539)")
cells([r for r in rows if r['source']!='e7_farside_test'], "SENSITIVITY (incl UNRESOLVABLE)")
