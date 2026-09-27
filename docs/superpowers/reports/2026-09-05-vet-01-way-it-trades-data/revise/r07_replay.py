import csv,math,sqlite3
from lib import *
UNRES={530,539,545,546,566,571,580}
PV=2.0
con_=con(); cur=con_.cursor()
cur.execute("""SELECT id,side,entry_price,entry_time,exit_price,exit_time,pnl_corrected
 FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL ORDER BY id""")
pos={r[0]:dict(id=r[0],side=r[1].lower(),ep=r[2],et=r[3],xp=r[4],xt=r[5],pnl=r[6]) for r in cur.fetchall()}
csvr={int(r['id']):r for r in csv.DictReader(open('/home/hoang/nofx-analysis/vet-01-0905/q21_trades_final.csv'))}
for pid,p in pos.items():
    r=csvr[pid]
    p['stop']=float(r['stop_pts']) if r['stop_pts'] else None
    p['atr']=float(r['atr5m']) if r['atr5m'] else None
    p['mfe']=float(r['mfe']) if r['mfe']!='' else None
    p['mae']=float(r['mae']) if r['mae']!='' else None
    p['cond']=r['cond']; p['sess']=r['session']
# bars
cur.execute("SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms")
bars=cur.fetchall()
import bisect
bt=[b[0] for b in bars]
def path(p):
    i=bisect.bisect_left(bt,p['et']); j=bisect.bisect_right(bt,p['xt'])
    return bars[i:j]
print("== sanity: point value check")
for pid in (521,528,532):
    p=pos[pid]; print("  ",pid,p['side'],"pnl",p['pnl'],"pts",(p['xp']-p['ep']) if p['side']=='long' else (p['ep']-p['xp']),"=> PV",p['pnl']/((p['xp']-p['ep']) if p['side']=='long' else (p['ep']-p['xp'])))

def realized(pop): return sum(pos[i]['pnl'] for i in pop)

def replayA(pop,minutes,thresh):
    """no-progress scratch: at `minutes` after entry, if MFE-so-far < thresh*stop -> exit at that bar close"""
    tot=0.0; scr=[]; killed_w=[]; helped=[]
    for i in pop:
        p=pos[i]
        if p['stop'] is None: tot+=p['pnl']; continue
        pb=path(p)
        if not pb: tot+=p['pnl']; continue
        cut=p['et']+minutes*60000
        if p['xt']<=cut: tot+=p['pnl']; continue     # already closed
        best=-1e9; close_at=None
        for (t,o,h,l,c) in pb:
            if t>cut: break
            fav=(h-p['ep']) if p['side']=='long' else (p['ep']-l)
            best=max(best,fav)
            close_at=c
        if close_at is None: tot+=p['pnl']; continue
        if best < thresh*p['stop']:
            newp=((close_at-p['ep']) if p['side']=='long' else (p['ep']-close_at))*PV
            tot+=newp; scr.append(i)
            if p['pnl']>0 and newp<p['pnl']: killed_w.append((i,p['pnl'],round(newp,2)))
            if p['pnl']<0 and newp>p['pnl']: helped.append((i,p['pnl'],round(newp,2)))
        else: tot+=p['pnl']
    return tot,scr,killed_w,helped

def replayB(pop,k):
    """target = k*ATR5m; trade wins the target iff MFE>=k*ATR, else unchanged"""
    tot=0.0; changed=0
    for i in pop:
        p=pos[i]
        if p['atr'] is None or p['mfe'] is None: tot+=p['pnl']; continue
        tgt=k*p['atr']
        if p['mfe']>=tgt: tot+=tgt*PV; changed+=1
        else: tot+=p['pnl']
    return tot,changed

def replayC(pop,k):
    """stop tightened to k * current stop: MAE>=k*stop -> loss of k*stop"""
    tot=0.0; killed=[]
    for i in pop:
        p=pos[i]
        if p['stop'] is None or p['mae'] is None: tot+=p['pnl']; continue
        if p['mae']>=k*p['stop']:
            newp=-k*p['stop']*PV; tot+=newp
            if p['pnl']>0: killed.append((i,p['pnl'],round(newp,2)))
        else: tot+=p['pnl']
    return tot,killed

for label,pop in (("n65",sorted(pos)),("n58",[i for i in sorted(pos) if i not in UNRES])):
    print(f"\n######## {label}: realized {realized(pop):.2f} (n={len(pop)})")
    print("-- A: no-progress scratch grid (minutes x thresh) : book / #scratched / winners killed")
    for m in (10,15,20,30,45):
        for th in (0.2,0.3,0.5):
            tot,scr,kw,hp=replayA(pop,m,th)
            print(f"   {m:2d}min x {th:.1f}: book {tot:9.2f}  delta {tot-realized(pop):+9.2f}  scratched {len(scr):2d}  winners killed {len(kw)}")
    tot,scr,kw,hp=replayA(pop,15,0.3)
    print(f"   >> 15/0.3 detail: scratched {scr}")
    print(f"      winners killed {kw}")
    print(f"      losers helped {hp}")
    # avg loss after
    print("-- B: target = k x ATR5m")
    for k in (1.0,1.2,1.4,1.5,1.835,2.0,2.5,3.0):
        tot,ch=replayB(pop,k)
        print(f"   k={k:5.3f}: book {tot:9.2f}  delta {tot-realized(pop):+9.2f}  trades re-priced {ch}")
    print("-- B on the reject cell only")
    rej=[i for i in pop if pos[i]['cond']=='reject']
    print(f"   reject realized {realized(rej):.2f} n={len(rej)}")
    for k in (1.0,1.2,1.4,1.5,1.835,2.0,2.5):
        tot,ch=replayB(rej,k)
        print(f"   k={k:5.3f}: reject book {tot:9.2f}  delta {tot-realized(rej):+9.2f}")
    print("-- C: stop tightened to k x current")
    for k in (0.6,0.7,0.75,0.8,0.9,1.0):
        tot,kill=replayC(pop,k)
        print(f"   k={k:.2f}: book {tot:9.2f}  delta {tot-realized(pop):+9.2f}  winners killed {len(kill)} {kill}")
