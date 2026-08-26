-- 激活回调幂等:同订单唯一回调行(环节11 激活回调落账闭环)。
-- 背景:activation_callbacks 消费端(列表/重试/补偿台账)齐备但生产端为零——
-- NotifyActivation 只推进 stage 不落账,补偿台账扫 FAILED 永远是死配置。
-- 本迁移加 order_id 唯一约束,配合 AppendActivationCallback 的 ON CONFLICT upsert,
-- 保证重复确认(worker/admin 重试)不产生重复行,幂等重放。

BEGIN;

CREATE UNIQUE INDEX uq_activation_callbacks_order ON activation_callbacks(order_id);

COMMIT;