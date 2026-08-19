#!/usr/bin/env node
// 对账脚本: api/openapi 三端契约 vs internal/app 实际注册路由。
// 解析 gin Group 前缀,把相对路径还原为完整路径;路径参数归一化为 {}。
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const NL = String.fromCharCode(10);
const Q = String.fromCharCode(34);
const METHODS = ['get','post','put','delete','patch'];

function parseSpec(file) {
  const lines = fs.readFileSync(file, 'utf8').split(NL);
  const routes = [];
  let inPaths = false;
  let curPath = null;
  for (const ln of lines) {
    if (ln.startsWith('paths:')) { inPaths = true; continue; }
    if (!inPaths) continue;
    if (/^[a-z]+:/.test(ln)) { inPaths = false; continue; }
    if (ln.startsWith('  /') && ln.endsWith(':')) { curPath = ln.trim().slice(0, -1); continue; }
    if (curPath) {
      const t = ln.trim();
      for (const m of METHODS) if (t === m + ':') routes.push(m.toUpperCase() + ' ' + curPath);
    }
  }
  return routes;
}

const dirs = ['api/openapi/admin', 'api/openapi/user', 'api/openapi/worker'];
let spec = [];
for (const d of dirs)
  for (const f of fs.readdirSync(d).filter(x => x.endsWith('.yaml')))
    spec.push(...parseSpec(path.join(d, f)));

// Go: 汇总源码,先收集 Group 定义,再解析路由调用
const goFiles = execSync('ls internal/app/*.go').toString().trim().split(NL);
const src = goFiles.map(f => ({ f, t: fs.readFileSync(f, 'utf8') }));
const groupOf = {}; // var -> prefix expr var name
const groupPrefix = {}; // var -> literal prefix
for (const { f, t } of src) {
  // 形如 x := y.Group("p") 或 x = y.Group("p")
  const re = new RegExp('([A-Za-z_][A-Za-z0-9_]*)\\s*:?=\\s*([A-Za-z_][A-Za-z0-9_]*)\\.Group\\((' + Q + ')([^' + Q + ']*)' + Q, 'g');
  let m;
  while ((m = re.exec(t))) groupOf[m[1]] = { parent: m[2], prefix: m[4], file: f };
}
function fullPrefix(v, seen) {
  seen = seen || new Set();
  if (seen.has(v)) return null;
  seen.add(v);
  const g = groupOf[v];
  if (!g) return v === 'r' || v === 'engine' ? '' : null; // 根引擎
  const p = fullPrefix(g.parent, seen);
  return p === null ? null : p + g.prefix;
}
const impl = [];
const unresolved = [];
for (const { f, t } of src) {
  for (const m of METHODS.map(x => x.toUpperCase())) {
    const re = new RegExp('([A-Za-z_][A-Za-z0-9_]*)\\.' + m + '\\((' + Q + ')([^' + Q + ']*)' + Q, 'g');
    let mm;
    while ((mm = re.exec(t))) {
      const p = fullPrefix(mm[1]);
      if (p === null) unresolved.push(m + ' ' + mm[3] + '  (var ' + mm[1] + ' @' + f + ')');
      else impl.push(m + ' ' + p + mm[3]);
    }
  }
}
const paramRe = /:[A-Za-z]+/g;
const braceRe = /\{[^}]+\}/g;
const implN = [...new Set(impl.map(l => { const sp = l.split(' '); let p = sp[1].replace(paramRe, '{}'); p = p.replace(/^\/api\/worker\/v1/, '').replace(/^\/api\/v1/, ''); return sp[0] + ' ' + p; }))];
const specN = new Set(spec.map(s => s.replace(braceRe, '{}')));
const implSet = new Set(implN.map(s => s.replace(braceRe, '{}')));
const missing = [...specN].filter(x => !implSet.has(x)).sort();
const extra = [...implSet].filter(x => !specN.has(x)).sort();
console.log('SPEC routes:', specN.size, ' IMPL routes:', implSet.size);
console.log('--- MISSING IN IMPL (' + missing.length + ') ---');
console.log(missing.join(NL) || '(none)');
console.log('--- EXTRA IN IMPL (' + extra.length + ') ---');
console.log(extra.join(NL) || '(none)');
const seen = new Map();
for (const r of implN) seen.set(r, (seen.get(r) || 0) + 1);
const dups = [...seen.entries()].filter(([, n]) => n > 1);
console.log('--- DUPLICATE REGISTRATIONS (' + dups.length + ') ---');
console.log(dups.map(([k, n]) => k + ' x' + n).join(NL) || '(none)');
console.log('--- UNRESOLVED VARS (' + unresolved.length + ') ---');
console.log([...new Set(unresolved)].join(NL) || '(none)');