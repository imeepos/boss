# 优惠券/代金券/赠送 促销体系 审查与设计

> 版本 V1.1（2026-08-22）｜状态：P1~P3 已实施（迁移 000102 + promotion 域 + admin/user 端点），
> 赠送时长阶梯已实施且经用户端续费闭环生效（POST /plans/{planId}/renew）。
> V1.0（2026-08-26 设计稿）原文见下,实施差异以本节为准。
>
> **V1.1 实施增补（赠送时长,原设计未覆盖）**
> - 表：gift_duration_rules（阶梯：scope ALL/PRODUCT + buy_months/gift_months）+ gift_duration_records。
> - 触发：用户端续费 `POST /plans/{planId}/renew {months,payMethod,couponId}` ——
>   金额 = N 月 × 产品基础月费（区域覆盖不参与续费口径），命中阶梯自动加赠，
>   user_plans.contract_end 延长 N+赠送 月，赠送落痕关联缴费流水。
> - 规则语义：取 buy_months ≤ 实购月数的最大档（购 13 月命中 12送3）。
> - 订单链路（2026-08-22 已落地，000104）：下单 SubmitReq 带 buyMonths（0=按月缴），
>   预付费环节4 收 N×月费（区域覆盖口径同出账），阶梯命中回填 orders.gift_months
>   并落 gift_duration_records。
> 审查范围：现有 coupons 实现 + 计费缴费链路 + 契约文档
> 结论：已有功能为"演示级券账本"，不满足市场需求；本文给出升级设计。

## 1. 现状审查

### 1.1 已实现的部分

| 层 | 现状 | 位置 |
|:----|:-----|:-----|
| DB | 单表 `coupons`（coupon_id VARCHAR 主键 / customer_id / name / amount / status / expire_at） | 迁移 000046_userdata |
| admin API | 3 端点：列表 / 发放（POST /coupons）/ 停发（PUT disable） | internal/httpapi/admin/userdata_more*.go |
| user API | GET /coupons?status=（我的券，未接 userdata 时降级静态假数据） | internal/httpapi/user/misc.go |
| 聚合 | 用户详情聚合含 coupons 切片 | internal/domain/customer/userdata/pg_aggregate.go |

### 1.2 市场需求对标缺口（审查结论）

市场主流促销能力（对标电信运营商 BOSS + 电商通用券体系）逐项核对：

| 能力 | 市场预期 | 现状 | 判定 |
|:-----|:---------|:-----|:-----|
| 券模板（面额/门槛/类型） | 满减券/折扣券/代金券分型，门槛（满 X 可用） | 无模板，发放时手填 name+amount | 缺失 |
| 批量发放 | 按客群/活动批量发、每人限领 N 张 | 一次发一张，无批次无限额 | 缺失 |
| 兑换码 | 生成码批次，客户自助兑换 | 无 | 缺失 |
| 缴费抵扣 | 缴费/充值时选券，自动计算抵扣，同事务核销 | billing 域零 coupon 引用，券完全不影响账务 | **核心缺失** |
| 使用门槛与范围 | 满 100 减 20；限产品/限运营主体 | 无 | 缺失 |
| 转赠/赠送 | 转赠好友、邀请奖励、开业赠送 | invite_config 有奖励金额字段但与券无关联 | 缺失 |
| 有效期规则 | 领取后 N 天生效 / 固定窗口 | 仅发放时手填 expire_at，可空 | 缺失 |
| 核销记录与对账 | 每次抵扣留痕（关联 payment），可对账可退券 | 无核销概念 | 缺失 |
| 防重/并发 | 行锁核销，防一券多用 | 无核销故无此问题；一旦接入抵扣即为风险点 | 缺失 |
| 状态契约 | 状态枚举入 terms.md/fields.md | DB 用 active/disabled，user API 映射 available/used/expired，三处不一致且未入契约 | 违规 |

**总结论：已"设计了优惠券"，但只覆盖了发放-展示的最小闭环；抵扣核销、模板化营销、赠送转赠三类市场需求全部缺失。需要升级设计而非推翻——`coupons` 表已有存量数据，走增量迁移。**

## 2. 设计方案

### 2.1 域归属与边界

- 新建 `internal/domain/promotion`（营销促销域）。现有能力域表（domain-map.md 表 1）无营销域，本次登记为新增域 PROMO，阶段=增量；不挂在 customer(userdata) 下——userdata 是用户端演示数据聚合域，券的核心生命周期（模板/核销/对账）超出其边界。
- 与待建 LOY（忠诚度积分）域的关系：LOY 管积分等级，PROMO 管券。积分换券未来经 LOY→PROMO 契约调用，本期不实现。
- 跨域规则（遵守 domain-map.md 第 4 节）：billing 不 import promotion 实现，经接口契约 `CouponDeductor` 注入（app 层 wiring 绑定），保持"禁止跨 A 域 import 他域 implementation"。

