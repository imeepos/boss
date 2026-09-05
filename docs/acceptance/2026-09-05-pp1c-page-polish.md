# PP1-C 订单履约与资产域页面打磨 · 验收证据

> 分支 `feat/pp1c-boss-ams-oss` ｜ worktree `boss-pp1c-boss-ams-oss` ｜ 日期 2026-09-05
> 对接环境 http://192.168.0.102:28080(admin 前缀 /api/admin/v1,冒烟账号 admin/admin123);无本机服务、无 mock。
> 契约依据:docs/contract/terms.md(12 环节/状态枚举)、data-relations.md §1/§2.6、fields.md;冲突以契约为准。

## 一、定点修复清单逐项处置结论

### 1. 裸 alert() 红线 —— 已修复
- 审计命中 3 处:`boss/worker/TeamDialogs.tsx:211`(解散队伍失败)、`boss/worker/index.tsx:71`(设队长失败)、`:84`(加入装维队失败)。
- 全部替换为 sonner toast(`toast.error`,失败文案沿用既有 `workerPage.actionFail`);成功路径补 `toast.success`(新增 i18n 键 `captainSet`/`addedToGroup`,三语闭环,append-only 独立提交 c45bf132)。
- 提交:0720d00b `fix(boss-worker): 裸 alert 替换为 sonner toast 反馈`。

