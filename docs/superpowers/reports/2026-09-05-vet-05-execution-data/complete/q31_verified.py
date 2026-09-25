"""Read-only final audit. Run in original scratch directory. No application writes."""
import sqlite3,json,csv,math,datetime,pathlib
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON')
c.execute('PRAGMA query_only=ON')
ERA=1786770000000; START=1788325200000; END=1788584400000
out={}
def rows(sql,args=()): return [dict(r) for r in c.execute(sql,args)]
def rate(k,n):
 if not n:return {'k':k,'n':n,'pct':None,'wilson95':None}
 p=k/n; z=1.96; d=1+z*z/n; m=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
 return {'k':k,'n':n,'pct':round(100*p,2),'wilson95':[round(100*max(0,m-h),2),round(100*min(1,m+h),2)]}
def pct(a):
 a=sorted(a)
 def q(p):
  i=(len(a)-1)*p; j=int(i);return round(a[j]+(a[min(j+1,len(a)-1)]-a[j])*(i-j),3) if a else None
 return {str(p):q(p) for p in [.5,.8,.95]} if a else {}
ps=rows("""SELECT * FROM trader_positions WHERE entry_time>=? AND entry_time<? AND upper(trim(coalesce(plan_id,'')))<>'UNRESOLVABLE' AND pnl_corrected IS NOT NULL AND coalesce(source,'')<>'e7_farside_test' AND lower(coalesce(close_reason,'')) NOT IN ('unresolved','unresolvable','reconcile_flat','e7_farside_test') AND upper(coalesce(pnl_correction_note,'')) NOT LIKE '%UNRESOLVABLE%' ORDER BY id""",(ERA,END))
assert len(ps)==58
assert abs(sum(r['pnl_corrected'] for r in ps)-(-466.428572))<0.000001
# whitelist position columns only, no accounts/config data
out['positions']=[{k:r[k] for k in ['id','side','entry_price','exit_price','entry_time','exit_time','entry_order_id','exit_order_id','pnl_corrected','mae','mfe','source','close_reason','plan_id','plan_version','cited_scenario_id']} for r in ps]
win=[r for r in ps if r['pnl_corrected']>0]; lose=[r for r in ps if r['pnl_corrected']<0]
aw=sum(r['pnl_corrected'] for r in win)/len(win);al=sum(r['pnl_corrected'] for r in lose)/len(lose)
out['performance']={'ids':[r['id'] for r in ps],'winner_ids':[r['id'] for r in win],'loser_ids':[r['id'] for r in lose],'scratch_ids':[r['id'] for r in ps if r['pnl_corrected']==0],'win_rate_ex_scratch':rate(len(win),len(win)+len(lose)),'average_win':aw,'average_loss':al,'payoff':aw/abs(al),'sum':sum(r['pnl_corrected'] for r in ps),'mean':sum(r['pnl_corrected'] for r in ps)/len(ps)}
out['excursions']={'table_rows':c.execute('select count(*) from trade_excursions').fetchone()[0]}
for label,g in [('winners',win),('losers',lose)]:
 out['excursions'][label]={key:{'ids':[r['id'] for r in g if r[key] is not None],'percentiles':pct([r[key] for r in g if r[key] is not None])} for key in ['mae','mfe']}
# Legacy exit reason labels are time/side/price matches from logs, not authoritative position FK.
legacy={int(r['id']):r for r in csv.DictReader(open('q14_mae_mfe.csv'))}
out['exit_reasons']={}
for reason in sorted(set(r['reason'] for r in legacy.values())):
 ids=[r['id'] for r in ps if legacy[r['id']]['reason']==reason]
 out['exit_reasons'][reason]={'ids':ids,'rate':rate(len(ids),len(ps))}
# Recompute an explicitly backward-looking simple TR14 proxy using only fully closed 5m bars.
floors=[]
for r in ps:
 b=rows("select rowid as bar_id,open_time_ms,h,l,c from bars where symbol='MNQ' and tf='5m' and open_time_ms+300000<=? order by open_time_ms desc limit 15",(r['entry_time'],))[::-1]
 if len(b)<15: continue
 atr=sum(max(b[i]['h']-b[i]['l'],abs(b[i]['h']-b[i-1]['c']),abs(b[i]['l']-b[i-1]['c'])) for i in range(1,15))/14
 floors.append({'position_id':r['id'],'pnl_corrected':r['pnl_corrected'],'mae':r['mae'],'bar_ids':[x['bar_id'] for x in b],'latest_close_ms':b[-1]['open_time_ms']+300000,'age_ms':r['entry_time']-(b[-1]['open_time_ms']+300000),'atr_simple_tr14':atr,'floor_proxy':1.5*atr,'mae_over_floor':r['mae']/(1.5*atr) if r['mae'] is not None else None})
