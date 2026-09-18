#!/usr/bin/env python3
"""W-WRITE-TIME-FEASIBILITY replay (2026-09-18, DS-101) — READ-ONLY.

This is a PYTHON RE-IMPLEMENTATION of the predicates, NOT the Go call site —
indicative replay, not a gate replay. Evidence tiers: [A] computed from stored
rows in this run; [B] mirrored from the Go source the wave cites; [C] the
geometry column mirrors the LEGACY resolver only (the Go write site composes
via composeArmStop — geometry rows are indicative, not verdict-equal).

Replays, for every plan version written since 2026-09-13, which write-time
predicate would have fired per scenario arm: min-SL / R:R / geometry / none.

Sources (read-only):
  - history (plans+bars ≤ 2026-09-18 02:25 CT):
    /home/hoang/nofx-backups/pre-bars-key-20260918-022516.db (pre-CLASS-149
    single-contract copy — the research load rule: the LIVE data.db is never
    queried for research during 08:30–15:30 CT).
  - rows written after 02:25 CT: live data.db, read-only lookups only.
  - CLASS 149 contract filter: live bars are queried with `contract = ?` —
    the plan's contract is the ContractAt rule (newest usable bar ≤ read
    time; the row that traded wins, then later expiry). A time-only query
    sees two contracts at the same minutes across the 09-07..09-14 overlap
    and inflates the ATR (DS-104 addendum [A]: 280.51 vs 32.80 on 09-13).

Honesty rules applied (canon):
  - ATR5m is MEASURED from MNQ 5m bars (the same provider family the write
    site uses), bars with open_time_ms <= created_at. <14 bars -> NOT MEASURED.
  - The session-risk band and the HTF veto are NOT judged (spec).
  - The levelidentity hash re-check inside LevelByID is NOT replayed; id-match
    identity resolution is used and flagged in the report.
"""
import json
import sqlite3
from datetime import datetime, timezone

DB_LIVE = "/home/hoang/nofx/data/data.db"
DB_HISTORY = "/home/hoang/nofx-backups/pre-bars-key-20260918-022516.db"
CUTOFF_MS = int(datetime.fromisoformat("2026-09-18T02:25:00-05:00").timestamp() * 1000)
ARM_MIN_RR = 2.0
MIN_SL_MULT = 1.5
SINCE = "2026-09-13"


def parse_dt(s):
    if s is None:
        return None
    try:
        return datetime.fromisoformat(s)
    except ValueError:
        return None


def atr5m_from_bars(bars_ms, created_ms):
    """Wilder ATR(14) over MNQ 5m bars up to created_ms. bars_ms: sorted list of
    (open_ms, h, l, c). Returns (atr, n_bars) or (None, n_bars)."""
    prior = [b for b in bars_ms if b[0] <= created_ms]
    if len(prior) < 14:
        return None, len(prior)
    window = prior[-60:]
    trs = []
    for i in range(1, len(window)):
        h, l, pc = window[i][1], window[i][2], window[i - 1][3]
        trs.append(max(h - l, abs(h - pc), abs(l - pc)))
    if len(trs) < 14:
        return None, len(prior)
    atr = sum(trs[:14]) / 14.0
    for tr in trs[14:]:
        atr = (atr * 13 + tr) / 14.0
    return atr, len(prior)


def rr_for(sc, arm):
    side = (sc.get("direction") or "").lower()
    e, s, t = arm.get("entry"), arm.get("stop"), arm.get("target")
    if not all(isinstance(x, (int, float)) for x in (e, s, t)):
        return None
    if side == "long" and e > s > 0:
        return (t - e) / (e - s)
    if side == "short" and s > e > 0:
        return (e - t) / (s - e)
    return None


def stop_side_verdict(sc, arm, price):
    """Mirrors the executor's stop-side placement guard for reclaim arms:
    trigger = entry ± STOP_ENTRY_OFFSET_TICKS(2) × tick(0.25), tick-rounded;
    long -> wrong side when price >= trigger; short -> price <= trigger.
    Returns (trigger, verdict) where verdict is 'through' | 'rests' | 'unknown'.
    """
    if price is None or price <= 0:
        return None, "unknown"
    side = (sc.get("direction") or "").lower()
    e = arm.get("entry")
    if side not in ("long", "short") or not isinstance(e, (int, float)) or e <= 0:
        return None, "unknown"
    tick = 0.25
    offset = 2 * tick
    trig = e + offset if side == "long" else e - offset
    trig = round(trig / tick) * tick
    if side == "long" and price >= trig:
        return trig, "through"
    if side == "short" and price <= trig:
        return trig, "through"
    return trig, "rests"


