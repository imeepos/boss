// mock server —— 用户端假数据 HTTP 服务(零依赖 Node 标准库)。
// 路由复用 routes/user.js,静态托管 docs/user;启动: node api/mock/server.js
// 默认 0.0.0.0:8090,MOCK_PORT 覆盖;跨域放行供 file:// 页面调试。
'use strict';

const http = require('http');
const path = require('path');
const { URL } = require('url');
const { setCors, readBody, send404, serveStatic } = require('./lib/http.js');
const userRoute = require('./routes/user.js');

const PORT = Number(process.env.MOCK_PORT || 8090);
const STATIC_ROOT = path.resolve(__dirname, '../../docs/user');

const server = http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const method = req.method.toUpperCase();

  setCors(res);
  if (method === 'OPTIONS') {
    res.statusCode = 204;
    return res.end();
  }

  if (serveStatic(req, res, u.pathname, STATIC_ROOT)) return undefined;

  if (u.pathname.startsWith('/api/v1')) {
    const apiPath = u.pathname.replace(/^\/api\/v1/, '') || '/';
    const result = userRoute.route(method, apiPath);
    if (result === null) return send404(res, method, u.pathname);
    res.statusCode = 200;
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    return res.end(JSON.stringify(result));
  }

  return send404(res, method, u.pathname);
});

server.listen(PORT, '0.0.0.0', () => {
  console.log(`boss user mock listening on http://0.0.0.0:${PORT}`);
});
