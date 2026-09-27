import json,csv,pathlib,datetime,collections,math,statistics
from zoneinfo import ZoneInfo
R=pathlib.Path(__file__).resolve().parent;CT=ZoneInfo('America/Chicago')
def read(n):return list(csv.DictReader((R/n).open()))
def ms(s):return int(datetime.datetime.fromisoformat(s).replace(tzinfo=CT).timestamp()*1000)
def ct(t):return datetime.datetime.fromtimestamp(float(t)/1000,CT).isoformat()
def write(n,rs):
 with (R/n).open('w') as f:
  w=csv.DictWriter(f,lineterminator="\n",fieldnames=list(rs[0]));w.writeheader();w.writerows(rs)
def wil(k,n):
 if not n:return None
 p=k/n;z=1.95996398454;v=z*z/n;m=(p+v/2)/(1+v);h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/(1+v);return [k,n,100*p,100*(m-h),100*(m+h)]
ops=read('opportunities.csv');allplans=json.loads((R/'retained_plan_versions.json').read_text());arms=read('arms.csv');ints=read('intents.csv');trades=read('trades.csv');cp=read('replay_checkpoints.csv');st=read('replay_static.csv');bars=read('bars.csv')
for b in bars:
 for k in ['rowid','open_time_ms']:b[k]=int(b[k])
 for k in ['o','h','l','c','v']:b[k]=float(b[k])
d={r['opportunity']:dict(r,scope='overlapping_plan',arm_ids=[],intent_ids=[],position_ids=[],refusal_ids=[],placed_arm_ids=[],filled_arm_ids=[]) for r in ops}
def key(p,v,s):return p+'/v'+str(v)+'/'+s
for a in arms:
 k=key(a['plan_id'],a['armed_under_version'] if int(a['armed_under_version']) else a['version'],a['scenario'])
 if k not in d:
  p=next((p for p in allplans if p['plan_id']==a['plan_id'] and str(p['version'])==a['version']),None)
  sc=next((s for s in p['doc']['scenarios'] if s['id']==a['scenario']),{}) if p else {}
  d[k]=dict(opportunity=k,plan_id=a['plan_id'],version=a['version'],date=a['plan_id'][:10],session=a['session'],born=p['born'] if p else '',start=ms('2026-09-02'),end='',scenario=a['scenario'],direction=a['side'],condition=sc.get('condition',''),quality=sc.get('quality',''),arm_enabled=bool((sc.get('arm') or {}).get('enabled')),bias='',doc=json.dumps(sc),scope='carry_in_arm_only',arm_ids=[],intent_ids=[],position_ids=[],refusal_ids=[],placed_arm_ids=[],filled_arm_ids=[])
 d[k]['arm_ids'].append(a['id'])
 if a['signal_id']:d[k]['placed_arm_ids'].append(a['id'])
 if a['state']=='filled':d[k]['filled_arm_ids'].append(a['id'])
for r in ints:
 k=key(r['plan_id'],r['version'],r['scenario']);assert k in d,k;d[k]['intent_ids'].append(r['id'])
 if r['risk_check_error'] or 'refus' in r['execution_log'].lower():d[k]['refusal_ids'].append(r['id'])
for r in trades:
 k=key(r['plan_id'],r['plan_version'],r['cited_scenario_id']);assert k in d,k;d[k]['position_ids'].append(r['id'])
for r in d.values():
 for k in ['arm_ids','intent_ids','position_ids','refusal_ids','placed_arm_ids','filled_arm_ids']:r[k]=';'.join(r[k])
write('opportunity_ledger.csv',list(d.values()))
by=collections.defaultdict(list)
for r in cp:by[r['opportunity']].append(r)
result=[]
for k,rs in by.items():
 good=[r for r in rs if r['status']=='passes_known_local_checks'];ap=[r for r in good if r['actual_guard'].startswith('send')];co=[r for r in good if r['corrected_guard'].startswith('send')]
 result.append(dict(opportunity=k,checkpoints=len(rs),statuses=json.dumps(dict(collections.Counter(r['status'] for r in rs))),pass_local_count=len(good),first_pass_ct=ct(good[0]['time_ms']) if good else '',actual_send_checkpoints=len(ap),corrected_send_checkpoints=len(co),guard_values=';'.join(sorted(set(r['actual_guard'] for r in good))),first_actual_send_ct=ct(ap[0]['time_ms']) if ap else '',fills='UNMEASURABLE',pnl_points='UNMEASURABLE'))
