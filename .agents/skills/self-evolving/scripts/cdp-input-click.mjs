#!/usr/bin/env node
// T14 专用 CDP 采集:admin 免登录注入 + 可信鼠标点击(Input.dispatchMouseEvent)。
// OL 地图不吃合成 MouseEvent,必须走 CDP Input 才能触发 singleclick。
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'

function parseArgs(argv) {
  const args = { out: argv[0], path: '/intel/gis', theme: '', lang: '', base: 'http://localhost:5199',
    api: 'http://192.168.0.102:28080/api/admin/v1', user: 'admin', pass: 'admin123',
    evals: [], clicks: [], settle: 2500, width: 1600, height: 900 }
  for (let i = 1; i < argv.length; i++) {
    const key = argv[i].replace(/^--/, '')
    if (key === 'eval') { args.evals.push(argv[i + 1]); i += 1; continue }
    if (key === 'click') { args.clicks.push(argv[i + 1]); i += 1; continue }
    args[key] = /^[-]?[0-9]+$/.test(argv[i + 1] ?? '') ? Number(argv[i + 1]) : argv[i + 1]
    i += 1
  }
  if (!args.out) { console.error('usage: t14-cdp.mjs <out.png> [--path p] [--theme t] [--lang l] [--eval js]... [--click expr]... [--logs f]'); process.exit(2) }
  return args
}

async function launchChrome(width, height) {
  const profile = mkdtempSync(join(tmpdir(), 't14-cdp-'))
  const proc = spawn(CHROME, ['--headless=new', '--remote-debugging-port=0', '--user-data-dir=' + profile,
    '--window-size=' + width + ',' + height, '--no-proxy-server', 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] })
  const wsUrl = await new Promise((resolve, reject) => {
    let buf = ''
    proc.stderr.on('data', (d) => { buf += d; const m = buf.match(/DevTools listening on (ws:\/\/\S+)/); if (m) resolve(m[1]) })
    proc.on('exit', () => reject(new Error('chrome exited early')))
    setTimeout(() => reject(new Error('chrome ws timeout')), 10000)
  })
  return { proc, profile, port: new URL(wsUrl).port }
}

function connectCdp(wsDebuggerUrl) {
  let seq = 0
  const pending = new Map()
  let onEvent = () => {}
  const ws = new WebSocket(wsDebuggerUrl)
  const failPending = (why) => { for (const [, p] of pending) p.rej(new Error(why)); pending.clear() }
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id).res(msg); pending.delete(msg.id) }
    else if (msg.method) onEvent(msg)
  }
  const ready = new Promise((res, rej) => {
    ws.onopen = () => res()
    ws.onerror = () => rej(new Error('cdp ws error'))
    ws.onclose = (e) => { failPending('closed'); rej(new Error('ws closed')) }
  })
  const send = (method, params = {}) => new Promise((res, rej) => {
    const id = ++seq
    const timer = setTimeout(() => { pending.delete(id); rej(new Error('cdp timeout: ' + method)) }, 30000)
    pending.set(id, { res: (v) => { clearTimeout(timer); res(v) }, rej: (e2) => { clearTimeout(timer); rej(e2) } })
    ws.send(JSON.stringify({ id, method, params }))
  })
  return { ready, send, close: () => ws.close(), setOnEvent: (fn) => (onEvent = fn) }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function createCollector(cdp) {
  const entries = []
  const requests = new Map()
  const consoleTypes = new Set(['log', 'info', 'warn', 'error', 'debug', 'warning'])
  const argVal = (a) => (a.type === 'string' ? a.value : (a.description ?? JSON.stringify(a.value)))
  cdp.setOnEvent((msg) => {
    const { method, params = {} } = msg
    if (method === 'Runtime.consoleAPICalled' && consoleTypes.has(params.type)) {
      entries.push({ kind: 'console', level: params.type, text: (params.args || []).map(argVal).join(' ') })
    } else if (method === 'Runtime.exceptionThrown') {
      const d = params.exceptionDetails
      entries.push({ kind: 'exception', text: (d.text + ' ' + (d.exception?.description || '')).trim() })
    } else if (method === 'Network.requestWillBeSent') {
      requests.set(params.requestId, { method: params.request.method, url: params.request.url })
    } else if (method === 'Network.responseReceived') {
      const r = requests.get(params.requestId); if (r) r.status = params.response.status
    } else if (method === 'Network.loadingFailed') {
      const r = requests.get(params.requestId); if (r) r.error = params.errorText
    }
  })
  await cdp.send('Runtime.enable'); await cdp.send('Log.enable'); await cdp.send('Network.enable')
  return { async dump(file) {
    for (const [id, r] of requests) {
      const ok = r.status && r.status < 400 && !r.error
      if (!ok) { try { const b = await cdp.send('Network.getResponseBody', { requestId: id }); r.responseBody = b.result?.body?.slice(0, 1500) } catch (e) { /* navigation races */ } }
      entries.push({ kind: 'network', ...r })
    }
    writeFileSync(file, JSON.stringify(entries, null, 2)); console.log('logs', file, '(' + entries.length + ')')
  } }
}

