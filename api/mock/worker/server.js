// mock server —— 师傅端假数据 HTTP 服务(零依赖 Node 标准库)。
// 严格按 api/openapi/worker.yaml 的方法/路径返回 data.js 中的假数据。
// 启动: node api/mock/worker/server.js  (默认 0.0.0.0:8091,局域网可访问)
// 跨域已放开,供 docs/worker 页面本地 file:// 直接调试。
'use strict';

const http = require('http');
const fs = require('fs');
const path = require('path');
const { URL } = require('url');
const data = require('./data.js');

const PORT = Number(process.env.MOCK_WORKER_PORT || 8091);
const STATIC_ROOT = path.resolve(__dirname, '../../../docs/worker');

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.svg': 'image/svg+xml',
};

const ok = (message) => ({ code: 0, message: message || 'success' });

// 扫码绑定按请求体 epc 决定一致/不一致;offline=true 走离线缓存分支
function scanBindResult(body) {
  const b = body || {};
  if (b.offline) return { matched: true, result: 'OFFLINE_CACHED', message: '当前无网络，扫码结果已本地缓存，联网后将自动同步，不会丢失。', quad: data.quad };
  return b.epc === 'EPC-0001' ? data.scanOk : data.scanBad;
}

function ticketDetailOf(ticketNo) {
  if (ticketNo.indexOf('TKT') === 0 || ticketNo.indexOf('EMG') === 0) return data.repairDetail;
  if (ticketNo === 'ORD-20250817-009') return data.dismantleDetail;
  return data.installDetail;
}

// 路径 → 处理器;路径已去 /api/worker/v1 前缀与首尾斜杠。
function route(method, pathname) {
  const p = pathname.replace(/^\/+|\/+$/g, '');
  const seg = p.split('/');
  const ticketNo = seg[1];

  if (method === 'GET') {
    if (p === 'home') return data.home;
    if (p === 'tickets') return { items: data.tickets.doing.concat(data.tickets.todo, data.tickets.done) };
    if (p === 'tickets/history') return data.history;
    if (p === 'tickets/' + ticketNo) return ticketDetailOf(ticketNo);
    if (p === 'tickets/' + ticketNo + '/navi') return data.navi;
    if (p === 'tickets/' + ticketNo + '/photos') return data.photos;
    if (p === 'tickets/' + ticketNo + '/report') return data.report;
    if (p === 'tickets/' + ticketNo + '/activation') return data.activation;
    if (p === 'tickets/' + ticketNo + '/charge') return data.charge;
    if (p === 'tickets/' + ticketNo + '/replace') return data.replace;
    if (p === 'tickets/' + ticketNo + '/measure') return data.measure;
    if (p === 'tickets/' + ticketNo + '/resources') return data.resources;
    if (p === 'hall') return data.hall;
    if (p === 'materials') return data.materials;
    if (p === 'materials/tools') return { items: data.materials.tools };
    if (p === 'maintenance') return data.maintenance;
    if (p === 'profile') return data.profile;
    if (p === 'performance') return data.performance;
    if (p === 'schedule') return data.schedule;
    if (p === 'settings') return data.settings;
    if (p === 'feedbacks') return data.feedbacks;
    if (p === 'messages') return data.messages;
    if (p === 'notices') return data.notices;
    if (p === 'help/faq') return data.faq;
    if (p === 'service/messages') return data.serviceMessages;
  }

  if (method === 'POST') {
    if (p === 'auth/sms-code') return ok();
    if (p === 'auth/login') return { token: 'mock-worker-jwt-' + Date.now(), workerId: 1024 };
    if (p === 'auth/logout') return ok();
    if (p === 'tickets/' + ticketNo + '/accept') return ok('已领取，工单进入我的工单');
    if (p === 'tickets/' + ticketNo + '/checkin') return { checkedInAt: '10:05' };
    if (p === 'tickets/' + ticketNo + '/scan-bind') {
      // epc=EPC-0001 视为一致,其余为不一致;offline=true 走离线缓存分支
      return data.scanOk;
    }
    if (p === 'tickets/' + ticketNo + '/scan-abnormal') return ok('异常已上报，转调度复核');
    if (p === 'tickets/' + ticketNo + '/photos') return { photoId: 'p' + Date.now(), fileName: 'photo_0' + (Math.floor(Math.random() * 90) + 10) + '.jpg', linked: true };
    if (p === 'tickets/' + ticketNo + '/report') return ok('装维结果已提交');
    if (p === 'tickets/' + ticketNo + '/activate') return data.activationDone;
    if (p === 'tickets/' + ticketNo + '/sign') return ok('签收成功，激活完成，工单流转为完成');
    if (p === 'tickets/' + ticketNo + '/charge') return { payNo: 'PAY20250817' + (Math.floor(Math.random() * 900) + 100), receiptUrl: '/receipts/mock.pdf' };
    if (p === 'tickets/' + ticketNo + '/transfer') return ok('已退回调度池，等待重新派发');
    if (p === 'tickets/' + ticketNo + '/reschedule') return ok('改约申请已提交，客户将收到短信确认');
    if (p === 'tickets/' + ticketNo + '/rollback') return ok('已回退至上一完成环节');
    if (p === 'tickets/' + ticketNo + '/retry') return ok('已重新触发当前失败环节');
    if (p === 'tickets/' + ticketNo + '/complaint') return ok('投诉已登记，转客服回访闭环');
    if (p === 'tickets/' + ticketNo + '/repair-report') return { reviewPassed: true, status: 'CLOSED' };
    if (p === 'tickets/' + ticketNo + '/dismantle/scan') return { unbound: true, portReleased: true };
    if (p === 'tickets/' + ticketNo + '/replace') return ok('换件登记完成，旧件转返修，新件沿用绑定');
    if (seg[0] === 'assets' && seg[2] === 'return') return ok('已登记返库');
    if (p === 'materials/' + seg[1] + '/out') return ok('已扫码出库');
    if (p === 'materials/tools/' + seg[2] + '/borrow') return ok('已登记借用');
    if (p === 'materials/tools/' + seg[2] + '/give-back') return ok('已登记归还');
    if (p === 'hall/' + ticketNo + '/grab') return ok('抢单成功，已进入我的工单');
    if (p === 'schedule/clock') return { clockedAt: '09:00' };
    if (p === 'messages/read-all') return ok('全部标记已读');
    if (p === 'messages/clear') return ok('已清空已读消息');
    if (p === 'service/messages') return ok();
    if (p === 'safety/checks') return ok('安全确认已上报留痕');
  }

  if (method === 'PUT') {
    if (p === 'settings') return ok('接单设置已保存');
  }

  return null;
}

