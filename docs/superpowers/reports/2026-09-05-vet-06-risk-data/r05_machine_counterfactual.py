#!/usr/bin/env python3
"""r05 — the GUARDRAIL'S OWN counterfactual, computed the way the machine computes it.
GetSessionDayActivity (store/position_query.go:123-155): SUM(pnl_corrected) over CLOSED rows,
close_reason NOT IN (reconcile_flat, unresolved, test_seam), pnl_corrected NOT NULL, by EXIT time
since the CME session-day start, account-scoped. Entries = COUNT(*) by ENTRY time, no other filter.
The machine does NOT apply the analyst's UNRESOLVABLE exclusion — so the control's counterfactual
uses the machine's population, while performance statistics use the compliant one."""
import sqlite3, datetime, zoneinfo, math
from collections import defaultdict
import numpy as np
ct=zoneinfo.ZoneInfo("America/Chicago")
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
def sday(ms):
    t=datetime.datetime.fromtimestamp(ms/1000, ct)
    return (t - datetime.timedelta(hours=17)).date().isoformat()
ERA_MIN=1787000000000  # well before the plan era's first entry (id 521, 08-19 03:22 CT)
first_entry=con.execute("SELECT MIN(entry_time) FROM trader_positions WHERE id>=521").fetchone()[0]
print("era first entry_time:", first_entry, sday(first_entry))
rows=con.execute("""SELECT id, exit_time, pnl_corrected, close_reason, account, plan_id, source
                    FROM trader_positions
                    WHERE status='CLOSED' AND pnl_corrected IS NOT NULL
                      AND close_reason NOT IN ('reconcile_flat','unresolved','test_seam')
                      AND exit_time >= ? ORDER BY exit_time""",(first_entry,)).fetchall()
days=defaultdict(list)
for r in rows: days[sday(r[1])].append(r)
keys=sorted(days)
print(f"machine population: {len(rows)} closed rows over {len(keys)} session-days")
print("  per-day: n, total, running-min")
runmin={}
for k in keys:
    seq=[r[2] for r in days[k]]; cum=np.cumsum(seq); runmin[k]=float(cum.min())
    print(f"    {k}: n={len(seq):>2} tot={sum(seq):+8.2f} runmin={cum.min():+8.2f}  ids={[r[0] for r in days[k]]}")
print("  DAILY-LOSS counterfactual on the MACHINE's input:")
for L in (300,450,600,900):
    trips=[k for k in keys if runmin[k]<=-L]; forf=0.0
    for k in trips:
        seq=[r[2] for r in days[k]]; cum=np.cumsum(seq); i=int(np.argmax(cum<=-L)); forf+=sum(seq[i+1:])
    print(f"    ${L}: {len(trips)}/{len(keys)} days {trips}  forfeited after trip = {forf:+.2f}")
# entries per session-day, machine's counting rule
ent=con.execute("SELECT id, entry_time, source FROM trader_positions WHERE entry_time >= ? ORDER BY entry_time",(first_entry,)).fetchall()
ed=defaultdict(list)
for r in ent: ed[sday(r[1])].append(r)
print("  ENTRY COUNT the machine sees (all rows, no filter) vs the compliant analysis count:")
import csv
comp=defaultdict(int)
for r in csv.DictReader(open('trade_sample_58.csv')): comp[r['session_day_ct']]+=1
over=0
for k in sorted(ed):
    print(f"    {k}: machine entries={len(ed[k]):>2}  compliant trades={comp.get(k,0):>2}")
    if len(ed[k])>3: over+=1
print(f"  MAX_DAILY_TRADES=3 on the machine's count: exceeded on {over}/{len(ed)} session-days")
