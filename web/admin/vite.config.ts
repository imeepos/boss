import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// dev 代理:/api → Go 后端(同源免 CORS)。目标可用 BOSS_API_TARGET 覆盖(默认 8080)。
const apiTarget = process.env.BOSS_API_TARGET ?? 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
