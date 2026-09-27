"""Summarize every registered geometry/fill cell; no best-cell selection.

Inference: circular moving-block bootstrap of five observed CME days, common
draws across cells. Bonferroni 95% two-sided family bounds for 3 buffers x 3
fills. The original held-out year has already been exposed in previous rounds.
"""
import collections
import csv
import json
import math
from pathlib import Path

import numpy as np

ROOT = Path('docs/superpowers/reports/2026-09-12-structural-stop/evidence')
groups = collections.defaultdict(list)
with open('data/structural-stop/sweep.jsonl') as f:
    for line in f:
        r = json.loads(line)
        groups[(r['Cell'], r['Fill'])].append(r)

def stats(rows):
    filled = [r for r in rows if r['Filled']]
    x = np.array([r['Net'] for r in filled])
    hi = np.array([r['NetUpper'] for r in filled])
    gains, losses = x[x > 0].sum(), -x[x < 0].sum()
    return dict(opportunities=len(rows), filled=len(filled),
        geometry_eligible=sum(r['Reason'] in ('', 'unfilled') for r in rows),
        mean_net=float(x.mean()) if len(x) else None,
        mean_optimistic_bound=float(hi.mean()) if len(hi) else None,
        profit_factor=float(gains/losses) if losses else None,
        ambiguity_n=sum(r['FillBarAmbiguous'] for r in filled),
        days=len({r['Day'] for r in filled}),
        reasons=dict(collections.Counter(r['Reason'] or 'filled' for r in rows)),
        owner_cap_unset_refusals=sum(r['ConfiguredAdmission']=='risk_cap_missing' for r in rows),
        median_one_contract_risk_usd=float(np.median([r['RiskUSD'] for r in filled])) if filled else None)

def occupancy(rows, ny_only):
    """Occupancy diagnostic, not reproduction of the live one-setup selector.
    Every selected unfilled opportunity reserves its first-touch minute too.
    This avoids silently selecting only eventual fills. Same-minute ties use ID.
    """
    available = -1
    selected = []
    for r in sorted(rows, key=lambda r:(r['Time'],r['ID'])):
        if ny_only and r['Session']!='NY': continue
        if r['Reason'] not in ('','unfilled') or r['Time']<available: continue
        available = (r['ExitTime'] if r['Filled'] else r['Time'])+60000
        if r['Filled']: selected.append(r)
    x = np.array([r['Net'] for r in selected])
    equity = np.r_[0, np.cumsum(x)]
    dd = np.maximum.accumulate(equity)-equity
    return dict(n=len(x),ids=[r['ID'] for r in selected],
        mean_net=float(x.mean()) if len(x) else None,
        max_drawdown_points=float(dd.max()),net_points=float(x.sum()),
        caveat='One-position occupancy diagnostic; original one-setup ranking/permission and queue selection not recreated. Not a claimed attainable live equity curve.')

surface=[]
rng=np.random.default_rng(20260912)
for era in ['all','in_sample','held_out']:
    subset={k:[r for r in rows if era=='all' or r['Era']==era] for k,rows in groups.items()}
    days=sorted({r['Day'] for rows in subset.values() for r in rows})
    index={d:i for i,d in enumerate(days)}
    n=len(days); B=4000;block=5
    starts=rng.integers(0,n,size=(B,math.ceil(n/block)))
    draw=((starts[:,:,None]+np.arange(block))%n).reshape(B,-1)[:,:n]
    for (cell,fill),rows in sorted(subset.items()):
        s=stats(rows);s.update(cell=cell,fill=fill,era=era)
        amounts=np.zeros(n);counts=np.zeros(n)
        for r in rows:
            if r['Filled']:
                i=index[r['Day']];amounts[i]+=r['Net'];counts[i]+=1
        denominator=counts[draw].sum(axis=1)
        draws=np.divide(amounts[draw].sum(axis=1),denominator,out=np.full(B,np.nan),where=denominator>0)
        draws=draws[np.isfinite(draws)]
        if len(draws) and s['mean_net'] is not None:
            delta=draws-s['mean_net']
            # Basic centered bootstrap confidence bounds; family of nine
            # candidate cells, control excluded. Validity is approximate and
            # relies on weak dependence, not symmetry of raw trade returns.
            s['family_lower_95']=float(s['mean_net']-np.quantile(delta,1-.025/9))
            s['family_upper_95']=float(s['mean_net']-np.quantile(delta,.025/9))
            s['adjusted_one_sided_p_positive']=min(1.,9*(1+np.sum(delta>=s['mean_net']))/(len(delta)+1))
        s['occupancy_all_sessions']=occupancy(rows,False)
        s['occupancy_NY_only']=occupancy(rows,True)
        surface.append(s)

with (ROOT/'e-complete-surface.json').open('w') as f:
    json.dump(dict(method='5-day circular moving-block bootstrap, 4000 draws, seed 20260912; Bonferroni 95% two-sided family=9 structural buffer/fill cells. Period splits are descriptive, held-out period previously exposed. No monetary cap supplied: geometry diagnostics do not authorize orders.',surface=surface),f,indent=2)
    f.write('\n')

fields=['ID','Day','Session','Era','Cell','Fill','Reason','ConfiguredAdmission','Filled','Entry','Stop','Target','RiskUSD','Net','NetUpper','Time','ExitTime','Exit','FillBarAmbiguous']
with (ROOT/'e-trades.csv').open('w',newline='') as f:
    w=csv.DictWriter(f,fieldnames=fields);w.writeheader()
    for rows in groups.values():
        for r in rows:
            if r['Filled']:w.writerow(r)

# Preserve every geometry opportunity's admission reason, not just fills.
with (ROOT/'geometry-opportunities.csv').open('w',newline='') as f:
    fields=['ID','Era','Session','Day','Cell','Reason','ConfiguredAdmission','Entry','Stop','Target','RiskUSD']
    w=csv.DictWriter(f,fieldnames=fields);w.writeheader()
    for (cell,fill),rows in groups.items():
        if fill!='A_touch' or cell=='legacy_corrected': continue
        for r in rows:
            d={k:r[k] for k in fields}
            d['Reason']='geometry_pass' if r['Reason'] in ('','unfilled') else r['Reason']
            w.writerow(d)

for s in surface:
    if s['era']=='all' and s['cell']!='legacy_corrected':
        print(s['cell'],s['fill'],'n',s['filled'],'net',round(s['mean_net'],4),'optimistic_bound',round(s['mean_optimistic_bound'],4),'simultaneous_CI',round(s['family_lower_95'],4),round(s['family_upper_95'],4))
