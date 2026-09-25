import sqlite3,json,csv,datetime,pathlib,collections,hashlib
from zoneinfo import ZoneInfo
ROOT=pathlib.Path(__file__).resolve().parent
CT=ZoneInfo('America/Chicago')
def ms(s):return int(datetime.datetime.fromisoformat(s).replace(tzinfo=CT).timestamp()*1000)
def tm(s):
 if isinstance(s,(int,float)):return int(s)
 return int(datetime.datetime.fromisoformat(s.replace('Z','+00:00')).timestamp()*1000)
def ct(v):return datetime.datetime.fromtimestamp(v/1000,CT).isoformat()
def write(name,rows):
 if name.endswith('.json'): (ROOT/name).write_text(json.dumps(rows,indent=2));return
 rows=list(rows)
 if not rows:return
 with (ROOT/name).open('w') as f:
  w=csv.DictWriter(f,lineterminator="\n",fieldnames=list(rows[0]));w.writeheader();w.writerows(rows)
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('pragma query_only=on');c.execute('begin')
def q(s,p=()):return [dict(r) for r in c.execute(s,p)]
a,z=ms('2026-09-02'),ms('2026-09-05')
pos=q('select id,plan_id,plan_version,cited_scenario_id,plan_trade_date,plan_session,source,side,entry_quantity,entry_time,exit_time,entry_price,exit_price,pnl_corrected,close_reason,mae,mfe from trader_positions where entry_time>=1786770000000 order by id')
for r in pos:r['eligible']=r['source']!='e7_farside_test' and bool(r['plan_id']) and r['plan_id']!='UNRESOLVABLE' and r['pnl_corrected'] is not None
write('era.csv',pos);ep=[r for r in pos if r['eligible']];write('trades.csv',[dict(r,entry_ct=ct(r['entry_time']),exit_ct=ct(r['exit_time'])) for r in ep if a<=r['entry_time']<z])
plans=q("select plan_id,version,trade_date,session,created_at,lifecycle,trigger_reason,doc from plans where trade_date between '2026-09-01' and '2026-09-04' order by created_at")
# Include Sep-1 ASIA carry-in versions alive or authored after midnight Sep-2.
plans=[r for r in plans if r['trade_date']>='2026-09-02' or r['session']=='ASIA']
for r in plans:r['born']=tm(r['created_at']);r['doc']=json.loads(r['doc'])
write('retained_plan_versions.json',plans)
# Keep only versions overlapping the wall-clock audit and session; no premature clipping of pre-session publication.
for r in plans:
 d=r['trade_date'];s=r['session'];st={'ASIA':'17:00','LONDON':'02:00','NY':'08:30'}[s];end={'ASIA':'02:00','LONDON':'08:30','NY':'14:45'}[s]
 r['session_start']=ms(d+'T'+st);r['session_end']=ms(d+'T'+end)+(86400000 if s=='ASIA' else 0)
 nxt=[p['born'] for p in plans if p['plan_id']==r['plan_id'] and p['version']>r['version']]
 r['end']=min([r['session_end'],z]+nxt);r['start']=max(r['born'],r['session_start'],a)
plans=[r for r in plans if r['end']>r['start']]
write('plans.json',plans)
write('plans.csv',[{k:(len(r['doc'].get('scenarios',[])) if k=='scenarios' else r[k]) for k in ['plan_id','version','trade_date','session','created_at','lifecycle','trigger_reason','start','end','scenarios']} for r in plans])
sc=[]
for p in plans:
 for s in p['doc'].get('scenarios',[]):sc.append(dict(opportunity=p['plan_id']+'/v'+str(p['version'])+'/'+s['id'],plan_id=p['plan_id'],version=p['version'],date=p['trade_date'],session=p['session'],born=p['born'],start=p['start'],end=p['end'],scenario=s['id'],direction=s.get('direction'),condition=s.get('condition'),quality=s.get('quality'),arm_enabled=bool((s.get('arm') or {}).get('enabled')),bias=p['doc'].get('bias',{}).get('direction'),doc=json.dumps(s)))
