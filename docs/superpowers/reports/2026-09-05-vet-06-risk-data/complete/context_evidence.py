#!/usr/bin/env python3
import pathlib,sqlite3,json,hashlib,subprocess,datetime
from zoneinfo import ZoneInfo
ROOT=pathlib.Path('/home/hoang/nofx-vet-06-complete'); OUT=pathlib.Path(__file__).resolve().parent
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);con.execute('PRAGMA query_only=ON');con.execute('BEGIN');con.row_factory=sqlite3.Row
r={'calendar_count':con.execute('select count(*) from calendar_slices').fetchone()[0], 'calendar_recent':[dict(z) for z in con.execute("SELECT * FROM calendar_slices WHERE trade_date>='2026-09-03' ORDER BY trade_date")], 'static_files':[]}
for p in [ROOT/'calendar_static_t1.json',pathlib.Path('/home/hoang/nofx/calendar_static_t1.json')]:
 data=p.read_bytes(); ev=json.loads(data);r['static_files'].append(dict(path=str(p),sha256=hashlib.sha256(data).hexdigest(),events=len(ev),t1=sum(e['impact']=='T1' for e in ev),notes=sum(e['impact']=='note' for e in ev)))
# Current guard's close-reason filter is deliberately different from research plan eligibility.
q="""SELECT id,entry_time,exit_time,pnl_corrected,plan_id,close_reason FROM trader_positions WHERE status='CLOSED' AND close_reason NOT IN ('reconcile_flat','unresolved','e7_farside_test') AND pnl_corrected IS NOT NULL AND entry_time>=1786770000000 ORDER BY exit_time,id"""
z=[dict(v) for v in con.execute(q)]; buckets={}
for v in z:
 day=(datetime.datetime.fromtimestamp(v['exit_time']/1000,ZoneInfo('America/Chicago'))-datetime.timedelta(hours=17)).date().isoformat();buckets.setdefault(day,[]).append(v)
r['guard_scope_sensitivity']={'sql':q,'note':'All stored traders/accounts aggregate; account identity not reconstructed. Analytic comparison only, NOT exact live risk replay.','days':[dict(day=k,ids=[v['id'] for v in vs],pnl=sum(v['pnl_corrected'] for v in vs)) for k,vs in buckets.items()]}
con.rollback();con.close();(OUT/'context.json').write_text(json.dumps(r,indent=2)+'\n')
code_ranges={'trader/auto_trader_calendar.go':[(25,50),(65,86),(110,139),(142,217)],'kernel/calendar_blackout.go':[(13,41),(94,109)],'calendar/calendar.go':[(172,195)],'store/calendar.go':[(103,130)],'trader/auto_trader_clock.go':[(643,707)],'kernel/risk_limits.go':[(164,205),(305,319),(357,387)],'kernel/engine_analysis.go':[(183,198)],'store/position_query.go':[(92,114),(125,153)],'trader/entry_gate.go':[(157,172)],'trader/auto_trader_orders.go':[(107,129),(230,254),(277,286)],'trader/auto_trader_session.go':[(37,47),(68,95)],'trader/contract_roll.go':[(110,140)],'ninjascript/VLContractResolver.cs':[(51,55),(78,81),(128,146)],'ninjascript/VLTraderTCPClient.cs':[(899,904),(1091,1097)],'kernel/regime.go':[(31,45),(69,77)],'docs/superpowers/AUDIT-CHECKLIST.md':[(175,188),(641,650)],'calendar_static_t1.json':[(1,19)]}
lines=[]
for path,ranges in code_ranges.items():
 text=(ROOT/path).read_text().splitlines()
 for lo,hi in ranges:
  lines.extend(f'{path}:{i}: {text[i-1]}' for i in range(lo,min(hi,len(text))+1))
for path in ['kernel/engine_analysis.go','trader/entry_gate.go']:
 text=subprocess.check_output(['git','show','36648655:'+path],cwd=ROOT,text=True).splitlines()
 if path.endswith('engine_analysis.go'):lines.extend(f'RUNNING 36648655:{path}:{i}: {text[i-1]}' for i in range(183,189))
 else:lines.append('RUNNING 36648655:'+path+': DailyForceFlat occurrences='+str(sum('DailyForceFlat' in s for s in text)))
log=pathlib.Path('/home/hoang/nofx/data/nofx_2026-09-03.log')
for i,s in enumerate(log.read_text(errors='replace').splitlines(),1):
 if s.startswith('09-03 11:10:33') and ('ledger boot:' in s or 'ScanIntervalMinutes=2' in s):lines.append(f'{log}:{i}: {s}')
(OUT/'source-evidence.txt').write_text('\n'.join(line.rstrip() for line in lines)+'\n')
print(json.dumps(r['static_files']));print(json.dumps(r['guard_scope_sensitivity']['days']))
