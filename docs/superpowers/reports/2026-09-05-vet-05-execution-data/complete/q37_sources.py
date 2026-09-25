"""Supplementary source and selected read-only store evidence at b4376246."""
from pathlib import Path
import sqlite3,json,subprocess,datetime
root=Path('/home/hoang/nofx-vet-05-complete')
print('Source base b4376246; captured',datetime.datetime.now(datetime.timezone.utc).isoformat())
spans={'trader/armed_executor.go':[(380,432),(1345,1410)],'kernel/min_sl.go':[(18,50)],'trader/auto_trader_clock.go':[(454,522),(736,768)],'trader/exit_mechs_suspend.go':[(10,66)],'trader/f12_leg4.go':[(193,210)],'kernel/session_registry.go':[(78,114)],'store/strategy.go':[(1005,1040)],'ninjascript/VLTraderTCPClient.cs':[(1338,1357),(1587,1607),(2351,2377)],'trader/ninjatrader/tcp_trader.go':[(232,245)],'trader/trade_excursion_hook.go':[(35,85)]}
for file,rr in spans.items():
 text=(root/file).read_text().splitlines()
 for lo,hi in rr:
  for n in range(lo,hi+1):print(f'{file}:{n}: {text[n-1]}')
for base in ['2a66d91c','36648655']:
 args=['git','diff','--name-only',base,'b4376246','--','trader/armed_executor.go','ninjascript/VLTraderTCPClient.cs','provider/ninjatrader/tcp_framing.go','trader/ninjatrader','store/armed_orders.go']
 r=subprocess.run(args,cwd=root,text=True,capture_output=True,check=True);print('COMPARE',base,'to b4376246 selected execution paths:',r.stdout.strip() or 'NO DIFFERENCES')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('pragma query_only=ON')
rows=lambda q,a=():[dict(r) for r in c.execute(q,a)]
print('exit-bar query',json.dumps(rows("select rowid as bar_id,open_time_ms,o,h,l,c from bars where symbol='MNQ' and tf='1m' and open_time_ms=1788445200000")))
print('exit fill433',json.dumps(rows('select id,exchange_trade_id,side,price,created_at from trader_fills where id=433')))
d=json.loads(c.execute("select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8'").fetchone()[0]);dp=d.get('day_plan',{})
print('Selected stored session fields; missing offsets use source default, not a runtime override audit',json.dumps({k:dp.get(k) for k in ['plan_enabled','plan_mode','eod_flat_ct','eod_flat_offset_min','sessions']}))
x=json.load(open('q31_verified.json'));stopids=[r['signal_id'] for r in x['funnel_arms'] if r['kind']=='stop_entry' and r['signal_id']]
seen=set();states=[]
for r in rows('select id,received_at_ms,orders_json from nt8_order_snapshots order by id'):
 if not any(s in r['orders_json'] for s in stopids):continue
 orders=json.loads(r['orders_json']);orders=orders if isinstance(orders,list) else orders.get('orders',[])
 for a in orders:
  # Persist only an already-audited stop signal and its first snapshot state.
  if not any(s in json.dumps(a) for s in stopids):continue
  key=json.dumps(a,sort_keys=True)
  if key in seen:continue
  seen.add(key);states.append({'snapshot_id':r['id'],'received_at_ms':r['received_at_ms'],'order':a})
Path('stop_snapshot_states.json').write_text(json.dumps(states,indent=2)+'\n')
print('first distinct stop snapshot states',len(states),'see stop_snapshot_states.json')

print('trace position591 selected fields',json.dumps(rows("select id,source,entry_time,exit_time,entry_price,exit_price,pnl_corrected,mae,mfe,plan_id,plan_version,cited_scenario_id,adherence_grade,plan_matched from trader_positions where id=591")))
print('trace arm35 selected fields',json.dumps(rows("select id,plan_id,version,scenario,leg_index,state,signal_id,entry_px,stop_px,target_px,fill_price,created_at,updated_at from armed_orders where id=35")))