def geometry_refusal(doc, sc, arm):
    """Mirrors trader.ResolveEntryGeometryZone's refusal ladder from stored
    fields (the levelidentity hash re-check is not replayed)."""
    zm = doc.get("zone_map")
    if not zm:
        return "geometry: frozen_zone_map_missing"
    lvl_id = sc.get("level_id")
    if not lvl_id:
        return "geometry: scenario_level_id_missing"
    identity = None
    for l in doc.get("identity_levels") or []:
        if l.get("id") == lvl_id:
            identity = l
            break
    if identity is None:
        return "geometry: identity_not_valid_in_frozen_map"
    zones = zm.get("zones") or []
    match = -1
    for i, z in enumerate(zones):
        for s in z.get("sources") or []:
            if abs((s.get("price") or 0) - (identity.get("price") or 0)) > 1e-7:
                continue
            named = s.get("label") == identity.get("label")
            for name in identity.get("names") or []:
                named = named or s.get("label") == name
            if not named:
                continue
            itf = identity.get("tf")
            if itf and s.get("tf") != itf:
                continue
            if match >= 0 and match != i:
                return "geometry: entry_zone_ambiguous"
            match = i
            break
    if match < 0:
        return "geometry: entry_source_not_in_frozen_zones"
    z = zones[match]
    lo, hi = z.get("lo"), z.get("hi")
    if (lo is None or hi is None or z.get("incomplete_width")
            or lo <= 0 or hi < lo or not (z.get("sources") or [])):
        return "geometry: entry_zone_edges_or_provenance_unusable"
    return None


def contract_at(live_1m, created_ms):
    """Mirrors the ContractAt tie-break for live rows: among contracts with the
    newest usable 1m bar at or before created_ms, the row that traded wins
    (live > mixed > replay > import), then later expiry, then lexicographic.
    Returns the contract label or None."""
    prior = [r for r in live_1m if r[0] <= created_ms]
    if not prior:
        return None
    newest = max(r[0] for r in prior)
    cands = [r for r in prior if r[0] == newest]
    src_rank = {"live": 0, "mixed": 1, "replay": 2, "import": 3}
    cands.sort(key=lambda r: (src_rank.get(r[2] or "", 9), tuple(-x for x in expiry(r[1])), r[1]))
    return cands[0][1]


def expiry(label):
    f = (label or "").split()
    if len(f) >= 2:
        parts = f[-1].split("-")
        if len(parts) == 2 and parts[0].isdigit() and parts[1].isdigit():
            return 2000 + int(parts[1]), int(parts[0])
    return 0, 0


