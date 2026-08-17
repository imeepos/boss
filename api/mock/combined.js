// combined mock —— 单进程三前缀聚合 mock 服务(零依赖 Node 标准库)。
// /api/v1         → 用户端(api/openapi/user.yaml)
// /api/worker/v1  → 师傅端(api/openapi/worker.yaml)
// /api/admin/v1   → 管理后台(api/openapi/admin.yaml)
// 静态托管: /          → docs/user,/worker/ → docs/worker,/admin/ → docs/admin
// 启动: node api/mock/combined.js  (默认 0.0.0.0:8090,MOCK_PORT 覆盖)
'use strict';

const http = require('http');
const path = require('path');
const { URL } = require('url');
const { setCors, readBody, send404, serveStatic } = require('./lib/http.js');
const userRoute = require('./routes/user.js');
const workerRoute = require('./routes/worker.js');
const adminRoute = require('./routes/admin.js');

const PORT = Number(process.env.MOCK_PORT || 8090);
const USER_ROOT = path.resolve(__dirname, '../../docs/user');
const WORKER_ROOT = path.resolve(__dirname, '../../docs/worker');
const ADMIN_ROOT = path.resolve(__dirname, '../../docs/admin');

const PREFIXES = [
  { prefix: '/api/worker/v1', route: (m, p, q, b) => workerRoute.route(m, p, q, b) },
  { prefix: '/api/admin/v1', route: (m, p, q, b) => adminRoute.route(m, p, Object.fromEntries(q), b || {}) },
  { prefix: '/api/v1', route: (m, p) => userRoute.route(m, p) },
];

// 静态页面分流:/admin、/worker 前缀各映射对应 docs 目录,其余托管 docs/user。
function servePage(req, res, u) {
  if (u.pathname === '/admin' || u.pathname.startsWith('/admin/')) {
    let p = u.pathname.replace(/^\/admin/, '') || '/';
    if (p === '/') p = '/dashboard.html';
    return serveStatic(req, res, p, ADMIN_ROOT);
  }
  if (u.pathname === '/worker' || u.pathname.startsWith('/worker/')) {
    const p = u.pathname.replace(/^\/worker/, '') || '/';
    return serveStatic(req, res, p, WORKER_ROOT);
  }
  return serveStatic(req, res, u.pathname, USER_ROOT);
}

const server = http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const method = req.method.toUpperCase();

  setCors(res);
  if (method === 'OPTIONS') {
    res.statusCode = 204;
    return res.end();
  }

  // API 分流:按前缀匹配,路径去前缀后交给对应路由。
  for (const { prefix, route } of PREFIXES) {
    if (!u.pathname.startsWith(prefix)) continue;
    const apiPath = u.pathname.slice(prefix.length) || '/';
    const body = method === 'POST' || method === 'PUT' ? await readBody(req) : null;
    const result = route(method, apiPath, u.searchParams, body);
    if (result === null) return send404(res, method, u.pathname);
    res.statusCode = 200;
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    return res.end(JSON.stringify(result));
  }

  if (servePage(req, res, u)) return undefined;
  return send404(res, method, u.pathname);
});

server.listen(PORT, '0.0.0.0', () => {
  console.log(`boss combined mock listening on http://0.0.0.0:${PORT}`);
  console.log(`  user   prefix : http://0.0.0.0:${PORT}/api/v1`);
  console.log(`  worker prefix : http://0.0.0.0:${PORT}/api/worker/v1`);
  console.log(`  admin  prefix : http://0.0.0.0:${PORT}/api/admin/v1`);
});
