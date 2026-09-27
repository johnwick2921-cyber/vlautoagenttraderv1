"""Docs-only evidence runner. DB opened exclusively mode=ro; no live mutations."""
import sqlite3,json,math,datetime,pathlib,zoneinfo
ROOT=pathlib.Path('/home/hoang/nofx-analysis/vet-04-0905')
conn=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
conn.row_factory=sqlite3.Row
conn.execute('BEGIN')
ct=zoneinfo.ZoneInfo('America/Chicago')
def ms(day,h=0):return int(datetime.datetime.fromisoformat(day).replace(hour=h,tzinfo=ct).timestamp()*1000)
def wilson(k,n):
 if not n:return None
 z=1.95996398454;p=k/n;d=1+z*z/n;c=(p+z*z/(2*n))/d;w=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
 return [round(c-w,6),round(c+w,6)]
def query(label,sql,p):
 rows=[dict(r) for r in conn.execute(sql,p)];return {'label':label,'sql':sql,'parameters':p,'rows':rows}
result={'method':'Snapshot read transaction, retrospective state with explicit as-of limitations; no raw realized_pnl used. Wilson 95% descriptive, rows not assumed independent.','days':{}}
for day in ['2026-09-03','2026-09-04']:
 p={'start':ms(day),'end':ms(day,15),'era':ms('2026-08-15'),'day':day}
 blocks=[]
 blocks.append(query('1 closes before 15 CT, exclusive categories',"""SELECT id,side,entry_time,exit_time,pnl_corrected,plan_id,close_reason,source,
CASE WHEN source='e7_farside_test' THEN 'test_seam' WHEN entry_time<:era THEN 'pre_aug15' WHEN plan_id='UNRESOLVABLE' THEN 'UNRESOLVABLE' WHEN pnl_corrected IS NULL THEN 'null_pnl' WHEN COALESCE(close_reason,'') IN ('reconcile_flat','unresolved','test_seam') THEN 'excluded_close_reason' ELSE 'eligible' END category
FROM trader_positions WHERE status='CLOSED' AND exit_time>=:start AND exit_time<:end ORDER BY exit_time,id""",p))
 eligible=[r for r in blocks[-1]['rows'] if r['category']=='eligible'];n=len(eligible);k=sum(r['pnl_corrected']>0 for r in eligible)
 blocks[-1]['summary']={'n':n,'ids':[r['id'] for r in eligible],'pnl_corrected_sum':sum(r['pnl_corrected'] for r in eligible) if n else None,'wins':k,'wilson95':wilson(k,n),'exclusions':{c: [r['id'] for r in blocks[-1]['rows'] if r['category']==c] for c in ['test_seam','pre_aug15','UNRESOLVABLE','null_pnl','excluded_close_reason']}}
 blocks.append(query('2 arms created before cutoff; later terminal states are unknown at cutoff',"""SELECT id,session,scenario,entry_px,stop_px,target_px,created_at,updated_at,
CASE WHEN julianday(updated_at)>julianday(:end/1000.0,'unixepoch') THEN 'UNKNOWN_AT_CUTOFF' ELSE state END state_asof,
CASE WHEN julianday(updated_at)<=julianday(:end/1000.0,'unixepoch') THEN state_reason ELSE NULL END reason_asof
FROM armed_orders WHERE julianday(created_at)>=julianday(:start/1000.0,'unixepoch') AND julianday(created_at)<julianday(:end/1000.0,'unixepoch') AND COALESCE(session,'')<>'TEST-E7' ORDER BY id""",p))
 blocks.append(query('3 refusal rows (count cycles, not distinct order attempts)',"""SELECT id,timestamp,substr(execution_log,1,220) excerpt FROM decision_records WHERE julianday(timestamp)>=julianday(:start/1000.0,'unixepoch') AND julianday(timestamp)<julianday(:end/1000.0,'unixepoch') AND (execution_log LIKE '%entry_gate%' OR execution_log LIKE '%refused%') ORDER BY timestamp,id""",p))
 blocks.append(query('4 decision gaps including trailing silence',"""WITH t AS (SELECT id,timestamp,CAST(strftime('%s',timestamp) AS INTEGER) s,LAG(id) OVER(ORDER BY timestamp,id) pid,LAG(CAST(strftime('%s',timestamp) AS INTEGER)) OVER(ORDER BY timestamp,id) ps FROM decision_records WHERE julianday(timestamp)<julianday(:end/1000.0,'unixepoch')) SELECT pid,id,ps,s,ROUND((s-ps)/60.0,2) minutes,'interior' kind FROM t WHERE s>=:start/1000 AND s-ps>600 UNION ALL SELECT id,NULL,s,:end/1000,ROUND((:end/1000-s)/60.0,2),'trailing_to_1500' FROM (SELECT id,s FROM t ORDER BY s DESC LIMIT 1)""",p))
 blocks.append(query('4b observed MNQ 1m bars; missing hours require calendar context',"""SELECT strftime('%H',open_time_ms/1000,'unixepoch','-5 hours') ct_hour,COUNT(*) n,MIN(open_time_ms) first_ms,MAX(open_time_ms) last_ms FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms>=:start AND open_time_ms<:end GROUP BY ct_hour""",p))
 blocks.append(query('5 outcomes known before cutoff, row IDs and descriptive Wilson',"""SELECT level_kind,COUNT(*) n,SUM(outcome LIKE 'ambiguous%') ambiguous,GROUP_CONCAT(id) ids FROM touch_outcomes WHERE julianday(created_at)>=julianday(:start/1000.0,'unixepoch') AND julianday(created_at)<julianday(:end/1000.0,'unixepoch') AND closed_at_ms<:end GROUP BY level_kind""",p))
 for r in blocks[-1]['rows']:r['wilson95_ambiguous']=wilson(r['ambiguous'],r['n'])
 blocks.append(query('6 planner reads known before cutoff',"""SELECT id,session,version,atr5m,stop_floor_pts,void_count,bias_regime,created_at FROM planner_read_facts WHERE trade_date=:day AND julianday(created_at)<julianday(:end/1000.0,'unixepoch') ORDER BY id""",p))
 blocks.append(query('6b plan versions; lifecycle may have changed after cutoff',"""SELECT plan_id,version,session,created_at,trigger_reason FROM plans WHERE trade_date=:day AND julianday(created_at)<julianday(:end/1000.0,'unixepoch') ORDER BY created_at,version""",p))
 blocks.append(query('alerts emitted before cutoff; ack flags intentionally not reconstructed',"""SELECT id,level,kind,CASE WHEN created_at>100000000000 THEN created_at ELSE created_at*1000 END created_ms FROM day_plan_alerts WHERE (CASE WHEN created_at>100000000000 THEN created_at ELSE created_at*1000 END)>=:start AND (CASE WHEN created_at>100000000000 THEN created_at ELSE created_at*1000 END)<:end ORDER BY id""",p))
 blocks.append(query('broker order snapshot at cutoff is NOT a position book',"""SELECT id,account,symbol,working_count,received_at_ms,(:end-received_at_ms)/1000.0 age_seconds FROM nt8_order_snapshots WHERE received_at_ms<:end ORDER BY received_at_ms DESC LIMIT 1""",p))
 # Log source/line receipts, bounded to 15 CT across rotation files; no configuration contents.
 events=[];prefix=day[5:];cut=prefix+' 15:00:00'
 for f in sorted(pathlib.Path('/home/hoang/nofx/data').glob('nofx_*.log')):
  for lineno,line in enumerate(f.open(errors='replace'),1):
   if line.startswith(prefix) and line[:14]<cut and any(s in line for s in ['BOOT INTEGRITY','FEED DOWN','cycle_skip=no_new_data']):events.append({'path':str(f),'line':lineno,'text':line.rstrip()})
 blocks.append({'label':'7 boots/feed/skips before 15 CT, exact log lines','rows':events,'counts':{s:sum(s in r['text'] for r in events) for s in ['BOOT INTEGRITY','FEED DOWN','cycle_skip=no_new_data']}})
 result['days'][day]=blocks
