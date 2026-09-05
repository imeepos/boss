# PP1-B 客户计费与经营域页面打磨 · 验收证据(2026-09-05)

> 分支 feat/pp1b-bss-billing(worktree boss-pp1b-bss-billing)· 对接环境 http://192.168.0.102:28080(admin 前缀 /api/admin/v1,冒烟 admin/admin123)· 范围冻结:web/admin/src/pages/{bss,billing,intel} + 集中登记文件(i18n,独立 append-only 小提交)。

## 一、定点清单逐项处置结论

### 1. intel/report 原生 select 红线 —— 已修
- 周期选择器(原 L131 原生 select 元素)替换为 components/Dropdown 体系,value/onChange/disabled 语义不变。提交 80a2f272。
- 机械验收:对 pages/{bss,billing,intel} 三目录 grep select 标签,零命中(exit 1)。

### 2. bss/marketing 七页复用迁移主项 —— 已迁(行为保持)
- coupons/earn/gift/levels/recon-coupons/recon-points/tasks 七页统一接入 DataTable + PageHead + Pagination 骨架;裸 table/thead/tbody 全部移除。提交 e9169b3c。
- PageHead 由宿主 index.tsx/recon.tsx 移入各 Tab(标题/描述复用既有 i18n 键,文案不变)。
- 列表新增客户端分页(TableFooter 无使用先例,沿用全站 DataTable+Pagination 主流排布);recon 两页 driftOnly 筛选变化显式 setPage(1)。
- earn 页为单规则摘要页:规则以单行 DataTable 呈现(空态=未配置),分页不适用故不加。
- API/表单/校验/toast 行为零变更;coupons.test.ts(toCents/toCount 纯函数)原样通过。

### 3. 删除类操作确认核实 —— 已核实并处置
102 实测(2026-09-05,Bearer token 探测,不存在的 id 999999 用于判别路由存在性):

| 探测 | 结果 | 结论 |
|---|---|---|
| PUT /coupon-templates/999999/disable | 200 code=50000(路由存在,空档报错) | 停用路由存在 |
| PUT /coupon-templates/999999/enable | 404 page not found | 无启用反向口 |
| gift-rules / loy/levels / loy/tasks 的 disable 与 enable 同口径探测 | disable 路由在,enable 全 404 | 停用不可逆 |

- coupons/gift/levels/tasks 四页停用入口接入 useConfirm 危险确认(danger,文案 i18n disableConfirm 三语,明示暂无启用接口不可直接恢复)。提交 94d593d2。
- bss/userdata:动作处理器(bss/userdata/index.tsx 的 act())既有 confirmDialog(danger)包络,columns.tsx 仅列定义——核实无需改动。
- billing/billing/invoices.tsx:作废/重开/人工回填为更正类操作(POST /invoices/:id/{void,reissue,tax-backfill}),非物理删除;既有自带输入确认弹窗(原因/税号必填),ConfirmDialog 不支持输入收集,保持现状。账单/流水/对账类只读事实记录无物理删除入口,符合红线。

### 4. 建/删能力对齐核实 —— 已核实,页面已对齐(零改动)
102 实测 + 本地同 commit 路由源(internal/httpapi/admin/customer.go)双重核对:

| 域 | 102 API 实际能力 | 页面入口 | 结论 |
|---|---|---|---|
| 客户 | GET/POST /customers、GET /:id、GET /:id/verify-logs、POST /address;无 DELETE/停用路由(DELETE /customers/999999 = 404 page not found) | 有新建/详情/核验/地址/开户,无删除 | 对齐 |
| 产品 | GET/POST /products、PUT /:id、PUT /:id/status(999999 = 200 code=40400 业务信封)、price-history、provision-binding;无 DELETE /products/:id(404 page not found) | 有新建/编辑/上下架(toggleStatus)/调价/绑定,无删除 | 对齐 |
| 入驻 | 复用 POST /customers 建档 | CustomerCreateDrawer/RegistrationQueueDrawer 在页 | 对齐 |

### 5. intel/monthly 与 CounterPaymentForm 复用核对 —— 已迁
- intel/monthly/EditDrawer.tsx:裸 label/button 改 FormField + SubmitButton。提交 1b518b3a。
- intel/monthly/ImportExport.tsx:裸 BTN 常量按钮改 ToolbarButton。
- billing/payment/CounterPaymentForm.tsx:裸 label/input/footer 按钮改 FormField + Input(表单输入唯一入口)+ ToolbarButton + SubmitButton;既有 Dropdown/useConfirm/CustomerPicker 保留。

## 二、需求四维度落地

