import sqlite3, json, csv, math
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
con.row_factory=sqlite3.Row
ERA=1786770000000
trades=[dict(r) for r in con.execute("SELECT * FROM trader_positions WHERE entry_time>=? ORDER BY id",(ERA,))]
plans={}
for r in con.execute("SELECT plan_id, version, session, doc FROM plans"):
    try: plans[(r['plan_id'],int(r['version']))]=json.loads(r['doc'])
    except Exception: pass
arms={}
for r in con.execute("SELECT * FROM armed_orders WHERE session NOT LIKE 'TEST%'"):
    arms.setdefault((r['plan_id'],int(r['version']),r['scenario']),[]).append(dict(r))
# decision records with open_* actions (parsed)
decs=[]
for r in con.execute("SELECT id, trader_id, timestamp, decision_json, risk_check_error, execution_log FROM decision_records WHERE timestamp>='2026-08-18' AND (decision_json LIKE '%open_long%' OR decision_json LIKE '%open_short%')"):
    try: dj=json.loads(r['decision_json'])
    except Exception: continue
    if isinstance(dj,dict): dj=[dj]
    for d in dj:
        if isinstance(d,dict) and str(d.get('action','')).startswith('open_'):
            decs.append(dict(id=r['id'], ts=r['timestamp'], action=d['action'], sl=d.get('stop_loss'), tp=d.get('take_profit'), entry=d.get('entry'), conf=d.get('confidence'), cited=d.get('cited_scenario') or d.get('scenario') or '', rce=r['risk_check_error'], el=r['execution_log']))
import datetime
def ts_ms(s):
    s=s.replace('Z','+00:00')
    # trim nanoseconds to microseconds
    if '.' in s:
        head,rest=s.split('.',1)
        import re
        m=re.match(r'(\d+)(.*)',rest); frac,tz=m.group(1)[:6],m.group(2)
        s=f"{head}.{frac}{tz}"
    return int(datetime.datetime.fromisoformat(s).timestamp()*1000)
