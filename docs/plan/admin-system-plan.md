# 后台管理系统（admin 端）规划 · 技术栈与落地路线

> 版本 V1.0｜权威源：`docs/contract/{terms,domain-map,fields}.md`、`docs/admin/menu.js`、`api/openapi/admin.yaml`、`技术栈方案-一步到位.md`
> 定位：管理后台前端工程的**唯一执行规划**。原型（`docs/admin/*.html`）是交互与字段的视觉契约，本文档回答"用什么栈、放哪里、分几步做"。

---

## 一、现状事实（规划输入）

| 项 | 现状 |
|---|---|
| 原型 | `docs/admin/` 共 13 分组 45 菜单页 + login 散页，jQuery 静态 HTML，`menu.js` 定义分组结构 |
| 接口契约 | `api/openapi/admin.yaml` 聚合 + `api/openapi/admin/*.yaml` 按域拆分，前缀 `/api/v1`、envelope `{code,msg,data}`（V1.1 已对齐 Go 实现） |
| Mock | `api/mock/combined.js`（:8092），与真实网关同路由形状 |
| 后端 | Go 模块化单体已完成 12 域 REST handler（W1–W8），经 APISIX 暴露 |
| 用户端/师傅端 | 另有 `/api/v1`、`/api/worker/v1`，与 admin 隔离 |

**结论：不需要新建接口或改契约，admin 前端是"纯消费端"工程**，原型→真实前端的映射是本规划的全部内容。

## 二、技术栈（对齐《技术栈方案-一步到位.md》2.4）

| 层 | 选型 | 说明 |
|---|---|---|
| 框架 | React 18 + TypeScript + Vite | 与总方案一致；Vite 也与 DSH 前端生态同构 |
| UI | Ant Design 5 + ProComponents（ProTable/ProForm/ProLayout） | 45 页 90% 是"列表+筛选+抽屉表单"，ProComponents 直接覆盖 |
| 路由 | react-router v6，约定式路由按分组目录生成 | 菜单结构 = `menu.js` 的分组/页面一一迁移 |
| 状态/数据 | TanStack Query（服务端态）+ 轻量 zustand（本地态） | 列表缓存、失效、轮询（告警/订单进度）开箱即用 |
| 请求层 | openapi-typescript + openapi-fetch 从 `admin.yaml` 生成类型化 client | 字段对齐 `fields.md` 由编译期保证，替代原型 `api.js` |
| 图表 | ECharts（dashboard/analytics/report） | 与总方案一致 |
| GIS | Cesium（阶段8 接入，独立懒加载 chunk） | 大屏联动 WS 推送增量 |
| 实时 | WebSocket/SSE 订阅告警、订单状态、派单 | 复用网关中心化推送 |
| 权限 | 后端 RBAC 接口驱动菜单（`/auth/me` 返回菜单树+数据域），前端仅渲染不做权限判断 | 对齐总方案"7 角色权限由后端驱动菜单" |
| 对接后端 | 直连 Go 后端（cmd/server :8080，前缀 `/api/v1`，envelope `{code,msg,data}`）；dev 走 Vite proxy，prod 走 APISIX 同源反代；**不对接 mock**（契约漂移以 Go 实现为准回改 admin.yaml） | 详见 `docs/plan/admin-a0-plan.md` T2 |
| 测试 | Vitest + Testing Library（关键表单/权限渲染）；Playwright 冒烟（登录→下单→出账主链） | — |
| 规范 | ESLint + Prettier + strict TS；单文件 ≤300 行、函数 ≤30 行、无 emoji（AGENTS.md 强制） | CI 复用现有 `make lint/check` 门禁思路 |

**明确不引入**：Redux（无复杂客户端态）、Next.js（纯内网后台，无 SEO/SSR 需求）、微前端（45 页单团队单体足够）。

## 三、工程位置与结构

```
web/admin/                    # 新建，与 server-ts/ 平级
├── package.json / vite.config.ts / tsconfig.json
├── src/
│   ├── main.tsx / App.tsx           # ProLayout 壳 + 路由生成
│   ├── api/                         # openapi 生成物 + query hooks（按域一个文件）
│   ├── layouts/                     # 侧边栏（menu.js 结构）、面包屑、登录守卫
│   ├── components/                  # 通用：状态标签（terms.md 枚举色）、地址级联、
│   │                                #   附件上传（MinIO 预签名）、审计字段抽屉
│   ├── pages/<group>/<page>/        # 分组目录 = menu.js 分组 id
│   │   ├── list.tsx                 # ProTable 列定义严格照抄 fields.md 三列对齐
│   │   ├── detail.tsx / form.tsx    # 超 300 行再拆
│   │   └── api.ts                   # 本页 query/mutation hooks
│   └── styles/
└── e2e/                             # Playwright
```

