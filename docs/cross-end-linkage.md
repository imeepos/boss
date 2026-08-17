# 三端功能联动清单（跨端成节点对账）

> 目标：以需求《BOSS综合业务支撑平台-需求全案.md》§2.4「6 业务闭环」与 §3.3「业务闭环状态机」为唯一权威，
> 把每一闭环的**环节**逐点映射到 admin / user / worker 三端的**成节点**（相互承接的页面/入口），
> 证明「每个流程可跑通、两端间有对应成节点」；凡映射断裂处即为需补全/修正的联动缺口。
>
> 权威索引：三端各自的 `simulation-register.md`（角色内模拟）→ 本清单（跨端联动）→ 最终落地 `*.html` + `menu.js` / `nav.js`。
> **术语/状态/环节以 `docs/contract/terms.md` 与 `docs/contract/domain-map.md` 为单一事实源**（12 环节含「4 合同收费」、
> 订单状态 PENDING/RESERVED/INSTALLING/DONE、端口 IDLE/RESERVED/USED/DISABLED、资产 IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED）。

## 〇、三端角色与承接边界

| 端 | 目录 | 角色 | 主导口径 |
|---|---|---|---|
| admin | `docs/admin/` | 客服/运营/财务/调度/资源/资产/系统（7 角色） | 订单受理、资源、计费、派单、对账、审计 |
| user | `docs/user/` | 客户（自助门户） | 下单、缴费、报障、变更、评价 |
| worker | `docs/worker/` | 装维师傅 | 接单、上门、扫码绑定、激活、修复、换件、拆机 |

> 成节点定义：同一业务事实在两（或三）端均有可见承接页/入口，且字段/状态一致 —— 例：用户「报障」→ 客服「投诉受理」→ 师傅「修复工单」三段同源单号。

## 一、6 闭环 × 环节 × 三端成节点总表

> 执行方列取自需求 3.3；「成节点」= 该端承载该环节的页面；「✘」= 该端当前无承接页（联动缺口）。

### 1.1 新装开通闭环（REQ-CL-001 · 12 环节）

| # | 环节 | 执行方 | admin 成节点 | user 成节点 | worker 成节点 |
|---|---|---|---|---|---|
| 1 | 下单 | 客户/客服 | `order.html` | `order.html` | （只读跟踪）`order.html` |
| 2 | 资源核查 | 系统 | `resource.html` | `order.html`（进度） | `order.html`（进度） |
| 3 | 端口预占 | 系统 | `reserve.html` | `order.html`（进度） | `order.html`（进度） |
| 4 | 合同收费 | 财务/客服 | `payment.html` | `pay.html`/`payresult.html` | ✘（无需） |
| 5 | 标签预绑定 | 系统 | `tag.html` | `order.html`（进度） | `order.html`（预绑定四码） |
| 6 | 创建账号 | 系统 | `loaccount.html` | ✘（后台可见，可回填进度） | `order.html`（进度） |
| 7 | 预下发配置 | 系统 | `provision.html`/`template.html` | `order.html`（进度） | `report.html`（下发日志） |
| 8 | 派单 | 调度/系统 | `dispatch.html` | `order.html`（进度） | `orders.html`（领取） |
| 9 | 扫码绑定 | 装维 | `scanlog.html`/`quadlink.html` | `order.html`（进度） | `scan.html`/`report.html` |
| 10 | 激活 | 装维/系统 | `order.html` | `order.html`（进度） | `activate.html`/`report.html` |
| 11 | 激活回调 | 系统 | `callback.html` | `order.html`（进度） | `report.html` |
| 12 | 更新 GIS | 系统 | `gis.html` | `order.html`（进度） | ✘（后台异步） |

### 1.2 报障服务闭环（REQ-CL-006 · 6 环节）

| # | 环节 | 执行方 | admin 成节点 | user 成节点 | worker 成节点 |
|---|---|---|---|---|---|
| 1 | 报障 | 任一渠道 | `complaint.html` | `fault.html` | ✘（诉求由用户/客服发起） |
| 2 | 诊断 | 受理 | `complaint.html`（诊断） | `faultdetail.html`（进度） | `repair.html`（诊断结果） |
| 3 | 派单 | 调度 | `dispatch.html` | `faultdetail.html`（进度） | `repair.html`/`orders.html` |
| 4 | 修复 | 装维 | `complaint.html`（处理） | `faultdetail.html`（进度） | `repair.html` |
| 5 | 复核 | 系统 | `check.html`/`callback.html` | `faultdetail.html`（进度） | `repair.html`（复核） |
| 6 | 回访 | 关闭 | `complaint.html`（回访） | `rate.html` | `feedback.html` |

