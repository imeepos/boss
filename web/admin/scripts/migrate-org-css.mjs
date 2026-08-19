// org.css 迁移脚本:把 org-* 类名与元素级 th/td 样式替换为 tailwind 原子类。
// 只处理包含 org- 类的 .tsx 文件;幂等(已迁移文件无 org- 前缀不再匹配)。
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const ROOT = 'src'
const TH = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const TD = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'

// 字符串类名替换:先替换组合/长串,再替换短串(org-btn 会命中 org-btn-primary)。
const CLASS_MAP = [
  ['org-btn org-btn-primary', 'h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]'],
  ['org-page-head', 'mb-4'],
  ['org-page-title', 'm-0 text-xl font-bold text-[var(--shell-heading)]'],
  ['org-page-desc', 'mt-1 text-xs text-[var(--shell-crumb-text)]'],
  ['org-card-title', 'px-4 pt-3.5 text-[15px] font-semibold text-[var(--shell-heading)]'],
  ['org-card-extra', 'ml-2 text-xs font-normal text-[var(--shell-group-title)]'],
  ['org-card', 'mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'],
  ['org-toolbar', 'flex flex-wrap items-center gap-2 p-4'],
  ['org-input', 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'],
  ['org-select', 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]'],
  ['org-btn-primary', 'h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]'],
  ['org-btn', 'h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]'],
  ['org-table-wrap', 'overflow-x-auto px-4 pb-4'],
  ['org-table', 'w-full border-collapse text-[13px] text-[var(--shell-content-text)]'],
  ['org-empty', 'py-8 text-center text-[13px] text-[var(--shell-group-title)]'],
  ['org-footer', 'flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]'],
  ['org-act', 'inline-flex items-center'],
  ['org-error', 'mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'],
  ['org-form', 'flex flex-col gap-3.5'],
  ['org-field', 'flex flex-col gap-1.5'],
  ['org-detail-list', 'flex flex-col gap-2.5'],
  ['org-detail-item', 'flex gap-3 text-[13px]'],
  ['className="sep"', 'className="text-[var(--shell-side-border)]"'],
  ['className="req"', 'className="mr-0.5 text-[var(--color-danger)]"'],
  ['className="hint"', 'className="text-[11px] text-[var(--shell-input-placeholder)]"'],
  ['className="k"', 'className="w-24 flex-none text-[var(--shell-group-title)]"'],
  ['className="v"', 'className="break-all text-[var(--shell-content-text)]"'],
  ['className="mono"', 'className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs"'],
  ['post-roles', 'flex flex-wrap gap-2'],
  ['post-role-chip', 'inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-content-text)] [&:has(input:checked)]:border-[var(--shell-fab-bg)] [&:has(input:checked)]:bg-[var(--shell-fab-bg)] [&:has(input:checked)]:text-[var(--shell-fab-icon)]'],
  ['apikey-plain-banner', 'fixed bottom-6 right-6 z-[60] flex max-w-[420px] flex-col gap-2 rounded-md border border-[var(--shell-fab-bg)] bg-[var(--shell-card-bg)] p-4 text-[13px] text-[var(--shell-content-text)] shadow-[0_6px_24px_rgba(0,0,0,0.18)]'],
]

function walk(dir, out = []) {
  for (const e of readdirSync(dir)) {
    const p = join(dir, e)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (p.endsWith('.tsx')) out.push(p)
  }
  return out
}

function hasOrg(file) {
  const src = readFileSync(file, 'utf8')
  return /org-(?:page-head|page-title|page-desc|card|card-title|card-extra|toolbar|input|select|btn|btn-primary|table|table-wrap|empty|footer|act|error|form|field|detail-list|detail-item)|post-role|apikey-plain-banner|className="sep"|className="req"|className="hint"/.test(src)
}

let touched = 0
for (const file of walk(ROOT)) {
  if (!hasOrg(file)) continue
  let src = readFileSync(file, 'utf8')
  // 元素级 th/td:仅在含 org-table 的文件中,给无 className 的 th/td 补类。
  if (src.includes('org-table')) {
    src = src.replace(/<th(?![^>]*className)([^>]*)>/g, (m, rest) => `<th${rest} className="${TH}">`)
    src = src.replace(/<td(?![^>]*className)([^>]*)>/g, (m, rest) => `<td${rest} className="${TD}">`)
  }
  for (const [from, to] of CLASS_MAP) {
    src = src.split(from).join(to)
  }
  // 移除 org.css 导入(文件即将删除)
  src = src.replace(/^import\s+['"][^'"]*org\.css['"]\s*;?$/gm, '')
  writeFileSync(file, src)
  touched++
  console.log('touched', file)
}
console.log('total', touched)
