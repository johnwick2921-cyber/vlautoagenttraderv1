"""Compliant fill-quality, exit-reason, floor-proxy re-derivation. READ-ONLY."""
import sqlite3,math,json,datetime,csv,os
os.chdir('/home/hoang/nofx-analysis/vet-05-0905')
c=sqlite3.connect('file:/home/hoang/nofx/data/data.db?mode=ro',uri=True); c.row_factory=sqlite3.Row
c.execute('PRAGMA query_only=ON')
ERA=1786770000000; END=1788584400000
def rows(sql,a=()): return [dict(r) for r in c.execute(sql,a)]
def rate(k,n):
    if not n: return (k,n,None,None)
    p=k/n; z=1.96; d=1+z*z/n; m=(p+z*z/(2*n))/d; h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/d
    return (k,n,round(100*p,2),[round(100*max(0,m-h),2),round(100*min(1,m+h),2)])
def pct(a):
    a=sorted(a); return {p:round(a[int((len(a)-1)*p)]+(a[min(int((len(a)-1)*p)+1,len(a)-1)]-a[int((len(a)-1)*p)])*(((len(a)-1)*p)-int((len(a)-1)*p)),3) for p in (.5,.8,.95)}
comp=("FROM trader_positions WHERE entry_time>=? AND entry_time<? AND pnl_corrected IS NOT NULL "
 "AND coalesce(source,'')<>'e7_farside_test' AND lower(coalesce(close_reason,'')) NOT IN ('unresolved','unresolvable','reconcile_flat','e7_farside_test') "
 "AND upper(coalesce(pnl_correction_note,'')) NOT LIKE '%UNRESOLVABLE%' AND coalesce(plan_id,'')<>'UNRESOLVABLE'")
ps=rows("SELECT * "+comp+" ORDER BY id",(ERA,END))
old={int(r['pos_id']):r for r in csv.DictReader(open('q11_fill_vs_bar.csv'))}
legacy={int(r['id']):r for r in csv.DictReader(open('q14_mae_mfe.csv'))}
# ---- fill quality (same method as q31) ----
res={'entry':{'inside':[],'at_low':[],'at_high':[],'outside':[],'nobar':[]},'exit':{'inside':[],'at_low':[],'at_high':[],'outside':[],'nobar':[]}}
for r in ps:
    for leg in ('entry','exit'):
        refms=r[leg+'_time']; price=r[leg+'_price']
        action=('BUY' if r['side'].upper()=='LONG' else 'SELL') if leg=='entry' else ('SELL' if r['side'].upper()=='LONG' else 'BUY')
        fr=rows('select id,created_at from trader_fills where side=? and abs(price-?)<0.01 and abs(created_at-?)<600000 order by abs(created_at-?) limit 1',(action,price,refms,refms))
        ms=fr[0]['created_at'] if fr else refms
        if leg=='entry':
            oo=old[r['id']]
            if oo['bar_open_ct']:
                ms=int(datetime.datetime.strptime('2026-'+oo['bar_open_ct'],'%Y-%m-%d %H:%M').replace(tzinfo=datetime.timezone(datetime.timedelta(hours=-5))).timestamp()*1000)
        b=rows("select rowid as bar_id,h,l from bars where symbol='MNQ' and tf='1m' and open_time_ms<=? and open_time_ms+60000>? order by open_time_ms desc limit 1",(ms,ms))
        if not b: res[leg]['nobar'].append(r['id']); continue
        b=b[0]
        if b['l']<=price<=b['h']:
            res[leg]['inside'].append(r['id'])
            if abs(price-b['l'])<.126: res[leg]['at_low'].append(r['id'])
            if abs(price-b['h'])<.126: res[leg]['at_high'].append(r['id'])
        else: res[leg]['outside'].append((r['id'],price,b['l'],b['h']))
