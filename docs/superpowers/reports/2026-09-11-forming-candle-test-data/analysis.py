#!/usr/bin/env python3
"""
DISPATCH 02 — TEST 13f: DOES THE FORMING CANDLE PREDICT? (read-only)
Consolidated analysis. Every figure in the report is produced by this script,
run read-only against the live DB:

    python3 analysis.py

Inputs : file:/home/hoang/nofx/data/data.db?mode=ro (SQLite, read-only)
Outputs: stdout (all figures) + pairs_primary.csv (the A21 sample-id artifact:
          every joined pair with its episode id, outcome id, fields, verdict).

Key decisions (see report §B1/B2):
  * The strict join (level identity + session-day + ordinal) is measured but
    reported as UNRELIABLE: median |opened gap| = 30 min.
  * The analysis population is the time-verified join: same (trader, symbol,
    level_price, CME session-day), |opened_at_ms difference| <= 10 min.
    PRIMARY = pairs whose outcome row is claimed by exactly one episode.
  * Verdicts come from the D1' detector (touch_outcomes), k=3, H=12 x 1m,
    exit_on=close. Ambiguous rows are excluded from rates, never dropped.
  * Multiple comparison: Westfall-Young max-T permutation (20k perms) over the
    7 tested fields + Bonferroni reference. Sullivan-Timmermann-White lesson.
"""
import sqlite3, datetime, zoneinfo, json, math
from collections import defaultdict, Counter
import numpy as np
from scipy import stats

CT = zoneinfo.ZoneInfo("America/Chicago")
DB = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
DB.row_factory = sqlite3.Row
Z = 1.959963984540054
RNG = np.random.default_rng(7)
R = 20000

FIELDS = ['wick_pen_pts', 'body_pen_pts', 'penetration_pts', 'vol_ratio',
          'approach_atr', 'bars_in', 'touch_number']


def cmeday(ms):
    t = datetime.datetime.fromtimestamp(ms / 1000, CT)
    b = t.replace(hour=17, minute=0, second=0, microsecond=0)
    if t.hour < 17:
        b -= datetime.timedelta(days=1)
    return b.strftime("%Y-%m-%d")


def sess(ms):
    t = datetime.datetime.fromtimestamp(ms / 1000, CT)
    h = t.hour + t.minute / 60
    if h >= 17 or h < 2:
        return "ASIA"
    if h < 8.5:
        return "LONDON"
    return "NY"


def ct(ms):
    return datetime.datetime.fromtimestamp(ms / 1000, CT).strftime('%m-%d %H:%M')


def wilson(k, n):
    if n <= 0:
        return (0.0, 0.0)
    p = k / n
    den = 1 + Z * Z / n
    c = (p + Z * Z / (2 * n)) / den
    h = Z * math.sqrt(p * (1 - p) / n + Z * Z / (4 * n * n)) / den
    return 100 * (c - h), 100 * (c + h)


def fam(label):
    l = label.split('·')[0]
    if l.startswith('SWG'):
        return 'SWG'
    if l.startswith('EQ'):
        return 'EQH/EQL'
    if l.startswith('OR-'):
        return 'OR'
    if l.startswith('PD'):
        return 'PD'
    if l.startswith('ON'):
        return 'ON'
    if l.startswith('RTH'):
        return 'RTH'
    if 'VWAP' in l:
        return 'VWAP-family'
    if l in ('POC', 'nPOC'):
        return 'POC/nPOC'
    if l.split('(')[0] in ('OB', 'DEMAND', 'SUPPLY', 'FVG', 'IFVG'):
        return 'OB/FVG/SD'
    return 'other'


def tf(label):
    if '·' in label:
        return label.split('·')[1]
    return 'daily/line'


def section(t):
    print("\n" + "=" * 70 + f"\n{t}\n" + "=" * 70)


# ─────────────────────────────────────────────────────────────────────────────
section("S1 — table state (read 2026-09-11, live DB, mode=ro)")
for tbl in ('touch_episodes', 'touch_outcomes', 'candidate_pool', 'trade_excursions'):
    r = DB.execute(f"SELECT COUNT(*) FROM {tbl}").fetchone()
    print(f"  {tbl}: {r[0]} rows")
