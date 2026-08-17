# 三端 mock 数据逻辑对齐(api/DATA-ALIGNMENT)

> 版本 V1.0(2026-08-17)。本文件记录用户端/师傅端/管理后台三份 mock 数据与 openapi 契约的
> **统一事实基线**与对齐修正清单。字段/状态/术语权威源仍为 `docs/contract/{terms,fields,domain-map}.md`,
> 与本文件冲突时以契约为准。

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

1. 新增/修改任一端 mock 数据时,先对照第 1 节事实基线;跨端共享实体(订单/工单/客户/端口/资产/LOID/账单)必须三端同值。
2. 新增订单必须声明:客户(customerId)、环节(stage)、状态(status)、端口/标签/LOID 归属,并检查是否触发第 2 节口径(派单前置、GIS、回调时点)。
3. 环节名展示标签允许各端有措辞差异(如"创建账号"vs"创建认证账号"),但环节序号与 12 环节顺序以 terms.md 为准,禁止增删改序。
