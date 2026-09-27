import sqlite3, math, re
from collections import defaultdict
con=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (0,0,0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return (p,(c-h)/d,(c+h)/d)
rows=con.execute("SELECT label, level_price, touch_number, shape, close_1m, close_5m, penetration_pts, session_day, opened_at_ms FROM touch_episodes").fetchall()
def fam(label):
    l=label.split('·')[0]
    l=re.sub(r'\(.*?\)','',l)
    return l
agg=defaultdict(lambda:[0,0]); ordn=defaultdict(lambda:[0,0]); famord=defaultdict(lambda:[0,0]); c5=defaultdict(lambda:[0,0])
for lab,px,tn,shape,c1,c5m,pen,sd,o in rows:
    f=fam(lab); hold = 1 if shape=='rejection' else 0
    agg[f][0]+=hold; agg[f][1]+=1
    ob = 1 if tn==1 else (2 if tn==2 else 3)
    ordn[ob][0]+=hold; ordn[ob][1]+=1
    famord[(f,ob)][0]+=hold; famord[(f,ob)][1]+=1
    if c5m in ('reject','accept'):
        c5[f][0]+= (c5m=='reject'); c5[f][1]+=1
print('touch_episodes n=',len(rows),'distinct (label,price,opened)=',len(set((r[0],r[1],r[8]) for r in rows)))
print('\n== hold(rejection) rate by level family, n>=30 reported, Wilson 95% ==')
for f,(k,n) in sorted(agg.items(), key=lambda x:-x[1][1]):
    p,lo,hi=wilson(k,n); flag='' if n>=30 else '  (n<30 no verdict)'
    print(f"{f:10s} hold {k:4d}/{n:4d} = {p:.3f} [{lo:.3f},{hi:.3f}]{flag}")
print('\n== by ordinal (1 / 2 / 3+) pooled ==')
for ob in (1,2,3):
    k,n=ordn[ob]; p,lo,hi=wilson(k,n); print(f"ordinal {ob}: hold {k}/{n} = {p:.3f} [{lo:.3f},{hi:.3f}]")
print('\n== family x ordinal (n>=30 only) ==')
for (f,ob),(k,n) in sorted(famord.items(), key=lambda x:(x[0][0],x[0][1])):
    if n>=30:
        p,lo,hi=wilson(k,n); print(f"{f:10s} ord{ob}: hold {k}/{n} = {p:.3f} [{lo:.3f},{hi:.3f}]")
print('\n== close_5m reject rate by family (n>=30) ==')
for f,(k,n) in sorted(c5.items(), key=lambda x:-x[1][1]):
    if n>=30: p,lo,hi=wilson(k,n); print(f"{f:10s} 5m-reject {k}/{n} = {p:.3f} [{lo:.3f},{hi:.3f}]")
# by session bucket via opened_at_ms CT hour
import datetime
sess=defaultdict(lambda:[0,0])
for lab,px,tn,shape,c1,c5m,pen,sd,o in rows:
    h=(datetime.datetime.utcfromtimestamp(o/1000)-datetime.timedelta(hours=5)).hour
    s='NY' if 8<=h<15 else ('LONDON' if 2<=h<8 else 'ASIA')
    sess[s][0]+= (shape=='rejection'); sess[s][1]+=1
print('\n== by session ==')
for s,(k,n) in sess.items(): p,lo,hi=wilson(k,n); print(f"{s:7s} hold {k}/{n} = {p:.3f} [{lo:.3f},{hi:.3f}]")
