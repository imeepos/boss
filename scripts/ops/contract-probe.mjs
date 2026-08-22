#!/usr/bin/env node
// Q4 契约对账探针: api/openapi 契约路由 -> 102 部署环境实测。
// 判定: 非 404 即路由已部署(401/403/4xx 均证明路由存在); 404 = 契约有而环境无 = 红项。
// 用法: node scripts/ops/contract-probe.mjs [--base http://192.168.0.102:28080]
import fs from 'node:fs';
import path from 'node:path';

const BASE = (process.argv.includes('--base')
  ? process.argv[process.argv.indexOf('--base') + 1] : 'http://192.168.0.102:28080');
const METHODS = ['get', 'post', 'put', 'delete', 'patch'];
const PORTALS = [
  { name: 'admin', prefix: '/api/admin/v1', dir: 'api/openapi/admin' },
  { name: 'user', prefix: '/api/user/v1', dir: 'api/openapi/user' },
  { name: 'worker', prefix: '/api/worker/v1', dir: 'api/openapi/worker' },
];

function parseFile(file) {
  const routes = [];
  let curPath = null, curMethod = null;
  for (const ln of fs.readFileSync(file, 'utf8').split('\n')) {
    if (/^  '?\//.test(ln)) { curPath = ln.trim().replace(/^'|'$/g, '').slice(0, -1); curMethod = null; continue; }
    const t = ln.trim();
    const ind = ln.length - ln.replace(/^ +/, '').length;
    if (curPath && ind === 4 && t.endsWith(':')) {
      const m = t.slice(0, -1);
      if (METHODS.includes(m)) { curMethod = m; routes.push({ method: m.toUpperCase(), path: curPath }); }
    }
  }
  return routes;
}

function concrete(p) {
  return p.replace(/\{[^}]+\}/g, '1'); // 路径参数以 1 代入,只验证路由存在性
}

async function probe(method, url) {
  try {
    const res = await fetch(url, { method, signal: AbortSignal.timeout(8000) });
    return res.status;
  } catch { return 'ERR'; }
}

const report = { base: BASE, total: 0, present: 0, missing: [], errors: [] };
for (const p of PORTALS) {
  const dir = path.resolve(p.dir);
  if (!fs.existsSync(dir)) continue;
  for (const f of fs.readdirSync(dir).filter(x => x.endsWith('.yaml'))) {
    for (const r of parseFile(path.join(dir, f))) {
      const url = BASE + p.prefix + concrete(r.path);
      const code = await probe(r.method, url);
      report.total++;
      if (code === 404) report.missing.push(`${p.name} ${r.method} ${r.path}`);
      else if (code === 'ERR') report.errors.push(`${p.name} ${r.method} ${r.path}`);
      else report.present++;
    }
  }
}
console.log(JSON.stringify({
  base: report.base, total: report.total, present: report.present,
  missingCount: report.missing.length, errorCount: report.errors.length,
  missing: report.missing, errors: report.errors,
}, null, 2));
process.exit(report.missing.length + report.errors.length > 0 ? 1 : 0);
