#!/usr/bin/env python3
"""q11: D1' replay over the FULL persisted 1m MNQ tape (read-only), mechanical level kinds.
Detector = line-for-line port of kernel/detector_d1prime.go DetectTouchOutcomes (k=3, H=12, exit_on=close).
Delta = mean |close increment| over the trailing 5 completed session days (DetectorDeltaDays=5).
Levels are scanned only from their formation time (lookahead-free). CT = UTC-5.
"""
import sqlite3, math, json, csv, random, sys, collections
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
OUT="/home/hoang/nofx-analysis/vet-02-0905/"
CT=-5*3600
con=sqlite3.connect(DB, uri=True)
rows=con.execute("SELECT open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
bars=[dict(t=r[0],o=r[1],h=r[2],l=r[3],c=r[4],v=r[5]) for r in rows]
def ct(ms): return (ms//1000)+CT
def sess_day(ms):  # CME session day key (17:00 CT roll): date of (ct + 7h)
    s=ct(ms)+7*3600; return s//86400
def cal_day(ms): return ct(ms)//86400
def tod(ms):  # seconds since CT midnight
    return ct(ms)%86400
def session_of(ms):
    t=tod(ms)
    if t>=17*3600 or t<2*3600: return 'ASIA'
    if t<8*3600+30*60: return 'LONDON'
    if t<16*3600: return 'NY'
    return 'CLOSE'
def ymd(day): 
    import datetime; return (datetime.date(1970,1,1)+datetime.timedelta(days=day)).isoformat()

def detect(bs, level, k, delta, horizon, exit_on='close'):
    """port of DetectTouchOutcomes"""
    if level<=0 or k<=0 or delta<=0 or horizon<=0 or len(bs)<2: return []
    up, lo = level+k*delta, level-k*delta
    eps=[]; ordinal=0; n=len(bs); i=1
    while i<n:
        b,pb=bs[i],bs[i-1]
        touch = pb['l']<=level<=pb['h']; cur = b['l']<=level<=b['h']
        if not (cur and not touch): i+=1; continue
        entry='above' if pb['c']>=level else 'below'
        ordinal+=1; outcome=''; exit_side=''; mfe=0.0; mae=0.0; j=i+1
        while j<n and (j-i)<=horizon:
            c=bs[j]
            if entry=='below': mfe=max(mfe,c['h']-level); mae=min(mae,c['l']-level)
            else: mfe=max(mfe,level-c['l']); mae=min(mae,level-c['h'])
            cu,cd=(c['c']>up, c['c']<lo) if exit_on=='close' else (c['h']>up, c['l']<lo)
            if cu and cd: outcome='ambiguous_span'; break
            if cu: outcome=('hold' if entry=='above' else 'break'); exit_side='above'; break
            if cd: outcome=('hold' if entry=='below' else 'break'); exit_side='below'; break
            j+=1
        if outcome=='': outcome='ambiguous_horizon'
        eps.append(dict(ordinal=ordinal,opened=b['t'],closed=(bs[j]['t'] if j<n else 0),entry=entry,exit=exit_side,outcome=outcome,bars=j-i,mfe=mfe,mae=mae))
        i=j+1
    return eps

# group bars by session day and calendar day
by_sd=collections.OrderedDict(); by_cd=collections.OrderedDict()
for b in bars:
    by_sd.setdefault(sess_day(b['t']),[]).append(b); by_cd.setdefault(cal_day(b['t']),[]).append(b)
sdays=[d for d,bs in by_sd.items() if len(bs)>=1300]  # completed days only (1380 bars); partial 09-04 excluded from level formation but used as scan tape? exclude fully
print("completed session days:", [ymd(d) for d in sdays])
def rng(bs): return max(x['h'] for x in bs), min(x['l'] for x in bs)
def vwap_sd(bs):
    tv=0.0; tpv=0.0
    for b in bs:
        tp=(b['h']+b['l']+b['c'])/3; tv+=b['v']; tpv+=tp*b['v']
    if tv<=0: return 0,0
    vw=tpv/tv; var=0.0
    for b in bs:
        tp=(b['h']+b['l']+b['c'])/3; var+=b['v']*(tp-vw)**2
    return vw, math.sqrt(var/tv)
def profile(bs, nbins=120):
    hi,lo=rng(bs)
    if hi<=lo: return None
    w=(hi-lo)/nbins; vol=[0.0]*nbins
    for b in bs:
        # spread bar volume evenly across the bins its range covers (engine: levels_volume.go profileLevels) — approximation
        b0=int(min(nbins-1,max(0,(b['l']-lo)//w))); b1=int(min(nbins-1,max(0,(b['h']-lo)//w)))
        nb=b1-b0+1
        for k in range(b0,b1+1): vol[k]+=b['v']/nb
    poc=max(range(nbins), key=lambda k: vol[k]); total=sum(vol); need=0.70*total
    lo_i=hi_i=poc; acc=vol[poc]
    while acc<need and (lo_i>0 or hi_i<nbins-1):
        up=vol[hi_i+1] if hi_i<nbins-1 else -1; dn=vol[lo_i-1] if lo_i>0 else -1
        if up>=dn: hi_i+=1; acc+=up
        else: lo_i-=1; acc+=dn
    px=lambda k: lo+(k+0.5)*w
    return dict(POC=px(poc), VAH=lo+(hi_i+1)*w, VAL=lo+lo_i*w)

K=3.0; H=12
episodes=[]  # dicts with kind, day, price, ...
cal_vs_sess=[]
for idx,d in enumerate(sdays):
    if idx==0: continue
    prev=sdays[idx-1]; pbs=by_sd[prev]; cbs=by_sd[d]
    # delta: trailing 5 completed session days before d
    trail=[b for dd in sdays[max(0,idx-5):idx] for b in by_sd[dd]]
    delta=sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/max(1,len(trail)-1)
    day_start=cbs[0]['t']
    rth_open=[b for b in cbs if tod(b['t'])>=8*3600+30*60 and tod(b['t'])<16*3600]
    levels=[]  # (kind, price, scan_from_ms, scan_to_ms)
    # prior SESSION day extremes (trader's PDH/PDL/PDC)
    ph,pl=rng(pbs); pc=pbs[-1]['c']
    levels += [('PDH',ph,day_start,None),('PDL',pl,day_start,None),('PDC',pc,day_start,None)]
    # engine's CALENDAR-day PDH/PDL as the NY 08:00 read sees them: prior calendar day = calendar date of d-1 (CT)
    cd_prev = cal_day(rth_open[0]['t'])-1 if rth_open else None
    if cd_prev in by_cd and len(by_cd[cd_prev])>=900:
        ch,cl=rng(by_cd[cd_prev]); cal_vs_sess.append(dict(day=ymd(d),sess_PDH=ph,cal_PDH=ch,sess_PDL=pl,cal_PDL=cl,dH=round(ch-ph,2),dL=round(cl-pl,2)))
        levels += [('PDH_cal',ch,rth_open[0]['t'],None),('PDL_cal',cl,rth_open[0]['t'],None)]
    # prior RTH high/low (08:30-15:00 CT of prev session day)
    prth=[b for b in pbs if 8*3600+30*60<=tod(b['t'])<15*3600]
    if prth:
        rh,rl=rng(prth); levels += [('RTH-H',rh,day_start,None),('RTH-L',rl,day_start,None)]
        # settlement proxy: last 1m close before 15:00 CT
        levels.append(('SETT',prth[-1]['c'],day_start,None))
    # ONH/ONL frozen at 08:30, scanned during RTH only
    on=[b for b in cbs if tod(b['t'])>=17*3600 or tod(b['t'])<8*3600+30*60]
    if on and rth_open:
        oh,ol=rng(on); t0=rth_open[0]['t']
        levels += [('ONH',oh,t0,None),('ONL',ol,t0,None),('MID-O',(oh+ol)/2,t0,None)]
    # OR5 / IB
    if rth_open:
        t0=rth_open[0]['t']
        or5=[b for b in rth_open if tod(b['t'])<8*3600+35*60]; ib=[b for b in rth_open if tod(b['t'])<9*3600+30*60]
        if or5:
            h5,l5=rng(or5); levels += [('OR-H',h5,t0+5*60000,None),('OR-L',l5,t0+5*60000,None)]
        if ib:
            hi_,lo_=rng(ib); levels += [('IB-H',hi_,t0+60*60000,None),('IB-L',lo_,t0+60*60000,None)]
        # VWAP frozen at RTH open (session bars 17:00->08:29), scanned in RTH
        pre=[b for b in cbs if b['t']<t0]
        if pre:
            vw,sd=vwap_sd(pre)
            if vw>0: levels += [('VWAP@0830',vw,t0,None),('VWAP+1σ@0830',vw+sd,t0,None),('VWAP-1σ@0830',vw-sd,t0,None),('VWAP+2σ@0830',vw+2*sd,t0,None),('VWAP-2σ@0830',vw-2*sd,t0,None)]
    # prior-day profile
    pr=profile(pbs)
    if pr: levels += [('POC',pr['POC'],day_start,None),('VAH',pr['VAH'],day_start,None),('VAL',pr['VAL'],day_start,None)]
    # round numbers within the day's range +-100
    dh,dl=rng(cbs)
    for step,tag in ((100,'RN100'),(50,'RN50'),(25,'RN25')):
        m=math.ceil((dl-100)/step)*step
        while m<=dh+100:
            if not((tag=='RN50' and m%100==0) or (tag=='RN25' and m%50==0)):
                levels.append((tag,float(m),day_start,None))
            m+=step
    # prior week (Mon..Fri session days) high/low: needs >=4 completed days in prior ISO week
    import datetime
    dd=datetime.date(1970,1,1)+datetime.timedelta(days=d)
    wk_start=dd-datetime.timedelta(days=dd.weekday())  # Monday
    pw=[x for x in sdays if wk_start-datetime.timedelta(days=7) <= (datetime.date(1970,1,1)+datetime.timedelta(days=x)) < wk_start]
    if len(pw)>=4:
        pwb=[b for x in pw for b in by_sd[x]]; wh,wl=rng(pwb); levels += [('PWH',wh,day_start,None),('PWL',wl,day_start,None)]
    for kind,price,t_from,t_to in levels:
        scan=[b for b in cbs if b['t']>=t_from]
        for e in detect(scan,price,K,delta,H,'close'):
            e.update(kind=kind,day=ymd(d),price=round(price,2),delta=round(delta,3),session=session_of(e['opened']))
            episodes.append(e)

def wilson(h,n,z=1.96):
    if n==0: return (0,0)
    p=h/n; den=1+z*z/n; cen=(p+z*z/(2*n))/den; half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return (cen-half, cen+half)
def agg(eps, keyf):
    out={}
    for e in eps:
        k=keyf(e); a=out.setdefault(k,dict(n=0,hold=0,brk=0,amb=0,mfe=0.0,mae=0.0,days=set(),prices=set()))
        a['n']+=1; a['days'].add(e['day']); a['prices'].add((e['day'],e['price']))
        if e['outcome']=='hold': a['hold']+=1
        elif e['outcome']=='break': a['brk']+=1
        else: a['amb']+=1
        a['mfe']+=e['mfe']; a['mae']+=e['mae']
    return out
def table(name, out, order=None):
    lines=[f"\n== {name}", f"{'key':14s} {'n':>4s} {'hold':>4s} {'brk':>4s} {'amb':>4s} {'p_hold':>6s} {'wilson95':>16s} {'amb%':>5s} {'days':>4s} {'lvls':>4s} {'mfe':>6s} {'mae':>7s}"]
    keys=order or sorted(out, key=lambda k:-out[k]['n'])
    for k in keys:
        a=out[k]; den=a['hold']+a['brk']; p=a['hold']/den if den else float('nan'); lo,hi=wilson(a['hold'],den)
        lines.append(f"{str(k):14s} {a['n']:4d} {a['hold']:4d} {a['brk']:4d} {a['amb']:4d} {p:6.3f} [{lo:.3f},{hi:.3f}] {100*a['amb']/a['n']:5.1f} {len(a['days']):4d} {len(a['prices']):4d} {a['mfe']/a['n']:6.2f} {a['mae']/a['n']:7.2f}")
    return "\n".join(lines)
rep=[]
rep.append(f"episodes total={len(episodes)} over {len(sdays)-1} scanned session days ({ymd(sdays[1])}..{ymd(sdays[-1])}); k={K} H={H} exit=close; delta per-day trailing-5d")
rep.append(table("per kind (all sessions, all ordinals)", agg(episodes, lambda e:e['kind'])))
rep.append(table("per kind — FIRST touch of the day only (ordinal 1)", agg([e for e in episodes if e['ordinal']==1], lambda e:e['kind'])))
rep.append(table("pooled by ordinal", agg(episodes, lambda e: '1' if e['ordinal']==1 else ('2' if e['ordinal']==2 else '3+')), ['1','2','3+']))
rep.append(table("pooled by session", agg(episodes, lambda e:e['session'])))
rep.append(table("pooled by kind family", agg(episodes, lambda e: ('RN' if e['kind'].startswith('RN') else 'VWAPfam' if e['kind'].startswith('VWAP') else 'profile' if e['kind'] in ('POC','VAH','VAL') else 'prior-day' if e['kind'] in ('PDH','PDL','PDC','RTH-H','RTH-L','SETT','PDH_cal','PDL_cal') else 'overnight' if e['kind'] in ('ONH','ONL','MID-O') else 'open' if e['kind'] in ('OR-H','OR-L','IB-H','IB-L') else 'week'))))
# fade-vs-break framing per kind: entry side matters. hold = price returned to the side it came from.
rep.append(table("RTH-L / PDL / ONL / VAL / IB-L by entry side (from above = a support test)", agg([e for e in episodes if e['kind'] in ('RTH-L','PDL','ONL','VAL','IB-L','OR-L','PDL_cal')], lambda e:(e['kind'],e['entry']))))
rep.append(table("RTH-H / PDH / ONH / VAH / IB-H by entry side (from below = a resistance test)", agg([e for e in episodes if e['kind'] in ('RTH-H','PDH','ONH','VAH','IB-H','OR-H','PDH_cal')], lambda e:(e['kind'],e['entry']))))
# random-level (Osler) null: B=100 random prices per day uniformly inside the day's range, scanned from day start
random.seed(7); null=[]
for idx,d in enumerate(sdays):
    if idx==0: continue
    cbs=by_sd[d]; trail=[b for dd in sdays[max(0,idx-5):idx] for b in by_sd[dd]]
    delta=sum(abs(trail[i]['c']-trail[i-1]['c']) for i in range(1,len(trail)))/max(1,len(trail)-1)
    dh,dl=rng(cbs)
    for _ in range(100):
        px=random.uniform(dl,dh)
        for e in detect(cbs,px,K,delta,H,'close'): e.update(kind='NULL',day=ymd(d),price=px,session=session_of(e['opened'])); null.append(e)
rep.append(table("RANDOM-LEVEL NULL (100 uniform prices/day inside the day range)", agg(null, lambda e:'NULL')))
rep.append(table("NULL by ordinal", agg(null, lambda e: '1' if e['ordinal']==1 else ('2' if e['ordinal']==2 else '3+')), ['1','2','3+']))
rep.append("\n== calendar-day PDH/PDL (engine, NY read) vs session-day PDH/PDL (trader): per day")
for r in cal_vs_sess: rep.append(json.dumps(r))
txt="\n".join(rep); print(txt)
open(OUT+"q11_replay.txt","w").write(txt)
with open(OUT+"q11_episodes.csv","w",newline="") as f:
    w=csv.DictWriter(f, fieldnames=['kind','day','price','session','ordinal','opened','closed','entry','exit','outcome','bars','mfe','mae','delta']); w.writeheader()
    for e in episodes: w.writerow({k:e.get(k) for k in w.fieldnames})
