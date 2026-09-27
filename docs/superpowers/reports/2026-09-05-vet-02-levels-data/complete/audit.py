#!/usr/bin/env python3
"""SECTION 2 audit. Read-only SQLite; outputs only next to this script.
Replay is frozen level price AFTER recorded read, first completed 1m bar onwards.
Not an exact current-strategy replay, not a profitability backtest.
"""
import sqlite3,json,math,datetime,collections,bisect,pathlib,hashlib
from zoneinfo import ZoneInfo
OUT=pathlib.Path(__file__).resolve().parent
DB='file:/home/hoang/nofx/data/data.db?mode=ro'
c=sqlite3.connect(DB,uri=True);c.row_factory=sqlite3.Row;c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
ct=ZoneInfo('America/Chicago')
def clock(t): return datetime.datetime.fromtimestamp(t/1000,ct).isoformat()
def day(t):
 d=datetime.datetime.fromtimestamp(t/1000,ct)-datetime.timedelta(hours=17)
 return str((d+datetime.timedelta(days=1)).date())
def rows(q,args=()):return [dict(r) for r in c.execute(q,args)]
def save(n,x): (OUT/n).write_text(json.dumps(x,indent=2,allow_nan=False)+'\n')
def rate(k,n):
 if not n:return {'k':k,'n':n,'p':None,'wilson95':None}
 p=k/n;z=1.959963984540054;den=1+z*z/n;mid=(p+z*z/(2*n))/den;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
 return {'k':k,'n':n,'p':p,'wilson95':[mid-h,mid+h]}
pool=rows('SELECT * FROM candidate_pool ORDER BY read_at_ms,id')
touches=rows('SELECT * FROM touch_outcomes ORDER BY id')
positions=rows('SELECT id,entry_time,exit_time,status,source,plan_id,plan_version,cited_scenario_id,pnl_corrected FROM trader_positions WHERE entry_time>=1786770000000 ORDER BY id')
elig=[r for r in positions if r['status']=='CLOSED' and r['source']!='e7_farside_test' and r['plan_id'] not in (None,'','UNRESOLVABLE') and r['pnl_corrected'] is not None]
# Save row IDs, P&L and exact exclusions. No mutable stop used; no realized R claimed.
save('positions.json',{'sql':'SELECT ... FROM trader_positions WHERE entry_time>=1786770000000; eligible CLOSED, non-test, resolved nonempty plan_id, pnl_corrected NOT NULL','eligible':elig,'excluded':[r for r in positions if r not in elig],'n':len(elig),'pnl_corrected':sum(r['pnl_corrected'] for r in elig),'wins':rate(sum(r['pnl_corrected']>0 for r in elig),len(elig)),'losses':rate(sum(r['pnl_corrected']<0 for r in elig),len(elig)),'flats':rate(sum(r['pnl_corrected']==0 for r in elig),len(elig)),'cme_days':sorted(set(day(r['entry_time']) for r in elig)),'post_strict': [r for r in elig if r['entry_time']>=int(datetime.datetime(2026,9,3,11,10,tzinfo=ct).timestamp()*1000)]})
bs=rows("SELECT open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms")
times=[b['open_time_ms'] for b in bs];byday=collections.defaultdict(list)
for b in bs: byday[day(b['open_time_ms'])].append(b)
plans=rows('SELECT rowid AS row_id,plan_id,version,trade_date,session,created_at,doc FROM plans ORDER BY created_at')
for p in plans:
 try:p['doc']=json.loads(p['doc'])
 except:pass
