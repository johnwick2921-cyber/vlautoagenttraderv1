import pathlib,re,csv,json,datetime
from zoneinfo import ZoneInfo
R=pathlib.Path(__file__).resolve().parent;ct=ZoneInfo('America/Chicago');rows=[];events=[]
pat=re.compile(r'REFUSED|refused|sl_too_tight|last_entry_cutoff|cancelled —|NOT authored|gate changed|stop-entry|armed .*WORKING|plan .* (DORMANT|REARMED)|FEED DOWN|NT8 TCP link DOWN|dead-man|BOOT INTEGRITY|wake would_skip|wake skipped|cooldown|arm stop NY S1|no balance frame|daily.*OFF')
for p in sorted(pathlib.Path('/home/hoang/nofx/data').glob('nofx_2026-09-0[234].log')):
 for n,line in enumerate(p.open(errors='replace'),1):
  if not re.match(r'09-0[234] ',line):continue
  if pat.search(line):rows.append({'source':str(p),'line':n,'text':line.strip()})
  m=re.search(r'^(09-0[234] \d\d:\d\d:\d\d).*plan (2026-\d\d-\d\d) (ASIA|LONDON|NY) v(\d+) (DORMANT|REARMED)',line)
  if m:events.append(dict(time_ms=int(datetime.datetime.fromisoformat('2026-'+m[1]).replace(tzinfo=ct).timestamp()*1000),date=m[2],session=m[3],version=int(m[4]),event=m[5],source=str(p),line=n))
with (R/'log_evidence.csv').open('w') as f:
 w=csv.DictWriter(f,lineterminator="\n",fieldnames=['source','line','text']);w.writeheader();w.writerows(rows)
(R/'life_events.json').write_text(json.dumps(events,indent=2))
# Broker records only order shape and timestamps; remove account names.
out=[]
for date in ['20260903','20260904']:
 for p in pathlib.Path('/mnt/c/Users/hoang/Documents/NinjaTrader 8/log').glob('log.'+date+'*.txt'):
  if '.en.' in p.name:continue
  for n,line in enumerate(p.open(errors='replace'),1):
   if ("Type='Stop Market'" in line and ('Stop price=0' in line or 'Stop price=29355' in line)) or ('29355' in line and "Order state='Filled'" in line):
    line=re.sub(r"Account='[^']*'","Account='[redacted]'",line)
    out.append(dict(source=str(p),line=n,text=line.strip()))
with (R/'broker_evidence.csv').open('w') as f:
 w=csv.DictWriter(f,lineterminator="\n",fieldnames=['source','line','text']);w.writeheader();w.writerows(out)
print('log evidence',len(rows),'life events',len(events),'broker lines',len(out))
