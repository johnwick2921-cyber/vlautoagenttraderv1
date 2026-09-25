# Dashboard v6 — production UI branch, before boot

Branch: `fix/dashboard-v6-live-ui-2026-09-06`
Session: `01a07347-1083-75f0-b9c1-b1d7a454828a`
Base at acceptance: `fb6b29e6f826b9e0c83f0d214fa02718ab3f170a`
Final comparison base: `dd35112117d0b06ca90387236a1c636666fd4025` (origin/dev, fetched again before publication).

[A] v6 is **after boot 13**. Its component baseline `d3a3c29c` descends from marker `d6ba826f`, whose RELEASE/GUIDE revision is `2a66bf5d`. The old v5 packet baseline `e28b604d` is not the v6 baseline. This branch starts from today's dev and selects changes from the v6 proposal; it does not install the proposal source tree.

## The two diff questions

### 1. Does removing the old PlanCard block remove the plan or its futures guard?

[A] **No. Relocation is intentional; removal of the guard or lifecycle is not.** The page replaces the old two-column grouping with direct grid children in this DOM order:

1. Market chart (`ChartTabs marketOnly`).
2. Account equity (`EquityChart`, exactly once).
3. `<section id="planner">`, spanning both columns on wide screens.
4. Open positions.
5. Activity / recent decisions.

The entire relocated Planner section remains guarded by `isFutures && selectedTrader.trader_id`. It contains the original `<PlanCard>` with the same trader, symbol and resolved exchange props. Desk remains a child of that original PlanCard. Crypto renders neither Planner nor an empty Planner section.

The overview grid still uses its original CSS `hidden` class when Decisions is selected. **There is no new `dashboardTab === 'overview' &&` around PlanCard.** This deliberately rejects the preview's incidental unmount behavior: Planner drafts/disclosure state and Desk polling survive tab changes. The production-call-site test pins the same instance and an edited draft. Chromium separately checked the real PlanCard/Desk node identity, retained collapse state and continued Desk reads while Decisions was selected.

### 2. Was deleting the six-line status-word comment intentional?

[A] **No: baseline drift in the preview page.** The six-line `2026-09-06 (desk strip, D6b)` comment about ONLINE, HTTP health and 113 minutes of feed silence is retained verbatim from today's dev. The shipped label remains `PROCESS::RESPONDING`; this branch changes neither the label nor the health poll. There is no status-word deletion hunk in this branch.

## Selected production hunks

- `DeskStrip.tsx`: first read now has a visible `role="status"` loading state; it does not assert current data. The existing toggle receives an accessible name and `aria-expanded`. Loading uses the existing translation; toggle names cover en/zh/id. All row semantics, cadence, unknown/stale handling and backend calls are retained.
- `ChartTabs.tsx`: the proposal's `previewMarketOnly` behavior is promoted to the production prop **`marketOnly`**, default false. The Dashboard opts in; other callers retain the original equity default. A derived visible tab also handles a prop change after mount. The mobile select uses the same six market definitions and handler as desktop pills, and appears only for the market view. Existing exchange resolution is unchanged. The toolbar wraps within its narrower grid cell.
- `TraderDashboardPage.tsx`: direct grid cells align market and equity on desktop and stack on mobile; Planner follows them. Existing translated Dashboard label and trader name are added to the header while retaining the short trader ID. Exactly **six** `hidden md:table-cell` removals expose Entry, Mark and Value, each header plus cell. The four crypto-only Leverage/Liquidation header/cell classes remain. The table retains its horizontal-scroll container. Activity follows the other sections rather than retaining the old top-of-column sticky placement.
- A small `guide/content/status.ts` paragraph documents this layout and loading/disclosure behavior. RELEASE and GUIDE_BUILT_REV remain byte-identical to dev: no boot is being claimed or performed.

No `preview-*` class or file is included. AuthContext, httpClient, global index.css, live account bindings, API code and trading code are unchanged. The temporary browser entry is outside the committed tree. No Plan 1 critical file is touched.

## Verification and limits

[A] Production frontend build (`npm run build`): exit 0. Existing bundle-size warning remains. Targeted ESLint over changed TS/TSX: exit 0. `git diff --check`: exit 0.

[A] Focused component tests: **15 passed** (Desk 8, ChartTabs 3, page composition 4). The tests render the production call sites; the page tests mock heavyweight children to pin composition and lifecycle. Existing asynchronous React act warnings were present; no test failed. Guide tests: **15 passed**.

[A] Isolated Chromium browser: **5 groups passed**, real page/components/AuthProvider/httpClient with synthetic responses intercepted at the browser request boundary. Checked first-load Desk announcement, exactly one account equity with market beside it, real Desk node retention and continued polling off-tab, mobile select and visible Entry/Mark/Value, no page exceptions and no unknown API paths. At width 390, document scroll width was 390. No live API, account or order request was sent. This is DOM evidence, not a live broker or backend verification.

[A] `go build ./...` and `go test ./...`: exit 0 in the isolated branch. These checks do not start the bot. No frontend bundle was copied into the main checkout.

The data-quality and trading findings from the earlier precheck are not claimed fixed here. In particular, existing market/exchange resolution and the futures Value calculation are unchanged; this dispatch only reveals the requested cells and changes composition.

## Freshness and review method

Applied `docs/superpowers/AUDIT-CHECKLIST.md` (R1/R9/R10, call-site and artifact verification) and the precheck's mount/single-equity regression requirements. Latest spec/checklist commit at final base:

`dd351121 docs(checklist): class 83 and R10 — an acknowledgement mistaken for a settlement, outside the broker`

Dev advanced from fb6b29e6 during verification only in those two documentation files. The delta was read and the branch rebased to dd351121. No application source changed underneath the UI checks. Publication will be verified from commit-pinned GitHub content, not a cached branch URL.

For semantic review, use GitHub's Hide whitespace option (`?w=1`) or `git diff -w dd351121...HEAD -- web/src/pages/TraderDashboardPage.tsx`. The larger raw page hunk mainly unindents the former left-column wrapper. The full diff still includes all indentation changes. **Review branch only; no merge, deployment, boot or restart.**
