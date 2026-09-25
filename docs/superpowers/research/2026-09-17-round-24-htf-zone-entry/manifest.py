#!/usr/bin/env python3
"""Emit the report's MANIFEST + cell tables + sample-id appendix from the JSON traces, so the
report cannot drift from what ran. Usage: python3 manifest.py > out-r24/manifest.md"""
import json, os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from r24lib import OUT, P_NULL
J = lambda n: json.load(open(os.path.join(OUT, n)))
q0, q1, q2, q3, q5 = J("q0_density.json"), J("q1_cells.json"), J("q2_cells.json"), J("q3_cells.json"), J("q5_cells.json")
short = lambda p: p.split("/")[-1]
def cellrow(r):
    if r["n"] == 0: return "NOT MEASURED (n=0)"
    return "%.3f [%.3f,%.3f] · n=%d (h=%d/b=%d, ambig=%d) · lift %+.1f pt" % (r["rate"], r["ci_lo"], r["ci_hi"], r["n"], r["hold"], r["break"], r["ambiguous"], r["lift_pt"])
samples = []
def S(key, r):
    samples.append((key, r.get("first5", [])))

print("## MANIFEST (every cell below is reproducible from this table; anything not here is NOT MEASURED)\n")
print("| Q | script | commit | inputs (sha256[:16]) | output | lookahead |")
print("|---|---|---|---|---|---|")
for name, j in (("Q0", q0), ("Q1", q1), ("Q2", q2), ("Q3", q3), ("Q5", q5)):
    t = j["trace"]
    ins = ", ".join("%s=%s" % (short(k), v[:16]) for k, v in t["inputs"].items())
    la = t.get("lookahead") or t.get("definitions", {}).get("lookahead", "")
    print("| %s | `%s` | `%s` | %s | `%s` | %s |" % (name, short(t["script"]), t["commit"], ins, short(t["output"]), la[:160]))
print()
print("Harness pass: `run_harness.sh` → binary sha256 df59fd66d1ea2ad8 (built from `git archive d15db077` + this branch's `harness/eval.go`, diff vs d15db077 = 120 lines), `out-r24/harness.log` line 1; episodes.jsonl sha256 529b41059b81cbdd (4,471,482 lines = Round 23's count), zones.jsonl sha256 37df53ab742675e2 (556,237 zones).\n")

# Q0
print("## Q0 — HTF zone density per read (reads=%d)\n" % q0["reads_total"])
print("| tf | kind | reads with ≥1 | mean/read | median | p90 | total |")
print("|---|---|---|---|---|---|---|")
for k, r in q0["per_tf_kind"].items():
    tf, kind = k.split("|")
    if tf == "15m" and kind != "*": continue
    print("| %s | %s | %d | %.2f | %.1f | %d | %d |" % (tf, kind, r["reads_with"], r["mean_per_read"], r["median_per_read"], r["p90_per_read"], r["total"]))
print("\nHTF (1h+4h+1d) zones per read by year: " + " · ".join("%s %.1f (median %s, reads %d)" % (y, v["mean_htf_zones_per_read"], v["median"], v["reads"]) for y, v in q0["htf_per_year"].items()) + "\n")
print("Q1 membership counts (ordinal-1 episodes = %d): %s\n" % (q1["episodes_ordinal1"], q1["counts"]))

# Q1
C1 = q1["cells"]
def q1row(v, scope, *rest):
    return C1.get("|".join((v, scope) + rest))
print("## Q1 — ZONE × TREND\n")
for v in ("own", "noconflict", "all"):
    print("### Q1 variant `%s`%s\n" % (v, " (PRIMARY — CTO ruling)" if v == "own" else " (context)"))
    print("| scope | cell | hold rate [Wilson 95%] · n · lift |")
    print("|---|---|---|")
    scopes = ["era"] + sorted({k.split("|")[1] for k in C1 if k.startswith(v + "|") and not k.startswith(v + "|era")})
    for sc in scopes:
        for zrel in ("with", "against", "unknown"):
            r = q1row(v, sc, "zone", zrel, "*", "*")
            if r: print("| %s | (a) %s-zone | %s |" % (sc, zrel, cellrow(r))); S("Q1/%s/%s/(a) %s-zone" % (v, sc, zrel), r)
        for tfl, tag in (("4h", "b"), ("D", "c")):
            for zrel in ("with", "against"):
                for rel in ("with", "against", "range"):
                    r = q1row(v, sc, "zone", zrel, tfl, rel)
                    if r and (sc == "era" or rel != "range"): print("| %s | (%s) %s-zone × %s-%s | %s |" % (sc, tag, zrel, tfl, rel, cellrow(r))); S("Q1/%s/%s/(%s) %s-zone × %s-%s" % (v, sc, tag, zrel, tfl, rel), r)
        for zrel in ("with", "against"):
            for r4 in ("with", "against"):
                for rD in ("with", "against"):
                    r = q1row(v, sc, "cross", zrel, "4h=" + r4, "D=" + rD)
                    if r: print("| %s | 4-way %s-zone × 4h-%s × D-%s | %s |" % (sc, zrel, r4, rD, cellrow(r))); S("Q1/%s/%s/4-way %s-zone × 4h-%s × D-%s" % (v, sc, zrel, r4, rD), r)
        if sc == "era":
            for d in ("long", "short"):
                for zrel in ("with", "against"):
                    r = q1row(v, sc, "side", d, zrel, "*")
                    if r: print("| %s | side %s × %s-zone | %s |" % (sc, d, zrel, cellrow(r))); S("Q1/%s/%s/side %s × %s-zone" % (v, sc, d, zrel), r)
    print()

