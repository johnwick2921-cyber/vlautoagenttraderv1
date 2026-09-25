import sqlite3, json
c = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
rows = c.execute("""SELECT id, datetime(received_at_ms/1000,'unixepoch','-5 hours') ct, reason, orders_json
 FROM nt8_order_snapshots WHERE working_count>0 AND date(received_at_ms/1000,'unixepoch','-5 hours') BETWEEN '2026-09-03' AND '2026-09-04' ORDER BY id""").fetchall()
seen = {}
for id_, ct, reason, oj in rows:
    for o in json.loads(oj):
        k = (o.get('name','')[:8], o.get('type'), o.get('action'))
        v = (o.get('limit_price'), o.get('stop_price'), o.get('quantity'), o.get('state'))
        seen.setdefault(k, {}).setdefault(v, [id_, ct, 0])[2] += 1
print("-- distinct (name8,type,action) -> (limit_price, stop_price, qty, state): first snapshot id, ct, count")
for k in sorted(seen, key=lambda k: min(v[0] for v in seen[k].values())):
    for v, (fid, fct, n) in sorted(seen[k].items(), key=lambda x: x[1][0]):
        print("  ", k, v, "first", fid, fct, "n=", n)
