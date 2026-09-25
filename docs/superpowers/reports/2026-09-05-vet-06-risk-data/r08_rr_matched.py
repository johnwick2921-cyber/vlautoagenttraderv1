#!/usr/bin/env python3
"""r08 — planned vs REALISED R on the arms that actually filled (the 2.55:1 / 1.66:1 premise, re-measured)."""
import sqlite3, statistics as st
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
arms=con.execute("""SELECT id, session, scenario, side, entry_px, stop_px, target_px, signal_id, fill_price
                    FROM armed_orders WHERE state='filled' AND session NOT LIKE 'TEST%' ORDER BY id""").fetchall()
print("real filled arms:", len(arms))
rows=[]
for a in arms:
    p=con.execute("""SELECT id, side, entry_price, exit_price, pnl_corrected, plan_id, close_reason, source
                     FROM trader_positions WHERE entry_order_id=?""",(a[7],)).fetchone()
    risk=abs(a[4]-a[5]); planned=abs(a[6]-a[4])/risk if risk else None
    if p and p[4] is not None and risk:
        realR = p[4]/ (risk*2.0)   # $ per point = 2 for MNQ, size 1
        rows.append((a[0],p[0],a[1],a[2],planned,realR,p[4],risk))
        print(f"  arm {a[0]:>3} -> pos {p[0]:>3}  {a[1]:<7} {a[2]:<4} risk={risk:6.2f}pts (${risk*2:6.2f})  planned R:R={planned:.3f}  pnl={p[4]:+8.2f}  realised R={realR:+.3f}  src={p[7]}")
    else:
        print(f"  arm {a[0]:>3} -> NO MATCHED POSITION (signal_id {a[7][:8]}…)")
if rows:
    pl=[r[4] for r in rows]; rr=[r[5] for r in rows]
    print(f"\nMATCHED n={len(rows)} ids arms {[r[0] for r in rows]} / positions {[r[1] for r in rows]}")
    print(f"  planned R:R  mean={st.mean(pl):.3f}  min={min(pl):.3f}  max={max(pl):.3f}")
    print(f"  realised R   mean={st.mean(rr):+.3f}  min={min(rr):+.3f}  max={max(rr):+.3f}  sum$={sum(r[6] for r in rows):+.2f}")
    wins=[r for r in rows if r[6]>0]
    print(f"  wins {len(wins)}/{len(rows)}; realised-to-planned ratio of means = {st.mean(rr)/st.mean(pl):.3f}")
