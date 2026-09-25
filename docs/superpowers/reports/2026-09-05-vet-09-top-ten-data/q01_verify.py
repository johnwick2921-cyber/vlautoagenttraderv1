import sqlite3,json,statistics,math,urllib.request,datetime,re,pathlib
from zoneinfo import ZoneInfo
out=pathlib.Path('/home/hoang/nofx-analysis/vet-09-0905')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON');c.execute('BEGIN')
lines=[]
def emit(k,v): lines.append(k+'\n'+json.dumps(v,ensure_ascii=False,indent=2))
def query(k,q):
 rows=[dict(x) for x in c.execute(q)];emit(k+' SQL: '+q,rows);return rows
def wilson(k,n):
 z=1.959963984540054;p=k/n;d=1+z*z/n;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d;m=(p+z*z/(2*n))/d
 return {'k':k,'n':n,'rate':p,'wilson95':[m-h,m+h]}
emit('Captured UTC',datetime.datetime.now(datetime.timezone.utc).isoformat())
try:
 with urllib.request.urlopen('http://127.0.0.1:8080/api/health',timeout=10) as r: emit('GET /api/health',{'HTTP':r.status,'body':json.load(r)})
except Exception as e:emit('health error',str(e))
base="entry_time >= 1786770000000"
query('Era exclusions',"SELECT id,source,pnl_corrected,pnl_correction_note FROM trader_positions WHERE "+base+" AND (source='e7_farside_test' OR pnl_corrected IS NULL OR upper(coalesce(pnl_correction_note,'')) LIKE '%UNRESOLVABLE%') ORDER BY id")
rows=query('Canonical era',"SELECT id,entry_time,exit_time,side,source,plan_session,cited_scenario_id,pnl_corrected,fee,mae,mfe FROM trader_positions WHERE "+base+" AND coalesce(source,'') <> 'e7_farside_test' AND pnl_corrected IS NOT NULL AND upper(coalesce(pnl_correction_note,'')) NOT LIKE '%UNRESOLVABLE%' ORDER BY entry_time,id")
x=[r['pnl_corrected'] for r in rows];w=[v for v in x if v>0];l=[v for v in x if v<0]
days={}
for r in rows:
 day=(datetime.datetime.fromtimestamp(r['entry_time']/1000,ZoneInfo('America/Chicago'))-datetime.timedelta(hours=17)).date().isoformat();days.setdefault(day,[]).append(r)
emit('Canonical totals',{'n':len(x),'ids':[r['id'] for r in rows],'pnl_corrected_sum':sum(x),'mean':statistics.mean(x),'sd':statistics.stdev(x),'normal_approx_CI95':[statistics.mean(x)-1.96*statistics.stdev(x)/math.sqrt(len(x)),statistics.mean(x)+1.96*statistics.stdev(x)/math.sqrt(len(x))],'wins':len(w),'losses':len(l),'flats':len(x)-len(w)-len(l),'win_ex_flats':wilson(len(w),len(w)+len(l)),'win_including_flats':wilson(len(w),len(x)),'payoff':statistics.mean(w)/-statistics.mean(l),'session_days':len(days),'days':{k:{'ids':[v['id'] for v in a],'pnl':sum(v['pnl_corrected'] for v in a)} for k,a in days.items()}})
query('Empty excursion table','SELECT count(*) AS n FROM trade_excursions')
query('D1 raw RTH-L',"SELECT outcome,count(*) n,group_concat(id) ids FROM touch_outcomes WHERE level_kind='RTH-L' GROUP BY outcome")
query('D1 distinct units',"SELECT count(*) n, count(distinct level_kind||'|'||opened_at_ms) kind_time, count(distinct level_kind||'|'||level_price||'|'||opened_at_ms) kind_price_time FROM touch_outcomes")
query('RTH-L distinct units',"SELECT count(*) n,count(distinct opened_at_ms) times,count(distinct level_price||'|'||opened_at_ms) price_times FROM touch_outcomes WHERE level_kind='RTH-L'")
query('Snapshot witness rows',"SELECT id,emitted_at_ms,received_at_ms,order_count,working_count,orders_json FROM nt8_order_snapshots WHERE id IN (1604,1608,1688,1689)")
# Broker snapshots contain no credentials; omit account names and broker opaque ids from derived summary.
query('Read facts empty columns',"SELECT count(*) n,group_concat(id) ids,sum(tokens_in=0) zero_tokens,sum(bias_ai='') empty_ai,sum(bias_tree='') empty_tree FROM planner_read_facts")
query('Candidate components',"SELECT count(*) n,group_concat(id) ids,sum(score_components='{}') empty_components FROM candidate_pool")
allowed={'guardrails_enabled','daily_loss_enabled','daily_loss_limit_usd','max_daily_trades_enabled','max_contracts_enabled','plan_mode','max_levels','min_risk_reward_ratio','sessions_enabled','enable','session','name'}
def walk(v,path=''):
 if isinstance(v,dict):
  for k,a in v.items():
   if k in allowed and not isinstance(a,(dict,list)):emit('Bound strategy '+path+k,a)
   if k=='sessions_enabled':emit('Bound strategy '+path+k,a)
   walk(a,path+k+'.')
 elif isinstance(v,list):
  for i,a in enumerate(v):walk(a,path+str(i)+'.')
for row in c.execute('SELECT t.scan_interval_minutes,s.config FROM traders t JOIN strategies s ON s.id=t.strategy_id'):
 emit('Bound cadence minutes',row['scan_interval_minutes']);walk(json.loads(row['config']))
c.rollback();c.close()
(out/'q01_verify.txt').write_text('\n\n'.join(lines)+'\n')
print('Saved q01_verify.txt; canonical n',len(x),'days',len(days),'P&L',round(sum(x),2),'win',wilson(len(w),len(w)+len(l)))
