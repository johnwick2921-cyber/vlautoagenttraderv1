import csv, datetime, pickle
b1=list(csv.DictReader(open('bars_1m_0831_0904.csv')))
# aggregate to 5m epoch-floor
agg={}
for b in b1:
    ms=int(b['open_time_ms']); k=ms-(ms%300000)
    o,h,l,c=float(b['o']),float(b['h']),float(b['l']),float(b['c'])
    if k not in agg: agg[k]=[o,h,l,c]
    else:
        a=agg[k]; a[1]=max(a[1],h); a[2]=min(a[2],l); a[3]=c
keys=sorted(agg)
atr=None; prev_c=None; trs=[]; out={}
for k in keys:
    o,h,l,c=agg[k]
    tr = h-l if prev_c is None else max(h-l, abs(h-prev_c), abs(l-prev_c))
    prev_c=c
    if atr is None:
        trs.append(tr)
        if len(trs)==14: atr=sum(trs)/14
    else: atr=(atr*13+tr)/14
    out[k+300000]=atr   # available once the bar closes
pickle.dump(out, open('atr5m_by_ms.pkl','wb'))
def atr_at(ct):
    t=datetime.datetime.strptime(ct,'%Y-%m-%d %H:%M:%S'); ms=int((t+datetime.timedelta(hours=5)).timestamp()*1000)
    ks=[k for k in out if k<=ms and out[k] is not None]; return out[max(ks)] if ks else None
checks=[('2026-09-03 08:00:54',29.0154),('2026-09-03 09:06:54',45.0976),('2026-09-03 10:10:54',48.405),('2026-09-03 11:20:33',32.461),('2026-09-04 08:32:00',32.589),('2026-09-04 10:28:02',36.371),('2026-09-04 12:01:01',23.289),('2026-09-03 16:30:21',12.743),('2026-09-03 01:30:56',22.102),('2026-09-04 01:30:48',13.318)]
print("5m bars aggregated from 1m:", len(keys))
for ct,ref in checks:
    a=atr_at(ct); print(ct, 'closed-bar ATR14 %.2f'%a, 'recorded %.2f'%ref, 'diff %+.2f'%(a-ref))
