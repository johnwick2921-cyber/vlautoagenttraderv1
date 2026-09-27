#!/usr/bin/env python3
"""r02 — REBUILD the sample under the COMPLIANT population.
Correction to q04: the dispatch's 'UNRESOLVABLE' exclusion is plan_id='UNRESOLVABLE',
NOT a substring of pnl_correction_note. q04 tested the wrong column and kept 7 rows it
should have dropped (ids 530,539,545,546,566,571,580).
Laws: pnl_corrected only; exclude source='e7_farside_test'; exclude pnl_corrected IS NULL;
      exclude plan_id='UNRESOLVABLE'; era = plan_id set (id>=521).
Writes trade_sample_58.csv (compliant, primary) and trade_sample_65.csv (sensitivity)."""
import sqlite3, csv, datetime, zoneinfo
ct = zoneinfo.ZoneInfo("America/Chicago")
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
rows = con.execute("""SELECT id, plan_session, cited_scenario_id, pnl_corrected, entry_time, created_at, source,
                             pnl_correction_note, side, entry_price, exit_price, mae, mfe, close_reason, plan_id
                      FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' ORDER BY id""").fetchall()
print("plan-era rows:", len(rows), "ids", rows[0][0], "..", rows[-1][0])
excl = {"e7_farside_test":[], "pnl_null":[], "unresolvable_plan_id":[]}
compliant, broad = [], []
for (id_, sess, scen, pc, et, ca, src, note, side, ep, xp, mae, mfe, cr, pid) in rows:
    if src == "e7_farside_test": excl["e7_farside_test"].append(id_); continue
    if pc is None: excl["pnl_null"].append(id_); continue
    t = datetime.datetime.fromtimestamp(et/1000, ct)
    sday = (t - datetime.timedelta(hours=17)).date().isoformat()
    rec = dict(id=id_, session=sess or "", condition=scen or "", pnl_corrected=pc, session_day_ct=sday,
               created_ms=et, created_ct=t.strftime("%Y-%m-%d %H:%M:%S"), source=src, plan_id=pid,
               side=side, entry_price=ep, exit_price=xp, mae=mae, mfe=mfe, close_reason=cr or "")
    broad.append(rec)
    if pid == "UNRESOLVABLE": excl["unresolvable_plan_id"].append(id_); continue
    compliant.append(rec)
for k,v in excl.items(): print(f"excluded {k}: n={len(v)} ids={v}")
for name, out in (("COMPLIANT", compliant), ("BROAD(sensitivity)", broad)):
    s = sum(r["pnl_corrected"] for r in out)
    w = sum(1 for r in out if r["pnl_corrected"]>0); l = sum(1 for r in out if r["pnl_corrected"]<0); f0 = sum(1 for r in out if r["pnl_corrected"]==0)
    days = sorted({r["session_day_ct"] for r in out})
    print(f"{name}: n={len(out)} ids {out[0]['id']}..{out[-1]['id']} sum={s:+.2f} mean={s/len(out):+.4f} W/L/F={w}/{l}/{f0} days={len(days)} {days}")
for fn, out in (("trade_sample_58.csv", compliant), ("trade_sample_65.csv", broad)):
    with open(fn,"w",newline="") as fh:
        wr = csv.DictWriter(fh, fieldnames=list(out[0])); wr.writeheader(); wr.writerows(out)
    print("wrote", fn)
