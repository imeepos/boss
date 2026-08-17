# 三端 mock 数据逻辑对齐(api/DATA-ALIGNMENT)

> 版本 V1.2(2026-08-17)。本文件记录用户端/师傅端/管理后台三份 mock 数据与 openapi 契约的
> **统一事实基线**与对齐修正清单。字段/状态/术语权威源仍为 `docs/contract/{terms,fields,domain-map}.md`,
> 与本文件冲突时以契约为准。

## 0. 关系型事实库(V1.1 新增)

数据是关系型的,不是孤立记录。三端实体一律存于 **`api/mock/db.js` 单一事实库**,以外键关联:

```
customer ─┬─→ order ─┬─→ port(占用/预占)
          │          ├─→ tagEpc(预绑定) ──→ asset
          │          ├─→ loid(认证账号)
          │          └─→ worker(派单)
          ├─→ bill ─→ payment ─→ receipt
          └─→ repairTicket(报障 6 环节) ─→ port/tag/loid/worker
order(stage≥9) ─→ quad(四码: asset/loid/port/addr)
order(stage=12) ─→ gis 同步
```

- 三端视图(`api/mock/data.js`、`api/mock/worker/data.js`、`api/mock/admin/data/*.js`)**只做形状映射,
  不再各自持有实体数据**;改 db 一处,三端自动一致。
- **`api/mock/selfcheck.js`** 固化了 100+ 条关系不变量(引用完整性/未收费不派单/四码时点/GIS 时点/
  端口占用一致/账单三端同口径等),`node api/mock/selfcheck.js` 全绿即对齐成立,改数据后必须重跑。

## 1. 统一事实基线(叙事时间 2025-08-17 10:40)

| 单号 | 客户 | 品类 | 地址 | 环节 | 订单状态 | 三端口径 |
|:-----|:-----|:-----|:-----|:-----|:---------|:---------|
| ORD-20250817-001 | 王先生(1) | 1000M | 望京X·3栋501 | 9 扫码绑定(进行中) | INSTALLING | 用户端订单/详情、师傅端工单(stage 9 SCAN_PENDING)、admin 订单/我的工单 WO-01、扫码记录 MATCH(10:40)、端口 P-001-02(quad P-SPL03-07)RESERVED、预占 RSV-0001、资产 EPC-0001、LOID-88A1、配置下发 PRV-1001 成功 |
| ORD-20250817-002 | 赵女士(4) | 500M | 望京X·5栋302 | 3 端口预占 | RESERVED | 用户端订单、admin 订单、预占 RSV-0002(预占中)。**未收费,三端均不派单**(REQ-CL-001) |
| ORD-20250817-003 | 郑先生(5) | 300M | 望京X·12栋906 | 8 派单(已完成,待领取) | INSTALLING | admin 订单、师傅端 todo 待领取、调度池 WO-20250817-05(跨区) |
| ORD-20250817-004 | 王先生(1) | 宽带变更 | 望京X·3栋501 | 8 派单 | INSTALLING | admin 订单、师傅端 todo、admin 我的工单 WO-02(待接单) |
| ORD-20250816-018 | 孙先生(3) | 300M | 望京Y·1栋101 | 12 | DONE | 用户端订单(可评价)、admin 订单/工单 WO-03、端口 P-001-03 USED、四码 EPC-0003/LOID-88A3/P-SPL02-03/A-1-101、GIS 已同步 |
| ORD-20250817-009 | 刘女士(7) | 拆机 | 望京X·2栋902 | 扫码解绑待办 | — | admin 拆机单、师傅端拆机详情(quad EPC-0110/P-SPL03-01/A-2-902)、端口 P-001-01 USED |
| TKT-20250817-012 | 陈先生(8) | 断网抢修 | 望京X·10栋1801 | 4/6 修复中 | — | admin 投诉工单、师傅端 doing(SLA 52min)。**不属于用户端王先生** |
| TKT-20250817-015 | 王先生(1) | 断网抢修 | 望京X·3栋501 | 4/6 修复中 | — | 仅用户端报障列表/详情(张师傅 138****8899 处理) |

## 2. 全局口径约定

- **stage 语义**:三端统一为"当前所处环节序号"(最近到达的环节;上一步刚完成且下一步未开始时停在上一步)。师傅端工单时间轴的"已完成 N 环节"由 `stages` 数组表达,不再用 ticket.stage 表达完成数。
- **未收费不派单**:stage<4(合同收费未完成)的订单,任何端不得出现工单/派单记录(terms.md 第 5 节)。
- **GIS 同步**:仅 stage=12 完成的订单进入 GIS 已同步;未完成订单一律待同步。
- **激活回调(环节 11)**:仅 stage≥11 的订单出现回调记录;ORD-001 在环节 9,激活状态为"待扫码绑定完成后触发"。
- **四码字段**:统一 `assetCode/customerCode/portCode/addrCode`(fields.md 5.1:第二码为客户域;`userCode` 系历史笔误)。用户码取值=客户认证账号 LOID,与师傅端一致。
- **客户主数据**:admin 客户档案 customerId 1~8 覆盖全部被订单/工单/欠费引用的客户;billing 欠费行的 customerId 与之对齐(周女士=6)。
- **计费对账**:王先生 2025-08 账单 ¥158 三端一致为未缴;缴费流水 PAY20250725001(¥158,2025-07-25)用户端=admin 端;未缴账期不出现在可开票列表与凭证中。
- **地址码**:admin/worker 用 `A-<栋>-<门>`(如 A-3-501);用户端档案 addressId 为页面演示值 `ADDR-001/ADDR-002`,映射关系 ADDR-001≙A-3-501、ADDR-002≙A-1-101。
- **空闲端口**:师傅端 idlePonPorts 不包含已被 ORD-001 预占的 P7。

