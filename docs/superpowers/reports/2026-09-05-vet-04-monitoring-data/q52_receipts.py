import pathlib,json,subprocess,datetime,zoneinfo
root=pathlib.Path('/home/hoang/nofx-analysis/vet-04-complete-0905');o={'captured_ct':datetime.datetime.now(zoneinfo.ZoneInfo('America/Chicago')).isoformat(),'receipts':[]}
for filename,day in [('log.20260903.00001.txt','2026-09-03'),('log.20260904.00000.txt','2026-09-04')]:
 f=pathlib.Path('/mnt/c/Users/hoang/Documents/NinjaTrader 8/log')/filename
 for n,line in enumerate(f.open(errors='replace'),1):
  if (day=='2026-09-03' and line.startswith(day+' 15:06:') and any(t in line for t in ['Session Break','CONNECTED','Connected'])) or (day=='2026-09-04' and line.startswith(day+' 12:2') and any(t in line for t in ['data feed lost','Connection lost','Disconnected','could not be resolved'])):
   o['receipts'].append({'path':str(f),'line':n,'text':line.rstrip()})
for f in sorted(pathlib.Path('/home/hoang/nofx/data').glob('nofx_*.log')):
 for n,line in enumerate(f.open(errors='replace'),1):
  if ((line.startswith('09-03 11:10:') or line.startswith('09-03 14:18:')) and 'BOOT INTEGRITY' in line) or (line.startswith('09-03 15:06:45') and 'bars: ingest' in line):
   o['receipts'].append({'path':str(f),'line':n,'text':line.rstrip()})
r=subprocess.run(['git','diff','--name-only','2a66d91c','b4376246','--','.',':(exclude)docs'],capture_output=True,text=True);o['non_docs_source_diff']={'command':r.args,'exit_code':r.returncode,'output':r.stdout}
r=subprocess.run(['curl','--silent','--show-error','--max-time','10','--write-out','\nHTTP %{http_code}\n','http://127.0.0.1:8080/api/health'],capture_output=True,text=True);o['public_health']={'command':r.args,'exit_code':r.returncode,'output':r.stdout,'error':r.stderr,'meaning':'Process response only; no readiness or historical availability claim. No JWT used.'}
root.joinpath('q52_receipts.json').write_text(json.dumps(o,indent=2));print(json.dumps(o,indent=2))
