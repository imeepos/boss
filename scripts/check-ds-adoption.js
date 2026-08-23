// Admin 设计系统采用率审计(docs/admin/page-patterns.md §5)。
// 用法: node scripts/check-ds-adoption.js [阈值,缺省 0.6]
// 阈值棘轮:当前 60%,随存量收敛逐步上调,目标 80%(page-patterns.md §0)。
// 规则:pages/<模块> 下引用 components/business 或 components/ui 的 .tsx 占比 ≥ 阈值;
// 模块页数 < 3 不判(样本过小);任一大模块低于阈值退出非零。
const fs = require('fs')
const path = require('path')

const PAGES = path.join(__dirname, '..', 'web', 'admin', 'src', 'pages')
const THRESHOLD = parseFloat(process.argv[2] || '0.6')
const MIN_PAGES = 3
// home=公开官网落地页(自有视觉体系),error/login/placeholder=壳页,均非 Admin 业务页。
const EXEMPT = new Set(['home', 'error', 'login', 'placeholder', 'auth-ads.tsx', 'auth-shell.tsx'])

const dirs = fs.readdirSync(PAGES, { withFileTypes: true })
  .filter((d) => d.isDirectory() && !EXEMPT.has(d.name))

let bad = 0
const rows = []
for (const d of dirs) {
  const files = fs.readdirSync(path.join(PAGES, d.name))
    .filter((f) => f.endsWith('.tsx') && !EXEMPT.has(f) && !f.endsWith('.test.tsx'))
  if (files.length < MIN_PAGES) continue
  const uses = files.filter((f) => {
    const src = fs.readFileSync(path.join(PAGES, d.name, f), 'utf8')
    return src.includes('components/business') || src.includes("components/ui") ||
      src.includes('../business') || src.includes('../ui')
  }).length
  const pct = uses / files.length
  rows.push({ mod: d.name, pages: files.length, uses, pct })
  if (pct < THRESHOLD) bad++
}

rows.sort((a, b) => a.pct - b.pct)
for (const r of rows) {
  const mark = r.pct < THRESHOLD ? 'FAIL' : 'ok'
  console.log(`${mark}  ${r.mod.padEnd(14)} ${String(r.uses).padStart(2)}/${String(r.pages).padStart(2)}  ${(r.pct * 100).toFixed(0)}%`)
}
console.log(`\nthreshold=${(THRESHOLD * 100).toFixed(0)}%  below=${bad}`)
if (bad > 0) {
  console.log('采用率低于阈值的模块见 FAIL 行;规则与收敛清单见 docs/admin/page-patterns.md')
  process.exit(1)
}
