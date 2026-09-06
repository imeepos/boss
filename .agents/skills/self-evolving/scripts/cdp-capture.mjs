#!/usr/bin/env node
// cdp-capture: 零依赖网页截图(Node>=22 + macOS 系统 Chrome)。自带 Chrome 启停,全新临时 profile 防状态泄漏。
// 用法:
//   node cdp-capture.mjs <url> <out.png> [--eval 'js'] [--settle 2500] [--width 1600] [--height 900]
//                          [--logs out.json] [--user-data-dir /tmp/boss-cdp-profile]
//   --eval 可重复多次,按顺序在页面加载后执行(如填表登录);返回 Promise 会被 await。
//   --logs 把浏览器 console 输出 + 网络请求(含失败响应体)写成 JSON,截图外补充"为什么"层面的调试信息。
//   --user-data-dir 指定持久 Chrome profile,多次调用复用 cookie/localStorage;未指定仍用一次性临时 profile。
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'

function parseArgs(argv) {
  const args = { url: argv[0], out: argv[1], evals: [], settle: 2500, width: 1600, height: 900 }
  for (let i = 2; i < argv.length; i++) {
    const key = argv[i].replace(/^--/, '')
    if (key === 'eval') { args.evals.push(argv[i + 1]); i += 1; continue }
    if (key === 'no-proxy') { args.noProxy = true; continue } // 布尔旗标:内网直连,绕过系统代理
    args[key] = /^\d+$/.test(argv[i + 1] ?? '') ? Number(argv[i + 1]) : argv[i + 1]
    i += 1
  }
  if (!args.url || !args.out) {
    console.error('usage: cdp-capture.mjs <url> <out.png> [--eval js]... [--settle ms] [--width px] [--height px] [--logs out.json]')
    process.exit(2)
  }
  return args
}

