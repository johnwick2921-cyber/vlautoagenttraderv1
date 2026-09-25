import sqlite3,json,csv,re,math,hashlib,datetime,zoneinfo,pathlib,urllib.request,urllib.error,base64,hmac,time,os
W=pathlib.Path('/home/hoang/nofx-vet-07-complete'); D=W/'docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data'; CT=zoneinfo.ZoneInfo('America/Chicago')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); c.row_factory=sqlite3.Row;c.execute('pragma query_only=on');c.execute('begin')
def rows(q,args=()):return [dict(r) for r in c.execute(q,args)]
def save(n,x): (D/n).write_text(json.dumps(x,indent=2,ensure_ascii=False)+'\n')
def csvout(n,rs):
 with (D/n).open('w') as f:
  w=csv.DictWriter(f,fieldnames=list(rs[0]));w.writeheader();w.writerows(rs)
def wi(k,n):
 if not n:return None
 p=k/n;z=1.96;d=1+z*z/n;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n));return [round((p+z*z/(2*n)-h)/d,6),round((p+z*z/(2*n)+h)/d,6)]
# Seven complete CT calendar days; timestamp filter, never session-label filter.
start='2026-08-29 05:00:00';end='2026-09-05 05:00:00'
r=rows('select id,trade_date,session,attempt,reject_reason,created_at,prompt_text from planner_rejected_prompts where datetime(created_at)>=datetime(?) and datetime(created_at)<datetime(?) order by id',(start,end))
for x in r:
 x['prompt_shape']='full' if '# DAY-PLAN READER' in x['prompt_text'] else 'repair'
 x['sha256']=hashlib.sha256(x['prompt_text'].encode()).hexdigest()
 if x['id'] in (132,131): (D/f'planner-{x["id"]}-actual.txt').write_text(x['prompt_text'])
 del x['prompt_text']
csvout('rejects-seven-days.csv',r)
e=rows('select id,timestamp,system_prompt,input_prompt from decision_records where length(system_prompt)>0 and length(input_prompt)>0 order by id desc limit 1')[0]
for k in ('system_prompt','input_prompt'): (D/f'executor-{e["id"]}-{k}.txt').write_text(e[k])
facts=rows('select id,trade_date,session,plan_id,version,prompt_hash,void_count,stop_floor_pts,atr5m,stop_floor_mlt,bias_ai,bias_tree,bias_regime,tokens_in,scope_since_ms,scope_bars,scope_intv,created_at from planner_read_facts order by id');csvout('read-facts.csv',facts)
q='''select id,source,plan_id,plan_version,cited_scenario_id,plan_session,entry_time,exit_time,pnl_corrected from trader_positions where entry_time>=1786770000000 and status='CLOSED' and pnl_corrected is not null and coalesce(source,'')!='e7_farside_test' and coalesce(plan_id,'')!='UNRESOLVABLE' order by id'''
p=rows(q); by={};days=set()
for x in p:
 dt=datetime.datetime.fromtimestamp(x['entry_time']/1000,CT); day=(dt+datetime.timedelta(hours=7)).date().isoformat();days.add(day);x['session_day_17ct']=day
 doc=c.execute('select doc from plans where plan_id=? and version=?',(x['plan_id'],x['plan_version'])).fetchone()
 sc=next((s for s in json.loads(doc[0]).get('scenarios',[]) if s.get('id')==x['cited_scenario_id']),{}) if doc else {}
 x['condition']=sc.get('condition','unmatched');by.setdefault(x['condition'],[]).append(x)
