import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// dev 代理:/api → Go 后端(同源免 CORS)。目标可用 BOSS_API_TARGET 覆盖(默认 102 部署)。
// 注意：集成测试必须使用真实后端数据，不能使用 mock 服务器。
const apiTarget = process.env.BOSS_API_TARGET ?? 'http://192.168.0.102:28080'

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        // vendor 拆分:react 全家桶独立 chunk,业务改动不失效浏览器缓存。
        manualChunks(id: string) {
          if (!id.includes('node_modules')) return undefined
          if (/[\\/](react|react-dom|react-router|react-router-dom|scheduler)[\\/]/.test(id)) return 'vendor'
          return undefined
        },
      },
    },
  },
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
