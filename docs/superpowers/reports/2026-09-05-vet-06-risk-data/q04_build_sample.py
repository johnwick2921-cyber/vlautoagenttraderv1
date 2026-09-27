#!/usr/bin/env python3
"""q04 — rebuild trade_sample.csv from the store as of today, mode=ro.
Era = plan era: plan_id set (first row id 521, 2026-08-19 03:22 CT).
Laws: pnl_corrected only; exclude source='e7_farside_test'; exclude pnl_corrected IS NULL (counted);
      exclude 'UNRESOLV' notes (counted). session_day_ct = CT date of (entry_time - 17h): the CME
      session-day labelled by its 17:00 CT OPEN date — same convention as the 09-03 rig (id 521 at
      2026-08-19 03:22 CT -> 2026-08-18)."""
import sqlite3, csv, datetime, zoneinfo
ct = zoneinfo.ZoneInfo("America/Chicago")
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
rows = con.execute("""SELECT id, plan_session, cited_scenario_id, pnl_corrected, entry_time, created_at, source,
                             pnl_correction_note, side, entry_price, exit_price, mae, mfe, close_reason, plan_id
                      FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' ORDER BY id""").fetchall()
print("plan-era rows:", len(rows), "ids", rows[0][0], "..", rows[-1][0])
excl = {"e7_farside_test":[], "pnl_null":[], "unresolvable":[]}
out = []
for (id_, sess, scen, pc, et, ca, src, note, side, ep, xp, mae, mfe, cr, pid) in rows:
    if src == "e7_farside_test": excl["e7_farside_test"].append(id_); continue
    if pc is None: excl["pnl_null"].append(id_); continue
    if note and "UNRESOLV" in note.upper(): excl["unresolvable"].append(id_); continue
    t = datetime.datetime.fromtimestamp(et/1000, ct)
    sday = (t - datetime.timedelta(hours=17)).date().isoformat()
    out.append(dict(id=id_, session=sess or "", condition=scen or "", pnl_corrected=pc, session_day_ct=sday,
                    created_ms=et, created_ct=t.strftime("%Y-%m-%d %H:%M:%S"), source=src,
                    side=side, entry_price=ep, exit_price=xp, mae=mae, mfe=mfe, close_reason=cr or ""))
for k,v in excl.items(): print(f"excluded {k}: n={len(v)} ids={v}")
print("usable n =", len(out), "ids", out[0]["id"], "..", out[-1]["id"], "sum =", round(sum(r["pnl_corrected"] for r in out),2))
with open("trade_sample.csv","w",newline="") as f:
    w = csv.DictWriter(f, fieldnames=list(out[0])); w.writeheader(); w.writerows(out)
days = sorted({r["session_day_ct"] for r in out}); print("session-days:", len(days), days)