write('replay_opportunities.csv',result)
# Tape: disjoint session windows, plus carry-in midnight fragment and final ASIA fragment.
sessions=[]
for date in ['2026-09-02','2026-09-03','2026-09-04']:
 for name,hh,mins in [('ASIA_carry','00:00',120),('LONDON','02:00',390),('NY','08:30',375),('ASIA_evening','17:00',420)]:
  a=ms(date+'T'+hh);z=a+mins*60000;x=[b for b in bars if a<=b['open_time_ms']<z]
  sessions.append(dict(date=date,session=name,start_ct=ct(a),end_exclusive_ct=ct(z),n=len(x),expected_wall_minutes=mins,first_bar_rowid=x[0]['rowid'] if x else '',last_bar_rowid=x[-1]['rowid'] if x else '',first_open=x[0]['o'] if x else '',high=max(b['h'] for b in x) if x else '',low=min(b['l'] for b in x) if x else '',last_close=x[-1]['c'] if x else '',net=x[-1]['c']-x[0]['o'] if x else '',volume=sum(b['v'] for b in x) if x else '',availability='retained final bars; original arrival time unrecorded'))
write('sessions.csv',sessions)
bef=[b for b in bars if ms('2026-09-03T08:30')<=b['open_time_ms']<ms('2026-09-03T10:00')];ib=bef[:60];v56=next(b for b in bef if b['open_time_ms']==ms('2026-09-03T09:56'));v59=bef[-1];prior=[b['v'] for b in bef if b['open_time_ms']<v56['open_time_ms']]
q2={'asof_ct':'2026-09-03 10:00:00','RTH_bars':len(bef),'RTH_open':bef[0]['o'],'RTH_known_high':max(b['h'] for b in bef),'RTH_known_low':min(b['l'] for b in bef),'last_close':v59['c'],'known_net':v59['c']-bef[0]['o'],'IB_high':max(b['h'] for b in ib),'IB_low':min(b['l'] for b in ib),'0956':v56,'0959':v59,'0956_volume_vs_prior_RTH_median':v56['v']/statistics.median(prior),'0956_volume_prior_comparison_n':len(prior),'0956_rank_of_87':1+sum(v>v56['v'] for v in prior),'RTH_vwap_proxy':sum((b['h']+b['l']+b['c'])/3*b['v'] for b in bef)/sum(b['v'] for b in bef)}
(R/'q2_asof.json').write_text(json.dumps(q2,indent=2))
# Strict is exact local routing for retained proposals. Not future LLM outputs.
strict=[dict(opportunity=key(r['plan_id'],r['version'],r['scenario']),decision_id=r['id'],current_verdict='strict_refused',source='trader/entry_gate.go:179',counterfactual_pnl='not inferred') for r in ints];write('strict_replay.csv',strict)
summary={'opportunity_n':len(d),'carry_in_only':[k for k,v in d.items() if v['scope']=='carry_in_arm_only'],'overlap_arm_enabled_opps':sum(r['arm_enabled']=='True' for r in ops),'observed_arm_opps':sum(bool(r['arm_ids']) for r in d.values()),'observed_placed_opps':sum(bool(r['placed_arm_ids']) for r in d.values()),'observed_filled_arm_opps':sum(bool(r['filled_arm_ids']) for r in d.values()),'observed_intent_opps':sum(bool(r['intent_ids']) for r in d.values()),'observed_refused_intent_opps':sum(bool(r['refusal_ids']) for r in d.values()),'observed_traded_opps':sum(bool(r['position_ids']) for r in d.values()),'local_pass_opps':sum(r['pass_local_count']>0 for r in result),'actual_send_opps':sum(r['actual_send_checkpoints']>0 for r in result),'corrected_send_opps':sum(r['corrected_send_checkpoints']>0 for r in result),'static':dict(collections.Counter(r['status'] for r in st)),'win_era':wil(18,58),'win_decisive':wil(18,56),'win_3day':wil(0,5),'observed_traded_rate':wil(sum(bool(r['position_ids']) for r in d.values()),len(d)),'trades_pnl_corrected':sum(float(r['pnl_corrected']) for r in trades)}
(R/'analysis_summary.json').write_text(json.dumps(summary,indent=2));print(json.dumps(summary,indent=2));print(json.dumps(q2,indent=2))
