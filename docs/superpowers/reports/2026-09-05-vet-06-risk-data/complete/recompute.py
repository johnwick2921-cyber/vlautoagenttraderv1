#!/usr/bin/env python3
"""Section 6, corrected population. Read-only extraction; offline rerun supported.
python3 recompute.py --db /home/hoang/nofx/data/data.db --out PATH
python3 recompute.py --sample PATH/trade_sample.csv --out PATH
All outputs stay in --out. No store.New, API token, trading action or mutable risk inference.
"""
import argparse,csv,datetime as dt,hashlib,json,math,pathlib,sqlite3
from collections import defaultdict
from zoneinfo import ZoneInfo
import numpy as np
from scipy.stats import t
P=argparse.ArgumentParser(); P.add_argument('--db'); P.add_argument('--sample'); P.add_argument('--out',required=True); a=P.parse_args()
out=pathlib.Path(a.out); out.mkdir(parents=True,exist_ok=True)
CT=ZoneInfo('America/Chicago'); CUT=1786770000000; STRICT=int(dt.datetime(2026,9,3,11,10,tzinfo=CT).timestamp()*1000)
B=100000; SEED=2026090506
SQL="""SELECT id, entry_time, exit_time, created_at, symbol, side, entry_quantity,
entry_price, exit_price, pnl_corrected, fee, plan_id, plan_session, source,
close_reason, mae, mfe FROM trader_positions
WHERE entry_time >= 1786770000000 AND status='CLOSED'
AND plan_id IS NOT NULL AND TRIM(plan_id)<>'' AND plan_id<>'UNRESOLVABLE'
AND COALESCE(source,'')<>'e7_farside_test' AND pnl_corrected IS NOT NULL
ORDER BY entry_time,id"""
def time_ct(ms): return dt.datetime.fromtimestamp(int(ms)/1000,CT)
def day(ms): return (time_ct(ms)-dt.timedelta(hours=17)).date().isoformat()
def writecsv(name,rows):
 with (out/name).open('w',newline='') as f:
  w=csv.DictWriter(f,fieldnames=list(rows[0]),lineterminator='\n'); w.writeheader(); w.writerows(rows)
def writejson(name,obj): (out/name).write_text(json.dumps(obj,indent=2,allow_nan=False)+'\n')
if a.db:
 c=sqlite3.connect('file:'+a.db+'?mode=ro',uri=True); c.execute('PRAGMA query_only=ON'); c.execute('BEGIN'); c.row_factory=sqlite3.Row
 rows=[dict(r) for r in c.execute(SQL)]
 for r in rows:
  r.update(session_day_ct=day(r['entry_time']),exit_day_ct=day(r['exit_time']),entry_ct=time_ct(r['entry_time']).isoformat(),exit_ct=time_ct(r['exit_time']).isoformat(),created_ms=r['created_at'])
 era=[dict(r) for r in c.execute('SELECT id,status,source,plan_id,pnl_corrected FROM trader_positions WHERE entry_time>=? ORDER BY id',(CUT,))]
 excluded=[dict(r,reasons=[k for k,v in {'test':r['source']=='e7_farside_test','unresolvable':r['plan_id']=='UNRESOLVABLE','null_pnl':r['pnl_corrected'] is None,'missing_plan':not r['plan_id'],'not_closed':r['status']!='CLOSED'}.items() if v]) for r in era if r['id'] not in {x['id'] for x in rows}]
 meta=dict(extraction_utc=dt.datetime.now(dt.timezone.utc).isoformat(),sql=SQL,cut_ms=CUT,strict_ms=STRICT,query_only=c.execute('PRAGMA query_only').fetchone()[0],base='b4376246c2c502ecedd119c6a44a27956ed2f616',era_count=len(era),excluded=excluded,trade_excursions=c.execute('SELECT COUNT(*) FROM trade_excursions').fetchone()[0],position_rows=c.execute('SELECT COUNT(*) FROM trader_positions').fetchone()[0])
 # Values allowlist only: never export entire strategy config, which may contain credentials.
 allowed={'guardrails_enabled','daily_loss_enabled','daily_loss_limit_usd','max_daily_trades_enabled','max_daily_trades','consecutive_loss_halt','max_contracts_per_order','max_contracts_enabled','plan_mode','max_trades','name','enabled','eod_flat_ct'}
 def walk(x,p=''):
  result={}
  if isinstance(x,dict):
   for k,v in x.items():
    pp=p+'.'+k
    if k in allowed and not isinstance(v,(dict,list)): result[pp]=v
    if isinstance(v,(dict,list)): result.update(walk(v,pp))
  elif isinstance(x,list):
   for i,v in enumerate(x): result.update(walk(v,p+'['+str(i)+']'))
  return result
 meta['bound_strategies']=[dict(trader_id=r[0],strategy_id=r[1],scan_minutes=r[2],values=walk(json.loads(r[3]))) for r in c.execute('SELECT t.id,t.strategy_id,t.scan_interval_minutes,s.config FROM traders t JOIN strategies s ON s.id=t.strategy_id')]
 meta['calendar_future']=[dict(r) for r in c.execute("SELECT * FROM calendar_slices WHERE trade_date>='2026-09-05' ORDER BY trade_date")]
 c.rollback(); c.close(); writejson('extraction.json',meta); writecsv('trade_sample.csv',rows); (out/'population.sql').write_text(SQL+';\n')
