import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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
