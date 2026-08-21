# A0 基座计划 · web/admin 工程搭建（登录 / RBAC 菜单 / 请求层直连 Go 后端 / StatusTag）

> 版本 V1.1｜上级规划：`docs/plan/admin-system-plan.md` 第五节批次 A0
> 契约依据：`internal/app/http.go`（已交付实现，权威）、`api/openapi/admin/auth.yaml`、`docs/contract/terms.md`
> **V1.1 变更：不对接 mock（:8092）。请求层只对接 Go 后端真实接口。**

---

## 0. 已核实的契约事实（决定 A0 设计，以 Go 实现为准）

| 事实 | 来源 | 对 A0 的影响 |
|---|---|---|
| Go 服务 `cmd/server`：HTTP `:8080`、gRPC `:9090`，`configs/config.example.yaml` | cmd/server/main.go | dev 直连 `http://127.0.0.1:8080` |
| **真实前缀是 `/api/v1`**，不是 admin.yaml 写的 `/api/admin/v1`（三端共用前缀，网关层再分流） | internal/app/http.go `r.Group("/api/v1")` | client baseUrl 用 `/api/v1`；契约文件与实现的偏差按"以实现为准"处理并回改 admin.yaml |
| **响应统一 envelope `{code, msg, data}`，HTTP 恒 200**；业务错误靠 code（apitypes） | http.go `respond/respondErr` | 请求层必须解 envelope：code≠0 抛业务错误，data 才是负载 |
| `POST /auth/login` 入参 `{username,password}`，返回 `data:{token, accountId, realName, roleName}` | http.go L71-93 | 登录只存 token，用户信息以 `/auth/me` 为准 |
| `GET /auth/me` 返回 `data:{accountId, username, realName, roleCode, roleName, legalEntityName, regionScope}`，**不含菜单树** | http.go L99 + user.Service.Profile | `roleCode`（RoleCode 枚举）直接驱动菜单可见性 |
| RBAC 为逐接口 permCode 注入 + Redis 快照，权限变更即时生效 | user.Service 注释、middleware | 前端不做权限判断，仅按 roleCode 渲染菜单、兜底 403 |
| 枚举权威源已存在：Go 枚举定义（与 terms.md 对齐，含 RoleCode） | Go 实体 | StatusTag 与菜单映射的枚举注册表从这里导出，不手抄 |
| 原型登录页 `docs/admin/login.html`，菜单结构 `docs/admin/menu.js` 13 组 45 页 | docs/admin | 交互与结构照抄 |

## 1. 任务分解（按依赖顺序）

### T1 工程脚手架
- `web/admin/`：Vite + React 18 + TS strict；ESLint/Prettier；目录结构按上级规划 §三。
- `package.json` scripts：`dev / build / preview / lint / typecheck / test / gql:none`（不引 codegen 以外的生成器）。
- CI：接入现有门禁思路，`lint + typecheck + vitest run` 三命令过才绿。
- 产出：空壳页可 `pnpm dev` 访问。

### T2 请求层（直连 Go 后端，单通道）
- `openapi-typescript` 以 `api/openapi/admin.yaml` 生成 `src/api/schema.d.ts`（脚本 `pnpm gen:api`，契约更新后重跑）；生成前先用 `@redocly/cli bundle` 合并相对 `$ref`。
- `openapi-fetch` 创建 client，**仅一条通道**：
  - dev：Vite `server.proxy` 把 `/api` 代理到 `http://127.0.0.1:8080`（同源无 CORS，token 走 header）
  - prod：同源 `/api/v1`（APISIX/Nginx 反代到 Go server）
  - baseUrl 固定 `/api/v1`（真实实现前缀，非 admin.yaml 的 `/api/admin/v1`）
- envelope 解包中间层（核心）：HTTP 200 不代表成功——`code===0` 取 `data` 为负载；`code===401`（CodeUnauthorized）清 token 跳登录；其余 code 抛携带 `code+msg` 的业务错误，供 ProTable/ProForm message 展示。列表接口若实现返回裸数组，则在同一中间层适配为 `{items}`（以 e2e/集成测试实测形状为准）。
- 依赖启动：dev 前置 `make` 起 Go server + PG/Redis（连接串见 `configs/config.example.yaml`；e2e 已用 PG=192.168.0.102:25432）；README 写明启动步骤。
- 产出：`src/api/client.ts` + envelope 工具 + 集成冒烟脚本（`scripts/smoke.sh`：curl 登录→me 断言 envelope 形状）。

### T3 登录与路由守卫
- `/login` 页：照抄 `docs/admin/login.html` 交互（账号密码 + 错误提示）；`adminLogin` → 存 token → 跳 dashboard。
- `AuthGuard`：无 token 跳 `/login`；有 token 预取 `/auth/me`（TanStack Query `queryKey:['me']`），失败登出。
- `POST /auth/logout` 清态。
- 顶栏展示 `realName / roleName / legalEntityName`，数据域徽标显示 `regionScope`（空=全集团）。

