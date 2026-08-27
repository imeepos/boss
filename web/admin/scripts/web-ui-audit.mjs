// UI 治理门禁(构建前置):① 幽灵 CSS 令牌——var(--x) 引用集合对 `--x:` 定义集合做差集,
// 未定义令牌在计算值阶段静默失效(亮暗主题双坏无报错),只能机械拦截;
// ② 菜单图标——menu.def.ts 的 key 必须有 public/icons/items/<key>.svg(MaskIcon 按 key 取,缺文件即菜单空位);
// ③ 浮层裸 z 值——业务浮层唯一合法出口是 styles.css --z-* 令牌(z-* 语义类 / var(--z-*));
//    chrome 白名单(外壳布局/登录页/营销壳,非业务浮层)按 文件→允许裸值 放行,禁止为业务浮层加白。
// 白名单仅放行运行时注入的变量(Radix 弹层定位/Sidebar 拖宽/MaskIcon 遮罩),补注释即可追加。
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const SRC = join(root, 'src')
const ALLOWED = ['--icon-url', '--side-w']
const ALLOWED_PREFIX = '--radix-'

/* chrome 层裸 z 白名单:外壳自身布局(侧栏手柄/移动端抽屉/FAB 悬浮球/登录水印/营销页) */
const Z_WHITELIST = {
  'layouts/AdminLayout.tsx': [30],
  'layouts/Fab.tsx': [20],
  'layouts/UCenterLayout.tsx': [20],
  'layouts/Sidebar.tsx': [4, 5, 40],
  'pages/auth-shell.tsx': [1],
  'pages/home/Hero.tsx': [10],
  'pages/home/TopNav.tsx': [20],
}

/** 裸 z 扫描:z-[N] 任意值类 / tailwind z-N 档位类 / zIndex:N 与 z-index:N 字面量。 */
function zViolations(files) {
  const out = []
  const Z_RAW = /z-\[(\d+)\]|(?:^|[\s:'"[(])z-(\d+)(?=[\s'"`)}]|$)|zIndex:\s*(\d+)|z-index:\s*(\d+)/g
  for (const p of files.filter((f) => /\.(tsx?|css)$/.test(f))) {
    const rel = p.slice(SRC.length + 1)
    const allow = Z_WHITELIST[rel]
    readFileSync(p, 'utf8').split('\n').forEach((line, i) => {
      const t = line.trim()
      if (t.startsWith('//') || t.startsWith('*') || t.startsWith('/*')) return
      for (const m of line.matchAll(Z_RAW)) {
        const v = Number(m[1] ?? m[2] ?? m[3] ?? m[4])
        if (!allow || !allow.includes(v)) out.push(`${rel}:${i + 1} z=${v}`)
      }
    })
  }
  return out
}

function walk(dir, out = []) {
  for (const f of readdirSync(dir)) {
    const p = join(dir, f)
    if (statSync(p).isDirectory()) walk(p, out)
    else out.push(p)
  }
  return out
}

const files = walk(SRC)
const used = new Set()
for (const p of files.filter((f) => /\.(tsx?|css)$/.test(f))) {
  for (const m of readFileSync(p, 'utf8').matchAll(/var\((--[a-zA-Z0-9-]+)/g)) used.add(m[1])
}
const defined = new Set()
for (const p of files.filter((f) => /\.css$/.test(f))) {
  for (const m of readFileSync(p, 'utf8').matchAll(/(--[a-zA-Z0-9-]+)\s*:/g)) defined.add(m[1])
}

const ghosts = [...used].filter(
  (t) => !defined.has(t) && !ALLOWED.includes(t) && !t.startsWith(ALLOWED_PREFIX),
)

const menuSrc = readFileSync(join(root, 'src/router/menu.def.ts'), 'utf8')
const menuKeys = [...menuSrc.matchAll(/key:\s*'([^']+)'/g)].map((m) => m[1])
const iconDir = join(root, 'public/icons/items')
const haveIcons = new Set(readdirSync(iconDir).map((f) => f.replace(/\.svg$/, '')))
const missingIcons = menuKeys.filter((k) => !haveIcons.has(k))

let failed = false
if (ghosts.length) {
  console.error(`[audit] 幽灵令牌 ${ghosts.length} 个(引用了未定义的 var):\n  ${ghosts.join('\n  ')}`)
  console.error('[audit] 修复:在 src/styles.css :root 或 theme/tokens.css 双主题块补定义;禁止只加白名单。')
  failed = true
}
if (missingIcons.length) {
  console.error(`[audit] 菜单图标缺失 ${missingIcons.length} 个(public/icons/items/<key>.svg 不存在):\n  ${missingIcons.join('\n  ')}`)
  failed = true
}
const zBad = zViolations(files)
if (zBad.length) {
  console.error(`[audit] Z-INDEX FAILED ${zBad.length} 处(浮层裸 z 值,唯一合法出口 styles.css --z-* 令牌):\n  ${zBad.join('\n  ')}`)
  console.error('[audit] 修复:className 用 z-drawer-mask/z-drawer/z-page-modal/z-dialog/z-fab/z-popover,inline 用 zIndex:\'var(--z-*)\';chrome 白名单仅限外壳布局,禁止为业务浮层加白。')
  failed = true
}
if (!failed) console.log(`[audit] ok — tokens=${used.size} menus=${menuKeys.length} z-clean`)
process.exit(failed ? 1 : 0)
