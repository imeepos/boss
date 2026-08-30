import { execSync } from 'node:child_process'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// 后端已配置 CORS,前端直连绝对接口地址(服务端配置页/登录页选择器,localStorage 记忆)。
// 禁止再挂 /api 开发代理:请求通道唯一 = client.ts 的 apiBaseUrl()。
// 注意：集成测试必须使用真实后端数据，不能使用 mock 服务器。

// 构建期注入 git 短 SHA,供 VersionBadge 与 /healthz 自报 commit 比对(版本自证消费端);
// 非 git 树(理论不发生)回退空串,VersionBadge 以空值跳过比对。
function buildCommit(): string {
  try {
    return execSync('git rev-parse --short HEAD').toString().trim()
  } catch {
    return ''
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '../..', '')
  return {
  // 读仓库根 .env,将 AMAP_KEY 映射为客户端公开变量;CI 可直接注入 AMAP_KEY。
  envDir: '../..',
  define: {
    'import.meta.env.VITE_AMAP_KEY': JSON.stringify(env.AMAP_KEY ?? ''),
    'import.meta.env.VITE_BUILD_COMMIT': JSON.stringify(buildCommit()),
  },
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
  }
})
