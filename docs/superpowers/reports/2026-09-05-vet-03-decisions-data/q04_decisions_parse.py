# q04: parse decision_records.decision_json action per cycle (not LIKE), by CT day since 08-27; cycle_type; planner durations
import sqlite3, json, collections, datetime
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
rows=con.execute("""SELECT id, datetime(timestamp,'-5 hours') ct, decision_json, cited_scenario_id, risk_check_passed, risk_check_error, execution_status, cycle_type, cycle_trigger, ai_request_duration_ms, plan_id, plan_version
 FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-08-27' ORDER BY id""").fetchall()
def act(dj):
    if not dj: return 'NONE'
    try:
        d=json.loads(dj)
    except Exception: return 'PARSE_ERR'
    if isinstance(d,list):
        acts=[str(x.get('action','')) for x in d if isinstance(x,dict)]
        return ','.join(sorted(set(acts))) or 'EMPTY_LIST'
    if isinstance(d,dict):
        if 'decisions' in d and isinstance(d['decisions'],list):
            acts=[str(x.get('action','')) for x in d['decisions'] if isinstance(x,dict)]
            return ','.join(sorted(set(acts))) or 'EMPTY_LIST'
        return str(d.get('action','NOACT'))
    return 'OTHER'
byday=collections.defaultdict(collections.Counter)
ctype=collections.Counter()
intents=[]
for r in rows:
    a=act(r[2]); day=r[1][:10]
    byday[day][a]+=1
    ctype[(r[7],r[8])]+=1
    if a and ('open_long' in a or 'open_short' in a):
        intents.append((r[0],r[1],a,r[3],r[4],(r[5] or '')[:70],r[6],r[7]))
print('## actions by CT day (parsed)')
for d in sorted(byday):
    c=byday[d]; tot=sum(c.values())
    print(d, 'cycles',tot, dict(c.most_common(6)))
print('## cycle_type x cycle_trigger', ctype.most_common(15))
print('## open intents since 08-27 (id, ct, action, cited, risk_ok, err, exec_status, cycle_type)')
for i in intents: print(i)
print('n intents', len(intents))
