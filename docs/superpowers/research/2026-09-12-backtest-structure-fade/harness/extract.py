#!/usr/bin/env python3
"""Extract BACKTEST 1B report numbers from a harness run dir."""
import json, sys

D = sys.argv[1] if len(sys.argv) > 1 else "data/run"

def load(name):
    with open(f"{D}/{name}") as f:
        return json.load(f)

s = load("summary.json")
print("== READS ==")
print(f"built={s['reads_built']} skipped={s['reads_skipped']} in={s['reads_in_sample']} out={s['reads_held_out']}")
print(f"events_total={s['events_total']} seated={s['seated_events']}")
h = s["hold_context"]
print("== HOLD CONTEXT ==")
print(f"h12 hold={h['h12_hold_p']:.4f} n={h['h12_n']} amb={h['h12_amb']} wilson={[round(x,4) for x in h['h12_wilson']]}")
print(f"5mH10 hold={h['5m_h10_hold_p']:.4f} n={h['5m_h10_n']} amb={h['5m_h10_amb']}")
ref = s["reference_cell"]
print("== REFERENCE (backtest-1 geometry on this tape) ==")
for era, r in ref.items():
    fa, fb, fc = r["fill_a"], r["fill_b"], r["fill_c"]
    print(f"{era}: touches={r['n_touches']} hold={r['hold_p']:.4f} fillA exp={fa['exp_pts_net']:+.3f} pf={fa['pf_net']:.3f} | fillB rate={r['fill_b_rate_vs_a']:.3f} exp={fb['exp_pts_net']:+.3f} | fillC exp={fc['exp_pts_net']:+.3f}")
print("== GEOMETRY CELLS (all eras, fill A headline) ==")
for c in s["geometry_cells"]:
    fa = c["fill_a"]
    print(f"{c['buffer']:<12} minR={c['min_r']:<4} touches={c['n_touches']:<5} expA={fa['exp_pts_net']:+7.3f} pf={fa['pf_net']:.3f} winA={fa['win_net']:.3f} stop={fa['stop_share']:.2f} target={fa['target_share']:.2f} flat={fa['flat_share']:.2f} | B(rate={c['fill_b_rate_vs_a']:.2f},exp={c['fill_b']['exp_pts_net']:+.3f}) C(exp={c['fill_c']['exp_pts_net']:+.3f}) | R2share={c['r2_share']:.3f} R2exp={c['r2_gated_exp_a']:+.3f}")
ss = load("surface_summary.json")
print("== SURFACE ==")
print(json.dumps(ss["summary"], indent=1))
print("perms", ss["perms"], "correction", ss["correction"])
print("== SEAM ==")
print(load("seam.json")["summary"])

