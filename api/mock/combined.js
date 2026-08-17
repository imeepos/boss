// combined mock —— 单进程双前缀聚合 mock 服务(零依赖 Node 标准库)。
// /api/v1        → 用户端(api/openapi/user.yaml)
// /api/worker/v1 → 师傅端(api/openapi/worker.yaml)
// 静态托管: /          → docs/user,/worker/  → docs/worker
// 启动: node api/mock/combined.js  (默认 0.0.0.0:8090,MOCK_PORT 覆盖)
'use strict';

const http = require('http');
const path = require('path');
const { URL } = require('url');
const { setCors, readBody, send404, serveStatic } = require('./lib/http.js');
const userRoute = require('./routes/user.js');
const workerRoute = require('./routes/worker.js');

const PORT = Number(process.env.MOCK_PORT || 8090);
const USER_ROOT = path.resolve(__dirname, '../../docs/user');
const WORKER_ROOT = path.resolve(__dirname, '../../docs/worker');

const server = http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const method = req.method.toUpperCase();

  setCors(res);
  if (method === 'OPTIONS') {
    res.statusCode = 204;
    return res.end();
  }

  // API 分流:先匹配师傅端前缀,再匹配用户端前缀。
  let result = null;
  if (u.pathname.startsWith('/api/worker/v1')) {
    const apiPath = u.pathname.replace(/^\/api\/worker\/v1/, '') || '/';
    const body = method === 'POST' || method === 'PUT' ? await readBody(req) : null;
    result = workerRoute.route(method, apiPath, u.searchParams, body);
    if (result === null) return send404(res, method, u.pathname);
  } else if (u.pathname.startsWith('/api/v1')) {
    const apiPath = u.pathname.replace(/^\/api\/v1/, '') || '/';
    result = userRoute.route(method, apiPath);
    if (result === null) return send404(res, method, u.pathname);
  } else {
    // 静态页面:/worker/* 托管 docs/worker,其余托管 docs/user。
    const root = u.pathname.startsWith('/worker') ? WORKER_ROOT : USER_ROOT;
    const stPath = root === WORKER_ROOT
      ? u.pathname.replace(/^\/worker/, '') || '/'
      : u.pathname;
    if (serveStatic(req, res, stPath, root)) return undefined;
    return send404(res, method, u.pathname);
  }

  res.statusCode = 200;
  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  res.end(JSON.stringify(result));
});

server.listen(PORT, '0.0.0.0', () => {
  console.log(`boss combined mock listening on http://0.0.0.0:${PORT}`);
  console.log(`  user   prefix : http://0.0.0.0:${PORT}/api/v1`);
  console.log(`  worker prefix : http://0.0.0.0:${PORT}/api/worker/v1`);
});
