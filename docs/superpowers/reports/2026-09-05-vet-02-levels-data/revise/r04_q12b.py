import csv, math, collections
def wilson(h,n,z=1.96):
    if n==0: return (float('nan'),float('nan'))
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (round(cen-half,4), round(cen+half,4))
rows=list(csv.DictReader(open('/home/hoang/nofx-vet-02/docs/superpowers/reports/2026-09-05-vet-02-levels-data/q12b_plan_levels_forward_dedup.csv')))
print("rows", len(rows))
def touched(r): return int(r['n_eps'])>0
def bucket(rs, name):
    n=len(rs); t=sum(1 for r in rs if touched(r)); print(f"{name}: seats {n} ({100*n/len(rows):.1f}%) touched {t} = {100*t/n if n else 0:.1f}% = 1 in {n/t if t else float('inf'):.1f}")
d=lambda r: abs(float(r['dist']))
bucket([r for r in rows if d(r)>100], "dist>100")
bucket([r for r in rows if 100<d(r)<200], "100<dist<200")
bucket([r for r in rows if 100<=d(r)<200], "100<=dist<200")
bucket([r for r in rows if d(r)>=200], "dist>=200")
bucket([r for r in rows if d(r)>200], "dist>200")
bucket([r for r in rows if d(r)<175], "dist<175")
bucket([r for r in rows if d(r)<100], "dist<100")
bucket([r for r in rows if d(r)<50], "dist<50")
bucket([r for r in rows if d(r)<25], "dist<25")
bucket(rows, "all")
# EQH first touch
eqh=[r for r in rows if r['fam']=='EQH']; h=sum(1 for r in eqh if r['first']=='hold'); b=sum(1 for r in eqh if r['first']=='break'); print("EQH seats",len(eqh),"first h/b",h,b, round(h/(h+b),3), wilson(h,h+b))
eql=[r for r in rows if r['fam']=='EQL']; h=sum(1 for r in eql if r['first']=='hold'); b=sum(1 for r in eql if r['first']=='break'); print("EQL seats",len(eql),"first h/b",h,b, round(h/(h+b),3), wilson(h,h+b))
ib=[r for r in rows if r['fam']=='IB*']; print("IB* seats",len(ib), collections.Counter(r['first'] for r in ib), "touched", sum(1 for r in ib if touched(r)))
poc=[r for r in rows if r['fam']=='POC']; print("POC seats",len(poc), collections.Counter(r['first'] for r in poc))
v2=[r for r in rows if '2σ' in r['label']]; print("VWAP±2σ plan-forward rows",len(v2), collections.Counter(r['first'] for r in v2), "sessions", len(set((r['day'],r['session']) for r in v2)))
ev=[r for r in rows if r['fam']=='eVWAP']; h=sum(1 for r in ev if r['first']=='hold'); b=sum(1 for r in ev if r['first']=='break'); print("eVWAP seats",len(ev),"h/b",h,b, wilson(h,h+b))
print("hand Wilson 32/45:", wilson(32,45), " 4/12:", wilson(4,12), " 4/13:", wilson(4,13), " 34/60:", wilson(34,60), " 3977/7857:", wilson(3977,7857))
