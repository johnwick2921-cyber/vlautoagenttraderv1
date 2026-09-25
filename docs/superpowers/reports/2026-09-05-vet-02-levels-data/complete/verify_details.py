#!/usr/bin/env python3
import pathlib,json,sqlite3,re,collections,math
p=pathlib.Path(__file__).resolve().parent
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row;c.execute('pragma query_only=ON');c.execute('BEGIN')
rows=lambda q:[dict(r) for r in c.execute(q)]
plans={(r['plan_id'],r['version']):json.loads(r['doc']) for r in rows('select plan_id,version,doc from plans')}
def ref(s):
 a=s.get('arm') or {};e=a.get('entry') if isinstance(a,dict) else None
 if e:return float(e),'arm.entry'
 m=re.findall(r'(\d{5}\.\d+|\d{5})',s.get('trigger') or '')
 return (float(m[0]),'first numeric trigger') if m else (None,None)
def match(d,px):
 if px is None:return []
 return [l for l in d.get('levels',[]) if str(l.get('label','')).startswith('RTH-L') and abs(float(l['price'])-px)<=1]
s=[]
for (pid,v),d in plans.items():
 for a in d.get('scenarios',[]):
  px,m=ref(a);lv=match(d,px)
  if lv:s.append({'plan_id':pid,'version':v,'scenario_id':a.get('id'),'condition':a.get('condition'),'arm_enabled':(a.get('arm') or {}).get('enabled',False),'reference':px,'method':m,'level_matches':lv,'scenario':a})
arms=rows("select id,plan_id,version,scenario,session,side,entry_px,condition,state,state_reason,fill_price,fill_quantity from armed_orders WHERE session<>'TEST-E7' ORDER BY id")
a=[];missing=[]
for r in arms:
 d=plans.get((r['plan_id'],r['version']))
 if d is None:missing.append(r['id']);continue
 lv=match(d,r['entry_px'])
 if lv:a.append(dict(r,level_matches=lv))
elig=json.loads((p/'positions.json').read_text())['eligible'];pos=[];unjoined=[]
for r in elig:
 d=plans.get((r['plan_id'],r['plan_version']))
 if not d:unjoined.append(r['id']);continue
 sc=next((s for s in d.get('scenarios',[]) if s.get('id')==r['cited_scenario_id']),None)
 if sc:
  px,m=ref(sc);lv=match(d,px)
  if lv:pos.append(dict(r,condition=sc.get('condition'),method=m,reference=px,level_matches=lv))
# Exactly reproduce prior-day registry RTH low with bar IDs (low available at interval close).
bars=rows("SELECT open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf='1m' AND open_time_ms>=1788442200000 AND open_time_ms<1788464700000 ORDER BY open_time_ms")
# bounds above =2026-09-03 08:30..14:45 CT, asserted from UTC conversion in audit records.
lo=min(b['l'] for b in bars)
res={'read_only':True,'trade_binding':rows('select id,name,strategy_id,is_running from traders'),'rthl_authored':s,'authored_conditions':dict(collections.Counter(s['condition'] for s in s)),'rthl_arms_exact_version':a,'arms_total_non_test':len(arms),'arms_missing_exact_plan':missing,'rthl_eligible_positions':pos,'eligible_missing_exact_plan':unjoined,'sept3_rth':{'window_ms':[1788442200000,1788464700000],'n_bars':len(bars),'low':lo,'low_bar_ids':[b['open_time_ms'] for b in bars if b['l']==lo]}}
(p/'details.json').write_text(json.dumps(res,indent=2)+'\n')
obs=json.loads((p/'forward_rows.json').read_text());first=[];seen=set()
for r in obs:
 key=(r['level_kind'],r['price'])
 if key not in seen:seen.add(key);first.append(r)
def wi(k,n):
 if n==0:return {'k':k,'n':0,'p':None,'wilson95':None}
 z=1.95996398454;v=k/n;d=1+z*z/n;m=(v+z*z/(2*n))/d;h=z*math.sqrt(v*(1-v)/n+z*z/(4*n*n))/d
 return {'k':k,'n':n,'p':v,'wilson95':[max(0,m-h),min(1,m+h)]}
grade=[]
for g in ['A','B','C']:
 rs=[r for r in first if r['seated'] and r['grade']==g and r['status']=='complete_60m'];ts=[r for r in rs if r['touch_ms']];h=sum(r['outcome']=='hold' for r in rs);b=sum(r['outcome']=='break' for r in rs);amb=sum(r['outcome']=='ambiguous_horizon' for r in rs)
 grade.append({'grade':g,'candidate_ids':[r['candidate_id'] for r in rs],'score_range':[min(r['score'] for r in rs),max(r['score'] for r in rs)] if rs else None,'outcomes':dict(collections.Counter(r['outcome'] for r in rs)),'touch':wi(len(ts),len(rs)),'hold':wi(h,h+b),'break':wi(b,h+b),'ambiguous':wi(amb,len(ts))})
(p/'grade_forward.json').write_text(json.dumps(grade,indent=2)+'\n')
print('RTH-L authored',res['authored_conditions'],'enabled',dict(collections.Counter(s['condition'] for s in s if s['arm_enabled'])))
print('RTH-L arms',a,'missingplan',missing)
print('RTH-L positions',pos)
print('RTH bounds',res['sept3_rth'])
print('grades',grade)
c.rollback();c.close()
