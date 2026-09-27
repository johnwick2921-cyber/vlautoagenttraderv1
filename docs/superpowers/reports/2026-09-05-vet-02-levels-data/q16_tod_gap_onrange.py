#!/usr/bin/env python3
"""q16: descriptive tape facts for Q5 — hour-of-day |dClose| (the detector's delta is session-wide), RTH gap fill, ON-range vs RTH-range, opening drive."""
import collections, math
exec(open('/home/hoang/nofx-analysis/vet-02-0905/q11_replay.py').read().split("# group bars by session day")[0])
by_sd=collections.OrderedDict()
for b in bars: by_sd.setdefault(sess_day(b['t']),[]).append(b)
sdays=[d for d,bs in by_sd.items() if len(bs)>=1300]
# hour-of-day mean |dClose| (CT), all completed days
hod=collections.defaultdict(list)
for d in sdays:
    bs=by_sd[d]
    for i in range(1,len(bs)): hod[tod(bs[i]['t'])//3600].append(abs(bs[i]['c']-bs[i-1]['c']))
print("== hour-of-day (CT) mean |1m close increment| over", len(sdays), "completed session days; session-wide Δ used by D1′ ≈", round(sum(sum(v) for v in hod.values())/sum(len(v) for v in hod.values()),3))
for h in sorted(hod): print(f"  {h:02d}:00  n={len(hod[h]):5d}  mean|Δ|={sum(hod[h])/len(hod[h]):6.3f}  band k=3 -> ±{3*sum(hod[h])/len(hod[h]):5.1f} pt")
print("\n== per day: prior RTH close -> RTH open gap, filled within RTH? ; ON range vs RTH range ; opening drive (first 30m dir vs RTH close)")
prev=None; fills=0; n=0; ratios=[]; drives=0; dn=0
for d in sdays:
    bs=by_sd[d]; rth=[b for b in bs if 8*3600+30*60<=tod(b['t'])<15*3600]; on=[b for b in bs if b['t']<rth[0]['t']] if rth else []
    if prev and rth and on:
        prc=prev[-1]['c']; o=rth[0]['o']; gap=o-prc
        hi=max(b['h'] for b in rth); lo=min(b['l'] for b in rth); filled = (lo<=prc<=hi)
        onr=max(b['h'] for b in on)-min(b['l'] for b in on); rthr=hi-lo; ratios.append(onr/rthr if rthr else float('nan'))
        f30=[b for b in rth if tod(b['t'])<9*3600]; drive=(f30[-1]['c']-o); close=rth[-1]['c']-o
        agree = (drive>0)==(close>0) if drive!=0 else None
        if agree is not None: dn+=1; drives+= 1 if agree else 0
        n+=1; fills+= 1 if filled else 0
        print(f"  {ymd(d)}  gap={gap:+7.2f}  filled={'Y' if filled else 'n'}  ON={onr:6.1f} RTH={rthr:6.1f} ratio={onr/rthr:4.2f}  drive30={drive:+6.1f} close={close:+7.1f} agree={agree}")
    prev=[b for b in bs if tod(b['t'])<15*3600 and tod(b['t'])>=8*3600+30*60] or prev
print(f"gap filled within RTH: {fills}/{n} ; ON/RTH range median={sorted(ratios)[len(ratios)//2]:.2f} ; opening-drive direction == RTH close direction: {drives}/{dn}")