### 1.3 预付生命周期闭环（REQ-CL-002）

| 环节 | admin | user | worker |
|---|---|---|---|
| 充值/续费 | `payment.html` | `topup.html` | ✘ |
| 到账通知 | `payment.html` | `messages.html` | `messages.html` |
| 余额预警 | `billing.html` | `home.html`/`messages.html` | ✘ |
| 到期停机 | `arrears.html`/`stopsrv.html` | `myplan.html` | ✘ |
| 复机 | `stopsrv.html` | `myplan.html` | ✘ |
| 流失预警 | `analytics.html` | ✘ | ✘ |

### 1.4 后付账单周期闭环（REQ-CL-003）

| 环节 | admin | user | worker |
|---|---|---|---|
| 出账 | `billing.html` | `bills.html`/`bill.html` | ✘ |
| 账单通知 | `billing.html` | `messages.html` | `messages.html` |
| 缴费 | `payment.html`/`paycheck.html` | `pay.html`/`payresult.html` | `charge.html`（现场收款） |
| 催缴 | `arrears.html` | `messages.html`/`myplan.html` | ✘ |
| 停机 | `arrears.html`/`stopsrv.html` | `myplan.html` | ✘ |
| 复机 | `stopsrv.html` | `myplan.html` | ✘ |
| 关账对账 | `paycheck.html` | ✘ | ✘ |

### 1.5 企业专线闭环（REQ-CL-004）

> 企业/政企客户与家庭用户共用 `docs/user/`（门户未按客户类型单独立项，属 P1 延后）；
> admin 侧由「客户与资费（`product.html` 含 `政企专线 100M`）+ 订单与工单 + 网络资源」复用承载，
> 品牌隔离维度见 `company.html`（`LEG-A 主品牌·企业`）。无独立企业门户页，非断链，P1 阶段补门户。

| 环节 | admin | user | worker |
|---|---|---|---|
| 方案报价 | `customer.html`/`product.html`（政企专线） | `product.html` | ✘ |
| 资源核查建设 | `resource.html`/`expand.html` | ✘ | ✘ |
| 开通 | `order.html`/`provision.html` | `order.html` | `order.html`（复用新装） |
| SLA 保障 | `alarm.html` | ✘ | `repair.html`（SLA 倒计时） |
| 计费账期 | `billing.html` | `bill.html` | ✘ |
| 续约流失 | `analytics.html`/`report.html` | `myplan.html` | ✘ |

### 1.6 批发服务闭环（REQ-CL-005）

> 批发为 P2 品牌（`company.html` 的 `LEG-C 批发品牌`），当前无独立批发门户/工单，
> 用量采集→计费→对账→毛利由 `aaalog`（话单）、`billing`（出账）、`paycheck`（对账）、`analytics`（毛利）复用承载。非断链，P2 阶段补批发门户与结算。

| 环节 | admin | user | worker |
|---|---|---|---|
| 签约 | `customer.html`/`company.html`（LEG-C） | ✘（未立项） | ✘ |
| 供应/用量 | `resource.html`/`aaalog.html` | ✘ | ✘ |
| 计费/对账/毛利 | `billing.html`/`paycheck.html`/`analytics.html` | ✘ | ✘ |

## 二、跨端联动缺口与修复

> 由总表映射出的「两端间无成节点」或「状态口径不一致」项，逐一处置。状态：✅ 已修复。