## 3. 本次修正清单

> V1.2 追加(师傅域全量后台管理): 师傅端用到的**全部**数据(工单/四码/激活/收款/绩效提成/评价/
> 消息/公告/FAQ/物料/工具/旧件回收/设备健康/抢单池/调度会话/排期/接单设置)一律入库
> `api/mock/worker/entities.js`(经 `db.js` 聚合,加入 `TABLE_NAMES` 可 CRUD),师傅端视图
> `worker/data.js` 全量派生(实体视图用 getter,admin 改动即时生效);admin 端新增
> `api/mock/admin/data/worker.js` 路由 + `docs/admin/worker.html`「师傅管理」页 + 契约
> `api/openapi/admin/worker.yaml`,菜单挂"订单与工单"分组。selfcheck 第 16/17 节固化
> 师傅域引用完整性与 admin 覆盖不变量。

| # | 文件 | 修正 |
|:--|:-----|:-----|
| 1 | `api/mock/data.js` | 王先生报修单改为 TKT-20250817-015(3栋501),师傅电话改 138****8899(原 7788 为陈先生号码) |
| 2 | `api/mock/data.js` | 删除 PAY20250817001 凭证、发票可开票期移除未缴的 2025-08;home 套餐名统一"1000M 极速宽带" |
| 3 | `api/mock/routes/user.js` | 凭证兜底键改为 PAY20250725001 |
| 4 | `docs/user/faultdetail.html` | 默认工单号同步为 TKT-20250817-015 |
| 5 | `api/mock/worker/data.js` | 删除 ORD-20250817-002 已接单工单/排期/超时消息(未收费不派单);ORD-001 stage 8→9;今日接单数 3→2 |
| 6 | `api/mock/worker/data.js` | 抢修/拆机详情改用各自 quad(EPC-0088/P-SPL05-03/A-10-1801、EPC-0110/P-SPL03-01/A-2-902),不再复用王先生的 quad |
| 7 | `api/mock/worker/data.js` | 激活状态改"待扫码绑定完成后触发";idlePonPorts 去掉已预占的 P7 |
| 8 | `api/mock/admin/data/order.js` | 我的工单 WO-02 改挂 ORD-20250817-004;回调 CB-8841 改挂 ORD-20250817-000(ORD-001 未到环节 11) |
| 9 | `api/mock/admin/data/billing.js` | 王先生 2025-08 账单 ¥158 未缴(原 ¥299 已缴与用户端矛盾);缴费流水对齐 PAY20250725001;周女士 customerId 4→6 |
| 10 | `api/mock/admin/data/customer.js` | 补齐被引用客户:赵女士 4、郑先生 5、周女士 6、刘女士 7、陈先生 8 |
| 11 | `api/mock/admin/data/oss.js` | RSV-0002 改预占中(ORD-002 仍持预留),新增 RSV-0004 超时样例挂 ORD-20250816-021;端口 P-001-01 改 USED/2栋/挂拆机单 ORD-009 |
| 12 | `api/mock/admin/data/intel.js` | GIS 同步行重排:仅 ORD-20250816-018 已同步,在途订单待同步 |
| 13 | `api/mock/admin/data/quad.yaml`+`quad.js`+`docs/admin/quadlink.html` | 四码字段 userCode/addressCode → customerCode/addrCode,与 worker 契约及 fields.md 5.1 一致 |
| 14 | `docs/admin/billing.html` | STATUS_TAG 补"未缴"标签色 |

## 4. 维护规则

1. **改实体先改 `db.js`**,三端视图随派生自动一致;禁止在视图文件手抄实体数据。
2. 改完必须跑 `node api/mock/selfcheck.js`,退出码非 0 视为破坏对齐。
3. 新增订单必须声明:客户(customerId)、环节(stage)、端口/标签/LOID 归属,并检查是否触发第 2 节口径(派单前置、GIS、回调时点)。
4. 环节名展示标签允许各端有措辞差异(如"创建账号"vs"创建认证账号"),但环节序号与 12 环节顺序以 terms.md 为准,禁止增删改序。
5. **用户端数据一律后端管理(V1.2 新增)**:用户端用到的全部数据必须在 admin 端可管理,不得有遗漏。用户域实体
   (账户/地址簿/套餐订购/增值服务目录与订购/通知订阅/FAQ/消息/优惠券/邀请配置/流量/自助排障指南/协议/
   余额/充值面额/电子发票/投诉/实名记录/产品卖点/账单明细项)统一定义于 `api/mock/user/entities.js`,
   经 `db.js` 聚合并挂 uuid CRUD;admin 端按产品化结构管理:用户列表 `GET /users` + 用户详情聚合
   `GET /users/{customerId}`(单接口返回该用户在用户端可见的全部数据,页面 `docs/admin/user.html` →
   `user-detail.html?customerId=X` 分 Tab 聚合),全局配置(增值服务目录/FAQ/排障指南/协议/邀请/充值面额/
   产品卖点)在 `docs/admin/userdata.html`(用户端配置);管理视图与写操作见 `api/mock/admin/data/userdata.js`
   (契约 `api/openapi/admin/userdata.yaml`,聚合入口 `api/openapi/admin.yaml`),
   用户端视图由 `api/mock/data.js` 派生,selfcheck 第 18~22 组固化"引用完整/发票口径/两端同源/admin 可管理/详情聚合同源"不变量。
   其余用户端数据(客户档案/产品资费/订单/账单/缴费/报障)沿用既有 admin 域(customer/billing/order)。
