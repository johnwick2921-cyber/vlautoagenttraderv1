import math,csv,sqlite3
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
def con():
    return sqlite3.connect(DB,uri=True)
def wilson(k,n,z=1.96):
    if n==0: return (0.0,0.0)
    p=k/n; d=1+z*z/n; c=(p+z*z/(2*n))/d
    h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (c-h,c+h)
def pct(vals,q):
    """linear-interpolation percentile, numpy-style"""
    v=sorted(vals)
    if not v: return None
    if len(v)==1: return v[0]
    idx=(len(v)-1)*q
    lo=int(math.floor(idx)); hi=int(math.ceil(idx))
    if lo==hi: return v[lo]
    return v[lo]+(v[hi]-v[lo])*(idx-lo)
def mean(v): return sum(v)/len(v) if v else None
def sd_pop(v):
    m=mean(v); return math.sqrt(sum((x-m)**2 for x in v)/len(v)) if v else None
def sd_samp(v):
    m=mean(v); return math.sqrt(sum((x-m)**2 for x in v)/(len(v)-1)) if len(v)>1 else None
