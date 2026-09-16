import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  plugins: [react()],
  server: {
    fs: {
      // The shared product name is outside web; allow only its source tree
      // alongside web, independently of the runner's workspace discovery.
      allow: [
        fileURLToPath(new URL('.', import.meta.url)),
        fileURLToPath(new URL('../branding', import.meta.url)),
      ],
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    css: true,
    // E4: e2e/*.spec.ts are PLAYWRIGHT suites — vitest collecting them threw
    // "Playwright Test did not expect test.describe() to be called here" on
    // every full run. They run via playwright, not here.
    exclude: ['**/node_modules/**', '**/dist/**', 'e2e/**'],
  },
})
