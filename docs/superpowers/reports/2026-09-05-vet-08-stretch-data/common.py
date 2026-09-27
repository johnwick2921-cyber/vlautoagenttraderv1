import csv, datetime, bisect
from zoneinfo import ZoneInfo
CT=ZoneInfo('America/Chicago')
def ct_ms(s):
    """'YYYY-MM-DD HH:MM[:SS]' CT -> epoch ms"""
    if len(s)==16: s+=':00'
    return int(datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').replace(tzinfo=CT).timestamp()*1000)
def ms_ct(ms):
    return datetime.datetime.fromtimestamp(ms/1000, CT).strftime('%Y-%m-%d %H:%M:%S')
def load_1m(path='/home/hoang/nofx-analysis/vet-08-0905/bars_1m_0831_0904.csv'):
    out=[]
    for b in csv.DictReader(open(path)):
        out.append((int(b['open_time_ms']), float(b['o']), float(b['h']), float(b['l']), float(b['c']), float(b['v'])))
    out.sort(); return out
def agg5(b1):
    agg={}
    for ms,o,h,l,c,v in b1:
        k=ms-(ms%300000)
        if k not in agg: agg[k]=[o,h,l,c]
        else:
            a=agg[k]; a[1]=max(a[1],h); a[2]=min(a[2],l); a[3]=c
    return [(k,*agg[k]) for k in sorted(agg)]
def atr_series(b5, n=14):
    """Wilder ATR(n) per 5m bar; dict close_ms -> atr (value available once that bar has closed); closed-bar only"""
    out={}; keys=[]; atr=None; prev=None; trs=[]
    for k,o,h,l,c in b5:
        tr=h-l if prev is None else max(h-l,abs(h-prev),abs(l-prev)); prev=c
        if atr is None:
            trs.append(tr)
            if len(trs)==n: atr=sum(trs)/n
        else: atr=(atr*(n-1)+tr)/n
        if atr is not None: out[k+300000]=atr; keys.append(k+300000)
    return out, keys
class ATR:
    def __init__(self):
        self.b1=load_1m(); self.b5=agg5(self.b1); self.d,self.keys=atr_series(self.b5)
    def at(self, ms):
        i=bisect.bisect_right(self.keys, ms)-1
        return self.d[self.keys[i]] if i>=0 else None
