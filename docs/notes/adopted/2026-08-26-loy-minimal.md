# 2026-08-26 最小 LOY 积分域与跨域兑换补偿模式

## 裁定

积分归新包 `internal/domain/loy`(000104,账本 loy_point_ledgers + 流水 loy_point_entries),
积分换券价挂 coupon_templates.points_price;兑换流程 = LOY 先扣积分(行锁,独立事务)
→ 调 PROMOTIOIN IssueToCustomer(source=LOYALTY) → 发券失败补偿回补积分。

## Why

- adopted note 2026-08-26-promotion-domain 已裁定积分不并入 promotion;LOY 持账本、
  PROMO 出券,各自事务边界清晰。
- 跨域同事务需要共享 pgx.Tx 穿透两层域(billing→promotion 已有一个注入点,再加一层
  复杂度不可控);兑换失败窗口极小,补偿回补(EXCHANGE_REVERSAL 流水)可审计可对账。

## 放弃了什么

- 不做跨域同事务兑换(代价:极端场景发券成功但客户端超时未见结果,流水可查券号)。
- 缴费自动积分/等级/任务不本期做:全案无具体规则,臆造违反"待建域不臆造"纪律。

## 关联

docs/contract/fields.md §8D;domain-map.md LOY 行;terms.md entry.reason 枚举。
