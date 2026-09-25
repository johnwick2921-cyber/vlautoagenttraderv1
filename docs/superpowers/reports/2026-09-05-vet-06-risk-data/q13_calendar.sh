#!/bin/bash
# q13 — calendar slices in the store: coverage, FOMC/red events, upcoming dates; static fallback path
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
echo "--- calendar tables ---"; sqlite3 "$DB" "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%calendar%';"
T=$(sqlite3 "$DB" "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%calendar%' LIMIT 1;")
echo "--- columns of $T ---"; sqlite3 "$DB" "PRAGMA table_info($T);" | cut -d'|' -f2 | tr '\n' ' '; echo
echo "--- slices: count, min/max trade_date ---"; sqlite3 -header "$DB" "SELECT COUNT(*), MIN(trade_date), MAX(trade_date) FROM $T;" 2>&1
echo "--- last 12 slices: trade_date, n events, red events, source ---"
python3 - <<PY
import sqlite3, json
con = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
cols=[r[1] for r in con.execute("PRAGMA table_info($T)")]
print("cols:", cols)
dcol = 'trade_date' if 'trade_date' in cols else cols[0]
ecol = 'events_json' if 'events_json' in cols else [c for c in cols if 'json' in c.lower()][0]
rows = con.execute(f"SELECT {dcol}, {ecol}, * FROM $T ORDER BY {dcol} DESC LIMIT 14").fetchall()
fomc_any=0
for r in rows:
    try: evs=json.loads(r[1])
    except Exception as e: print(r[0], "unparseable", e); continue
    red=[e for e in evs if str(e.get('impact','')).lower() in ('high','red','3')]
    names=[str(e.get('title') or e.get('name') or e.get('event') or '')[:28] for e in red]
    print(f"{r[0]}: n={len(evs)} red={len(red)} {names[:5]}")
allrows = con.execute(f"SELECT {dcol}, {ecol} FROM $T").fetchall()
fomc=[r[0] for r in allrows if 'fomc' in (r[1] or '').lower() or 'fed interest' in (r[1] or '').lower() or 'federal funds' in (r[1] or '').lower()]
print("slices mentioning FOMC/Fed rate:", fomc)
keys=set()
for r in allrows:
    try:
        for e in json.loads(r[1] or '[]'): keys.update(e.keys())
    except: pass
print("event keys:", sorted(keys))
PY
echo "--- static fallback path ---"; grep -n "calendarStaticPath\|static.*\.json\|T1_STATIC\|CALENDAR_STATIC" /home/hoang/nofx-vet-06/trader/auto_trader_calendar.go | head -5
