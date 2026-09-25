#!/usr/bin/env python3
"""Read-only extraction: python3 audit.py --extract. Offline reproduce: python3 audit.py.
Only stdlib; run in assigned scratch. Never imports trading code or initializes store.
"""
import sqlite3,json,csv,math,statistics as st,datetime as dt,random,gzip,sys,re
from pathlib import Path
from collections import defaultdict,Counter
from zoneinfo import ZoneInfo
CT=ZoneInfo('America/Chicago'); UTC=dt.timezone.utc; ERA=1786770000000
ZERO_B=int(dt.datetime(2026,9,2,7,49,6,tzinfo=CT).timestamp()*1000)
STRICT=int(dt.datetime(2026,9,3,11,10,33,tzinfo=CT).timestamp()*1000)
def time_ms(s):
    if not s:return None
    x=dt.datetime.fromisoformat(s.replace('Z','+00:00'))
    if x.tzinfo is None:x=x.replace(tzinfo=UTC)
    return int(x.timestamp()*1000)
def ct(ms):return dt.datetime.fromtimestamp(ms/1000,CT)
def day(ms):
    t=ct(ms);return (t.date()-dt.timedelta(days=t.hour<17)).isoformat()
def pct(xs,p):
    xs=sorted(xs)
    if not xs:return None
    k=(len(xs)-1)*p;i=int(k);return xs[i]+(xs[min(i+1,len(xs)-1)]-xs[i])*(k-i)
def wil(k,n):
    if not n:return None
    p=k/n;z=1.96;d=1+z*z/n;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return [p,(p+z*z/(2*n)-h)/d,(p+z*z/(2*n)+h)/d]
