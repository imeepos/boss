# 裁定：开户工作台聚合页（2026-08-29）

> 目标："提供一个聚合工作台 提供用户开户全流程 在这个工作台完成所有操作 缺什么加什么"。
> 前置：同日已落地代客受理目录端点与三抽屉（adopted/2026-08-29-admin-onbehalf-catalog.md），
> 但能力分散在 客户档案页 与 订单管理页 两处，操作员需跨页跳转。

## 裁定

新增 `/bss/onboarding`（key=onboarding，menu:onboarding，迁移 000170 授 sysadmin/ops）：
一页聚合 开户全流程——

1. 建档：CustomerPicker 检索既有客户 / 新建客户抽屉（复用 CustomerCreateDrawer）
2. 实名：状态卡 + 代录（复用 RealNameDrawer）+ 后台核验通过/驳回（verify 端点内联）
3. 下单：代客下单（复用 OrderCreateDrawer，新增 fixedCustomerId——工作台已选客户在抽屉内锁定）
4. 推进：GET /orders?customerId= 聚合本客户订单（后端补 customerId 过滤），逐单下一动作
   （核查→预占→收费→激活；取消二次确认）
5. 派单：/dispatch/pool 按 orderId 匹配本客户工单 + WorkerPicker 检索指派

边界：扫码绑定/现场激活归装维师傅端（线下作业），工作台推进到 指派完成为止，
订单表内对 stage9+ 提示"等待装维扫码/激活"。

## 配套改动（缺什么加什么清单）

- `GET /orders` 补 `customerId` 过滤：域层 OrderQuery/PG/Memory 早已支持，仅 HTTP 层透传。
- 注册审核队列抽屉在工作台同步挂载（复用 RegistrationQueueDrawer，不重复造）。
- 实名核验通过/驳回内联到工作台（此前只有实名审核中心整页有 PASS/驳回动作）。

## 放弃了什么

- 工作台内嵌"工单按订单寻址"新查询（GetDispatchTicketByOrder）：接口面+接口测试成本
  高于收益，复用 /dispatch/pool（orderId 字段在响应中）与派单页同模式即可；
  若日后工单量级变大再补专用查询。
- 步骤条不做强制门禁（未实名也可下单——与后端口径一致：下单仅要求客户存在），
  只做状态展示，避免前端重复实现业务校验。

## 与前一条 note 的关系

2026-08-29-admin-onbehalf-catalog.md 的"注册审核不设独立菜单页"维持有效
（注册审核仍是抽屉）；本页是目标显式要求的聚合工作台，属新增裁定而非推翻。

## 验证（102 真实环境）

- make check 全绿 + web 三门禁 + ds-adoption（新页面全部走模式件）。
- UI DOM 断言（cdp，102:5180）：建档（新建客户抽屉全字段提交成功并自动选中）→
  实名（代录提交+核验通过确认弹窗→已实名/步骤条变绿）→ 下单（客户锁定+地址默认
  带出+产品渠道选择→ORD 生成）→ 推进（核查→预占→收费→INSTALLING@8）→
  派单（WorkerPicker 选师傅→指派→"已指派给 王测试"）。全流程无一跳出工作台。
- 过程中发现并修复：OrderCreateDrawer 档案地址"默认带出"只改提示未改表单值
  （保存不可用），d79ed8a3 修复。
- 验收造数全部 acc_ 标记，收尾 acceptance-cleanup --apply + 孤儿巡检 OK。
