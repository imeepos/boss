# 数据建模 ↔ 三端页面 对齐审计台账（contract/alignment-audit）

> 版本 V1.0（2026-08-17）｜权威源：`terms.md` / `fields.md` / `domain-map.md` / migrations / `server-ts` 实体
> 定位：把「数据建模（62 表）与 admin/user/worker 三端页面」的对齐状态固化为可销项清单。
> 规则：字段/状态/术语以 `terms.md` 为准；页面列名 ↔ 字段名 ↔ 枚举以 `fields.md` 为准；已销项标 ✅ 并注明提交。

## 1. 结论

主链路可对齐，已消除「双模型字段漂移 / 字段命名不一致 / 状态枚举不一致」三类问题；剩余为按 `domain-map` 明确标注的「待建域」与派生视图，本期不臆造。

## 2. 已修复（销项清单）

### 2.1 枚举与状态（D 类）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| D1 | `RoleCode` 混入 `dispatcher`（实为岗位码），与 `fields.md`/migration 种子的 7 角色集冲突 | 统一为 `customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin` | ✅ e9a2932 |
| D2 | `tag.status` 在 `terms.md` 有、`enums.ts` 与 `Tag` 实体均缺 | 新增 `TagStatus=UNBOUND/BOUND/DISABLED` 并落到 `Tag.status` | ✅ e9a2932 |

### 2.2 字段回写（A 类·实体缺字段）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| A1 | `Account` 缺 `phone`（migrations/fields 有） | 补 `phone` | ✅ e9a2932 |
| A2 | `AuditLog` 缺 `detail`（migrations/fields 有） | 补 `detail(jsonb)` | ✅ e9a2932 |
| A3 | `Tag` 缺 `status` | 补 `status` | ✅ e9a2932 |
| A4 | `Address` 缺 `parent_id`/`geom`（migrations 有） | 补 `parentId`（派生）、`geom(geography)` | ✅ 本台账 |

### 2.3 字段命名统一（B 类）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| B1 | assets 资产编码 `asset_no` vs 实体 `asset_code` | 统一 `asset_code`（fields.md 改） | ✅ e9a2932 |
| B2 | ports 端口编号 `port_no` vs 实体 `port_code` | 统一 `port_code` | ✅ e9a2932 |
| B3 | orders 产品 `product_id→products` vs 实体 `offer_id→product_offers` | 统一 `offer_id→product_offers` | ✅ e9a2932 |
| B4 | quad_link 地址 `addr_id` vs 实体 `address_id` | 统一 `address_id` | ✅ e9a2932 |
| B5 | orders `channel_id/brand_id` 指向不存在的 `channels/brands` | 从 orders 字段表删除；渠道/品牌改口径（见 §3） | ✅ e9a2932 |

### 2.4 缺失实体补齐（A 类·页面有列实体缺失）

| # | 页面 | 缺实体 | 处置 | 状态 |
|:-:|:-----|:-------|:-----|:----:|
| E1 | settings.html 业务参数 | `biz_params` | 补 `BizParam` | ✅ e9a2932 |
| E2 | provlog.html 下发日志 | `provision_logs` | 补 `ProvisionLog` | ✅ f4c4fb7 |
| E3 | customer.html 实名核验 | `real_name_verifications` | 补 `RealNameVerification` | ✅ f4c4fb7 |

### 2.5 Go 骨架字段漂移（A 类·双模型）

| # | 问题 | 处置 | 状态 |
|:-:|:-----|:-----|:----:|
| G1 | `order.Order.ProductID→products`（表不存在） | `OfferID→product_offers` | ✅ 0048f56 |
| G2 | `order.Order.BrandID` / `customer.Customer.BrandID` → brands（表不存在） | `LegalEntityID`（品牌=运营主体） | ✅ 0048f56 |
| G3 | `order.Order.ChannelID` 渠道必填但无 channels 表 | 保留字段，标注 CH 域待建 | ✅ 0048f56 |

## 3. 待建域 / 派生视图（本期不臆造，依据 domain-map）

| 项 | 页面 | 归属域 | 说明 |
|:---|:-----|:-------|:-----|
| 渠道 `channels` | order/dispatch 下单渠道 | CH 渠道经销商 | REQ-ORD-006 必填；`ChannelID` 已保留占位 |
| 告警 `alarms` | alarm.html | MON 网络监控告警 | 待建 |
| 设备监控指标（光功率/丢包率） | device.html | MON | 设备身份复用 `resources`，监控指标待建 |
| 话单 CDR | aaalog.html | AAA 认证计费 | Go `aaa/billing/cdr.go` 已有，TS 未建 |
| 渠道对账 | paycheck.html | BIL/PAY | 派生聚合（对 payments 的汇总），非基表 |

## 4. 遗留轻微不一致

| 项 | 说明 | 建议 |
|:---|:-----|:-----|
| migrations `addresses.parent_id` 物化列 vs TS 实体 `parentId` 派生列 | 均已补 `parentId`，语义一致（派生，应用层不手填） | 保持 |
| migrations `addresses.geom` vs TS `geom` | 已补 `geom(geography)`，阶段8 GIS 预留 | 保持 |

## 5. 销项规则

1. 新缺口按「类别字母 + 序号」登记（A=实体缺字段/缺表、B=命名、D=枚举、G=Go 漂移、E=缺实体）。
2. 字段/状态/术语冲突一律先查 `terms.md`，再回写 `enums.ts` + 实体 + `fields.md`。
3. 页面新增列时，同步在 `fields.md` 登记英文字段名与枚举。
