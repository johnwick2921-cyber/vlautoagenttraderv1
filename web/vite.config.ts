import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    {
      // CLASS 240, caught in this wave's own code: the guard in
      // src/guide/types.ts is a `throw` at MODULE SCOPE. Vite never EXECUTES
      // the module during a build — it substitutes import.meta.env and
      // bundles — so `VITE_GUIDE_BUILT_REV= npm run build` exited 0 and the
      // throw shipped INTO the bundle, where it fires on page load. Proven:
      // the built asset contained 'must be a 40-hex commit sha'. That is
      // strictly worse than the hand-edited constant it replaced, because a
      // stale rev misinforms while a white screen takes the trading UI down.
      // The check has to run in the BUILD, which is here.
      name: 'guide-built-rev-is-a-build-input',
      apply: 'build',
      config() {
        const raw = process.env.VITE_GUIDE_BUILT_REV
        if (typeof raw !== 'string' || !/^[0-9a-f]{40}$/.test(raw)) {
          throw new Error(
            'VITE_GUIDE_BUILT_REV must be a 40-hex commit sha for a production build ' +
              `(got ${raw === undefined ? 'nothing' : JSON.stringify(raw)}). ` +
              'The release workflow sets it from the tag; see deploy/release/README.md.'
          )
        }
      },
    },
    {
      name: 'visible-product-name',
      transformIndexHtml(html) {
        const name = readFileSync(
          new URL('../branding/product.txt', import.meta.url),
          'utf8'
        )
        return html.replace('%PRODUCT_NAME%', name)
      },
    },
  ],
  server: {
    // LOOPBACK ONLY. This dev server proxies /api straight to the bot on :8080,
    // so binding it to 0.0.0.0 handed the whole API to the LAN and defeated the
    // P0 loopback bind on 8080 (proven: GET http://<lan-ip>:3000/api/traders
    // returned 200 while the same request to :8080 was refused). WSL2 runs in
    // mirrored networking mode here, so a Windows-side browser still reaches
    // http://localhost:3000 — the sandbox UI has bound 127.0.0.1 all along.
    host: '127.0.0.1',
    port: 3000,
    // Dev server must never let the browser cache modules — stale cached JS was
    // causing "edit page won't open / can't click" symptoms that source fixes
    // never reached. no-store forces a fresh fetch on every load.
    headers: {
      'Cache-Control': 'no-store',
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