r = DB.execute("SELECT MIN(opened_at_ms), MAX(opened_at_ms), MIN(session_day), MAX(session_day) FROM touch_episodes").fetchone()
print(f"  touch_episodes opened {ct(r[0])} → {ct(r[1])} CT; session_days {r[2]} → {r[3]}")
r = DB.execute("SELECT MIN(created_at), MAX(created_at) FROM touch_episodes").fetchone()
print(f"  touch_episodes created {r[0]} → {r[1]} (UTC) — STILL BEING WRITTEN")
r = DB.execute("SELECT MIN(opened_at_ms), MAX(opened_at_ms) FROM touch_outcomes").fetchone()
print(f"  touch_outcomes opened {ct(r[0])} → {ct(r[1])} CT")
r = DB.execute("SELECT MIN(created_at), MAX(created_at) FROM touch_outcomes").fetchone()
print(f"  touch_outcomes created {r[0]} → {r[1]}")
print("  touch_outcomes outcome mix:", [dict(x) for x in
      DB.execute("SELECT outcome, COUNT(*) n FROM touch_outcomes GROUP BY 1")])
print("  touch_outcomes validity:", [dict(x) for x in
      DB.execute("SELECT validity, COUNT(*) n FROM touch_outcomes GROUP BY 1")])
print("  touch_outcomes k/horizon/exit_on:",
      [dict(x) for x in DB.execute(
          "SELECT k, horizon, exit_on, COUNT(*) n FROM touch_outcomes GROUP BY 1,2,3")])
print("  touch_episodes close_1m==close_5m on all rows:",
      DB.execute("SELECT COUNT(*)=SUM(close_1m=close_5m) FROM touch_episodes").fetchone()[0])
print("  touch_episodes duplicate (trader,symbol,label,price,day,number) keys:",
      DB.execute("""SELECT COUNT(*)-COUNT(DISTINCT trader_id||'|'||symbol||'|'||label||'|'||
                    level_price||'|'||session_day||'|'||touch_number) FROM touch_episodes""").fetchone()[0])
print("  touch_episodes touch_number distribution:",
      [dict(x) for x in DB.execute(
          "SELECT touch_number, COUNT(*) n FROM touch_episodes GROUP BY 1 ORDER BY 1 LIMIT 12")])
print("  W1 opportunity_outcome:", [dict(x) for x in DB.execute(
      "SELECT opportunity_outcome, COUNT(*) n FROM touch_outcomes GROUP BY 1")])

# ─────────────────────────────────────────────────────────────────────────────
section("S2 — B4 base-rate reproduction attempts (pinned: 48.8%, n=423, Wilson[44.1,53.6])")
for cond in (
    "ordinal=1 AND created_at<='2026-09-05 15:00:00'",
    "created_at<='2026-09-05 15:00:00'",
    "ordinal=1 AND validity IN ('legacy:unverified','valid','invalid:duplicate')",
    "ordinal=1 AND validity='unverified:no_formation'",
    "ordinal=1",
    "ordinal=1 AND validity!='invalid:duplicate'",
    "1=1",
):
    r = DB.execute(f"""SELECT SUM(outcome='hold') h, SUM(outcome='break') b,
                       SUM(ambiguous) a FROM touch_outcomes WHERE {cond}""").fetchone()
    n = r['h'] + r['b']
    p = r['h'] / n if n else 0.0
    lo, hi = wilson(r['h'], n)
    print(f"  {cond}: n={n} hold={r['h']} break={r['b']} amb={r['a']} p={p:.4f} Wilson[{lo:.1f},{hi:.1f}]")
r = DB.execute("SELECT COUNT(*) n, COUNT(DISTINCT level_price||'|'||opened_at_ms) keys "
               "FROM touch_outcomes WHERE created_at<='2026-09-05 15:00:00'").fetchone()
print(f"  note: the 677-row 09-05 corpus holds {r['n']} rows / {r['keys']} price-time keys "
      "— n=423 reproduces as the KEY count of that corpus, not as any first-touch rate today.")

# ─────────────────────────────────────────────────────────────────────────────
section("S3 — B2 join yields")
eps = [dict(r) for r in DB.execute("""SELECT id, trader_id, symbol, label, level_price, session_day,
        touch_number, opened_at_ms, closed_at_ms, bars_in, penetration_pts, wick_pen_pts,
        body_pen_pts, vol_ratio, approach_atr FROM touch_episodes""")]
outs = [dict(r) for r in DB.execute("""SELECT id, trader_id, symbol, level_price, level_kind, ordinal,
        outcome, ambiguous, entry_side, opened_at_ms, closed_at_ms, k, delta, band_pts, horizon,
        validity FROM touch_outcomes""")]
for o in outs:
    o['cme_day'] = cmeday(o['opened_at_ms'])
for e in eps:
    e['sess'] = sess(e['opened_at_ms'])

