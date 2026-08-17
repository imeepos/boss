// admin 路由 —— 按 api/openapi/admin.yaml 合并 admin/data/*.js 的路由表。
// 输入 pathname 已去 /api/admin/v1 前缀;query 为普通对象,body 为已解析 JSON。
// data 文件导出 { 'METHOD /path': 数据或函数 },新增域只需加 data 文件。
'use strict';

const fs = require('fs');
const path = require('path');

const DATA_DIR = path.join(__dirname, '../admin/data');

// 合并 data/*.js 的路由表 -> Map { 'GET /accounts': handler }
function loadRoutes() {
  const routes = new Map();
  // seed.js 是数据基座(非路由表),跳过
  const files = fs.readdirSync(DATA_DIR).filter((n) => n.endsWith('.js') && n !== 'seed.js').sort();
  for (const f of files) {
    const mod = require(path.join(DATA_DIR, f));
    for (const key of Object.keys(mod)) {
      if (routes.has(key)) throw new Error('duplicate route in mock data: ' + key);
      routes.set(key, mod[key]);
    }
  }
  return routes;
}

const routes = loadRoutes();

// 'GET /accounts/{accountId}' -> { method, rx, names }
function compile(key) {
  const [method, rawPattern] = key.split(' ');
  const pattern = rawPattern.replace(/^\/+|\/+$/g, '');
  const names = [];
  const rx = new RegExp(
    '^' + pattern.replace(/\{([a-zA-Z]+)\}/g, (_, n) => { names.push(n); return '([^/]+)'; }) + '$'
  );
  return { method, rx, names };
}

const compiled = [...routes.keys()].map((key) => ({ key, ...compile(key) }));

function route(method, pathname, query, body) {
  // 调试:GET _routes 列出全部可用路由
  if (method === 'GET' && pathname.replace(/^\/+|\/+$/g, '') === '_routes') {
    return { routes: [...routes.keys()].sort() };
  }
  const p = pathname.replace(/^\/+|\/+$/g, '');
  for (const c of compiled) {
    if (c.method !== method) continue;
    const hit = c.rx.exec(p);
    if (!hit) continue;
    const params = {};
    c.names.forEach((n, i) => (params[n] = decodeURIComponent(hit[i + 1])));
    const handler = routes.get(c.key);
    return typeof handler === 'function' ? handler({ params, query, body }) : handler;
  }
  return null;
}

module.exports = { route };
