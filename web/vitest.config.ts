import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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
