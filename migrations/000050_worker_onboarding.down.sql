-- 回滚 000050:师傅注册 → 审核 → 实名认证 闭环。
BEGIN;
DROP TABLE IF EXISTS worker_real_name_verifications;
DROP TABLE IF EXISTS worker_registrations;
COMMIT;
