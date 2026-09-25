"""Section 5 documentary measurements only: explicit eligible ids and uncertainty."""
import sqlite3,json,collections,datetime,math,csv,pathlib
from zoneinfo import ZoneInfo
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
x=json.load(open('q31_verified.json'));z=json.load(open('q34_integration.json'));o={}
def rows(q,a=()):return [dict(r) for r in c.execute(q,a)]
def rate(ids,den):
 k=len(ids);n=len(den)
 if not n:return dict(ids=ids,denominator_ids=den,k=k,n=n,pct=None,wilson95=None)
 p=k/n;zz=1.96;d=1+zz*zz/n;m=(p+zz*zz/(2*n))/d;h=zz*math.sqrt(p*(1-p)/n+zz*zz/(4*n*n))/d
 return dict(ids=ids,denominator_ids=den,k=k,n=n,pct=100*p,wilson95=[max(0,100*(m-h)),min(100,100*(m+h))])
ps=x['positions'];ids=[p['id'] for p in ps]
era=rows('select id,plan_id,source,pnl_corrected,entry_time,close_reason from trader_positions where entry_time>=1786770000000 and entry_time<1788584400000 order by id')
o['read_at_ct']=datetime.datetime.now(ZoneInfo('America/Chicago')).isoformat();o['excluded']=[r for r in era if r['id'] not in ids]
assert {r['id'] for r in o['excluded']}=={530,539,545,546,566,571,580,572,573,574,576,577,579}
assert len(era)==71 and len(ids)==58 and z['cme_days']==12
p=x['performance'];o['performance']=p.copy();o['performance']['all_row_win_rate']=rate(p['winner_ids'],ids)
o['performance']['breakeven_nonflat_win_probability_at_observed_payoff']=1/(1+p['payoff'])
o['performance']['payoff_required_at_observed_nonflat_win_frequency']=len(p['loser_ids'])/len(p['winner_ids'])
o['performance']['initial_risk_R_distribution']='UNMEASURABLE: immutable initial broker-accepted stops absent across cohort'
o['fill_provenance']={}
for leg in ['entry','exit']:
 g=[r for r in x['fill_rows'] if r['leg']==leg]
 methods=collections.defaultdict(list)
 for r in g:methods[r['method'].split(':')[0]].append(r['position_id'])
 o['fill_provenance'][leg]={'methods':dict(methods),'missing_bar_ids':[r['position_id'] for r in g if not r['bar']],'outside_ids':[r['position_id'] for r in g if r['bar'] and not r['inside']]}
# Per-position entry AND exit evidence: a matched bar bucket is not exact execution time.
with open('eligible_fill_audit.csv','w') as f:
 fields=['position_id','leg','price','method','fill_id','trade_id','time_used_ms','bar_id','bar_open_time_ms','bar_low','bar_high','inside','adverse_extreme','favorable_extreme']
 w=csv.DictWriter(f,fields);w.writeheader()
 for r in x['fill_rows']:
  b=r['bar'] or {};v={k:r.get(k) for k in fields};v.update(bar_id=b.get('bar_id'),bar_open_time_ms=b.get('open_time_ms'),bar_low=b.get('l'),bar_high=b.get('h'));w.writerow(v)
# Each exit label is included even if P&L sign differs.
o['exit_reason_pnl']={}
for label,g in x['exit_reasons'].items():
 gg=[r for r in ps if r['id'] in g['ids']];o['exit_reason_pnl'][label]={'ids':g['ids'],'n':len(gg),'pnl_corrected':sum(r['pnl_corrected'] for r in gg),'winner_ids':[r['id'] for r in gg if r['pnl_corrected']>0],'flat_ids':[r['id'] for r in gg if r['pnl_corrected']==0]}
