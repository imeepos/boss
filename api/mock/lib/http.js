// mock 公共 HTTP 工具 —— CORS / 请求体读取 / JSON 响应 / 静态文件(防穿越)。
'use strict';

const fs = require('fs');
const path = require('path');

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.svg': 'image/svg+xml',
};

function setCors(res) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Authorization');
}

function readBody(req) {
  return new Promise((resolve) => {
    let body = '';
    req.on('data', (c) => { body += c; });
    req.on('end', () => {
      if (!body) return resolve(null);
      try { resolve(JSON.parse(body)); } catch (e) { resolve(null); }
    });
  });
}

function sendJson(res, status, value) {
  res.statusCode = status;
  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  res.end(JSON.stringify(value));
}

function send404(res, method, pathname) {
  sendJson(res, 404, { code: 40400, message: 'not found: ' + method + ' ' + pathname });
}

// 静态文件:pathname 映射到 rootDir 下文件;'/api' 前缀不走静态。
// 返回 { file, contentType } 或 null。
function resolveStatic(pathname, rootDir) {
  if (pathname.startsWith('/api/')) return null;
  let rel = pathname === '/' ? 'home.html' : pathname.replace(/^\/+/, '');
  rel = rel.split('?')[0];
  const file = path.resolve(rootDir, rel);
  if (!file.startsWith(rootDir)) return null; // 防 ../ 穿越
  return { file, contentType: MIME[path.extname(file).toLowerCase()] || 'application/octet-stream' };
}

// 尝试按静态文件响应;命中并存在则写响应返回 true。
function serveStatic(req, res, pathname, rootDir) {
  if (req.method.toUpperCase() !== 'GET') return false;
  const st = resolveStatic(pathname, rootDir);
  if (!st || !fs.existsSync(st.file)) return false;
  res.setHeader('Content-Type', st.contentType);
  fs.createReadStream(st.file).pipe(res);
  return true;
}

module.exports = { setCors, readBody, sendJson, send404, serveStatic };