earliest = min(o['opened_at_ms'] for o in outs)
eps_ov = [e for e in eps if e['opened_at_ms'] >= earliest]
out_idx = defaultdict(list)
for o in outs:
    out_idx[(o['trader_id'], o['symbol'], o['level_price'], o['cme_day'], o['ordinal'])].append(o)
strict = [e for e in eps_ov if (e['trader_id'], e['symbol'], e['level_price'], e['session_day'],
                                e['touch_number']) in out_idx]
gaps = []
for e in strict:
    for o in out_idx[(e['trader_id'], e['symbol'], e['level_price'], e['session_day'], e['touch_number'])]:
        gaps.append(o['opened_at_ms'] - e['opened_at_ms'])
g = np.array(gaps) / 60000
print(f"  overlap episodes (opened >= {ct(earliest)}): {len(eps_ov)} of {len(eps)} total")
print(f"  STRICT join (trader,symbol,level_price,session_day,ordinal): matched {len(strict)} "
      f"({len(strict)/len(eps_ov)*100:.1f}%); pair gaps n={len(g)} median={np.median(g):.1f} min "
      f"|gap|<=10min: {(np.abs(g)<=10).mean()*100:.1f}%  → UNRELIABLE pairing")
out_key2 = defaultdict(list)
for o in outs:
    out_key2[(o['trader_id'], o['symbol'], o['level_price'], o['cme_day'])].append(o)
has_any = [e for e in eps_ov if (e['trader_id'], e['symbol'], e['level_price'], e['session_day']) in out_key2]
print(f"  episodes with ANY outcome row for (price, cme_day): {len(has_any)}; with NONE: {len(eps_ov)-len(has_any)}")
TOL = 10 * 60000
pairs = []
for e in eps_ov:
    cands = [o for o in out_key2.get((e['trader_id'], e['symbol'], e['level_price'], e['session_day']), [])
             if abs(o['opened_at_ms'] - e['opened_at_ms']) <= TOL]
    if not cands:
        continue
    best = min(cands, key=lambda o: abs(o['opened_at_ms'] - e['opened_at_ms']))
    pairs.append({**e, 'outcome_id': best['id'], 'verdict': best['outcome'],
                  'ambiguous': best['ambiguous'], 'entry_side': best['entry_side'],
                  'kind': best['level_kind'], 'validity': best['validity'], 'k': best['k'],
                  'delta': best['delta'], 'band': best['band_pts'], 'horizon': best['horizon'],
                  'out_opened': best['opened_at_ms'], 'out_closed': best['closed_at_ms'],
                  'gap_min': (best['opened_at_ms'] - e['opened_at_ms']) / 60000})
for p in pairs:
    p['family'] = fam(p['label'])
    p['tf'] = tf(p['label'])
cnt = Counter(p['outcome_id'] for p in pairs)
unq = [p for p in pairs if cnt[p['outcome_id']] == 1]
g2 = np.array([p['gap_min'] for p in pairs])
print(f"  TIME join (same price+day, |gap|<=10min): {len(pairs)} pairs "
      f"({len(pairs)/len(eps_ov)*100:.1f}% of overlap episodes); median gap {np.median(g2):.1f} min; "
      f"unique-outcome pairs: {len(unq)}")
print(f"  why episodes do not join: {len(eps_ov)-len(has_any)} have no outcome row for that "
      "(price, day) at all (level not seated at any read, price re-anchored, or a band-only touch "
      "that never crossed the level — D1' requires a genuine touch); "
      f"{len(has_any)-len(pairs)} have outcome rows but none within 10 min (different touches / "
      "ordinal drift across the two instruments)")

with open('pairs_primary.csv', 'w') as f:
    cols = ['id', 'outcome_id', 'label', 'level_price', 'session_day', 'touch_number', 'sess',
            'opened_at_ms', 'closed_at_ms', 'out_opened', 'out_closed', 'verdict', 'ambiguous',
            'entry_side', 'kind', 'validity', 'family', 'tf', 'bars_in', 'penetration_pts',
            'wick_pen_pts', 'body_pen_pts', 'vol_ratio', 'approach_atr']
    f.write(','.join(cols) + '\n')
    for p in unq:
        f.write(','.join(str(p[c]) for c in cols) + '\n')
print("  wrote pairs_primary.csv (%d rows: the A21 sample-id artifact)" % len(unq))

# ─────────────────────────────────────────────────────────────────────────────
section("S4 — B3 population cells (time-verified pairs)")
for which, data in (("all pairs", pairs), ("unique-outcome pairs (PRIMARY)", unq)):
    vc = Counter(p['verdict'] for p in data)
    print(f"  [{which}] n={len(data)} verdicts {dict(vc)}")
