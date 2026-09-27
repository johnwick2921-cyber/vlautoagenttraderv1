# q05: peek one plan doc's structure (latest NY 2026-09-04, max version) — keys only + one scenario in full
import sqlite3, json
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
pid,ver,doc,ind=con.execute("SELECT plan_id, version, doc, indicators_block FROM plans WHERE trade_date='2026-09-04' AND session='NY' ORDER BY version DESC LIMIT 1").fetchone()
d=json.loads(doc)
print('plan', pid[:30], 'v', ver, 'top keys:', list(d.keys()))
def show(k):
    v=d.get(k)
    s=json.dumps(v, ensure_ascii=False)
    print(f'-- {k}: {s[:600]}')
for k in d:
    if k!='scenarios': show(k)
sc=d.get('scenarios') or []
print('scenarios n', len(sc))
if sc:
    print('scenario keys', list(sc[0].keys()))
    print(json.dumps(sc[0], ensure_ascii=False, indent=1)[:3000])
