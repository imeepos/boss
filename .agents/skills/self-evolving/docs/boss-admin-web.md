# boss 项目速查手册（web/admin 前端）

> 只记"需要查证才知道"的事实；常识与 Read 即得的代码结构不记。
> 来源：2026-08-18 app shell 任务期间的实际查证。

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

## 前端约定（本次确立）

- 主题：`localStorage('boss.theme')` = light/dark，`documentElement[data-theme]` 驱动 `src/theme/tokens.css` 变量；`index.html` 内联脚本首屏前写入防闪白
- 语言：`localStorage('boss.locale')`，三语言 zh-CN/en-US/ms-MY，新增文案须同步三个 locale 文件
- URL 一次性覆盖（`src/lib/urlPrefs.ts`）：`?theme=`、`?lang=`、`?token=`（dev 限定）启动时写入 localStorage 后 replaceState 抹除；权威存储仍是 localStorage
- 可视化验证：`.agents/skills/self-evolving/scripts/cdp-capture.mjs`（零依赖 CDP 截图）
