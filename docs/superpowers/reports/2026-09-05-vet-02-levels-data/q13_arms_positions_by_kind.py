#!/usr/bin/env python3
"""q13: which level KINDS are actually traded (arms + positions), as what (condition), and how they did (pnl_corrected).
Map: arm.entry_px / cited scenario arm.entry -> nearest level in the SAME plan doc (|dp|<=1.0 pt) -> label -> family."""
import sqlite3, json, re, collections
DB="file:/home/hoang/nofx/data/data.db?mode=ro"; con=sqlite3.connect(DB, uri=True); con.row_factory=sqlite3.Row
plans={}
for r in con.execute("SELECT plan_id, version, doc FROM plans"):
    try: plans[(r['plan_id'],r['version'])]=json.loads(r['doc'])
    except Exception: pass
def fam(label):
    l=label or ''
    for pre,f in (('VWAP','VWAP*'),('SWG-H','SWG-H'),('SWG-L','SWG-L'),('OB(','OB'),('Demand','DEMAND'),('Supply','SUPPLY'),('FVG','FVG'),('iFVG','IFVG'),('EQH','EQH'),('EQL','EQL'),('nPOC','nPOC'),('RN ','RN'),('IB','IB*')):
        if l.startswith(pre): return f
    return l
def nearest_level(doc, px, tol=1.0):
    best=None
    for l in (doc or {}).get('levels') or []:
        try: d=abs(float(l.get('price'))-px)
        except Exception: continue
        if d<=tol and (best is None or d<best[0]): best=(d,l.get('label'),l.get('grade'))
    return best
def scen(doc, sid):
    for s in (doc or {}).get('scenarios') or []:
        if s.get('id')==sid: return s
    return None
print("== ARMS (all, excluding TEST-E7): kind family of the arm level, condition, state")
arms=collections.Counter(); arm_rows=[]
for r in con.execute("SELECT id, plan_id, version, session, scenario, side, entry_px, state, state_reason, condition, fill_price FROM armed_orders WHERE session<>'TEST-E7' ORDER BY id"):
    doc=plans.get((r['plan_id'],r['version']))
    if doc is None:  # fall back to any version of that plan
        cand=[v for (p,v) in plans if p==r['plan_id']]; doc=plans.get((r['plan_id'],max(cand))) if cand else None
    nl=nearest_level(doc, r['entry_px']) if doc else None
    s=scen(doc, r['scenario']) if doc else None
    cond=r['condition'] or (s.get('condition') if s else '') or ''
    f=fam(nl[1]) if nl else 'NO-LEVEL-MATCH'
    arms[(f,cond,r['side'].lower(),r['state'])]+=1
    arm_rows.append((r['id'],r['session'],r['scenario'],r['side'],cond,r['entry_px'],f,(nl[1] if nl else None),(nl[2] if nl else None),r['state'],(r['state_reason'] or '')[:40]))
for k,v in sorted(arms.items(), key=lambda kv:-kv[1]): print(f"{v:3d}  {k}")
print("\nid session scen side cond entry fam label grade state reason")
for a in arm_rows: print(a)
print("\n== POSITIONS since 2026-08-15 with cited_scenario_id (source<>e7): scenario condition, level family at the scenario's arm/entry, pnl_corrected")
pos=collections.defaultdict(lambda: dict(n=0,pnl=0.0,null=0,wins=0,ids=[]))
for r in con.execute("SELECT id, plan_id, plan_version, cited_scenario_id, side, entry_price, pnl_corrected, source, plan_session, close_reason FROM trader_positions WHERE entry_time>=1786856400000 AND source<>'e7_farside_test' AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' ORDER BY id"):
    doc=plans.get((r['plan_id'],r['plan_version']))
    if doc is None:
        cand=[v for (p,v) in plans if p==r['plan_id']]; doc=plans.get((r['plan_id'],max(cand))) if cand else None
    s=scen(doc, r['cited_scenario_id']) if doc else None
    cond=(s.get('condition') if s else 'NO-SCEN') or ''
    ref=None
    if s:
        arm=s.get('arm') or {}
        ref=arm.get('entry') if isinstance(arm,dict) and arm.get('entry') else None
        if ref is None:
            m=re.findall(r'(\d{5}\.\d{2}|\d{5})', s.get('trigger') or ''); ref=float(m[0]) if m else None
    nl=nearest_level(doc, float(ref), 1.0) if (doc and ref) else None
    f=fam(nl[1]) if nl else 'NO-LEVEL-MATCH'
    key=(f,cond)
    a=pos[key]; a['n']+=1; a['ids'].append(r['id'])
    if r['pnl_corrected'] is None: a['null']+=1
    else:
        a['pnl']+=r['pnl_corrected']; a['wins']+= 1 if r['pnl_corrected']>0 else 0
for k,a in sorted(pos.items(), key=lambda kv:-kv[1]['n']):
    print(f"{a['n']:3d} n  pnl_corrected_sum={a['pnl']:9.2f} wins={a['wins']} null_excluded={a['null']}  {k}  ids={a['ids']}")
print("\n== by condition only"); byc=collections.defaultdict(lambda: dict(n=0,pnl=0.0,null=0,wins=0))
for (f,c),a in pos.items():
    b=byc[c]; b['n']+=a['n']; b['pnl']+=a['pnl']; b['null']+=a['null']; b['wins']+=a['wins']
for c,a in sorted(byc.items(), key=lambda kv:-kv[1]['n']): print(f"{a['n']:3d} pnl={a['pnl']:9.2f} wins={a['wins']} null={a['null']}  {c}")
print("\n== by level family only"); byf=collections.defaultdict(lambda: dict(n=0,pnl=0.0,null=0,wins=0))
for (f,c),a in pos.items():
    b=byf[f]; b['n']+=a['n']; b['pnl']+=a['pnl']; b['null']+=a['null']; b['wins']+=a['wins']
for f,a in sorted(byf.items(), key=lambda kv:-kv[1]['n']): print(f"{a['n']:3d} pnl={a['pnl']:9.2f} wins={a['wins']} null={a['null']}  {f}")
print("\n== how each level family is WRITTEN into scenarios (all plans): family x condition (arm entry or first price in trigger matched to a level within 1pt)")
sc=collections.Counter(); armed=collections.Counter()
for (pid,ver),doc in plans.items():
    for s in doc.get('scenarios') or []:
        arm=s.get('arm') or {}; ref=arm.get('entry') if isinstance(arm,dict) else None
        if not ref:
            m=re.findall(r'(\d{5}\.\d{2}|\d{5})', s.get('trigger') or ''); ref=float(m[0]) if m else None
        nl=nearest_level(doc, float(ref), 1.0) if ref else None
        f=fam(nl[1]) if nl else 'NO-LEVEL-MATCH'
        sc[(f,s.get('condition'))]+=1
        if isinstance(arm,dict) and arm.get('enabled'): armed[(f,s.get('condition'))]+=1
for k,v in sorted(sc.items(), key=lambda kv:-kv[1])[:45]: print(f"{v:4d} scen  {armed[k]:3d} arm-enabled  {k}")
