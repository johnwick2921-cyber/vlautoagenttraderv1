import sqlite3, datetime
con=sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro",uri=True)
c=con.cursor()
# RTH 08:30-14:59 CT per day from 1m bars
rows=c.execute("""
SELECT date(datetime(open_time_ms/1000,'unixepoch','-5 hours')) d,
       COUNT(*) n, MAX(h) hi, MIN(l) lo, ROUND(MAX(h)-MIN(l),2) rng
FROM bars WHERE symbol='MNQ' AND tf='1m'
  AND time(datetime(open_time_ms/1000,'unixepoch','-5 hours')) BETWEEN '08:30' AND '14:59'
GROUP BY 1 ORDER BY 1""").fetchall()
comp=[r for r in rows if r[1]>=389]
print("all RTH days:", len(rows))
for r in rows: print(r)
print("complete (n=390):", len(comp))
vals=[r[4] for r in comp]
print("mean %.2f min %.2f max %.2f"%(sum(vals)/len(vals),min(vals),max(vals)))
print("max day:", max(comp,key=lambda r:r[4]))
# daily bars
d=c.execute("SELECT date(datetime(open_time_ms/1000,'unixepoch','-5 hours')) d, o,h,l,c,ROUND(h-l,2) FROM bars WHERE symbol='MNQ' AND tf='1d' ORDER BY open_time_ms DESC LIMIT 12").fetchall()
print("\n1d bars (latest 12):")
for r in d: print(r)
# what window does the 1d bar dated 2026-09-02 cover?
r=c.execute("SELECT open_time_ms, datetime(open_time_ms/1000,'unixepoch','-5 hours'), h,l FROM bars WHERE symbol='MNQ' AND tf='1d' AND date(datetime(open_time_ms/1000,'unixepoch','-5 hours'))='2026-09-02'").fetchone()
print("\n1d 09-02:", r)
agg=c.execute("""SELECT MAX(h),MIN(l),COUNT(*) FROM bars WHERE symbol='MNQ' AND tf='1m'
 AND datetime(open_time_ms/1000,'unixepoch','-5 hours') >= '2026-09-02 17:00'
 AND datetime(open_time_ms/1000,'unixepoch','-5 hours') <  '2026-09-03 16:00'""").fetchone()
print("1m agg 09-02 17:00 -> 09-03 16:00:", agg)
agg2=c.execute("""SELECT MAX(h),MIN(l),COUNT(*) FROM bars WHERE symbol='MNQ' AND tf='1m'
 AND date(datetime(open_time_ms/1000,'unixepoch','-5 hours'))='2026-09-02'""").fetchone()
print("1m agg calendar 09-02 CT:", agg2)
# ATR14 on 1d
allb=c.execute("SELECT date(datetime(open_time_ms/1000,'unixepoch','-5 hours')),o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1d' ORDER BY open_time_ms").fetchall()
trs=[]
for i in range(1,len(allb)):
    _,o,h,l,cl=allb[i]; pc=allb[i-1][4]
    trs.append((allb[i][0],max(h-l,abs(h-pc),abs(l-pc))))
def atr(idx,n=14):
    seg=[t[1] for t in trs[max(0,idx-n+1):idx+1]]
    return sum(seg)/len(seg)
for i,(d_,tr) in enumerate(trs[-6:], start=len(trs)-6):
    print("ATR14 at %s = %.2f (TR %.2f)"%(trs[i][0],atr(i),tr))
