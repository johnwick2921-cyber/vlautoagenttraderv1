#!/usr/bin/env python3
"""B1 (planner-contract wave 2026-08-26) — MPM look-ahead fixture.

Proves why every hypothetical stop/target resolution MUST use 1m bars, never
the 2-min decision/confirm bar. Synthetic series: a LONG with TP +10 and SL −10
from entry. On the 1m series price prints TP first (+10) then SL (−10) — a WIN.
Aggregated to 2-min bars, the same move lands in ONE bucket whose H/L ordering
depends on the aggregator's extreme-sequencing convention — here the aggregator
records the extremes in arrival order, so SL (which arrives second intra-bucket
but is collated as the bucket low) is judged FIRST — a LOSS. The 73%→50% edge
degradation from the research is exactly this class of coarsening error.

Run: python3 scripts/mpm_resolution_fixture.py
"""

# 1m series: entry at t=0 price 100.0 (LONG, TP 110, SL 90).
# minutes 0..7: grind to 109.5; minute 8: wick to 110.0 (TP hit); minute 9:
# collapse to 90.0 (SL hit). Minutes 8-9 share ONE 2-min bucket → the trap.
one_min = [
    (100.0, 100.5),   # m0
    (100.5, 101.5),
    (101.5, 103.0),
    (103.0, 104.5),
    (104.5, 106.0),
    (106.0, 107.0),
    (107.0, 108.5),
    (108.5, 109.5),
    (110.0, 109.5),   # m8: wick HIGH 110.0 = TP touched
    (109.0, 90.0),    # m9: SL touched at 90.0 — same 2-min bucket as m8
]
ENTRY, TP, SL = 100.0, 110.0, 90.0

def resolve_1m():
    for i, (h, l) in enumerate(one_min):
        if h >= TP:
            return ("TP", i)
        if l <= SL:
            return ("SL", i)
    return ("none", len(one_min))

def resolve_2min():
    # 2-min buckets; extremes recorded in arrival order within the bucket.
    for b in range(0, len(one_min), 2):
        bucket = one_min[b:b+2]
        hi = max(x[0] for x in bucket)
        lo = min(x[1] for x in bucket)
        # The trap: within a bucket, which came first is unknown; the coarse
        # resolver checks the bucket LOW before the HIGH (SL-before-TP bias)
        # because stop-loss lookups conventionally precede targets.
        if lo <= SL:
            return ("SL", b)
        if hi >= TP:
            return ("TP", b)
    return ("none", len(one_min))

r1, t1 = resolve_1m()
r2, t2 = resolve_2min()
print(f"1m resolution:      {r1} first at minute {t1}")
print(f"2-min resolution:   {r2} first at minute {t2}")
print("MPM look-ahead verdict: the 2-min bar flips a WIN into a LOSS.")
assert r1 == "TP" and r2 == "SL", "fixture must demonstrate the inversion"
print("fixture OK — rule B1 holds: resolve on 1m bars, never the 2-min bar.")
