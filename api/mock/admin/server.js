// mock server —— 管理后台(admin 端)假数据 HTTP 服务(零依赖 Node 标准库)。
// 严格按 api/openapi/admin.yaml 的方法/路径返回假数据。
// 启动: node api/mock/admin/server.js  (默认 0.0.0.0:8092,局域网可访问)
// 跨域已放行;同时托管 docs/admin/ 静态页面。
// 假数据按域拆分至 data/*.js,每个文件导出 { 'METHOD /path': 数据或函数 },
// server 自动合并,新增域只需增加 data 文件,无需改本文件。
'use strict';

const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = Number(process.env.MOCK_ADMIN_PORT || 8092);
const PREFIX = '/api/admin/v1';
const DATA_DIR = path.join(__dirname, 'data');
const STATIC_ROOT = path.resolve(__dirname, '../../../docs/admin');

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
};

const ok = () => ({ code: 0, message: 'success' });

// 合并 data/*.js 的路由表 -> { 'GET /accounts': handler }
function loadRoutes() {
  const routes = new Map();
  for (const f of fs.readdirSync(DATA_DIR).filter((n) => n.endsWith('.js')).sort()) {
    const mod = require(path.join(DATA_DIR, f));
    for (const key of Object.keys(mod)) {
      if (routes.has(key)) throw new Error('duplicate route in mock data: ' + key);
      routes.set(key, mod[key]);
    }
  }
  return routes;
}

// 'GET /accounts/{accountId}' -> /^GET \/accounts\/([^/]+)$/ + 参数名列表
function compile(key) {
  const [method, rawPattern] = key.split(' ');
  const pattern = rawPattern.replace(/^\/+|\/+$/g, '');
  const names = [];
  const rx = new RegExp(
    '^' + pattern.replace(/\{([a-zA-Z]+)\}/g, (_, n) => { names.push(n); return '([^/]+)'; }) + '$'
  );
  return { method, rx, names };
}

function route(routes, method, apiPath, query, body) {
  const p = apiPath.replace(/^\/+|\/+$/g, '');
  for (const [key, handler] of routes) {
    const { method: m, rx, names } = compile(key);
    if (m !== method) continue;
    const hit = rx.exec(p);
    if (!hit) continue;
    const params = {};
    names.forEach((n, i) => (params[n] = decodeURIComponent(hit[i + 1])));
    return typeof handler === 'function' ? handler({ params, query, body }) : handler;
  }
  return null;
}

// 静态文件解析:URL 路径映射到 docs/admin 下文件,防目录穿越。
function resolveStatic(pathname) {
  if (pathname.startsWith('/api/')) return null;
  const rel = (pathname === '/' ? 'dashboard.html' : pathname.replace(/^\/+/, '')).split('?')[0];
  const file = path.resolve(STATIC_ROOT, rel);
  if (!file.startsWith(STATIC_ROOT)) return null;
  return { file, contentType: MIME[path.extname(file).toLowerCase()] || 'application/octet-stream' };
}

function readBody(req) {
  return new Promise((resolve) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => {
      try { resolve(raw ? JSON.parse(raw) : {}); } catch { resolve({}); }
    });
  });
}

const routes = loadRoutes();

const server = http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const method = req.method.toUpperCase();

  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization');

  if (method === 'OPTIONS') { res.statusCode = 204; return res.end(); }

  // 调试:列出全部可用路由
  if (method === 'GET' && u.pathname === PREFIX + '/_routes') {
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    return res.end(JSON.stringify({ routes: [...routes.keys()].sort() }, null, 2));
  }

  if (method === 'GET') {
    const st = resolveStatic(u.pathname);
    if (st && fs.existsSync(st.file) && fs.statSync(st.file).isFile()) {
      res.setHeader('Content-Type', st.contentType);
      return fs.createReadStream(st.file).pipe(res);
    }
  }

  if (!u.pathname.startsWith(PREFIX)) {
    res.statusCode = 404;
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    return res.end(JSON.stringify({ code: 40400, message: 'not found: ' + u.pathname }));
  }

  const query = Object.fromEntries(u.searchParams);
  const body = method === 'GET' ? {} : await readBody(req);
  const result = route(routes, method, u.pathname.slice(PREFIX.length), query, body);
  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  if (result === null) {
    res.statusCode = 404;
    return res.end(JSON.stringify({ code: 40400, message: 'not found: ' + method + ' ' + u.pathname }));
  }
  res.statusCode = 200;
  res.end(JSON.stringify(result));
});

server.listen(PORT, '0.0.0.0', () => {
  console.log(`boss admin mock listening on http://0.0.0.0:${PORT} (${routes.size} routes)`);
});
