import sqlite3, json
c = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
for pid, ver, sid in [("2026-09-03:NY:8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",2,"S1"),
                      ("2026-09-04:NY:8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",3,"S2")]:
    row = c.execute("SELECT plan_id, version, lifecycle, created_at, doc FROM plans WHERE plan_id LIKE ? AND version=?", (pid[:20]+'%', ver)).fetchone()
    if not row: print("MISSING", pid, ver); continue
    doc = json.loads(row[4])
    print("== plan", row[0][:18], "v", row[1], "lifecycle", row[2], "created_at(UTC)", row[3])
    print("   bias:", json.dumps(doc.get("bias"))[:200])
    for s in doc.get("scenarios", []):
        if s.get("id") == sid:
            print("   scenario", sid, json.dumps(s, ensure_ascii=False)[:1800])
