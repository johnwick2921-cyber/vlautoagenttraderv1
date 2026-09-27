import json,sqlite3
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
cur=con.cursor()
cur.execute("SELECT id,timestamp,decision_json FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-08-18'")
closes=[]
for did,ts,dj in cur.fetchall():
    if not dj: continue
    try: o=json.loads(dj)
    except Exception: continue
    ds=o if isinstance(o,list) else (o.get('decisions') or [o] if isinstance(o,dict) else [])
    if isinstance(ds,dict): ds=[ds]
    for d in ds:
        if isinstance(d,dict):
            a=str(d.get('action','')).lower()
            if a.startswith('close'):
                closes.append((did,ts,a,d.get('symbol')))
print("close_* decisions since 08-18:",len(closes))
for c in closes: print("  ",c)
# now exit times of the 18
ids=[526,530,538,542,551,557,558,560,566,568,569,570,571,575,578,580,581,591]
cur.execute("SELECT id,exit_time FROM trader_positions WHERE id IN (%s)"%",".join(map(str,ids)))
ex=dict(cur.fetchall())
import datetime
def toms(ts):
    # timestamp like '2026-08-28 13:44:43.xxx+00:00'
    s=ts.replace('T',' ')
    if '+' in s: s=s.split('+')[0]
    s=s.split('.')[0]
    dt=datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').replace(tzinfo=datetime.timezone.utc)
    return int(dt.timestamp()*1000)
print("\nmatches within 4 minutes of an exit:")
for did,ts,a,sym in closes:
    ms=toms(ts)
    for pid,et in ex.items():
        if et and abs(ms-et)<=4*60*1000:
            print(f"  pos {pid} exit {datetime.datetime.utcfromtimestamp(et/1000)} UTC | decision {did} {a} at {ts} | delta_s {(et-ms)/1000:.0f}")