for dim, key in (("entry_side", 'entry_side'), ("family", 'family'), ("tf", 'tf'), ("session", 'sess'),
                 ("validity", 'validity')):
    print(f"  verdict x {dim} (all pairs; 'OK' = both hold&break cells >=30):")
    for k in sorted(set(p[key] for p in pairs)):
        vc = Counter(p['verdict'] for p in pairs if p[key] == k)
        ok = (vc.get('hold', 0) >= 30 and vc.get('break', 0) >= 30)
        print(f"    {k:<16} n={sum(vc.values()):>4} {dict(vc)} {'OK' if ok else 'n<30'}")

# ─────────────────────────────────────────────────────────────────────────────
section("S5 — B5 field-by-field, hold vs break")
for which, data in (("unique-outcome pairs (PRIMARY)", unq), ("all pairs (sensitivity)", pairs)):
    hb = [p for p in data if p['verdict'] in ('hold', 'break')]
    H = np.array([p['verdict'] == 'hold' for p in hb])
    nH, n = int(H.sum()), len(hb)
    V = {f: np.array([p[f] for p in hb], float) for f in FIELDS}
    s0 = np.array([abs(stats.mannwhitneyu(V[f][H], V[f][~H]).statistic - nH * (n - nH) / 2)
                   for f in FIELDS])
    maxs = np.empty(R)
    for r in range(R):
        perm = RNG.permutation(n)
        best = 0.0
        for f in FIELDS:
            v = V[f][perm]
            u = stats.mannwhitneyu(v[:nH], v[nH:]).statistic
            best = max(best, abs(u - nH * (n - nH) / 2))
        maxs[r] = best
    pwy = np.array([(maxs >= s).mean() for s in s0])
    print(f"\n  [{which}] n={n} hold={nH} break={n-nH}  (Westfall-Young max-T over {len(FIELDS)} "
          f"fields, {R} perms; Bonferroni x{len(FIELDS)})")
    print(f"  {'field':<15}{'medH':>8}{'medB':>8}{'U':>10}{'p_raw':>8}{'p_WY':>8}{'p_bonf':>8}")
    for f, s, pw in zip(FIELDS, s0, pwy):
        u, p = stats.mannwhitneyu(V[f][H], V[f][~H])
        print(f"  {f:<15}{np.median(V[f][H]):8.2f}{np.median(V[f][~H]):8.2f}"
              f"{u:10.1f}{p:8.4f}{pw:8.4f}{min(p * len(FIELDS), 1):8.4f}")
    v = V['wick_pen_pts']
    med = np.median(v)
    hi, lo = v > med, v <= med
    for nm, m in (("high-wick half", hi), ("low-wick half", lo)):
        k, nn = int((H & m).sum()), int(m.sum())
        a, b = wilson(k, nn)
        print(f"    {nm}: hold {k}/{nn} = {k/nn:.3f} Wilson[{a:.1f},{b:.1f}]")
    print("    wick quartiles (hold rate):")
    qs = np.quantile(v, [.25, .5, .75])
    prev = -1.0
    for bnd in list(qs) + [np.inf]:
        m = (v > prev) & (v <= bnd)
        if m.sum():
            k, nn = int((H & m).sum()), int(m.sum())
            a, b = wilson(k, nn)
            print(f"      ({prev:>5.1f},{bnd:>5.1f}]: n={nn} hold={k} {k/nn:.3f} Wilson[{a:.1f},{b:.1f}]")
        prev = bnd

# ─────────────────────────────────────────────────────────────────────────────
section("S6 — mechanical-overlap check + B6 duration confound (PRIMARY, H=12)")
hb = [p for p in unq if p['verdict'] in ('hold', 'break')]
H = np.array([p['verdict'] == 'hold' for p in hb])
for nm, m in (("hold", H), ("break", ~H)):
    sub = [p for p, mm in zip(hb, m) if mm]
    g = np.array([(p['out_closed'] - p['closed_at_ms']) / 60000 for p in sub])
    print(f"  {nm}: n={len(sub)} gap(outcome.closed - episode.closed) min "
          f"median={np.median(g):.1f} p25={np.percentile(g,25):.1f} p75={np.percentile(g,75):.1f}")
b = np.array([p['band'] for p in hb])
print(f"  recorded band k*delta: median={np.median(b):.2f} pts "
      f"(break wick median {np.median(np.array([p['wick_pen_pts'] for p in hb])[~H]):.2f} pts)")
