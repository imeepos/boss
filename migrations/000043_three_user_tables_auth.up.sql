-- 三类用户主体固定为三张表：accounts / customers / workers。
-- 登录边界：accounts=管理后台；customers=客户 App；workers=师傅端。
BEGIN;

COMMENT ON TABLE accounts IS '管理后台登录用户；禁止客户 App 和师傅端复用';
COMMENT ON TABLE customers IS '普通客户主体及客户 App 登录用户；禁止登录管理后台';
COMMENT ON TABLE workers IS '安装师傅主体及师傅端登录用户；禁止登录管理后台';

-- 客户 App 登录：手机号作为登录名，密码只存哈希。
ALTER TABLE customers
    ADD COLUMN password_hash TEXT,
    ADD COLUMN auth_status SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE customers
    ADD CONSTRAINT customers_auth_status_check CHECK (auth_status IN (0, 1));
CREATE UNIQUE INDEX uq_customers_app_login_phone ON customers(phone);
COMMENT ON COLUMN customers.phone IS '客户 App 登录名及联系电话；全局唯一，同一客户主体使用该手机号登录';
COMMENT ON COLUMN customers.password_hash IS '客户 App 密码哈希；仅存哈希，不存明文；为空时不可密码登录';

-- 师傅端登录：工号作为登录名，密码只存哈希。
ALTER TABLE workers
    ADD COLUMN password_hash TEXT;
ALTER TABLE workers
    ADD CONSTRAINT workers_status_check CHECK (status IN (0, 1));
COMMENT ON COLUMN workers.staff_no IS '师傅端登录名及工号；全局唯一';
COMMENT ON COLUMN workers.password_hash IS '师傅端密码哈希；仅存哈希，不存明文；为空时不可密码登录';

COMMIT;
