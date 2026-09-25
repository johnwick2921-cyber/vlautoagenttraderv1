#!/usr/bin/env python3
"""Render BACKTEST 1B geometry tables as markdown from run/*.json."""
import json, sys

D = sys.argv[1] if len(sys.argv) > 1 else "data/run"
g = json.load(open(f"{D}/geometry_cells.json"))
ref = json.load(open(f"{D}/reference_cell.json"))
surf = json.load(open(f"{D}/surface_summary.json"))

print("## The 16-cell geometry surface (seated zones, fill (a) headline)\n")
print("| buffer | minR | touches | exp A net | PF A | win A | stop/target/flat | fillB rate / exp | fillC exp | R≥2 share / gated exp A |")
print("|---|---|---|---|---|---|---|---|---|---|---|")
for c in g["all"]:
    fa = c["fill_a"]
    print(f"| {c['buffer']} | {c['min_r']} | {c['n_touches']} | {fa['exp_pts_net']:+.2f} | {fa['pf_net']:.2f} | {fa['win_net']*100:.0f}% | "
          f"{fa['stop_share']*100:.0f}/{fa['target_share']*100:.0f}/{fa['flat_share']*100:.0f} | "
          f"{c['fill_b_rate_vs_a']*100:.0f}% / {c['fill_b']['exp_pts_net']:+.2f} | {c['fill_c']['exp_pts_net']:+.2f} | "
          f"{c['r2_share']*100:.0f}% / {c['r2_gated_exp_a']:+.2f} |")

print("\n## Era split (fill A, net pts/trade)\n")
print("| buffer | minR | in-sample exp / PF | held-out exp / PF |")
print("|---|---|---|---|")
for a, o in zip(g["in_sample"], g["held_out"]):
    print(f"| {a['buffer']} | {a['min_r']} | {a['fill_a']['exp_pts_net']:+.2f} / {a['fill_a']['pf_net']:.2f} | {o['fill_a']['exp_pts_net']:+.2f} / {o['fill_a']['pf_net']:.2f} |")

print("\n## Reference cell — backtest-1 geometry on this tape\n")
print("| era | touches | hold | fillA exp / PF | fillB rate / exp | fillC exp |")
print("|---|---|---|---|---|---|")
for era, r in ref.items():
    print(f"| {era} | {r['n_touches']} | {r['hold_p']*100:.1f}% | {r['fill_a']['exp_pts_net']:+.2f} / {r['fill_a']['pf_net']:.2f} | "
          f"{r['fill_b_rate_vs_a']*100:.0f}% / {r['fill_b']['exp_pts_net']:+.2f} | {r['fill_c']['exp_pts_net']:+.2f} |")

print("\n## Surface summary\n")
s = surf["summary"]
print(f"cells with data={s['cells_with_data']} · net-positive={s['cells_net_positive']} · median={s['median_mean']:+.3f} · min={s['min_mean']:+.3f} · max={s['max_mean']:+.3f} · best p_maxT={s['best_p_maxT']:.4f}")
print(f"correction: {surf['correction']}; perms={surf['perms']}")

