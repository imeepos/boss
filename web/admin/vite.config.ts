import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 后端已配置 CORS,前端直连绝对接口地址(服务端配置页/登录页选择器,localStorage 记忆)。
// 禁止再挂 /api 开发代理:请求通道唯一 = client.ts 的 apiBaseUrl()。
// 注意：集成测试必须使用真实后端数据，不能使用 mock 服务器。

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
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
