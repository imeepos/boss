import { defineConfig } from '@playwright/test'

// 冒烟配置:dev server 起在 5173,API 经 vite proxy 代理到 BOSS_API_TARGET(真实 Go 后端)。
// 后端需先行启动(make run / go run ./cmd/server,PG 取 config 默认 102)。
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    locale: 'zh-CN',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  webServer: process.env.E2E_NO_DEV_SERVER
    ? undefined
    : {
        command: 'pnpm dev --port 5173 --strictPort',
        url: 'http://localhost:5173',
        reuseExistingServer: true,
        timeout: 60_000,
        env: { BOSS_API_TARGET: process.env.BOSS_API_TARGET ?? 'http://127.0.0.1:8080' },
      },
})
