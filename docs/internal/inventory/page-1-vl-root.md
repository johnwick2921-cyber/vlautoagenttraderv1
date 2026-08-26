# Page 1 — VL / (root)

## Quick reference

- **Route:** `/`
- **Behavior:** Pure redirect. No content of its own.
  - Authenticated user → `Navigate to {ROUTES.settings} replace`
  - Unauthenticated user → `Navigate to {ROUTES.login} replace`
- **Primary source:** [web/src/router/AppRoutes.tsx:432-441](web/src/router/AppRoutes.tsx#L432-L441)
- **Catch-all (`*`):** also routes here (line 519): every unknown path bounces through `/` → `/settings` or `/login`.

## Layer 1: Source code map

### The `/` route declaration

```tsx
// web/src/router/AppRoutes.tsx:432-441
<Route
  path={ROUTES.home}
  element={
    user ? (
      <Navigate to={ROUTES.settings} replace />
    ) : (
      <Navigate to={ROUTES.login} replace />
    )
  }
/>
```

Where `ROUTES.home = '/'` ([paths.ts:11](web/src/router/paths.ts#L11)).

### Auth state hook — `useAuth()`

- **File:** `web/src/contexts/AuthContext.tsx`
- **Consumed at:** `AppRoutes.tsx:416` (`const { user, token, isLoading } = useAuth()`)
- The `<Navigate>` decision keys on `!!user` (boolean coercion of `user` field; `token` is checked separately for protected routes via `isAuthenticated = !!user && !!token`).

### Pre-redirect gating in `AppRoutes`

[`AppRoutes`](web/src/router/AppRoutes.tsx#L415) does three checks BEFORE the route table:

1. **Loading screen** ([line 420-422](web/src/router/AppRoutes.tsx#L420-L422)) — if `isLoading || configLoading`, render `<LoadingScreen />`. No URL change.
2. **System uninitialized** ([line 424-426](web/src/router/AppRoutes.tsx#L424-L426)) — if `systemConfig.initialized === false` AND `!user`, render `<SetupPage />`. This bypasses the route table entirely — `/` (or any path) shows SetupPage.
3. **Otherwise** → route table evaluates → `/` redirect fires.

### LoadingScreen component

- **Defined inline:** [`AppRoutes.tsx:67-85`](web/src/router/AppRoutes.tsx#L67-L85)
- Renders centered `<img src="/icons/vl.svg">` + `t('loading', language)` text.
- Background `#0B0E11`, text `#EAECEF` (the dark theme defaults).

### `LegacyHashRedirect` (sibling of all routes)

- **Defined inline:** [`AppRoutes.tsx:87-111`](web/src/router/AppRoutes.tsx#L87-L111)
- Mounts at the top of `<Routes>` (line 430) and converts pre-router `#agent`, `#traders`, etc. hash URLs to real paths.
- Reads `LEGACY_HASH_ROUTES` from [paths.ts:35-41](web/src/router/paths.ts#L35-L41) (only `agent`, `traders`, `trader`, `details`, `strategy` — the surviving pages).

### Catch-all (`*`) route

```tsx
// web/src/router/AppRoutes.tsx:519
<Route path="*" element={<Navigate to={ROUTES.home} replace />} />
```

Every unknown path (e.g. the 3 deleted pages `/data`, `/strategy-market`, `/competition`) hits this and bounces to `/`, which then re-redirects per auth state. **Two redirects, then resting state.**

## Layer 2: Runtime DOM + network

### Test 1 — Navigate to `/` (user is authenticated)

```
Navigate: http://localhost:3000/
↓
Final URL: http://localhost:3000/settings
```

- Two-step redirect (`/` → home logic decides → `/settings`).
- **Console:** 2 errors, both `favicon.ico 404`. No app errors.
- **Snapshot ref:** `.playwright-mcp/page-2026-05-27T23-44-36-441Z.yml`
- **Hard-reload (F5):** same destination, same console state.

### Test 2 — Negative-space: deleted pages

| Attempted URL | Final URL | Behavior |
|---|---|---|
| `/data` | `/settings` | catch-all → `/` → auth redirect |
| `/strategy-market` | `/settings` | same |
| `/competition` | `/settings` | same |

All three resolve in ≤200ms with no flicker of stale content. **No 404 page — the catch-all silently redirects.** This is intentional: there is no `PageNotFound` component wired into the route table even though `web/src/pages/PageNotFound.tsx` exists on disk (orphan; see Open Questions).

### Network requests fired by the redirect chain

The initial `/` navigation does NOT hit any backend. Both redirects are client-side via React Router `<Navigate>`. The backend traffic that follows belongs to whichever destination page renders (`/settings` or `/login`), not to `/` itself.

## Layer 3: Backend code path

**None.** `/` is purely a client-side router behavior. No HTTP endpoints are exercised by visiting `/`.

The auth check uses `useAuth()` which reads from `AuthContext`. Token rehydration on app boot happens in `AuthContext.tsx` (separate from `/` — happens on every page load) and may hit `/api/auth/me` or similar, but that's `AuthContext`'s responsibility, not the root route's.

## Layer 4: Cross-references + gaps + open questions

### Shipped fixes (Plan tags)

| Plan | File:line | What |
|------|-----------|------|
| Plan 1 Task 11 (commit `45c434a3`) | `web/src/router/paths.ts`, `AppRoutes.tsx` | Removed Data / StrategyMarket / Competition routes — catch-all absorbs them now |

### Known gaps

None specific to `/`. The redirect behavior is correct.

### NEW observations

| Description | File:line | Severity | Recommended scope |
|---|---|---|---|
| `web/src/pages/PageNotFound.tsx` exists on disk but is NOT wired into the route table. The catch-all `*` route redirects to home instead of rendering a 404 page. | `web/src/pages/PageNotFound.tsx` + `AppRoutes.tsx:519` | Cosmetic / orphan code | 10-min decision: either delete `PageNotFound.tsx` or replace the catch-all to render it. |
| Favicon 404s on every page load (`/favicon.ico`). Cosmetic console noise. | n/a — missing asset | Cosmetic | 5-min: add `/web/public/favicon.ico` (or favicon.svg with HTML link tag). |

### Open questions

- Why does `paths.ts` keep `LEGACY_HASH_ROUTES` for `agent` if there's never going to be a `#agent` hash URL anymore? Possibly dead code from the pre-React-Router era. Worth a 5-min cleanup pass.
- `getCurrentPageForPath()` in paths.ts:43-63 lacks a `'settings'` case in its `Page` union — yet `/settings` is the most-visited authenticated route. The function returns `undefined` for `/settings`. Confirm whether this matters for `HeaderBar` nav highlighting (likely it does — Settings tab may not light up in the header).

### Cross-page dependencies

- **`AppChrome` is shared** by every authenticated content page (`/agent`, `/settings`, `/traders`, `/dashboard`, `/strategy`, `/faq`). The `currentPage` prop drives `HeaderBar` highlighting; see Open Questions above.
- **`useAuth()` + `useLanguage()`** wrap everything via `App.tsx` provider chain. The root redirect depends on `useAuth()` resolving — that's `AuthContext`'s job (likely fetches `/api/auth/me` or reads localStorage token).
- **`SetupPage` bypass** (line 424) preempts every route. Documented here because `/` is the route this affects most visibly: a brand-new install shows SetupPage at `/`, not the login redirect.