else:
 rows=list(csv.DictReader(open(a.sample)))
 for r in rows:
  for k in ['id','entry_time','exit_time','created_at','created_ms']: r[k]=int(r[k])
  for k in ['pnl_corrected','fee','entry_quantity','entry_price','exit_price']: r[k]=float(r[k])
  for k in ['mae','mfe']: r[k]=float(r[k]) if r[k] not in ['', 'None'] else None
x=np.array([r['pnl_corrected'] for r in rows]); ids=[r['id'] for r in rows]
assert len(x)==58 and abs(x.sum()+466.428572)<0.00001
assert [int((x>0).sum()),int((x<0).sum()),int((x==0).sum())]==[18,38,2]
groups=defaultdict(list)
for r in rows: groups[r['session_day_ct']].append(r)
keys=sorted(groups); D=len(keys); assert D==12
dayrows=[dict(day=k,n=len(groups[k]),pnl=sum(r['pnl_corrected'] for r in groups[k]),ids=' '.join(str(r['id']) for r in groups[k])) for k in keys]
writecsv('days.csv',dayrows)
blocks=[np.array([r['pnl_corrected'] for r in groups[k]]) for k in keys]; sums=np.array([b.sum() for b in blocks]); counts=np.array([len(b) for b in blocks])
def wilson(k,n):
 z=1.959963984540054; p=k/n; mid=(p+z*z/(2*n))/(1+z*z/n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/(1+z*z/n)
 return dict(successes=int(k),n=int(n),rate=float(p),wilson95=[mid-h,mid+h])
def ddpaths(paths):
 eq=np.cumsum(paths,axis=1); peaks=np.maximum.accumulate(np.concatenate([np.zeros((len(paths),1)),eq],axis=1),axis=1)[:,1:]
 return np.max(peaks-eq,axis=1)
def streakpaths(paths):
 cur=np.zeros(len(paths),int); best=cur.copy()
 for col in paths.T: cur=np.where(col<0,cur+1,0); best=np.maximum(best,cur)
 return best
def qs(v): return dict(zip(['p50','p90','p95','p99'],map(float,np.percentile(v,[50,90,95,99]))))
def day_paths(h,seed,pairs=False):
 rng=np.random.default_rng(seed); bks=blocks if not pairs else [np.concatenate(blocks[i:i+2]) for i in range(D-1)]
 lengths=np.array([len(z) for z in bks]); table=np.zeros((len(bks),max(lengths)))
 for i,z in enumerate(bks):table[i,:len(z)]=z
 paths=np.empty((B,h)); cursor=np.zeros(B,int); used=np.zeros(B,int)
 while (cursor<h).any():
  active=np.flatnonzero(cursor<h); pick=rng.integers(len(bks),size=len(active)); lengths_now=np.minimum(lengths[pick],h-cursor[active]); starts=cursor[active].copy()
  for j in range(max(lengths_now)):
   mask=lengths_now>j; paths[active[mask],starts[mask]+j]=table[pick[mask],j]
  cursor[active]+=lengths_now; used[active]+=1
 return paths,used
mean=float(x.mean()); sd=float(x.std(ddof=1)); dmean=float(sums.mean()); dsd=float(sums.std(ddof=1))
rng=np.random.default_rng(SEED); chosen=rng.integers(D,size=(B,D)); boot_daily=sums[chosen].mean(axis=1); boot_trade=sums[chosen].sum(axis=1)/counts[chosen].sum(axis=1)
ci=lambda z:list(map(float,np.percentile(z,[2.5,97.5])))
res=dict(seed=SEED,B=B,n=len(x),D=D,ids=ids,sum=float(x.sum()),mean=mean,sd=sd,wins=wilson((x>0).sum(),len(x)),losses=wilson((x<0).sum(),len(x)),flats=wilson((x==0).sum(),len(x)),nonflat_win=wilson((x>0).sum(),(x!=0).sum()),red_days=wilson((sums<0).sum(),D),day_mean=dmean,day_sd=dsd,day_t_ci95=[dmean-v*dsd/math.sqrt(D) for v in [float(t.ppf(.975,D-1)),-float(t.ppf(.975,D-1))]],day_bootstrap_ci95=ci(boot_daily),trade_cluster_bootstrap_ci95=ci(boot_trade),trade_iid_normal_ci95=[mean-1.96*sd/math.sqrt(len(x)),mean+1.96*sd/math.sqrt(len(x))],trade_cluster_se=float(boot_trade.std(ddof=1)),bootstrap_positive_trade_fraction=wilson((boot_trade>0).sum(),B),observed_maxdd=float(ddpaths(x[None,:])[0]),observed_losing_streak=int(streakpaths(x[None,:])[0]),observed_red_day_streak=int(streakpaths(sums[None,:])[0]),worst_trade=dict(id=ids[int(x.argmin())],pnl=float(x.min())),best_trade=dict(id=ids[int(x.argmax())],pnl=float(x.max())),worst_day=dayrows[int(sums.argmin())],best_day=dayrows[int(sums.argmax())],post_strict_ids=[r['id'] for r in rows if r['entry_time']>=STRICT],cross_day_ids=[r['id'] for r in rows if r['session_day_ct']!=r['exit_day_ct']],entry_exit_order_agrees=ids==[r['id'] for r in sorted(rows,key=lambda z:(z['exit_time'],z['id']))],recorded_fee_sum=sum(r['fee'] for r in rows),mae_nonnull=sum(r['mae'] is not None for r in rows),mfe_nonnull=sum(r['mfe'] is not None for r in rows),win_average=float(x[x>0].mean()),loss_average=float(x[x<0].mean()),gross_profit_factor=float(x[x>0].sum()/-x[x<0].sum()))
res['leave_one_day_out']=[dict(omitted=keys[i],mean_per_trade=float((x.sum()-sums[i])/(len(x)-counts[i]))) for i in range(D)]
res['trade_horizons']={}
for h in [20,50,100]:
 paths,used=day_paths(h,SEED+h); dd=ddpaths(paths); st=streakpaths(paths)
 res['trade_horizons'][str(h)]=dict(drawdown=qs(dd),terminal_pnl=qs(paths.sum(axis=1)),sampled_blocks=qs(used),sampled_blocks_mean=float(used.mean()),streaks={str(k):wilson((st>=k).sum(),B) for k in [4,6,8,10]},loss_probability=wilson((paths.sum(axis=1)<0).sum(),B))
 pair,_=day_paths(h,SEED+1000+h,True); res['trade_horizons'][str(h)]['two_observed_day_pair_sensitivity']=qs(ddpaths(pair))
res['day_horizons']={}
for h in [10,20,40]:
 picks=np.random.default_rng(SEED+2000+h).integers(D,size=(B,h)); paths=sums[picks]; st=streakpaths(paths)
 res['day_horizons'][str(h)]=dict(close_to_close_dd=qs(ddpaths(paths)),red_streaks={str(k):wilson((st>=k).sum(),B) for k in [3,5]})
res['limits']={}
for limit in [225,300,450,600]:
 trips=[]; forfeited=0.; kept=0.
 for k,b in zip(keys,blocks):
  cum=np.cumsum(b); cross=np.flatnonzero(cum<=-limit)
  if len(cross):
   i=int(cross[0]); trips.append(dict(day=k,trip_id=groups[k][i]['id'],pnl_at_trip=float(cum[i]),forfeited=float(b[i+1:].sum()))); forfeited+=float(b[i+1:].sum()); kept+=float(cum[i])
  else:kept+=float(b.sum())
 res['limits'][str(limit)]=dict(trip_rate=wilson(len(trips),D),trips=trips,forfeited=forfeited,kept=kept)
weeks=defaultdict(list)
for r in rows:
 date=dt.date.fromisoformat(r['session_day_ct']); sunday=date-dt.timedelta(days=(date.weekday()+1)%7); weeks[str(sunday)].append(r)
res['weeks']=[dict(sunday_open=k,n=len(v),ids=[r['id'] for r in v],pnl=sum(r['pnl_corrected'] for r in v),worst_running_pnl=float(min(0,np.cumsum([r['pnl_corrected'] for r in v]).min()))) for k,v in sorted(weeks.items())]
res['session_descriptive']={}
for s in sorted({r['plan_session'] for r in rows}):
 rr=[r for r in rows if r['plan_session']==s]; xx=np.array([r['pnl_corrected'] for r in rr]); res['session_descriptive'][s]=dict(n=len(xx),D=len({r['session_day_ct'] for r in rr}),ids=[r['id'] for r in rr],sum=float(xx.sum()),mean=float(xx.mean()),wins=wilson((xx>0).sum(),len(xx)))
res['cost_sensitivity']=[dict(assumed_additional_round_trip_cost=c,mean=mean-c,trade_cluster_ci95=[z-c for z in res['trade_cluster_bootstrap_ci95']]) for c in [0,2,4,8]]
res['sampling']={'active_day_keys':keys,'calendar_dates_without_active_block_between_first_and_last':[str(dt.date.fromisoformat(keys[0])+dt.timedelta(days=i)) for i in range((dt.date.fromisoformat(keys[-1])-dt.date.fromisoformat(keys[0])).days+1) if str(dt.date.fromisoformat(keys[0])+dt.timedelta(days=i)) not in keys], 'trades_per_active_day_mean':float(counts.mean()),'counts':counts.tolist()}
writejson('results.json',res)
print(json.dumps({k:v for k,v in res.items() if k not in ['ids','leave_one_day_out','session_descriptive','weeks','trade_horizons','day_horizons']},indent=2))
print('TRADE HORIZONS',json.dumps(res['trade_horizons'],indent=2)); print('DAYS',json.dumps(dayrows,indent=2))
