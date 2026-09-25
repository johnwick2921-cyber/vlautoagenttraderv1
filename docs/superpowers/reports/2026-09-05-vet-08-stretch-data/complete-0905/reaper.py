# Observed-row component replay of trader/reaper_snapshot.go:51,112.
# Does not synthesize broker events, or interpret mutable updated_at as initial risk.
import csv,json,pathlib,datetime,collections,bisect
R=pathlib.Path(__file__).resolve().parent
arms=list(csv.DictReader((R/'arms.csv').open()));sn=list(csv.DictReader((R/'snapshots.csv').open()));sn.sort(key=lambda r:int(r['received_at_ms']));ts=[int(r['received_at_ms']) for r in sn]
terminal={'filled','cancelled','canceled','rejected','expired','unknown','partfilled_done'};out=[]
for a in arms:
 if 'no order_update within stale window' not in a['state_reason']:continue
 now=int(datetime.datetime.fromisoformat(a['updated_at']).timestamp()*1000);j=bisect.bisect_left(ts,now)-1;s=sn[j] if j>=0 else None
 age=now-int(s['received_at_ms']) if s else None;v='unknown';matches=[]
 if a['signal_id'] and s and age<=60000:
  for o in json.loads(s['orders_json']):
   sig=a['signal_id'].strip().lower();name=o.get('name','').strip().lower()
   if o.get('symbol','').strip().upper()=='MNQ' and o.get('state','').strip().lower() not in terminal and (name in [sig,sig+'-sl',sig+'-tp',sig+'-lx'] or o.get('order_id','').lower()==sig):matches.append(o.get('order_id'))
  v='alive' if matches else 'gone'
 out.append(dict(arm_id=a['id'],opportunity=a['plan_id']+'/v'+a['version']+'/'+a['scenario'],observed_cancel_updated_at=a['updated_at'],latest_pre_cancel_snapshot=s['id'] if s else '',snapshot_received_age_ms=age if s else '',current_verdict_at_observed_time=v,action='cancel_if_silence_trigger_still_true' if v=='gone' else 'no_cancel',matched_order_ids=';'.join(matches),limitation='observed cancellation clock; assumes 30s configured interval and cache survived; not counterfactual inventory'))
with (R/'reaper_observed.csv').open('w') as f:
 w=csv.DictWriter(f,lineterminator="\n",fieldnames=list(out[0]));w.writeheader();w.writerows(out)
print(dict(collections.Counter(r['current_verdict_at_observed_time'] for r in out)));print([(r['arm_id'],r['current_verdict_at_observed_time'],r['latest_pre_cancel_snapshot']) for r in out])
