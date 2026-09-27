# Dashboard Fix — Trader Selection Must Persist (no more snap-back to last-created)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **HEAD at start:** `9fd32410` (≥ 3624a2a4) · **FE-only, additive** — no kernel/trader/risk logic touched; running `nofx-bin` (rev `3624a2a4`) untouched.

## Root cause [A]
Two defects in the dashboard selection logic, both in `web/src/router/AppRoutes.tsx`:
1. **First-run / deleted-selection default was `traders[0]`** — `AppRoutes.tsx:305` (pre-fix) `const fallback = traders[0]`. The traders API returns **creation-DESC** (`api.getTraders`), so `traders[0]` is the **LAST-CREATED** trader. That is exactly the owner's symptom ("hard-defaults to the last-created trader").
2. **Selection was never persisted.** It lived only in the URL `?trader=` param + in-memory React state (`AppRoutes.tsx:226-227`). The effect had no `localStorage`, so any load **without** the param — the header "Dashboard" nav to a bare `/dashboard`, a new tab, a bookmark, or F5 after the param was dropped — reset in-memory state to undefined → fell through to (c) `traders[0]` → **snapped back to the newest trader**. Clicking "View" (`TradersList.tsx:349-356` → `AppRoutes.tsx:210-214`) DID set `?trader=`, so the view showed correctly, but the choice did not survive a fresh load.

What "View" set / where selection lived (pre-fix): View → `?trader=<id>` (URL only) + in-memory state; **nowhere durable.** The slug itself was already correct (full immutable `trader_id`, `router/traderSlug.ts` — a prior fix); the gap was persistence + a stable default.

## Fix (additive, FE-scoped)
**New `web/src/router/selectedTrader.ts`** — selection is now first-class:
- `resolveSelectedTrader(traders, urlSlug, currentId, storedId)` — one pure authority, priority **URL → in-memory → localStorage → a STABLE-sorted first trader** (`byStableTraderOrder`: name then immutable id — never creation-order `[0]`). Reports `clearStored` when a persisted id no longer resolves (deleted trader).
- `loadStoredTraderId` / `saveStoredTraderId` / `clearStoredTraderId` (localStorage, guarded for private-mode/SSR).

**`AppRoutes.tsx` `DashboardRoute`** — the resolution effect now uses the resolver and **persists every resolution to BOTH the URL and localStorage**; idempotent on a stable selection (a poll/remount never silently switches). A View click resolves via URL → writes storage, so a later bare-`/dashboard` load restores it. A deleted stored id → graceful fallback + storage cleared.

**Every dashboard data hook already keys off `selectedTraderId`** (`AppRoutes.tsx` account/status/positions/decisions/equity/accounts SWR keys; `TraderDashboardPage.tsx:174,221,243,408-429`) — unchanged, so 2c was already satisfied.

**Active marker (2d)** — `AITradersPage` reads the persisted selection (`loadStoredTraderId`) and passes `activeTraderId` → `TradersList` → `TraderRow`; the selected trader shows a gold **"Viewing"** badge + highlighted card border (`TradersList.tsx`). i18n `viewing` added for en/zh/id (`i18n/translations.ts`).

## Tests
`web/src/router/selectedTrader.test.ts` (8 new): priority order (url > memory > storage > fallback), localStorage-restores-on-fresh-load, stable-sort fallback ≠ creation-order `[0]`, deleted-id → fallback + `clearStored`, stale-URL fall-through, empty list → none, deterministic sort. **20/20 router tests green** (8 new + 12 existing `traderSlug`/`paths`); **`tsc && vite build` green.**

## Commits
- `8e8591a8` — fix(web): trader selection persists — no more snap-back to last-created (resolver + AppRoutes wiring + persistence + tests)
- `e3276a7d` — feat(web): active-trader marker in the traders list (badge + i18n)

## Deploy status — HOT, no Go restart
The Go binary does **not** serve/embed the frontend (verified: zero `go:embed`/`StaticFS`/`web/dist` references in `main.go`/`api/server.go`; web/CLAUDE.md confirms nginx-or-Vite serves the FE). The **Vite dev server is running** (`:3000 LISTENING`, PID 390/391) and is the live surface — so this change ships **hot via HMR**. No Go rebuild/restart is required; the running `nofx-bin` (rev `3624a2a4`) is untouched.

**Owner action:** hard-reload the dashboard tab — **Ctrl+Shift+R** (`Cmd+Shift+R` on Mac) — to clear any stale Vite HMR module cache (web/CLAUDE.md gotcha) and pick up the new modules. Nothing else. (If a production/nginx surface is later used, it serves `web/dist/`, so a `cd web && npm run build` regenerates it — still no Go restart.)

## Verify after reload
1. View trader A → refresh (F5) → still A (URL carries `?trader=<A>`).
2. Click header "Dashboard" (drops `?trader=`) → still A (localStorage restores).
3. Open a new tab to `/dashboard` → A, not the last-created.
4. The traders list shows the gold **"Viewing"** badge on A.
5. Delete A's stored selection scenario (delete A) → graceful fallback to the stable-first trader, storage cleared.
