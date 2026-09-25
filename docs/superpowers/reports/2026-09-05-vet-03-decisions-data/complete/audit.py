#!/usr/bin/env python3
"""Read-only Section 3 audit. Outputs only under explicitly supplied scratch directory.
No trading modules imported. UTC parses preserve offsets; Chicago session boundaries.
"""
import argparse,sqlite3,json,csv,math,statistics as st,hashlib,datetime as dt,collections as co,bisect,pathlib,re
from zoneinfo import ZoneInfo
ap=argparse.ArgumentParser();ap.add_argument('--out',required=True);ap.add_argument('--repo',required=True);ap.add_argument('--db',default='/home/hoang/nofx/data/data.db');a=ap.parse_args()
out=pathlib.Path(a.out);out.mkdir(parents=True,exist_ok=True);repo=pathlib.Path(a.repo)
c=sqlite3.connect('file:'+a.db+'?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
CT=ZoneInfo('America/Chicago');UTC=dt.timezone.utc
START=dt.datetime(2026,8,29,tzinfo=CT);END=dt.datetime(2026,9,5,tzinfo=CT);STRICT=dt.datetime(2026,9,3,11,10,33,tzinfo=CT);ERA=1786770000000
summary={}
def tm(s):
 if isinstance(s,(int,float)):return dt.datetime.fromtimestamp(s/1000,UTC)
 t=dt.datetime.fromisoformat(s.replace('Z','+00:00'));return t if t.tzinfo else t.replace(tzinfo=UTC)
def ms(t):return int(t.timestamp()*1000)
def rows(q,args=()):return [dict(r) for r in c.execute(q,args)]
def save(name,rr):
 if not rr:return
 with (out/name).open('w',newline='') as f:
  w=csv.DictWriter(f,fieldnames=list(rr[0]),lineterminator="\n");w.writeheader();w.writerows(rr)
def rate(k,n):
 if not n:return {'k':k,'n':n,'pct':None,'wilson95':None}
 z=1.96;p=k/n;d=1+z*z/n;ctr=(p+z*z/(2*n))/d;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
 return {'k':k,'n':n,'pct':round(100*p,3),'wilson95':[round(100*(ctr-h),3),round(100*(ctr+h),3)]}
def stats(rr):
 v=[r['pnl_corrected'] for r in rr];w=[x for x in v if x>0];l=[x for x in v if x<0]
 return {'n':len(rr),'ids':[r['id'] for r in rr],'pnl':sum(v),'mean':st.mean(v) if v else None,'WLF':[len(w),len(l),len(v)-len(w)-len(l)],'win_exflat':rate(len(w),len(w)+len(l)),'win_all':rate(len(w),len(v)),'payoff':st.mean(w)/abs(st.mean(l)) if w and l else None,'days':len({(tm(r['entry_time']).astimezone(CT)-dt.timedelta(hours=17)).date() for r in rr})}
p=rows('SELECT * FROM trader_positions WHERE entry_time>=? ORDER BY id',(ERA,))
elig=[r for r in p if r['source']!='e7_farside_test' and r['plan_id']!='UNRESOLVABLE' and r['pnl_corrected'] is not None]
summary['population']={'era_ms':ERA,'raw_n':len(p),'excluded':[{'id':r['id'],'test':r['source']=='e7_farside_test','unresolvable':r['plan_id']=='UNRESOLVABLE','null_pnl':r['pnl_corrected'] is None} for r in p if r not in elig],'eligible':stats(elig),'post_strict':stats([r for r in elig if tm(r['entry_time'])>=STRICT])}
save('population58.csv',[{k:r[k] for k in ['id','source','plan_id','plan_version','cited_scenario_id','plan_session','plan_band','entry_time','exit_time','side','entry_price','exit_price','pnl_corrected','mae','mfe']} for r in elig])
for kind,col in [('source','source'),('session','plan_session'),('side','side'),('slot','cited_scenario_id')]:summary[kind]={str(k):stats([r for r in elig if r[col]==k]) for k in sorted({r[col] for r in elig},key=str)}
summary['excursions_count']=c.execute('select count(*) from trade_excursions').fetchone()[0]
plans=rows('SELECT rowid AS plan_rowid,* FROM plans ORDER BY created_at');planmap={(r['plan_id'],r['version']):r for r in plans}
for r in plans:r['d']=json.loads(r['doc']);r['t']=tm(r['created_at'])
# Source=system subset, exact plan/version/scenario join; ticket heuristic is explicitly separated.
decs=rows("SELECT id,trader_id,timestamp,plan_id,plan_version,cited_scenario_id,decision_json,execution_log,ai_request_duration_ms,cycle_number FROM decision_records WHERE decision_json LIKE '%open_%'")
for d in decs:
 d['t']=tm(d['timestamp'])
 try:x=json.loads(d['decision_json']);d['items']=x if isinstance(x,list) else [x]
 except Exception:d['items']=[]
coverage=[]
for r in elig:
 if r['source']!='system':continue
 pr=planmap.get((r['plan_id'],r['plan_version']));sc=next((s for s in (pr['d'].get('scenarios') or []) if s.get('id')==r['cited_scenario_id']),None) if pr else None
 matches=[]
 for d in decs:
  lag=(tm(r['entry_time'])-d['t']).total_seconds()
  if d['trader_id']!=r['trader_id'] or not abs(lag)<=180:continue
  for x in d['items']:
   if isinstance(x,dict) and x.get('action')=='open_'+r['side'].lower() and x.get('stop_loss') and x.get('take_profit'):
    matches.append((d,x,lag))
 exact=[m for m in matches if m[0]['plan_id']==r['plan_id'] and m[0]['plan_version']==r['plan_version'] and m[1].get('cited_scenario')==r['cited_scenario_id'] and ('open_'+r['side'].lower()+' succeeded') in (m[0]['execution_log'] or '')]
 best=exact[0] if len(exact)==1 else None
 coverage.append({'position_id':r['id'],'pnl_corrected':r['pnl_corrected'],'plan_id':r['plan_id'],'plan_version':r['plan_version'],'plan_rowid':pr['plan_rowid'] if pr else '', 'plan_precedes_entry':bool(pr and pr['t']<=tm(r['entry_time'])),'scenario':r['cited_scenario_id'],'condition':sc.get('condition') if sc else '', 'same_trader_side_abs180s_candidates':','.join(str(m[0]['id']) for m in matches),'exact_executed_candidates':','.join(str(m[0]['id']) for m in exact),'ticket_id':best[0]['id'] if best else '', 'decision_record_minus_fill_seconds':-best[2] if best else '', 'ticket_stop':best[1]['stop_loss'] if best else '', 'ticket_target':best[1]['take_profit'] if best else '', 'initial_broker_risk_verified':False,'mae_mfe_kind':'position-field proxy; censored; not trade_excursions'})
save('subset47_coverage.csv',coverage)
summary['subset47']={'n':len(coverage),'plan_scenario_join':rate(sum(bool(r['condition']) for r in coverage),len(coverage)),'plan_precedes_entry':rate(sum(r['plan_precedes_entry'] for r in coverage),len(coverage)),'unique_exact_executed_ticket':rate(sum(bool(r['ticket_id']) for r in coverage),len(coverage)),'by_condition':{k:stats([p for p in elig if p['id'] in {r['position_id'] for r in coverage if r['condition']==k}]) for k in sorted({r['condition'] for r in coverage},key=str)}}
# Last 7 full CT dates; persisted plan versions are NOT all successful authoring calls.
week=[p for p in plans if START<=p['t']<END and p['session'] in ('ASIA','LONDON','NY')]
save('plans_last7.csv',[{k:r[k] for k in ['plan_rowid','plan_id','version','trade_date','session','trigger_reason','lifecycle','created_at','prompt_hash']} for r in week])
rj=[r for r in rows('SELECT id,trader_id,trade_date,session,prompt_hash,attempt,reject_reason,created_at FROM planner_rejected_prompts ORDER BY created_at') if START<=tm(r['created_at'])<END]
def family(s):
 q=s.lower()
 if 'came back across' in q or 'void' in q:return 'continuation_void'
 if 'displacement' in q and '0.00' in q:return 'continuation_displacement_zero'
 if 'no confirming close' in q or 'bd_min_closes' in q:return 'continuation_missing_close'
 if 'not allowed for' in q or 'invalid (touch' in q:return 'confirmation_vocabulary'
 if 'fade_requires_touch' in q:return 'fade_requires_touch'
 if 'entry_mode=pullback' in q:return 'arm_requires_pullback'
 if 'arm on' in q or 'arm legs' in q:return 'arm_legs'
 if '503' in q or 'stream interrupted' in q or 'eof' in q:return 'transport'
 if 'gap-up' in q:return 'gap_up'
 if 'too many levels' in q:return 'level_cap'
 if 'unreachable' in q:return 'retest_distance'
 return 'other'
for r in rj:r['family']=family(r['reject_reason'])
save('rejects_last7.csv',rj)
summary['last7']={'all_session_types_versions':sum(START<=p['t']<END for p in plans),'excluded_non_intraday_rows':[{k:p[k] for k in ['plan_rowid','plan_id','version','session','trade_date','trigger_reason','created_at']} for p in plans if START<=p['t']<END and p['session'] not in ('ASIA','LONDON','NY')],'window_ct':[START.isoformat(),END.isoformat()],'plan_versions':len(week),'triggers':dict(co.Counter(p['trigger_reason'] for p in week)),'reject_rows':len(rj),'first_reject':rj[0]['created_at'],'family_shares_of_persisted_rejects':{k:{**rate(sum(r['family']==k for r in rj),len(rj)),'ids':[r['id'] for r in rj if r['family']==k]} for k in sorted({r['family'] for r in rj})},'true_attempt_reject_rate':'UNMEASURABLE: no read UUID or exhaustive success/transport attempt table; lifecycle rows and fail-closed versions are not accepted LLM attempts'}
# Read episode matching based on ordered attempts within same trader/session/trade_date; reset at attempt1 or non-increasing attempt. No arbitrary twenty-minute cluster.
episodes=[]
for key in sorted({(r['trader_id'],r['trade_date'],r['session']) for r in rj}):
 rr=[r for r in rj if (r['trader_id'],r['trade_date'],r['session'])==key];grp=[]
 for r in rr:
  if grp and r['attempt']<=grp[-1]['attempt']:episodes.append(grp);grp=[]
  grp.append(r)
 if grp:episodes.append(grp)
epout=[]
for ep in episodes:
 first,last=ep[0],ep[-1];t0=tm(first['created_at']);tl=tm(last['created_at'])
 nextfirst=min((tm(e[0]['created_at']) for e in episodes if e[0]['trade_date']==first['trade_date'] and e[0]['session']==first['session'] and tm(e[0]['created_at'])>tl),default=END)
 candidate=[p for p in week if p['trade_date']==first['trade_date'] and p['session']==first['session'] and tl<=p['t']<nextfirst and not p['trigger_reason'].startswith(('dormant','rearmed'))]
 nxt=min(candidate,key=lambda x:x['t']) if candidate else None
 epout.append({'reject_ids':','.join(str(r['id']) for r in ep),'attempts':','.join(str(r['attempt']) for r in ep),'session':first['session'],'trade_date':first['trade_date'],'first_reject':first['created_at'],'last_reject':last['created_at'],'within_reject_span_seconds':(tl-t0).total_seconds(),'next_plan_rowid':nxt['plan_rowid'] if nxt else '', 'next_trigger':nxt['trigger_reason'] if nxt else '', 'first_reject_to_next_plan_seconds':(nxt['t']-t0).total_seconds() if nxt else '', 'quality':'temporal association only; no read UUID; not measured model latency'})
save('reauthor_episodes.csv',epout)
v=[r['first_reject_to_next_plan_seconds'] for r in epout if r['next_plan_rowid']]
summary['reauthor']={'episode_count':len(epout),'matched':len(v),'median_seconds_association':st.median(v) if v else None,'min':min(v) if v else None,'max':max(v) if v else None,'fail_closed_associations':sum(r['next_trigger']=='planner_fail_closed' for r in epout),'true_latency':'UNMEASURABLE without read/attempt start-end IDs; use worked log example separately'}
# Reachability proxy: windows clip active lifecycle, next version, real session start/end.
# Bars must open AFTER birth and close BEFORE window end; excludes birth-minute leakage.
bars=rows("SELECT open_time_ms,o,h,l,c,convention FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms");times=[b['open_time_ms'] for b in bars]
summary['bar_conventions']=dict(co.Counter(b['convention'] for b in bars))
groups=co.defaultdict(list)
for p in plans:groups[p['plan_id']].append(p)
scrows=[]
for vs in groups.values():
 vs.sort(key=lambda p:p['t'])
 for i,p in enumerate(vs):
  if p not in week:continue
  td=dt.date.fromisoformat(p['trade_date']);sess=p['session'];sh,sm,eh,em,plus={'ASIA':(17,0,2,0,1),'LONDON':(2,0,8,30,0),'NY':(8,30,14,45,0)}[sess]
  s0=dt.datetime.combine(td,dt.time(sh,sm),CT);s1=dt.datetime.combine(td+dt.timedelta(days=plus),dt.time(eh,em),CT)
  t0=max(p['t'],s0);t1=min(s1,vs[i+1]['t']) if i+1<len(vs) else s1
  lo=ms(t0);hi=ms(t1);firstminute=((lo+59999)//60000)*60000;lastminute=(hi//60000)*60000-60000
  seg=bars[bisect.bisect_left(times,firstminute):bisect.bisect_right(times,lastminute)] if t1>t0 else []
  expected=max(0,(lastminute-firstminute)//60000+1);complete=len(seg)==expected and expected>0
  for index,s in enumerate(p['d'].get('scenarios') or []):
   conf=s.get('confirm') or {};ref=conf.get('ref_price');hit=next((b for b in seg if ref and b['l']<=ref<=b['h']),None)
   touchonly=conf.get('rule')=='touch' and not s.get('confirm2') and not s.get('breakdown')
   k=f"{p['plan_id']}|v{p['version']}|{s.get('id')}|idx{index}"
   geometry={q:s.get(q) for q in ['condition','direction','confirm','confirm2','breakdown','arm','trigger','invalid','target_chain']}
   ghash=hashlib.sha256(json.dumps(geometry,sort_keys=True).encode()).hexdigest()[:16]
   scrows.append({'opportunity_version_key':k,'geometry_hash':ghash,'plan_rowid':p['plan_rowid'],'plan_id':p['plan_id'],'version':p['version'],'scenario':s.get('id'),'condition':s.get('condition'),'direction':s.get('direction'),'lifecycle':p['lifecycle'],'rule':conf.get('rule'),'confirm2_rule':(s.get('confirm2') or {}).get('rule'),'ref_price':ref,'arm_enabled':bool((s.get('arm') or {}).get('enabled')),'start_ct':t0.astimezone(CT).isoformat(),'end_ct':t1.astimezone(CT).isoformat(),'expected_bars':expected,'stored_bars':len(seg),'complete_grid':complete,'first_touch_bar_ms':hit['open_time_ms'] if hit else '', 'touch_proxy':('EXCLUDED_INACTIVE' if p['lifecycle']!='active' else 'NO_REF' if not ref else 'NO_WINDOW' if not expected else 'TOUCHED' if hit else 'NOT_TOUCHED' if complete else 'UNKNOWN_GAP'),'full_confirm_supported':touchonly,'full_live_fire':'UNMEASURABLE: arrival-time feed, formation, alive/void and parameter history not complete'})
save('scenario_windows_last7.csv',scrows)
valid=[r for r in scrows if r['touch_proxy'] in ('TOUCHED','NOT_TOUCHED')];fullgrid=[r for r in valid if r['complete_grid']];simple=[r for r in fullgrid if r['full_confirm_supported']]
summary['trigger']={'all_version_scenarios':len(scrows),'statuses':dict(co.Counter(r['touch_proxy'] for r in scrows)),'reachability_observable':rate(sum(r['touch_proxy']=='TOUCHED' for r in valid),len(valid)),'complete_grid_touch':rate(sum(r['touch_proxy']=='TOUCHED' for r in fullgrid),len(fullgrid)),'touch_only_confirmation_component':rate(sum(r['touch_proxy']=='TOUCHED' for r in simple),len(simple)),'n_distinct_geometry_per_plan':len({(r['plan_id'],r['geometry_hash']) for r in scrows}),'by_condition_complete_grid':{k:rate(sum(r['touch_proxy']=='TOUCHED' for r in fullgrid if r['condition']==k),sum(r['condition']==k for r in fullgrid)) for k in sorted({r['condition'] for r in fullgrid})},'full_trigger_share':'UNMEASURABLE: full ordered confirmation/invalidation and live availability; bar reachability is not execution'}
# Exact raw source refusal rows, no aggregate causal P&L.
fpath=repo/'docs/superpowers/reports/2026-09-04-two-day-audit-data/refusals.csv'
refusals=list(csv.DictReader(fpath.open()));refout=[]
for i,r in enumerate(refusals,2):
 ds=rows('select id,trader_id,plan_id,plan_version,cited_scenario_id,execution_log from decision_records where cycle_number=?',(r['decision_cycle'],)) if r['decision_cycle'] else []
 refout.append({'source_csv_line':i,**r,'linked_decision_ids':','.join(str(d['id']) for d in ds),'same_cycle_executed':any('open_long succeeded' in (d['execution_log'] or '') or 'open_short succeeded' in (d['execution_log'] or '') for d in ds),'counterfactual_quality':'legacy OHLC event proxy; refusal minute may predate decision; not causal or broker-fill verified'})
save('refusal_events_legacy_audit.csv',refout)
summary['refusals']={k:{'event_count':len([r for r in refout if r['leg']==k]),'csv_lines':[r['source_csv_line'] for r in refout if r['leg']==k],'same_cycle_executed_lines':[r['source_csv_line'] for r in refout if r['leg']==k and r['same_cycle_executed']],'legacy_cf_usd_sum_not_causal':sum(float(r['cf_usd'] or 0) for r in refout if r['leg']==k)} for k in sorted({r['leg'] for r in refout})}
# Persist actual executor prompt, not a re-render under today's config.
for did in [37304,37322]:
 d=rows('select id,timestamp,system_prompt,decision_json,execution_log,plan_id,plan_version from decision_records where id=?',(did,))[0]
 (out/f'executor_{did}_system.txt').write_text(d['system_prompt'])
 (out/f'executor_{did}_result.json').write_text(json.dumps({k:v for k,v in d.items() if k!='system_prompt'},indent=2))
(out/'plan_row253.json').write_text(json.dumps(next(p['d'] for p in plans if p['plan_rowid']==253),indent=2))
strict=[d for d in decs if d['t']>=STRICT and d['t']<END and any(isinstance(x,dict) and x.get('action') in ('open_long','open_short') for x in d['items'])]
summary['strict_open_intents']={'ids':[d['id'] for d in strict], 'refused':rate(sum('refused: strict' in (d['execution_log'] or '') for d in strict),len(strict))}
# Count cumulative gate counters separately, not refusals or opportunities.
cc=rows("SELECT key,value FROM system_config WHERE key LIKE 'arm_refusals_0b:%'")
save('arm_gate_counters.csv',cc)
save('planner_facts32.csv',rows('SELECT id,trade_date,session,plan_id,version,bias_ai,bias_tree,bias_regime,tokens_in,created_at FROM planner_read_facts ORDER BY id'))
summary['facts']={'n':c.execute('select count(*) from planner_read_facts').fetchone()[0],'empty_bias_n':c.execute("select count(*) from planner_read_facts where coalesce(bias_ai,'')='' and coalesce(bias_tree,'')=''").fetchone()[0]}
(out/'summary.json').write_text(json.dumps(summary,indent=2,default=str)+'\n')
c.rollback();c.close()
print(json.dumps({k:summary[k] for k in ['population','subset47','last7','reauthor','trigger','refusals','strict_open_intents','facts']},indent=2,default=str))
