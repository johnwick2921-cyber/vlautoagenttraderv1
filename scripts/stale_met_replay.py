#!/usr/bin/env python3
"""E2 (S-wave, register S2) — stale-MET replay: count how many of the week's
stored MET confirms the NEW rule (|price−ref| > 2.0×ATR5m) marks stale.

Two ATR5m sources, both quoted honestly:
  1. structure_json["5m"].atr — stored at decision time (partial coverage:
     rows where the structure snapshot existed).
  2. bars-table Wilder ATR14 on 5m aggregates (08-24 14:21 CT onward — the
     table's coverage start).

Run: python3 scripts/stale_met_replay.py [db]
"""
import json
import re
import sqlite3
import sys
from datetime import datetime, timezone

DB = sys.argv[1] if len(sys.argv) > 1 else "data/data.db"
db = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
db.row_factory = sqlite3.Row

def parse_ts(ts: str):
    # decision_records.timestamp: "YYYY-MM-DD HH:MM:SS" or with fraction+tz.
    t = ts.replace(" ", "T", 1)
    return datetime.fromisoformat(t)

PRICE_RE = re.compile(r"KEY LEVELS \(map, nearest-first; price ([\d.]+)\)")
PRICE_RE2 = re.compile(r"bias_ctx: price ([\d.]+)")
CONFIRM_RE = re.compile(r"(\S+) confirm: [^\n]*?([\d.]+) — (MET|NOT MET)")

def atr_from_structure(struct: str) -> float:
    if not struct:
        return 0.0
    try:
        return float((json.loads(struct).get("5m") or {}).get("atr") or 0)
    except Exception:
        return 0.0

# --- ATR5m series from the bars table (Wilder-14 on 5m aggregates) ---------
bars = db.execute("SELECT open_time_ms,o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
five = {}
for oms, o, h, l, c in bars:
    k = oms // 300_000
    d = five.setdefault(k, [o, h, l, c])
    if o < d[0]:
        d[0] = o
    if h > d[1]:
        d[1] = h
    if l < d[2]:
        d[2] = l
    d[3] = c
ks = sorted(five)
atr_by_key = {}
if len(ks) >= 14:
    trs = []
    for i, k in enumerate(ks):
        o, h, l, c = five[k]
        if i == 0:
            trs.append(h - l)
        else:
            po, ph, pl, pc = five[ks[i - 1]]
            trs.append(max(h - l, abs(h - pc), abs(l - pc)))
    a = sum(trs[:14]) / 14
    atr_by_key[ks[13]] = a
    for i in range(14, len(trs)):
        a = (a * 13 + trs[i]) / 14
        atr_by_key[ks[i]] = a

def bars_atr5m(ts_ms: int) -> float:
    key = ts_ms // 300_000
    avail = [k for k in atr_by_key if k <= key]
    return atr_by_key[avail[-1]] if avail else 0.0

# --- replay ---------------------------------------------------------------
week_total = week_stale = 0
struct_total = struct_stale = 0
bars_total = bars_stale = 0

rows = db.execute("SELECT timestamp, system_prompt, structure_json FROM decision_records WHERE timestamp >= '2026-08-20 00:00' AND system_prompt LIKE '%confirm:%'").fetchall()
for r in rows:
    m = PRICE_RE.search(r["system_prompt"]) or PRICE_RE2.search(r["system_prompt"])
    if not m:
        continue
    price = float(m.group(1))
    a_struct = atr_from_structure(r["structure_json"])
    ts_ms = int(parse_ts(r["timestamp"]).timestamp() * 1000)
    a_bars = bars_atr5m(ts_ms)
    for mm in CONFIRM_RE.finditer(r["system_prompt"]):
        if mm.group(3) != "MET":
            continue
        week_total += 1
        ref = float(mm.group(2))
        if a_struct > 0:
            struct_total += 1
            if abs(price - ref) > 2.0 * a_struct:
                struct_stale += 1
        if a_bars > 0:
            bars_total += 1
            if abs(price - ref) > 2.0 * a_bars:
                bars_stale += 1

print(f"week METs (all): {week_total}")
print(f"structure-covered: n={struct_total} stale@2.0×ATR5m={struct_stale} ({100*struct_stale/max(struct_total,1):.0f}%)")
print(f"bars-covered (08-24 14:21 CT→now): n={bars_total} stale@2.0×ATR5m={bars_stale} ({100*bars_stale/max(bars_total,1):.0f}%)")
db.close()
