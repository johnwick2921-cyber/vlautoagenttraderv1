import csv, math, collections
def wilson(h,n,z=1.96):
    if n==0: return (float('nan'),float('nan'))
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (round(cen-half,3), round(cen+half,3))
eps=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-02-0905/q11_episodes.csv')))
print("total episodes", len(eps))
key=lambda e:(e['day'],e['price'],e['opened'])
def fam(k):
    if k.startswith('RN'): return 'RN'
    if k.startswith('VWAP'): return 'VWAPfam'
    if k in ('POC','VAH','VAL'): return 'profile'
    if k in ('PDH','PDL','PDC','RTH-H','RTH-L','SETT','PDH_cal','PDL_cal'): return 'prior-day'
    if k in ('ONH','ONL','MID-O'): return 'overnight'
    if k in ('OR-H','OR-L','IB-H','IB-L'): return 'open'
    return 'week'
def rate(es):
    h=sum(1 for e in es if e['outcome']=='hold'); b=sum(1 for e in es if e['outcome']=='break'); return h,b,(h/(h+b) if h+b else float('nan')),wilson(h,h+b)
print("\n== family raw vs dedup (day,price,opened) ==")
for f in ['prior-day','open','profile','VWAPfam','overnight','RN']:
    fe=[e for e in eps if fam(e['kind'])==f]
    seen={}
    for e in fe: seen.setdefault(key(e),e)
    de=list(seen.values())
    print(f, "raw", len(fe), rate(fe), "| dedup", len(de), rate(de), "| level-days", len(set((e['day'],e['price']) for e in de)))
# which kinds duplicate inside prior-day
pd=[e for e in eps if fam(e['kind'])=='prior-day']
byk=collections.defaultdict(set)
for e in pd: byk[key(e)].add(e['kind'])
dupkinds=collections.Counter(tuple(sorted(v)) for v in byk.values() if len(v)>1)
print("prior-day duplicate kind-sets:", dupkinds)
print("\n== RTH-L below vs PDL below overlap ==")
rl=[e for e in eps if e['kind']=='RTH-L' and e['entry']=='below']
pl=[e for e in eps if e['kind']=='PDL' and e['entry']=='below']
print("RTH-L below", len(rl), rate(rl), "PDL below", len(pl), rate(pl), "shared keys", len(set(map(key,rl))&set(map(key,pl))))
print("RTH-L below level-days", sorted(set((e['day'],e['price']) for e in rl)))
print("PDL below level-days", sorted(set((e['day'],e['price']) for e in pl)))
c=collections.Counter((e['day'],e['outcome']) for e in rl)
for d in sorted(set(e['day'] for e in rl)): print("  RTH-L below",d,"hold",c[(d,'hold')],"brk",c[(d,'break')])
print("RTH-L level-days all:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='RTH-L')))
print("PDL level-days all:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='PDL')))
print("\n== flip vs classic pooled (RTH-L/PDL/RTH-H/PDH), dedup ==")
flip=[e for e in eps if (e['kind'] in ('RTH-L','PDL') and e['entry']=='below') or (e['kind'] in ('RTH-H','PDH') and e['entry']=='above')]
classic=[e for e in eps if (e['kind'] in ('RTH-L','PDL') and e['entry']=='above') or (e['kind'] in ('RTH-H','PDH') and e['entry']=='below')]
for nm,ss in (('flip',flip),('classic',classic)):
    seen={}
    for e in ss: seen.setdefault(key(e),e)
    de=list(seen.values()); print(nm, len(de), rate(de), "level-days", len(set((e['day'],e['price']) for e in de)))
print("RTH-H below", rate([e for e in eps if e['kind']=='RTH-H' and e['entry']=='below']), "RTH-H above", rate([e for e in eps if e['kind']=='RTH-H' and e['entry']=='above']))
print("\n== (kind,entry) cells n_res>=15 ==")
cells=collections.defaultdict(lambda:[0,0])
for e in eps:
    if e['outcome'] in ('hold','break'): cells[(e['kind'],e['entry'])][0 if e['outcome']=='hold' else 1]+=1
big=[(k,v) for k,v in cells.items() if sum(v)>=15]
print("count", len(big)); print("p>=0.62:", sorted([(k,v,round(v[0]/sum(v),3)) for k,v in big if v[0]/sum(v)>=0.62], key=lambda x:-x[2]))
print("p<=0.38:", sorted([(k,v,round(v[0]/sum(v),3)) for k,v in big if v[0]/sum(v)<=0.38], key=lambda x:x[2]))
print("\n== session ==")
for s in ['NY','ASIA','LONDON']:
    se=[e for e in eps if e['session']==s]; h,b,p,w=rate(se); print(s,"eps",len(se),"res",h+b,"p",round(p,3),w,"amb%",round(100*(len(se)-h-b)/len(se),1), "amb", len(se)-h-b)
print("\n== per-kind rows vs resolved (n convention) ==")
for k in ['VWAP+2σ@0830','VWAP-2σ@0830','ONH','ONL','PDL','PDH','PDH_cal','EQL','RTH-L','IB-H','IB-L']:
    ke=[e for e in eps if e['kind']==k]
    if ke: h,b,p,w=rate(ke); print(k,"rows",len(ke),"res",h+b,"amb",len(ke)-h-b,"p",round(p,3),w)
v2=[e for e in eps if e['kind'] in ('VWAP+2σ@0830','VWAP-2σ@0830')]; h,b,p,w=rate(v2); print("VWAP±2σ pooled rows",len(v2),"res",h+b,"p",round(p,3),w)
