# 2026-08-26 新建 promotion 营销促销域(000102)

## 裁定

优惠券/赠送从 `customer/userdata`(用户端演示数据聚合域)独立为 `internal/domain/promotion` 新域,
带 5 张表:coupon_templates / coupon_codes / coupon_redemptions / gift_rules / gift_records,
coupons 存量表增量改造(status 值映射 active→ISSUED 等,新列全带默认值)。

## Why

- userdata 是演示数据聚合域;券的模板化营销/核销对账/退款回退属账务邻域能力,混入会重演
  billing 域踩过的边界问题。
- billing 缴费抵扣需要同事务核销(防一券多用),跨域禁 import 实现 → billing 定义
  `CouponDeductor func(ctx, tx, ...)` 注入点,app 装配绑 promotion 实现,事务由 billing 开启。

## 放弃了什么

- 不推翻 coupons 表重建:存量数据 + 三端已联调,增量迁移成本低于重建;老 INSERT 不改即兼容。
- 不做一单多券与券找零:对账复杂度高,首期一单一券(deductAmount 单券计算)。
- 转赠不做撤回/部分转赠:code 载体 + 派生展示态 gifting,不加独立状态。
- LOY(忠诚度积分)域不并入:积分换券未来经 LOY→PROMO 契约调用。

## 关联

设计:docs/design/promotion-coupon.md;契约:terms.md §4 券枚举、fields.md §8C、domain-map.md PROMO 行。