| # | 缺口 | 影响闭环 | 处置 | 状态 |
|---|---|---|---|---|
| L1 | 三端 `order.html` 进度「11 环节」与需求「12 环节」、worker `report.html`「10/12」口径不一致 | 1.1 | 统一 12 环节，补「合同收费」节点（下单→…→合同收费→…→更新GIS），`admin/user/worker order.html`、`orders.html`、`home.html` 全部对齐 | ✅ |
| L2 | admin 端 129 处 `href="#"` 死链，无跨端成节点 | 全部 | admin 走 `menu.js` 侧栏可达（非断链）；四码等跨端承接页已补成节点口径 | ✅ |
| L3 | 报障闭环单号三端不一（admin `BS-*` / user `FR-*` / worker `TKT-*`）且 user 报障 4 环节 vs 需求 6 环节 | 1.2 | 统一报障工单号 `TKT-*`（报障修复），投诉单号独立 `CP-*`；user `faultdetail.html` 补全 6 环节（报障→诊断→派单→修复→复核→回访） | ✅ |
| L4 | worker `charge.html` 现场收款无入口 | 1.3/1.4 | `sign.html` 补「现场收款」入口链接 `charge.html` | ✅ |
| L5 | 拆机三端同源单号不一致（admin `DSM-*`/`DMS-*` + `HG-*`，与 worker `ORD-*`+`EPC-*` 冲突） | 拆机 | 统一 `ORD-*`（退订拆机单）+ `EPC-*`（光猫），admin `dismantle.html` 对齐 worker 值 | ✅ |
| L6 | 激活回调状态三端矛盾（worker `activate` 未生效 vs admin `callback` 成功） | 1.1 | 统一 `ORD-20250817-001` 激活回调为「失败/重试中」，worker `activate.html` 环节号由 11/12 改 10/12 | ✅ |
| L7 | 四码合一编码体系分裂：admin `quadlink/check` 用 `A-2025/U-10/P-001/L3-`，worker 用 `EPC/LOID/P-SPL/A-` | 四码 | 统一四码口径：资产码 `EPC-*`、用户码 `LOID-*`、端口码 `P-SPLxx-yy`、地址码 `A-x-yyy`；`quadlink/check` 对齐 worker 权威值 | ✅ |
| L8 | 设备/资产码混用 `HG-*`/`RFID-*`，与权威 `EPC-*` 冲突 | 四码/资产 | `scanlog` 改 `EPC-*`、`replace` 改 `EPC-*`、`provlog` 改 `EPC-*` | ✅ |
| L9 | worker 4 个孤儿页（`activate/retire/dismantle/complaint`）无入口，流程跑不通 | 多处 | `order.html` 补 拆机/投诉/激活入口；`replace.html` 补 旧件回收入口 | ✅ |
| L10 | 资产两级身份（资产ID `A-2025` ↔ 标签 `TAG-*` ↔ EPC 码）未显式，四码「资产码=EPC」与资产台账「资产ID=A-2025」对不上 | 资产/四码 | `tag.html` 补「EPC 码」列，建立 `资产ID(A-2025)` ← `标签(TAG)` ← `EPC码` 映射链 | ✅ |
| L11 | 抢修单用 `ORD-*` 编码，与报障修复 `TKT-*` 口径冲突 | 1.2 | worker `hall.html` 抢修单 `ORD-20250817-006` → `TKT-20250817-006` | ✅ |
| L12 | worker `schedule.html` 排期用简写 `ORD-001/002/003`，与全库 `ORD-20250817-*` 不一致 | 1.1 | 补全 `ORD-20250817-001/002/003` | ✅ |
| L13 | master 单 `ORD-20250817-002` 地址三端不一致（admin/user `5栋302` vs worker `7栋1203`）；`ORD-016-018` 日期不一致（admin `-016-` vs user `-010-`） | 1.1 | worker 地址统一 `5栋302`；user 单号统一 `ORD-20250816-018` | ✅ |
| L14 | 预付生命周期「余额预警/到期停机/复机」环节 user 端无消息成节点（messages 仅后付账单类） | 1.3 | user `messages.html` 补「余额预警」「到期停机」通知（下钻 topup），建立停复机联动 | ✅ |
| L15 | 台风应急批量复测（MON-004）`EMG-*` 任务 worker 端有承接（hall），admin 端无下发成节点 | 告警/应急 | admin `alarm.html` 补「台风应急·批量复测」卡片，与 worker `hall.html` 的 `EMG-20250817-001` 同号对齐 | ✅ |
| L16 | worker `messages.html` 改派通知单号 `ORD-005 抢修` 用 `ORD-*`（孤儿单，且抢修应 `TKT-*`） | 1.2 | 改 `TKT-20250817-005`，消除孤儿单号 | ✅ |
| L17 | 端口预占看板 `reserve.html` 用占位订单 `ORD-...-001/002/018` + 端口码 `P-望京X-*`/`SPL-01-07` 非权威格式；`resource.html` 端口台账用 `P-001-xx`+`分光器-3`+截断订单 `ORD-20250817`，与四码 `P-SPLxx-yy` 不一 | 1.1/四码 | `reserve` 订单补全 `ORD-20250817-001/002`、`ORD-20250816-018`，端口码统一 `P-SPLxx-yy`；`resource` 补「四码端口码」桥接列、分光器 `SPL-03`、订单补全 | ✅ |
| L18 | 师傅端「转单/改派」（`transfer.html`/`service.html` 退回调度池）在 admin 调度侧无承接成节点；且 `dispatch.html` 师傅视图用 `WO-*` 号未映射 `ORD-*`，师傅归属与 master 单（`ORD-001`=张师傅）矛盾 | 派单/改派 | admin `dispatch.html` 补「改派/转单处理」卡片（承接师傅退回调度池）；师傅视图补「订单号」列，`WO-* ↔ ORD-*` 显式对齐（ORD-001=张师傅·3栋501） | ✅ |
| L19 | 认证账号 LOID 编码体系分裂：admin `loaccount/stopsrv/aaalog` 用 `LOID-310xxx`，worker/user 用 `LOID-88Ax`，用户码（四码）不一 | 四码/AAA | 统一 `LOID-88Ax`：王先生=88A1、赵女士=88A2、孙先生=88A3、李女士=88A4，`loaccount/stopsrv/aaalog` 对齐 | ✅ |
| L20 | 资产台账 `asset.html` 只列「标签 TAG」未显式 EPC 码，四码「资产码=EPC」无法从资产台账反查 | 资产/四码 | `asset.html` 资产清单补「EPC 码」列，贯通 `资产ID(A-2025) ← 标签(TAG) ← EPC` 链 | ✅ |
| L21 | GIS `gis.html` 为纯占位，无地址码(ltree)/订单/资产联动；`quadlink/check` 冲突样例数据与 master 客户(EPC-0002/88A2)不一致 | 1.1/GIS | `gis.html` 补「地址层级码→GIS实体联动」表（`A-3-501`→ORD-001→EPC-0001）；`quadlink/check` 冲突样例对齐 master 数据（EPC-0002 端口不一致） | ✅ |
| L22 | 配置下发模板号/任务号分裂：worker `report` 用 `GPON-1000M-v3`，admin `template/provision` 用「光猫预配置」无编号、`provision` 用 `CFG-*` 任务号、`provlog` 用 `PRV-*` | 1.1 预下发 | 统一模板号 `GPON-1000M-v3`/`GPON-500M-v2`/`OLT-VLAN-v2`，任务号统一 `PRV-*`，设备统一 `EPC-*`/`OLT-*`，`template/provision/provlog` 三处与 worker 对齐 | ✅ |
| L23 | 告警中心 `alarm.html` 只列告警无「派单处理/确认/关闭」动作，告警无法流转到派单→修复闭环（S14 OLT 离线告警→派单→恢复→消除） | 告警/修复 | `alarm.html` 告警表补「派单处理/确认/关闭」操作列，形成告警→派单→修复闭环 | ✅ |
| L24 | 报障闭环单号三端不齐：user `faultdetail` 用 `TKT-20250816-001`（无法上网/李师傅）在 admin/worker 无承接，仅 `TKT-20250817-012` 为 admin+worker 共享，user 侧无共享单号 | 1.2 | user `fault.html`/`faultdetail.html` 统一为共享单 `TKT-20250817-012`（单户断网/陈先生·10栋1801/张师傅），6 环节时间轴与 worker `repair` 对齐 | ✅ |
| L25 | 变更/迁址单号 user 侧缺失：`change.html`/`move.html` 提交后无成节点单号，与 worker `ORD-20250817-004/008`（变更/迁址）脱节；拆机回收资产码 `EPC-0005` 与拆机在网光猫 `EPC-0110` 不一致 | 变更/资产 | `change` 提交标注生成 `ORD-20250817-004`、`move` 标注生成 `ORD-20250817-008`（迁址工单）；`retire` 拆机回收改 `EPC-0110`（对齐 dismantle） | ✅ |
| L26 | 变更/迁址单 `ORD-20250817-004/008` 及新装 `ORD-20250817-003` 在 admin `order.html` 无承接行（订单管理仅列 001/002/018 新装单），变更闭环 admin 成节点缺失 | 变更 | admin `order.html` 订单表补 `ORD-003`（300M）、`ORD-004`（宽带变更）、`ORD-008`（迁址移机）行，地址对齐王先生 3栋501 | ✅ |
| L27 | 抢修单 `TKT-20250817-005/006`（worker hall 抢单池）在 admin dispatch 无承接，且调度工单池 `WO-05/06/07` 未映射来源单号 | 报障/派单 | admin `dispatch.html` 工单池补「来源单」列，`WO-06↔TKT-005`、`WO-07↔TKT-006`、`WO-05↔ORD-003` 显式对齐 | ✅ |
| L28 | 停复机状态三页矛盾：赵女士 arrears「待停机·在线」vs loaccount/stopsrv「停机」；孙先生 stopsrv「复机」vs arrears「已停机35天」vs loaccount「待激活」 | 1.3/1.4 | 统一赵女士=已停机·已停服；孙先生=已复机·在服（复机生效 09:30 后），loaccount 改「在服」、arrears 改「已缴清·已复机」 | ✅ |
| L29 | 客户身份冲突：赵女士同时作为「订单demo 500M 新装（端口预占 3/12）」与「计费demo 1000M 欠费停机 ¥199」，套餐/状态矛盾；孙先生 customer「已停机」与复机状态矛盾 | 1.3/1.4/档案 | 解耦双叙事：赵女士保留为订单demo（500M 新装），欠费停机客户改「吴女士」（1000M ¥199，LOID-88A5）；孙先生统一「在网/已复机」 | ✅ |
| L30 | 状态术语与契约对不齐：resource/asset 用「故障」描述端口/资产非可用态，契约 terms.md 定义为端口「DISABLED 禁用」/ 资产「MAINTENANCE 维修」，缺显式映射 | 端口/资产 | 见 contract/terms.md 权威枚举；三端页面「故障」= 端口 DISABLED（禁用）、资产 MAINTENANCE（维修）的 UI 标签，已在本文档 §四 权威口径表登记映射 | ✅ |
| L31 | 状态枚举与 fields.md/terms.md 对不齐：order「已派单/变更中」、asset「故障」、tag「库存」、port「故障」均非契约枚举 | 全状态 | 统一契约枚举：order 待核查/已预占/装维中/已完成、asset 在库/在用/维修/报废、tag 已绑定/未绑定、port 空闲/预占/在用/禁用；order.html 补「待核查」筛选态 | ✅ |

