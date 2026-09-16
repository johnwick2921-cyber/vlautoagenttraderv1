/* eslint-disable react-hooks/rules-of-hooks --
 * Playwright's fixture API hands each fixture a callback named `use`, which the
 * react-hooks lint rule mistakes for a React Hook. There is no React here.
 */
import { test as base, chromium, Page, expect } from '@playwright/test'

/**
 * Browser fixture.
 *
 * With E2E_CDP_URL set, tests drive an ALREADY-RUNNING browser over the DevTools
 * protocol instead of Playwright's bundled chromium (which cannot start on this
 * WSL2 box — no libnspr4.so, and `playwright install-deps` needs root).
 *
 * This deliberately depends on NOTHING from the base fixtures: destructuring
 * `page` or `browser` would instantiate Playwright's own browser first, which is
 * exactly the launch that fails here.
 */
const CDP = process.env.E2E_CDP_URL || ''

export const test = base.extend<{ page: Page }>({
  page: async ({ viewport }, use) => {
    const browser = CDP
      ? await chromium.connectOverCDP(CDP)
      : await chromium.launch()
    const context = await browser.newContext({
      viewport: viewport || { width: 1440, height: 900 },
    })
    const page = await context.newPage()
    await use(page)
    await context.close()
    await browser.close()
  },
})

export { expect }
