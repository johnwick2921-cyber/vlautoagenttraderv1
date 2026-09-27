"""Audit copied application log lines and the verified read-only DB backup.
Input was captured with rg -n '🛑 arm stop ' ... | tail -200. These are logged
composition changes, never represented as 200 independent arms or all attempts.
"""
import collections
import datetime as dt
import json
from pathlib import Path
import re
import sqlite3
from zoneinfo import ZoneInfo

c=sqlite3.connect('file:data/structural-stop/backtest.db?mode=ro',uri=True)
plans=[]
for pid,v,created,raw in c.execute('SELECT plan_id,version,created_at,doc FROM plans'):
    t=dt.datetime.fromisoformat(created)
    if t.tzinfo is None:t=t.replace(tzinfo=dt.timezone.utc)
    plans.append((t,pid,v,json.loads(raw)))
plans.sort(key=lambda x:x[0],reverse=True)
rows=[]
for line in Path('data/structural-stop/last200-stop-lines.txt').read_text().splitlines():
    m=re.search(r'^(.*?):(\d+):(\d\d-\d\d \d\d:\d\d:\d\d).*?trader_id=(\S+).*?arm stop (\S+) (\S+) leg (\d+) (long|short): stop ([\d.]+).*?bound=(\w+)',line)
    if not m:raise ValueError('unparsed line '+line[:160])
    path,no,ts,tid,sess,sc,leg,side,stop,bound=m.groups()
    t=dt.datetime.strptime('2026-'+ts,'%Y-%m-%d %H:%M:%S').replace(tzinfo=ZoneInfo('America/Chicago'))
    row=dict(id=Path(path).name+':'+no,time_ct=t.isoformat(),session=sess,scenario=sc,leg=int(leg),side=side,stop=float(stop),bound=bound)
    a=re.search(r'authored ([\d.]+)',line);authored=float(a[1]) if a else float(stop)
    candidate=next(((pid,v,d) for pt,pid,v,d in plans if pt<=t+dt.timedelta(seconds=1) and pid.endswith(':'+tid) and ':'+sess+':' in pid),None)
    if candidate:
        pid,v,d=candidate
        s=next((s for s in d.get('scenarios',[]) if s['id']==sc),None)
        ar=s.get('arm',{}) if s else {};legs=ar.get('legs') or [ar]
        l=legs[int(leg)-1] if len(legs)>=int(leg) else {}
        if abs(l.get('stop',0)-authored)<=.011:
            entry=l['entry'];row.update(plan_id=pid,version=v,entry=entry)
            anchor=re.search(r'→ beyond ([\d.]+)',line);floor=re.search(r'atr_floor ([\d.]+)',line)
            row['anchor_distance']=abs(entry-float(anchor[1])) if anchor else None
            row['atr_floor_distance']=abs(entry-float(floor[1])) if floor else None
            row['stop_distance']=abs(entry-float(stop))
    row['raw']=line
    rows.append(row)

def q(a,p):
    a=sorted(a);x=(len(a)-1)*p;i=int(x)
    return a[i]+(a[min(i+1,len(a)-1)]-a[i])*(x-i)
summary=dict(n=len(rows),cohort='last 200 logged changed/unanchored composition lines',counts=dict(collections.Counter(r['bound'] for r in rows)),
    matched_plan_n=sum('plan_id' in r for r in rows),
    distinct_plan_scenario_legs=len({(r['plan_id'],r['version'],r['scenario'],r['leg']) for r in rows if 'plan_id' in r}),
    limitations='The complete composition-attempt denominator is not persisted. Unchanged anchored/authored winners can be absent; log rounding limits distances. Latest preceding plan must match authored stop, otherwise no reconstructed entry is reported. These logs precede current running revision; the inspected old composer is unchanged between those revisions.')
summary['distances']={}
for k in ['anchor_distance','atr_floor_distance','stop_distance']:
    a=[r[k] for r in rows if r.get(k) is not None]
    summary['distances'][k]=dict(n=len(a),mean=sum(a)/len(a),median=q(a,.5),p75=q(a,.75),p90=q(a,.9),minimum=min(a),maximum=max(a)) if a else dict(n=0)
path=Path('docs/superpowers/reports/2026-09-12-structural-stop/evidence/c1-compositions.json')
path.write_text(json.dumps(dict(summary=summary,rows=rows),indent=2)+'\n')
print(json.dumps(summary,indent=2))
