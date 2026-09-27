#!/usr/bin/env python3
"""Render the C1 spine + C4 tables as markdown from run/c1c4_spine.json."""
import json, sys

D = sys.argv[1] if len(sys.argv) > 1 else "docs/superpowers/research/2026-09-12-backtest-zone-fade/data/run"
spine = json.load(open(f"{D}/c1c4_spine.json"))

print("| group | n touches | decided | hold% | Wilson | n<30 | fillA n / win-net% / win-gross% / exp net pts / PF | fillB fill-rate / exp | fillC exp | MAE med | maxDD | streak |")
print("|---|---|---|---|---|---|---|---|---|---|---|---|")
for dim, rows in spine.items():
    for r in rows:
        fa, fb, fc = r["FillA"], r["FillB"], r["FillC"]
        flag = "⚠️" if r["NDet"] < 30 else ""
        print(f"| {r['Group']} | {r['NAll']} | {r['NDet']} (amb {r['NAmb']}) | "
              f"{r['HoldP']*100:.1f}% | [{r['HoldLo']*100:.1f}, {r['HoldHi']*100:.1f}] | {flag} | "
              f"{fa['n_trades']} / win net {fa['win_rate_net']*100:.1f}% / win gross {fa['win_rate_gross']*100:.1f}% / {fa['expectancy_pts_net']:+.2f} / PF {fa['profit_factor_net']:.2f} | "
              f"{fb['fill_rate_vs_a']*100:.1f}% / {fb['expectancy_pts_net']:+.2f} | "
              f"{fc['expectancy_pts_net']:+.2f} | {fa['mae_med']:.1f} | {fa['max_dd_pts_net']:.1f} | {fa['longest_losing_streak']} |")
    print()
