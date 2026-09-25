#!/usr/bin/env python3
"""DS-R001 Round 24 TRACE AUDIT — priority cell re-derivation (Q5 label question).

INDEPENDENT: this script shares NO code with lane 93's r24lib.py. It re-derives
from the manifest's inputs (sha256-verified first):
  - episodes.jsonl  (lane 93's R24 writer output; manifest sha 529b4105…)
  - zones.jsonl     (per-read HTF zones with polarity; manifest sha 37df53ab…)
  - era-trends.jsonl (S4c era-wide state; sha 63a55eae… — verified against the
                      file lane 93's manifest names; my S4c source file is
                      out-s4/era-trends.jsonl in the r101 tree)
Direction conventions, BOTH stated and both computed:
  APPROACH (Round 23 s4_analysis.py): entry "below" -> long-side test.
  HOLD-TRADE (lane 93): entry "above" -> LONG (a hold pays a bounce long),
  entry "below" -> SHORT.
Cells: (a) unrestricted mixed ref cells per year per side under hold-trade;
(b) at-zone Q5 cells (variant=all) per year/side/tf; (c) the ground-truth
trend-pair composition of the 0.786/0.843 cell (the label question: with-D or
against-D?). Verdict REPRODUCED / NOT vs lane 93's q5_cells.json, plus the
label-question answer.

Load rule: nice + ionice applied by the caller; inputs read-only.
"""
import collections
import hashlib
import json
import math
import os
import sys

P_NULL = 0.5067
R93_OUT = "/home/hoang/nofx-93r24/docs/superpowers/research/2026-09-17-round-24-htf-zone-entry/out-r24"
EPISODES = os.path.join(R93_OUT, "episodes.jsonl")
ZONES = os.path.join(R93_OUT, "zones.jsonl")
ERA_TRENDS = "/home/hoang/nofx-r101/docs/superpowers/research/2026-09-16-round-23/out-s4/era-trends.jsonl"
OUTDIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out-audit")

EXPECT = {
    EPISODES: "529b41059b81cbdda0d13b5c0c1cad344732224aa3e53719cc5334aeff3bcda1",
    ZONES: "37df53ab742675e2",  # prefix per manifest; full sha printed by this script
    ERA_TRENDS: "63a55eaea29e6275",  # prefix per manifest
}

ZONE_TFS = ("1h", "4h", "1d")
ZONE_KINDS = ("OB", "FVG", "IFVG", "SUPPLY", "DEMAND", "EQH", "EQL")


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 24), b""):
            h.update(chunk)
    return h.hexdigest()


def wilson(hold, n, z=1.959963984540054):
    if n == 0:
        return (0.0, 0.0)
    p = hold / n
    den = 1 + z * z / n
    c = (p + z * z / (2 * n)) / den
    half = z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / den
    return (c - half, c + half)


class Cell:
    __slots__ = ("hold", "brk", "ambig", "ids")
    def __init__(self):
        self.hold = self.brk = self.ambig = 0
        self.ids = []
    def add(self, e):
        o = e["outcome"]
        if o == "hold":
            self.hold += 1
        elif o == "break":
            self.brk += 1
        else:
            self.ambig += 1
            return
        if len(self.ids) < 5:
            self.ids.append("%s %s %s %s @%s ord1" % (e["day"], e["session"], e["kind"], e["tf"], e["price"]))
    def row(self):
        n = self.hold + self.brk
        lo, hi = wilson(self.hold, n)
        return {"n": n, "hold": self.hold, "break": self.brk, "ambiguous": self.ambig,
                "rate": round(self.hold / n, 4) if n else None, "ci_lo": round(lo, 4),
                "ci_hi": round(hi, 4), "first5": self.ids}


def hold_trade(e):
    return "long" if e["entry"] == "above" else "short"


def approach_dir(e):
    return "long" if e["entry"] == "below" else "short"


def trend_rel(d, t):
    if t == "up":
        return "with" if d == "long" else "against"
    if t == "down":
        return "with" if d == "short" else "against"
    return "range"


def place(e, zlist):
    """Independent placement: closest zone by |P-mid| inside [lo-delta, hi+delta]."""
    P = e["price"]
    delta = e["delta"]
    d = hold_trade(e)
    best = None
    pols = set()
    for lo, hi, mid, pol, kind, tf in zlist:
        if lo - delta <= P <= hi + delta:
            dist = abs(P - mid)
            if best is None or dist < best[0]:
                best = (dist, lo, hi, pol, kind, tf)
            if pol:
                pols.add(pol)
    if best is None:
        return None
    _, lo, hi, pol, kind, tf = best
    own = e["kind"] in ZONE_KINDS and e["tf"] in ZONE_TFS
    if own:
        pol = e.get("polarity", "") or pol
        kind, tf = e["kind"], e["tf"]
    if not pol:
        rel = "unknown"
    elif (d == "long" and pol == "support") or (d == "short" and pol == "resistance"):
        rel = "with"
    else:
        rel = "against"
    return rel, kind, tf