async function launchChrome(width, height, userDataDir, noProxy) {
  const profile = userDataDir || mkdtempSync(join(tmpdir(), 'cdp-shot-'))
  if (userDataDir && !existsSync(userDataDir)) mkdirSync(userDataDir, { recursive: true })
  const proc = spawn(CHROME, [
    '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
    `--window-size=${width},${height}`, ...(noProxy ? ['--no-proxy-server'] : []), 'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] })
  const wsUrl = await new Promise((resolve, reject) => {
    let buf = ''
    proc.stderr.on('data', (d) => {
      buf += d
      const m = buf.match(/DevTools listening on (ws:\/\/\S+)/)
      if (m) resolve(m[1])
    })
    proc.on('exit', () => reject(new Error('chrome exited early')))
    setTimeout(() => reject(new Error('chrome ws timeout')), 10000)
  })
  return { proc, profile, port: new URL(wsUrl).port }
}

function connectCdp(wsDebuggerUrl) {
  let seq = 0
  const pending = new Map() // id -> { res, rej }
  let onEvent = () => {}
  const ws = new WebSocket(wsDebuggerUrl)
  // chrome 中途崩溃时必须拒绝在途请求,否则 await cdp.send 永挂(2026-09-07 W2 实测:chrome 退出后 node 静默挂死)。
  const failPending = (why) => { for (const [, p] of pending) p.rej(new Error(why)); pending.clear() }
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.id && pending.has(msg.id)) {
      pending.get(msg.id).res(msg)
      pending.delete(msg.id)
    } else if (msg.method) onEvent(msg)
  }
  // ws 打不开/被关必须 reject,否则 ready 永挂(2026-08-24 实测:静默 60s 超时无任何输出)。
  const ready = new Promise((res, rej) => {
    ws.onopen = () => res()
    ws.onerror = (e) => rej(new Error('cdp ws error: ' + (e?.message ?? 'unknown')))
    ws.onclose = (e) => {
      const why = 'cdp ws closed code=' + (e?.code ?? '?')
      failPending(why)
      rej(new Error(why))
    }
  })
  const send = (method, params = {}) =>
    new Promise((res, rej) => {
      const id = ++seq
      // eval 触发 location.href 会撕掉执行上下文,响应可能永不回来(2026-09-07 W2 实测竞态挂死):
      // 30s 超时兜底,失败方干净退出可重试,不再无限等待。
      const timer = setTimeout(() => { pending.delete(id); rej(new Error('cdp send timeout: ' + method)) }, 30000)
      pending.set(id, {
        res: (v) => { clearTimeout(timer); res(v) },
        rej: (e2) => { clearTimeout(timer); rej(e2) },
      })
      ws.send(JSON.stringify({ id, method, params }))
    })
  return { ready, send, close: () => ws.close(), setOnEvent: (fn) => (onEvent = fn) }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

// console + 网络采集:截图只给"结果",这里给"原因"。
async function createCollector(cdp) {
  const entries = []
  const requests = new Map() // requestId → {method,url,postData}
  const consoleTypes = new Set(['log', 'info', 'warn', 'error', 'debug', 'warning'])
  const argVal = (a) => (a.type === 'string' ? a.value : (a.description ?? JSON.stringify(a.value)))
  const onEvent = async (msg) => {
    const { method, params = {} } = msg
    if (method === 'Runtime.consoleAPICalled' && consoleTypes.has(params.type)) {
      entries.push({ kind: 'console', level: params.type, text: (params.args || []).map(argVal).join(' '), at: params.timestamp })
    } else if (method === 'Runtime.exceptionThrown') {
      const d = params.exceptionDetails
      entries.push({ kind: 'exception', text: `${d.text} ${d.exception?.description || ''}`.trim(), at: params.timestamp })
    } else if (method === 'Log.entryAdded') {
      entries.push({ kind: 'browser', level: params.entry.level, text: `${params.entry.source}: ${params.entry.text}`, at: params.entry.timestamp })
    } else if (method === 'Network.requestWillBeSent') {
      requests.set(params.requestId, { method: params.request.method, url: params.request.url, postData: params.request.postData })
    } else if (method === 'Network.responseReceived') {
      const r = requests.get(params.requestId)
      if (r) r.status = params.response.status
    } else if (method === 'Network.loadingFailed') {
      const r = requests.get(params.requestId)
      if (r) r.error = params.errorText
    }
  }
  cdp.setOnEvent(onEvent)
  await cdp.send('Runtime.enable')
  await cdp.send('Log.enable')
  await cdp.send('Network.enable')
  return {
    // 失败请求(>=400 或网络错误)顺带抓响应体,是排 4xx/5xx 最直接的证据。
    async dump(file) {
      for (const [id, r] of requests) {
        const ok = r.status && r.status < 400 && !r.error
        if (!ok) {
          const body = await cdp.send('Network.getResponseBody', { requestId: id })
          r.responseBody = body.result?.body?.slice(0, 2000)
        }
        entries.push({ kind: 'network', ...r })
      }
      writeFileSync(file, JSON.stringify(entries, null, 2))
      console.log('logs', file, `(${entries.length} entries)`)
    },
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const persistentProfile = args['user-data-dir'] || ''
  const { proc, profile, port } = await launchChrome(args.width, args.height, persistentProfile, args.noProxy)
  try {
    const targets = await fetch(`http://127.0.0.1:${port}/json`).then((r) => r.json())
    const page = targets.find((t) => t.type === 'page')
    const cdp = connectCdp(page.webSocketDebuggerUrl)
    await cdp.ready
    await cdp.send('Page.enable')
    const collector = await createCollector(cdp)
    await cdp.send('Emulation.setDeviceMetricsOverride', {
      width: args.width, height: args.height, deviceScaleFactor: 1, mobile: false,
    })
    await cdp.send('Page.navigate', { url: args.url })
    await sleep(args.settle)
    for (const expr of args.evals) {
      const r = await cdp.send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true })
      console.log('eval:', JSON.stringify(r.result?.result?.value ?? r.result))
      await sleep(args.settle)
    }
    const shot = await cdp.send('Page.captureScreenshot', { format: 'png' })
    writeFileSync(args.out, Buffer.from(shot.result.data, 'base64'))
    console.log('saved', args.out)
    if (args.logs) await collector.dump(args.logs)
    cdp.close()
  } finally {
    proc.kill('SIGTERM')
    await Promise.race([once(proc, 'exit'), sleep(2000)])
    if (!persistentProfile) rmSync(profile, { recursive: true, force: true })
  }
}

main().catch((e) => {
  console.error(e.message)
  process.exit(1)
})