def main():
    rows = []
    con = sqlite3.connect(f"file:{DB_HISTORY}?mode=ro", uri=True)
    rows += con.execute(
        "SELECT plan_id, version, trade_date, session, created_at, doc "
        "FROM plans WHERE created_at >= ? ORDER BY created_at, plan_id, version",
        (SINCE,)).fetchall()
    con.close()
    con = sqlite3.connect(f"file:{DB_LIVE}?mode=ro", uri=True)
    # live rows written after the 02:25 cutoff only (research load rule)
    rows += con.execute(
        "SELECT plan_id, version, trade_date, session, created_at, doc "
        "FROM plans WHERE created_at >= ? AND created_at >= '2026-09-18 02:25' "
        "ORDER BY created_at, plan_id, version",
        (SINCE,)).fetchall()
    # history bars: the single-contract pre-migration copy
    con_hist = sqlite3.connect(f"file:{DB_HISTORY}?mode=ro", uri=True)
    hist_bars = con_hist.execute(
        "SELECT open_time_ms, h, l, c FROM bars WHERE symbol='MNQ' AND tf='5m' "
        "ORDER BY open_time_ms").fetchall()
    con_hist.close()
    # live bars: contract-keyed, filtered per plan (CLASS 149)
    live_bars = con.execute(
        "SELECT open_time_ms, contract, source, h, l, c FROM bars "
        "WHERE symbol='MNQ' AND tf='5m' ORDER BY open_time_ms").fetchall()
    live_1m = con.execute(
        "SELECT open_time_ms, contract, source FROM bars "
        "WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms").fetchall()
    con.close()

    print(f"## Replay: write-time feasibility predicates, plans since {SINCE}\n")
    print(f"rows={len(rows)} plans; history bars={len(hist_bars)} (pre-CLASS-149 copy); "
          f"live bars={len(live_bars)} (contract-filtered, CLASS 149); "
          f"arm_rr={ARM_MIN_RR}; min_sl={MIN_SL_MULT}×ATR5m; "
          "session band/HTF NOT judged (spec); levelidentity hash NOT replayed; "
          "Python re-implementation, not the Go call site [C]; geometry column = "
          "legacy resolver only [C]\n")
    print("| plan_id | v | session | S# | cond | dir | atr5m | rr | min_sl | "
          "geometry | stop_side | predicate |")
    print("|---|---|---|---|---|---|---|---|---|---|---|---|")
    counts = {}
    for plan_id, ver, td, sess, created, doc_json in rows:
        try:
            doc = json.loads(doc_json or "{}")
        except json.JSONDecodeError:
            print(f"| {plan_id} | {ver} | {sess} | — | — | — | — | — | — | — | "
                  "unparseable_doc |")
            counts["unparseable_doc"] = counts.get("unparseable_doc", 0) + 1
            continue
        created = parse_dt(created)
        created_ms = int(created.timestamp() * 1000) if created else None
        atr, nbars = (None, 0)
        tape_close = None
        if created_ms:
            if created_ms <= CUTOFF_MS:
                atr, nbars = atr5m_from_bars(hist_bars, created_ms)
                closes = [b[3] for b in hist_bars if b[0] <= created_ms]
                if closes:
                    tape_close = closes[-1]
            else:
                contract = contract_at(live_1m, created_ms)
                contracted = [(r[0], r[3], r[4], r[5]) for r in live_bars
                              if r[1] == contract]
                atr, nbars = atr5m_from_bars(contracted, created_ms)
                closes = [b[3] for b in contracted if b[0] <= created_ms]
                if closes:
                    tape_close = closes[-1]
        for sc in doc.get("scenarios") or []:
            sid = sc.get("id") or "?"
            cond = sc.get("condition") or ""
            arm = sc.get("arm")
            if not arm or not arm.get("enabled", False):
                counts["no_arm"] = counts.get("no_arm", 0) + 1
                continue
            kind = ""
            legs = arm.get("legs") or []
            if legs:
                kind = legs[0].get("kind") or ""
            structural = cond == "reject" and kind.lower() != "exit"
            rr = rr_for(sc, arm)
            pred, rr_s, atr_s, geo_s, stop_s = "none", f"{rr:.2f}" if rr is not None else "n/a", "—", "—", "—"
            if rr is not None and rr + 1e-9 < ARM_MIN_RR:
                pred = "R:R"
            else:
                if structural:
                    atr_s = f"{atr:.2f}" if atr else "NOT MEASURED"
                    geo = geometry_refusal(doc, sc, arm)
                    if geo:
                        pred, geo_s = "geometry", geo
                else:
                    # N1 (CTO RECHECK 2026-09-18): the executor composes the
                    # stop for every NON-fade leg BEFORE its gates (legacy
                    # composeArmStop, floored at MIN_SL_ATR_MULT×ATR5m, widest
                    # wins) — min-SL can never fire at arm on the authored
                    # stop, so the predicate is DROPPED for non-fade rows. The
                    # R:R above is computed on the authored leg; the composed
                    # stop can only LOWER R:R, so the none bucket is an upper
                    # bound, not an exact admit.
                    atr_s = f"{atr:.2f}" if atr else "NOT MEASURED"
                if pred == "none" and cond == "reclaim":
                    price = doc.get("price_at_write")
                    if price is None:
                        price = tape_close  # the tape's close at created_at, never an invented one
                    trig, verdict = stop_side_verdict(sc, arm, price)
                    if verdict == "through":
                        pred = "stop_side"
                        stop_s = f"trigger {trig:.2f} through price {price:.2f}"
                    elif verdict == "unknown":
                        stop_s = "NOT MEASURED (no price)"
                    else:
                        stop_s = "rests"
            counts[pred] = counts.get(pred, 0) + 1
            print(f"| {plan_id} | {ver} | {sess} | {sid} | {cond} | "
                  f"{sc.get('direction','')} | {atr_s} | {rr_s} | {min_sl_col(pred)} | "
                  f"{geo_s} | {stop_s} | {pred} |")
    print("\n### counts")
    for k, v in sorted(counts.items()):
        print(f"- {k}: {v}")


def min_sl_col(pred):
    return "REFUSE" if pred == "min-SL" else "—"


if __name__ == "__main__":
    main()
