#!/usr/bin/env python3
"""Read-only economics census; no production validators, writes, or invented prices.
Run from repository root. Frozen membership is extracted from its pinned Git blob.
"""
import collections, csv, datetime, decimal, hashlib, io, json, pathlib, re, sqlite3, subprocess
from zoneinfo import ZoneInfo
D = decimal.Decimal
out = pathlib.Path(__file__).parent
pin = '6095ca58fe5901ba398be374e4f9d3488d0bed6b'
frozen_csv = subprocess.check_output(['git','show',pin+':docs/superpowers/reports/2026-09-07-planner-preparation-audit/all-scenarios.csv'],text=True)
frozen_keys = {(r['plan'],r['scenario']) for r in csv.DictReader(io.StringIO(frozen_csv))}
c = sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
c.row_factory = sqlite3.Row
c.execute('BEGIN')
plans = list(c.execute("select rowid,plan_id,version,trade_date,session,doc,indicators_block,created_at from plans where trade_date >= '2026-08-15' order by rowid"))
# Date matches store.DayPlanEraStart's CT date; no P&L or trade outcome query.
rows=[]; roles=[]; london=[]
for p in plans:
 doc=json.loads(p['doc'],parse_float=D)
 logical=f"{p['trade_date']}:{p['session']}:v{p['version']}"
 atr_match=re.search(r'(?ms)^### 5m\n(?:(?!^### ).)*?^ATR14: ([0-9.]+)',p['indicators_block'])
 atr=D(atr_match[1]) if atr_match else None
 for idx,s in enumerate(doc.get('scenarios') or []):
  arm=s.get('arm') or {}; e,st,t=[D(str(arm.get(k) or 0)) for k in ('entry','stop','target')]
  complete=all(v>0 for v in (e,st,t)) and e!=st
  risk=abs(e-st) if complete else None
  side=1 if s.get('direction')=='long' else -1 if s.get('direction')=='short' else 0
  chain=[D(str(v)) for v in s.get('target_chain') or []]
  first_r=side*(chain[0]-e)/risk if complete and chain and side else None
  target_r=abs(t-e)/risk if complete else None
  r={'plan_rowid':p['rowid'],'plan_id':p['plan_id'],'version':p['version'],'logical':logical,'scenario_index':idx,'scenario':s.get('id'),'frozen':(logical,s.get('id')) in frozen_keys,'direction':s.get('direction'),'arm_enabled':arm.get('enabled'),'complete':complete,'entry':str(e) if e else None,'stop':str(st) if st else None,'arm_target':str(t) if t else None,'target_chain':[str(v) for v in chain],'risk_points':str(risk) if risk else None,'arm_R':str(target_r) if complete else None,'first_listed_R':str(first_r) if first_r is not None else None,'sub1':first_r is not None and 0<first_r<1,'arm_under2':target_r is not None and target_r<2,'correct_side':side*(e-st)>0 and side*(t-e)>0 if complete else None,'absent_half_tick':not any(abs(t-v)<=D('.125') for v in chain) if complete else None,'absent_tick':not any(abs(t-v)<=D('.25') for v in chain) if complete else None,'first_obstacle_present':'first_obstacle' in s,'stored_5m_atr':str(atr) if atr else None,'stored_atr_floor':str(atr*D('1.5')) if atr else None,'authored_equals_stored_floor':risk==atr*D('1.5') if risk and atr else None,'authored_within_tick_stored_floor':abs(risk-atr*D('1.5'))<=D('.25') if risk and atr else None}
  rows.append(r)
  if logical=='2026-09-08:LONDON:v2': london.append(r)
  # Literal role/use diagnostics ONLY. No map from free prose to allowed roles
  # exists; this enumerates the two narrowly defined claim families, not a gate.
  invalid_prices={D(v) for v in re.findall(r'(?<![\w.])\d{4,5}(?:\.\d+)?(?![\w.])',s.get('invalid') or '')}
  for li,l in enumerate(doc.get('levels') or []):
   price=D(str(l.get('price') or 0)); role=(l.get('instruction') or '').strip().lower()
   norm=re.sub(r'[_\s]+',' ',role)
   uses=[]
   if t and abs(t-price)<=D('.25'): uses.append('arm_target')
   if any(abs(v-price)<=D('.25') for v in chain): uses.append('target_chain')
   if any(abs(v-price)<=D('.25') for v in invalid_prices): uses.append('invalidation')
   diagnostic=[]
   if norm in {'confluence','confluence only','confluence reference','confluence ref','htf confluence','htf confluence only','htf confluence reference only'}:
    diagnostic += [u for u in uses if u in {'arm_target','target_chain'}]
   if norm in {'target','target only'} and 'invalidation' in uses: diagnostic.append('invalidation')
   if diagnostic: roles.append({'plan_rowid':p['rowid'],'logical':logical,'scenario':s.get('id'),'frozen':r['frozen'],'level_index':li,'label':l.get('label'),'price':str(price),'instruction':l.get('instruction'),'uses':diagnostic})
# The connection remains read-only throughout, including this snapshot.
summary={'captured_at_ct':datetime.datetime.now(ZoneInfo('America/Chicago')).isoformat(),'source_head':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'plan_rows':len(plans),'plan_rowids':[p['rowid'] for p in plans],'frozen_csv_sha256':hashlib.sha256(frozen_csv.encode()).hexdigest(),'scopes':{}}
for name,sample in [('frozen',[r for r in rows if r['frozen']]),('all_retained',rows)]:
 geom=[r for r in sample if r['complete']]; matched=[r for r in roles if name=='all_retained' or r['frozen']]
 summary['scopes'][name]={'scenario_n':len(sample),'plan_n':len({r['plan_rowid'] for r in sample}),'complete_n':len(geom),'sub1_n':sum(r['sub1'] for r in geom),'under2_n':sum(r['arm_under2'] for r in geom),'correct_side_n':sum(r['correct_side'] for r in geom),'absent_half_tick_n':sum(r['absent_half_tick'] for r in geom),'absent_tick_n':sum(r['absent_tick'] for r in geom),'missing_obstacle_n':sum(not r['first_obstacle_present'] for r in sample),'literal_role_diagnostic_n':len(matched),'literal_role_scenario_n':len({(r['plan_rowid'],r['scenario']) for r in matched}),'stored_atr_geometry_n':sum(r['stored_5m_atr'] is not None for r in geom),'authored_exact_floor_n':sum(r['authored_equals_stored_floor'] is True for r in geom),'authored_within_tick_floor_n':sum(r['authored_within_tick_stored_floor'] is True for r in geom)}
for name,data in [('scenarios.json',rows),('role-diagnostics.json',roles),('census.json',summary)]: (out/name).write_text(json.dumps(data,indent=2,ensure_ascii=False)+'\n')
print(json.dumps(summary['scopes'],indent=2))
for name,pred in [('C2',lambda r:r['arm_under2'] and r['frozen']),('C3',lambda r:r['absent_half_tick'] and r['frozen']),('current_extra_C3',lambda r:r['absent_half_tick'] and not r['frozen'])]:
 print(name,json.dumps([r for r in rows if pred(r)],ensure_ascii=False))
print('London',json.dumps(london,ensure_ascii=False))
