#!/usr/bin/env python3
"""T3 E-proof — missed-turns % BEFORE/AFTER swing seats.

Baseline (master recheck, 2026-08-27): 78-86% of 5m fractal swing turns had no
seated level within ±8pt across the last 3 sessions. This script recomputes the
baseline from stored 1m bars vs each session's ACTUAL plan levels, then re-runs
the same measure with the T3 swing levels (SWG-H/SWG-L, 5m+15m, the detector's
own fractal) ADDED to the map (capped at 8 seats like the live assembler).

READ-ONLY. Usage: python3 scripts/leveltruth_missed_turns.py [db_path]
"""
import json
import math
import sqlite3
import sys
from collections import defaultdict
from datetime import datetime, timezone, timedelta

CT = timezone(timedelta(hours=-5))
DB_PATH = sys.argv[1] if len(sys.argv) > 1 else "/home/hoang/nofx/data/data.db"
BAND = 8.0
K = 2  # structureSwingK


def ts(y, mo, d, hh, mm=0):
    return int(datetime(y, mo, d, hh, mm, tzinfo=CT).timestamp() * 1000)


def m1(lo, hi):
    return db.execute(
        "SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' "
        "AND open_time_ms>=? AND open_time_ms<? ORDER BY open_time_ms", (lo, hi)
    ).fetchall()


def m5(lo, hi):
    b = m1(lo, hi)
    out = []
    for i in range(0, len(b) - 4, 5):
        w = b[i:i + 5]
        out.append((w[0][0], max(x[2] for x in w), min(x[3] for x in w)))
    return out


def swings(b5):
    t = []
    for i in range(K, len(b5) - K):
        hi = max(x[1] for x in b5[i - K:i + K + 1])
        lo = min(x[2] for x in b5[i - K:i + K + 1])
        if b5[i][1] >= hi:
            t.append(('H', b5[i][1]))
        if b5[i][2] <= lo:
            t.append(('L', b5[i][2]))
    return t


def plan_levels(ds, sess):
    rows = db.execute(
        "SELECT doc FROM plans WHERE plan_id LIKE ? ORDER BY version DESC LIMIT 1",
        (f"{ds}:{sess}:8d5c8af5%",),
    ).fetchall()
    if not rows:
        return []
    doc = json.loads(rows[0][0])
    return [l["price"] for l in doc.get("levels", [])]


def missed(b5, seated):
    n = 0
    for _, px in swings(b5):
        if not any(abs(px - s) <= BAND for s in seated):
            n += 1
    return n


def windows(ds, sess):
    d = datetime.strptime(ds, "%Y-%m-%d")
    if sess == "LONDON":
        return ts(d.year, d.month, d.day, 2), ts(d.year, d.month, d.day, 8, 30)
    if sess == "NY":
        return ts(d.year, d.month, d.day, 8, 30), ts(d.year, d.month, d.day, 16)
    return ts(d.year, d.month, d.day, 17), ts(d.year, d.month, d.day, 23, 59)


db = sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)
sessions = [("2026-08-27", "LONDON"), ("2026-08-26", "NY"), ("2026-08-26", "LONDON")]
print(f"band=±{BAND:.0f}pt · k={K} · 8-seat cap (swings added, weakest proximity seats dropped)")
tot_b, tot_a = 0, 0
for ds, sess in sessions:
    lo, hi = windows(ds, sess)
    b5 = m5(lo, hi)
    sw = swings(b5)
    seated = plan_levels(ds, sess)
    base = missed(b5, seated)
    # after: add every swing price as a candidate seat, keep 8 nearest to the
    # session's last price (proxy for the assembler's proximity seating).
    last_px = b5[-1][1] if b5 else 0
    cands = sorted(set(seated) | {px for _, px in sw}, key=lambda p: abs(p - last_px))[:8]
    after = missed(b5, cands)
    tot_b += base
    tot_a += after
    print(f"{ds} {sess}: swings={len(sw)} baseline missed={base} ({100*base/max(1,len(sw)):.1f}%) → with swing seats={after} ({100*after/max(1,len(sw)):.1f}%)")
print(f"TOTAL: baseline {tot_b} missed turns → {tot_a} with swing seats")