for d in decs: d['ms']=ts_ms(d['ts'])
snaps=[dict(r) for r in con.execute("SELECT id, received_at_ms, orders_json FROM nt8_order_snapshots WHERE received_at_ms>=?",(ERA,))]
print('snapshots since era:', len(snaps), 'min', datetime.datetime.utcfromtimestamp(min(s['received_at_ms'] for s in snaps)/1000) if snaps else None)
out=[]
for t in trades:
    pid,ver,sid=t['plan_id'],int(t['plan_version'] or 0),(t['cited_scenario_id'] or '')
    doc=plans.get((pid,ver)) or {}
    sc=None
    for s in doc.get('scenarios',[]) or []:
        if s.get('id')==sid: sc=s
    cond=sc.get('condition') if sc else ''
    q=sc.get('quality') if sc else ''
    rule=(sc.get('confirm') or {}).get('rule') if sc else ''
    sdir=sc.get('direction') if sc else ''
    arm=(sc.get('arm') or {}) if sc else {}
    regime=doc.get('bias_label','')
    daytype=doc.get('day_type','')
    bias=(doc.get('bias') or {}).get('direction','')
    side=t['side'].lower()
    ep=float(t['entry_price'] or 0); xp=float(t['exit_price'] or 0)
    sign=1 if side=='long' else -1
    pnl_pts=(xp-ep)*sign if ep and xp else None
    # stop sources
    stop_src=''; stop=None; tgt=None
    a=arms.get((pid,ver,sid),[])
    fa=[x for x in a if x['state']=='filled']
    if t['plan_band']=='armed_fill' or t['plan_link_note']=='armed_fill' or t['source']=='armed_entry':
        cand=fa or a
        if cand:
            x=cand[0]; stop=x['stop_px']; tgt=x['target_px']; stop_src=f"armed_orders#{x['id']}"
    if stop is None:
        # nearest open_* decision within 6 min before entry, same side
        et=int(t['entry_time'])
        c=[d for d in decs if d['action']=='open_'+side and et-360000<=d['ms']<=et+60000]
        if c:
            d=min(c,key=lambda d:abs(d['ms']-et)); stop=d['sl']; tgt=d['tp']; stop_src=f"decision#{d['id']}"
            if not t['entry_confidence']: pass
    if stop is None and arm.get('stop'):
        stop=arm['stop']; tgt=arm['target']; stop_src='plan_arm'
    # NT8 snapshot stop (broker truth) within 3 min after entry
    et=int(t['entry_time']); nt8stop=None; nt8src=''
    for s in snaps:
        if et-5000<=s['received_at_ms']<=et+180000:
            try: oj=json.loads(s['orders_json'])
            except Exception: continue
            for o in oj:
                if o.get('type')=='stop' and str(o.get('state','')).lower() in ('accepted','working','triggerpending','submitted'):
                    # a protective stop is on the opposite side
                    if (side=='long' and o.get('action')=='sell') or (side=='short' and o.get('action')=='buy'):
                        px=o.get('stop_price') or o.get('limit_price')
                        if px: nt8stop=px; nt8src=f"nt8snap#{s['id']}"; break
            if nt8stop: break
    stop_pts=abs(ep-stop) if stop else None
    nt8_stop_pts=abs(ep-nt8stop) if nt8stop else None
    R=(pnl_pts/stop_pts) if (stop_pts and pnl_pts is not None) else None
    mfe=t['mfe']; mae=t['mae']
    out.append(dict(id=t['id'], source=t['source'], session=t['plan_session'] or '', side=side, entry_ct=datetime.datetime.utcfromtimestamp(et/1000-5*3600).strftime('%m-%d %H:%M'), hour_ct=int(datetime.datetime.utcfromtimestamp(et/1000-5*3600).strftime('%H')), hold_min=round((int(t['exit_time'])-et)/60000,1) if t['exit_time'] else None, entry=ep, exit=xp, pnl=t['pnl_corrected'], pnl_pts=round(pnl_pts,2) if pnl_pts is not None else None, cond=cond, quality=q, rule=rule, scen_dir=sdir, plan_bias=bias, regime=regime, day_type=daytype, stop=stop, stop_pts=round(stop_pts,2) if stop_pts else None, stop_src=stop_src, tgt=tgt, plan_rr=round(abs(tgt-ep)/stop_pts,2) if (tgt and stop_pts) else None, R=round(R,2) if R is not None else None, mfe=mfe, mae=mae, mfe_over_stop=round(mfe/stop_pts,2) if (mfe is not None and stop_pts) else None, mae_over_stop=round(mae/stop_pts,2) if (mae is not None and stop_pts) else None, nt8_stop=nt8stop, nt8_stop_pts=round(nt8_stop_pts,2) if nt8_stop_pts else None, nt8src=nt8src, close_reason=t['close_reason'], grade=t['adherence_grade'], conf=t['entry_confidence'], plan_ver=ver, cited=sid))
w=csv.DictWriter(open('q13_trades_enriched.csv','w'),fieldnames=list(out[0].keys())); w.writeheader(); w.writerows(out)
for o in out:
    print(o['id'], o['source'][:5], o['session'][:6].ljust(6), o['side'][:1], o['entry_ct'], str(o['hold_min']).rjust(6), str(o['pnl']).rjust(8), (o['cond'] or '-').ljust(15), (o['rule'] or '-').ljust(10), 'stop', str(o['stop_pts']).rjust(6), o['stop_src'].ljust(18), 'nt8', str(o['nt8_stop_pts']).rjust(6), 'R', str(o['R']).rjust(6), 'mfe/s', str(o['mfe_over_stop']).rjust(5), 'mae/s', str(o['mae_over_stop']).rjust(5), 'pRR', o['plan_rr'], o['regime'][-12:] if o['regime'] else '-')
