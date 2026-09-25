#!/usr/bin/env python3
"""S4b cells (CTO dispatch 2026-09-17 02:26Z): (1) mixed-cell split D-with/4h-against
vs D-against/4h-with, per side, per session; (3) oppose-D short penalty by month.
Read-only. Run: python3 s4b.py <out-s4-dir>
"""
import json
import math
import sys
from collections import defaultdict
from statistics import NormalDist

NULL = 0.5067
Z = 1.959963984540054


def wilson(p, n):
    if n <= 0:
        return (0.0, 0.0)
    den = 1 + Z * Z / n
    c = (p + Z * Z / (2 * n)) / den
    half = Z * math.sqrt(p * (1 - p) / n + Z * Z / (4 * n * n)) / den
    return (c - half, c + half)


def two(h1, n1, h2, n2):
    if n1 <= 0 or n2 <= 0:
        return 0.0, 1.0
    p1, p2 = h1 / n1, h2 / n2
    pp = (h1 + h2) / (n1 + n2)
    se = math.sqrt(pp * (1 - pp) * (1 / n1 + 1 / n2))
    if se == 0:
        return 0.0, 1.0
    z = (p1 - p2) / se
    return z, 2 * (1 - NormalDist().cdf(abs(z)))


def load(p):
    rows = []
    with open(p) as f:
        for line in f:
            if line.strip():
                rows.append(json.loads(line))
    return rows


def hb(rows):
    return [r for r in rows if r["outcome"] in ("hold", "break")]


def cell(rows):
    rows = hb(rows)
    n = len(rows)
    if n == 0:
        return "NOT MEASURED"
    h = sum(1 for r in rows if r["outcome"] == "hold")
    p = h / n
    lo, hi = wilson(p, n)
    return f"{p:.3f} [{lo:.3f},{hi:.3f}] n={n}"


def eid(e):
    return f"{e['day']} {e['session']} {e['kind']} {e['tf']}@{e['opened_at_ms']}"


def main(out):
    eps = load(f"{out}/episodes.jsonl")
    trs = {(t["day"], t["session"]): t for t in load(f"{out}/trends.jsonl")}
    o1 = [e for e in eps if e["ordinal"] == 1]

    def direction(e):
        return "long" if e.get("entry") == "below" else "short"

    def rel_d(e):
        t = trs.get((e["day"], e["session"])) or {}
        d = t.get("d_trend")
        if d not in ("up", "down"):
            return "range"
        return "agree" if ((d == "up") == (direction(e) == "long")) else "oppose"

    def rel_h(e):
        t = trs.get((e["day"], e["session"])) or {}
        h = t.get("h4_trend")
        if h not in ("up", "down"):
            return "range"
        return "agree" if ((h == "up") == (direction(e) == "long")) else "oppose"

    # ── S4b(1): mixed-cell split ──────────────────────────────────────────
    print("=" * 72)
    print("S4b(1) mixed-cell split: D-with/4h-against vs D-against/4h-with")
    print("=" * 72)
    mixed = [e for e in o1 if rel_d(e) in ("agree", "oppose") and rel_h(e) in ("agree", "oppose")
             and rel_d(e) != rel_h(e)]
    dwa = [e for e in mixed if rel_d(e) == "agree"]  # D-with, 4h-against
    daw = [e for e in mixed if rel_d(e) == "oppose"]  # D-against, 4h-with
    print(f"  D-with/4h-against : {cell(dwa)}")
    print(f"  D-against/4h-with : {cell(daw)}")
    h1, n1 = sum(1 for r in hb(dwa) if r["outcome"] == "hold"), len(hb(dwa))
    h2, n2 = sum(1 for r in hb(daw) if r["outcome"] == "hold"), len(hb(daw))
    z, p = two(h1, n1, h2, n2)
    print(f"  two-prop: z={z:+.2f} p={p:.4f}")
    print("  per side:")
    for sd in ("long", "short"):
        a = [e for e in dwa if direction(e) == sd]
        b = [e for e in daw if direction(e) == sd]
        print(f"    {sd:5s}: D-with/4h-against {cell(a)} | D-against/4h-with {cell(b)}")
    print("  per session:")
    for sess in ("LONDON", "NY", "ASIA"):
        a = [e for e in dwa if e["session"] == sess]
        b = [e for e in daw if e["session"] == sess]
        print(f"    {sess:8s}: D-with/4h-against {cell(a)} | D-against/4h-with {cell(b)}")
    print("  ids D-with/4h-against (first 5):", ", ".join(eid(r) for r in dwa[:5]))
    print("  ids D-against/4h-with (first 5):", ", ".join(eid(r) for r in daw[:5]))

    # ── S4b(3): oppose-D short penalty by month ───────────────────────────
    print()
    print("=" * 72)
    print("S4b(3) oppose-D short penalty by month (era-artifact check)")
    print("=" * 72)
    shorts = [e for e in o1 if direction(e) == "short"]
    by_month = defaultdict(lambda: {"agree": [], "oppose": []})
    for e in shorts:
        rd = rel_d(e)
        if rd in ("agree", "oppose"):
            by_month[e["day"][:7]][rd].append(e)
    print(f"  {'month':8s} {'agree':>22s} {'oppose':>22s}")
    for m in sorted(by_month):
        a = cell(by_month[m]["agree"])
        b = cell(by_month[m]["oppose"])
        print(f"  {m:8s} {a:>22s} {b:>22s}")
    # era summary + year halves
    for label, months in [("2022", [m for m in sorted(by_month) if m.startswith("2022")]),
                          ("2023", [m for m in sorted(by_month) if m.startswith("2023")]),
                          ("2024", [m for m in sorted(by_month) if m.startswith("2024")]),
                          ("2025", [m for m in sorted(by_month) if m.startswith("2025")]),
                          ("2026", [m for m in sorted(by_month) if m.startswith("2026")])]:
        ag = [e for m in months for e in by_month[m]["agree"]]
        op = [e for m in months for e in by_month[m]["oppose"]]
        if not ag and not op:
            continue
        print(f"  {label}: agree {cell(ag)} | oppose {cell(op)}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    main(sys.argv[1])
