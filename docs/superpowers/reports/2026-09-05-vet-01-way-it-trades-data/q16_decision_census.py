import sqlite3, json
from collections import Counter, defaultdict
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
act=Counter(); byday=defaultdict(Counter); rce=Counter(); refuse=Counter(); n=0
for ts,dj,rc,el in con.execute("SELECT timestamp, decision_json, risk_check_error, execution_log FROM decision_records WHERE timestamp>='2026-08-19'"):
    n+=1
    try: d=json.loads(dj)
    except Exception: act['<unparsable>']+=1; continue
    if isinstance(d,dict): d=[d]
    acts=[x.get('action','') for x in d if isinstance(x,dict)] or ['<empty>']
    day=ts[:10]
    for a in acts: act[a]+=1; byday[day][a]+=1
    if rc: rce[rc[:70]]+=1
    if el:
        try: L=json.loads(el)
        except Exception: L=[el]
        for line in L:
            if 'refused' in line or 'REFUSED' in line or 'blocked' in line or 'skip' in line.lower():
                # classify
                key=line.split(':')[1].strip()[:60] if ':' in line else line[:60]
                refuse[key]+=1
print('cycles since 08-19:', n); print('actions:', act.most_common())
print('\nopen_* by day:'); 
for d in sorted(byday): 
    c=byday[d]; print(d, 'open_long',c['open_long'],'open_short',c['open_short'],'wait',c['wait'],'hold',c['hold'],'close',c['close_long']+c['close_short'])
print('\nrisk_check_error top:'); [print(' ',v,k) for k,v in rce.most_common(15)]
print('\nexecution_log refusal classes top:'); [print(' ',v,k) for k,v in refuse.most_common(25)]