### 2.2 数据模型（3 张新表 + 1 张改造）

#### coupon_templates 券模板（L1，依赖 legal_entity）

| 字段（Go PascalCase） | DB 列 | 说明 |
|:----------------|:------|:-----|
| TemplateID | template_id BIGSERIAL PK | |
| LegalEntityID | legal_entity_id BIGINT NOT NULL → legal_entities | 运营主体锚点 |
| Name | name VARCHAR(128) | 模板名（如"开户满减券"） |
| Type | type VARCHAR(16) | FULL_CUT 满减 / DISCOUNT 折扣 / CASH 代金券 |
| FaceValue | face_value BIGINT | 满减/代金券面额（分）；折扣券为折扣率‱（如 8500=85 折） |
| Threshold | threshold BIGINT DEFAULT 0 | 使用门槛（分），0=无门槛 |
| MaxDiscount | max_discount BIGINT | 折扣券封顶（分），可空 |
| ScopeType | scope_type VARCHAR(16) | ALL 全部 / PRODUCT 限产品 / FIRST_ORDER 首单 |
| ScopeRef | scope_ref BIGINT | scope_type=PRODUCT 时指向 product_offers.id |
| TotalQty | total_qty BIGINT | 发行总量，0=不限 |
| IssuedQty | issued_qty BIGINT DEFAULT 0 | 已发数量（发放事务内 +1 行锁） |
| PerCustomerLimit | per_customer_limit INT DEFAULT 1 | 每人限领 |
| ValidDays | valid_days INT | 领取后 N 天有效；与 ValidTo 二选一 |
| ValidFrom / ValidTo | valid_from / valid_to TIMESTAMPTZ | 固定窗口（可空） |
| Status | status VARCHAR(16) | DRAFT / ENABLED / DISABLED |

#### coupons（改造现有表，增量迁移）

保留 coupon_id 存量数据；新增列：

| 新增字段 | DB 列 | 说明 |
|:-----|:------|:-----|
| TemplateID | template_id BIGINT → coupon_templates | 手发存量券可空 |
| Type | type VARCHAR(16) | 冗余自模板，缺省 CASH |
| FaceValue / Threshold / MaxDiscount / ScopeType / ScopeRef | 同名 snake_case | 冗余快照（防模板事后改动影响已发券语义） |
| Source | source VARCHAR(16) | ADMIN_ISSUE 手发 / CAMPAIGN 活动 / REDEEM 兑换码 / GIFT 转赠 / INVITE 邀请奖励 |
| Code | code VARCHAR(32) UNIQUE | 转赠/兑换载体，可空 |
| IssuedAt / UsedAt | issued_at / used_at TIMESTAMPTZ | |
| PaymentID | payment_id BIGINT | 核销关联的缴费流水 |

status 存量映射：`active`→`ISSUED`，`disabled`→`DISABLED`，`used`→`USED`。

#### coupon_redemptions 核销记录（L3，依赖 payments + coupons）

| 字段 | DB 列 | 说明 |
|:-----|:------|:-----|
| RedemptionID | redemption_id BIGSERIAL PK | |
| CouponID | coupon_id VARCHAR(32) NOT NULL → coupons | |
| PaymentID | payment_id BIGINT NOT NULL → payments.id | 核销发生在哪笔缴费 |
| CustomerID | customer_id BIGINT NOT NULL | |
| DeductedAmount | deducted_amount BIGINT NOT NULL | 实际抵扣（分） |
| CreatedAt | created_at TIMESTAMPTZ | |

> payments 无外键（billing 域既有约定"关联完整性由本域应用层保证"），此处同样仅逻辑关联。

#### coupon_codes 兑换码批次（L1）

| 字段 | DB 列 | 说明 |
|:-----|:------|:-----|
| CodeID | code_id BIGSERIAL PK | |
| Code | code VARCHAR(32) UNIQUE | 随机码 |
| TemplateID | template_id → coupon_templates | 兑换后按模板发券 |
| Status | status VARCHAR(16) | UNUSED / REDEEMED / DISABLED |
| RedeemedBy | redeemed_by BIGINT → customers | 可空 |
| RedeemedAt | redeemed_at TIMESTAMPTZ | 可空 |

### 2.3 状态枚举（待实施时同步 terms.md/fields.md）

| 域 | 状态码集 | 说明 |
|:---|:---------|:-----|
| 券模板 template.status | DRAFT / ENABLED / DISABLED | 草稿 / 启用 / 停用 |
| 券实例 coupon.status | ISSUED / USED / EXPIRED / DISABLED | 已发放 / 已使用 / 已过期 / 已停用 |
| 券来源 coupon.source | ADMIN_ISSUE / CAMPAIGN / REDEEM / GIFT / INVITE | 手发 / 活动 / 兑换码 / 转赠 / 邀请 |
| 兑换码 code.status | UNUSED / REDEEMED / DISABLED | 未兑换 / 已兑换 / 已停用 |
| 券类型 coupon.type | FULL_CUT / DISCOUNT / CASH | 满减 / 折扣 / 代金券 |

