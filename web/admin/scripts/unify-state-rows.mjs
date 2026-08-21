// 一次性迁移脚本:把页面内联 loading/空数据样式统一替换为 feedback 组件。
// 用法: node scripts/unify-state-rows.mjs [--dry]
import { readFileSync, writeFileSync, readdirSync } from 'node:fs'

const dry = process.argv.includes('--dry')
function walk(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap(e => {
    const p = dir + '/' + e.name
    return e.isDirectory() ? walk(p) : e.name.endsWith('.tsx') && !e.name.includes('.test.') && readFileSync(p, 'utf8').includes('py-8 text-center') ? [p] : []
  })
}
const files = walk('src/pages')

const TD = 'h-11 px-3 whitespace-nowrap border-b border-\\[var\\(--shell-side-border\\)\\] text-\\[var\\(--shell-content-text\\)\\] hover:bg-\\[var\\(--shell-menu-hover-bg\\)\\]'
const DIV = 'py-8 text-center text-\\[13px\\] text-\\[var\\(--shell-group-title\\)\\]'
const reRow = new RegExp(`\\{([\\w!.]+(?:\\.length)?) && <tr><td colSpan=\\{(\\d+)\\} className="${TD}"><div className="${DIV}">\\{([^{}]+)\\}</div></td></tr>\\}`, 'g')
const reDiv = new RegExp(`<div className="${DIV}">\\{([^{}]+)\\}</div>`, 'g')

let report = []
for (const f of files) {
  const path = f
  let src = readFileSync(path, 'utf8')
  let rows = 0, divs = 0
  src = src.replace(reRow, (_m, cond, span, text) => { rows++; return `{${cond} && <TableStateRow colSpan={${span}} loading={busy} text={${text}} />}` })
  src = src.replace(reDiv, (_m, text) => { divs++; return `<EmptyState text={${text}} />` })
  if (!rows && !divs) { report.push(`SKIP  ${f}`); continue }
  // 无 busy 变量的文件去掉 loading prop
  if (!/const \[busy,/.test(src)) src = src.replace(/ loading=\{busy\}/g, '')
  // 注入 import
  const need = ['TableStateRow', 'EmptyState'].filter(n => new RegExp(`<${n}[ />]`).test(src) && !new RegExp(`\\b${n}\\b.*from|import.*\\b${n}\\b`).test(src.split('\n').filter(l => l.startsWith('import')).join('\n')))
  if (need.length) {
    const segs = f.split('/').length - 1 // 相对 src/ 的深度
    const rel = '../'.repeat(segs - 1) + 'components/business'
    const lines = src.split('\n')
    let last = 0
    lines.forEach((l, i) => { if (l.startsWith('import ')) last = i })
    lines.splice(last + 1, 0, `import { ${need.join(', ')} } from '${rel}'`)
    src = lines.join('\n')
  }
  if (!dry) writeFileSync(path, src)
  report.push(`OK    ${f}  rows=${rows} divs=${divs}${/const \[busy,/.test(src) ? '' : ' (no-busy)'}`)
}
console.log(report.join('\n'))
console.log(`\n${files.length} files scanned`)