- A 增删改查完整性:见清单 3/4。未发现页面有按钮而 API 不存在的情形;未给只读事实记录强加删除。
- B 关联关系(bss/customer 详情):提交 a392d300。详情抽屉按 data-relations §0 铁律 4 重构:
  - 归属链:运营主体(GET /legal-entities 反查名,取不到降级 #ID 并 console.warn)→ 经营区域(regionName)→ 客户(#id);
  - 关联区块:订单计数(GET /orders?customerId= 服务端过滤,102 实测)+「查看」下钻 /bss/onboarding?customerId=;账单计数(GET /bills?customerId=);缴费显示 —(GET /payments 仅支持 billId,customerId 参数 102 反例实测被忽略、返回全量 18 条,前端不做全量拉取臆造);
  - 附带修复:详情里 regionName 裸键改走 i18n regionLabel。
- C 组件复用:见清单 2/5。marketing 七页 + 三表单全部纳入 components/business(DataTable/PageHead/ToolbarButton/EmptyState/FormField/SubmitButton/ActionLink)+ Dropdown + ConfirmDialog + pickers。
- D 样式统一:触达文件零 JSX 内联 style、零裸十六进制色值(grep 验证);新增文案 16 键三语闭环(types.ts + zh-CN/en-US/ms-MY,keys.test 通过);无 emoji(node Unicode 扫描)。

## 三、机械验收输出摘要

| 验收项 | 命令/口径 | 结果 |
|---|---|---|
| typecheck | cd web/admin && pnpm typecheck | PASS(tsc --noEmit 零输出) |
| test | pnpm test | 69 文件/423 用例全绿(TZ=Asia/Shanghai;见债务 D1) |
| build | pnpm build | PASS(✓ built in 4.47s,含 tsc) |
| select 红线 | 对 pages/{bss,billing,intel} grep select 标签 | 零命中 |
| 七页骨架 | grep -c DataTable/PageHead 七文件 | DataTable 2-3 命中、PageHead 均 2 命中 |
| 筛选回页 | recon 两页 driftOnly onChange setPage(1) | 各 1 处 |
| 停用确认 | grep -l useConfirm marketing 四页 | 4/4 |
| 文件行数 | wc -l 触达文件 | 最大 243(bss/customer/index.tsx)≤300 |
| 102 冒烟 | cdp-admin-capture.mjs --base http://192.168.0.102:5180 | 见下 |

### 102 实截冒烟(2026-09-05,部署基线 = 合并前 main)

- /bss/marketing?theme=dark&lang=zh-CN:h2=营销与积分规则;五 Tab 齐全;表 8 列 4 行;console 0 error、失败请求 0(logs 203 entries)。截图 docs/acceptance/assets/pp1b-marketing-dark-zh.png。
- /bss/customer?theme=light&lang=zh-CN:点首行「详情」后 aside[role=dialog] 抽屉开启(title=详情,k-v 18 span);console 0 error、失败请求 0(logs 209 entries)。截图 docs/acceptance/assets/pp1b-customer-detail-light-zh.png。
- 说明:5180 为合并前 main 的 CI 部署,冒烟证明域页面线上基线健康;本分支改动落地图形验证随负责人合并后 CI 部署复核。当前会话模型不读图,以上以 DOM 断言 + logs JSON + 截图文件(110KB/142KB)佐证,截图供人工复核。

## 四、范围外遗留债务登记(只登记不修)

- D1 时区敏感测试(存量):web/admin/src/pages/bss/user/filter.test.ts「注册时间本地时区格式化」断言 2026-08-21T10:00:00 格式化为原样串,仅 UTC+8 时区通过;美西时区下 pnpm test 该 1 例红(其余 422 绿)。属 bss/user 域,不在本任务定点清单;建议测试内钉 TZ 或改断言含偏移格式。
- D2 StatusTag 域缺口:marketing 规则状态(ENABLED/DISABLED)与 recon 差异(COUNTER_DRIFT/REDEMPTION_LOST)未注册 StatusTag registry(components/StatusTag/registry.ts,范围外文件);注册域+颜色+i18n statusTags 键需随下次 registry 扩域一并处理,当前保留 Badge 变体语义色。
- D3 payments 客户维度过滤:GET /payments 无 customerId 过滤(后端 billing_handlers.go 仅 billId;102 已部署版同样忽略该参)。客户详情缴费计数现显示 —;后端补过滤后前端一行接入。
- D4 enable 反向口:券模板/赠送时长/等级/任务停用无对应启用接口(102 实测 404);如业务需要可逆停用,须后端补 :id/enable 后再放开前端确认文案中不可恢复表述。

## 五、提交清单(本分支)

| 提交 | 类型 | 内容 |
|---|---|---|
| a35e54e3 | feat(i18n) | marketing 分页/停用确认 + customer 关联区块文案三语(append-only 独立小提交) |
| e9169b3c | refactor(bss-marketing) | 七页接入 business 骨架(行为保持) |
| 94d593d2 | fix(bss-marketing) | 四处停用补 ConfirmDialog |
| 80a2f272 | fix(intel-report) | 原生 select 改 Dropdown |
| a392d300 | feat(bss-customer) | 详情抽屉归属链 + 关联记录区块 |
| 1b518b3a | refactor(bss-billing-intel) | EditDrawer/ImportExport/CounterPaymentForm 迁移 |
| (随文档) | docs(acceptance) | 本证据文档 + 两张截图 |