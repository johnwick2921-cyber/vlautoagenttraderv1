// SANDBOX UI (port 3001) — the same React app pointed at the isolated sandbox API
// on 127.0.0.1:8081. The live dev server (3000 → 8080) is untouched.
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 3001,
    strictPort: true,
    headers: { 'Cache-Control': 'no-store' },
    proxy: { '/api': { target: 'http://127.0.0.1:8081', changeOrigin: true } },
  },
})
