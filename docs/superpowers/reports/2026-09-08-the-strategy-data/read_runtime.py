#!/usr/bin/env python3
"""GET-only final runtime observation; credentials remain solely in memory."""
import pathlib,json,sqlite3,time,base64,hmac,hashlib,urllib.request,urllib.parse,subprocess,datetime,zoneinfo,re
P=pathlib.Path(__file__).parent
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
tid,uid=c.execute('SELECT id,user_id FROM traders').fetchone()
private=[tid,uid]+[r[0] for tab in ['trader_positions','nt8_order_snapshots'] for r in c.execute('SELECT DISTINCT account FROM '+tab) if r[0]]
env={}
for line in pathlib.Path('/home/hoang/nofx/.env').read_text().splitlines():
 k,sep,v=line.partition('=')
 if sep:env[k.strip()]=v.strip().strip('"').strip("'")
secret=env['JWT_SECRET'];now=int(time.time())
b64=lambda x:base64.urlsafe_b64encode(x).rstrip(b'=')
h=b64(json.dumps({'alg':'HS256','typ':'JWT'}).encode());b=b64(json.dumps({'user_id':uid,'iss':'nofxAI','iat':now,'nbf':now,'exp':now+180}).encode());payload=h+b'.'+b;token=(payload+b'.'+b64(hmac.new(secret.encode(),payload,hashlib.sha256).digest())).decode()
queries={}
for endpoint in ['/api/health','/api/positions','/api/cutover-gate','/api/desk']:
 url='http://127.0.0.1:8080'+endpoint+('' if endpoint.endswith('health') else '?'+urllib.parse.urlencode({'trader_id':tid}))
 req=urllib.request.Request(url,headers={'Authorization':'Bearer '+token})
 with urllib.request.urlopen(req,timeout=30) as r:queries[endpoint]=json.load(r)
pid=subprocess.check_output(['systemctl','show','nofx','--property=MainPID','--value'],text=True).strip();exe=pathlib.Path('/proc')/pid/'exe'
allow=['MIN_SL_ATR_MULT','ARM_STOP_ANCHOR_MAX_ATR','ARM_PLACE_TICKS','STOP_ENTRY_SEAM','EXIT_MECHS_SUSPENDED','HTF_VETO_MODE','SHADOW_CONDITIONS','LIVE_CONDITIONS']
initial=dict(v.decode().split('=',1) for v in (pathlib.Path('/proc')/pid/'environ').read_bytes().split(b'\0') if b'=' in v)
result={'read_ct':datetime.datetime.now(zoneinfo.ZoneInfo('America/Chicago')).isoformat(),'queries':queries,'pid':pid,'exe':str(exe.resolve()),'loaded_sha256':hashlib.sha256(exe.read_bytes()).hexdigest(),'disk_sha256':hashlib.sha256(pathlib.Path('/home/hoang/nofx/nofx-bin').read_bytes()).hexdigest(),'build':subprocess.check_output(['go','version','-m',str(exe)],text=True),'process_started_ct':subprocess.check_output(['ps','-p',pid,'-o','lstart='],text=True).strip(),'dotenv_allowlist':{k:env.get(k,'UNSET') for k in allow},'initial_process_env_allowlist':{k:initial.get(k,'UNSET') for k in allow}}
def clean(x):
 if isinstance(x,dict):return {k:clean(v) for k,v in x.items() if k not in ['account','user_id','trader_id']}
 if isinstance(x,list):return [clean(v) for v in x]
 if isinstance(x,str):
  for v in private:x=x.replace(v,'[redacted]')
  x=re.sub(r'\b(?:sk-|cm_)[A-Za-z0-9_-]{12,}','[redacted-key]',x)
 return x
(P/'runtime-final.json').write_text(json.dumps(clean(result),indent=2)+'\n')
print(json.dumps({'read_ct':result['read_ct'],'health':queries['/api/health'],'positions_n':len(queries['/api/positions']),'cutover_ready':queries['/api/cutover-gate']['ready'],'pid':pid,'loaded_sha256':result['loaded_sha256'],'disk_sha256':result['disk_sha256'],'env_allowlist':result['dotenv_allowlist']}))
