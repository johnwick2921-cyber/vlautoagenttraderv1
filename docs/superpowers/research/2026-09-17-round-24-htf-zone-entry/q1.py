#!/usr/bin/env python3
"""Q1 — ZONE x TREND. Ordinal-1 episodes at an HTF zone (inside or within 1*delta; zone tf
1h/4h/1d; kinds OB/FVG/IFVG/SUPPLY/DEMAND/EQH/EQL), split by (a) with-zone/against-zone,
(b) 4h trend with/against/range, (c) D trend with/against/range; the four-way cross.
Definitions: r24lib.DEFINITIONS (shared with q2/q5). Waits for the harness sentinel."""
import os, sys, json, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import *

OUTF = os.path.join(OUT, "q1_cells.json")
log = open(os.path.join(OUT, "q1.progress"), "a")

def main():
    wait_for(SENTINEL, log)
    trace = {"script": os.path.abspath(__file__), "commit": git_sha(),
             "inputs": {EPISODES: sha256_file(EPISODES), ZONES: sha256_file(ZONES), ERA_TRENDS: sha256_file(ERA_TRENDS)},
             "output": OUTF, "definitions": DEFINITIONS}
    zones = load_zones(); trends = load_trends()
    cells = collections.defaultdict(Cell)
    counts = collections.Counter()
    n = 0
    for e in iter_episodes(True):
        n += 1
        if n % 500000 == 0:
            log.write("episodes %d\n" % n); log.flush()
        z = zones.get((e["day"], e["session"]))
        if not z:
            counts["reads_without_htf_zones"] += 1; continue
        pl = place(e, z)
        if pl is None:
            counts["not_at_zone"] += 1; continue
        member, zrel, zkind, ztf, conflict, own = pl
        counts["at_zone_" + member] += 1
        if conflict: counts["conflict_both_polarities"] += 1
        if own: counts["own_zone_episode"] += 1
        d = hold_trade(e)
        dt, h4 = trends.get((e["day"], e["session"]), (None, None))
        r4, rD = trend_rel(d, h4), trend_rel(d, dt)
        y = year_of(e)
        for v in variants_of(conflict, own):
          for scope in ("era", y):
            cells[(v, scope, "zone", zrel, "*", "*")].add(e)                 # (a)
            cells[(v, scope, "zone", zrel, "4h", r4)].add(e)                 # (b)
            cells[(v, scope, "zone", zrel, "D", rD)].add(e)                  # (c)
            cells[(v, scope, "cross", zrel, "4h=" + r4, "D=" + rD)].add(e)  # four-way
            cells[(v, scope, "member", member, zrel, "*")].add(e)
            cells[(v, scope, "side", d, zrel, "*")].add(e)
    out = {"trace": trace, "episodes_ordinal1": n, "counts": dict(counts),
           "cells": {"|".join(k): c.row() for k, c in sorted(cells.items())}}
    json.dump(out, open(OUTF, "w"), indent=1)
    print("Q1 — ZONE x TREND (ordinal-1, hold-trade direction; R23 'approach' label = the opposite side; p_null=%.4f)" % P_NULL)
    print("counts:", dict(counts))
    for v in VARIANTS:
      print("\n#################### VARIANT %s ####################" % v)
      for scope in ["era"] + sorted({k[1] for k in cells if k[1] != "era"}):
        print("\n== %s / %s ==" % (v, scope))
        cells_v = {k[1:]: c for k, c in cells.items() if k[0] == v}
        _print_scope(cells_v, scope)
    print("\ndone episodes=%d output=%s" % (n, OUTF))

def _print_scope(cells, scope):
    if True:
        for zrel in ("with", "against", "unknown"):
            r = cells.get((scope, "zone", zrel, "*", "*"))
            if r: print("  (a) %-8s zone: %s" % (zrel, fmt(r.row())))
        for tfl in ("4h", "D"):
            for zrel in ("with", "against"):
                for rel in ("with", "against", "range", "n/a"):
                    r = cells.get((scope, "zone", zrel, tfl, rel))
                    if r: print("  (%s) %-7s zone x %s-%-7s: %s" % ("b" if tfl == "4h" else "c", zrel, tfl, rel, fmt(r.row())))
        for zrel in ("with", "against"):
            for r4 in ("with", "against"):
                for rD in ("with", "against"):
                    r = cells.get((scope, "cross", zrel, "4h=" + r4, "D=" + rD))
                    if r: print("  4-way %-7s zone x 4h-%-7s x D-%-7s: %s" % (zrel, r4, rD, fmt(r.row())))
        for m in ("inside", "near"):
            for zrel in ("with", "against"):
                r = cells.get((scope, "member", m, zrel, "*"))
                if r: print("  member %-6s x %-7s: %s" % (m, zrel, fmt(r.row())))
        for d in ("long", "short"):
            for zrel in ("with", "against"):
                r = cells.get((scope, "side", d, zrel, "*"))
                if r: print("  side %-5s x %-7s: %s" % (d, zrel, fmt(r.row())))

if __name__ == "__main__":
    main()