### 2. 内联 style 收敛 —— 已修复
- 审计命中:`boss/message/WorkerMessagesTab.tsx` 21 处、`NoticesTab.tsx` 15 处,现两文件 `style={{` 计数均为 0。
- 全部落 theme tokens(shell-card-border/shell-fab-bg/color-danger/color-success/color-border-focus 等),删除 ctl/th/td CSSProperties 裸色值对象(#d9d9d9/#fafafa/#1677ff/#52c41a/#e54545/#999/#666/#fff);
  上架/下架操作链接改复用 business `ActionLink`;发布/下发成功反馈改 `toast.success`,失败留抽屉内联错误。
- 提交:97bf606b `style(boss-message): 消息中心两页签内联 style 收敛为令牌 className`。

### 3. 超 300 行拆分 —— 已修复
- 原 `ams/purchase/index.tsx` 381 行 → 拆为:index.tsx 195 + CreateOrderDrawer.tsx 127 + ConfirmReceiptDrawer.tsx 104(+SuppliersDrawer.tsx 113,见第 5 项),单文件均 ≤300 行、单函数均 ≤60 行。
- 拆分提交 bf96da39 为纯结构迁移(零行为变更),行为变更(取消/供应商)独立 feat 提交。
- 提交:bf96da39 `refactor(ams-purchase): 拆分采购单页,单文件均低于 300 行`。

### 4. 删除类操作确认核实 —— 核实为无真删除路径,不改码
- `boss/site/editor.tsx` 全文无 DELETE 调用;唯一"移除"为封面附件 `coverAttachmentId=0`,属表单本地状态清空(可重新选取撤销),非服务端删除,无需确认弹窗。
- 交叉核实 `boss/site/index.tsx`:列表页真删除走 `DELETE /site-posts/:id` 且已带 `confirm(s.deleteConfirm, { danger: true })`(ConfirmDialog 体系),无裸 confirm/prompt。
- 结论:该文件不存在"真删除无确认"问题;登记备查,零改动。

### 5. 建/删能力对齐(102 实测后处置)—— 已补齐
102 实测方式:Bearer 登录取 token → 只读 GET 核实数据形态 → 假 ID POST 探测路由(返业务错不落库)。

| 实体 | 102 API 能力 | 页面原状 | 处置 |
|---|---|---|---|
| 采购单 | POST /procurement/orders/:id/cancel 存在(契约 000163:任意非 RECEIVED 可 CANCELLED;假 ID 探测返 42200 业务错,路由在) | 缺取消入口 | 补「取消订单」按钮(DRAFT/SUBMITTED/PARTIAL 可见),ConfirmDialog(danger)+ 状态机 API,禁物理删除 |
| 供应商 | GET/POST /procurement/suppliers、POST /:id/disable 存在 | 全站无任何管理入口 | 新增 SuppliersDrawer(列表/新增/停用),停用走 ConfirmDialog(danger) |
| 盘点 ams/stock | POST /stocktakes 存在 | 已有创建入口 | 对齐,无改动 |
| 换新 ams/replace | POST /replacements、/:id/assign 存在 | 已有创建+派单入口 | 对齐,无改动 |
| 调拨 oss/transfer | POST /transfers、/:transferNo/approve|reject 存在 | 已有创建+审批入口 | 对齐,无改动 |
| 扩容 oss/expand | POST /expansions 存在(无审批类 API) | 已有创建入口 | 对齐,无改动 |
| 师傅停用 boss/worker | **后台无师傅停用/离职 API**(worker.go 路由全表核实:仅 create/settings/password/regions/transfer) | 页面无停用按钮 | 对齐正确:API 不存在则页面不得造按钮;登记结论 |
| 装维队解散 boss/worker | DELETE /worker-groups/:groupId 存在 | 已有解散入口(Shell 确认框) | 对齐,仅按第 1 项把失败反馈迁 toast |

- 提交:e77a251c `feat(ams-purchase): 补订单取消与供应商管理入口,对齐 102 API 能力`。

### 6. 组件复用核对 —— 已迁移/核实
- `provision/template/TemplateForm.tsx`:footer 裸 `<button>`(样式被截断、无 hover/cursor)→ 迁 ui `Button`;硬编码英文标签 "Status"/"Content JSON"/"Edit template" → i18n 键 `statusLabel`/`contentLabel`(复用既有 `edit`),三语闭环。
- `ams/stock/ItemsDrawer.tsx`:扫码成功反馈由内联 msg 文本迁 `toast.success`;错误横幅/表格骨架保持既有令牌化写法(已达标)。
- `boss/worker/WorkerDialogs.tsx`:已复用 Shell/Err/compact(TeamDialogs)+ ui Button/Input + Dropdown + MultiSelect,达标无改动。
- 提交:2ebe6c05 `refactor(provision-template,ams-stock): 表单/抽屉迁移既有组件体系`。

## 二、需求 B:关联关系(方案二选一 → asset 链)

- 选 asset 批次-采购-标签链:order 12 环节链所需 order_stages 读 API 在 admin 面不存在(internal/httpapi/admin 无 stages 路由),页面无数据来源;asset 链全部 API 已具备。
- 实现:AssetTrailDrawer 顶部新增「关联关系」区块(data-relations §2.6 口径):
  - 所属批次:GET /assets/batches 按 batchId 命中(#id + code + name;两接口均为小表全量,102 实测 receipts 1 行、batches 193 行,客户端过滤可行);
  - 采购入库单/采购订单:GET /procurement/receipts 按 batchId 命中 receiptNo/orderNo(取不到显示 "—",禁止臆造);
  - 绑定标签:由列表页 /tags 联表传入 tagNo · epcCode。
- 顺带修复该抽屉页签内联 style(#1677ff)→ business TabBar 组件。
- 提交:ebb01c8b `feat(ams-asset): 资产详情抽屉补批次-采购-标签关联链`。

## 三、quad / alarm / provision / aaa 四域扫描债务登记(只登记不修)

| 域 | alert | 内联 style | 裸色值 | >300 行文件 | 原生 select | 反馈方式 | 债务明细 |
|---|---|---|---|---|---|---|---|
| quad | 0 | 1 | 1 | 无 | 0 | 内联 error banner + ConfirmDialog×2 | quad/check/index.tsx:81 hint div 内联 style+#1677ff(低) |
| alarm | 0 | 1 | 1 | 无 | 0 | 内联 error banner + ConfirmDialog×2 | alarm/index.tsx:94 hint div 内联 style+#1677ff(低) |
| provision | 0 | 0 | 0 | 无 | 0 | 内联 error banner(无 toast)+ ConfirmDialog×4 | 指标全净;成功反馈未走 toast(低,与页面既有惯例一致) |
| aaa | — | — | — | — | — | — | pages/aaa 目录不存在;AAA 域实际页面为 pages/aaa-dashboard 与 pages/aaalog(域命名漂移,建议后续归位) |

## 四、机械验收规则输出摘要

1. `cd web/admin && pnpm typecheck && pnpm test && pnpm build` 全绿:
   - typecheck:`tsc --noEmit` 零输出通过;
   - test:`Test Files 69 passed (69)` / `Tests 423 passed (423)`;
   - build:`✓ built in 8.43s`。
   - 注:test 以 `TZ=Asia/Shanghai` 运行(与 102 部署环境同时区)。存量域外债务:`pages/bss/user/filter.test.ts` 注册时间格式化用例按 +08:00 本地时区断言,在本机 PDT 时区必红,本轮按范围外只登记不修(bss 域不在冻结范围)。
2. `git grep -nE '(^|[^.a-zA-Z])alert\(' -- web/admin/src/pages/boss` → 零命中。
3. `wc -l web/admin/src/pages/ams/purchase/*.tsx` → 104 / 127 / 113 / 195,均 ≤300。
4. 定点清单第 4 项:editor.tsx 核实无真删除路径(见一.4);第 1 项触达的对话框操作(TeamDialogs 解散、setCaptain、addWorkerToGroup)均已无原生 alert,失败走 toast。
5. 102 实截冒烟(cdp-admin-capture.mjs,--base http://192.168.0.102:5180,theme=light lang=zh-CN):
   - /boss/worker:`102-worker.png`(241KB);console 采集 211 条,error 0、失败请求 0;DOM 断言 title=师傅管理 ✓ team=装维队 ✓ table ✓;
   - /ams/purchase:`102-purchase.png`(87KB);console 采集 203 条,error 0、失败请求 0;DOM 断言 title=采购单 ✓ table ✓ url=/ams/purchase ✓;
   - 说明:102 部署为既有前端版本,实截验证的是本域页面对 102 后端的真实运行基线(console/网络零报错);本轮新增代码由本地门禁(typecheck/test/build)覆盖,未部署 102。
6. 证据文档:本文档 `docs/acceptance/2026-09-05-pp1c-page-polish.md`。

## 五、提交清单(feature 分支,自下而上)

1. c45bf132 chore(i18n): workerPage 增 captainSet/addedToGroup 两键(append-only,三语闭环)
2. 0720d00b fix(boss-worker): 裸 alert 替换为 sonner toast 反馈
3. 97bf606b style(boss-message): 消息中心两页签内联 style 收敛为令牌 className
4. bf96da39 refactor(ams-purchase): 拆分采购单页,单文件均低于 300 行
5. e77a251c feat(ams-purchase): 补订单取消与供应商管理入口,对齐 102 API 能力
6. ebb01c8b feat(ams-asset): 资产详情抽屉补批次-采购-标签关联链
7. 2ebe6c05 refactor(provision-template,ams-stock): 表单/抽屉迁移既有组件体系
8. (本文档)docs(acceptance): PP1-C 页面打磨验收证据

变更范围核对:全部位于 web/admin/src/pages/{boss,ams,oss,provision,alarm,quad,aaa/} 与集中登记文件(i18n types+locales,append-only 独立小提交);oss/quad/alarm 域本轮零改动(扫描仅登记);范围外问题(bss 时区用例)只登记未修。