import csv,json,pathlib,collections,datetime
from zoneinfo import ZoneInfo
R=pathlib.Path(__file__).resolve().parent;CT=ZoneInfo('America/Chicago')
def read(n):return list(csv.DictReader((R/n).open()))
bars=read('bars.csv');cp=read('replay_checkpoints.csv');st=read('replay_static.csv');plans=json.loads((R/'plans.json').read_text());life=json.loads((R/'life_events.json').read_text());ops={r['opportunity']:r for r in read('opportunities.csv')};ss={(r['opportunity'],r['leg']):r for r in st}
rows=[]
for phase in ['all','even','odd']:
 for variant in ['implemented','corrected']:
  candidates=collections.defaultdict(list)
  for r in cp:
   if r['status']!='passes_known_local_checks':continue
   if phase!='all' and int(r['time_ms'])//60000%2 != (0 if phase=='even' else 1):continue
   field='actual_guard' if variant=='implemented' else 'corrected_guard'
   if r[field].startswith('send'):candidates[r['opportunity']].append(r)
  for key,rs in candidates.items():
   o=ops[key];end=int(o['end']);ev=[x['time_ms'] for x in life if x['date']==o['date'] and x['session']==o['session'] and str(x['version'])==o['version'] and x['event']=='DORMANT' and x['time_ms']>=int(o['born'])];end=min([end]+ev)
   hit=[]
   for r in rs:
    leg=ss[key,r['leg']];px=float(leg['entry']);long=o['direction']=='long'
    if r['kind']=='stop_entry':px=0 if variant=='implemented' else px+(.5 if long else -.5)
    for b in bars:
     t=int(b['open_time_ms'])
     if t<int(r['time_ms']) or t+60000>end:continue
     lo,hi=float(b['l']),float(b['h'])
     reached=(lo<=px if long else hi>=px) if r['kind']=='limit' else (hi>=px if long else lo<=px)
     if reached:hit.append((t,b['rowid'],px));break
   hit.sort()
   rows.append(dict(phase=phase,variant=variant,opportunity=key,send_checkpoints=len(rs),post_checkpoint_entry_reach=bool(hit),first_reach_time_ms=hit[0][0] if hit else '',bar_rowid=hit[0][1] if hit else '',price_reference=hit[0][2] if hit else '',fill_count_lower=0,fill_count_upper=1 if hit else 0,scope='independent opportunity; subsequent cancels/gates/queue not resolved; not portfolio bound'))
with (R/'reach_bounds.csv').open('w') as f:
 w=csv.DictWriter(f,lineterminator="\n",fieldnames=list(rows[0]));w.writeheader();w.writerows(rows)
out=[]
for phase in ['all','even','odd']:
 for v in ['implemented','corrected']:
  rs=[r for r in rows if r['phase']==phase and r['variant']==v];out.append(dict(phase=phase,variant=v,send_opportunities=len(rs),reachable_opportunities=sum(r['post_checkpoint_entry_reach'] for r in rs),ids=[r['opportunity'] for r in rs if r['post_checkpoint_entry_reach']]))
(R/'bounds_summary.json').write_text(json.dumps(out,indent=2));print(json.dumps(out,indent=2))
