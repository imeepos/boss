#!/usr/bin/env node
// cdp-admin-capture: boss admin 免登录主题/语言采集包装器(零依赖,Node>=22)。
// 封装 templates.md 模板 C/E 的两步注入:任意页写 localStorage(boss.token+servers+active)
// → location.href 跳目标页带 ?theme=&lang=。token 走 localStorage 注入而非 ?token= URL 参数,
// 故 vite dev 与 102 生产(5180)通用;servers 必须先于受保护页启动写入,否则 AuthGuard
// 预取 /auth/me 失败静默清除 boss.token 弹回登录页(2026-08-27 recidivism)。
// 用法:
//   node cdp-admin-capture.mjs <out.png> [--path /bss/marketing] [--theme dark] [--lang en-US]
//        [--base http://localhost:5173] [--token JWT | --user admin --pass admin123]
//        [--eval 'js']... [--settle ms] [--logs out.json] [--width px] [--height px] [--user-data-dir dir]
//   --eval 透传给 cdp-capture.mjs,在目标页加载后按序执行,返回值打印到 stdout(断言即产物)。
// 例:
//   node cdp-admin-capture.mjs dark-en.png --path /bss/marketing --theme dark --lang en-US \
//     --eval "const d=document.querySelector('[role=dialog]');d?'open':'closed'"
import { spawn } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const THEMES = new Set(['light', 'dark'])
const LANGS = new Set(['zh-CN', 'en-US', 'ms-MY'])

function parseArgs(argv) {
  const args = {
    out: argv[0], path: '/', theme: '', lang: '',
    base: 'http://localhost:5173', api: 'http://192.168.0.102:28080/api/admin/v1',
    user: 'admin', pass: 'admin123', token: '',
    evals: [], settle: 2500, logs: '', userDataDir: '',
  }
  for (let i = 1; i < argv.length; i += 2) {
    const key = argv[i].replace(/^--/, '')
    const val = argv[i + 1]
    if (key === 'eval') { args.evals.push(val); i += 1; continue } // 可重复:消耗 flag+value 两格
    if (key === 'no-proxy') { args['no-proxy'] = true; continue } // 布尔旗标:透传给 cdp-capture
    else args[key] = /^\d+$/.test(val) ? Number(val) : val
  }
  if (!args.out) {
    console.error('usage: cdp-admin-capture.mjs <out.png> [--path p] [--theme t] [--lang l] [--base u] [--token jwt] [--eval js]... [--logs f] [--settle ms]')
    process.exit(2)
  }
  if (args.theme && !THEMES.has(args.theme)) { console.error(`bad --theme: ${args.theme}`); process.exit(2) }
  if (args.lang && !LANGS.has(args.lang)) { console.error(`bad --lang: ${args.lang}`); process.exit(2) }
  return args
}

async function fetchToken(api, user, pass) {
  const res = await fetch(`${api}/auth/login`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: user, password: pass }),
  })
  const body = await res.json()
  const token = body?.data?.token
  if (!token) throw new Error(`login failed: http ${res.status} code=${body?.code} msg=${body?.msg}`)
  return token
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const token = args.token || (await fetchToken(args.api, args.user, args.pass))
  const query = [
    args.theme && `theme=${args.theme}`, args.lang && `lang=${args.lang}`,
  ].filter(Boolean).join('&')
  const target = `${args.base}${args.path}${query ? `?${query}` : ''}`
  // 两步注入:第 1 步任意页(origin 必须同源,取 base+/login)写 localStorage;第 2 步同 eval 内跳转。
  const shellBase = args.api.replace(/\/api\/admin\/v1$/, '')
  const servers = JSON.stringify([{ id: 's102', name: '102', baseUrl: shellBase }])
  const seed = `localStorage.setItem('boss.token',${JSON.stringify(token)});`
    + `localStorage.setItem('boss.servers',${JSON.stringify(servers)});`
    + `localStorage.setItem('boss.server.active','s102');location.href=${JSON.stringify(target)}`
  const script = join(dirname(fileURLToPath(import.meta.url)), 'cdp-capture.mjs')
  const passthrough = []
  for (const ev of args.evals) passthrough.push('--eval', ev)
  if (args.settle) passthrough.push('--settle', String(args.settle))
  if (args.logs) passthrough.push('--logs', args.logs)
  if (args.width) passthrough.push('--width', String(args.width))
  if (args.height) passthrough.push('--height', String(args.height))
  if (args['user-data-dir']) passthrough.push('--user-data-dir', args['user-data-dir'])
  if (args['no-proxy']) passthrough.push('--no-proxy')
  const child = spawn(process.execPath, [script, `${args.base}/login`, args.out, '--eval', seed, ...passthrough], { stdio: 'inherit' })
  const code = await new Promise((res) => child.on('exit', res))
  process.exit(code ?? 1)
}

main().catch((e) => {
  console.error(e.message)
  process.exit(1)
})
