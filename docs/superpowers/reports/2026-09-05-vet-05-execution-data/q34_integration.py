import sqlite3,json,datetime,collections
from zoneinfo import ZoneInfo
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row
ct=ZoneInfo('America/Chicago');x=json.load(open('q31_verified.json')); days=collections.defaultdict(list);calendar=collections.defaultdict(list)
for r in x['positions']:
 t=datetime.datetime.fromtimestamp(r['entry_time']/1000,ct)
 calendar[str(t.date())].append(r)
 # CME session beginning 17:00 CT; label by opening calendar date.
 d=(t-datetime.timedelta(hours=17)).date();days[str(d)].append(r)
def detail(g):return {k:{'position_ids':[r['id'] for r in v],'n':len(v),'pnl_corrected':sum(r['pnl_corrected'] for r in v)} for k,v in sorted(g.items())}
cut=int(datetime.datetime(2026,9,3,11,10,tzinfo=ct).timestamp()*1000)
a=dict(c.execute('select id,signal_id,stop_px,fill_price from armed_orders where id=35').fetchone());p=dict(c.execute('select id,entry_time,exit_time,entry_price,exit_price,pnl_corrected from trader_positions where id=591').fetchone())
o={'cme_open_date_groups':detail(days),'cme_days':len(days),'calendar_days':len(calendar),'calendar_date_groups':detail(calendar),'trades':len(x['positions']),'pnl_corrected':sum(r['pnl_corrected'] for r in x['positions']),'strict_cut_assumption_ct':'2026-09-03 11:10:00 America/Chicago (parent boot boundary)','positions_after_cut':[dict(r) for r in c.execute('select id,entry_time,source,pnl_corrected from trader_positions where entry_time>=?',(cut,))],'latest_position':dict(c.execute('select id,entry_time from trader_positions order by entry_time desc limit 1').fetchone()),'arm35':a,'position591':p,'broker_stop_from_q32_nt8_log':29355,'slippage_pts_vs_authorized_broker_stop':p['exit_price']-29355,'ledger_to_broker_geometry_drift_pts':29355-a['stop_px'],'snapshots1604':[dict(r) for r in c.execute('select id,received_at_ms,reason,orders_json from nt8_order_snapshots where id in (1604,1605,1606)')],'trace_5m_bar':[dict(r) for r in c.execute("select rowid as bar_id,open_time_ms,o,h,l,c from bars where symbol='MNQ' and tf='5m' and open_time_ms=?",(int(datetime.datetime(2026,9,3,9,10,tzinfo=ct).timestamp()*1000),))]}
json.dump(o,open('q34_integration.json','w'),indent=2)
print('CME days',o['cme_days'],'calendar days',o['calendar_days'],'trades',o['trades'],'pnl',o['pnl_corrected'],'after strict cut',o['positions_after_cut'])
print('Broker-referenced stop slippage',o['slippage_pts_vs_authorized_broker_stop'],'ledger geometry drift',o['ledger_to_broker_geometry_drift_pts'])
