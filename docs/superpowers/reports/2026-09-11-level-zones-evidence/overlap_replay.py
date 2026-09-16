"""Read-only Section C replay: existing native bounds; no assumed point widths.
Run from any directory with Python 3. No production code or DB writes.
An undirected edge means native bands overlap OR original anchors are within
m*ATR5m. Connected components implement repeated union without order dependence.
"""
import json
from pathlib import Path

snapshot = json.loads(Path(__file__).with_name("snapshot.json").read_text())
rows = snapshot["candidates"]
results = []
for m in (0, 0.25, 0.5, 0.75):
    parents = list(range(len(rows)))

    def root(i):
        while parents[i] != i:
            parents[i] = parents[parents[i]]
            i = parents[i]
        return i

    for i, a in enumerate(rows):
        x = a["raw_origin"]
        for j in range(i + 1, len(rows)):
            y = rows[j]["raw_origin"]
            overlap = max(x["lo"], y["lo"]) <= min(x["hi"], y["hi"])
            near = abs(x["price"] - y["price"]) <= m * snapshot["atr5m"]
            if overlap or near:
                parents[root(j)] = root(i)
    groups = {}
    for i, row in enumerate(rows):
        groups.setdefault(root(i), []).append(row)
    chart = next(g for g in groups.values() if any(r["id"] == 23094164 for r in g))
    results.append({
        "m": m,
        "tolerance": m * snapshot["atr5m"],
        "components": len(groups),
        "component_sizes": sorted((len(g) for g in groups.values()), reverse=True),
        "chart_component": {
            "n": len(chart),
            "lo": min(r["raw_origin"]["lo"] for r in chart),
            "hi": max(r["raw_origin"]["hi"] for r in chart),
            "row_ids": sorted(r["id"] for r in chart),
            "contains_daily_demand": any(r["id"] == 23094181 for r in chart),
        },
    })
print(json.dumps(results, indent=2))
