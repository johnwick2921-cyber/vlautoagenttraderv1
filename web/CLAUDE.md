# web/ — React + TypeScript + Vite frontend

## Stack

React 18 + TypeScript + Vite + TailwindCSS. SWR for data fetching. Lucide-react for icons. Routing in `web/src/router/`.

## Dev server

- `npm run dev` — port 3000, proxies `/api` to `localhost:8080` (Go backend)
- `npm run build` — production build to `web/dist/` (Go server does NOT serve frontend; nginx is used in Docker/prod, Vite dev server in development)
- `npm run lint` / `npm run format`
- Proxy config: `vite.config.ts`

## 4 control surfaces (the user-facing system)

1. `pages/SettingsPage.tsx` — **Config** (AI models + exchanges + telegram + account)
2. `pages/TraderDashboardPage.tsx` — **Dashboard** (positions, decisions, P&L, charts)
3. `pages/StrategyStudioPage.tsx` — **Strategy** editor (coin source + indicators + risk + prompt)
4. `pages/AgentChatPage.tsx` — **AgentBeta** (NOFXi chat with tool-use)

## Removed pages

`DataPage`, `StrategyMarketPage`, and `CompetitionPage` were deleted in Plan 1
Task 11 (commit `45c434a3` and following). Routes + nav entries also removed.
The remaining pages are Agent, Traders, Trader Dashboard, Strategy Studio,
Settings, FAQ. If you find stray references, delete them.

## Crypto-coupling hotspots (watch when adding non-crypto support)

- `components/strategy/CoinSourceEditor.tsx:69-79` — auto-appends "USDT" to symbols. Skip USDT for CME futures patterns.
- `pages/TraderDashboardPage.tsx:513,522,530` — hardcoded `unit="USDT"` on StatCards. Make conditional on `exchange_type`.
- `pages/TraderDashboardPage.tsx:609,663,673` — Leverage + Liquidation Price columns. Hide for non-margin exchanges.
- `i18n/translations.ts:347,1693,2971` — `invalidSymbolFormat` requires "must end with USDT". Soften.
- `i18n/translations.ts:250` — Aster USDT warning (crypto-specific).
- ~80 i18n keys mention USDT/altcoin/BTC-ETH/funding-rate. Add conditional rendering rather than wholesale i18n rewrite.

## i18n

- `i18n/translations.ts` (main, large)
- `i18n/strategy-translations.ts` (strategy editor specifics)
- Three languages: `en`, `zh`, `id`. Every new label needs all three.

## Component patterns

- Modals use `web/src/components/trader/*Modal.tsx` (step-indicator pattern in ExchangeConfigModal)
- Charts: `components/charts/ChartTabs.tsx` wraps `EquityChart` + `AdvancedChart`
- DecisionCard renders AI decisions with collapsible reasoning/prompt sections
- HeaderBar has BOTH desktop nav (~80-150) AND mobile nav (~450-510) — edit both when changing nav

## State

- `useAuth()` from `contexts/AuthContext` — user/token/logout
- `useLanguage()` — language switching
- SWR for server state (positions, decisions, etc.)
- Per-trader data scoped via query params (`?trader=<slug>`)

## Dev Environment Gotchas

### Vite HMR stale module cache (Playwright verification trap)

**Symptom:** A frontend fix lands cleanly in the source file, `npm run build` succeeds, TypeScript compiles — but the running browser still shows the old behavior (e.g. console warning that should be gone, label that should have changed).

**Cause:** Vite's HMR module cache may serve a stale version of the `.tsx` file. Diagnostic: in the browser console or network tab, the `.tsx` URL returns HTTP 500 instead of 200, serving the cached old module.

**First occurrence:** 2026-05-28 during Plan 4.9 Playwright verification. Initial Test 2 showed React key warning still firing in `DecisionAudit.tsx` despite the fix being live on disk. Hard-reload cleared it. On-disk source was correct; just the Vite HMR cache was stale.

**Workaround (in order of preference):**

1. Hard-reload the browser tab: `Ctrl+Shift+R` (`Cmd+Shift+R` on Mac).
2. Restart the Vite dev server:
   ```bash
   pkill -f 'vite' || pkill -f 'npm run dev'
   cd web && npm run dev > /tmp/frontend.log 2>&1 &
   ```
3. Clear the Vite cache directory:
   ```bash
   rm -rf web/node_modules/.vite
   ```

**Diagnostic command:**

```bash
# Check if Vite is serving the file correctly
curl -sI http://localhost:3000/src/components/path/to/file.tsx
# Expected: HTTP 200
# Stale cache: HTTP 500
```

**For Playwright verification agents:** if a fix appears to not have landed but `git diff` confirms the source change is present and `npm run build` succeeds, ALWAYS hard-reload the browser (use `mcp__playwright__browser_press_key(key='F5')` or navigate with a cache-busting query param) before flagging the test as FAIL. Many false-negatives trace to this.
