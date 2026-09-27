# SANDBOX ADD-ON — diagnosis + full seed

**LINE 1 — DIAGNOSIS: the W13 backend worked (3 calls, 200, real verdicts); the card
was EMPTY because level facts come from NT8 bars the sandbox never had. Fixed —
SANDBOX SEEDED, go test.** Commit `517df64c`.

## 1. Diagnosis (evidence before fixing)
| Question | Answer |
|---|---|
| Active plan row? | **Yes** — `2026-08-16:NY` v1 + v2, both `lifecycle=active` |
| API serving /plan/*? | **Yes** — 95 × `/plan/today`, 2 × owner-level save, **3 × `/plan/realign`**, all **200** |
| FE pointed at sandbox? | **Yes** — vite 3001 → 8081, requests landed |
| W13 path reachable? | **Yes** — handler logged `DEFEND`, then two `PROPOSE-MERGE` (mock, $0.0002, 0ms) |
| So why "nothing happened"? | **Not skip-by-design, not a missing backend, not a missing mock.** `planLevelFacts` computes from the NT8 BarCache *at request time*; a sandbox has no NT8 wire → empty cache → `level_facts=[]`, `price=0` → **levels table rendered empty, chart gated off** (`facts.length>0`), and the B2 armor had no reference (a fat-fingered **305000** owner level got saved). The proposal card did appear — under an empty table — and the no-change chip auto-fades in 4.2s. |

FE was ruled out by reproducing the exact gesture in a test (＋Add level → Save →
assert the proposal card): **it passes** (`W13_integration.test.tsx`).

## 2-3. Fixes + seed
- **`api.InstallSandboxBars`** — deterministic synthetic 1m/5m/15m/1h/4h/1d series,
  installed from `main.go` **only** when `SANDBOX_MODE=1`. Card now serves
  **price 30245.75 with 9 level facts**; chart renders; armor has a reference again.
- **`level_facts` now emit `origin` / `note` / `scenario_id`** (tick-rounded match
  against the sticky owner-level store). This is the design audit's **#1 finding**:
  the FE always rendered 👤 / 📝 / [S-tag] / the ⚡ conflict chip from these, but the
  API never sent them — the whole owner-attribution layer was unreachable. **Also
  fixes live.**
- **`scenario_status` passthrough** (`system_config "scenario_status:<plan_id>"`) so
  ○ waiting ◉ armed ● triggered ✕ invalidated render. Explicitly a **scaffold, not a
  computation** — absent in production, so live keeps its honest fallback.
- **Seeded:** NY v1+v2 · 9 levels (PDH/RTH-H/nPOC/ONL/D-4h/EQH/RN/PDL) A/B/C ·
  **4 scenarios in all four states** · **5 owner levels** incl. the **⚡ conflict pair**
  (owner 30246 "watch reclaim" vs AI 30246.25 "fade first touch", 0.2 pts apart) and
  a **sticky** level carried from an earlier session · 7 alerts (5 unacked, 2× P0) ·
  3 graded trades (A/C/F) with MAE/MFE · digests · calendar with a T1 blackout ·
  a **burned** level · ASIA/LONDON disabled (night tab).

## 4-5. Your steps (URL: **http://127.0.0.1:3001/** — yellow 🧪 SANDBOX stripe = right port)
1. `bash scripts/sandbox-reset.sh` → wait for "SANDBOX READY", then open the URL and log in normally.
2. **Plan card** — expect a populated levels table (9 rows, distances, grades), 👤+📝 on your levels, an S-tag chip, one dimmed *consumed* row, and `v1 v2` chips.
3. **Look at 30246** — the ⚡ conflict chip with the AI row ghosted (owner wins).
4. **Scenarios** — S1 ● triggered · S2 ◉ armed · S3 ○ waiting · S4 ✕ invalidated.
5. **＋Add level → `30158` → Save** (near S1) → "⟳ planner reviewing…" → **PROPOSE-MERGE card** with −/+ rows and *"would become v1+o2"*. Tap **✅ Apply** → version chip bumps, rows flash gold. (Or **Keep as-is** → it just dismisses.)
6. **＋Add level → `29500` → Save** (far from everything) → green **"no plan change needed"** chip (auto-fades in ~4s — watch for it).
7. **💬 Ask Planner → "are you sure?"** → **DEFEND** (anti-sycophancy: a bare challenge never gets a patch).
8. **Bulk add** 3 lines → exactly **ONE** re-align call for the batch.
9. **Alerts bell (5)** → tap one to ack → badge drops.
10. Break anything, then `bash scripts/sandbox-reset.sh`. Stop with `bash scripts/sandbox-down.sh`.

## 6. Live untouched (receipts)
- Live bot **PID 101785** on :8080 + :36974 — unchanged throughout (that PID is from *your* W13 restart at 07:19, not from this work).
- Live dashboard **:3000** never touched; sandbox is 3001 → 8081 → `data/sandbox.db`.
- `data/data.db` never opened by the sandbox process (0 hits in `.sandbox/api.log`).

**Note:** the fat-finger you saved (305000) is gone after the reset, and the armor
now has a price reference, so that class of input is rejected again.
