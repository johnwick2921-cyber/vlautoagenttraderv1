#!/usr/bin/env python3
"""Extract report numbers from a harness run dir (reads run/*.json)."""
import json, sys, collections

D = sys.argv[1] if len(sys.argv) > 1 else "docs/superpowers/research/2026-09-12-backtest-zone-fade/data/run"

def load(name):
    with open(f"{D}/{name}") as f:
        return json.load(f)

def c4(r, indent="  "):
    fa, fb, fc = r["FillA"], r["FillB"], r["FillC"]
    return (f"{indent}n_touches={r['NAll']} hold={r['HoldP']:.4f} [{r['HoldLo']:.4f},{r['HoldHi']:.4f}] "
            f"n_decided={r['NDet']} n_amb={r['NAmb']}\n"
            f"{indent}fillA: n={fa['n_trades']} win_net={fa['win_rate_net']:.4f} win_gross={fa['win_rate_gross']:.4f} "
            f"exp_pts_net={fa['expectancy_pts_net']:.3f} "
            f"exp_usd_net={fa['expectancy_usd_net']:.3f} pf_net={fa['profit_factor_net']:.3f} maxdd={fa['max_dd_pts_net']:.3f} "
            f"streak={fa['longest_losing_streak']} mae_med={fa['mae_med']:.2f}\n"
            f"{indent}fillB: n={fb['n_trades']} fill_rate={fb['fill_rate_vs_a']:.4f} exp_pts_net={fb['expectancy_pts_net']:.3f}\n"
            f"{indent}fillC: n={fc['n_trades']} exp_pts_net={fc['expectancy_pts_net']:.3f}")

s = load("summary.json")
print("== READS ==")
print(f"built={s['reads_built']} skipped={s['reads_skipped']} in={s['reads_in_sample']} out={s['reads_held_out']}")
print(f"events_total={s['events_total']} zone={s['zone_events']} line={s['line_events']} seated_zone={s['seated_zone_events']} seated_line={s['seated_line_events']}")
print("== SEATED HEADLINE (all eras) ==")
print(c4(s["seated_headline"]))
print("== SEATED IN-SAMPLE ==")
print(c4(s["seated_in_sample"]))
print("== SEATED HELD-OUT ==")
print(c4(s["seated_held_out"]))
print("== ALL-ZONES HEADLINE ==")
print(c4(s["all_zones_headline"]))

c3 = load("c3_baseline.json")
print("\n== C3 ==")
for k, v in c3.items():
    print(f"{k}: p={v['p_hold']:.4f} n={v['n']} amb={v['ambiguous_excluded']} wilson={[round(x,4) for x in v['wilson']]}")

c7 = load("c7_splits.json")
zvsl = c7["zone_vs_line"]
print("\n== C7 zone vs line ==")
print(f"zone p={zvsl['zone_p']:.4f} n={zvsl['zone_n']} wilson={[round(x,4) for x in zvsl['zone_wilson']]}")
print(f"line p={zvsl['line_p']:.4f} n={zvsl['line_n']} wilson={[round(x,4) for x in zvsl['line_wilson']]}")
print(f"diff={zvsl['diff']:.4f} ci={[round(x,4) for x in zvsl['diff_ci']]}  {zvsl['round21_threshold_zone_vs_line']}")
ms = c7["multi_source"]
print(f"fam3+ vs fam1: diff={ms['fam3plus_vs_fam1_diff']:.4f} ci={[round(x,4) for x in ms['diff_ci']]} n1={ms['fam1_n']} n3={ms['fam3plus_n']}  {ms['round21_threshold_multi_source']}")
for k, v in c7["families"].items():
    print(f"  {k}: p={v['p_hold']:.4f} n={v['n']} amb={v['ambiguous']} touches={v['n_touches']} fillA_exp={v['trade_fill_a']['expectancy_pts_net']:.3f} fillA_n={v['trade_fill_a']['n_trades']}")

ss = load("c9_surface_summary.json")
print("\n== C9 surfaces ==")
for k in ("s1", "s2"):
    v = ss[k]
    print(f"{k}: cells={v['cells_with_data']} net_pos={v['cells_net_positive']} median={v['median_mean']:.4f} min={v['min_mean']:.4f} max={v['max_mean']:.4f} best_pmaxT={v['best_p_maxT']:.4f}")
print(f"perms={ss['perms']} correction={ss['correction']}")

seam = load("seam.json")
print("\n== SEAM ==")
print(seam["summary"])

spine = load("c1c4_spine.json")
print("\n== C1 SPINE (seated) ==")
for dim, rows in spine.items():
    print(f"-- {dim} --")
    for r in rows:
        flag = "" if r["NDet"] >= 30 else "  (n<30)"
        print(f"  {r['Group']:<28} n={r['NAll']:<6} decided={r['NDet']:<6} hold={r['HoldP']:.3f} wilson=[{r['HoldLo']:.3f},{r['HoldHi']:.3f}]{flag} expA={r['FillA']['expectancy_pts_net']:+.3f}")
