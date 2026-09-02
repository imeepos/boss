-- 券状态值集归一 + 库层 CHECK 兜底(2026-09 schema 审计 MEDIUM 组)。
-- 机理:000102 把 coupons.status 值集迁移为 ISSUED/DISABLED/USED 并改默认值,但未
-- 封死旧值写入路径——userdata.DisableCoupon 仍写小写 'disabled',与 promotion 域
-- 大写读写并存;report 补偿统计 status<>'ISSUED' 把小写停用行误计入已用。
-- 契约值集:fields.md PROMOTION 节 "ISSUED/USED/EXPIRED/DISABLED"(EXPIRED 为
-- 接口层预留态,代码当前不写,CHECK 放行超集)。
-- 上线姿势:归一数据 → ADD CONSTRAINT NOT VALID(不阻塞 DML)→ VALIDATE 校验。
BEGIN;

UPDATE coupons SET status = 'ISSUED'   WHERE status IN ('active', 'ACTIVE');
UPDATE coupons SET status = 'DISABLED' WHERE status IN ('disabled', 'DISABLED');
UPDATE coupons SET status = 'USED'     WHERE status IN ('used', 'USED');

ALTER TABLE coupons
    ADD CONSTRAINT ck_coupons_status
    CHECK (status IN ('ISSUED', 'USED', 'EXPIRED', 'DISABLED')) NOT VALID;

ALTER TABLE coupons VALIDATE CONSTRAINT ck_coupons_status;

COMMIT;
