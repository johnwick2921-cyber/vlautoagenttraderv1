# SUPERSEDED by r08_pnl_cond.py. Kept as the record of a trap: this draft mapped cited_scenario_id to the LATEST plan version (and used a 2025 era constant). Scenario ids S1/S2/S3 are REUSED across versions with DIFFERENT conditions, so the condition cells came out wrong. Map on (plan_id, plan_version).
import sqlite3, json, math
db=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
c=db.cursor()
ERA=1755234000000  # 2026-08-15 00:00 CT
rows=c.execute("""SELECT id, source, plan_id, cited_scenario_id, pnl_corrected, plan_session, symbol
 FROM trader_positions WHERE entry_time>=? AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>''
 ORDER BY id""",(ERA,)).fetchall()
print("all cited era rows:", len(rows))
e7=[r for r in rows if r[1]=='e7_farside_test']
unres=[r for r in rows if r[2]=='UNRESOLVABLE']
print("e7:", [r[0] for r in e7], " UNRESOLVABLE:", [(r[0],r[3],r[4]) for r in unres])
scope=[r for r in rows if r[1]!='e7_farside_test' and r[2]!='UNRESOLVABLE']
print("compliant in-scope:", len(scope))
nulls=[r[0] for r in scope if r[4] is None]
print("NULL pnl_corrected (excluded, UNRESOLVED):", nulls, "count", len(nulls))
usable=[r for r in scope if r[4] is not None]
print("usable:", len(usable), "sum", round(sum(r[4] for r in usable),2))
# condition per position: read from the plan doc scenario? use armed/dec? Use the q13 mapping: cited_scenario_id -> plans doc
# instead: map via decision_records? Simplest: use plans.doc scenario condition
import re
plans={}
for pid,ver,doc in c.execute("SELECT plan_id,version,doc FROM plans"):
    plans[(pid,ver)]=doc
def cond_for(r):
    pid=r[2]; sc=r[3]
    # find latest version of that plan
    vers=sorted([v for (p,v) in plans if p==pid])
    for v in reversed(vers):
        d=plans[(pid,v)]
        try: j=json.loads(d)
        except Exception: continue
        for s in j.get("scenarios",[]) if isinstance(j,dict) else []:
            if str(s.get("id"))==str(sc):
                return s.get("condition")
    return None
from collections import defaultdict
cells=defaultdict(list)
for r in usable:
    cells[cond_for(r)].append((r[0], r[4]))
for k in sorted(cells, key=lambda x:(x is None, str(x))):
    v=cells[k]; s=sum(x[1] for x in v); w=sum(1 for x in v if x[1]>0)
    print(f"{k}: n={len(v)} sum={round(s,2)} wins={w} ids={[x[0] for x in v]}")
# same but INCLUDING the 2 UNRESOLVABLE rows for the sensitivity line
cells2=defaultdict(list)
for r in [x for x in rows if x[1]!='e7_farside_test' and x[4] is not None]:
    cells2[cond_for(r)].append((r[0], r[4]))
print("--- sensitivity incl UNRESOLVABLE ---")
for k in sorted(cells2, key=lambda x:(x is None, str(x))):
    v=cells2[k]; print(f"{k}: n={len(v)} sum={round(sum(x[1] for x in v),2)}")