## 三、成节点落地规则

1. 成节点 = 两端页面可见同一业务单号/字段（如 `ORD-*`/`WO-*`），状态一致。
2. 每个闭环的主链路两端（发起端 → 承接端）页面必须能通过文字/链接互相指认，不求真实跳转（静态文档），但不得出现「只在一端存在、另一端无承接页」的孤儿环节。
3. 缺口按「L#」编号在本清单登记，修复后回填「处置」列并勾选完成。

## 四、权威编码口径（成节点一致性基线）

| 编码 | 含义 | 权威来源 | 承载页 |
|---|---|---|---|
| `ORD-*` | 订单（下单→开通主线） | 三端共享 | `order.html`×3 |
| `WO-*` | 派单工单 | admin `dispatch` | `dispatch.html` |
| `TKT-*` | 报障修复工单（6 环节） | user→admin→worker 共享 | `faultdetail`/`complaint`/`repair` |
| `CP-*` | 投诉单（资费/服务投诉，非断网） | admin `complaint` | `complaint.html` |
| `EPC-*` | 电子标签码（四码·资产码） | 标签台账 | `tag`/`scanlog`/`quadlink` |
| `LOID-*` | 认证账号/用户码（四码·用户码） | `loaccount` | worker `order`/`activate` |
| `P-SPLxx-yy` | 端口码（四码·端口码） | `resource` | worker `order` |
| `A-x-yyy` | 地址层级码（四码·地址码，ltree） | `address` | worker `order` |
| `A-2025xxxx` | 资产台账内部主键（区别于 EPC 标签码） | `asset` | `asset`/`stock` |
| `TAG-*` | 标签记录主键（承载 EPC 码） | `tag` | `tag.html` |

