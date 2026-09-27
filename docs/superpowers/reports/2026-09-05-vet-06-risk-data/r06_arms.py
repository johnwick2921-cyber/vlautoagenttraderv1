#!/usr/bin/env python3
"""r06 — arm stop distribution and planned R:R, with the test seam excluded by the report's own law."""
import sqlite3, statistics as st
import numpy as np
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
rows=con.execute("""SELECT id, session, scenario, state, plan_id, version, leg_index,
                           entry_px, stop_px, target_px, fill_price
                    FROM armed_orders WHERE entry_px IS NOT NULL AND stop_px IS NOT NULL ORDER BY id""").fetchall()
def q(a,p): return float(np.percentile(np.array(a),p))
def show(tag, recs):
    stops=[abs(r[7]-r[8]) for r in recs]
    rr=[abs(r[9]-r[7])/abs(r[7]-r[8]) for r in recs if abs(r[7]-r[8])>0 and r[9] is not None]
    print(f"  {tag}: n={len(recs)}  stop pts p25={q(stops,25):.2f} p50={q(stops,50):.2f} p75={q(stops,75):.2f} "
          f"p90={q(stops,90):.2f} max={max(stops):.2f} (${max(stops)*2:.0f})  "
          f"| $ p50=${q(stops,50)*2:.0f} p90=${q(stops,90)*2:.0f}")
    if rr: print(f"      planned R:R n={len(rr)} mean={st.mean(rr):.3f} min={min(rr):.3f} max={max(rr):.3f}")
allr=rows
real=[r for r in rows if not (r[1] or '').upper().startswith('TEST')]
print("ALL rows incl. test seam (what the report used):")
show("raw", allr)
print("REAL only (session NOT LIKE 'TEST%') — the report's own exclusion law:")
show("raw placements", real)
# dedupe: one row per distinct arm SPEC
seen={}; ded=[]
for r in real:
    k=(r[4],r[5],r[2],r[6],r[7],r[8],r[9])
    if k not in seen: seen[k]=1; ded.append(r)
show("distinct spec (plan,version,scenario,leg,entry,stop,target)", ded)
seen2={}; ded2=[]
for r in real:
    k=(r[4],r[2],round(r[7],2),round(r[8],2))
    if k not in seen2: seen2[k]=1; ded2.append(r)
show("distinct (plan,scenario,entry,stop)", ded2)
# the max real stop, and what state it is in
mx=max(real,key=lambda r: abs(r[7]-r[8]))
print(f"  LARGEST REAL arm stop: id={mx[0]} session={mx[1]} scenario={mx[2]} state={mx[3]} "
      f"stop={abs(mx[7]-mx[8]):.2f} pts = ${abs(mx[7]-mx[8])*2:.0f}")
# filled arms only
fil=[r for r in real if r[3]=='filled']
if fil: show("filled real arms", fil)
# how many real arms would a 75-pt cap refuse?
cap=[r for r in real if abs(r[7]-r[8])>75]
print(f"  a 75-pt ($150) per-trade cap would have refused {len(cap)} of {len(real)} real arm placements "
      f"({len([r for r in ded if abs(r[7]-r[8])>75])} of {len(ded)} distinct specs)")
# churn window ids 62..103
ch=[r for r in rows if 62<=r[0]<=103]
print(f"  ids 62..103: {len(ch)} rows")
from collections import Counter
c=Counter((r[2],r[3],round(abs(r[7]-r[8]),2)) for r in ch)
for k,v in sorted(c.items(), key=lambda x:-x[1]): print(f"      scenario={k[0]} state={k[1]} stop={k[2]} -> {v} rows")
