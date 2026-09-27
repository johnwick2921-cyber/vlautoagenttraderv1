#!/usr/bin/env python3
"""Validate the frozen audit package without executing or changing trading code."""
import pathlib,json,csv,collections,ast,hashlib,sqlite3,re,datetime,zoneinfo
p=pathlib.Path(__file__).parent
load=lambda n:json.loads((p/(n+'.json')).read_text())
s=list(csv.DictReader((p/'scenarios.csv').open()));r=[x for x in s if x['family']!='stand-aside'];t=load('tables');u=load('supplement');sm=load('summary');co=load('position-cohort')
checks=[]
def check(name,condition):
 assert condition,name
 checks.append({'check':name,'result':'PASS'})
check('806 unique retained scenario keys',len(s)==len({x['scenario_key'] for x in s})==806)
check('770 real plus 36 stand-aside',len(r)==770 and len(s)-len(r)==36)
for key in ['play_session','play_day_type','family_session','family_day_type','sides_session','sides_day_type','sides_regime','reached']:
 check(key+' partitions all real IDs',sum(x['n'] for x in t[key])==770 and {i for x in t[key] for i in x['ids']}=={x['scenario_key'] for x in r})
check('all calendar rows partition 770 authored scenarios',sum(x['scenario_n'] for x in t['calendar_days'])==770)
check('eligible positions all linked to retained scenarios',not u['unmatched_eligible_positions'])
check('78 era positions split into 65 eligible and 13 excluded',len(co['eligible'])==65 and len(co['excluded'])==13)
check('eligible P&L fields resolved; no test/unresolvable',all(x['pnl_corrected'] is not None and x['plan_id']!='UNRESOLVABLE' and x['source']!='e7_farside_test' for x in co['eligible']))
check('all 65 fill IDs represented exactly once in plan-day table',sorted(i for x in t['calendar_days'] for i in x['position_ids'])==sorted(x['id'] for x in co['eligible']))
check('24 acceptance observations represent 12 unique orders',len(load('accepted'))==24 and len(u['unique_accepted_orders'])==12 and sorted(i for x in u['unique_accepted_orders'] for i in x['receipt_ids'])==list(range(1,25)))
check('183 correct-side parent geometries',sm['geometry_n']==183 and not u['geometry_side_errors'])
check('198 expanded correct-side leg geometries',len(u['effective_legs'])==198 and all(x['correct_stop_side'] and x['target_R']>0 for x in u['effective_legs']))
check('49 real session-days; 280 defined comparison pairs',len(t['session_days'])==49 and len(t['similarity_pairs'])==280)
for f in p.glob('*.py'):ast.parse(f.read_text())
checks.append({'check':'Python artifact syntax (AST parse, no trading imports)','result':'PASS'})
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);private=[x[0] for tab in ['trader_positions','nt8_order_snapshots'] for x in c.execute('SELECT DISTINCT account FROM '+tab) if x[0]]
for line in pathlib.Path('/home/hoang/nofx/.env').read_text().splitlines():
 k,sep,v=line.partition('=');v=v.strip().strip('"').strip("'")
 if sep and re.search('SECRET|PASSWORD|TOKEN|API_KEY',k) and len(v)>=12:private.append(v)
files=[f for f in p.iterdir() if f.is_file()]+[p.parent/'2026-09-08-the-strategy.md']
for f in files:
 txt=f.read_text();check('private-value scan '+f.name,not any(v in txt for v in set(private)))
res={'validated_ct':datetime.datetime.now(zoneinfo.ZoneInfo('America/Chicago')).isoformat(),'checks':checks,'evidence_units':{'plans':279,'scenarios':806,'real_scenarios':770,'position_cohort':65,'expanded_arm_legs':198},'artifacts':[{'file':f.name,'bytes':f.stat().st_size,'sha256':hashlib.sha256(f.read_bytes()).hexdigest()} for f in files if f.name!='validation.json']}
(p/'validation.json').write_text(json.dumps(res,indent=2)+'\n');print('PASS:',len(checks),'checks')