> 关键区分：**资产 ID `A-2025xxxx`（台账主键）≠ 四码「资产码」`EPC-*`（贴在设备上的电子标签码）**。
> 四码对账用的是 EPC；资产台账/盘点用的是资产 ID，二者经 `tag.html` 的「EPC 码」列桥接。

## 五、契约「待建域」说明（非断链，P2/P3 期）

> 依据 `docs/contract/domain-map.md` §1 主映射表「待建」标注，以下能力域本期无三端承接页，属**规划内缺失**而非联动断链：

| 待建域 | 域码 | 现状 | 计划 |
|---|---|---|---|
| 批发结算 | WHO | 仅 `company.html`/`analytics.html` 有 `LEG-C 批发品牌`（品牌维度），无独立批发门户/结算 | P2 |
| 渠道经销商 | CH | `worker/performance.html` 有「装维师傅提成」，但非 `CH 经销商多级佣金`；经销商分销未落地 | P2 |
| 忠诚度积分 | LOY | 无积分/忠诚度承接页 | P3 |
| 营收保障 RA | RA | 无话单/收入稽核管理页 | P3 |
| 防欺诈案件管理 | FMS | `worker/order.html` 有「风控名单拦截卡」（FMS-004），但 FMS-005 欺诈案件管理未落地 | P3 |
| 结算互连 | SET | 无结算互连/争议处理页 | P3 |

> 已落地域（21 域中的 14 域）在三端均有承接页，见 §一 成节点总表。
