import csv, math
def wilson(h,n,z=1.96):
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (round(cen-half,3), round(cen+half,3))
rows=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-02-0905/q12b_plan_levels_forward_dedup.csv')))
n=len(rows); touched=sum(1 for r in rows if r['first']!='untouched')
print(f"all seats n={n} touched {touched} = {100*touched/n:.1f}% {wilson(touched,n)}")
for thr in (50,100,175,200):
    inb=[r for r in rows if float(r['dist'])<thr]; t=sum(1 for r in inb if r['first']!='untouched')
    print(f"dist<{thr}: seats {len(inb)} ({100*len(inb)/n:.1f}% of map) touched {t} = {100*t/len(inb):.1f}% {wilson(t,len(inb))}")
gt=[r for r in rows if float(r['dist'])>100]; t=sum(1 for r in gt if r['first']!='untouched')
print(f"dist>100: seats {len(gt)} touched {t} = {100*t/len(gt):.1f}% (one in {len(gt)/t:.1f})")
gt=[r for r in rows if 100<=float(r['dist'])<200]; t=sum(1 for r in gt if r['first']!='untouched'); print(f"100-200: {len(gt)} touched {t} = {100*t/len(gt):.1f}%")
gt=[r for r in rows if float(r['dist'])>=200]; t=sum(1 for r in gt if r['first']!='untouched'); print(f"200+: {len(gt)} touched {t} = {100*t/len(gt):.1f}% (one in {len(gt)/t:.1f})")
# EQH/EQL/eVWAP/IB cells
for fam in ('EQH','EQL','eVWAP','IB*','SWG-L','POC'):
    f=[r for r in rows if r['fam']==fam]; h=sum(1 for r in f if r['first']=='hold'); b=sum(1 for r in f if r['first']=='break')
    print(fam, "seats", len(f), "1st hold/brk", h, b, "p", round(h/(h+b),3) if h+b else None, wilson(h,h+b) if h+b else None)
# grade A/C avg dist
for g in ('A','C'):
    f=[r for r in rows if r['grade']==g]; print(g, len(f), "avg dist", round(sum(float(r['dist']) for r in f)/len(f),1))
print("--- wilson batch ---")
for lab,h,nn in (("EQH 32/45",32,45),("RTH-L dedup 4/12",4,12),("RTH-L dedup 4/13",4,13),("VWAP2s 34/60",34,60),("eVWAP 11/14",11,14),("IB 1/2",1,2),("RTH-H below 17/26",17,26),("RTH-H above 11/19",11,19),("flip 34/56",34,56),("classic 33/58",33,58),("prior-day dedup",164,292),("null 3977/7857",3977,7857),("RTH-L 27/43",27,43),("PDL 26/43",26,43)):
    print(lab, round(h/nn,3), wilson(h,nn))
# session cells from q11 episodes
eps=list(csv.DictReader(open('q11_episodes.csv')))
for s in ('NY','ASIA','LONDON'):
    se=[e for e in eps if e['session']==s]; h=sum(1 for e in se if e['outcome']=='hold'); b=sum(1 for e in se if e['outcome']=='break')
    print(s, "eps", len(se), "hold", h, "brk", b, "p", round(h/(h+b),3), wilson(h,h+b), "amb", len(se)-h-b, f"{100*(len(se)-h-b)/len(se):.1f}%")
# prior-day dedup exact
import collections
fam=lambda k: k in ('PDH','PDL','PDC','RTH-H','RTH-L','SETT','PDH_cal','PDL_cal')
seen={}
for e in eps:
    if fam(e['kind']): seen.setdefault((e['day'],e['price'],e['opened']), e)
h=sum(1 for e in seen.values() if e['outcome']=='hold'); b=sum(1 for e in seen.values() if e['outcome']=='break')
print("prior-day dedup exact", h, b, round(h/(h+b),3), wilson(h,h+b))
# ONH/ONL/PDL resolved
for k in ('ONH','ONL','PDL','PDH','PDH_cal','VWAP+2σ@0830','VWAP-2σ@0830','SETT'):
    f=[e for e in eps if e['kind']==k]; h=sum(1 for e in f if e['outcome']=='hold'); b=sum(1 for e in f if e['outcome']=='break')
    print(k, "eps", len(f), "res", h+b, "p", round(h/(h+b),3), wilson(h,h+b))
