# 后端权限种子 × 前端 ROLE_GROUPS diff 对齐表（权限测试基线附件）

- 产出人：小林（前端，布局/路由/菜单权限）
- 日期：2026-08-28（何平会上要求）
- 探测方式：只读。admin/admin123 登录 `http://192.168.0.102:28080` 取 token → `GET /api/admin/v1/menu-perms`（后端当前事实，含后续迁移追加授权）；前端侧读 `web/admin/src/router/role-menu.ts` + `menu.def.ts` 展开 ROLE_GROUPS。未改任何数据。

## 口径说明

- FE = 前端实际会渲染/放行的页面 key 集合：内置七角色按 ROLE_GROUPS 展开到页；**technician、customer 不在前端 RoleCode 类型内，运行时走 role-menu.ts:28 兜底 = 仅 overview 组（dashboard）**，下表 FE 以运行时真实行为计。
- BE = 线上 menu-perms 该角色持有的 `menu:<key>` 全集。
- 前端菜单全集实为 **16 组 85 页**（menu.def.ts 头注释写 84 页已过时，实际数得 85）。

## 对齐表

| 角色 | FE | BE | 后端有前端无（菜单被藏，接口可能可调） | 前端有后端无（菜单可见，接口大概率 403） | 结论 |
|---|---|---|---|---|---|
| sysadmin | 82 | 83 | `ai`（前端菜单.def 无此页面） | — | 两套真相源**零漂移**：82 项完全对齐。唯一差异 `menu:ai` 属前端缺页（后端授权了 AI 能力页，前端 16 组 85 页里没有） |
| ops | 42 | 12 | — | dashboard, user, userdata, realname-review, marketing, marketing-recon, aaadashboard, aaalog, alarm, analytics, gis, report, check, quadlink, scanlog, site, site-cats, knowledge, release, message, feedback, service-metrics, install-board, worker, worker-reg, worker-ops, provision, template, provlog, collection-tasks（30 项） | **漂移面最大**。前端按 10 个组粗粒度放行，后端种子仅 12 项。若接口按 menu:* 拦截，ops 点 30 个菜单全部 403 |
| asset_admin | 7 | 4 | — | dashboard, inventory, purchase | 采购单/库存查询可见但后端未授权；**首屏工作台 menu:dashboard 后端也没给** |
| resource_admin | 14 | 12 | `check`（四码对账页，后端给了授权但前端 quad 组不在其可见组内） | dashboard, provision, provlog | 双向漂移：check 是"有权的功能被藏"；provision 组前端可见后端仅授 template |
| analyst | 7 | 5 | — | aaalog, alarm | 前端 aaa 组整组可见，后端只授 aaadashboard |
| partner_admin | 3 | 4 | `partner-audit`（前端无此页面） | — | `menu:partner-audit`（入驻审核）授给企业管理员，语义可疑：入驻审核应为平台方权限，请老周确认 |
| partner_staff | 4 | 3 | `partner-audit` | dashboard, partner-staff | 企业员工前端可见员工管理（partner-staff）与工作台，后端均未授权 |
| technician | 1（兜底） | 6 | dispatch, order, provision, provlog, quadlink, scanlog | dashboard | **疑点①实锤**：后端明确授权 6 页，前端类型不含 technician → 全部静默藏掉且直访 403，6 项授权在 web 端完全不可达 |
| customer | 1（兜底） | 0 | — | dashboard | 后端 0 授权正确；但前端兜底仍渲染工作台。非后台角色（customer/technician）建议登录即拦截 |

## 横向发现（跨角色）

1. **`menu:dashboard` 仅授给 analyst 与 sysadmin**。ops/asset_admin/resource_admin/partner_staff/technician/customer 前端全都渲染工作台，后端均无授权——若 /dashboard 数据接口按 `menu:dashboard` 拦截，这五类角色登录后首屏即 403，属高优先级验证项。
2. **后端授权但前端 menu.def 不存在的 key 共 2 个**：`ai`、`partner-audit`（前端缺页，非组级漂移）。
3. 后端角色列共 9 个（含 technician、customer），前端 RoleCode 仅 7 个，类型与后端枚举不对齐。

## 测试注意（引用我会上发言的口径）

- BE 有 ≠ 接口一定放行：menu:* 是授权事实，页面数据接口实际用 menu:* 还是业务 permCode 拦截需抽样实测（建议每角色抽 2~3 个 feOnly 项调一次接口看 200/403）。
- FE 无 ≠ 一定 403：内置角色 canAccess 走组级判定（role-menu.ts:51 短路），permissionCodes 不参与；beOnly 项的直访 403 需逐页验证。
- 建议把本 diff 脚本逻辑固化为 CI 门禁（menu.def 全集 × ROLE_GROUPS × menu-perms 三方对账），防后续迁移继续漂移。