p={'era':ms('2026-08-15')}
result['coverage']=query('era attribution and P&L coverage, mutually exclusive categories',"""SELECT CASE WHEN source='e7_farside_test' THEN 'test_seam' WHEN entry_time<:era THEN 'pre_aug15' WHEN plan_id='UNRESOLVABLE' THEN 'UNRESOLVABLE' WHEN pnl_corrected IS NULL THEN 'null_pnl' ELSE 'eligible' END category,COUNT(*) n,GROUP_CONCAT(id) ids FROM trader_positions GROUP BY category""",p)
result['ack']=query('ack census is current at audit, not historical visibility','SELECT level,COUNT(*) n,SUM(acked=1) acked,GROUP_CONCAT(id) ids FROM day_plan_alerts GROUP BY level',{})
for r in result['ack']['rows']:r['wilson95']=wilson(r['acked'],r['n'])
result['mae']=query('MAE coverage post Aug15 including exclusions for inventory only','SELECT COUNT(*) n,SUM(mae IS NOT NULL AND mfe IS NOT NULL) covered,GROUP_CONCAT(id) ids FROM trader_positions WHERE entry_time>=:era',p)
r=result['mae']['rows'][0];r['wilson95']=wilson(r['covered'],r['n'])
ROOT.joinpath('q48_eod_verified.json').write_text(json.dumps(result,indent=2))
for day,blocks in result['days'].items():print(day,blocks[0]['summary'],[(b['label'].split()[0],len(b['rows'])) for b in blocks],blocks[-1]['counts'])
print('coverage',[(r['category'],r['n']) for r in result['coverage']['rows']])
