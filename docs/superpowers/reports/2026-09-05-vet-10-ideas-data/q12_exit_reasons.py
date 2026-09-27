# q12: exit reason per era trade, joined from the close-sync log line "📕 NT position closed: ... exit=X reason=Y" by exit price + time (±3 min)
import re, glob, sqlite3, datetime, csv, sys
sys.path.insert(0,'/home/hoang/nofx-analysis/vet-10-0905'); from wilson import wilson
lines=[]
for f in sorted(glob.glob('/home/hoang/nofx/data/nofx_*.log')):
    for ln in open(f, errors='ignore'):
        if 'NT position closed' in ln:
            m=re.match(r'^(\d\d)-(\d\d) (\d\d):(\d\d):(\d\d)',ln)
            if not m: continue
            ex=re.search(r'exit=([0-9.]+)',ln); rs=re.search(r'reason=([a-z_]+)',ln); pn=re.search(r'pnl=(-?[0-9.]+)',ln)
            if not ex or not rs: continue
            t=datetime.datetime(2026,int(m.group(1)),int(m.group(2)),int(m.group(3)),int(m.group(4)),int(m.group(5)))
            lines.append((t,float(ex.group(1)),rs.group(1),float(pn.group(1)) if pn else None))
print("close-sync lines parsed:",len(lines))
db=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); db.row_factory=sqlite3.Row
E=int(datetime.datetime(2026,8,15,5,0).timestamp()*1000)
rows=db.execute("select id, side, entry_price, exit_price, exit_time, pnl_corrected, mae, mfe, source from trader_positions where entry_time>=? and source<>'e7_farside_test' and pnl_corrected is not null order by id",(E,)).fetchall()
out=[]; unmatched=[]
for r in rows:
    xt=datetime.datetime.utcfromtimestamp(r['exit_time']/1000)-datetime.timedelta(hours=5)
    cands=[l for l in lines if abs((l[0]-xt).total_seconds())<=180 and abs(l[1]-r['exit_price'])<=0.5]
    if cands:
        l=min(cands,key=lambda l:abs((l[0]-xt).total_seconds())); out.append((r['id'],r['side'],r['pnl_corrected'],l[2],r['mae'],r['mfe']))
    else: unmatched.append(r['id'])
print("matched",len(out),"unmatched",unmatched)
from collections import Counter
c=Counter(o[3] for o in out); print("exit reasons (matched era trades):",dict(c))
for reason in c:
    T=[o for o in out if o[3]==reason]; w=sum(1 for o in T if o[2]>0); print(f"  {reason}: n={len(T)} wins={w} sum$={sum(o[2] for o in T):.1f} ids={[o[0] for o in T]}")
L=[o for o in out if o[2]<0]; sl=[o for o in L if o[3]=='sl']; lo,hi=wilson(len(sl),len(L)); print(f"losers exited by sl: {len(sl)}/{len(L)}={len(sl)/len(L):.3f} wilson[{lo:.3f},{hi:.3f}]")
W=[o for o in out if o[2]>0]; tp=[o for o in W if o[3]=='tp']; lo,hi=wilson(len(tp),len(W)); print(f"winners exited by tp: {len(tp)}/{len(W)}={len(tp)/len(W):.3f} wilson[{lo:.3f},{hi:.3f}]; other winner exits: {Counter(o[3] for o in W if o[3]!='tp')}")
w=csv.writer(open('/home/hoang/nofx-analysis/vet-10-0905/q12_exit_reasons.csv','w')); w.writerow(['id','side','pnl_c','exit_reason_log','mae','mfe']); [w.writerow(o) for o in out]
