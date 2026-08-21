# 订单归属公司由安装地址判定，客户与公司无直接归属关系

> Amended 2026-08-20:未匹配到子公司覆盖时不再拒单,兜底平台总公司处理。
> 机制:migrations/000077 给 legal_entities 加 is_platform 标志(全库唯一,种子 LEG-PLAT 平台总公司),
> root 根区域挂总公司覆盖——"最近覆盖祖先"天然兜底;地址未挂区域也直接归总公司。
> 仅平台总公司未配置(is_platform 无行)才报 ErrPlatformMissing 属配置错误。

日期：2026-08-20

## 决策

订单的 `legal_entity_id`（归属公司/运营主体）由**安装地址（`addressId`）所在经营区域**推导：
地址 → region（regions 物化路径）→ 该区域的运营主体 → 快照写入订单，下单后不可变。
客户（customer）与公司**没有直接归属关系**，`customers.legal_entity_id` 不作为订单归属判据。

## why

- 一个客户可以有多个地址（住宅/商铺/分公司），不同地址可能落在不同运营主体覆盖的区域内；
  若按客户归属判公司，同一客户的第二址订单会挂错主体，出账/开票/分成随之错位。
- 安装地址决定资源核查、端口预占、派单的物理范围，公司边界与网络覆盖边界天然对齐，
  由地址推导归属与履约链路（环节 2~12）自洽。

## 放弃了什么（被否决项）

- **客户归属判公司**：多地址客户场景下产生跨品牌脏数据，被用户裁定否决。
- 渠道判公司：渠道只决定返佣/来源，不决定主体。
- 下单账号 DataScope 判公司：DataScope 仍只做越权过滤，不是业务事实来源。

## 落地要点（实现看代码，此处只记边界）

- regions 到 legal_entities 的覆盖映射已落地（migrations/000076），新建子公司时在其经营区域节点上挂覆盖。
- 未覆盖/未挂区域兜底平台总公司（migrations/000077，is_platform 唯一，root 挂覆盖）。
- `customers.legal_entity_id` 存量字段语义降级为"开户时快照"，不参与订单判定；
  后续清理另行决策。
- `SubmitReq.LegalEntityID` 降为可选校验值，服务端按地址推导为准。

## 关联

- docs/contract/terms.md（订单 12 环节、预占）
- docs/contract/domain-map.md（ORD 域 / BRAND 品牌区域）
- internal/domain/order/order.go（Order.LegalEntityID 快照字段）
- internal/domain/user/org.go（Region / LegalEntity）
