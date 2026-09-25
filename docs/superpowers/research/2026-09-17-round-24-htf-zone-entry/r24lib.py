"""Round 24 shared helpers — Wilson CI, sha256 (computed once, cached), trace rows.
Every cell a script emits carries: n_hold, n_break, n (hold+break; ambiguous excluded
exactly as Round 23's hold_of), rate, Wilson 95% CI, first-5 episode ids."""
import collections, hashlib, json, math, os, subprocess, time

P_NULL = 0.5067  # D1' IID calibration (Round 23 HANDOVER §1)
R23 = "/home/hoang/nofx-r101/docs/superpowers/research/2026-09-16-round-23/out-s4"
HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out-r24")
_SHA_CACHE = os.path.join(OUT, "input-sha256.json")

def wilson(h, n, z=1.96):
    if n == 0:
        return (0.0, 0.0, 0.0)
    p = h / n
    d = 1 + z * z / n
    c = (p + z * z / (2 * n)) / d
    w = z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / d
    return (p, c - w, c + w)

def sha256_file(path):
    """Computed once per path, cached in out-r24/input-sha256.json (shared across the four)."""
    cache = {}
    if os.path.exists(_SHA_CACHE):
        try:
            cache = json.load(open(_SHA_CACHE))
        except Exception:
            cache = {}
    st = os.stat(path)
    key = os.path.abspath(path)
    ent = cache.get(key)
    if ent and ent.get("size") == st.st_size and ent.get("mtime") == int(st.st_mtime):
        return ent["sha256"]
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 24), b""):
            h.update(chunk)
    cache[key] = {"sha256": h.hexdigest(), "size": st.st_size, "mtime": int(st.st_mtime)}
    os.makedirs(OUT, exist_ok=True)
    tmp = _SHA_CACHE + ".%d" % os.getpid()
    json.dump(cache, open(tmp, "w"), indent=1)
    os.replace(tmp, _SHA_CACHE)
    return cache[key]["sha256"]

def git_sha():
    try:
        return subprocess.check_output(["git", "rev-parse", "--short", "HEAD"], cwd=HERE, text=True).strip()
    except Exception:
        return "unknown"

def ep_id(e):
    """(day, session, kind, tf, price|ordinal) — the sample-id law."""
    return "%s %s %s %s @%s ord%s" % (e.get("day"), e.get("session"), e.get("kind"), e.get("tf"),
                                        e.get("price", e.get("opened_at_ms")), e.get("ordinal", 1))

class Cell:
    __slots__ = ("hold", "brk", "ambig", "ids")
    def __init__(self):
        self.hold = 0; self.brk = 0; self.ambig = 0; self.ids = []
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
            self.ids.append(ep_id(e))
    def row(self):
        n = self.hold + self.brk
        p, lo, hi = wilson(self.hold, n)
        return {"n": n, "hold": self.hold, "break": self.brk, "ambiguous": self.ambig,
                "rate": round(p, 4), "ci_lo": round(lo, 4), "ci_hi": round(hi, 4),
                "lift_pt": round((p - P_NULL) * 100, 1) if n else None, "first5": self.ids}

def fmt(r):
    if r["n"] == 0:
        return "NOT MEASURED (n=0)"
    return "%.3f [%.3f,%.3f] n=%d (h=%d b=%d)" % (r["rate"], r["ci_lo"], r["ci_hi"], r["n"], r["hold"], r["break"])

def wait_for(path, log, every=30):
    t0 = time.time()
    while not os.path.exists(path):
        log.write("waiting for %s (%ds)\n" % (path, int(time.time() - t0))); log.flush()
        time.sleep(every)

def year_of(e):
    return e["day"][:4]

# ── Round 24 classification (ONE definition, shared by q1/q2/q5) ─────────────
ZONE_TFS = ("1h", "4h", "1d")
ZONE_KINDS = ("OB", "FVG", "IFVG", "SUPPLY", "DEMAND", "EQH", "EQL")
EPISODES = os.path.join(OUT, "episodes.jsonl")
ZONES = os.path.join(OUT, "zones.jsonl")
ERA_TRENDS = os.path.join(R23, "era-trends.jsonl")
SENTINEL = os.path.join(OUT, "harness.done")