write('opportunities.csv',sc)
arms=q('select id,plan_id,version,armed_under_version,session,scenario,side,entry_px,stop_px,target_px,state,state_reason,signal_id,fill_price,fill_quantity,created_at,updated_at,leg_index,leg_count,kind,condition,placement_seq from armed_orders order by id')
arms=[r for r in arms if tm(r['created_at'])<z and tm(r['updated_at'])>=a and not r['scenario'].startswith('TEST') and 'e7_farside_test' not in (r['state_reason'] or '')];write('arms.csv',arms)
ds=q("select id,timestamp,plan_id,plan_version,cited_scenario_id,decisions,execution_log,risk_check_error,success,cycle_type,cycle_trigger from decision_records where date(timestamp,'-5 hours') between '2026-09-02' and '2026-09-04' order by id")
write('decisions.csv',ds)
intents=[]
for r in ds:
 try: acts=json.loads(r['decisions'] or '[]') or []
 except:acts=[]
 for j,d in enumerate(acts):
  if d.get('action') not in ['open_long','open_short']:continue
  intents.append(dict(id=r['id'],action_index=j,timestamp=r['timestamp'],plan_id=r['plan_id'],version=r['plan_version'],scenario=r['cited_scenario_id'],action=d['action'],proposal=json.dumps(d),execution_log=r['execution_log'],risk_check_error=r['risk_check_error']))
write('intents.csv',intents)
write('refusals.csv',[r for r in ds if 'refus' in ((r['execution_log'] or '')+(r['risk_check_error'] or '')).lower()])
bars=q("select rowid,symbol,tf,open_time_ms,o,h,l,c,v,convention from bars where symbol='MNQ' and tf='1m' and open_time_ms>=? and open_time_ms<? order by open_time_ms",(ms('2026-08-30'),z));write('bars.csv',bars)
write('lifecycle.csv',q('select * from plan_lifecycle_log'))
write('read_facts.csv',q('select id,trade_date,session,plan_id,version,atr5m,stop_floor_pts,bias_regime,created_at,scope_bars,scope_intv from planner_read_facts'))
sn=q('select id,reason,orders_json,order_count,working_count,emitted_at_ms,received_at_ms from nt8_order_snapshots where emitted_at_ms>=? and emitted_at_ms<? order by id',(a,z))
for r in sn:
 orders=json.loads(r['orders_json'] or '[]');r['orders_json']=json.dumps([{k:v for k,v in o.items() if k in ['symbol','name','signal_id','order_id','type','state','action','side','quantity','filled','limit_price','stop_price','avg_fill_price','order_type']} for o in orders])
write('snapshots.csv',sn)
summary={'asof_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'query_only':c.execute('pragma query_only').fetchone()[0],'era_n':len(ep),'era_ids':[r['id'] for r in ep],'excluded_ids':[r['id'] for r in pos if not r['eligible']],'era_pnl_corrected':sum(r['pnl_corrected'] for r in ep),'WLF':[sum(r['pnl_corrected']>0 for r in ep),sum(r['pnl_corrected']<0 for r in ep),sum(r['pnl_corrected']==0 for r in ep)],'CME_days':sorted(set((datetime.datetime.fromtimestamp(r['entry_time']/1000,CT)-datetime.timedelta(hours=17)).date().isoformat() for r in ep)),'trade_excursions_count':q('select count(*) n from trade_excursions')[0]['n'],'plans_n':len(plans),'opportunities_n':len(sc),'arms_n':len(arms),'decisions_n':len(ds),'intents_n':len(intents)}
write('extract_summary.json',summary);print(json.dumps(summary,indent=2))

# Additional exact pre-10:00 context, with the same read-only transaction.
write('q2_decisions_before_1000.csv',q("select id,timestamp,plan_id,plan_version,cited_scenario_id,decisions,cot_trace from decision_records where datetime(timestamp,'-5 hours')>='2026-09-03 09:15:00' and datetime(timestamp,'-5 hours')<'2026-09-03 10:00:00' order by id"))
news=[]
for row in q('select id,timestamp,input_prompt from decision_records where id in (37097,37098)'):
 for line in (row['input_prompt'] or '').splitlines():
  if any(k in line.lower() for k in ['ism','news','calendar','claims','pmi','payroll']):news.append(dict(id=row['id'],timestamp=row['timestamp'],text=line))
write('q2_news_at_time.json',news)
config=json.loads(q("select config from strategies where id like 'a5b7662e%'")[0]['config']);selected={}
def walk_config(obj,prefix=''):
 if isinstance(obj,dict):
  for k,v in obj.items():
   if any(term in k for term in ['guardrails_enabled','daily_loss','max_contracts_per_order','min_risk_reward','htf_veto','min_scenario_quality','condition_status','acceptance_rule']):selected[prefix+k]=v
   walk_config(v,prefix+k+'.')
 elif isinstance(obj,list):
  for i,v in enumerate(obj):walk_config(v,prefix+str(i)+'.')
walk_config(config);write('rule_config.json',selected)
write('snapshot_account_check.json',q('select count(distinct account) as distinct_accounts from nt8_order_snapshots where emitted_at_ms>=? and emitted_at_ms<?',(a,z)))
