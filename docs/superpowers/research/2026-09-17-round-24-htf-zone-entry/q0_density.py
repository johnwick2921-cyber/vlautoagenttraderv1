#!/usr/bin/env python3
"""Q0 — HTF ZONE DENSITY per read (the headline of the round): how many 1h/4h/1d (and 15m)
zones exist in a read's raw universe, by TF and kind; how much of the read's +-1.5*ATR-ish
neighbourhood they cover is left to Q1's membership counts. Input: out-r24/zones.jsonl."""
import os, sys, json, collections, statistics
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import *
OUTF = os.path.join(OUT, "q0_density.json")
per_read = collections.defaultdict(collections.Counter)   # (day,session) -> Counter[(tf,kind)]
reads = set(); years = collections.defaultdict(set)
for l in open(ZONES):
    d = json.loads(l); k = (d["day"], d["session"]); reads.add(k); years[d["day"][:4]].add(k)
    per_read[k][(d["tf"], d["kind"])] += 1; per_read[k][(d["tf"], "*")] += 1
    if d["tf"] in ZONE_TFS: per_read[k][("HTF", "*")] += 1
# reads with zero zones at all are not in zones.jsonl; count them from era-trends (3,093 reads)
all_reads = {(json.loads(l)["day"], json.loads(l)["session"]) for l in open(ERA_TRENDS)}
rows = {}
keys = sorted({k for c in per_read.values() for k in c})
for key in keys:
    vals = [per_read[r].get(key, 0) for r in all_reads]
    nz = [v for v in vals if v > 0]
    rows["%s|%s" % key] = {"reads_total": len(all_reads), "reads_with": len(nz),
                            "mean_per_read": round(statistics.mean(vals), 2), "median_per_read": statistics.median(vals),
                            "p90_per_read": sorted(vals)[int(0.9 * len(vals))], "max_per_read": max(vals), "total": sum(vals)}
byyear = {}
for y, rs in sorted(years.items()):
    v = [per_read[r].get(("HTF", "*"), 0) for r in rs]
    byyear[y] = {"reads": len(rs), "mean_htf_zones_per_read": round(statistics.mean(v), 1), "median": statistics.median(v)}
out = {"trace": {"script": os.path.abspath(__file__), "commit": git_sha(), "inputs": {ZONES: sha256_file(ZONES), ERA_TRENDS: sha256_file(ERA_TRENDS)}, "output": OUTF,
                 "definition": "zones.jsonl = every level of kind OB/FVG/IFVG/SUPPLY/DEMAND/EQH/EQL with tf in 15m/1h/4h/1d in the read's RAW universe (kernel.AssembleResearchLevels raw return at the read, >=30 min before the window)"},
       "reads_total": len(all_reads), "per_tf_kind": rows, "htf_per_year": byyear}
json.dump(out, open(OUTF, "w"), indent=1)
print("Q0 — HTF zone density per read (reads=%d)" % len(all_reads))
print("%-5s %-7s %8s %8s %8s %6s %8s" % ("tf", "kind", "reads>0", "mean", "median", "p90", "total"))
for k, r in rows.items():
    tf, kind = k.split("|")
    print("%-5s %-7s %8d %8.2f %8.1f %6d %8d" % (tf, kind, r["reads_with"], r["mean_per_read"], r["median_per_read"], r["p90_per_read"], r["total"]))
print("HTF (1h+4h+1d) zones per read by year:", byyear)
print("done output=%s" % OUTF)
