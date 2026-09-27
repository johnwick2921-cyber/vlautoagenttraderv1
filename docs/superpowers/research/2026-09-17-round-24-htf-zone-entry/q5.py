#!/usr/bin/env python3
"""Q5 — PULLBACK ENTRY AT A ZONE: the D-with / 4h-against cell (hold-trade direction WITH
the D trend and AGAINST the 4h trend) restricted to ordinal-1 entries AT a 1h or 4h zone on
the pullback side (= with-zone: long at support in a D-uptrend, short at resistance in a
D-downtrend), per year, per side. The against-zone counterpart is printed for contrast.
Round 23 S4d(2)'s 0.786/0.810 cell was labelled by the APPROACH direction; see
DEFINITIONS['hold_trade'] for the mapping. Definitions shared: r24lib.DEFINITIONS."""
import os, sys, json, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import *

OUTF = os.path.join(OUT, "q5_cells.json")
log = open(os.path.join(OUT, "q5.progress"), "a")

def main():
    wait_for(SENTINEL, log)
    trace = {"script": os.path.abspath(__file__), "commit": git_sha(),
             "inputs": {EPISODES: sha256_file(EPISODES), ZONES: sha256_file(ZONES), ERA_TRENDS: sha256_file(ERA_TRENDS)},
             "output": OUTF, "definitions": DEFINITIONS,
             "cell": "relD=with AND rel4h=against (hold-trade direction), zone tf in {1h,4h}, per year, per hold-trade side; with-zone vs against-zone; plus the unrestricted D-with/4h-against cell for reference (R23 S4d(2) comparability under the R24 labelling)"}
    zones = load_zones(); trends = load_trends()
    cells = collections.defaultdict(Cell); n = 0
    for e in iter_episodes(True):
        n += 1
        if n % 500000 == 0:
            log.write("episodes %d\n" % n); log.flush()
        d = hold_trade(e)
        dt, h4 = trends.get((e["day"], e["session"]), (None, None))
        rD, r4 = trend_rel(d, dt), trend_rel(d, h4)
        y = year_of(e)
        # reference: the unrestricted mixed cell under R24 labelling (both mixed cells)
        if rD in ("with", "against") and r4 in ("with", "against") and rD != r4:
            for scope in ("era", y):
                cells[("all", scope, "ref", "D-%s/4h-%s" % (rD, r4), d, "*")].add(e)
        if not (rD == "with" and r4 == "against"):
            continue
        z = zones.get((e["day"], e["session"]))
        if not z: continue
        pl = place(e, z)
        if pl is None: continue
        member, zrel, zkind, ztf, conflict, own = pl
        if ztf not in ("1h", "4h"): continue
        for v in variants_of(conflict, own):
          for scope in ("era", y):
            cells[(v, scope, "zone", zrel, d, ztf)].add(e)
            cells[(v, scope, "zone", zrel, d, "*")].add(e)
            cells[(v, scope, "zone", zrel, "*", "*")].add(e)
    out = {"trace": trace, "episodes_ordinal1": n, "cells": {"|".join(k): c.row() for k, c in sorted(cells.items())}}
    json.dump(out, open(OUTF, "w"), indent=1)
    print("Q5 — pullback entry at a 1h/4h zone: D-with & 4h-against (hold-trade; R23 approach label = D-against/4h-with), per year, per side")
    for v in VARIANTS:
      print("\n#################### VARIANT %s ####################" % v)
      for scope in ["era"] + sorted({k[1] for k in cells if k[1] != "era"}):
        print("\n== %s / %s ==" % (v, scope))
        for zrel in ("with", "against", "unknown"):
            r = cells.get((v, scope, "zone", zrel, "*", "*"))
            if r: print("  %-7s zone, all sides: %s" % (zrel, fmt(r.row())))
            for d in ("long", "short"):
                r = cells.get((v, scope, "zone", zrel, d, "*"))
                if r: print("     %-7s zone %-5s: %s" % (zrel, d, fmt(r.row())))
                for tf in ("1h", "4h"):
                    r = cells.get((v, scope, "zone", zrel, d, tf))
                    if r: print("        %-7s zone %-5s @%s: %s" % (zrel, d, tf, fmt(r.row())))
        if v == "all":
          for mix in ("D-with/4h-against", "D-against/4h-with"):
            for d in ("long", "short"):
                r = cells.get(("all", scope, "ref", mix, d, "*"))
                if r: print("  ref unrestricted %s %-5s: %s" % (mix, d, fmt(r.row())))
    print("\ndone episodes=%d output=%s" % (n, OUTF))

if __name__ == "__main__":
    main()
