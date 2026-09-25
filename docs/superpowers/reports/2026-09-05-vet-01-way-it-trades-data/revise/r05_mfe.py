import csv
from lib import *
UNRES={530,539,545,546,566,571,580}
rows=list(csv.DictReader(open('/home/hoang/nofx-analysis/vet-01-0905/q21_trades_final.csv')))
def f(r,k):
    v=r.get(k,'')
    return float(v) if v not in ('','None',None) else None
for r in rows:
    r['id']=int(r['id'])
    for k in ('stop_pts','atr5m','stop_atr','mfe','mae','pnl_usd','mfe_over_stop','mae_over_stop'):
        r[k+'_v']=f(r,k)
print("== rows with atr5m:",len([r for r in rows if r['atr5m_v'] is not None]))
print("== rows with atr and mfe:",len([r for r in rows if r['atr5m_v'] is not None and r['mfe_v'] is not None]))
print("== rows with atr and stop:",len([r for r in rows if r['atr5m_v'] is not None and r['stop_pts_v'] is not None]))
print("== ids missing atr:",[r['id'] for r in rows if r['atr5m_v'] is None])
print("== ids with atr but no mfe:",[r['id'] for r in rows if r['atr5m_v'] is not None and r['mfe_v'] is None])
# reject MFE/ATR percentiles
rej=[r['mfe_v']/r['atr5m_v'] for r in rows if r['cond']=='reject' and r['atr5m_v'] and r['mfe_v'] is not None]
print("\n== reject MFE/ATR n=%d"%len(rej))
for q in (0.4,0.5,0.55,0.6,0.7,0.8,0.95):
    print(f"   p{int(q*100)} = {pct(rej,q):.3f}")
for k in (1.4,1.5,1.835,1.88,2.0):
    hit=len([x for x in rej if x>=k]); lo,hi=wilson(hit,len(rej))
    print(f"   >= {k}: {hit}/{len(rej)} = {hit/len(rej):.3f} [{lo:.3f},{hi:.3f}]")
# all trades MFE/ATR
alla=[r['mfe_v']/r['atr5m_v'] for r in rows if r['atr5m_v'] and r['mfe_v'] is not None]
print("\n== all-trades MFE/ATR n=%d p50 %.3f p80 %.3f p95 %.3f"%(len(alla),pct(alla,.5),pct(alla,.8),pct(alla,.95)))
print("   >=3.0x:",len([x for x in alla if x>=3.0]))
# P8: MFE >= 2*1.5*ATR (min stop) and >= 1*1.5*ATR ; and >= 2*actual stop
p8=[r for r in rows if r['atr5m_v'] and r['mfe_v'] is not None]
n2=[r['id'] for r in p8 if r['mfe_v']>=2*1.5*r['atr5m_v']]
n1=[r['id'] for r in p8 if r['mfe_v']>=1.5*r['atr5m_v']]
print("\n== P8 base (atr+mfe) n=%d ; >=2x minstop %d %s ; >=1x minstop %d"%(len(p8),len(n2),n2,len(n1)))
lo,hi=wilson(len(n2),len(p8)); print("   2x: [%.3f,%.3f]"%(lo,hi))
lo,hi=wilson(len(n1),len(p8)); print("   1x: [%.3f,%.3f]"%(lo,hi))
p8b=[r for r in rows if r['stop_pts_v'] and r['mfe_v'] is not None]
n2b=[r['id'] for r in p8b if r['mfe_v']>=2*r['stop_pts_v']]
lo,hi=wilson(len(n2b),len(p8b))
print("   2x ACTUAL stop: %d/%d [%.3f,%.3f] %s"%(len(n2b),len(p8b),lo,hi,n2b))
# P9 winners MAE/ATR
win=[r for r in rows if r['pnl_usd_v']>0]
winA=[r for r in win if r['atr5m_v'] is not None and r['mae_v'] is not None]
print("\n== winners n=%d ; with atr+mae n=%d"%(len(win),len(winA)))
for r in sorted(winA,key=lambda r:-(r['mae_v']/(1.5*r['atr5m_v']))):
    print("   id %d mae %.2f atr %.3f floor %.2f ratio %.4f pnl %.1f"%(r['id'],r['mae_v'],r['atr5m_v'],1.5*r['atr5m_v'],r['mae_v']/(1.5*r['atr5m_v']),r['pnl_usd_v']))
# losers MAE/stop
los=[r for r in rows if r['pnl_usd_v']<0]
losS=[r for r in los if r['stop_pts_v'] and r['mae_v'] is not None]
print("\n== losers n=%d with stop+mae n=%d"%(len(los),len(losS)))
atstop=[r['id'] for r in losS if r['mae_v']>=0.95*r['stop_pts_v']]
print("   MAE>=0.95xstop:",len(atstop),"/",len(losS))
never05=[r['id'] for r in losS if r['mfe_v'] is not None and r['mfe_v']<0.5*r['stop_pts_v']]
print("   MFE<0.5xstop:",len(never05))
full1=[r['id'] for r in losS if r['mfe_v'] is not None and r['mfe_v']>=r['stop_pts_v']]
print("   MFE>=1xstop:",len(full1),full1)
