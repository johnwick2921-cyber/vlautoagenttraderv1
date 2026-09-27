#!/usr/bin/env python3
import sqlite3,json,csv,pathlib,datetime as dt,math,statistics as st,collections as co,re,argparse
from zoneinfo import ZoneInfo
ap=argparse.ArgumentParser();ap.add_argument('--repo',required=True);ap.add_argument('--out',required=True);a=ap.parse_args();repo=pathlib.Path(a.repo);out=pathlib.Path(a.out);CT=ZoneInfo('America/Chicago')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
def save(name,rs):
 if not rs:return
 with (out/name).open('w',newline='') as f:w=csv.DictWriter(f,fieldnames=list(rs[0]),lineterminator="\n");w.writeheader();w.writerows(rs)
def rate(k,n):
 z=1.96;p=k/n;d=1+z*z/n;x=(p+z*z/(2*n))/d;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
 return {'k':k,'n':n,'pct':100*p,'wilson95':[100*(x-h),100*(x+h)]}
res={}
# Recount the archived calibration observations; do not re-label as current MNQ trading edge.
for sym in ['MNQ','ES']:
 p=repo/f'docs/superpowers/reports/2026-09-02-bias-calibration-csvs/control_{sym}_weekly.csv';rr=[{'source_line':i,**r} for i,r in enumerate(csv.DictReader(p.open()),2) if r['seg']=='holdout'];called=[r for r in rr if float(r['pos'])!=0];k=sum(float(r['pos'])==float(r['target_sign']) for r in called)
 res['weekly_'+sym]={'all_holdout_n':len(rr),'neutral_n':len(rr)-len(called),'called':rate(k,len(called)) if called else None,'raw_success_over_all':rate(k,len(rr)) if rr else None}
 save(f'weekly_{sym}_holdout.csv',rr)
p=repo/'docs/superpowers/reports/2026-09-02-live-bias-replay-csvs/calls.csv';rr=[{'source_line':i,**r} for i,r in enumerate(csv.DictReader(p.open()),2) if r['period']=='holdout'];save('bias_calls_holdout.csv',rr)
for col in ['tree','regime','composite']:
 called=[r for r in rr if r[col] in ('long','short')];k=sum((1 if r[col]=='long' else -1)==float(r['realized']) for r in called)
 res[col]={'called':rate(k,len(called)) if called else None,'neutral_n':len(rr)-len(called)}