vr = np.array([p['vol_ratio'] for p in hb])
bi = np.array([p['bars_in'] for p in hb], float)
vpb = np.divide(vr, bi, out=np.zeros_like(vr), where=bi > 0)
rho = stats.spearmanr(vr, bi)
print(f"  spearman(vol_ratio, bars_in) = {rho.statistic:.3f} (p={rho.pvalue:.2e}, n={len(hb)})")
print(f"  vol_ratio raw:      medH={np.median(vr[H]):.3f} medB={np.median(vr[~H]):.3f} "
      f"U p={stats.mannwhitneyu(vr[H], vr[~H])[1]:.4f}")
print(f"  vol_ratio/bars_in:  medH={np.median(vpb[H]):.3f} medB={np.median(vpb[~H]):.3f} "
      f"U p={stats.mannwhitneyu(vpb[H], vpb[~H])[1]:.4f}")

# ─────────────────────────────────────────────────────────────────────────────
section("S7 — B7 horizons (PRIMARY set; 5m verdicts recomputed from bars table)")
brows = DB.execute("SELECT open_time_ms, c FROM bars WHERE symbol='MNQ' AND tf='5m' "
                   "ORDER BY open_time_ms").fetchall()
seen = set()
opens, closes = [], []
for r in brows:
    if r['open_time_ms'] in seen:
        continue
    seen.add(r['open_time_ms'])
    opens.append(r['open_time_ms'])
    closes.append(r['c'])
opens = np.array(opens)
closes = np.array(closes)


def hv5(p, h):
    up, lo = p['level_price'] + p['k'] * p['delta'], p['level_price'] - p['k'] * p['delta']
    j0 = int(np.searchsorted(opens, p['opened_at_ms'], side='right'))
    if j0 + h > len(closes):
        return None
    e = p['entry_side']
    for j in range(j0, j0 + h):
        c = closes[j]
        cu, cd = c > up, c < lo
        if cu and cd:
            return None
        if cu:
            return ('hold' if e == 'above' else 'break')
        if cd:
            return ('break' if e == 'above' else 'hold')
    return None


for hor, unit in ((12, 'recorded D1\' (12x1m bars)'), (10, '10x5m recomputed'), (20, '20x5m recomputed')):
    hb = []
    cens = 0
    for p in unq:
        if p['verdict'] == 'ambiguous_horizon':
            continue
        v = p['verdict'] if hor == 12 else hv5(p, hor)
        if v is None:
            cens += 1
            continue
        hb.append((p, v))
    H = np.array([v == 'hold' for _, v in hb])
    print(f"\n  horizon {unit}: resolved {len(hb)} (censored {cens}); hold {H.sum()}/{len(hb)} = {H.mean():.4f}")
    if len(hb) >= 60:
        print(f"  {'field':<15}{'medH':>8}{'medB':>8}{'p_raw':>8}")
        for f in FIELDS:
            v = np.array([p[f] for p, _ in hb], float)
            u, pr = stats.mannwhitneyu(v[H], v[~H])
            print(f"  {f:<15}{np.median(v[H]):8.2f}{np.median(v[~H]):8.2f}{pr:8.4f}")
    v = np.array([p['vol_ratio'] for p, _ in hb])
    b_ = np.array([p['bars_in'] for p, _ in hb], float)
    vpb = np.divide(v, b_, out=np.zeros_like(v), where=b_ > 0)
    if hor != 12:
        print(f"  vol_ratio/bars_in at this horizon: medH={np.median(vpb[H]):.3f} "
              f"medB={np.median(vpb[~H]):.3f} U p={stats.mannwhitneyu(vpb[H], vpb[~H])[1]:.4f}")

# ─────────────────────────────────────────────────────────────────────────────
section("S8 — closing: minimum n per cell for a 5-point effect")
for alpha, name in ((0.05, "alpha=0.05"), (0.05 / len(FIELDS), f"Bonferroni alpha/{len(FIELDS)}"),
                    (0.05 / 2, "alpha=0.025")):
    z1 = stats.norm.ppf(1 - alpha / 2)
    zb = stats.norm.ppf(0.8)
    p0, p1 = 0.488, 0.538
    pbar = (p0 + p1) / 2
    n = ((z1 * math.sqrt(2 * pbar * (1 - pbar)) + zb * math.sqrt(p0 * (1 - p0) + p1 * (1 - p1))) /
         (p1 - p0)) ** 2
    print(f"  {name:<22} n per group = {n:.0f}  (two-proportion, 80% power, 2-sided, "
          f"p0={p0}, p1={p1})")
print("\n  Today's largest joined cell: hold n=130 / break n=80 (H=12, unique pairs).")
