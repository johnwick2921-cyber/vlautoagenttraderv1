import sqlite3, math, datetime
from collections import defaultdict
def wilson(k,n,z=1.96):
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n)); return (round((c-h)/d,3),round((c+h)/d,3))
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True)
rows=con.execute("SELECT open_time_ms,o,c FROM bars WHERE tf='1d' AND symbol='MNQ' ORDER BY open_time_ms").fetchall()
print("1d MNQ bars:",len(rows), "first", datetime.datetime.fromtimestamp(rows[0][0]/1000, datetime.UTC), "last", datetime.datetime.fromtimestamp(rows[-1][0]/1000, datetime.UTC))
# The 1d bar is stamped at 00:00 CT of the SESSION-OPEN date (Sun..Thu); the CME trade date is the NEXT calendar day.
up=defaultdict(int); n=defaultdict(int)
for t,o,c in rows:
    dt=datetime.datetime.fromtimestamp(t/1000, datetime.UTC)-datetime.timedelta(hours=5)+datetime.timedelta(days=1)
    wd=dt.strftime('%a')
    if c==o: continue
    n[wd]+=1; up[wd]+=(c>o)
print("trade weekday | n | up | up_share | wilson95   (full session bar 17:00→16:00 CT, close>open)")
for wd in ['Mon','Tue','Wed','Thu','Fri']:
    print(wd, n[wd], up[wd], round(up[wd]/n[wd],3), wilson(up[wd],n[wd]))