csvout('eligible-positions.csv',p)
counts={k:{'n':len(v),'wins':sum(x['pnl_corrected']>0 for x in v),'pnl_corrected':round(sum(x['pnl_corrected'] for x in v),6),'ids':[x['id'] for x in v],'win_wilson95':wi(sum(x['pnl_corrected']>0 for x in v),len(v))} for k,v in by.items()}
save('population.json',{'query':q,'n':len(p),'sum_pnl_corrected':round(sum(x['pnl_corrected'] for x in p),6),'wins':sum(x['pnl_corrected']>0 for x in p),'losses':sum(x['pnl_corrected']<0 for x in p),'flats':sum(x['pnl_corrected']==0 for x in p),'days_17ct':sorted(days),'win_wilson95':wi(sum(x['pnl_corrected']>0 for x in p),len(p)),'conditions':counts,'post_strict_boot_positions':[x['id'] for x in p if x['entry_time']>=1788451833000],'trade_excursions_n':c.execute('select count(*) from trade_excursions').fetchone()[0]})
# Resolve owner identity in memory only; never print or persist auth material.
t=c.execute('select id,user_id,strategy_id from traders where is_running=1 limit 1').fetchone()
if not t:t=c.execute('select id,user_id,strategy_id from traders limit 1').fetchone()
s=c.execute('select config,updated_at from strategies where id=?',(t['strategy_id'],)).fetchone(); cfg=json.loads(s['config'])
# Include only trading blocks; recursively redact any credentials.
def redact(x):
 if isinstance(x,dict): return {k:('<redacted>' if re.search('key|secret|password|token|account|email',k,re.I) else redact(v)) for k,v in x.items()}
 if isinstance(x,list):return [redact(v) for v in x]
 return x
save('saved-trading-settings.json',{'updated_at':s['updated_at'],'day_plan':cfg.get('day_plan'),'regime':cfg.get('regime'),'ai_config':redact(cfg.get('ai_config'))})
meta={'captured_at_ct':datetime.datetime.now(CT).isoformat(),'base':'b4376246c2c502ecedd119c6a44a27956ed2f616','interval_ct':'[2026-08-29 00:00,2026-09-05 00:00)','interval_utc':[start,end],'sqlite_query_only':c.execute('pragma query_only').fetchone()[0],'reject_count':len(r),'reject_ids':[x['id'] for x in r],'executor_id':e['id'],'executor_timestamp':e['timestamp'],'planner132_created_at':next(x['created_at'] for x in r if x['id']==132),'readfacts_ids':[x['id'] for x in facts]}
for path in ('/api/health','/api/config/resolved'):
 try:
  resp=urllib.request.urlopen('http://127.0.0.1:8080'+path,timeout=10);save('api-'+path.split('/')[-1]+'.json',json.load(resp));meta[path]=resp.status
 except urllib.error.HTTPError as ex:meta[path]=ex.code
if meta['/api/config/resolved']==401:
 secret=None
 for line in pathlib.Path('/home/hoang/nofx/.env').read_text().splitlines():
  if line.startswith('JWT_SECRET='):secret=line.split('=',1)[1].strip().strip('\"\'')
 if secret:
  u=c.execute('select id,email from users where id=?',(t['user_id'],)).fetchone();now=int(time.time())
  b64=lambda b:base64.urlsafe_b64encode(b).rstrip(b'=')
  msg=b64(b'{"alg":"HS256","typ":"JWT"}')+b'.'+b64(json.dumps({'user_id':u['id'],'email':u['email'],'iss':'nofxAI','iat':now,'nbf':now,'exp':now+300},separators=(',',':')).encode())
  tok=(msg+b'.'+b64(hmac.new(secret.encode(),msg,hashlib.sha256).digest())).decode()
  req=urllib.request.Request('http://127.0.0.1:8080/api/config/resolved?trader_id='+t['id']+'&session=NY',headers={'Authorization':'Bearer '+tok})
  try:
   resp=urllib.request.urlopen(req,timeout=10);save('api-resolved.json',json.load(resp));meta['/api/config/resolved_authenticated']=resp.status
  except urllib.error.HTTPError as ex:meta['/api/config/resolved_authenticated']=ex.code
save('capture-meta.json',meta);c.rollback();c.close()
print(json.dumps(meta,indent=2));print((D/'population.json').read_text())
