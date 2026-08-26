// DEMO PREVIEW ONLY (untracked). Same app, pointed at the isolated preview
// backend on :8232 (a COPY of the DB, all traders stopped). Live bot untouched.
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 3001,
    strictPort: true,
    headers: { 'Cache-Control': 'no-store' },
    proxy: { '/api': { target: 'http://127.0.0.1:8232', changeOrigin: true } },
  },
})
