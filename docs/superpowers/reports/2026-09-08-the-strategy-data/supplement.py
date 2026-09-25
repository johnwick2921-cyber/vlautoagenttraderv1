#!/usr/bin/env python3
"""Offline supplemental counts; row identities retained for every cell."""
import csv,json,pathlib,collections as c,datetime as d,zoneinfo,statistics
p=pathlib.Path(__file__).parent
load=lambda n:json.loads((p/(n+'.json')).read_text())
z=zoneinfo.ZoneInfo('America/Chicago')
s=list(csv.DictReader((p/'scenarios.csv').open()));s=[x for x in s if x['family']!='stand-aside']
a=load('arms');acc=load('accepted');plans=load('plans');t=load('tables');out={}
def group(rows,fields,key):
 g={}
 for r in rows:g.setdefault(tuple(r[f] for f in fields),[]).append(r[key])
 return [{**dict(zip(fields,k)),'n':len(v),'ids':v} for k,v in sorted(g.items())]
out['arm_enabled']=group(s,['arm_enabled','has_arm_evidence'],'scenario_key')
out['family_session_daytype']=group(s,['session','day_type_group','family'],'scenario_key')
out['scenario_levels_session_grade']=group(s,['session','level_kind','level_grade'],'scenario_key')
out['quality_side']=group(s,['quality','side'],'scenario_key')
out['authored_hour']=group(s,['hour_ct'],'scenario_key')
receipts=[]
for r in acc:
 ar=[x for x in a if x['signal_id']==r['signal_id']]
 receipts.append({'accepted_id':r['id'],'ct':d.datetime.fromtimestamp(r['accepted_at_ms']/1000,z).isoformat(),'hour_ct':d.datetime.fromtimestamp(r['accepted_at_ms']/1000,z).hour,'arm_ids':[x['id'] for x in ar],'session':ar[0]['session'] if len(ar)==1 else 'unresolved','side':r['side']})
out['acceptance_receipts']=receipts;
unique=[]
for signal in dict.fromkeys(x['signal_id'] for x in acc):
 rr=[x for x in acc if x['signal_id']==signal];first=min(rr,key=lambda x:(x['accepted_at_ms'],x['id']));r=next(x for x in receipts if x['accepted_id']==first['id']);unique.append({**r,'receipt_ids':[x['id'] for x in rr]})
out['unique_accepted_orders']=unique;out['unique_acceptance_hour']=group(unique,['hour_ct'],'accepted_id')
out['acceptance_hour']=group(receipts,['hour_ct'],'accepted_id')
out['acceptance_session_side']=group(receipts,['session','side'],'accepted_id')
out['geometry_side_errors']=[];out['first_target_nonpositive']=[];out['first_target_below_one_R']=[];out['authored_stop_below_floor']=[]
for x in s:
 if x['complete_geometry']!='True':continue
 sign=1 if x['side']=='long' else -1;e=float(x['entry']);stop=float(x['stop']);tar=float(x['target']);risk=abs(e-stop)
 if (e-stop)*sign<=0 or (tar-e)*sign<=0:out['geometry_side_errors'].append(x['scenario_key'])
 if float(x['stop_atr'])<1.5:out['authored_stop_below_floor'].append(x['scenario_key'])
 pl=next(p1 for p1 in plans if p1['row_id']==int(x['plan_row']));sc=json.loads(pl['doc'])['scenarios'][int(x['scenario_key'].rsplit(':',1)[1])]
 if sc.get('target_chain'):
  r=(sc['target_chain'][0]-e)*sign/risk
  if r<=0:out['first_target_nonpositive'].append({'id':x['scenario_key'],'signed_R':r})
  elif r<1:out['first_target_below_one_R'].append({'id':x['scenario_key'],'signed_R':r})
