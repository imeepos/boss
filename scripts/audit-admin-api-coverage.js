// 核查 docs/admin 页面脚本中调用的 API 路径是否在 api/mock/routes/admin.js 中有对应路由。
// 用法: node scripts/audit-admin-api-coverage.js
const fs = require('fs');
const path = require('path');
const ADMIN = path.join(__dirname, '..', 'docs', 'admin');
const routesSrc = fs.readFileSync(path.join(__dirname, '..', 'api', 'mock', 'routes', 'admin.js'), 'utf8');

// 收集 mock 路由(形如 r('GET', '/orders') 或 route('GET', ...))
const routePaths = new Set();
for (const m of routesSrc.matchAll(/['"`](\/[a-z0-9\-_/:.{}]+)['"]/gi)) {
  routePaths.add(m[1]);
}

// 收集 api.js 中声明的路径模板
const apiSrc = fs.readFileSync(path.join(ADMIN, 'api.js'), 'utf8');
const declared = new Set();
for (const m of apiSrc.matchAll(/(?:get|post|put|del)\(\s*'([^']+)'/g)) declared.add(m[1]);

// 收集页面中直接 API.get/post/put/del 调用的字面路径
const pageCalls = new Set();
for (const f of fs.readdirSync(ADMIN).filter((x) => x.endsWith('.html'))) {
  const src = fs.readFileSync(path.join(ADMIN, f), 'utf8');
  for (const m of src.matchAll(/API\.(?:get|post|put|del)\(\s*'([^']+)'/g)) {
    pageCalls.add(m[1]);
  }
}

function coverable(p) {
  // 将 '/a/' + id + '/toggle' 之类的拼接与模板统一成段比较
  const norm = p.replace(/'/g, '');
  for (const r of routePaths) {
    if (r === norm) return true;
    // 前缀段匹配:模板含 :id 或 {id}
    const rSeg = r.split('/').filter(Boolean);
    const pSeg = norm.split('/').filter(Boolean);
    if (rSeg.length !== pSeg.length) continue;
    const ok = rSeg.every((s, i) => s === pSeg[i] || /^[:{]/.test(s));
    if (ok) return true;
  }
  return false;
}

const missing = [];
for (const p of [...declared, ...pageCalls].sort()) {
  if (!coverable(p)) missing.push(p);
}
console.log('api.js 声明路径数:', declared.size);
console.log('页面直调路径数:', pageCalls.size);
console.log('mock 路由字面量数:', routePaths.size);
if (missing.length) {
  console.log('\n疑似未被 mock 覆盖的路径:');
  missing.forEach((p) => console.log('  - ' + p));
} else {
  console.log('\n全部路径均有 mock 路由覆盖(按字面/段匹配)');
}