页面清单以 `docs/admin/menu.js` 为单一事实源：overview/base/org/bss/billing/ams/oss/boss/quad/provision/alarm/aaa/intel 共 13 组 45 页（+login）。跨域归属页（complaint/arrears/loaccount/callback/importer）按 domain-map 2.1 标注的**真实能力域**组织代码，菜单仍挂原分组。

## 四、迁移映射规则（原型 → React）

1. **列名即契约**：原型表格列 ↔ `fields.md` 字段列 ↔ 后端 struct，三列必须一致；差异一律改前端，提契约变更单再动后端。
2. **枚举即组件**：`terms.md` 状态枚举统一走 `<StatusTag domain="order" value="ASSIGNED">` 一类映射组件，颜色全局注册一次。
3. **交互照抄原型**：原型已沉淀筛选区/抽屉/批量操作/二级页交互，React 版本不得自创交互；原型缺失的交互（如行内编辑）先补原型评审再实现。
4. **`api.js` 方法名 → 生成 client 的 operationId**：一一对应，页面迁移时只换调用点不改语义。
5. 原型 `docs/admin/` 保留为视觉契约不删除、不演进；新交互需求先改原型再进 React。

## 五、分阶段落地（对齐服务端 9 阶段 / 三期）

| 批次 | 范围（分组） | 依赖的后端域 | 验收 |
|---|---|---|---|
| A0 基座 | 工程搭建、登录、RBAC 菜单、布局、请求层+mock 打通、StatusTag/审计组件 | user（阶段1） | 登录后菜单随角色渲染；mock/真实双通道可切 |
| A1 组织与基础 | base + org（account/address/settings/audit/importer/company/department/post/region/menuperm/datascope） | user（阶段1） | 与原型逐页比对列名/筛选/操作项一致 |
| A2 客户与资费 | bss（customer/product/user/userdata） | customer（阶段2） | 同上；地址级联走 ltree 接口 |
| A3 资产与资源 | ams + oss（asset/tag/stock/replace/resource/reserve/transfer/device/loaccount/expand） | asset（3）+ resource（4） | 盘点导入走 MinIO 预签名上传 |
| A4 订单与计费闭环 | boss + billing（order/worker/dispatch/dismantle/complaint/callback + billing/payment/arrears/stopsrv/paycheck） | order + billing（5） | 下单→派单→扫码→激活→出账全链页面可操作；订单进度 WS 实时 |
| A5 四码/下发/告警/AAA | quad + provision + alarm + aaa | quadlink/provision/device/aaa（6/7） | 扫码绑定、下发重试留痕、告警轮询+推送、话单查询 |
| A6 数字孪生与经营 | intel（gis 懒加载 Cesium / analytics / report）+ dashboard 增强 | gis（8）+ analytics（9，Doris） | 热力图/下钻/报告导出 |

每批交付物：页面代码 + Vitest 关键用例 + 与原型的一致性自查清单；A4、A6 追加 Playwright 主链冒烟。

## 六、风险与对策

| 风险 | 对策 |
|---|---|
| 45 页手工迁移字段抄错 | client 从 OpenAPI 生成，列名类型化；列定义 review 对照 fields.md |
| ProComponents 定制样式与原型出入大 | 全局 theme token 一次性对齐原型 `style.css` 的间距/色板，页面级不再散改 |
| mock 与真实后端响应形状漂移 | `api/mock/selfcheck.js` 已有自检；CI 加 openapi 校验，mock 形状不过不让合 |
| GIS chunk 体积 | Cesium 独立 lazy chunk + CDN 子资源，仅 intel 分组加载 |
| userdata/worker 页面契约最杂 | A2/A4 内最后做，先跑通 `api/openapi/admin/userdata.yaml` 类型生成验证 |

## 七、下一步（开工顺序）

1. 建 `web/admin/` 工程 + A0 基座（登录/RBAC 菜单/请求层双通道/StatusTag）。
2. 用 base 组 account 页做首个迁移样板，确立"列名对照表→ProTable→自查清单"流程。
3. 样板评审通过后按 A1→A6 批量推进，每批一个 PR 合入。
