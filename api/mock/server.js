// mock server —— 用户端假数据 HTTP 服务(零依赖 Node 标准库)。
// 严格按 api/openapi/user.yaml 的方法/路径返回 data.js 中的假数据。
// 启动: node api/mock/server.js  (默认 0.0.0.0:8090,局域网可访问)
// 跨域已放行,供 docs/user 页面本地 file:// 直接调试。
'use strict';

const http = require('http');
const fs = require('fs');
const path = require('path');
const { URL } = require('url');
const data = require('./data.js');

const PORT = Number(process.env.MOCK_PORT || 8090);

// 静态页面根目录(docs/user),mock 同时托管 HTML 供局域网直接访问。
const STATIC_ROOT = path.resolve(__dirname, '../../docs/user');

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
};

const ok = () => ({ code: 0, message: 'success' });

// 路径 → 处理器;处理器返回 JSON 对象/值。
// 匹配顺序:先精确路径,再带 {id} 路径参数(用正则/前缀)。
function route(method, pathname) {
  const strip = (s) => s.replace(/^\/+|\/+$/g, '');
  const p = strip(pathname);

  if (method === 'GET') {
    if (p === 'home') return data.home;
    if (p === 'profile') return data.profile;
    if (p === 'profile/security') return data.security;
    if (p === 'profile/notify-settings') return data.notifySettings;
    if (p === 'auth/verify') return data.verify;
    if (p === 'products') return data.products;
    if (p === 'addons') return data.addons;
    if (p === 'orders') return data.orders;
    if (p === 'bills') return data.bills;
    if (p === 'payments') return data.payments;
    if (p === 'topups') return data.balance;
    if (p === 'invoices') return data.invoice;
    if (p === 'faults') return data.faults;
    if (p === 'complaints') return data.complaints;
    if (p === 'service/faq') return data.faq;
    if (p === 'messages') return data.messages;
    if (p === 'coupons') return data.coupons;
    if (p === 'usage') return data.usage;
    if (p === 'diy/steps') return data.diySteps;
    if (p === 'agreement') return data.agreement;
    if (p === 'addresses') return { items: data.profile.addresses };

    // 带路径参数的 GET
    if (p.startsWith('products/')) return data.productDetail; // /products/{id}
    if (p.startsWith('orders/')) {
      const seg = p.split('/');
      const orderNo = seg[1];
      if (seg[2] === 'rate') {
        const o = data.orders.items[2];
        return { orderNo: o.orderNo, productName: o.productName, address: o.address, finishedAt: '08-11' };
      }
      return data.orderDetail; // /orders/{orderNo}
    }
    if (p.startsWith('bills/')) return data.billDetail; // /bills/{billNo}
    if (p.startsWith('payments/')) {
      const payNo = p.split('/')[1].replace(/\/receipt$/, '');
      return data.receipts[payNo] || data.receipts['PAY20250817001'];
    }
    if (p.startsWith('faults/')) return data.faultDetail; // /faults/{ticketNo}
    if (p.startsWith('plans/')) {
      const planId = p.split('/')[1];
      if (p.endsWith('/cancel')) {
        // 退订预检
        return {
          unpaidBills: [data.bills.items[0]],
          penalty: 716.4,
          penaltyDesc: '剩余 12 个月 · ¥199 × 30%',
        };
      }
      return data.productDetail; // /plans/{planId} 改套餐预取
    }
  }

  if (method === 'POST') {
    if (p === 'auth/sms-code') return ok();
    if (p === 'auth/login') return { token: 'mock-jwt-token-' + Date.now(), customerId: 1 };
    if (p === 'auth/register') return { token: 'mock-jwt-token-' + Date.now(), customerId: 1 };
    if (p === 'auth/reset-password') return ok();
    if (p === 'auth/logout') return ok();
    if (p === 'auth/verify') return ok();
    if (p === 'orders') {
      // 下单成功返回新订单
      return {
        orderNo: 'ORD-20250817-0' + (Math.floor(Math.random() * 90) + 10),
        status: 'PENDING',
        statusLabel: '待核查',
        productName: '1000M 极速宽带',
        address: '望京X · 3栋 · 501',
        stage: 1,
        stageLabel: '用户下单',
        canRate: false,
      };
    }
    if (p === 'payments') {
      return { payNo: 'PAY20250817' + String(Math.floor(Math.random() * 900) + 100), amount: 158, billPeriod: '2025-08', payMethod: '微信支付', status: 'SUCCESS' };
    }
    if (p === 'topups') {
      return { payNo: 'PAY20250817' + String(Math.floor(Math.random() * 900) + 100), amount: 100, billPeriod: null, payMethod: '微信支付', status: 'SUCCESS' };
    }
    if (p === 'invoices') return ok();
    if (p === 'faults') {
      return { ticketNo: 'TKT-20250817-0' + (Math.floor(Math.random() * 90) + 10), faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA）', address: '望京X · 3栋 · 501', createdAt: new Date().toISOString().slice(0, 16).replace('T', ' '), status: 'PROCESSING', statusLabel: '处理中' };
    }
    if (p === 'complaints') return ok();
    if (p === 'service/chat') {
      return { reply: '已为您检测到当前服务在线正常。建议先自助排障：检查光猫指示灯，重启光猫与路由器。是否需要转报修？', toHuman: false };
    }
    if (p === 'messages/read-all') return ok();
    if (p === 'addresses') return ok();

    // 带参数的 POST(动作类)
    if (p.startsWith('orders/')) {
      const seg = p.split('/');
      if (seg[2] === 'cancel' || seg[2] === 'urge') return ok();
      if (seg[2] === 'change-address') return ok();
      if (seg[2] === 'rate') return ok();
    }
    if (p.startsWith('addons/')) return ok(); // subscribe/unsubscribe
    if (p.startsWith('plans/')) {
      // change / move / cancel 均生成工单
      return {
        orderNo: 'ORD-20250817-0' + (Math.floor(Math.random() * 90) + 10),
        status: 'PENDING',
        statusLabel: '待核查',
        productName: '1000M 极速宽带',
        address: '望京X · 3栋 · 501',
        stage: 1,
        stageLabel: '用户下单',
        canRate: false,
      };
    }
  }

  if (method === 'PUT') {
    if (p === 'profile/security/password') return ok();
    if (p === 'profile/security/phone') return ok();
    if (p === 'profile/notify-settings') return ok();
    if (p === 'profile/language') return ok();
  }

  return null;
}

// 静态文件解析:把 URL 路径映射到 STATIC_ROOT 下的文件,防目录穿越。
// 返回 { file, contentType } 或 null(非静态请求)。
function resolveStatic(pathname) {
  if (pathname.startsWith('/api/')) return null; // API 前缀不走静态
  let rel = pathname === '/' ? 'home.html' : pathname.replace(/^\/+/, '');
  rel = rel.split('?')[0];
  const file = path.resolve(STATIC_ROOT, rel);
  if (!file.startsWith(STATIC_ROOT)) return null; // 防 ../ 穿越
  const ext = path.extname(file).toLowerCase();
  return { file, contentType: MIME[ext] || 'application/octet-stream' };
}

const server = http.createServer((req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const pathname = u.pathname;
  const method = req.method.toUpperCase();

  // CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization');

  if (method === 'OPTIONS') {
    res.statusCode = 204;
    return res.end();
  }

  // 静态页面/资源(GET,非 /api 前缀)
  if (method === 'GET') {
    const st = resolveStatic(pathname);
    if (st && fs.existsSync(st.file)) {
      res.setHeader('Content-Type', st.contentType);
      return fs.createReadStream(st.file).pipe(res);
    }
  }

  // 否则走 JSON API(去 /api/v1 前缀)
  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  const apiPath = pathname.replace(/^\/api\/v1/, '') || '/';
  const result = route(method, apiPath);
  if (result === null) {
    res.statusCode = 404;
    return res.end(JSON.stringify({ code: 40400, message: 'not found: ' + method + ' ' + pathname }));
  }
  res.statusCode = 200;
  res.end(JSON.stringify(result));
});

// 监听 0.0.0.0,允许局域网内其它设备(手机/另一台电脑)访问。
server.listen(PORT, '0.0.0.0', () => {
  console.log(`boss user mock listening on http://0.0.0.0:${PORT}`);
});