# Deduplicate versioned scenarios only for stable plan-id/scenario identity. Do not pretend versions are independent.
sc=x['funnel_scenarios'];enabled={ (s['plan_id'],s['scenario']) for s in sc if s['arm_enabled']};ars=x['funnel_arms'];aligned=[r for r in ars if r['id'] in x['aligned_plan_dates']['row_ids']]
key=lambda r:(r['plan_id'],r['scenario'])
armed=set(map(key,aligned));placed={key(r) for r in aligned if r['signal_id']};filled={key(r) for r in aligned if r['state']=='filled'}
assert armed<=enabled
hits=[r['arm_id'] for r in x['reached_proxy'] if r['touch_bar_ids'] and r['arm_id'] in x['aligned_plan_dates']['row_ids']]
reached={key(r) for r in aligned if r['id'] in hits}
o['funnel']={'plan_versions':len({(s['plan_id'],s['version']) for s in sc}),'plan_ids':sorted({s['plan_id'] for s in sc}),'scenario_versions':len(sc),'enabled_versions':sum(s['arm_enabled'] for s in sc),'enabled_keys':sorted(enabled),'armed_keys':sorted(armed),'placed_keys':sorted(placed),'reached_proxy_keys':sorted(reached),'filled_keys':sorted(filled),'unarmed_keys':sorted(enabled-armed),'placed_to_fill':rate(sorted(filled),sorted(placed)),'armed_to_place':rate(sorted(placed),sorted(armed)),'authored_to_arm':rate(sorted(armed),sorted(enabled)),'placed_to_reached_proxy':rate(sorted(reached),sorted(placed)),'reached_proxy_to_fill':rate(sorted(filled),sorted(reached)),'filled_to_win':rate([],sorted(filled)),'whole_bar_hit_arm_ids':hits,'stop_entry_sent_ids':[r['id'] for r in aligned if r['signal_id'] and r['kind']=='stop_entry']}
# Boundary-inclusive sensitivity is not a valid broker-life election test either.
bound=[]
for a in aligned:
 if not a['signal_id']:continue
 trigger=a['entry_px']+(.5 if a['side'].lower()=='long' else -.5) if a['kind']=='stop_entry' else a['entry_px']
 bs=rows("select rowid as bar_id,open_time_ms,h,l from bars where symbol='MNQ' and tf='1m' and open_time_ms<? and open_time_ms+60000>? and l<=? and h>=?",(a['updated_ms'],a['created_ms'],trigger,trigger))
 if bs:bound.append({'arm_id':a['id'],'key':key(a),'bar_ids':[b['bar_id'] for b in bs]})
bkeys={tuple(r['key']) for r in bound}
o['funnel']['boundary_inclusive_hits']=bound
o['funnel']['boundary_inclusive_identity_rate']=rate(sorted(bkeys),sorted(placed))
# Gather exact arm33 bar proxy. Not a quote at decision time.
a=rows('select id,created_at,updated_at,side,entry_px,state_reason from armed_orders where id=33')[0]
t=int(datetime.datetime.fromisoformat(a['updated_at']).astimezone(datetime.timezone.utc).timestamp()*1000)
o['guard33']={'arm':a,'decision_minute_proxy':rows("select rowid as bar_id,open_time_ms,o,h,l,c from bars where symbol='MNQ' and tf='1m' and open_time_ms<=? and open_time_ms+60000>?",(t,t))}
o['plan35']=rows("select plan_id,version,created_at,session,doc from plans where plan_id=? and version=2",(next(a['plan_id'] for a in ars if a['id']==35),))
o['snapshot_ids']=rows('select id,received_at_ms,reason,orders_json from nt8_order_snapshots where id in (1604,1605,1606)')
# Only this trade has independently inspected immutable broker stop evidence here.
e=29285.;stop=29355.;target=29144.5;risk=stop-e;reward=e-target
first=29233.08
o['arm35_geometry']={'arm_id':35,'position_id':591,'entry':e,'accepted_initial_stop':stop,'target':target,'risk_pts':risk,'reward_pts':reward,'rr_accepted':reward/risk,'first_target_pts':e-first,'first_target_R':(e-first)/risk,'max_adverse_entry_pts_at_2R':(reward-2*risk)/3,'rr_one_adverse_tick':(reward-.25)/(risk+.25),'rr_four_adverse_ticks':(reward-1)/(risk+1),'invalidation_5m_bar_id':440248,'invalidation_close_proxy':29301.5,'proxy_loss_pts_at_invalidation_close':29301.5-e,'broker_stop_slippage_pts':0,'ledger_drift_pts':z['ledger_to_broker_geometry_drift_pts']}
pathlib.Path('q35_complete.json').write_text(json.dumps(o,indent=2)+'\n')
for k in ['performance','excluded','fill_provenance','exit_reason_pnl','funnel','guard33','arm35_geometry']:print(k,json.dumps(o[k]))
