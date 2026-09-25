#!/bin/bash
# q09 — the BOUND strategy's guardrail knob VALUES (traders.strategy_id -> strategies), never LIMIT 1 (class 9)
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
echo "--- traders (id, name, strategy_id, exchange, is_running) ---"
sqlite3 "$DB" "SELECT id, name, strategy_id, exchange_id, is_running FROM traders;" 2>&1 | cut -c1-200
echo "--- strategies columns ---"
sqlite3 "$DB" "PRAGMA table_info(strategies);" | cut -d'|' -f2 | tr '\n' ' '; echo
echo "--- bound strategy config: risk-related keys ---"
python3 - <<'PY'
import sqlite3, json
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
trs = con.execute("SELECT id, name, strategy_id FROM traders").fetchall()
for tid, name, sid in trs:
    row = con.execute("SELECT name, config FROM strategies WHERE id=?", (sid,)).fetchone()
    if not row: print("trader", name, "strategy", sid, "NOT FOUND"); continue
    sname, cfg = row
    print(f"trader={name} strategy={sname} ({sid[:8]}…)")
    try: c = json.loads(cfg)
    except Exception as e: print("  config parse error", e); continue
    def walk(o, path=""):
        if isinstance(o, dict):
            for k,v in o.items(): walk(v, path+"."+k)
        elif isinstance(o, list):
            for i,v in enumerate(o[:30]): walk(v, path+f"[{i}]")
        else:
            s = path.lower()
            if any(t in s for t in ("guardrail","daily","contract","max_trades","max_positions","loss","profit","blackout","consist","eod","flat","min_sl","atr","stop","risk","hold_lock","breakeven","trail","size","leverage","strict","plan_mode","max_daily")):
                print(f"  {path} = {o!r}")
    walk(c)
PY