for leg in ('entry','exit'):
    d=res[leg]; n=len(d['inside'])+len(d['outside'])
    print("%s: n_with_bar=%d inside=%s outside=%s at_low=%s at_high=%s nobar=%s"%(leg,n,rate(len(d['inside']),n),d['outside'],d['at_low'],d['at_high'],d['nobar']))
    # adverse/favorable per report convention: entry adverse = at_low for short? report used position 537 at low / 585 at high
print()
# ---- exit reasons ----
byr={}
for r in ps: byr.setdefault(legacy[r['id']]['reason'],[]).append(r['id'])
for k in sorted(byr): print("exit reason %-10s %s ids=%s"%(k,rate(len(byr[k]),len(ps)),byr[k]))
print()
# ---- floor proxy (15 closed 5m bars, simple TR14) ----
floors=[]
for r in ps:
    b=rows("select rowid as bar_id,open_time_ms,h,l,c from bars where symbol='MNQ' and tf='5m' and open_time_ms+300000<=? order by open_time_ms desc limit 15",(r['entry_time'],))[::-1]
    if len(b)<15: continue
    atr=sum(max(b[i]['h']-b[i]['l'],abs(b[i]['h']-b[i-1]['c']),abs(b[i]['l']-b[i-1]['c'])) for i in range(1,15))/14
    age=r['entry_time']-(b[-1]['open_time_ms']+300000)
    floors.append({'id':r['id'],'pnl':r['pnl_corrected'],'mae':r['mae'],'src':r['source'],'atr':atr,'floor':1.5*atr,'ratio':(r['mae']/(1.5*atr)) if r['mae'] is not None else None,'age_ms':age})
fw=[r for r in floors if r['pnl']>0 and r['mae'] is not None and r['age_ms']<=300000]
fl=[r for r in floors if r['pnl']<0 and r['mae'] is not None and r['age_ms']<=300000]
print("FLOOR winners n=%d ids=%s"%(len(fw),[r['id'] for r in fw]))
print("  ratios:",[round(r['ratio'],4) for r in fw]); print("  pct:",pct([r['ratio'] for r in fw]),"exceeded>1:",rate(sum(r['ratio']>1 for r in fw),len(fw)))
fwm=[r for r in fw if r['mae']!=0]
print("  ZERO-mae sentinels among winners:",[(r['id'],r['src']) for r in fw if r['mae']==0])
print("  measured-only n=%d pct=%s exceeded=%s"%(len(fwm),pct([r['ratio'] for r in fwm]),rate(sum(r['ratio']>1 for r in fwm),len(fwm))))
print("FLOOR losers n=%d pct=%s exceeded>1: %s"%(len(fl),pct([r['ratio'] for r in fl]),rate(sum(r['ratio']>1 for r in fl),len(fl))))
# ---- invalidation cost side: winners with MAE beyond 16.5 pts ----
w=[r for r in ps if r['pnl_corrected']>0]
beyond=[r for r in w if r['mae'] is not None and r['mae']>16.5]
print("\nWinners n=%d ; MAE>16.5 pts: %s ids=%s  dollars=%.2f of %.2f = %.1f%%"%(
 len(w),rate(len(beyond),len(w)),[r['id'] for r in beyond],sum(r['pnl_corrected'] for r in beyond),sum(r['pnl_corrected'] for r in w),
 100*sum(r['pnl_corrected'] for r in beyond)/sum(r['pnl_corrected'] for r in w)))
# ---- 14:45 flat: exits between 14:40 and 15:10 CT ----
ex=[(r['id'],datetime.datetime.fromtimestamp(r['exit_time']/1000,datetime.timezone(datetime.timedelta(hours=-5))).strftime('%H:%M:%S'),r['pnl_corrected']) for r in ps if r['exit_time']]
late=[e for e in ex if '14:40'<=e[1][:5]<='15:10']
print("\nExits 14:40-15:10 CT:",late)
print("All 14:xx exits:",[e for e in ex if e[1][:2]=='14'])
c.close()
