# 渠道目录归属裁定：C 案（平台代售 + 渠道标识 + 佣金归渠道法人）

日期：2026-08-28

## 决策

1. **归属放宽**：`checkPartnerOwnership` 允许渠道下单时客户/产品属于渠道法人 **或** 平台法人(1)——
   渠道可代售平台共享目录产品，也可售自有目录产品。
2. **佣金归属**：orders 新增 `partner_entity_id`（迁移 000162），渠道下单时由 handler 从
   partner profile 落入；佣金计提改用 `COALESCE(partner_entity_id, legal_entity_id)`，
   确保代售平台产品时佣金仍归渠道法人而非平台。
3. **数据域隔离不受影响**：渠道企业客户/产品/区域的自有目录仍可独立建管（A 案能力保留）；
   平台共享目录作为增量获客通道（B 案能力引入），佣金/审计/风控全链路不变。

## why

- 102 现状：5 家 APPROVED 渠道企业法人下 0 客户/0 产品/0 区域地址，按旧口径渠道实际无法独立
  下单（`m3-channel-evidence.md` §3），佣金台账恒空——放量红线。
- C 案让渠道"零建目录即可下单"，放量最快；同时佣金归属清晰、审计留痕完整、数据域隔离不削弱。
- 三年路线图 2028 Q1「渠道与直营订单共用 12 环节语义，数据域隔离有效，佣金可对账」——
  C 案是唯一同时满足三者选项。

## 放弃了什么

- A 渠道自营目录（纯渠道法人硬校验）：需渠道先建目录才能下单，起步太慢。
- B 纯平台共享：佣金归属无法区分渠道/平台，需额外映射层。
- 不加 `partner_entity_id` 列而用 channel→partner 联表推导：channels 与 partner_applications
  无外键关联，联表推导不可靠；显式列最干净。

## 关联

- `internal/domain/order/pg_partner_fraud.go`（归属校验放宽）
- `internal/domain/order/pg_workflow.go`（佣金 COALESCE）
- `migrations/000162_orders_partner_entity.up.sql`
- `docs/ops/m3-channel-evidence.md` §3（三案对比与推荐）
- `docs/ops/m5-acceptance.md` §5（遗留交接第一项）