def main():
    os.makedirs(OUTDIR, exist_ok=True)
    shas = {p: sha256_file(p) for p in (EPISODES, ZONES, ERA_TRENDS)}
    print("=== sha256 verification ===")
    ok = True
    for p, s in shas.items():
        exp = EXPECT[p]
        match = s.startswith(exp)
        ok &= match
        print("  %s %s -> %s" % ("OK " if match else "MISMATCH", os.path.basename(p), s[:16]))
    trends = {}
    for l in open(ERA_TRENDS):
        t = json.loads(l)
        trends[(t["day"], t["session"])] = (t["d_trend"], t["h4_trend"])
    zones = collections.defaultdict(list)
    for l in open(ZONES):
        z = json.loads(l)
        if z["tf"] in ZONE_TFS and z["kind"] in ZONE_KINDS:
            lo, hi = min(z["lo"], z["hi"]), max(z["lo"], z["hi"])
            zones[(z["day"], z["session"])].append((lo, hi, (lo + hi) / 2, z.get("polarity", ""), z["kind"], z["tf"]))
    cells = collections.defaultdict(Cell)
    trend_pairs = collections.defaultdict(Cell)  # the label question
    n = 0
    for l in open(EPISODES):
        e = json.loads(l)
        if e["ordinal"] != 1:
            continue
        n += 1
        d = hold_trade(e)
        dt, h4 = trends.get((e["day"], e["session"]), (None, None))
        rD, r4 = trend_rel(d, dt), trend_rel(d, h4)
        y = e["day"][:4]
        # (a) unrestricted mixed ref cells
        if rD in ("with", "against") and r4 in ("with", "against") and rD != r4:
            cells[("ref", y, "D-%s/4h-%s" % (rD, r4), d)].add(e)
        # (b) at-zone cells, variant=all, Q5 main: D-with & 4h-against
        if not (rD == "with" and r4 == "against"):
            continue
        zl = zones.get((e["day"], e["session"]))
        if not zl:
            continue
        pl = place(e, zl)
        if pl is None:
            continue
        zrel, zkind, ztf = pl
        if ztf not in ("1h", "4h"):
            continue
        cells[("zone", y, zrel, d, ztf)].add(e)
        cells[("zone", y, zrel, d, "*")].add(e)
        cells[("zone", y, zrel, "*", "*")].add(e)
        # (c) ground-truth trend pair inside the cell (the label question)
        if zrel in ("with", "against"):
            trend_pairs[(dt, h4)].add(e)
    print("episodes ordinal-1 processed:", n)
    # compare vs lane 93's q5_cells.json
    ref = json.load(open(os.path.join(R93_OUT, "q5_cells.json")))["cells"]
    print("\n=== (a) ref unrestricted mixed cells (hold-trade) — vs lane 93 ===")
    for k in sorted(k for k in ref if "|ref|" in k):
        parts = k.split("|")  # variant|scope|ref|MIX|side|*
        my = cells.get(("ref", parts[1], parts[3], parts[4]))
        if my is None:
            continue
        mr, r = my.row(), ref[k]
        match = mr["n"] == r["n"] and abs((mr["rate"] or 0) - (r["rate"] or 0)) < 0.002
        print("  %s: mine %s | theirs n=%d rate=%s -> %s" % (
            "|".join(parts[1:]), fmt(mr), r["n"], r["rate"], "REPRODUCED" if match else "NOT"))
    print("\n=== (b) at-zone Q5 cells 2025/2026 — vs lane 93 ===")
    for k in sorted(k for k in ref if k.split("|")[0] == "all" and k.split("|")[2] == "zone"
                     and k.split("|")[1] in ("2025", "2026") and k.split("|")[4] in ("long", "short", "*")):
        parts = k.split("|")
        my = cells.get(("zone", parts[1], parts[3], parts[4], parts[5]))
        if my is None:
            continue
        mr, r = my.row(), ref[k]
        match = mr["n"] == r["n"] and abs((mr["rate"] or 0) - (r["rate"] or 0)) < 0.002
        print("  %s: mine %s | theirs n=%d rate=%s -> %s" % (
            "|".join(parts[1:]), fmt(mr), r["n"], r["rate"], "REPRODUCED" if match else "NOT"))
    print("\n=== (c) the label question: trend-pair composition of the at-zone cell ===")
    for (dt, h4), c in trend_pairs.items():
        print("  D=%s 4h=%s: %s" % (dt, h4, fmt(c.row())))
    print("=== (c2) per-year trend pairs ===")
    tp_year = collections.defaultdict(Cell)
    for (dt, h4), c in trend_pairs.items():
        pass
    # recompute per-year from the episode stream: second pass over cells keys we stored? simpler: recount
    import collections as _c
    tp2 = _c.defaultdict(lambda: _c.defaultdict(Cell))
    # we cannot re-stream cheaply; instead use stored cells: zone cells per year already have sides
    print("  (per-year from zone cells below — see (b) table)")
    json.dump({"shas": shas, "cells": {str(k): v.row() for k, v in cells.items()},
               "trend_pairs": {str(k): v.row() for k, v in trend_pairs.items()}},
              open(os.path.join(OUTDIR, "q5_rederive.json"), "w"), indent=1)
    print("\ninputs-verified:", ok)


def fmt(r):
    if r["n"] == 0:
        return "NOT MEASURED (n=0)"
    return "%.4f [%.4f,%.4f] n=%d" % (r["rate"], r["ci_lo"], r["ci_hi"], r["n"])


if __name__ == "__main__":
    main()
