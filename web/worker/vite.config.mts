// Vite 多页配置:docs/worker 草稿页面原样收编为可构建工程。
// API 走同源 /api/worker/v1,dev/preview 代理到 mock(BOSS_API_TARGET 覆盖,默认 8091)。
import { defineConfig } from 'vite';
import { readdirSync } from 'node:fs';
import { resolve } from 'node:path';

const root = __dirname;
const inputs = Object.fromEntries(
  readdirSync(root)
    .filter((f) => f.endsWith('.html'))
    .map((f) => [f.replace(/\.html$/, ''), resolve(root, f)]),
);

const apiTarget = process.env.BOSS_API_TARGET || 'http://127.0.0.1:8091';
const proxy = { '/api/worker/v1': { target: apiTarget, changeOrigin: true } };

export default defineConfig({
  build: { rollupOptions: { input: inputs } },
  server: { port: 5174, proxy },
  preview: { port: 4174, proxy },
});
