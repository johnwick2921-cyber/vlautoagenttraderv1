# q12: per-leg refusal counts + counterfactual dollars (session-flat and CME-day horizons) from the two-day audit's refusals.csv; the single best saved-loss refusal per leg
import csv, collections
p='/home/hoang/nofx-vet-03/docs/superpowers/reports/2026-09-04-two-day-audit-data/refusals.csv'
rows=list(csv.DictReader(open(p)))
def f(x):
    try: return float(x)
    except: return None
byleg=collections.defaultdict(lambda: {'n':0,'cf_sf':0.0,'cf_sf_n':0,'cf_cme':0.0,'cf_cme_n':0,'outcomes':collections.Counter(),'best':None,'worst':None})
for r in rows:
    L=byleg[r['leg']]; L['n']+=1
    u=f(r['cf_usd']); uc=f(r['cf_usd_cme'])
    if u is not None: L['cf_sf']+=u; L['cf_sf_n']+=1
    if uc is not None: L['cf_cme']+=uc; L['cf_cme_n']+=1
    L['outcomes'][r['cf_outcome']]+=1
    if u is not None:
        if L['best'] is None or u<L['best'][0]: L['best']=(u,r['ts_ct'],r['session'],r['scenario'],r['entry_px'],r['stop_px'],r['target_px'],r['cf_outcome'],r['cf_time_ct'],r['decision_cycle'])
        if L['worst'] is None or u>L['worst'][0]: L['worst']=(u,r['ts_ct'],r['session'],r['scenario'],r['cf_outcome'])
print('## refusals.csv rows', len(rows), '(two-day audit window 09-02 00:00 → 09-03 23:34 CT)')
tot_sf=tot_cme=0
for leg,L in sorted(byleg.items(), key=lambda x:-x[1]['n']):
    tot_sf+=L['cf_sf']; tot_cme+=L['cf_cme']
    print(f"{leg}: n={L['n']} cf_session_flat=${L['cf_sf']:.2f} (n={L['cf_sf_n']}) cf_cme_day=${L['cf_cme']:.2f} outcomes={dict(L['outcomes'])}")
    print(f"    biggest saved loss: {L['best']}")
    print(f"    biggest missed win: {L['worst']}")
print(f'TOTAL cf session-flat ${tot_sf:.2f} · CME-day ${tot_cme:.2f}  (audit states −$860.64 / −$1,036.52 over 44 DISTINCT opportunities; this sum is over 61 EVENTS incl. re-proposals)')
