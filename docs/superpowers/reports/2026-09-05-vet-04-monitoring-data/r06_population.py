import sqlite3, datetime as dt, math
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True); c=con.cursor()
era=int(dt.datetime(2026,8,15,5,0,tzinfo=dt.timezone.utc).timestamp()*1000)
print("era cutoff (2026-08-15 00:00 CT) epoch-ms =",era)
rows=c.execute("SELECT id,source,plan_id,pnl_corrected,entry_time FROM trader_positions").fetchall()
cat={'test_seam':[], 'pre_aug15':[], 'UNRESOLVABLE':[], 'null_pnl':[], 'eligible':[]}
for r in rows:
    i,src,pid,pnl,et=r
    if src=='e7_farside_test': cat['test_seam'].append(r); continue
    if et is None or et<era: cat['pre_aug15'].append(r); continue
    if pid=='UNRESOLVABLE': cat['UNRESOLVABLE'].append(r); continue
    if pnl is None: cat['null_pnl'].append(r); continue
    cat['eligible'].append(r)
for k in ('test_seam','pre_aug15','UNRESOLVABLE','null_pnl','eligible'):
    v=cat[k]; ss=sum(x[3] for x in v if x[3] is not None)
    ids=sorted(x[0] for x in v)
    print("%-13s n=%-4d sum=%9.2f  ids=%s"%(k,len(v),ss,(ids if k!='pre_aug15' else '(516 pre-era ids omitted)')))
el=cat['eligible']; tot=sum(x[3] for x in el)
W=sum(1 for x in el if x[3]>0); L=sum(1 for x in el if x[3]<0); F=sum(1 for x in el if x[3]==0)
print("\nELIGIBLE (compliant primary population): n=%d  sum=%.2f  mean=%.4f  W=%d L=%d F=%d"%(len(el),tot,tot/len(el),W,L,F))
print("UNRESOLVABLE per-row pnl:",[(x[0],x[3]) for x in cat['UNRESOLVABLE']],"sum=%.2f"%sum(x[3] for x in cat['UNRESOLVABLE']))
b=el+cat['UNRESOLVABLE']; print("SENSITIVITY broader 65-row cut: n=%d sum=%.2f"%(len(b),sum(x[3] for x in b)))
def wilson(k,n,z=1.96):
    if n==0: return (0.0,1.0)
    p=k/n; d=1+z*z/n; ctr=(p+z*z/(2*n))/d
    h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (max(0.0,ctr-h),min(1.0,ctr+h))
print("\nwin rate 18/58 Wilson95 = [%.4f, %.4f]"%wilson(18,58))
print("09-03 eligible closes 0/1 Wilson95 = [%.4f, %.4f]"%wilson(0,1))
