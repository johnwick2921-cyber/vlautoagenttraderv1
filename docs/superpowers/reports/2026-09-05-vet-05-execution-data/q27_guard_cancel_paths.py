import sqlite3, datetime as dt
c = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
arms = c.execute("""SELECT id, scenario, side, entry_px, stop_px, target_px,
  datetime(strftime('%s',updated_at),'unixepoch','-5 hours') cancel_ct, strftime('%s',updated_at)*1000 cancel_ms
  FROM armed_orders WHERE state_reason LIKE 'level accepted through%' ORDER BY id""").fetchall()
print("arm | side entry | cancel_ct | close@cancel | thru_pts | ret_to_entry_5m/15m/30m | MFE30(dir) | MAE30 | stop_hit30 | tgt_hit30")
for id_, sc, side, e, s, t, cct, cms in arms:
    side = side.lower()
    bars = c.execute("""SELECT open_time_ms, o,h,l,c FROM bars WHERE tf='1m' AND symbol='MNQ'
        AND open_time_ms >= ? - 60000 AND open_time_ms < ? + 30*60000 ORDER BY open_time_ms""", (cms, cms)).fetchall()
    if not bars: print(id_, sc, side, e, cct, "NO BARS"); continue
    last = [b for b in bars if b[0] <= cms]
    p0 = last[-1][4] if last else bars[0][1]
    thru = (p0 - e) if side=='short' else (e - p0)
    def window(mins):
        return [b for b in bars if cms <= b[0] < cms + mins*60000]
    def touched(mins):
        w = window(mins)
        return any((b[2] >= e) if side=='short' else (b[3] <= e) for b in w)
    w30 = window(30)
    if side=='short':
        mfe = max((p0 - b[3]) for b in w30); mae = max((b[2] - p0) for b in w30)
        stop_hit = any(b[2] >= s for b in w30); tgt_hit = any(b[3] <= t for b in w30)
    else:
        mfe = max((b[2] - p0) for b in w30); mae = max((p0 - b[3]) for b in w30)
        stop_hit = any(b[3] <= s for b in w30); tgt_hit = any(b[2] >= t for b in w30)
    print(f"{id_:>3} {sc} | {side} {e:.2f} | {cct} | {p0:.2f} | {thru:+.2f} | {touched(5)}/{touched(15)}/{touched(30)} | {mfe:.2f} | {mae:.2f} | {stop_hit} | {tgt_hit}  (stop {s:.2f} tgt {t:.2f}, bars={len(w30)})")
