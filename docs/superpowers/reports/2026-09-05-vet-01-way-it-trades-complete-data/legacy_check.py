"""Recompute historical published claims, explicitly not primary evidence of strategy edge."""
import csv,json,statistics as st,shutil,math
from pathlib import Path
root=Path('/home/hoang/nofx-vet-01-complete/docs/superpowers/reports')
if not Path('legacy_floor_input.csv').exists():shutil.copyfile(root/'2026-09-04-research-conformance-data/E-d3-mae-mfe-per-trade.csv','legacy_floor_input.csv')
if not Path('legacy_plan_geometry.csv').exists():
 rows=[]
 for line_no,line in enumerate((root/'exports/2026-09-02-losses/plans.jsonl').read_text().splitlines(),1):
  p=json.loads(line);d=json.loads(p['doc'])
  for sc in d.get('scenarios') or []:
   a=sc.get('arm') or {};e=a.get('entry');sl=a.get('stop');tp=a.get('target')
   if e and sl and tp and e!=sl:rows.append(dict(source_line=line_no,plan_id=p['plan_id'],version=p['version'],scenario=sc['id'],entry=e,stop=sl,target=tp,rr=abs(tp-e)/abs(e-sl)))
 with open('legacy_plan_geometry.csv','w',newline='') as f:w=csv.DictWriter(f,fieldnames=list(rows[0]),lineterminator="\n");w.writeheader();w.writerows(rows)
r=list(csv.DictReader(open('legacy_floor_input.csv')));f=[x for x in r if x['floor_pts']];eligible=set(json.load(open('summary.json'))['ids']);g=[x for x in f if int(x['id']) in eligible]
def desc(rows):
 hits=[int(x['id']) for x in rows if float(x['mfe'])>=2*float(x['floor_pts'])];n=len(rows);p=len(hits)/n;z=1.96;dd=1+z*z/n;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n));return dict(n=n,ids=[int(x['id']) for x in rows],hits=hits,win_wilson=[p,(p+z*z/(2*n)-h)/dd,(p+z*z/(2*n)+h)/dd])
ps=list(csv.DictReader(open('legacy_plan_geometry.csv')))
out={'published_floor_subset':desc(f),'eligible_intersection_same_legacy_floor':desc(g),'published_planned_rr':{'n':len(ps),'median':st.median(float(x['rr']) for x in ps),'source':'exports/2026-09-02-losses/plans.jsonl; selected identities in legacy_plan_geometry.csv'},'floor_to_atr_max_error':max(abs(float(x['floor_pts'])/float(x['atr5m'])-1.5) for x in f),'limitation':'This reproduces published proxy arithmetic. It does not verify initial runtime ATR/floor, ordered MFE/stop path, or actual target outcomes. The 54-row fresh diagnostic uses a different causal bar coverage rule.'}
Path('legacy_check.json').write_text(json.dumps(out,indent=2))
with open('legacy_check.txt','w') as f:
 for k,v in out.items():f.write(k+' '+json.dumps(v)+'\n')
print(json.dumps(out,indent=2))
