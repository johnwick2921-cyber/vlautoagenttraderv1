#!/usr/bin/env python3
"""Independent stdlib cash-path checks against committed statistical outputs."""
import csv,json,pathlib,statistics,hashlib,datetime
from zoneinfo import ZoneInfo
p=pathlib.Path(__file__).resolve().parent;r=json.loads((p/'results.json').read_text());rows=list(csv.DictReader(open(p/'trade_sample.csv')))
ids=[int(z['id']) for z in rows]; xx=[float(z['pnl_corrected']) for z in rows]
assert len(set(ids))==58 and len(xx)==58
assert not set(ids)&{530,539,545,546,566,571,580,572,573,574,576,577,579}
assert abs(sum(xx)+466.428572)<1e-8
assert abs(statistics.stdev(xx)-r['sd'])<1e-10
assert sum(v>0 for v in xx)==18 and sum(v<0 for v in xx)==38 and sum(v==0 for v in xx)==2
cash=peak=dd=0.;run=streak=0
for value in xx:
 cash+=value;peak=max(peak,cash);dd=max(dd,peak-cash);run=run+1 if value<0 else 0;streak=max(streak,run)
assert abs(dd-969)<1e-8 and streak==7
assert r['post_strict_ids']==[] and r['cross_day_ids']==[] and r['entry_exit_order_agrees']
byday={}
for row in rows:
 local=datetime.datetime.fromtimestamp(int(row['exit_time'])/1000,ZoneInfo('America/Chicago'))
 label=(local-datetime.timedelta(hours=17)).date().isoformat();byday.setdefault(label,[]).append(float(row['pnl_corrected']))
assert len(byday)==12
trips=0
for day,xs in byday.items():
 cash=0
 for value in xs:
  cash+=value
  if cash<=-450:trips+=1;break
assert trips==0
for h,v in r['trade_horizons'].items():
 assert list(v['drawdown'].values())==sorted(v['drawdown'].values())
 for k,z in v['streaks'].items():
  assert z['n']==100000 and z['wilson95'][0]<=z['rate']<=z['wilson95'][1]
assert (p/'offline-check/results.json').read_bytes()==(p/'results.json').read_bytes()
print('PASS: 58 unique eligible IDs, all exclusions, sum, W/L/flats, independent SD, closed DD $969, losing streak 7, 12 exit-based CME days, zero $450 trips, strict n0, day/order equivalence, ordered quantiles, Wilson bounds, offline byte-for-byte results.')
print('Results SHA256',hashlib.sha256((p/'results.json').read_bytes()).hexdigest())
print('Original rig unchanged SHA256',hashlib.sha256((p/'legacy_mc_drawdown.py').read_bytes()).hexdigest())
