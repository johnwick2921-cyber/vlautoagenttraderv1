"""Verify C2 against raw positive-price ledger rows and inventory C3 plan maps.
No trade P&L is read: risk/target geometry is recomputed from stored prices.
"""
import csv
import json
import sqlite3
from pathlib import Path

root=Path('docs/superpowers/reports/2026-09-12-structural-stop/evidence')
c=sqlite3.connect('file:data/structural-stop/backtest.db?mode=ro',uri=True)
c.row_factory=sqlite3.Row
rows=list(c.execute('SELECT id,plan_id,version,scenario,leg_index,side,entry_px,stop_px,target_px,state,created_at,entry_class FROM armed_orders WHERE entry_px>0 AND stop_px>0 AND target_px>0 ORDER BY id DESC LIMIT 200'))
stored=json.loads((root/'c2-ledger.json').read_text())
assert [r['id'] for r in rows]==stored['summary']['ids']
for row,evidence in zip(rows,stored['rows']):
    for key in row.keys(): assert row[key]==evidence[key], (row['id'],key)
    risk=abs(row['entry_px']-row['stop_px']);target=abs(row['target_px']-row['entry_px'])
    assert abs(risk-evidence['risk_points'])<1e-8
    assert abs(target-evidence['target_points'])<1e-8
    assert abs(target/risk-evidence['rr'])<1e-8
selected=[r for r in rows if r['entry_class']=='armed_fill']
assert [r['id'] for r in selected]==stored['confirmed_arm_class']['ids']
plans=list(c.execute('SELECT plan_id,version,created_at,doc FROM plans ORDER BY created_at,plan_id,version'))
with (root/'c3-plans.csv').open('w',newline='') as f:
    w=csv.writer(f);w.writerow(['plan_id','version','created_at_db','zone_map_present'])
    for p in plans: w.writerow([p['plan_id'],p['version'],p['created_at'],json.loads(p['doc']).get('zone_map') is not None])
print(f'C2 parity: {len(rows)} positive-price ledger rows, {len(selected)} armed_fill rows; C3: {len(plans)} plans, {sum(json.loads(p["doc"]).get("zone_map") is not None for p in plans)} maps')