# Replay ONLY price path opportunity examples, forward from next complete minute.
# Hypothetical entries at quoted price; not evidence the gate alone prevented a fill.
cases=[{'case':'strict_first_37304','ts':'2026-09-03 20:35:06','end':'2026-09-04 02:00:00','side':'short','entry':29541.25,'stop':29561.5,'target':29500.75,'entry_model':'assumed_market_at_quoted_price'},
{'case':'minsl_cycle27229','ts':'2026-09-03 11:48:22','end':'2026-09-03 14:45:00','side':'long','entry':29525,'stop':29482.25,'target':29616.25,'entry_model':'assumed_market_at_quoted_price'},
{'case':'leg3_LONDON_v1_S2','ts':'2026-09-04 02:00:46','end':'2026-09-04 08:30:00','side':'long','entry':29579.5,'stop':29557.84,'target':29639.5,'entry_model':'hypothetical_touch_limit_no_other_gates'},
{'case':'leg3_NY_v3_S1','ts':'2026-09-04 09:02:09','end':'2026-09-04 14:45:00','side':'short','entry':29657.38,'stop':29720,'target':29503.38,'entry_model':'hypothetical_touch_limit_no_other_gates'}]
# exact target for minsl from legacy refusal row, not a guessed literal
legacy=list(csv.DictReader((repo/'docs/superpowers/reports/2026-09-04-two-day-audit-data/refusals.csv').open()))
r=next(r for r in legacy if r['ts_ct']=='2026-09-03 11:48:22');cases[1]['target']=float(r['target_px'])
br=[];output=[]
for q in cases:
 t=dt.datetime.fromisoformat(q['ts']).replace(tzinfo=CT);end=dt.datetime.fromisoformat(q['end']).replace(tzinfo=CT);start=((int(t.timestamp()*1000)+59999)//60000)*60000;endms=int(end.timestamp()*1000)
 bs=[dict(r) for r in c.execute("select open_time_ms,o,h,l,c from bars where symbol='MNQ' and tf='1m' and open_time_ms>=? and open_time_ms+60000<=? order by open_time_ms",(start,endms))];expected=(endms-start)//60000
 entryms=start if q['entry_model'].startswith('assumed_market') else None;state='NO_TOUCH';exitms=None;exitpx=None;amb=False
 for b in bs:
  br.append({'case':q['case'],**b})
  newfill=False
  if entryms is None and b['l']<=q['entry']<=b['h']:entryms=b['open_time_ms'];newfill=True
  if entryms is None:continue
  stop=b['l']<=q['stop'] if q['side']=='long' else b['h']>=q['stop'];target=b['h']>=q['target'] if q['side']=='long' else b['l']<=q['target']
  if stop or target:
   amb=bool((stop and target) or (newfill and (stop or target)))
   state='AMBIGUOUS_INTRABAR' if amb else 'STOP' if stop else 'TARGET';exitms=b['open_time_ms'];exitpx=None if amb else q['stop'] if stop else q['target'];break
 else:
  if entryms is not None:state='HORIZON' if len(bs)==expected else 'CENSORED_FEED_END';exitms=bs[-1]['open_time_ms'] if bs else None;exitpx=bs[-1]['c'] if bs and len(bs)==expected else None
 output.append({**q,'first_eligible_bar_ms':start,'entry_bar_ms':entryms,'exit_bar_ms':exitms,'result':state,'px':exitpx,'gross_usd_proxy':(exitpx-q['entry'])*(1 if q['side']=='long' else -1)*2 if exitpx is not None else None,'bars_full_window':len(bs),'bars_expected':expected,'quality':'price-path illustration only; hypothetical execution, unknown live availability, fees/slippage and competing gates excluded; no saved-money claim'})
save('counterfactual_examples.csv',output);save('counterfactual_example_bars.csv',br);res['counterfactuals']=output
# Exact stored arms supporting cancellation counts.
ars=[dict(r) for r in c.execute("select id,plan_id,version,scenario,condition,side,entry_px,stop_px,target_px,state,state_reason,kind,placement_seq,signal_id,created_at,updated_at from armed_orders order by id")];save('arms_ledger.csv',ars)
res['flap_ids']=[r['id'] for r in ars if 62<=r['id']<=102 and r['entry_px']==29591.02];res['rr_cancel_ids']=[r['id'] for r in ars if 'gate changed: rr' in r['state_reason']];res['marketable_since0902']=[r['id'] for r in ars if 'marketable' in r['state_reason'] and dt.datetime.fromisoformat(r['updated_at']).astimezone(CT).date()>=dt.date(2026,9,2)]
# Curated local source excerpts, original path:line retained; no config or credentials.
selections={
'kernel/levels_assemble.go':[(150,189)],'kernel/levels_score.go':[(420,510),(601,615)],
'kernel/planner_prompt.go':[(178,190),(704,718),(721,733),(737,752)],
'kernel/plan_confirm.go':[(47,115),(183,222)],'kernel/entry_law.go':[(38,76)],
'kernel/engine_prompt.go':[(18,29)],'kernel/engine_prompt_futures.go':[(245,260)],
'kernel/session_registry.go':[(83,110)],'trader/entry_gate.go':[(157,220),(229,319)],
 'trader/armed_executor.go':[(315,335),(383,395),(428,467)],'trader/arm_stop_anchor.go':[(20,25),(71,89)],
 'trader/auto_trader_planner.go':[(1491,1539),(1548,1555),(1630,1668),(1671,1697)],
 'trader/auto_trader_orders.go':[(304,316)],'api/server.go':[(449,452),(592,596)],
'docs/superpowers/AUDIT-CHECKLIST.md':[(452,467),(686,752),(754,789),(944,956),(1003,1012),(1820,1843),(1845,1865)]}
lines=[]
for fp,ranges in selections.items():
 ls=(repo/fp).read_text().splitlines()
 for lo,hi in ranges:
  lines += [f'{fp}:{i}: {ls[i-1]}' for i in range(lo,min(hi,len(ls))+1)]
(out/'source_evidence.txt').write_text('\n'.join(line.rstrip() for line in lines)+'\n')
logsel={'nofx_2026-09-03.log':[(32440,32444)],'nofx_2026-09-04.log':[(1950,1950),(2158,2158),(2197,2197),(2259,2262),(6180,6182)]}
lines=[]
for fn,ranges in logsel.items():
 p=pathlib.Path('/home/hoang/nofx/data')/fn;ls=p.read_text(errors='replace').splitlines()
 for lo,hi in ranges:lines += [f'{p}:{i}: {ls[i-1]}' for i in range(lo,hi+1)]
(out/'log_evidence.txt').write_text('\n'.join(line.rstrip() for line in lines)+'\n')
(out/'supplement_summary.json').write_text(json.dumps(res,indent=2)+'\n');c.rollback();print(json.dumps(res,indent=2))
# Additional reproducible decision gate extract (independent read snapshot).
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
rr=[]
for r in c.execute("select id,timestamp,plan_id,plan_version,cited_scenario_id,execution_log,risk_check_error from decision_records where timestamp>='2026-09-02 05:00:00' and timestamp<'2026-09-05 05:00:00' order by id"):
 s=(r['execution_log'] or '')+' '+(r['risk_check_error'] or '')
 if 'entry_gate:' in s and ('refused' in s.lower() or 'below floor' in s.lower() or 'too close' in s.lower()):rr.append(dict(r))
save('decision_gate_events.csv',rr);c.rollback();c.close()
