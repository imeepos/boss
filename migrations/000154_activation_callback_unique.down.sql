-- 回滚:激活回调幂等唯一约束。
BEGIN;

DROP INDEX IF EXISTS uq_activation_callbacks_order;

COMMIT;