save('candidate_rows.json',pool);save('touch_rows_quarantined.json',touches);save('plan_rows.json',plans);save('bars_1m.json',bs)
# Read-only strategy subset, never credentials/config secrets.
save('config_subset.json',rows("SELECT id,name,json_extract(config,'$.day_plan.max_levels') max_levels,json_extract(config,'$.day_plan.proximity_filter_atr') proximity_filter_atr,json_extract(config,'$.day_plan.min_grade') min_grade,json_extract(config,'$.day_plan.sessions') sessions,json_extract(config,'$.day_plan.seat_1h_zone') seat_1h_zone FROM strategies"))
inventory=[]
for k in sorted(set(x['level_kind'] for x in pool+touches)):
 ps=[x for x in pool if x['level_kind']==k];ts=[x for x in touches if x['level_kind']==k];h=sum(x['outcome']=='hold' for x in ts);b=sum(x['outcome']=='break' for x in ts);a=len(ts)-h-b
 keys=collections.defaultdict(list)
 for r in ts:keys[(r['level_price'],r['opened_at_ms'])].append(r)
 inventory.append({'kind':k,'pool_ids':[r['id'] for r in ps],'seated_ids':[r['id'] for r in ps if r['seated']],'seat_rate':rate(sum(r['seated'] for r in ps),len(ps)),'touch_ids':[r['id'] for r in ts],'raw_h_b_a':[h,b,a],'raw_hold':rate(h,h+b),'raw_break':rate(b,h+b),'raw_ambiguous':rate(a,len(ts)),'distinct_price_time_keys':len(keys),'valid_formation_adjusted_rate':'UNMEASURABLE: no per-read formed/known-at timestamp or evolving level path in touch rows; raw rates quarantined'})
save('inventory.json',inventory)
reads=[]
for t in sorted(set(r['read_at_ms'] for r in pool)):
 rs=[r for r in pool if r['read_at_ms']==t];j=bisect.bisect_right(times,t-60000)-1
 reads.append({'read_at_ms':t,'ct':clock(t),'ids':[r['id'] for r in rs],'n':len(rs),'seated':sum(r['seated'] for r in rs),'asof_price':bs[j]['c'],'price_bar_id':['MNQ','1m',times[j]],'grades':dict(collections.Counter(r['grade'] for r in rs if r['seated']))})
