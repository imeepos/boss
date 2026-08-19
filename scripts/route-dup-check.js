const fs = require('fs');
const { execSync } = require('child_process');
const NL = String.fromCharCode(10);
const METHODS = ['get','post','put','delete','patch'];
const goFiles = execSync('ls internal/app/*.go').toString().trim().split(NL).filter(f => !f.includes('_test'));
const groupOf = {};
for (const f of goFiles) {
  const t = fs.readFileSync(f, 'utf8');
  const re = /([A-Za-z_][A-Za-z0-9_]*)\s*:?=[^=]*([A-Za-z_][A-Za-z0-9_]*)\.Group\("([^"]*)"/g;
  let m; while ((m = re.exec(t))) groupOf[m[1]] = { p: m[2], pre: m[3] };
}
function full(v, seen) {
  seen = seen || new Set(); if (seen.has(v)) return null; seen.add(v);
  const g = groupOf[v]; if (!g) return '';
  const p = full(g.p, seen); return p === null ? null : p + g.pre;
}
const routes = [];
for (const f of goFiles) {
  const t = fs.readFileSync(f, 'utf8');
  for (const m of METHODS.map(x => x.toUpperCase())) {
    const re = new RegExp('([A-Za-z_][A-Za-z0-9_]*)\\.' + m + '\\("([^"]+)"', 'g');
    let mm; while ((mm = re.exec(t))) {
      const p = full(mm[1]); if (p !== null) routes.push({ m, path: p + mm[2], f });
    }
  }
}
const seen = new Map();
for (const r of routes) {
  const k = r.m + ' ' + r.path.replace(/:[A-Za-z]+/g, ':x');
  if (!seen.has(k)) seen.set(k, []);
  seen.get(k).push(r.f.replace('internal/app/', ''));
}
for (const [k, fs] of seen) if (fs.length > 1) console.log(k, '<=', fs.join(' + '));