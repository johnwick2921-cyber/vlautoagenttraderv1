#!/usr/bin/env python3
"""Round 23 analysis — consumes harness outputs (episodes.jsonl, q1_cells.json,
q4_ordinals.json) and prints Q1/Q2/Q3/Q4/Q5 tables with n + Wilson CI + p-vs-null.

Read-only. Run: python3 docs/superpowers/research/2026-09-16-round-23/analysis.py <out-dir>
"""
import json
import math
import sys
from collections import defaultdict

NULL = 0.5067  # D1' IID calibration p(hold)


def wilson(p, n, z=1.959963984540054):
    if n <= 0:
        return (0.0, 0.0)
    den = 1 + z * z / n
    c = (p + z * z / (2 * n)) / den
    half = z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / den
    return (c - half, c + half)


def fmt_cell(hold, n):
    p = hold / n
    lo, hi = wilson(p, n)
    return f"{p:.3f} [{lo:.3f},{hi:.3f}] n={n}"


def load_episodes(path):
    eps = []
    with open(path) as f:
        for line in f:
            eps.append(json.loads(line))
    return eps


def hold_count(rows):
    h = sum(1 for r in rows if r.get("outcome") == "hold")
    b = sum(1 for r in rows if r.get("outcome") == "break")
    return h, h + b


def main(out):
    eps = load_episodes(f"{out}/episodes.jsonl")
    print(f"episodes: {len(eps)}")

    # ── Q1: TF × kind first-touch hold (ordinal-1 only) ────────────────────
    print("\n== Q1: first-touch hold by detection-TF x kind (ordinal-1) ==")
    o1 = [e for e in eps if e["ordinal"] == 1]
    by = defaultdict(list)
    for e in o1:
        by[(e["tf"], e["kind"])].append(e)
    rows = []
    for (tf, kind), rs in sorted(by.items()):
        h, n = hold_count(rs)
        if n == 0:
            continue
        p = h / n
        lo, hi = wilson(p, n)
        rows.append((tf, kind, h, n, p, lo, hi))
    print(f"{'tf':>4} {'kind':<10} {'hold':>6} {'n':>7}  {'rate':>6} {'CI':<21} verdict")
    for tf, kind, h, n, p, lo, hi in rows:
        verdict = "UNMEASURED" if n < 200 else ("above" if lo > NULL else ("below" if hi < NULL else "null"))
        print(f"{tf:>4} {kind:<10} {h:>6} {n:>7}  {p:>6.3f} [{lo:.3f},{hi:.3f}] {verdict}")

    # ── Q1 families ────────────────────────────────────────────────────────
    print("\n== Q1 by family x TF (ordinal-1) ==")
    fam = defaultdict(list)
    for e in o1:
        fam[(e["tf"], e["family"])].append(e)
    for (tf, f_), rs in sorted(fam.items()):
        h, n = hold_count(rs)
        if n == 0:
            continue
        p = h / n
        lo, hi = wilson(p, n)
        print(f"{tf:>4} {f_:<18} {fmt_cell(h, n)}")

    # ── Q2: HTF vs intraday, distance + freshness controlled ───────────────
    print("\n== Q2: hold/MFE, HTF vs intraday, by distance bucket (ordinal-1) ==")
    def dist_bucket(d):
        return "0-25" if d <= 25 else "25-50" if d <= 50 else "50-100" if d <= 100 else "100-200" if d <= 200 else "200+"
    def age_bucket(a):
        if a < 0:
            return "unknown"
        d = a / 86_400_000
        return "<1d" if d < 1 else "1-3d" if d <= 3 else "3-7d" if d <= 7 else ">7d"
    def is_htf(e):
        return e["tf"] in ("4h", "1d") or e["kind"] in ("PWH", "PWL")
    for db in ("0-25", "25-50", "50-100", "100-200", "200+"):
        for htf in (False, True):
            rs = [e for e in o1 if is_htf(e) == htf and dist_bucket(e["dist_at_read"]) == db]
            h, n = hold_count(rs)
            if n == 0:
                continue
            mfe = sum(e["mfe"] for e in rs) / n
            label = "HTF" if htf else "intraday"
            print(f"dist {db:>7} {label:<9} {fmt_cell(h, n)}  meanMFE {mfe:7.2f}")
    print("freshness buckets (HTF only, dist 0-100):")
    for ab in ("<1d", "1-3d", "3-7d", ">7d"):
        rs = [e for e in o1 if is_htf(e) and dist_bucket(e["dist_at_read"]) in ("0-25", "25-50", "50-100") and age_bucket(e["age_at_read_ms"]) == ab]
        h, n = hold_count(rs)
        if n == 0:
            continue
        print(f"  age {ab:>5} {fmt_cell(h, n)}")

    # ── Q3: measured ladder vs zoneTFMult 1.0/1.1/1.2/1.3 ──────────────────
    print("\n== Q3: measured per-TF hold (ordinal-1, all kinds) vs the tier ladder ==")
    ladder = {"1m": 1.0, "5m": 1.05, "15m": 1.1, "1h": 1.2, "4h": 1.3, "1d": 1.3}
    base = None
    for tf in ("1m", "5m", "15m", "1h", "4h", "1d"):
        rs = [e for e in o1 if e["tf"] == tf]
        h, n = hold_count(rs)
        if n == 0:
            print(f"{tf:>4} (no rows)")
            continue
        p = h / n
        if base is None:
            base = p
        print(f"{tf:>4} {fmt_cell(h, n)}  measured/base={p/base if base else 0:.3f}  ladder={ladder[tf]}")

    # ── Q4: reference-line decay by ordinal ────────────────────────────────
    print("\n== Q4: reference kinds by touch ordinal (all episodes) ==")
    q4 = json.load(open(f"{out}/q4_ordinals.json"))
    for row in q4:
        print(f"{row['kind']:<8} ord {row['ordinal']:>3}: {fmt_cell(row['hold'], row['n'])}  ambig={row['ambig']}")

    # ── Q5: entry timing on ordinal-1 episodes ─────────────────────────────
    print("\n== Q5: entry timing (ordinal-1 episodes) ==")
    for variant in ("outcome", "confirm_5m_outcome", "mss_outcome"):
        h = b = amb = 0
        for e in o1:
            v = e.get(variant)
            if v == "hold":
                h += 1
            elif v == "break":
                b += 1
            else:
                amb += 1
        n = h + b
        print(f"{variant:<20} {fmt_cell(h, n)}  other/ambig={amb}")
    print("pairwise: episodes with BOTH touch and 5m-confirm resolved:")
    both = [e for e in o1 if e.get("confirm_5m_outcome") in ("hold", "break")]
    th = sum(1 for e in both if e["outcome"] == "hold")
    ch = sum(1 for e in both if e["confirm_5m_outcome"] == "hold")
    n = len(both)
    print(f"  n={n} touch-hold {th}/{n} ({th/n:.3f}) confirm-hold {ch}/{n} ({ch/n:.3f})")
    bothm = [e for e in o1 if e.get("mss_outcome") in ("hold", "break")]
    th = sum(1 for e in bothm if e["outcome"] == "hold")
    mh = sum(1 for e in bothm if e["mss_outcome"] == "hold")
    n = len(bothm)
    print(f"  n={n} touch-hold {th}/{n} ({th/n:.3f}) mss-hold {mh}/{n} ({mh/n:.3f})")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "out")