# Freeze level at read. Formation itself not recoverable, read proves it was known then.
# Use only trailing completed session-days >=1300 bars; threshold fixed before read.
obs=[]
for r in pool:
 t=r['read_at_ms'];sd=day(t);traildays=[d for d in sorted(byday) if d<sd and len(byday[d])>=1300][-5:];trail=[b for d in traildays for b in byday[d]]
 delta=sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/(len(trail)-1)
 start=((t+59999)//60000)*60000;end=start+60*60000
 tape=[b for b in bs if start<=b['open_time_ms']<end]
 j=bisect.bisect_right(times,t-60000)-1;px=bs[j]['c'];dist=r['level_price']-px
 x={'candidate_id':r['id'],'read_at_ms':t,'read_ct':clock(t),'level_kind':r['level_kind'],'label':r['label'],'price':r['level_price'],'seated':r['seated'],'score':r['score'] if r['seated'] else None,'grade':r['grade'] if r['seated'] else None,'distance':dist,'distance_bucket':int(abs(dist)//50)*50,'side':'above' if dist>=0 else 'below','read_price':px,'read_price_bar_ms':times[j],'start_ms':start,'end_exclusive_ms':end,'bar_count':len(tape),'delta':delta,'trail_days':traildays,'status':'right_censored_60m','touch_ms':None,'resolve_ms':None,'outcome':None}
 if len(tape)==60 and all(tape[i]['open_time_ms']==start+i*60000 for i in range(60)):
  x['status']='complete_60m';x['outcome']='untouched'
  for i,b in enumerate(tape):
   if b['l']<=r['level_price']<=b['h']:
    x['touch_ms']=b['open_time_ms'];before=bs[bisect.bisect_left(times,b['open_time_ms'])-1];entry='above' if before['c']>=r['level_price'] else 'below';x['entry_side']=entry
    follow=tape[i+1:i+13];x['outcome']='censored_12bar' if len(follow)<12 else 'ambiguous_horizon'
    for q in follow:
     exit='above' if q['c']>r['level_price']+3*delta else ('below' if q['c']<r['level_price']-3*delta else None)
     if exit:x['outcome']='hold' if exit==entry else 'break';x['resolve_ms']=q['open_time_ms'];break
    break
 obs.append(x)
save('forward_rows.json',obs)
def summary(rs):
 complete=[r for r in rs if r['status']=='complete_60m'];touched=[r for r in complete if r['touch_ms'] is not None];resolved=[r for r in touched if r['outcome'] in ('hold','break')]
 return {'candidate_ids':[r['candidate_id'] for r in rs],'complete_ids':[r['candidate_id'] for r in complete],'touched_ids':[r['candidate_id'] for r in touched],'outcome_counts':dict(collections.Counter(r['outcome'] for r in complete)),'touch':rate(len(touched),len(complete)),'hold':rate(sum(r['outcome']=='hold' for r in resolved),len(resolved)),'break':rate(sum(r['outcome']=='break' for r in resolved),len(resolved)),'ambiguous':rate(sum(r['outcome']=='ambiguous_horizon' for r in touched),len(touched))}
# match same read, same side, 50-point distance stratum; <=25pt absolute-distance caliper.
# Greedy nearest-distance matching without replacement within read, deterministic ID tie-break.
pairs=[]
for t in sorted(set(r['read_at_ms'] for r in obs)):
 seats=[r for r in obs if r['read_at_ms']==t and r['seated'] and r['status']=='complete_60m'];cuts=[r for r in obs if r['read_at_ms']==t and not r['seated'] and r['status']=='complete_60m'];used=set()
 for a in sorted(seats,key=lambda r:r['candidate_id']):
  cs=[b for b in cuts if b['candidate_id'] not in used and b['side']==a['side'] and b['distance_bucket']==a['distance_bucket'] and abs(abs(b['distance'])-abs(a['distance']))<=25]
  if cs:
   b=min(cs,key=lambda b:(abs(abs(b['distance'])-abs(a['distance'])),b['candidate_id']));used.add(b['candidate_id']);pairs.append({'seated':a,'cut':b})
save('matched_pairs.json',pairs)
first=[];seen=set()
for r in obs:
 k=(r['level_kind'],r['price'])
 if k not in seen:seen.add(k);first.append(r)
conflicts=[]
for k in set((r['level_kind'],r['level_price'],r['opened_at_ms']) for r in touches):
 rs=[r for r in touches if (r['level_kind'],r['level_price'],r['opened_at_ms'])==k]
 if len(set(r['outcome'] for r in rs))>1:conflicts.append({'key':k,'ids':[r['id'] for r in rs],'outcomes':dict(collections.Counter(r['outcome'] for r in rs))})
summary_out={'asof_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'query_only':c.execute('PRAGMA query_only').fetchone()[0],'counts':{t:c.execute('select count(*) from '+t).fetchone()[0] for t in ['candidate_pool','touch_outcomes','trade_excursions','plans','level_state']},'reads':reads,'pool_n':len(pool),'seated':sum(r['seated'] for r in pool),'cuts':sum(not r['seated'] for r in pool),'cut_reasons':dict(collections.Counter(r['cut_reason'] for r in pool if not r['seated'])),'score_components':dict(collections.Counter(r['score_components'] for r in pool)),'distinct_touch_keys':len(set((r['level_kind'],r['level_price'],r['opened_at_ms']) for r in touches)),'touch_conflicts':conflicts,'bars_bounds':[times[0],times[-1]],'all_forward':{str(s):summary([r for r in obs if r['seated']==s]) for s in [0,1]},'first_exposure':{str(s):summary([r for r in first if r['seated']==s]) for s in [0,1]},'matched_n':len(pairs),'matched':{k:summary([p[k] for p in pairs]) for k in ['seated','cut']},'matched_distinct_price_touch_keys':{k:len(set((p[k]['price'],p[k]['touch_ms']) for p in pairs if p[k]['touch_ms'])) for k in ['seated','cut']}}
save('summary.json',summary_out)
c.rollback();c.close()
print(json.dumps({k:v for k,v in summary_out.items() if k not in ['reads','all_forward','first_exposure','matched','touch_conflicts']},indent=2))
for group in ['first_exposure','matched']:
 for k,v in summary_out[group].items():print(group,k,{x:v[x] for x in ['outcome_counts','touch','hold']})
print('eligible',len(elig),sum(r['pnl_corrected'] for r in elig), 'days',len(set(day(r['entry_time']) for r in elig)))