async function trustedClick(cdp, x, y) {
  const base = { x, y, button: 'left', buttons: 1, clickCount: 1, pointerType: 'mouse' }
  await cdp.send('Input.dispatchMouseEvent', Object.assign({ type: 'mouseMoved' }, base, { buttons: 0 }))
  await cdp.send('Input.dispatchMouseEvent', Object.assign({ type: 'mousePressed' }, base))
  await sleep(120)
  await cdp.send('Input.dispatchMouseEvent', Object.assign({ type: 'mouseReleased' }, base))
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  const loginRes = await fetch(args.api + '/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: args.user, password: args.pass }) })
  const loginBody = await loginRes.json()
  const token = loginBody?.data?.token
  if (!token) throw new Error('login failed: ' + JSON.stringify(loginBody).slice(0, 200))
  const query = [args.theme && 'theme=' + args.theme, args.lang && 'lang=' + args.lang].filter(Boolean).join('&')
  const target = args.base + args.path + (query ? '?' + query : '')
  const shellBase = args.api.replace(/\/api\/admin\/v1$/, '')
  const servers = JSON.stringify([{ id: 's102', name: '102', baseUrl: shellBase }])
  const seed = "localStorage.setItem('boss.token','" + token + "');localStorage.setItem('boss.servers','" + servers.replace(/'/g, "\\'") + "');localStorage.setItem('boss.server.active','s102');location.href='" + target + "'"
  const { proc, profile, port } = await launchChrome(args.width, args.height)
  try {
    const targets = await fetch('http://127.0.0.1:' + port + '/json').then((r) => r.json())
    const page = targets.find((t) => t.type === 'page')
    const cdp = connectCdp(page.webSocketDebuggerUrl)
    await cdp.ready
    await cdp.send('Page.enable')
    const collector = await createCollector(cdp)
    await cdp.send('Emulation.setDeviceMetricsOverride', { width: args.width, height: args.height, deviceScaleFactor: 1, mobile: false })
    await cdp.send('Page.navigate', { url: args.base + '/login' })
    await sleep(1500)
    const seedRes = await cdp.send('Runtime.evaluate', { expression: seed, awaitPromise: true, returnByValue: true })
    console.log('seed ok')
    await sleep(args.settle)
    for (const expr of args.evals) {
      const r = await cdp.send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true })
      console.log('eval:', JSON.stringify(r.result?.result?.value ?? r.result))
      await sleep(600)
    }
    for (const expr of args.clicks) {
      const r = await cdp.send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true })
      const v = r.result?.result?.value
      console.log('click-eval:', JSON.stringify(v))
      if (Array.isArray(v) && v.length === 2) {
        await trustedClick(cdp, v[0], v[1])
        console.log('trusted click at', v[0], v[1])
      }
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
    rmSync(profile, { recursive: true, force: true })
  }
}
main().catch((e) => { console.error('FATAL', e.message); process.exit(1) })