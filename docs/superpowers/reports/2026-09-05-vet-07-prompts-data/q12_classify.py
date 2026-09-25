import csv, re, math
from collections import Counter, defaultdict
def wilson(k,n,z=1.96):
    if n==0: return (0,0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return ((c-h)/d,(c+h)/d)
rules=[
 ('TRANSPORT (503/timeout) — not a validator', r'status 503|stream interrupted|context deadline|Server Overloaded|API error'),
 ('B3 breakdown VOID (close came back across)', r'came back across'),
 ('B2 BD_MIN_CLOSES (no confirming close yet)', r'NO confirming close'),
 ('B1 BD_MIN_DISP_ATR (displacement below floor)', r'displacement.*(below|<).*floor|MEASURED displacement|displacement .* is below'),
 ('B2 BD_MAX_LEVEL_DIST_ATR (level too far)', r'BD_MAX_LEVEL_DIST|too far from|pts from price.*level'),
 ('ArmSpec entry_mode=pullback required', r'arm requires entry_mode=pullback'),
 ('ArmSpec legs/split contract', r'legs|split|EXACTLY 2'),
 ('ArmSpec wait_confirm/confirm required', r'requires wait_confirm|requires a confirm'),
 ('ArmSpec non-armable condition', r'non-armable'),
 ('ArmSpec entry/stop/target ordering or values', r'stop .* < entry|target .* < entry|exact entry/stop/target'),
 ('EntryLaw fade_requires_touch', r'fade_requires_touch'),
 ('EntryLaw rule not allowed for condition', r'not allowed for'),
 ('EntryLaw 2x5m_reserved', r'2x5m_reserved'),
 ('EntryLaw structure stop ≥2 ticks', r'structure stop required'),
 ('Schema: confirm rule enum invalid', r'confirm2?\.rule .* invalid'),
 ('Schema: condition invalid / combined', r'condition .* invalid'),
 ('Schema: ref_price/prose mismatch', r'does not match any number'),
 ('Reachability: target outside proximity band', r'proximity band|unreachable'),
 ('Levels: too many / duplicates / one-sided', r'too many levels|apart —|0 levels (below|above)'),
 ('Gap-direction order', r'gap-(down|up)'),
 ('Phantom relabel', r'phantom'),
]
rows=list(csv.DictReader(open('q12_rejects_all.csv')))
cls=[]
for r in rows:
    reason=r['reason']; hit='OTHER'
    for name,pat in rules:
        if re.search(pat,reason,re.I): hit=name; break
    r['cls']=hit; cls.append(hit)
n=len(rows)
print(f"n={n} rows, trade_date {min(r['trade_date'] for r in rows)}..{max(r['trade_date'] for r in rows)}")
c=Counter(cls)
print("\nclass | n | share | wilson95 | ids")
for k,v in c.most_common():
    lo,hi=wilson(v,n); ids=[r['id'] for r in rows if r['cls']==k]
    print(f"{k} | {v} | {v/n:.1%} | [{lo:.1%},{hi:.1%}] | {','.join(ids)}")
print("\nby trade_date x class")
bd=defaultdict(Counter)
for r in rows: bd[r['trade_date']][r['cls']]+=1
for d in sorted(bd): print(d, dict(bd[d]))
print("\nby session:", Counter(r['session'] for r in rows))
print("by attempt:", Counter(r['attempt'] for r in rows))
print("\nOTHER rows:")
for r in rows:
    if r['cls']=='OTHER': print(r['id'], r['reason'][:300])
# multi-defect rows: count ';' or numbered defects
multi=[r['id'] for r in rows if re.search(r'\b2\. |;\s*S\d', r['reason'])]
print("\nrows carrying >1 defect (heuristic):", len(multi), multi)