def writecsv(name,rows):
    if not rows:return
    with open(name,'w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=list(rows[0]),lineterminator="\n");w.writeheader();w.writerows(rows)
if '--extract' in sys.argv:
    c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
    c.execute('PRAGMA query_only=ON');c.execute('BEGIN');c.row_factory=sqlite3.Row
    queries={
      'positions':f'SELECT * FROM trader_positions WHERE entry_time >= {ERA} ORDER BY id',
      'plans':'SELECT rowid AS row_id,plan_id,version,session,trade_date,created_at,doc FROM plans ORDER BY rowid',
      'touches':'SELECT * FROM touch_outcomes ORDER BY id',
      'excursions':'SELECT * FROM trade_excursions ORDER BY id',
      'facts':'SELECT id,trader_id,trade_date,session,plan_id,version,bias_regime,bias_tree,bias_ai,atr5m,stop_floor_pts,created_at FROM planner_read_facts ORDER BY id',
      'bars':f"SELECT rowid AS row_id,* FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms>={ERA-86400000*5} ORDER BY open_time_ms,convention",
      'arms':'SELECT * FROM armed_orders ORDER BY id',
      'snapshots':'SELECT COUNT(*) AS n,MIN(received_at_ms) AS first_ms,MAX(received_at_ms) AS last_ms FROM nt8_order_snapshots'}
    raw={k:[dict(r) for r in c.execute(q)] for k,q in queries.items()}
    for p in raw['plans']:
        d=json.loads(p.pop('doc'));p['doc']={k:d.get(k) for k in ['scenarios','levels','bias','bias_label','day_type']}
        for sc in p['doc'].get('scenarios') or []:
            for k in list(sc):
                if k not in ['id','condition','direction','confirm','confirm2','arm','target_chain','quality']:sc.pop(k)
    raw['meta']={'asof_ct':dt.datetime.now(CT).isoformat(),'era_ms':ERA,'query_only':c.execute('PRAGMA query_only').fetchone()[0],'queries':queries,'base_revision':'b4376246c2c502ecedd119c6a44a27956ed2f616'}
    c.rollback();c.close()
    br=[]; manifests=[];root=Path('/mnt/c/Users/hoang/Documents/NinjaTrader 8/log')
    for p in sorted(root.glob('log.2026*.txt')):
        if '.en.' in p.name or p.name<'log.20260815':continue
        lines=p.read_text(errors='replace').splitlines();count=0
        for lineno,line in enumerate(lines,1):
            if "New state='" not in line or "Instrument='MNQ " not in line:continue
            try:ms=int(dt.datetime.strptime(line[:23],'%Y-%m-%d %H:%M:%S:%f').replace(tzinfo=CT).timestamp()*1000)
            except ValueError:continue
            def s(k):
                m=re.search(re.escape(k)+r"='([^']*)'",line);return m.group(1) if m else ''
            def n(k):
                m=re.search(re.escape(k)+r'=([\d.]+)',line);return float(m.group(1)) if m else None
            br.append(dict(ms=ms,order=s('Order').split('/')[0],name=s('Name'),state=s('New state'),oco=s('Oco'),action=s('Action'),stop=n('Stop price'),limit=n('Limit price'),quantity=n('Quantity'),filled=n('Filled'),fill=n('Fill price'),path=str(p),line=lineno));count+=1
        manifests.append({'path':str(p),'order_rows':count})
    raw['broker']=br;raw['broker_files']=manifests
    with gzip.open('inputs.json.gz','wt') as f:json.dump(raw,f)
else:
    with gzip.open('inputs.json.gz','rt') as f:raw=json.load(f)
P={(p['plan_id'],p['version']):p for p in raw['plans']}
T=[];excluded=[]
for t0 in raw['positions']:
    t=dict(t0);why=[]
    if t['source']=='e7_farside_test':why.append('test seam')
    if t['plan_id']=='UNRESOLVABLE':why.append('sentinel plan')
    if t['pnl_corrected'] is None:why.append('null corrected P&L')
    if why:excluded.append({'id':t['id'],'reasons':'; '.join(why),'pnl_corrected':t['pnl_corrected']});continue
    p=P.get((t['plan_id'],t['plan_version']));sc=next((s for s in (p['doc']['scenarios'] or []) if s['id']==t['cited_scenario_id']),{}) if p else {}
    t.update(condition=sc.get('condition','UNMATCHED'),plan_row_id=p['row_id'] if p else None,plan_created_ms=time_ms(p['created_at']) if p else None,confirm_rule=(sc.get('confirm') or {}).get('rule'),day_type=((p or {}).get('doc') or {}).get('day_type'),bias_label=((p or {}).get('doc') or {}).get('bias_label'),hour=ct(t['entry_time']).hour,cme_day=day(t['entry_time']),hold_min=(t['exit_time']-t['entry_time'])/60000,entry_ct=ct(t['entry_time']).isoformat(),era='pre_0B' if t['entry_time']<ZERO_B else '0B_pre_strict' if t['entry_time']<STRICT else 'strict',scenario_arm=sc.get('arm') or {},target_chain=sc.get('target_chain') or [])
    t['path']='arm-associated' if t['source']!='system' else 'decision'
    t['thesis']='range_rejection' if t['condition']=='reject' else 'failed_break_reversal' if t['condition']=='sweep_reclaim' else 'continuation_candidate' if t['condition'] in ['reclaim','breakout_retest','acceptance','breakdown_continue','breakup_continue'] else 'hold_other'
    t.update(realized_R=None,risk_reason='no immutable entry/OCO/initial accepted stop linkage',broker_stop_initial=None,broker_target_initial=None,broker_signal='',broker_stop_ref='',broker_entry_ref='')
    T.append(t)
assert len(T)==58 and abs(sum(t['pnl_corrected'] for t in T)+466.428572)<1e-5
assert Counter('W' if t['pnl_corrected']>0 else 'L' if t['pnl_corrected']<0 else 'F' for t in T)==Counter(W=18,L=38,F=2)
assert len(set(t['cme_day'] for t in T))==12
br=raw['broker'];byname=defaultdict(list);byorder=defaultdict(list)
for b in br:byname[b['name']].append(b);byorder[b['order']].append(b)
# R requires exact identity linkage, first Accepted protective stop within 10s of a matched Filled entry.
for t in T:
    sig=t['entry_order_id'] if t['entry_order_id'] not in ['<nil>','',None] else ''
    if not sig:
        cand={b['name'][:-3] for b in byorder.get(t['exit_order_id'],[]) if b['name'].endswith(('-sl','-tp'))}
        if len(cand)==1:sig=cand.pop()
    entries=[b for b in byname.get(sig,[]) if b['state']=='Filled' and b['quantity']==1 and b['filled']==1 and abs((b['fill'] or 0)-t['entry_price'])<1e-6 and abs(b['ms']-t['entry_time'])<180000]
    if not entries:continue
    ent=min(entries,key=lambda x:x['ms'])
    stops=[b for b in byname.get(sig+'-sl',[]) if b['state']=='Accepted' and b['stop'] and b['quantity']==1 and ent['ms']<=b['ms']<=ent['ms']+10000]
    if not stops:continue
    sl=min(stops,key=lambda x:x['ms']);sign=1 if t['side']=='LONG' else -1;risk=(t['entry_price']-sl['stop'])*sign
    if risk<=0:continue
    t.update(broker_signal=sig,broker_stop_initial=sl['stop'],realized_R=t['pnl_corrected']/(risk*2*t['entry_quantity']),risk_reason='exact broker entry fill plus first Accepted protective stop within 10s',broker_stop_ref=f"{sl['path']}:{sl['line']}",broker_entry_ref=f"{ent['path']}:{ent['line']}")
    targets=[b for b in byname.get(sig+'-tp',[]) if b['state']=='Accepted' and b['limit'] and ent['ms']<=b['ms']<=ent['ms']+10000]
    if targets:t['broker_target_initial']=min(targets,key=lambda x:x['ms'])['limit']
bg=defaultdict(list)
for b in raw['bars']:bg[b['open_time_ms']].append(b)
bars={};conflict=[]
for ms,g in bg.items():
    vals=set((x['o'],x['h'],x['l'],x['c']) for x in g)
    if len(vals)>1:conflict.append(ms);continue
    bars[ms]=min(g,key=lambda x:(x['convention']!='epoch_floor',x['row_id']))
for t in T:
    last=t['entry_time']//60000*60000-60000;bb=[bars.get(last-i*60000) for i in range(60,-1,-1)]
    t.update(rv60_bps=None,rv_bar_ids='',rv_reason='missing/conflicting 1m bar in 61-close pre-entry window',vol_tercile='UNMEASURABLE')
    if all(bb):
        lr=[math.log(bb[i]['c']/bb[i-1]['c']) for i in range(1,61)]
        t.update(rv60_bps=10000*math.sqrt(sum(x*x for x in lr)),rv_bar_ids=','.join(str(b['row_id']) for b in bb),rv_reason='61 consecutive closed 1m closes; no future bars')
vals=[t['rv60_bps'] for t in T if t['rv60_bps'] is not None];cuts=[pct(vals,1/3),pct(vals,2/3)]
for t in T:
    if t['rv60_bps'] is not None:t['vol_tercile']='low' if t['rv60_bps']<=cuts[0] else 'mid' if t['rv60_bps']<=cuts[1] else 'high'
    exact=[r for r in raw['facts'] if r['plan_id']==t['plan_id'] and r['version']==t['plan_version'] and time_ms(r['created_at'])<=t['entry_time']]
    t['machine_regime']=exact[-1]['bias_regime'] if exact else 'UNMEASURABLE';t['machine_fact_id']=exact[-1]['id'] if exact else None
    near=[r for r in raw['facts'] if r['session']==t['plan_session'] and 0<=t['entry_time']-time_ms(r['created_at'])<=1800000]
    t['regime_asof_sensitivity']=near[-1]['bias_regime'] if near else 'UNMEASURABLE';t['regime_asof_fact_id']=near[-1]['id'] if near else None
S=[]
def stat(dim,key,ts):
    xs=[t['pnl_corrected'] for t in ts];n=len(ts);w=sum(x>0 for x in xs);l=sum(x<0 for x in xs);rs=[t['realized_R'] for t in ts if t['realized_R'] is not None];ci=wil(w,w+l)
    s={'dimension':dim,'cell':key,'n':n,'W':w,'L':l,'F':n-w-l,'sum_usd':sum(xs),'mean_usd':st.mean(xs) if xs else None,'mean_normal95_lo':st.mean(xs)-1.96*st.stdev(xs)/math.sqrt(n) if n>1 else None,'mean_normal95_hi':st.mean(xs)+1.96*st.stdev(xs)/math.sqrt(n) if n>1 else None,'win_decided':w+l,'win_rate':ci[0] if ci else None,'wilson_lo':ci[1] if ci else None,'wilson_hi':ci[2] if ci else None,'R_n':len(rs),'R_mean':st.mean(rs) if rs else None,'R_p50':pct(rs,.5),'hold_p25':pct([t['hold_min'] for t in ts],.25),'hold_p50':pct([t['hold_min'] for t in ts],.5),'hold_p75':pct([t['hold_min'] for t in ts],.75),'days':len(set(t['cme_day'] for t in ts)),'verdict':'NO VERDICT: n<30' if n<30 else 'descriptive; no demonstrated positive edge','ids':','.join(str(t['id']) for t in ts),'R_ids':','.join(str(t['id']) for t in ts if t['realized_R'] is not None)};S.append(s);return s
stat('book','ALL',T)
for dim in ['condition','side','plan_session','hour','confirm_rule','path','thesis','era','cme_day','machine_regime','regime_asof_sensitivity','day_type','vol_tercile']:
    for v in sorted(set(t[dim] for t in T),key=str):stat(dim,str(v),[t for t in T if t[dim]==v])
stat('era','strict',[])
for name,lo,hi in [('0-15',0,15),('15-30',15,30),('30-60',30,60),('60+',60,math.inf)]:stat('hold_bucket',name,[t for t in T if lo<=t['hold_min']<hi])
for session in ['ASIA','LONDON','NY']:stat('reject_session',session,[t for t in T if t['condition']=='reject' and t['plan_session']==session])
for vol in ['low','mid','high','UNMEASURABLE']:
    for ses in ['ASIA','LONDON','NY']:stat('vol_session',vol+'/'+ses,[t for t in T if t['vol_tercile']==vol and t['plan_session']==ses])
writecsv('trade_stats.csv',S)
random.seed(90501);ds=defaultdict(list)
for t in T:ds[t['cme_day']].append(t)
days=list(ds);boot=[]
for _ in range(20000):
    chosen=[random.choice(days) for _ in days];xx=[t['pnl_corrected'] for d in chosen for t in ds[d]];boot.append(sum(xx)/len(xx))
G=defaultdict(list)
for r in raw['touches']:G[(r['trader_id'],r['symbol'],r['level_kind'],r['level_price'],r['opened_at_ms'])].append(r)
D=[]
for key,g in G.items():
    outs=set(r['outcome'] for r in g);out=next(iter(outs)) if len(outs)==1 else 'ambiguous_conflict';r=min(g,key=lambda x:x['id']);p=P.get((r['plan_id'],r['plan_version']));created=time_ms(p['created_at']) if p else None
    D.append(dict(representative_id=r['id'],kind=r['level_kind'],price=r['level_price'],opened_at_ms=r['opened_at_ms'],opened_ct=ct(r['opened_at_ms']).isoformat(),cme_day=day(r['opened_at_ms']),outcome=out,raw_n=len(g),raw_ids=','.join(str(x['id']) for x in g),stored_ordinals=','.join(str(x) for x in sorted(set(x['ordinal'] for x in g))),stored_ordinal=r['ordinal'],reconstructed_ordinal=None,ordinal_bucket=None,first_plan_row=p['row_id'] if p else None,plan_created_ms=created,plan_asof_ok=created is not None and created<=r['opened_at_ms']))
seq=Counter()
for d in sorted(D,key=lambda x:(x['opened_at_ms'],x['representative_id'])):
    k=(d['kind'],d['price'],d['cme_day']);seq[k]+=1;d['reconstructed_ordinal']=seq[k];d['ordinal_bucket']=str(seq[k]) if seq[k]<3 else '3+'
D.sort(key=lambda x:x['representative_id']);writecsv('touch_keys.csv',D);TS=[]
def touchstat(dim,key,rr):
    h=sum(x['outcome']=='hold' for x in rr);b=sum(x['outcome']=='break' for x in rr);a=len(rr)-h-b;n=h+b;wh=wil(h,n);wb=wil(b,n);wa=wil(a,len(rr))
    TS.append(dict(dimension=dim,cell=key,all_n=len(rr),hold=h,brk=b,amb=a,decided_n=n,hold_rate=wh[0] if wh else None,hold_lo=wh[1] if wh else None,hold_hi=wh[2] if wh else None,break_rate=wb[0] if wb else None,break_lo=wb[1] if wb else None,break_hi=wb[2] if wb else None,amb_rate=wa[0] if wa else None,amb_lo=wa[1] if wa else None,amb_hi=wa[2] if wa else None,verdict='NO VERDICT: n<30' if n<30 else 'NO EDGE VERDICT: retrospective formation/selection',key_ids=','.join(str(x['representative_id']) for x in rr)))
touchstat('all','all',D)
for kind in sorted(set(x['kind'] for x in D)):
    sub=[x for x in D if x['kind']==kind];touchstat('kind',kind,sub)
    for ord in ['1','2','3+']:touchstat('kind_ordinal',kind+'/'+ord,[x for x in sub if x['ordinal_bucket']==ord])
for ord in ['1','2','3+']:touchstat('ordinal',ord,[x for x in D if x['ordinal_bucket']==ord])
for kind in sorted(set(x['level_kind'] for x in raw['touches'])):
    for ord in sorted(set(x['ordinal'] for x in raw['touches'] if x['level_kind']==kind)):
        rr=[dict(outcome=x['outcome'],representative_id=x['id']) for x in raw['touches'] if x['level_kind']==kind and x['ordinal']==ord];touchstat('raw_kind_stored_ordinal',kind+'/'+str(ord),rr)
writecsv('touch_stats.csv',TS)
PS=[]
for p in raw['plans']:
    if time_ms(p['created_at'])<ERA:continue
    levels=p['doc']['levels'] or []
    for s in p['doc']['scenarios'] or []:
        a=s.get('arm') or {};e=a.get('entry');sl=a.get('stop');tp=a.get('target')
        if not (e and sl and tp and e!=sl):continue
        sign=1 if s.get('direction')=='long' else -1
        rr=(tp-e)*sign/((e-sl)*sign) if (e-sl)*sign>0 else None
        PS.append(dict(plan_row_id=p['row_id'],plan_id=p['plan_id'],version=p['version'],scenario=s['id'],created_at=p['created_at'],session=p['session'],condition=s['condition'],enabled=a.get('enabled',False),entry=e,stop=sl,target=tp,rr=rr,target_at_plan_level=any(abs(tp-l['price'])<=.25 for l in levels),target_in_chain=any(abs(tp-x)<=.25 for x in s.get('target_chain') or [])))
writecsv('planned_arms.csv',PS);EX=[]
for condition in sorted(set(t['condition'] for t in T)):
    ts=[t for t in T if t['condition']==condition];m=[t['mfe'] for t in ts if t['mfe'] is not None];a=[t['mae'] for t in ts if t['mae'] is not None]
    EX.append(dict(condition=condition,n=len(ts),mfe_n=len(m),mae_n=len(a),mfe_p50=pct(m,.5),mfe_p80=pct(m,.8),mfe_p95=pct(m,.95),mae_p50=pct(a,.5),mae_p80=pct(a,.8),mae_p95=pct(a,.95),ids=','.join(str(t['id']) for t in ts),status='position proxies; unordered/censored, not trade_excursions'))
writecsv('excursion_proxies.csv',EX)
for t in T:
    a=t['scenario_arm'];e=a.get('entry');sl=a.get('stop');tp=a.get('target')
    t['authored_RR']=abs(tp-e)/abs(e-sl) if e and sl and tp and e!=sl else None
    t['proxy_reached_authored_target']=t['mfe']>=abs(tp-t['entry_price']) if tp and t['mfe'] is not None else None
cols=['id','plan_row_id','plan_id','plan_version','cited_scenario_id','plan_created_ms','condition','side','plan_session','entry_ct','entry_time','exit_time','hour','cme_day','hold_min','pnl_corrected','fee','entry_quantity','source','path','thesis','confirm_rule','era','entry_price','exit_price','entry_order_id','exit_order_id','close_reason','broker_signal','broker_stop_initial','broker_target_initial','realized_R','risk_reason','broker_stop_ref','broker_entry_ref','rv60_bps','vol_tercile','rv_reason','rv_bar_ids','machine_regime','machine_fact_id','regime_asof_sensitivity','regime_asof_fact_id','bias_label','day_type','mae','mfe','authored_RR','proxy_reached_authored_target']
writecsv('trades.csv',[{k:t[k] for k in cols} for t in T]);writecsv('excluded.csv',excluded)
w=[t['pnl_corrected'] for t in T if t['pnl_corrected']>0];l=[t['pnl_corrected'] for t in T if t['pnl_corrected']<0];pay=st.mean(w)/abs(st.mean(l));se=st.stdev([t['pnl_corrected'] for t in T])/math.sqrt(len(T));mean=st.mean([t['pnl_corrected'] for t in T])
summary=dict(meta=raw['meta'],eligible_n=len(T),ids=[t['id'] for t in T],excluded=excluded,sum_usd=sum(t['pnl_corrected'] for t in T),mean_usd=mean,avg_win=st.mean(w),avg_loss=st.mean(l),payoff=pay,break_even_decided_win=1/(1+pay),win_wilson=wil(len(w),len(w)+len(l)),mean_naive_normal95=[mean-1.96*se,mean+1.96*se],cme_days=len(days),trades_per_occupied_day=len(T)/len(days),day_bootstrap95=[pct(boot,.025),pct(boot,.975)],bootstrap_draws=20000,seed=90501,strict_ms=STRICT,zeroB_ms=ZERO_B,trade_excursions_n=len(raw['excursions']),broker_R_ids=[t['id'] for t in T if t['realized_R'] is not None],R_missing_ids=[t['id'] for t in T if t['realized_R'] is None],rv_cuts_bps=cuts,rv_n=len(vals),rv_missing_ids=[t['id'] for t in T if t['rv60_bps'] is None],bar_conflicting_times=conflict,touches_raw=len(raw['touches']),touch_price_time_keys=len(D),touch_kind_time_only_keys=len(set((x['level_kind'],x['opened_at_ms']) for x in raw['touches'])),touch_conflicting_keys=[d['representative_id'] for d in D if d['outcome']=='ambiguous_conflict'],touch_plan_asof_count=sum(d['plan_asof_ok'] for d in D),stored_ordinal_counts=dict(Counter(x['ordinal'] for x in raw['touches'])),planned_arm_n=len(PS),planned_rr_p50=pct([x['rr'] for x in PS if x['rr'] is not None],.5),planned_arm_enabled_n=sum(x['enabled'] for x in PS),planned_arm_level_matches=sum(x['target_at_plan_level'] for x in PS),planned_arm_chain_matches=sum(x['target_in_chain'] for x in PS),snapshots=raw['snapshots'],plan_after_entry_ids=[t['id'] for t in T if t['plan_created_ms']>t['entry_time']],hold_max=max(t['hold_min'] for t in T))
Path('summary.json').write_text(json.dumps(summary,indent=2))
with open('results.txt','w') as f:
    f.write('AS OF '+raw['meta']['asof_ct']+' base b4376246; SQLite mode=ro, query_only=1\n')
    f.write('POPULATION '+json.dumps({k:summary[k] for k in ['eligible_n','ids','excluded','sum_usd','mean_usd','avg_win','avg_loss','payoff','win_wilson','cme_days','day_bootstrap95','mean_naive_normal95']})+'\n')
    for s in S:
        if s['dimension']!='vol_session':f.write('TRADE '+json.dumps(s)+'\n')
    f.write('RISK '+json.dumps({k:summary[k] for k in ['broker_R_ids','R_missing_ids','trade_excursions_n','snapshots','zeroB_ms','strict_ms']})+'\n')
    f.write('VOL '+json.dumps({k:summary[k] for k in ['rv_cuts_bps','rv_n','rv_missing_ids','bar_conflicting_times']})+'\n')
    f.write('TOUCH_META '+json.dumps({k:summary[k] for k in ['touches_raw','touch_price_time_keys','touch_kind_time_only_keys','touch_conflicting_keys','touch_plan_asof_count','stored_ordinal_counts']})+'\n')
    for s in TS:
        if s['dimension'] in ['kind','ordinal','all']:f.write('TOUCH '+json.dumps(s)+'\n')
    f.write('TARGET '+json.dumps({k:summary[k] for k in ['planned_arm_n','planned_arm_enabled_n','planned_rr_p50','planned_arm_level_matches','planned_arm_chain_matches','plan_after_entry_ids']})+'\n')
    for s in EX:f.write('PROXY '+json.dumps(s)+'\n')
print(json.dumps({k:v for k,v in summary.items() if k not in ['meta','ids','excluded','bar_conflicting_times']},indent=2))
