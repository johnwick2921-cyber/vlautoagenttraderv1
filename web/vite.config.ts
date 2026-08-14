import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
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