out['floor_rows']=floors
fw=[r for r in floors if r['pnl_corrected']>0 and r['mae'] is not None and r['age_ms']<=300000]
out['floor_winners']={'ids':[r['position_id'] for r in fw],'ratio_percentiles':pct([r['mae_over_floor'] for r in fw]),'exceeded':rate(sum(r['mae_over_floor']>1 for r in fw),len(fw))}
# Preserve old entry audit with explicit matched bar IDs and uncertainty groups; add all exits.
old={int(r['pos_id']):r for r in csv.DictReader(open('q11_fill_vs_bar.csv'))}
fills=[]
for r in ps:
 for leg in ['entry','exit']:
  refms=r[leg+'_time']; price=r[leg+'_price']; method='position_'+leg+'_time'
  action=('BUY' if r['side'].upper()=='LONG' else 'SELL') if leg=='entry' else ('SELL' if r['side'].upper()=='LONG' else 'BUY')
  fr=rows('select id,created_at,price,exchange_trade_id as trade_id from trader_fills where side=? and abs(price-?)<0.01 and abs(created_at-?)<600000 order by abs(created_at-?) limit 1',(action,price,refms,refms))
  ms=fr[0]['created_at'] if fr else refms
  if fr:method='fill_price_side_nearest_time'
  if leg=='entry':
   oo=old[r['id']]; method=oo['fill_time_source']
   ms=int(oo['fill_time_ms'])
  b=rows("select rowid as bar_id,open_time_ms,o,h,l,c from bars where symbol='MNQ' and tf='1m' and open_time_ms<=? and open_time_ms+60000>? order by open_time_ms desc limit 1",(ms,ms))
  row={'position_id':r['id'],'leg':leg,'price':price,'method':method,'fill_id':fr[0]['id'] if fr else None,'trade_id':fr[0]['trade_id'] if fr else None,'fill_join_warning':'nearest price/side/time candidate; not authoritative execution FK','time_used_ms':ms,'bar':b[0] if b else None}
  if b:
   b=b[0];row.update(inside=b['l']<=price<=b['h'],at_high=abs(price-b['h'])<.126,at_low=abs(price-b['l'])<.126)
   row['adverse_extreme']=row['at_high'] if action=='BUY' else row['at_low'];row['favorable_extreme']=row['at_low'] if action=='BUY' else row['at_high']
  fills.append(row)
out['fill_rows']=fills
out['fill_summary']={}
for leg in ['entry','exit']:
 g=[r for r in fills if r['leg']==leg and r['bar']]
 out['fill_summary'][leg]={k:{'position_ids':[r['position_id'] for r in g if r[k]],'rate':rate(sum(r[k] for r in g),len(g))} for k in ['inside','at_high','at_low','adverse_extreme','favorable_extreme']}
# Distinguish calendar-created ledger cohort from plan-trade-date cohort and versioned keys.
plans=rows("select plan_id,version,trade_date,session,doc from plans where trade_date between '2026-09-02' and '2026-09-04' order by plan_id,version")
sc=[]
for p in plans:
 for s in json.loads(p['doc']).get('scenarios',[]): sc.append({'plan_id':p['plan_id'],'version':p['version'],'scenario':s.get('id'),'condition':s.get('condition'),'arm_enabled':bool((s.get('arm') or {}).get('enabled'))})
ars=rows("select *,strftime('%s',created_at)*1000 as created_ms,strftime('%s',updated_at)*1000 as updated_ms from armed_orders order by id")
ars=[r for r in ars if START<=r['created_ms']<END and 'test' not in (r['state_reason']+r['plan_id']).lower()]
keys=set(p['plan_id'] for p in plans);aligned=[r for r in ars if r['plan_id'] in keys]
out['funnel_scenarios']=sc
out['funnel_arms']=[{k:r[k] for k in ['id','plan_id','version','scenario','leg_index','kind','side','entry_px','stop_px','target_px','signal_id','state','state_reason','created_ms','updated_ms']} for r in ars]
for label,g in [('calendar_created',ars),('aligned_plan_dates',aligned)]:
 placed=[r for r in g if r['signal_id']];filled=[r for r in g if r['state']=='filled']
 out[label]={'row_ids':[r['id'] for r in g],'placed_ids':[r['id'] for r in placed],'filled_ids':[r['id'] for r in filled],'scenario_keys':sorted(set((r['plan_id'],r['scenario']) for r in g)),'leg_keys':sorted(set((r['plan_id'],r['scenario'],r['leg_index']) for r in g)),'placed_leg_keys':sorted(set((r['plan_id'],r['scenario'],r['leg_index']) for r in placed)),'placed_rate':rate(len(placed),len(g))}
# Boundaries for proxy reached, retaining exact bar rowids; no fabricated +2min life for zero-duration rows.
out['reached_proxy']=[]
for r in ars:
 if not r['signal_id']:continue
 trigger=r['entry_px'] + (.5 if r['side'].lower()=='long' else -.5) if r['kind']=='stop_entry' else r['entry_px']
 bs=rows("select rowid as bar_id,open_time_ms,h,l from bars where symbol='MNQ' and tf='1m' and open_time_ms>=? and open_time_ms+60000<=? order by open_time_ms",(r['created_ms'],r['updated_ms']))
 hits=[b for b in bs if b['l']<=trigger<=b['h']]
 out['reached_proxy'].append({'arm_id':r['id'],'trigger':trigger,'whole_lifetime_bar_ids':[b['bar_id'] for b in bs],'touch_bar_ids':[b['bar_id'] for b in hits],'warning':'created/updated is ledger lifetime, not exact broker lifetime; filled arm lifetime can extend after fill'})
out['guard_rows']=rows("select id,entry_px,stop_px,target_px,side,created_at,updated_at,state_reason from armed_orders where state_reason like '%marketable%' order by id")
out['counts']={t:c.execute('select count(*) from '+t).fetchone()[0] for t in ['trader_positions','armed_orders','plans','trade_excursions','bars','nt8_order_snapshots']}
pathlib.Path('q31_verified.json').write_text(json.dumps(out,indent=2)+'\n')
for k in ['performance','excursions','exit_reasons','floor_winners','fill_summary','calendar_created','aligned_plan_dates','counts']:print(k,json.dumps(out[k]))
