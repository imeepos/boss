// 核查 docs/admin 页面/api.js 调用的 API 路径是否被 mock(api/mock/admin/data/*.js)覆盖。
// 用法: node scripts/audit-admin-api-coverage.js
const fs = require('fs');
const path = require('path');
const ADMIN = path.join(__dirname, '..', 'docs', 'admin');
const adminRoute = require(path.join(__dirname, '..', 'api', 'mock', 'routes', 'admin.js'));

const mockRoutes = adminRoute.route('GET', '_routes').routes; // ['GET /accounts', ...]

// 收集 api.js 声明: method + path 模板
const apiSrc = fs.readFileSync(path.join(ADMIN, 'api.js'), 'utf8');
const declared = [];
for (const m of apiSrc.matchAll(/\b(get|post|put|del)\(\s*((?:[^,()]|'[^']*')+?)\s*[,)]/g)) {
  const p = tplOf(m[2]);
  if (p) declared.push({ method: m[1].toUpperCase().replace('DEL', 'DELETE'), path: p });
}

// 页面内直调 API.get/post/put/del('...'),含 '.../' + x + '/suffix' 拼接
const pageCalls = [];
for (const f of fs.readdirSync(ADMIN).filter((x) => x.endsWith('.html'))) {
  const src = fs.readFileSync(path.join(ADMIN, f), 'utf8');
  for (const m of src.matchAll(/API\.(get|post|put|del)\(\s*((?:[^,()]|'[^']*')+?)\s*[,)]/g)) {
    const p = tplOf(m[2]);
    if (p) pageCalls.push({ method: m[1].toUpperCase().replace('DEL', 'DELETE'), path: p, file: f });
  }
}


// 将 '+ ' 拼接表达式(字符串字面量 + 变量)归一为 '/a/{}/b' 模板;无字面量返回 null。
function tplOf(expr) {
  const parts = expr.split('+').map((s) => s.trim()).filter(Boolean);
  let out = '', lit = 0;
  for (const part of parts) {
    const m = part.match(/^'([^']*)'$/);
    if (m) { out += m[1]; lit++; }
    else out += '{}';
  }
  return lit ? out : null;
}

function covered(method, pathTemplate) {
  // pathTemplate 中 {} 表示动态段
  const want = pathTemplate.replace(/^\//, '').replace(/\/$/, '');
  const wSeg = want.split('/').filter(Boolean);
  for (const key of mockRoutes) {
    const [m, raw] = key.split(' ');
    if (m !== method) continue;
    const rSeg = raw.replace(/^\//, '').replace(/\/$/, '').split('/').filter(Boolean);
    if (rSeg.length !== wSeg.length) continue;
    const ok = rSeg.every((s, i) => s === wSeg[i] || /^\{.+\}$/.test(s) || wSeg[i] === '{}');
    if (ok) return key;
  }
  return null;
}

let missing = 0;
for (const c of [...declared, ...pageCalls]) {
  const hit = covered(c.method, c.path);
  if (!hit) {
    missing++;
    console.log(`缺路由: ${c.method} ${c.path}${c.file ? '  (' + c.file + ')' : ''}`);
  }
}
console.log(`\nmock 路由数: ${mockRoutes.length};声明+页面调用数: ${declared.length + pageCalls.length};缺失: ${missing}`);
