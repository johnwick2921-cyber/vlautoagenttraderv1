import csv, math, sqlite3, statistics as st
D="/home/hoang/nofx-vet-10/docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q07_canonical_trades.csv"
rows=[r for r in csv.DictReader(open(D)) if r['realized_R']]
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
unres={str(r[0]) for r in con.execute("select id from trader_positions where plan_id='UNRESOLVABLE'")}
sess={str(r[0]):(r[1] or '') for r in con.execute("select id,plan_session from trader_positions")}
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); m=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return ((c-m)/d,(c+m)/d)
def meanci(v):
    n=len(v)
    if n<2: return (sum(v)/max(n,1),float('nan'),float('nan'))
    m=sum(v)/n; s=st.stdev(v); h=1.96*s/math.sqrt(n)
    return (m,m-h,m+h)
def rpt(tag,rs):
    R=[float(r['realized_R']) for r in rs]
    P=[float(r['pnl_c']) for r in rs]
    rr=[float(r['planned_rr']) for r in rs]
    risk=[float(r['risk_pts']) for r in rs]
    w=sum(1 for x in P if x>0); l=sum(1 for x in P if x<0); f=sum(1 for x in P if x==0)
    aw=[x for x in P if x>0]; al=[-x for x in P if x<0]
    hit=[r for r in rs if float(r['mfe_pts'])>=abs(float(r['tp'])-float(r['entry']))]
    m,lo,hi=meanci(R); mp,plo,phi=meanci(P)
    print(f"[{tag}] n={len(rs)} W/L/F={w}/{l}/{f} sum$={sum(P):.2f} mean$={mp:.2f} CI[{plo:.2f},{phi:.2f}]")
    print(f"   meanR={m:.3f} CI[{lo:.3f},{hi:.3f}]  payoff={sum(aw)/len(aw)/(sum(al)/len(al)):.2f} avgW={sum(aw)/len(aw):.2f} avgL={sum(al)/len(al):.2f}")
    print(f"   plannedRR mean={sum(rr)/len(rr):.2f} median={st.median(rr):.2f} max={max(rr):.2f}  below2.0={sum(1 for x in rr if x<2.0)}")
    k=len(hit); print(f"   MFE>=planned target {k}/{len(rs)}={k/len(rs):.3f} wilson{tuple(round(x,3) for x in wilson(k,len(rs)))} ids={[h['id'] for h in hit]}")
    for th in (0.5,1.0,2.0):
        kk=sum(1 for r in rs if float(r['mfe_pts'])>=th*float(r['risk_pts']))
        print(f"   MFE>={th}R {kk}/{len(rs)}={kk/len(rs):.3f} wilson{tuple(round(x,3) for x in wilson(kk,len(rs)))}")
    print(f"   risk pts min={min(risk):.2f} p25={st.quantiles(risk,n=4)[0]:.2f} med={st.median(risk):.2f} p75={st.quantiles(risk,n=4)[2]:.2f} max={max(risk):.2f}")
    wr=w/(w+l); print(f"   win rate {w}/{w+l}={wr:.3f} wilson{tuple(round(x,3) for x in wilson(w,w+l))}; breakeven need={1/(1+sum(aw)/len(aw)/(sum(al)/len(al))):.3f}")
print("### R-set as the report built it (n=60)")
rpt("R60",rows)
comp=[r for r in rows if r['id'] not in unres]
print("### compliant R-set (UNRESOLVABLE removed)")
rpt("Rcomp",comp)
comp2=[r for r in comp if r['id'] not in ('568','569')]
print("### compliant R-set excl 568/569 (class-27 artefacts)")
rpt("Rcomp-x",comp2)
print("### compliant R-set excl 568/569 and ASIA")
rpt("Rcomp-x-noASIA",[r for r in comp2 if sess.get(r['id'])!='ASIA'])
print("### ASIA only, compliant")
rpt("ASIA",[r for r in comp2 if sess.get(r['id'])=='ASIA'])
print()
print("### losers by MFE bucket, compliant excl 568/569")
b={'<0.25R':0,'0.25-0.5R':0,'0.5-1R':0,'>=1R':0}
for r in comp2:
    if float(r['pnl_c'])>=0: continue
    x=float(r['mfe_pts'])/float(r['risk_pts'])
    b['<0.25R' if x<0.25 else '0.25-0.5R' if x<0.5 else '0.5-1R' if x<1 else '>=1R']+=1
n=sum(b.values()); ge=b['0.5-1R']+b['>=1R']
print(b,"n=",n,f" MFE>=0.5R {ge}/{n}={ge/n:.3f} wilson{tuple(round(x,3) for x in wilson(ge,n))}",
      f" MFE>=1R {b['>=1R']}/{n} wilson{tuple(round(x,3) for x in wilson(b['>=1R'],n))}")
print()
print("### stop-distance quartiles, compliant excl 568/569 (mean R and $)")
s=sorted(comp2,key=lambda r: float(r['risk_pts']))
q=len(s)//4
for i,(a,bb) in enumerate([(0,q),(q,2*q),(2*q,3*q),(3*q,len(s))]):
    part=s[a:bb]; R=[float(r['realized_R']) for r in part]; P=[float(r['pnl_c']) for r in part]
    print(f"  Q{i+1} {float(part[0]['risk_pts']):.2f}-{float(part[-1]['risk_pts']):.2f} n={len(part)} meanR={sum(R)/len(R):+.3f} sum$={sum(P):+.2f}")
print()
print("### planned RR below 2.0 rows (compliant excl 568/569)")
for r in sorted(comp2,key=lambda r: float(r['planned_rr']))[:14]:
    print("  ",r['id'],r['planned_rr'],r['stop_src'],r['session'],r['scenario'])
