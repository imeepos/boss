// 采集 102 线上菜单权限矩阵,落成对账门禁基线快照(零依赖,Node>=22)。
// 用法: BOSS_USER=admin BOSS_PASS=admin123 node scripts/fetch-menu-perms.mjs
//   [--base http://192.168.0.102:28080] [--out src/router/__fixtures__/menu-perms-snapshot.json]
// 漂移修复后重跑本脚本刷新基线;CI 侧 menu-sync.test.ts 断言"只减不增"。
import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'

const argv = process.argv.slice(2)
const arg = (k, d) => { const i = argv.indexOf(`--${k}`); return i >= 0 ? argv[i + 1] : d }
const base = arg('base', 'http://192.168.0.102:28080')
const out = resolve(arg('out', 'src/router/__fixtures__/menu-perms-snapshot.json'))

const user = process.env.BOSS_USER ?? 'admin'
const pass = process.env.BOSS_PASS ?? 'admin123'

const login = await fetch(`${base}/api/admin/v1/auth/login`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: user, password: pass }),
}).then((r) => r.json())
if (login.code !== 0) { console.error('login failed:', JSON.stringify(login).slice(0, 200)); process.exit(1) }

const matrix = await fetch(`${base}/api/admin/v1/menu-perms`, {
  headers: { Authorization: `Bearer ${login.data.token}` },
}).then((r) => r.json())
if (matrix.code !== 0) { console.error('menu-perms failed'); process.exit(1) }

const roles = {}
for (const col of matrix.data.matrix.roleColumns) {
  roles[col.roleCode] = { roleName: col.roleName, isBuiltin: col.isBuiltin, be: [] }
}
for (const row of matrix.data.matrix.rows) {
  for (const rc of row.roles) roles[rc]?.be.push(row.code)
}
for (const r of Object.values(roles)) r.be.sort()

const snapshot = { capturedAt: new Date().toISOString(), base, roles }
mkdirSync(dirname(out), { recursive: true })
writeFileSync(out, JSON.stringify(snapshot, null, 2) + '\n')
console.log(`captured ${Object.keys(roles).length} roles -> ${out}`)
for (const [rc, r] of Object.entries(roles)) console.log(`  ${rc}: ${r.be.length} menu codes`)
