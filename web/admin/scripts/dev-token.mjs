#!/usr/bin/env node
// dev-token: 真实 /auth/login 换 JWT,输出免登录 URL(或裸 token)。禁止假数据,token 即真实会话。
// 用法:
//   node scripts/dev-token.mjs            # 打印 http://localhost:5173/?token=<jwt>
//   node scripts/dev-token.mjs --raw      # 只打印 JWT
// 环境变量:BOSS_API_TARGET(默认 102 部署)、BOSS_ADMIN_USER、BOSS_ADMIN_PASS
const API = process.env.BOSS_API_TARGET ?? 'http://192.168.0.102:28080'
const USER = process.env.BOSS_ADMIN_USER ?? 'admin'
const PASS = process.env.BOSS_ADMIN_PASS ?? 'admin123'
const APP = process.env.BOSS_WEB_ORIGIN ?? 'http://localhost:5173'

// 前缀与 src/lib/serverConfig.ts 的 API_PREFIX 对齐(/api/admin/v1);裸 /auth/login 会 404。
const res = await fetch(`${API}/api/admin/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: USER, password: PASS }),
})
if (!res.ok) {
  console.error(`登录请求失败(HTTP ${res.status}):${API}`)
  process.exit(1)
}
const env = await res.json()
if (env.code !== 0 || !env.data?.token) {
  console.error(`登录被拒:code=${env.code} msg=${env.msg ?? ''}`)
  process.exit(1)
}
console.log(process.argv.includes('--raw') ? env.data.token : `${APP}/?token=${env.data.token}`)
