#!/usr/bin/env python3
"""Read-only, single-transaction extraction. No credentials/account fields exported."""
import datetime as dt
import json
import pathlib
import re
import sqlite3
import zoneinfo

OUT = pathlib.Path(__file__).parent
CT = zoneinfo.ZoneInfo('America/Chicago')
ERA = int(dt.datetime(2026, 8, 15, tzinfo=CT).timestamp() * 1000)
c = sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
c.row_factory = sqlite3.Row
c.execute('PRAGMA query_only=ON')
c.execute('BEGIN')
queries = []
def query(label, sql, params=()):
    rows = [dict(r) for r in c.execute(sql, params)]
    queries.append({'label': label, 'sql': sql, 'parameters': list(params), 'n': len(rows), 'output': label + '.json'})
    return rows

owner = c.execute('SELECT id,strategy_id FROM traders').fetchall()
assert len(owner) == 1, 'Owner scope is ambiguous'
tid, sid = owner[0]
plans = query('plans', 'SELECT rowid AS row_id,plan_id,version,trade_date,session,trigger_reason,lifecycle,prompt_hash,ai_config_hash,doc,indicators_block,created_at FROM plans WHERE trade_date>=? ORDER BY rowid', ('2026-08-15',))
positions = query('positions', 'SELECT id,symbol,side,entry_quantity,quantity,entry_price,entry_time,exit_price,exit_time,status,close_reason,source,entry_confidence,plan_version,cited_scenario_id,plan_matched,pnl_corrected,plan_id,plan_trade_date,plan_session,plan_link_note FROM trader_positions WHERE trader_id=? AND entry_time>=? ORDER BY id', (tid, ERA))
arms = query('arms', 'SELECT id,plan_id,version,session,scenario,side,entry_px,stop_px,target_px,state,state_reason,entry_class,signal_id,fill_price,fill_quantity,created_at,updated_at,leg_index,leg_count,kind,armed_under_version,condition,placement_seq FROM armed_orders WHERE trader_id=? ORDER BY id', (tid,))
facts = query('facts', 'SELECT id,trade_date,session,plan_id,version,prompt_hash,atr5m,stop_floor_pts,stop_floor_mlt,bias_ai,bias_tree,bias_regime,scope_since_ms,scope_bars,scope_intv,created_at FROM planner_read_facts WHERE trader_id=? ORDER BY id', (tid,))
candidates = query('candidates', 'SELECT id,symbol,plan_id,plan_version,session,read_at_ms,level_price,level_kind,label,rank,seated,cut_reason,score,threshold,grade,score_components FROM candidate_pool WHERE trader_id=? ORDER BY id', (tid,))
excursions = query('excursions', 'SELECT id,position_id,plan_id,version,session,scenario,condition,side,entry_px,entry_ts,exit_px,exit_ts,exit_reason,stop_px_initial,stop_px_final,target_px,size,mae_pts,mfe_pts,bars_held,ambiguous_exit,atr5m_at_entry,atr_mult_stop_at_entry,resolution,source FROM trade_excursions WHERE entry_ts>=? ORDER BY id', (ERA,))
accepted = query('accepted', 'SELECT id,signal_id,order_name,symbol,side,order_type,quantity,accepted_entry_px,accepted_stop_px,accepted_target_px,ledger_entry_px,ledger_stop_px,ledger_target_px,book_age_ms,book_source,accepted_at_ms FROM accepted_risk WHERE trader_id=? ORDER BY id', (tid,))
bars = query('bars', "SELECT symbol,tf,open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf IN ('1m','5m') AND open_time_ms>=? ORDER BY tf,open_time_ms", (ERA,))
lifecycle = query('lifecycle', 'SELECT id,plan_id,version,event,reason,at FROM plan_lifecycle_log ORDER BY id')
raw_cfg = json.loads(c.execute('SELECT config FROM strategies WHERE id=?', (sid,)).fetchone()[0])
cfg = {'strategy_id':sid, 'day_plan':raw_cfg.get('day_plan'), 'risk_control':raw_cfg.get('ai_config',{}).get('risk_control'), 'custom_prompt':raw_cfg.get('ai_config',{}).get('custom_prompt'), 'prompt_sections':raw_cfg.get('ai_config',{}).get('prompt_sections')}
queries.append({'label':'config','sql':'SELECT config FROM strategies WHERE id=(SELECT strategy_id FROM traders WHERE id=?)','parameters':[tid], 'n':1,'output':'config.json','projection':'Only day_plan, risk_control, custom_prompt and prompt_sections; keys and connection details omitted.'})
# Account identifiers are only read to remove them from narrative strings; never exported.
private = [tid]
for table in ['trader_positions','nt8_order_snapshots']:
    private += [r[0] for r in c.execute('SELECT DISTINCT account FROM '+table) if r[0]]
def sanitize(x):
    if isinstance(x, dict): return {k:sanitize(v) for k,v in x.items() if k not in ['account','user_id','trader_id']}
    if isinstance(x, list): return [sanitize(v) for v in x]
    if isinstance(x, str):
        for s in private: x=x.replace(':'+s, '').replace(s, '[redacted]')
        x=re.sub(r'\b(?:sk-|cm_)[A-Za-z0-9_-]{12,}', '[redacted-key]', x)
    return x
snapshot = {'read_ct':dt.datetime.now(CT).isoformat(), 'era_ms':ERA, 'era_ct':'2026-08-15T00:00:00-05:00', 'queries':queries}
c.rollback()
for name, value in [('plans',plans),('positions',positions),('arms',arms),('facts',facts),('candidates',candidates),('excursions',excursions),('accepted',accepted),('bars',bars),('lifecycle',lifecycle),('config',cfg),('queries',snapshot)]:
    (OUT/(name+'.json')).write_text(json.dumps(sanitize(value),ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'read_ct':snapshot['read_ct'], 'counts':{q['label']:q['n'] for q in queries}},indent=2))