cohort=load('position-cohort')['eligible']
out['holds']=group([{**x,'hold_bin':'<30m' if x['hold_minutes']<30 else '30-60m' if x['hold_minutes']<=60 else '>60m'} for x in cohort],['hold_bin'],'id')
out['fill_calendar_day']=group([{**x,'date':x['entry_ct'][:10]} for x in cohort],['date'],'id')
out['quantity']=group(cohort,['entry_quantity'],'id')
out['declared_type_per_plan']=[]
for k in sorted(set(x['day_type_group'] for x in s)):
 ss=[x for x in s if x['day_type_group']==k];pp=sorted(set(x['plan_row'] for x in ss));v=[sum(x['family']=='fade' for x in ss if x['plan_row']==i)/sum(x['plan_row']==i for x in ss) for i in pp]
 out['declared_type_per_plan'].append({'type':k,'plan_n':len(pp),'scenario_n':len(ss),'mean_plan_fade_share':statistics.mean(v),'plan_ids':pp})
out['unmatched_eligible_positions']=[x['id'] for x in cohort if not any(x['id'] in json.loads(y['position_ids']) for y in s)]
for key in ['closest_pair','farthest_pair']:
 pair=load('summary')[key];out[key+'_quotes']=[{'plan_row':x['row_id'],'plan_key':x['plan_id']+':v'+str(x['version']),'created_at':x['created_at'],'day_type':json.loads(x['doc']).get('day_type'),'bias':json.loads(x['doc']).get('bias'),'reasoning':json.loads(x['doc']).get('reasoning'),'scenarios':json.loads(x['doc']).get('scenarios')} for x in plans if x['row_id'] in pair['a_plan_rows']+pair['b_plan_rows']]
(p/'supplement.json').write_text(json.dumps(out,indent=2)+'\n')
print(json.dumps({k:v for k,v in out.items() if k in ['geometry_side_errors','declared_type_per_plan','unmatched_eligible_positions']},indent=2))
print('receipt hours',out['acceptance_hour']);print('nonpositive T1',len(out['first_target_nonpositive']),'sub1 T1',len(out['first_target_below_one_R']))
# Expand the exact arm geometry consumed by the executor: explicit legs replace
# the parent; otherwise the parent is the single leg. Preserve both views.
legs=[]
for x in s:
 pl=next(p1 for p1 in plans if p1['row_id']==int(x['plan_row']));sc=json.loads(pl['doc'])['scenarios'][int(x['scenario_key'].rsplit(':',1)[1])];a=sc.get('arm') or {}
 if not a.get('enabled'):continue
 for i,leg in enumerate(a.get('legs') or [a]):
  e,sl,tp=(leg.get(k) for k in ['entry','stop','target'])
  if not all(isinstance(v,(int,float)) and v>0 for v in [e,sl,tp]) or e==sl:continue
  risk=abs(e-sl);atr=float(x['atr5m_saved']) if x['atr5m_saved'] else None;sign=1 if x['side']=='long' else -1
  legs.append({'id':x['scenario_key']+':leg'+str(i),'scenario_key':x['scenario_key'],'plan_row':x['plan_row'],'leg_index':i,'explicit_split':bool(a.get('legs')),'kind':leg.get('kind','unspecified'),'stop_pts':risk,'stop_atr':risk/atr if atr else None,'target_R':(tp-e)*sign/risk,'correct_stop_side':(e-sl)*sign>0})
out['effective_legs']=legs
out['effective_leg_stats']={}
for k in ['stop_pts','stop_atr','target_R']:
 good=[x for x in legs if x[k] is not None];v=sorted(x[k] for x in good)
 def q(f):
  a=(len(v)-1)*f;i=int(a);return v[i]+(v[min(i+1,len(v)-1)]-v[i])*(a-i)
 out['effective_leg_stats'][k]={'n':len(v),'ids':[x['id'] for x in good],'min':min(v),'p25':q(.25),'median':statistics.median(v),'p75':q(.75),'max':max(v)}
out['effective_legs_below_floor']=[x['id'] for x in legs if x['stop_atr'] is not None and x['stop_atr']<1.5]
(p/'supplement.json').write_text(json.dumps(out,indent=2)+'\n')
print('effective leg stats', {k:{a:b for a,b in v.items() if a!='ids'} for k,v in out['effective_leg_stats'].items()})