> user API 对外展示态沿用 available/used/expired（契约既有），由 ISSUED/USED/EXPIRED+有效期派生，DISABLED 不对客户展示。

### 2.4 核心流程

#### 缴费抵扣（本设计的核心闭环）

1. user 端缴费页调 `GET /coupons?status=available&billId=`：promotion 按账单金额过滤门槛，返回可用券+预估抵扣额。
2. 提交缴费 `POST /payments` 携带 `couponId`（可空）。
3. billing.CreatePayment 同事务内：
   - 调注入的 `CouponDeductor.Redeem(couponID, customerID, billAmount)`（promotion 域实现）；
   - `UPDATE coupons SET status='USED', used_at=now(), payment_id=$pay WHERE coupon_id=$1 AND customer_id=$2 AND status='ISSUED' AND expire_at > now() FOR UPDATE`，0 行受影响即返回冲突错误回滚整笔缴费——行锁保证一券多用在此被拦截；
   - 写 coupon_redemptions；
   - 实收 = 账单金额 - 抵扣额，payments.amount 记实收，抵扣额进 redemptions。
4. 退款（payment.status→REFUNDED）联动回退券状态为 ISSUED（额度未过期时）或作废（已过期）。

#### 发放

- admin 手发：选模板 + 客户列表批量发；事务内检查 per_customer_limit 与 total_qty（模板行 `UPDATE ... SET issued_qty=issued_qty+N WHERE template_id=? AND (total_qty=0 OR issued_qty+N<=total_qty)` 行锁防超发）。
- 兑换码：admin 按模板生成 N 个码；user `POST /coupons/redeem {code}` 领取。
- 转赠（赠送）：持有人 `POST /coupons/{id}/gift` 生成 code 并将自己的券置 GIFT_LOCKED 展示态（内部仍 ISSUED + code 非空）；受赠方 redeem 后持有人改写。为控制复杂度，本期转赠仅支持"整券转赠、一人一次"，不做转赠撤回。
- 邀请奖励：invite_config.reward_amount 语义升级为"邀请成功发 coupon_templates 中指定模板券"，在 userdata 邀请链路调 promotion 契约（本期仅留接口，不实现）。

### 2.5 API 增量

| 端 | 端点 | 说明 |
|:---|:-----|:-----|
| admin | GET/POST /coupon-templates，PUT /coupon-templates/:id/disable | 模板管理（perm: menu:userdata 同组，或新 menu:promotion） |
| admin | POST /coupon-templates/:id/issue {customerIds} | 批量发放 |
| admin | POST /coupon-templates/:id/codes {count} | 生成兑换码批次 |
| admin | GET /coupons / POST /coupons / PUT /coupons/:id/disable | 既有 3 端点保留兼容 |
| user | GET /coupons?status=&billId= | 扩展 billId 过滤可用性 |
| user | POST /coupons/redeem {code} | 兑换码领券 / 接收转赠 |
| user | POST /coupons/:couponId/gift | 转赠 |
| user | POST /payments | 增加可选 couponId（billing 契约变更） |

> 实施时每条路由须同步 api/openapi 对应 yaml + 顶层 $ref 行（A 门禁，techniques #25）。

### 2.6 迁移与兼容

- 新迁移（编号取当时最大+1）：建 3 新表；`coupons` 增列 + status 值映射 + type 冗余回填 CASH。
- 存量 userdata.Service 的 ListCoupons/CreateCoupon/DisableCoupon 保留为兼容门面，内部转发 promotion 域（admin 老端点不破坏）。
- user/misc.go 的静态降级假数据删除——违反红线"禁止用假数据替代真实后端"。

### 2.7 分期落地建议

| 期 | 内容 | 依赖 |
|:---|:-----|:-----|
| P1 | 模板表 + coupons 改造 + admin 模板/发放 + terms/fields 契约登记 | 无 |
| P2 | 缴费抵扣闭环（billing 契约注入 + 核销 + 退款回退） | P1 |
| P3 | 兑换码 + 转赠 + 邀请奖励接券 | P1 |

## 3. 放弃了什么（决策记录）

- 不推翻 `coupons` 表重建：存量数据 + 三端已联调，增量迁移成本低于重建。
- 不把 promotion 并入 customer(userdata)：userdata 是演示数据聚合域，核销对账属账务邻域，混入会重复 billing 域踩过的边界问题。
- 不做"券叠加使用"（一单多券）与"券找零"：市场有此玩法但对账复杂度高，首期一单一券。
- 转赠不做撤回/部分转赠：控制状态机规模，GIFT_LOCKED 只是一个展示派生态。
