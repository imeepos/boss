// W11 k6 压测:登录 → 管理端关键读链路(订单/资源/GIS/分析)。
// 运行: k6 run -e BASE=http://192.168.0.102:28080 scripts/load/api-load.js
// 阈值(roadmap §2):读接口 p95<500ms、错误率<1%、登录 p95<800ms。
import http from 'k6/http'
import { check, sleep } from 'k6'
import { Trend } from 'k6/metrics'

const BASE = __ENV.BASE || 'http://192.168.0.102:28080'
// 注意:勿用 __ENV.USER/PASS 这类常见 shell 变量名(k6 会继承全量环境变量)。
const USER = __ENV.LOAD_USER || 'admin'
const PASS = __ENV.LOAD_PASS || ''
const DRY_RUN = __ENV.DRY_RUN !== 'false' && PASS === ''

const loginLatency = new Trend('login_latency', true)

// 容量档位可用环境变量覆盖: k6 run -e TARGET=50 scripts/load/api-load.js
const TARGET = Number(__ENV.TARGET || 20)

export const options = {
  scenarios: {
    reads: {
      executor: 'ramping-vus',
      startVUs: 2,
      stages: [
        { duration: '30s', target: TARGET },
        { duration: '1m', target: TARGET },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'http_req_duration{scenario:reads}': ['p(95)<500'],
    'http_req_failed': ['rate<0.01'],
    'login_latency': ['p(95)<800'],
  },
}

export function setup() {
  if (DRY_RUN) {
    console.log(`dry-run: target=${BASE}; set LOAD_PASS and DRY_RUN=false to probe`)
    return { token: '' }
  }
  const res = http.post(`${BASE}/api/admin/v1/auth/login`, JSON.stringify({ username: USER, password: PASS }), {
    headers: { 'Content-Type': 'application/json' },
  })
  check(res, { 'login ok': (r) => r.status === 200 && r.json('code') === 0 })
  if (!res.json('data.token')) {
    throw new Error('login failed: ' + res.body)
  }
  return { token: res.json('data.token') }
}

export default function (data) {
  if (DRY_RUN) {
    sleep(1)
    return
  }
  const auth = { headers: { Authorization: 'Bearer ' + data.token } }

  const loginRes = http.post(`${BASE}/api/admin/v1/auth/login`, JSON.stringify({ username: USER, password: PASS }), {
    headers: { 'Content-Type': 'application/json' },
  })
  loginLatency.add(loginRes.timings.duration)

  for (const path of [
    '/api/admin/v1/auth/me',
    '/api/admin/v1/orders',
    '/api/admin/v1/resources',
    '/api/admin/v1/gis/levels',
    '/api/admin/v1/analytics/indicators',
  ]) {
    const res = http.get(BASE + path, auth)
    check(res, {
      [`${path} 200`]: (r) => r.status === 200,
      [`${path} code=0`]: (r) => r.status === 200 && r.json('code') === 0,
    })
  }
  sleep(1)
}
