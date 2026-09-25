#!/usr/bin/env python3
"""Docs-only Section 10 extract. Production connection is mode=ro + query_only.
Run: python3 recompute.py --db /home/hoang/nofx/data/data.db --out SCRATCH
Offline: python3 recompute.py --sample population.csv --out SCRATCH
"""
import argparse,csv,datetime as dt,hashlib,json,math,pathlib,random,sqlite3
from collections import defaultdict
from zoneinfo import ZoneInfo
p=argparse.ArgumentParser();p.add_argument('--db');p.add_argument('--sample');p.add_argument('--out',required=True);a=p.parse_args();out=pathlib.Path(a.out);out.mkdir(parents=True,exist_ok=True)
CT=ZoneInfo('America/Chicago');CUT=1786770000000;STRICT=int(dt.datetime(2026,9,3,11,10,tzinfo=CT).timestamp()*1000)
SQL="""SELECT id,entry_time,exit_time,plan_id,plan_session,source,side,pnl_corrected,fee,mae,mfe FROM trader_positions
WHERE entry_time>=1786770000000 AND status='CLOSED'
AND plan_id IS NOT NULL AND TRIM(plan_id)<>'' AND plan_id<>'UNRESOLVABLE'
AND COALESCE(source,'')<>'e7_farside_test' AND pnl_corrected IS NOT NULL
ORDER BY entry_time,id"""
def j(name,x): (out/name).write_text(json.dumps(x,indent=2,sort_keys=True)+'\n')
def csvout(name,rr):
 with (out/name).open('w',newline='') as f:
  w=csv.DictWriter(f,fieldnames=list(rr[0]),lineterminator='\n');w.writeheader();w.writerows(rr)
def ct(ms): return dt.datetime.fromtimestamp(int(ms)/1000,CT)
def day(ms): return str((ct(ms)-dt.timedelta(hours=17)).date())
def wil(k,n):
 if not n:return {'k':k,'n':n,'p':None,'wilson95':None}
 z=1.959963984540054;p=k/n;den=1+z*z/n;m=(p+z*z/(2*n))/den;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
 return {'k':k,'n':n,'p':p,'wilson95':[max(0.0,m-h),min(1.0,m+h)]}
