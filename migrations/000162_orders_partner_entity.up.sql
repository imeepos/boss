-- 000162: orders 渠道法人归属(C 案:渠道代售平台产品,佣金归渠道法人,2026-09)。
-- 渠道下单时由 handler 落 partner profile 的 legal_entity_id;直营为 NULL。
-- 佣金计提 COALESCE(partner_entity_id, legal_entity_id)。
ALTER TABLE orders ADD COLUMN partner_entity_id BIGINT;