// 静态文件解析(防目录穿越)
function resolveStatic(pathname) {
  if (pathname.startsWith('/api/')) return null;
  let rel = pathname === '/' ? 'home.html' : pathname.replace(/^\/+/, '');
  rel = rel.split('?')[0];
  const file = path.resolve(STATIC_ROOT, rel);
  if (!file.startsWith(STATIC_ROOT)) return null;
  return { file: file, contentType: MIME[path.extname(file).toLowerCase()] || 'application/octet-stream' };
}

const server = http.createServer((req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const method = req.method.toUpperCase();

  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization');
  if (method === 'OPTIONS') { res.statusCode = 204; return res.end(); }

  if (method === 'GET') {
    const st = resolveStatic(u.pathname);
    if (st && fs.existsSync(st.file)) {
      res.setHeader('Content-Type', st.contentType);
      return fs.createReadStream(st.file).pipe(res);
    }
  }

  let body = '';
  req.on('data', (c) => { body += c; });
  req.on('end', () => {
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    const apiPath = u.pathname.replace(/^\/api\/worker\/v1/, '') || '/';
    let parsed = null;
    if (body) { try { parsed = JSON.parse(body); } catch (e) { parsed = null; } }
    let result = route(method, apiPath);
    const clean = apiPath.replace(/^\/+|\/+$/g, '');
    // GET /tickets?status=doing|todo|done|all 服务端筛选
    if (method === 'GET' && clean === 'tickets') {
      const st = u.searchParams.get('status') || 'all';
      const all = data.tickets.doing.concat(data.tickets.todo, data.tickets.done);
      const map = { doing: data.tickets.doing, todo: data.tickets.todo, done: data.tickets.done, all: all };
      result = { items: map[st] || all };
    }
    if (method === 'POST' && clean.endsWith('/scan-bind')) {
      result = scanBindResult(parsed);
    }
    if (result === null) {
      res.statusCode = 404;
      return res.end(JSON.stringify({ code: 40400, message: 'not found: ' + method + ' ' + u.pathname }));
    }
    res.statusCode = 200;
    res.end(JSON.stringify(result));
  });
});

server.listen(PORT, '0.0.0.0', () => {
  console.log('boss worker mock listening on http://0.0.0.0:' + PORT);
});
