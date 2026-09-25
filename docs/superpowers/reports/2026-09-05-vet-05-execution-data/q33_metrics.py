import json,math,sqlite3
x=json.load(open('q31_verified.json'));out={}
def rate(k,n):
 p=k/n;z=1.96;d=1+z*z/n;m=(p+z*z/(2*n))/d;h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
 return f'{k}/{n} = {100*p:.1f}% (Wilson 95% {max(0,100*(m-h)):.1f}–{min(100,100*(m+h)):.1f}%)'
for k,n in [(0,21),(5,15),(3,15),(3,37),(3,11),(7,22),(57,177),(4,7),(4,10),(24,36),(25,37),(2,24),(3,24),(1,24),(1,2),(0,1),(5,6),(6,6),(21,63),(44,65),(13,65),(58,62),(61,62),(1,62),(0,62),(0,11),(5,11),(1,25)]:out[f'{k}/{n}']=rate(k,n)
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True);c.row_factory=sqlite3.Row
out['guard_prior_ids']=[r[0] for r in c.execute("select id from armed_orders where strftime('%s',created_at)*1000>=1788238800000 and strftime('%s',created_at)*1000<1788498000000 and lower(state_reason) not like '%test%' order by id")]
out['bar_buckets']={}
for name,lo,hi in [('eod','14:40','14:49'),('open','08:30','08:39')]:
 bs=[dict(r) for r in c.execute("select rowid as id,open_time_ms,h-l as range_pts,v from bars where symbol='MNQ' and tf='1m' and date(open_time_ms/1000,'unixepoch','-5 hours')>='2026-08-19' and strftime('%w',open_time_ms/1000,'unixepoch','-5 hours') between '1' and '5' and strftime('%H:%M',open_time_ms/1000,'unixepoch','-5 hours') between ? and ? order by open_time_ms",(lo,hi))]
 out['bar_buckets'][name]={'ids':[r['id'] for r in bs],'n':len(bs),'mean_volume':sum(r['v'] for r in bs)/len(bs),'mean_range':sum(r['range_pts'] for r in bs)/len(bs)}
json.dump(out,open('q33_metrics.json','w'),indent=2)
print(json.dumps({k:v for k,v in out.items() if k!='bar_buckets'},indent=2))
