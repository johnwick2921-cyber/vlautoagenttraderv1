#!/usr/bin/env python3
"""Q3 — FIRST TOUCH vs RE-ENTRY with the RE-ENTRY grader (kernel/levels_fresh_by_tf.go
@ dev 324927ad, ported verbatim below), on OWN-TF bars, per TF, per year.

Population: Round 23 out-s4/qa.jsonl — every HTF zone level-scan's ORDINAL-1 D1' episode
(kind, tf, price, lo, hi, origin_ms, anchor_ms=episode open, entry, outcome). 28,117 rows.
Own-TF bars: data/db.copy.db (read-only URI), series (contract-of-the-read, tf) with the
store's reader exclusion source NOT IN ('mixed','replay:off-scale'); contract of the read
from out-s4/trends.jsonl (day, session) -> contract, exactly as the harness keyed bd.byKey.

Grader (verbatim port): formation bars count nothing until the first own-TF bar that
CLOSES fully outside [lo,hi] after origin; thereafter each bar trading into the band
while the previous bar was outside = ONE test; consecutive in-band bars = one visit.
LOOKAHEAD: only own-TF bars that CLOSED at or before the anchor (open+tf <= anchor) are
evidence — stricter than the kernel's OpenTime<=now, which admits the forming bar.
Coverage: a zone whose own-TF series holds ZERO closed bars in [origin, anchor] is
NOT MEASURED (the S4c label-mismatch shape), never 'fresh'.
"""
import bisect, json, os, sqlite3, sys, collections
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import *

QA = os.path.join(R23, "qa.jsonl")
TR = os.path.join(R23, "trends.jsonl")
DB = "/home/hoang/nofx-r101/data/db.copy.db"
TF_MS = {"1h": 3_600_000, "4h": 14_400_000, "1d": 86_400_000}
OUTF = os.path.join(OUT, "q3_cells.json")
os.makedirs(OUT, exist_ok=True)
log = open(os.path.join(OUT, "q3.progress"), "a")

def main():
    trace = {"script": os.path.abspath(__file__), "commit": git_sha(),
             "inputs": {QA: sha256_file(QA), TR: sha256_file(TR), DB: sha256_file(DB)},
             "output": OUTF,
             "lookahead": "own-TF bars with open+tf <= anchor (closed strictly before the touch); origin <= open; grader port of kernel/levels_fresh_by_tf.go@324927ad",
             "definitions": {
                 "test": "re-entry: counted only after the first own-TF CLOSE fully outside [lo,hi]; consecutive in-band bars = one visit",
                 "grade": "fresh=0, tested-1=1, tested-2=2, stale>=3 (S2 display vocabulary)",
                 "hold": "D1' outcome=='hold' (ambiguous excluded from n, counted separately)",
                 "series": "(contract of the read from trends.jsonl, tf) from db.copy.db, source NOT IN ('mixed','replay:off-scale')",
                 "not_measured": "zero closed own-TF bars in [origin, anchor] -> excluded and counted",
             }}
    contract_of = {}
    for l in open(TR):
        t = json.loads(l); contract_of[(t["day"], t["session"])] = t["contract"]
    con = sqlite3.connect("file:%s?mode=ro" % DB, uri=True)
    series = {}
    def bars(contract, tf):
        k = (contract, tf)
        if k not in series:
            rows = con.execute("SELECT open_time_ms, o, h, l, c FROM bars WHERE symbol='MNQ' AND tf=? AND contract=? "
                               "AND COALESCE(source,'') NOT IN ('mixed','replay:off-scale') ORDER BY open_time_ms", (tf, contract)).fetchall()
            series[k] = ([r[0] for r in rows], rows)
        return series[k]

    cells = collections.defaultdict(Cell)     # (tf, year|'era', grade)
    cov = collections.Counter()               # (tf, year) -> measured / not_measured
    n_rows = 0
    for line in open(QA):
        e = json.loads(line); n_rows += 1
        tf = e["tf"]
        if tf not in TF_MS:
            continue
        contract = contract_of.get((e["day"], e["session"]))
        if not contract:
            cov[(tf, year_of(e), "no_contract")] += 1; continue
        opens, rows = bars(contract, tf)
        origin, anchor = e["origin_ms"], e["anchor_ms"]
        lo, hi = min(e["lo"], e["hi"]), max(e["lo"], e["hi"])
        if not e.get("has_origin", True):
            origin = 0
        i0 = bisect.bisect_left(opens, origin)
        # closed at or before the anchor: open + tf <= anchor
        i1 = bisect.bisect_right(opens, anchor - TF_MS[tf])
        if i1 <= i0:
            cov[(tf, year_of(e), "not_measured")] += 1
            continue
        tests = 0; left = False; prev_out = True
        for k in range(i0, i1):
            _, o, h, l, c = rows[k]
            in_band = l <= hi and h >= lo
            if not left:
                if c < lo or c > hi:
                    left = True; prev_out = True
                continue
            if in_band and prev_out:
                tests += 1; prev_out = False
            elif not in_band:
                prev_out = True
        grade = "fresh" if tests == 0 else "tested-1" if tests == 1 else "tested-2" if tests == 2 else "stale"
        cov[(tf, year_of(e), "measured")] += 1
        e2 = dict(e); e2["ordinal"] = 1
        for y in (year_of(e), "era"):
            cells[(tf, y, grade)].add(e2)
            cells[(tf, y, "tested-2+" if tests >= 2 else grade)].add(e2) if tests >= 2 else None
        if n_rows % 5000 == 0:
            log.write("rows %d\n" % n_rows); log.flush()

    out = {"trace": trace, "rows_read": n_rows,
           "coverage": {"%s|%s|%s" % k: v for k, v in sorted(cov.items())},
           "cells": {"%s|%s|%s" % k: c.row() for k, c in sorted(cells.items())}}
    json.dump(out, open(OUTF, "w"), indent=1)
    # human table
    print("Q3 — re-entry grader, own-TF bars, hold rate per TF x year x grade (p_null=%.4f)" % P_NULL)
    for tf in ("1h", "4h", "1d"):
        print("\n== %s ==" % tf)
        for y in sorted({k[1] for k in cells if k[0] == tf}):
            for g in ("fresh", "tested-1", "tested-2+", "stale"):
                r = cells.get((tf, y, g))
                if r: print("  %s %-5s %-9s %s" % (tf, y, g, fmt(r.row())))
            nm = cov.get((tf, y, "not_measured"), 0); m = cov.get((tf, y, "measured"), 0)
            print("  %s %-5s coverage measured=%d not_measured=%d" % (tf, y, m, nm))
    print("\ndone rows=%d output=%s" % (n_rows, OUTF))

if __name__ == "__main__":
    main()
