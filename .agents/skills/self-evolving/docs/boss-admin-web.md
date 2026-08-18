# boss 项目速查手册（web/admin 前端）

> 只记"需要查证才知道"的事实；常识与 Read 即得的代码结构不记。
> 来源：2026-08-18 app shell / geo CRUD / geo 页重构任务期间的实际查证；事实漂移时以代码与 102 库实测为准。

## 冒烟登录

- 账号：`admin / admin123`，角色 `sysadmin`（由 `scripts/devseed/main.go` 写入 PG）
- 已部署后端：`http://192.168.0.102:28080`（vite dev 默认代理目标）
- 本地联调覆盖：`cd web/admin && BOSS_API_TARGET=http://127.0.0.1:18080 pnpm dev`
- mock 服务（`scripts/mock-admin.sh`，端口 8092）只有 user/worker 端登录，**不能**用于 admin 登录冒烟
- **dev 免登录**：`node web/admin/scripts/dev-token.mjs` 真实 /auth/login 换 JWT，打印 `http://localhost:5173/?token=<jwt>`；token 是真实会话（不用假数据，能暴露后端问题），仅 dev 构建应用该参数；过期重跑脚本

## 门禁

- `pnpm typecheck && pnpm test && pnpm build`（vitest 4 文件 20 用例；build 含 tsc --noEmit）

## 设计资料位置

- 外壳规格：`docs/admin/design-spec.md`（亮/暗 token、尺寸、断点，逐条可查）
- 品牌 token 源头：`docs/design/visual-design-prompts.md`
- 原型页面：`docs/admin/*.html`（jQuery 草稿，45 页）
- 原型图标：`docs/admin/icons/*.svg`（13 组）+ `docs/admin/icons/items/*.svg`（49 页），stroke 色写死 `#a6adb4`，换色须用 CSS mask

## 契约文档（AGENTS.md 强制，开工前必读）

- `docs/contract/terms.md`：订单 12 环节、状态枚举
- `docs/contract/domain-map.md`：能力域/Agent/internal 包/页面 四套命名对齐
- `docs/contract/fields.md`：页面列名 ↔ 字段名 ↔ 状态枚举

## 菜单图标体系（2026-08-18 查证）

- 菜单源：`web/admin/src/router/menu.def.ts`（13 组 49 项，key 唯一）
- 运行时图标在 `web/admin/public/icons/`：组图标 `<组id>.svg`（13 个），项图标 `items/<项key>.svg`（49 个）；新增菜单项必须同时补对应 SVG，否则侧栏该行无图标（geo 曾漏）
- 渲染走 `MaskIcon`（`layouts/icons.tsx`）：CSS mask + `background: currentColor`，颜色随所在按钮状态自适应，与 SVG 文件内写死的 stroke 色无关
- 顶栏 TopNav 组按钮也渲染组图标（size 16 + gap 6）；侧栏组图标 size 14、项图标 18

## 前端约定（本次确立）

- 主题：`localStorage('boss.theme')` = light/dark，`documentElement[data-theme]` 驱动 `src/theme/tokens.css` 变量；`index.html` 内联脚本首屏前写入防闪白
- 语言：`localStorage('boss.locale')`，三语言 zh-CN/en-US/ms-MY，新增文案须同步三个 locale 文件
- URL 一次性覆盖（`src/lib/urlPrefs.ts`）：`?theme=`、`?lang=`、`?token=`（dev 限定）启动时写入 localStorage 后 replaceState 抹除；权威存储仍是 localStorage
- 可视化验证：`.agents/skills/self-evolving/scripts/cdp-capture.mjs`（零依赖 CDP 截图）

## geo 域速查（2026-08-18 查证）

- 页面 `/base/geo`（国家+区划双页签）；API 前缀 `/api/v1/geo/*`，门禁 `menu:geo`（仅 sysadmin，迁移 000039 授予）
- "删"= 软删除：`PUT /geo/{countries,subdivisions}/:code/active {"active":false}`，契约 fields.md 1.5.1 规定停用码保留不物理删
- 国家关联属性整体替换：`PUT /geo/countries/:code/attrs`，请求体 `timeZones: string[]`、`callingCodes: string[]`、`currencies: {currency,isPrimary,minorUnit}[]`（字段类型以 `internal/domain/geo/geo.go` struct 为准）
- 译名 locale 用 BCP-47 风格 `zh-Hans`/`en`（种子数据口径），不是 `zh-CN`
- 102 库有停用态冒烟数据：国家 `T1`、区划 `T1-TS`（is_active=false，勿清理也勿复用为主数据）
- 前端样板：主题化 CRUD 页全套见 `web/admin/src/pages/base/geo/`（antd Pro 布局 + 双主题），通用右侧抽屉 `web/admin/src/components/Drawer.tsx`

## 102 库与权限架构事实（2026-08-18 查证）

- `schema_migrations.version` 是 TEXT（如 `000039_geo_menu`），不是整型；手工补迁移后须 INSERT 记录版本
- 权限模型：`HasPermission` 只查 `role_permissions` 显式行（internal/domain/user/pg.go），**sysadmin 没有隐式全权**——新菜单必须同时有"permissions 行 + role_permissions 授权"才会放行
- DB 直连（本机无 psql 时）：`host=192.168.0.102 port=25432 user=boss password=boss dbname=boss`（与 configs/config.example.yaml 同源）；用 /tmp 临时 go 程序 + pgx
- 冒烟账号 admin 的 account_id=103（102 库）

- 102 远程环境（CI 自动部署）：`http://192.168.0.102:28080`，账号 `admin / Boss-admin-2026`（sysadmin；口令权威=仓库 `deployments/app.env`，该文件已入库——内网私有仓库裁定，转公网前必须移出）。geo 维护页在 基础配置→国家与行政区划（menu:geo）。