DEFINITIONS = {
    "population": "Round 24 re-evaluation of the Round 23 S4 pass (same kernel base d15db077, same DB copy, same reads, same D1' instrument); the writer adds price/lo/hi/label/polarity/delta/contract per episode and zones.jsonl = every HTF zone in each read's RAW universe",
    "instrument": "D1' kernel.DetectTouchOutcomes anchored at the level price, k=3, delta=trailing-5-day mean |1m close increment| (per read), H=12x1m, exit_on=close; hold = price exits on the side it came FROM",
    "lookahead": "zones and trend state are the READ snapshot (>=30 min before the session window, closed bars only); an episode is placed only against zones that existed at its read",
    "approach": "D1' entry: 'below' = price came from below (level tested as RESISTANCE), 'above' = price came from above (level tested as SUPPORT)",
    "hold_trade": "the trade a HOLD pays: approach 'above' -> LONG at support; approach 'below' -> SHORT at resistance. NOTE: Round 23's s4_analysis.py direction() labelled approach 'below' as 'long' (the APPROACH direction); Round 24 reports the HOLD-TRADE direction, so R23 'agree' == R24 'against-trend' and vice versa",
    "zone_polarity": "support = DEMAND, EQL, OB(bull), iFVG(bull), FVG formed as a gap UP; resistance = SUPPLY, EQH, OB(bear), iFVG(bear), FVG formed as a gap DOWN; unknown = FVG whose formation candle was not found on its own series (never guessed)",
    "zone_membership": "level price P vs zone [lo,hi] of the SAME read, zone tf in 1h/4h/1d: inside = lo<=P<=hi; near = lo-1*delta<=P<=hi+1*delta (EQH/EQL are lines: lo=hi=price, so 'near' is the +-1*delta band); the CLOSEST zone by |P-midpoint| decides",
    "variants": "all = every episode at a zone (closest zone decides); noconflict = zones of only ONE polarity hold P within the near band (the density finding: 42% of at-zone episodes sit in overlapping zones of both polarities, where the closest-zone rule is geometry); own = the episode's own level IS an HTF zone (kind in zone kinds, tf in 1h/4h/1d): the D1' anchored at the zone midpoint, polarity = its own",
    "with_zone": "hold-trade LONG at a support-polarity zone, or hold-trade SHORT at a resistance-polarity zone; else against-zone; unknown polarity -> 'unknown'",
    "trend": "S4c era-wide S1 state per read (era-trends.jsonl: d_trend, h4_trend in up/down/range); with = hold-trade direction agrees with the trend, against = opposes, range = trend is range; 4h has no state before 2025-05 (n/a)",
    "hold_rate": "hold/(hold+break); ambiguous_* excluded from n and counted; Wilson 95% CI; lift = rate - p_null(0.5067) in pt",
}

def load_zones():
    z = collections.defaultdict(list)
    for l in open(ZONES):
        d = json.loads(l)
        if d["tf"] in ZONE_TFS and d["kind"] in ZONE_KINDS:
            lo, hi = min(d["lo"], d["hi"]), max(d["lo"], d["hi"])
            z[(d["day"], d["session"])].append((lo, hi, (lo + hi) / 2, d.get("polarity", ""), d["kind"], d["tf"]))
    return z

def load_trends():
    t = {}
    for l in open(ERA_TRENDS):
        d = json.loads(l); t[(d["day"], d["session"])] = (d.get("d_trend"), d.get("h4_trend"))
    return t

def hold_trade(e):
    return "long" if e["entry"] == "above" else "short"

def trend_rel(direction, trend):
    if trend == "up":
        return "with" if direction == "long" else "against"
    if trend == "down":
        return "with" if direction == "short" else "against"
    if trend == "range":
        return "range"
    return "n/a"

def place(e, zones):
    """-> (membership, zone_rel, zkind, ztf, conflict, own) or None when not at any HTF zone.
    membership: 'inside' | 'near'; zone_rel: 'with' | 'against' | 'unknown';
    conflict: zones of BOTH polarities hold P within the near band (the closest-zone rule
    is then geometry, not signal); own: the episode's own level IS an HTF zone (the D1'
    anchored at the zone's midpoint — the cleanest 'entry at the zone')."""
    P = e["price"]; delta = e["delta"]; d = hold_trade(e)
    best = None; pols = set()
    for lo, hi, mid, pol, kind, tf in zones:
        if lo - delta <= P <= hi + delta:
            dist = abs(P - mid)
            if best is None or dist < best[0]:
                best = (dist, lo, hi, pol, kind, tf)
            if pol:
                pols.add(pol)
    if best is None:
        return None
    _, lo, hi, pol, kind, tf = best
    member = "inside" if lo <= P <= hi else "near"
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
    return member, rel, kind, tf, len(pols) > 1, own

VARIANTS = ("all", "noconflict", "own")

def variants_of(conflict, own):
    """all = every episode at a zone (closest zone decides); noconflict = only one polarity
    holds P within the near band; own = the episode's level is itself the HTF zone."""
    v = ["all"]
    if not conflict:
        v.append("noconflict")
    if own:
        v.append("own")
    return v

def iter_episodes(ordinal1_only=True):
    for l in open(EPISODES):
        e = json.loads(l)
        if ordinal1_only and e["ordinal"] != 1:
            continue
        yield e
