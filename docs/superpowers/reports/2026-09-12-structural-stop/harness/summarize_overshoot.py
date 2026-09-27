"""Reproduce C5 and preserve the complete event denominator from the frozen cache.
Uses original H12 labels, not profitable-trade selection. Missing primary-source
TF is reported as missing; it is never inferred from another zone member.
"""
import collections
import csv
import json
from pathlib import Path
import numpy as np

root=Path('docs/superpowers/reports/2026-09-12-structural-stop/evidence')
held=[];outcomes=collections.Counter();reads=set();cohort=0
with open('data/structural-stop/events.jsonl') as source, (root/'event-index.csv').open('w', newline='') as output:
    fields=['ID','Read','Day','Session','Contract','Era','Side','Outcome','TF','Families','Time','Entry','Lo','Hi','ATR']
    writer=csv.DictWriter(output,fieldnames=fields);writer.writeheader()
    for line in source:
        e=json.loads(line);writer.writerow({k:e[k] for k in fields})
        cohort+=1;reads.add(e['Read']);outcomes[e['Outcome']]+=1
        if e['Outcome']=='hold':
            held.append(dict(id=e['ID'],era=e['Era'],session=e['Session'],tf=e['TF'],families=e['Families'],points=e['Penetration'],zone_fraction=e['Penetration']/(e['Hi']-e['Lo']),atr_fraction=e['Penetration']/e['ATR'] if e['ATR']>0 else None))
groups=collections.defaultdict(list)
for e in held:
    for k in ['all',e['era'],'tf='+e['tf'],'families='+str(e['families']),'session='+e['session']]: groups[k].append(e)
summary={}
for k,rows in groups.items():
    stats={}
    for col in ['points','zone_fraction','atr_fraction']:
        values=[e[col] for e in rows if e[col] is not None]
        stats[col]=dict(zip(['p50','p75','p90','p95'],[float(x) for x in np.quantile(values,[.5,.75,.9,.95])])) if values else None
    summary[k]=dict(n=len(rows),statistics=stats)
result=dict(cohort_n=cohort,reads_with_touches_n=len(reads),outcomes=dict(outcomes),method='Original H12 close hold/break labels; maximum far-edge penetration from touch bar through label exit. Touch-bar OHLC extreme may precede touch: upper bound, not tick path. Winners-only conditional table does not give loss protection. Parameters from in_sample only.',summary=summary,held_event_ids_and_values=held)
original=json.loads((root/'c5-overshoot.json').read_text())
# Read denominator includes reads with zero touches; use the original detector
# run's recorded denominator only after verifying every emitted event and split.
assert original['cohort_n']==cohort and original['outcomes']==result['outcomes']
assert original['summary']==summary and original['held_event_ids_and_values']==held
print(f'C5 independent cache summary matches: {cohort} events, {len(held)} held; {len(reads)} reads with touches, original run {original["reads_n"]} reads total')