if a.db:
 c=sqlite3.connect('file:'+a.db+'?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
 rows=[dict(r) for r in c.execute(SQL)]
 for r in rows:r.update(cme_open_day=day(r['entry_time']),entry_ct=ct(r['entry_time']).isoformat(),hold_minutes=(r['exit_time']-r['entry_time'])/60000)
 ids={r['id'] for r in rows}
 excluded=[dict(r) for r in c.execute('select id,source,plan_id,pnl_corrected,status from trader_positions where entry_time>=? order by id',(CUT,)) if r['id'] not in ids]
 touches=[dict(r) for r in c.execute('select id,level_kind,level_price,opened_at_ms,outcome,plan_id,plan_version from touch_outcomes order by id')]
 keys=defaultdict(list)
 for r in touches:keys[(r['level_kind'],r['level_price'],r['opened_at_ms'])].append(r)
 tr=[{'kind':k[0],'price':k[1],'opened_at_ms':k[2],'ids':' '.join(str(r['id']) for r in v),'outcomes':' '.join(sorted({r['outcome'] for r in v}))} for k,v in keys.items()]
 csvout('touch_keys.csv',tr)
 plans=[dict(r) for r in c.execute('select plan_id,version,trade_date,session,trigger_reason,lifecycle,created_at from plans order by plan_id,version')];csvout('plan_versions.csv',plans)
 pg=defaultdict(list)
 for r in plans:pg[r['plan_id']].append(r['version'])
 rp=[{'plan_id':k,'versions':v} for k,v in pg.items() if len(v)>1]
 facts=[dict(r) for r in c.execute('select id,plan_id,version,bias_regime,bias_ai,bias_tree,tokens_in from planner_read_facts order by id')];csvout('read_facts.csv',facts)
 cps=[dict(r) for r in c.execute('select id,seated,score,grade,cut_reason,score_components,plan_id,plan_version from candidate_pool order by id')];csvout('candidate_provenance.csv',cps)
 meta={'extracted_utc':dt.datetime.now(dt.timezone.utc).isoformat(),'query_only':c.execute('pragma query_only').fetchone()[0],'sql':SQL,'cut_ms':CUT,'strict_cut_ms':STRICT,'excluded':excluded,'trade_excursions_count':c.execute('select count(*) from trade_excursions').fetchone()[0],'touch_rows':len(touches),'touch_price_time_keys':len(keys),'rthl_rows':sum(r['level_kind']=='RTH-L' for r in touches),'rthl_keys':sum(k[0]=='RTH-L' for k in keys),'plans':len(plans),'plan_ids':len(pg),'revised_plan_ids':rp,'planner_facts_n':len(facts),'planner_facts_ids':[r['id'] for r in facts],'candidate_n':len(cps),'candidate_cut_ids':[r['id'] for r in cps if not r['seated']]}
 # Allowlist only; never serialize full strategy configuration.
 cfg=[]
 for rr in c.execute('select t.scan_interval_minutes,s.config from traders t join strategies s on s.id=t.strategy_id'):
  d=json.loads(rr[1]).get('day_plan',{});cfg.append({'scan_minutes':rr[0],'plan_mode':d.get('plan_mode'),'sessions_enabled':d.get('sessions_enabled'),'sessions':[{'session':s.get('session'),'enable':s.get('enable')} for s in d.get('sessions',[])]})
 meta['saved_config_allowlist']=cfg;c.rollback();c.close();j('extraction.json',meta);csvout('population.csv',rows);(out/'population.sql').write_text(SQL+';\n')
else:
 rows=list(csv.DictReader(open(a.sample)))
 for r in rows:
  for k in ['id','entry_time','exit_time']:r[k]=int(r[k])
  for k in ['pnl_corrected','fee','hold_minutes']:r[k]=float(r[k])
assert len(rows)==58 and abs(sum(r['pnl_corrected'] for r in rows)+466.428572)<1e-6
assert [sum(r['pnl_corrected']>0 for r in rows),sum(r['pnl_corrected']<0 for r in rows),sum(r['pnl_corrected']==0 for r in rows)]==[18,38,2]
g=defaultdict(list)
for r in rows:g[r['cme_open_day']].append(r)
assert len(g)==12
days=[{'cme_open_day':k,'n':len(v),'pnl_corrected':sum(r['pnl_corrected'] for r in v),'ids':' '.join(str(r['id']) for r in v)} for k,v in sorted(g.items())];csvout('days.csv',days)
def summ(v):
 x=[r['pnl_corrected'] for r in v];w=[z for z in x if z>0];l=[z for z in x if z<0]
 return {'n':len(v),'D':len({r['cme_open_day'] for r in v}),'ids':[r['id'] for r in v],'sum':sum(x),'mean':sum(x)/len(x),'win':wil(len(w),len(x)),'loss':wil(len(l),len(x)),'flat':wil(sum(z==0 for z in x),len(x)),'nonflat_win':wil(len(w),len(w)+len(l)),'avg_win':sum(w)/len(w) if w else None,'avg_loss':sum(l)/len(l) if l else None}
r=summ(rows);r['sessions']={s:summ([x for x in rows if x['plan_session']==s]) for s in sorted({x['plan_session'] for x in rows})};r['post_strict_ids']=[x['id'] for x in rows if x['entry_time']>=STRICT];r['recorded_fee_sum']=sum(x['fee'] for x in rows)
r['payoff']=r['avg_win']/-r['avg_loss'];r['profit_factor']=sum(x['pnl_corrected'] for x in rows if x['pnl_corrected']>0)/-sum(x['pnl_corrected'] for x in rows if x['pnl_corrected']<0)
r['positive_ids']=[x['id'] for x in rows if x['pnl_corrected']>0];r['negative_ids']=[x['id'] for x in rows if x['pnl_corrected']<0];r['flat_ids']=[x['id'] for x in rows if x['pnl_corrected']==0]
rng=random.Random(2026090510);boot=[]
for _ in range(20000):
 picks=[days[rng.randrange(len(days))] for i in days];boot.append(sum(x['pnl_corrected'] for x in picks)/sum(x['n'] for x in picks))
boot.sort();r['day_cluster_bootstrap']={'seed':2026090510,'replicates':20000,'ci95':[boot[499],boot[19499]],'estimator':'resample 12 active CME open-day blocks; pooled trade mean; nearest rank percentile'}
r['cost_sensitivity']=[{'assumed_extra_round_trip_usd':cost,'mean':r['mean']-cost,'ci95':[v-cost for v in r['day_cluster_bootstrap']['ci95']]} for cost in [0,2,4,8]]
j('results.json',r)
lines=['SECTION 10 CORRECTED EVIDENCE','Population IDs: '+str(r['ids']),f"n={r['n']}; CME open-day blocks={r['D']}; pnl_corrected={r['sum']:.6f}; mean={r['mean']:.6f}"]
for k in ['win','loss','flat','nonflat_win']:lines.append(k+': '+json.dumps(r[k]))
lines += ['Positive IDs: '+str(r['positive_ids']),'Negative IDs: '+str(r['negative_ids']),'Flat IDs: '+str(r['flat_ids']),'Payoff='+str(r['payoff'])+'; profit factor='+str(r['profit_factor']),'Day-cluster CI: '+json.dumps(r['day_cluster_bootstrap']),'Post strict IDs: '+str(r['post_strict_ids'])+'; recorded fee sum='+str(r['recorded_fee_sum'])]
for s,v in r['sessions'].items():lines.append(s+': '+json.dumps(v))
lines.append('Costs: '+json.dumps(r['cost_sensitivity']));(out/'evidence.txt').write_text('\n'.join(lines)+'\n')
print('\n'.join(lines))