# Q2
C2 = q2["cells"]
print("## Q2 — KIND × TF × SIDE (with-zone), per year\n")
print("Criterion: %s. **PASSING CELLS: %s**\n" % (q2["trace"]["criterion"], q2["passing"] or "NONE, in any variant"))
for v in ("own", "noconflict", "all"):
    print("### Q2 variant `%s`%s\n" % (v, " (PRIMARY)" if v == "own" else " (context)"))
    print("| kind | tf | side | era with-zone | era against-zone | by year (with-zone) | passes every year |")
    print("|---|---|---|---|---|---|---|")
    for vk, vd in sorted(q2["verdicts"].items()):
        vv, kind, tf, d = vk.split("|")
        if vv != v: continue
        era = C2.get("|".join((v, "era", "with", kind, tf, d))); ag = C2.get("|".join((v, "era", "against", kind, tf, d)))
        if not era or era["n"] < (300 if v == "own" else 2000): continue
        yrs = "; ".join(x.replace(" FAIL", "✗") for x in vd["years"])
        print("| %s | %s | %s | %s | %s | %s | %s |" % (kind, tf, d, cellrow(era), cellrow(ag) if ag else "—", yrs, vd["passes_every_year"]))
        S("Q2/%s/era/with-zone %s %s %s" % (v, kind, tf, d), era)
    print()

# Q3
C3 = q3["cells"]
print("## Q3 — FIRST TOUCH vs RE-ENTRY (re-entry grader on own-TF bars)\n")
print("| tf | year | fresh | tested-1 | tested-2 | stale (≥3) | measured / not measured |")
print("|---|---|---|---|---|---|---|")
for tf in ("1h", "4h", "1d"):
    for y in sorted({k.split("|")[1] for k in C3 if k.startswith(tf + "|")}):
        cells = [C3.get("|".join((tf, y, g))) for g in ("fresh", "tested-1", "tested-2", "stale")]
        cov = q3["coverage"]
        m = sum(v for k, v in cov.items() if k.startswith("%s|%s|measured" % (tf, y))) if y != "era" else sum(v for k, v in cov.items() if k.startswith(tf + "|") and k.endswith("measured") and "not_" not in k)
        nm = sum(v for k, v in cov.items() if k.startswith("%s|%s|not_measured" % (tf, y))) if y != "era" else sum(v for k, v in cov.items() if k.startswith(tf + "|") and k.endswith("not_measured"))
        print("| %s | %s | %s | %s | %s | %s | %d / %d |" % (tf, y, *[cellrow(c) if c else "—" for c in cells], m, nm))
        for g, c in zip(("fresh", "tested-1", "tested-2", "stale"), cells):
            if c: S("Q3/%s/%s/%s" % (tf, y, g), c)
print()

# Q5
C5 = q5["cells"]
print("## Q5 — PULLBACK ENTRY AT A 1h/4h ZONE (D-with & 4h-against, hold-trade)\n")
for v in ("own", "noconflict", "all"):
    print("### Q5 variant `%s`%s\n" % (v, " (PRIMARY)" if v == "own" else " (context)"))
    print("| scope | cell | hold rate · n · lift |")
    print("|---|---|---|")
    scopes = ["era"] + sorted({k.split("|")[1] for k in C5 if k.startswith(v + "|") and "|zone|" in k and not k.startswith(v + "|era")})
    for sc in scopes:
        for zrel in ("with", "against"):
            for d in ("*", "long", "short"):
                for tf in ("*", "1h", "4h"):
                    if d == "*" and tf != "*": continue
                    r = C5.get("|".join((v, sc, "zone", zrel, d, tf)))
                    if r: print("| %s | %s-zone · %s · %s | %s |" % (sc, zrel, "all sides" if d == "*" else d, "1h+4h" if tf == "*" else tf, cellrow(r))); S("Q5/%s/%s/%s-zone %s %s" % (v, sc, zrel, d, tf), r)
    print()
print("### Q5 reference — the unrestricted mixed cells under BOTH labellings (R23 S4d(2) comparability)\n")
print("| scope | R24 hold-trade cell | R23 approach label | long | short |")
print("|---|---|---|---|---|")
for sc in ["era", "2025", "2026"]:
    for mix, r23 in (("D-with/4h-against", "D-against/4h-with"), ("D-against/4h-with", "D-with/4h-against")):
        L = C5.get("|".join(("all", sc, "ref", mix, "long", "*"))); Sh = C5.get("|".join(("all", sc, "ref", mix, "short", "*")))
        if L or Sh: print("| %s | %s | %s | %s | %s |" % (sc, mix, r23, cellrow(L) if L else "—", cellrow(Sh) if Sh else "—"))
print()
print("## APPENDIX — sample ids (first 5 per cell: day session kind tf @price ordN)\n")
for key, ids in samples:
    print("- **%s**: %s" % (key, "; ".join(ids) if ids else "—"))
