import sqlite3, math, statistics as st
from collections import Counter, defaultdict
db = sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro', uri=True); c=db.cursor()
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; cen=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (cen-h, cen+h)
def W(k,n): lo,hi=wilson(k,n); return f"{k}/{n} = {100*k/n:.1f}% [{100*lo:.1f}%, {100*hi:.1f}%]"
print("=== N11: entry_time cuts ===")
for cut in ['2026-08-09 05:00:00','2026-08-09 00:00:00','2026-08-10 05:00:00','2026-08-01 05:00:00','2026-08-15 05:00:00']:
    print(cut, c.execute("SELECT COUNT(*) FROM trader_positions WHERE entry_time >= strftime('%s',?)*1000",(cut,)).fetchone())
print("=== era rows: plan_id not null, pnl_corrected not null, source != e7 ===")
rows = c.execute("""SELECT id, source, plan_session, cited_scenario_id, plan_band, side, pnl_corrected, mae, mfe, entry_price, exit_price,
  datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE plan_id IS NOT NULL AND source != 'e7_farside_test' ORDER BY id""").fetchall()
print("plan-linked non-seam rows:", len(rows), "with pnl:", sum(1 for r in rows if r[6] is not None), "NULL pnl ids:", [r[0] for r in rows if r[6] is None])
res=[r for r in rows if r[6] is not None]
tot=sum(r[6] for r in res); print("era total", round(tot,2), "n", len(res), "mean", round(tot/len(res),2))
sys_=[r for r in res if r[1]=='system']; non=[r for r in res if r[1]!='system']
def summ(name, rs):
    if not rs: print(name, "n=0"); return
    p=[r[6] for r in rs]; w=sum(1 for x in p if x>0); l=sum(1 for x in p if x<0); s=sum(1 for x in p if x==0)
    m=st.mean(p); sd=st.stdev(p) if len(p)>1 else float('nan'); se=sd/math.sqrt(len(p)) if len(p)>1 else float('nan')
    t=m/se if se and se==se else float('nan')
    print(f"{name}: n={len(p)} sum={sum(p):.2f} mean={m:.2f} sd={sd:.2f} se={se:.2f} t={t:.2f} W/L/S={w}/{l}/{s} ids={[r[0] for r in rs]}")
    return (len(p), m, sd)
summ("system", sys_); summ("non-system", non)
print("non-system by band:", Counter((r[1], r[4]) for r in non))
for r in non: print("   ", r[0], r[1], r[4], r[2], r[6])
# armed-fill lineage per report: reconcile + plan_band=armed_fill
af=[r for r in non if r[1]=='reconcile' or r[4]=='armed_fill']
summ("armed-fill lineage (reconcile or band=armed_fill)", af)
rest=[r for r in non if r not in af]; summ("remainder non-system", rest)
print("\n=== N14/R1: slot by source ===")
def slot(r):
    s=r[3]; return s if s else 'uncited'
bys=defaultdict(list)
for r in res: bys[slot(r)].append(r)
for k in sorted(bys): summ("slot "+k, bys[k])
s1=[r for r in res if r[3]=='S1']; s24=[r for r in res if r[3] in ('S2','S3','S4')]
a=summ("S1", s1); b=summ("S2-S4", s24)
n1,m1,sd1=a; n2,m2,sd2=b
se=math.sqrt(sd1*sd1/n1+sd2*sd2/n2); print(f"S1 vs S2-4 diff={m1-m2:.2f} se={se:.2f} t={(m1-m2)/se:.2f}")
summ("S1 system", [r for r in s1 if r[1]=='system']); summ("S1 armed", [r for r in s1 if r[1]!='system'])
summ("S2 system", [r for r in res if r[3]=='S2' and r[1]=='system']); summ("S2 armed", [r for r in res if r[3]=='S2' and r[1]!='system'])
summ("S2-4 armed", [r for r in s24 if r[1]!='system'])
summ("S1 system 08-19/20", [r for r in s1 if r[1]=='system' and r[11][:10] in ('2026-08-19','2026-08-20')])
summ("S1 system after 08-20", [r for r in s1 if r[1]=='system' and r[11][:10] > '2026-08-20'])
summ("S1 any after 08-20", [r for r in s1 if r[11][:10] > '2026-08-20'])
print("\n=== N16/R4/R5: session ===")
byses=defaultdict(list)
for r in res: byses[r[2] or 'none'].append(r)
for k in sorted(byses): summ("session "+k, byses[k])
asia=byses['ASIA']; summ("ASIA system", [r for r in asia if r[1]=='system']); summ("ASIA armed", [r for r in asia if r[1]!='system'])
print("ASIA entry range:", min(r[11] for r in asia), max(r[11] for r in asia))
ln=byses['LONDON']+byses['NY']; x=summ("LONDON+NY", ln)
n,m,sd=x; print(f"LONDON+NY 95% CI mean [{m-1.96*sd/math.sqrt(n):.2f}, {m+1.96*sd/math.sqrt(n):.2f}]")
n,m,sd=summ("ASIA", asia); print(f"ASIA 95% CI mean [{m-1.96*sd/math.sqrt(n):.2f}, {m+1.96*sd/math.sqrt(n):.2f}]")
ma,sa,na=m,sd,n; ml,sl,nl=st.mean([r[6] for r in ln]), st.stdev([r[6] for r in ln]), len(ln)
se=math.sqrt(sa*sa/na+sl*sl/nl); print(f"ASIA vs L+NY diff={ma-ml:.2f} se={se:.2f} t={(ma-ml)/se:.2f}")
print("ASIA win rate:", W(sum(1 for r in asia if r[6]>0), sum(1 for r in asia if r[6]!=0)))
print("\n=== R12: by side ===")
for sd_ in ('long','short','buy','sell'):
    rs=[r for r in res if (r[5] or '').lower()==sd_]
    if rs: summ("side "+sd_, rs); summ("  system", [r for r in rs if r[1]=='system']); summ("  armed", [r for r in rs if r[1]!='system'])
print("sides present:", Counter(r[5] for r in res))
print("\n=== N13: Wilson ===")
for k,n in [(96,395),(36,343),(47,132),(19,31),(54,105),(4,56),(14,72),(13,13),(13,58),(22,58),(32,64),(38,64),(21,27),(2,15),(263,592),(269,603),(62,82),(13,49),(10,27),(8,29)]:
    print("  ", W(k,n))
print("\n=== N15: system t, non-system foot ===")
p=[r[6] for r in sys_]; print("system: n", len(p), "sum", round(sum(p),2), "W", sum(1 for x in p if x>0), "L", sum(1 for x in p if x<0))
print("era total foot:", round(sum(r[6] for r in sys_),2), "+", round(sum(r[6] for r in af),2), "+", round(sum(r[6] for r in rest),2), "=", round(tot,2))
