// user 端压测 v3: 60VU 并发, API key(客户 213)直连 user 端真实接口。
// 真实约束(102 实测):
//   - 登录: portal_accounts 仅 213 可登录, 验证码 60s 冷却 → 登录并发不可行,
//     登录时延由 user-load.sh 单次真码采样产出, 60VU 主体为已登录会话(API key 同身份)。
//   - 下单: 风控 phoneCap=5/24h、addrCap=3 在途/地址 → 并发下单会被 42300 拦截,
//     user_risk_blocked 单独计数(真实体验信号, 不计入 http_req_failed)。
//   - 支付: sk_test_ 测试环境, stripe-intent 安全。
// 阈值: P95<1500ms; 42300 风控拦截单独统计。
import http from 'k6/http'
import { sleep } from 'k6'
import { Trend, Rate, Counter } from 'k6/metrics'

const BASE = __ENV.BASE || 'http://192.168.0.102:28080'
const TARGET = Number(__ENV.TARGET || 60)
const APIKEY = __ENV.APIKEY || ''

const listLatency = new Trend('user_list_latency', true)
const orderLatency = new Trend('user_order_latency', true)
const payLatency = new Trend('user_pay_latency', true)
const profileLatency = new Trend('user_profile_latency', true)
const riskBlocked = new Counter('user_risk_blocked')   // 42300 风控拦截
const risk4100 = new Counter('user_risk_cooldown')     // 冷却类 42300(发码)
const reqFailed = new Rate('user_failed_requests')

export const options = {
  scenarios: {
    user_path: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '20s', target: TARGET },
        { duration: '1m', target: TARGET },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'user_list_latency': ['p(95)<1500'],
    'user_order_latency': ['p(95)<1500'],
    'user_pay_latency': ['p(95)<1500'],
    'user_profile_latency': ['p(95)<1500'],
    'user_failed_requests': ['rate<0.01'],
    'http_req_failed': ['rate<0.01'],
  },
}

const auth = { headers: { 'X-API-Key': APIKEY, 'Content-Type': 'application/json' } }

export default function () {
  // 读腿: 订单列表(体验关键路径)
  const l = http.get(`${BASE}/api/user/v1/orders?page=1&pageSize=10`, auth)
  listLatency.add(l.timings.duration)
  reqFailed.add(l.status !== 200 || l.json('code') !== 0)

  // 实名读腿: profile(realNameStatus)
  const pr = http.get(`${BASE}/api/user/v1/profile`, auth)
  profileLatency.add(pr.timings.duration)
  reqFailed.add(pr.status !== 200 || pr.json('code') !== 0)

  // 下单腿: 低频(__ITER 每 4 轮 1 次), 289/291 地址轮换, requestId=perf- 幂等可清理
  if (__ITER % 4 === 0) {
    const address = (__VU + __ITER) % 2 === 0 ? '289' : '291'
    const requestId = `perf-${Date.now()}-${__VU}-${__ITER}`
    const o = http.post(`${BASE}/api/user/v1/orders`,
      JSON.stringify({ productId: '103', addressId: address, channelId: '121', billingMode: 'POSTPAID', requestId }),
      auth)
    const code = o.status === 200 ? o.json('code') : -1
    if (code === 42300) { riskBlocked.add(1); return }
    orderLatency.add(o.timings.duration)
    reqFailed.add(o.status !== 200 || code !== 0)
    const orderNo = code === 0 ? o.json('data.orderNo') : ''
    if (orderNo) {
      const p = http.post(`${BASE}/api/user/v1/orders/${orderNo}/stripe-intent`,
        JSON.stringify({ buyMonths: 0 }), auth)
      payLatency.add(p.timings.duration)
      reqFailed.add(p.status !== 200 || p.json('code') !== 0)
    }
  }

  sleep(0.5)
}