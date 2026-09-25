import csv, math, collections
def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (round(cen-half,3), round(cen+half,3))
eps=list(csv.DictReader(open('q11_episodes.csv')))
print("total", len(eps))
# 1. RTH-L below vs PDL below: identical episodes?
def key(e): return (e['day'], e['price'], e['opened'])
rl=[e for e in eps if e['kind']=='RTH-L' and e['entry']=='below']
pl=[e for e in eps if e['kind']=='PDL' and e['entry']=='below']
print("RTH-L below", len(rl), "PDL below", len(pl), "shared (day,price,opened)", len(set(map(key,rl)) & set(map(key,pl))))
print("RTH-L below days", sorted(set(e['day'] for e in rl)), "PDL below days", sorted(set(e['day'] for e in pl)))
# per day breakdown of the from-below cell
c=collections.Counter((e['day'],e['outcome']) for e in rl)
for d in sorted(set(e['day'] for e in rl)):
    print("  RTH-L below", d, "hold",c[(d,'hold')],"brk",c[(d,'break')],"amb",c[(d,'ambiguous_horizon')]+c[(d,'ambiguous_span')], "sessions", collections.Counter(e['session'] for e in rl if e['day']==d))
# RTH-L all days & price
print("RTH-L level-days:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='RTH-L')))
print("PDL level-days:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='PDL')))
print("PDH level-days:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='PDH')))
print("PDH_cal level-days:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='PDH_cal')))
print("RTH-H level-days:", sorted(set((e['day'],e['price']) for e in eps if e['kind']=='RTH-H')))
print("PDL_cal episodes:", sum(1 for e in eps if e['kind']=='PDL_cal'))
# 2. prior-day family dedup on (day,price,opened)
fam=lambda k: ('RN' if k.startswith('RN') else 'VWAPfam' if k.startswith('VWAP') else 'profile' if k in ('POC','VAH','VAL') else 'prior-day' if k in ('PDH','PDL','PDC','RTH-H','RTH-L','SETT','PDH_cal','PDL_cal') else 'overnight' if k in ('ONH','ONL','MID-O') else 'open' if k in ('OR-H','OR-L','IB-H','IB-L') else 'week')
for f in ['prior-day','open','profile','VWAPfam','overnight','RN']:
    fe=[e for e in eps if fam(e['kind'])==f]
    seen={}; 
    for e in fe: seen.setdefault(key(e), e)
    de=list(seen.values())
    h=sum(1 for e in de if e['outcome']=='hold'); b=sum(1 for e in de if e['outcome']=='break')
    hr=sum(1 for e in fe if e['outcome']=='hold'); br=sum(1 for e in fe if e['outcome']=='break')
    print(f"{f:10s} raw n_res={hr+br} p={hr/(hr+br):.3f} {wilson(hr,hr+br)} | dedup(day,price,opened) n_res={h+b} p={h/(h+b) if h+b else float('nan'):.3f} {wilson(h,h+b)} | raw eps {len(fe)} dedup eps {len(de)} | level-days {len(set((e['day'],e['price']) for e in de))}")
# 3. session n + ambiguity
for s in ['NY','ASIA','LONDON']:
    se=[e for e in eps if e['session']==s]; h=sum(1 for e in se if e['outcome']=='hold'); b=sum(1 for e in se if e['outcome']=='break')
    print(s, "n_eps", len(se), "res", h+b, "p", round(h/(h+b),3), "amb%", round(100*(len(se)-h-b)/len(se),1))
# 4. sessions of RTH-L / PDL episodes overall
print("RTH-L sessions", collections.Counter(e['session'] for e in eps if e['kind']=='RTH-L'))
print("PDL sessions", collections.Counter(e['session'] for e in eps if e['kind']=='PDL'))
# 5. flip vs classic pooled across RTH-L/RTH-H/PDL/PDH
flip=[e for e in eps if (e['kind'] in ('RTH-L','PDL') and e['entry']=='below') or (e['kind'] in ('RTH-H','PDH') and e['entry']=='above')]
classic=[e for e in eps if (e['kind'] in ('RTH-L','PDL') and e['entry']=='above') or (e['kind'] in ('RTH-H','PDH') and e['entry']=='below')]
for nm,ss in (('flip',flip),('classic',classic)):
    seen={}
    for e in ss: seen.setdefault(key(e),e)
    de=list(seen.values()); h=sum(1 for e in de if e['outcome']=='hold'); b=sum(1 for e in de if e['outcome']=='break')
    print(nm, "dedup n_res", h+b, "p", round(h/(h+b),3), wilson(h,h+b), "level-days", len(set((e['day'],e['price']) for e in de)))
# 6. per-level-day hold rate for RTH-L below (cluster view)
# 7. A-grade... skip. Count of entry-side cells >= 0.6 among all (kind,entry) cells with n_res>=15
cells=collections.defaultdict(lambda:[0,0])
for e in eps:
    if e['outcome'] in ('hold','break'): cells[(e['kind'],e['entry'])][0 if e['outcome']=='hold' else 1]+=1
big=[(k,v) for k,v in cells.items() if sum(v)>=15]
print("cells n_res>=15:", len(big), "with p>=0.62:", [(k,v,round(v[0]/sum(v),3)) for k,v in big if v[0]/sum(v)>=0.62])
