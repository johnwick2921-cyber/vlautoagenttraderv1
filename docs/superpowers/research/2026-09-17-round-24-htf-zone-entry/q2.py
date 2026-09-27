#!/usr/bin/env python3
"""Q2 — KIND x TF x SIDE, per year: Q1's WITH-ZONE cells by zone kind x zone TF x hold-trade
side. Flags cells that hold >= p_null + 2pt with n >= 2,000 in EVERY year the cell is
measurable (n>=1 that year). Definitions shared: r24lib.DEFINITIONS."""
import os, sys, json, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import *

OUTF = os.path.join(OUT, "q2_cells.json")
log = open(os.path.join(OUT, "q2.progress"), "a")
THRESH = P_NULL + 0.02; MIN_N = 2000

def main():
    wait_for(SENTINEL, log)
    trace = {"script": os.path.abspath(__file__), "commit": git_sha(),
             "inputs": {EPISODES: sha256_file(EPISODES), ZONES: sha256_file(ZONES)},
             "output": OUTF, "definitions": DEFINITIONS,
             "criterion": "with-zone cell (kind, tf, side): rate >= p_null+0.02 AND n >= 2000 in every year with n>=1"}
    zones = load_zones()
    cells = collections.defaultdict(Cell); n = 0
    for e in iter_episodes(True):
        n += 1
        if n % 500000 == 0:
            log.write("episodes %d\n" % n); log.flush()
        z = zones.get((e["day"], e["session"]))
        if not z: continue
        pl = place(e, z)
        if pl is None: continue
        member, zrel, zkind, ztf, conflict, own = pl
        d = hold_trade(e)
        for v in variants_of(conflict, own):
            for scope in ("era", year_of(e)):
                cells[(v, scope, zrel, zkind, ztf, d)].add(e)
    rows = {"|".join(k): c.row() for k, c in cells.items()}
    # criterion
    passing = []; verdicts = {}
    print("Q2 — with-zone cells by kind x TF x side (hold-trade), per year (criterion: rate>=%.4f and n>=%d every measurable year)" % (THRESH, MIN_N))
    for v in VARIANTS:
        keys = {(k[3], k[4], k[5]) for k in cells if k[0] == v and k[2] == "with" and k[1] != "era"}
        print("\n#################### VARIANT %s ####################" % v)
        for kind, tf, d in sorted(keys):
            years = sorted({k[1] for k in cells if k[0] == v and k[2] == "with" and k[3:] == (kind, tf, d) and k[1] != "era"})
            ok = True; detail = []
            for y in years:
                r = cells[(v, y, "with", kind, tf, d)].row()
                good = r["n"] >= MIN_N and r["rate"] >= THRESH
                ok = ok and good
                detail.append("%s:%s%s" % (y, fmt(r), "" if good else " FAIL"))
            verdicts["%s|%s|%s|%s" % (v, kind, tf, d)] = {"passes_every_year": ok, "years": detail}
            if ok: passing.append((v, kind, tf, d))
            print("\n== %s / %s %s %s ==" % (v, kind, tf, d))
            for line in detail: print("   ", line)
            r = cells.get((v, "era", "with", kind, tf, d))
            if r: print("    era:", fmt(r.row()))
            ra = cells.get((v, "era", "against", kind, tf, d))
            if ra: print("    era against-zone (contrast):", fmt(ra.row()))
            print("    PASSES EVERY YEAR:", ok)
    out = {"trace": trace, "episodes_ordinal1": n, "cells": dict(sorted(rows.items())), "verdicts": verdicts, "passing": passing}
    json.dump(out, open(OUTF, "w"), indent=1)
    print("\nPASSING CELLS:", passing if passing else "NONE")
    print("done episodes=%d output=%s" % (n, OUTF))

if __name__ == "__main__":
    main()