### T4 RBAC 菜单驱动（本批核心决策）
- **方案（推荐）**：前端内置 `menu.js` 结构的静态菜单定义（45 页路由全集），可见性由 `/auth/me` 的 **roleCode**（RoleCode 枚举：sysadmin/ops/asset_admin/resource_admin/customer/technician/analyst）映射到 13 分组；映射表随菜单定义一起维护。
- 后端无 `/menu-perms`、`/auth/menus` 接口（admin.yaml 的 menu-perms 尚未在 Go 侧实现），A0 **不做**该依赖；同步提交契约变更单：`auth.yaml` 增加 `GET /auth/menus` 返回菜单树，后端补 handler 后前端切换，静态映射降级为 fallback。
- 前端仅按 roleCode 渲染菜单；接口级权限由后端 permCode 拦截，前端收到 code=403 类错误统一提示"无权限"，直访未授权菜单 URL → 前端 403 页。
- 产出：`src/layouts/`（Sidebar/Breadcrumb/AuthGuard）+ `src/router/`（含 roleCode→分组映射表）+ 403/404 页。

### T5 StatusTag 组件
- `src/components/StatusTag/`：`<StatusTag domain="order" value="ASSIGNED" />`，domain→枚举注册表。
- 注册表来源：Go 枚举定义导出 JSON（加一个导出脚本或直接复制为 `src/api/enums.json`，脚本校验与 terms.md 一致）；颜色按原型各页 `<span class="st-*">` 语义集中定义一次。
- 产出：组件 + 注册表 + 快照测试（全枚举渲染不出 unknown）。

### T6 account 页迁移样板（验证基座，非 A1 正式内容）
- base 组 `account.html` → `pages/base/account/list.tsx`：ProTable 列名严格对照 `fields.md`，操作列/筛选区照抄原型。
- 目的：跑通"列名对照表 → ProTable → 自查清单"流程，为 A1 定标。
- 产出：account 页 + 自查清单模板 `docs/plan/admin-page-checklist.md`。

## 2. 文件清单（A0 完成时）

```
web/admin/
├── package.json  vite.config.ts  tsconfig.json
│     # vite.config.ts: server.proxy['/api'] → http://127.0.0.1:8080
├── scripts/gen-api.sh              # redocly bundle + openapi-typescript 生成
├── scripts/smoke.sh                # curl 直连 Go server 冒烟(login→me envelope 断言)
└── src/
    ├── main.tsx  App.tsx
    ├── api/ client.ts  envelope.ts  schema.d.ts(生成)  enums.json  auth.ts
    ├── layouts/ AdminLayout.tsx  AuthGuard.tsx  HeaderUser.tsx
    ├── router/ index.tsx  menu.def.ts  role-menu.ts   # 菜单定义 + roleCode→分组映射
    ├── components/ StatusTag/  (index.tsx + registry.ts)
    ├── pages/ login/  dashboard/(占位)  base/account/  error/{403,404}.tsx
    └── styles/ theme.ts                      # 对齐原型 style.css 色板/间距
```

## 3. 验收标准（A0 Definition of Done）

1. 本地起 Go server（PG/Redis 就绪）后 `pnpm dev`：admin 账号登录成功 → 侧边栏仅显示该 roleCode 可见分组 → 顶栏显示 me 信息 → 登出生效。**全程不依赖 :8092 mock。**
2. `scripts/smoke.sh` 直连 `:8080` 断言 login/me 的 envelope 形状（`{code,msg,data}`、code=0）。
3. 手改 localStorage 伪 token → 任意路由跳登录（后端 401 code 触发）。
4. 直访未授权菜单 URL → 403。
5. StatusTag 覆盖 enums.ts 全部枚举，快照测试绿。
6. account 样板页与原型逐列一致（自查清单留档；数据来自 Go server 真实接口）。
7. `lint + typecheck + vitest` 全绿，单文件 ≤300 行、函数 ≤30 行。

## 4. 风险与对策

| 风险 | 对策 |
|---|---|
| admin.yaml 契约与 Go 实现漂移 | **已对齐（V1.1，本次完成）**：前缀改 `/api/v1`、响应统一 envelope `{code,msg,data}`、auth.yaml 补 roleCode、`/menu-perms` 标注未实现、修复 5 处存量 YAML 语法错误；gen:api 可直接消费 |
| `/auth/me` 无菜单树（契约缺口） | T4 用 roleCode 静态映射先跑，同时提 `/auth/menus` 契约变更单，避免前端硬编码权限 |
| 列表响应形状（裸数组 vs {items}）未在全部 handler 统一 | T2 冒烟脚本覆盖已交付域的代表性列表接口，envelope 中间层按实测形状适配并记录 |
| dev 依赖本地 PG/Redis/seed 账号 | README 写明启动步骤与最小 seed（e2e_seed 已有先例）；无账号时先跑 seed 脚本 |
| enums.ts 与前端类型漂移 | gen 脚本校验 enums.json 与 schema.d.ts 枚举值一致，不一致 CI 红 |
| openapi-typescript 对 admin.yaml 的 `$ref: './schemas.yaml#...'` 相对引用解析失败 | gen 脚本先用 `@redocly/cli bundle` 合并成单文件再生成（admin.yaml 已声明是聚合内联形态） |

## 5. 执行顺序与工作量估算

T1→T2→T3→T4→T5 串行依赖；T5 可与 T4 并行；T6 收尾。
预计 3~4 个工作日：T1 0.5d、T2 1d、T3 0.5d、T4 1d、T5 0.5d、T6 0.5d。
