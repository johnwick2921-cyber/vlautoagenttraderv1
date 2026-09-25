# SANDBOX PREVIEW — hands-on testing with zero risk to live

**LINE 1 — SANDBOX LIVE ON :3001.** Own DB, own ports, no NT8 wire, no order path,
canned AI replies. Live bot never touched. Commit `a6f39d7b`.

## Isolation receipts [A]
| Guarantee | Receipt |
|---|---|
| Live bot untouched | `nofx-bin` **PID 170** before *and* after (started 01:22:07, still :8080 + :36974) |
| Live DB never written | `data/data.db` mtime still **2026-08-14 23:06:11**; `grep data/data.db .sandbox/api.log` → **0 hits** |
| Separate DB | sandbox opens only `data/sandbox.db` (a `sqlite3 .backup` copy, then scrubbed) |
| Separate ports | API **127.0.0.1:8081** · UI **127.0.0.1:3001** · NT8 listener **127.0.0.1:36985** (never 36974) |
| Loopback only | all three bound `127.0.0.1` |
| No auto-trading | every `traders.is_running = 0` in the copy (verified: `sum(is_running)=0`) |
| No NT8 data/orders | its bridge listens on 36985 where NT8 never connects → no bars, no fills |
| No Telegram conflict | `telegram_configs` **deleted** from the copy (0 rows) — can't fight the live bot |
| No paid LLM | `SANDBOX_LLM=mock` default → canned replies; `=real` is opt-in |
| Can't be mistaken for live | 🧪 SANDBOX banner on **every** page, from `GET /api/config {"sandbox":true}` |

## Seeded (verified by query)
plans **v1+v2** · **8 levels** (PDH/RTH-H/nPOC·Tue/ONL/D-4h/EQH/RN/PDL — A/B/C) ·
**3 scenarios** · owner level with 📝 note + [S1] · **7 alerts (5 unacked, 2× P0)** ·
**3 closed trades graded A / C / F** with MAE+MFE · **6 digests** (session + daily +
tapered week) · **10 calendar slices** incl. a T1 red-news blackout · **level-state
with 1 BURNED level** · registry: NY always active, ASIA + LONDON disabled (night tab).

## ── FOR THE OWNER ─────────────────────────────────────────────
1. **Start:** `bash scripts/sandbox-up.sh` (takes ~30s; prints the URL)
2. **Open:** **http://127.0.0.1:3001/** — you should see a yellow 🧪 SANDBOX stripe at the top. If it's not there, you're on the wrong port (3000 = live).
3. **Log in** with your normal email/password (the sandbox DB is a copy of yours).
4. **Plan card** (dashboard, top-left): check bias + flip line, the 8-level table (👤 + 📝 + S1 on your owner level, one dimmed/consumed row), scenarios, the red NO-TRADE / PLAN-DIES-IF block, `v1 v2` version chips, and the alert bell showing **5**.
5. **Edit anything:** tap a level row → change grade/instruction → **Save**. Expect: "⟳ planner reviewing your change…", then either a green "no plan change needed" chip **or** a gold proposal card with a −/+ patch and **[✅ Apply] / [Keep as-is]**. Both are safe; Apply bumps the version chip and flashes the changed rows.
6. **Also try:** ＋Add level · Bulk add (paste `30156 D-zone` / `30246 S-zone` on separate lines) · delete a level · 💬 Ask Planner (type "are u sure about the long bias" → it will DEFEND you) · tap an alert to ack it (badge drops) · switch to the ASIA tab (night/disabled render).
7. **Studio → Day Plan block:** flip toggles, drag the proximity slider, open the sessions accordion (⚪ inherit / 🔸 override). Saves land in `sandbox.db` only.
8. **Break it freely.** To start over: `bash scripts/sandbox-reset.sh` (wipes + reseeds + restarts, one command).
9. **Stop:** `bash scripts/sandbox-down.sh`
10. **Real AI instead of canned** (costs pennies): `SANDBOX_LLM=real bash scripts/sandbox-up.sh`

**Your live bot was never touched:** same process (PID 170) on :8080 the whole time,
and `data/data.db` has not been written since Aug 14 — every sandbox write went to
`data/sandbox.db`.

## Notes
- The canned planner covers all three outcomes on purpose: a **bare** question →
  DEFEND · an **edit/delete** → no-change · an **add/bulk** → PROPOSE-MERGE with a
  3-op patch. They parse through the same `ParsePlannerReply` contract as live.
- Known cosmetic gap (pre-existing, from the design audit): scenario status dots all
  read ARMED because the backend never emits `scenario_status`; the sandbox shows the
  same behavior as live rather than faking it.
- ⚠️ Separately: your `sudo systemctl restart nofx` for W13 did **not** take — the
  binary was rebuilt 06:59 but PID 170 has run since 01:22, so the live bot is still
  on pre-W13 code. Re-run the restart when convenient.
