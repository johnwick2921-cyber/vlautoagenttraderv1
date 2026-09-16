import { defineConfig, devices } from '@playwright/test'

/**
 * ACCEPTANCE GATE — Part E live E2E.
 *
 * Runs the real frontend (vite dev server on :3000, which proxies /api to the
 * DEPLOYED bot on :8080) in a headless browser. READ-mostly: any test that
 * changes a setting restores it in-test.
 *
 * BROWSER: this WSL2 box has no Linux browser system libs — Playwright's bundled
 * chromium-headless-shell dies on libnspr4.so and `playwright install-deps`
 * needs root. Set E2E_CDP_URL to attach to an already-running browser instead
 * (see e2e/fixtures.ts). Windows Chrome works because WSL2 mirrored networking
 * shares localhost:
 *
 *   "/mnt/c/Program Files/Google/Chrome/Application/chrome.exe" \
 *     --headless=new --remote-debugging-port=9222 --user-data-dir='C:\Windows\Temp\vlgate' &
 *   export E2E_CDP_URL=http://127.0.0.1:9222
 *
 * AUTH: the app reads its JWT from localStorage['auth_token']
 * (web/src/lib/api/helpers.ts:10). Specs that need a session skip unless
 * E2E_TOKEN or E2E_EMAIL+E2E_PASSWORD are exported:
 *
 *   export E2E_TOKEN=$(go run ./cmd/gate-jwt johnwick2921@gmail.com)   # app's own signer
 *   cd web && npx playwright test -c e2e/playwright.config.ts
 */
export default defineConfig({
  testDir: '.',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 45_000,
  expect: { timeout: 10_000 },
  reporter: [['list'], ['html', { outputFolder: 'e2e-report', open: 'never' }]],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://127.0.0.1:3000',
    screenshot: 'only-on-failure',
    video: 'off',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
      },
    },
    {
      name: 'mobile',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 390, height: 844 },
      },
    },
  ],
  webServer: {
    command: 'npm run dev',
    cwd: '..',
    url: 'http://127.0.0.1:3000',
    reuseExistingServer: true,
    timeout: 120_000,
  },
})
