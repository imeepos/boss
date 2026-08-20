-- 回滚 000059:重建三张旧表并从 verifications 反向迁移(尽力而为)。
-- 限制:
--  1) 旧表 id 序列重置后不与 verifications.id 对齐,反向迁移用新自增 id,原 id 丢失;
--  2) 000026 real_name_verifications 本无 real_name/id_card_no,统一表合并后无法区分行来源,
--     客户 000026 行与 000051 行都回到 customer_real_name_verifications(超集无损,拆分有损);
--  3) created_at 为统一表新增,回滚丢弃。
BEGIN;

CREATE TABLE real_name_verifications (
    id                 BIGSERIAL PRIMARY KEY,
    customer_id        BIGINT NOT NULL REFERENCES customers(id),
    method             VARCHAR(32) NOT NULL,
    verified_at        TIMESTAMPTZ NOT NULL,
    result             VARCHAR(16) NOT NULL,
    operator_account_id BIGINT,
    operator_name      VARCHAR(64)
);
CREATE INDEX idx_real_name_verifications ON real_name_verifications(customer_id, verified_at);

CREATE TABLE customer_real_name_verifications (
    id                   BIGSERIAL PRIMARY KEY,
    customer_id          BIGINT       NOT NULL REFERENCES customers(id),
    method               VARCHAR(32)  NOT NULL,
    real_name            VARCHAR(64)  NOT NULL,
    id_card_no           VARCHAR(64)  NOT NULL,
    result               VARCHAR(16)  NOT NULL DEFAULT 'PENDING',
    verified_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    operator_account_id  BIGINT,
    operator_name        VARCHAR(64)
);
CREATE INDEX idx_customer_rnv_customer ON customer_real_name_verifications(customer_id, verified_at);

CREATE TABLE worker_real_name_verifications (
    id                   BIGSERIAL PRIMARY KEY,
    worker_id            BIGINT       NOT NULL REFERENCES workers(id),
    method               VARCHAR(32)  NOT NULL,
    real_name            VARCHAR(64)  NOT NULL,
    id_card_no           VARCHAR(64)  NOT NULL,
    result               VARCHAR(16)  NOT NULL DEFAULT 'PENDING',
    verified_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    operator_account_id  BIGINT,
    operator_name        VARCHAR(64)
);
CREATE INDEX idx_worker_rnv_worker ON worker_real_name_verifications(worker_id, verified_at);

-- 反向迁移:客户行按 real_name/id_card_no 是否为空分流(空=000026 审计轨迹行)。
INSERT INTO real_name_verifications(customer_id, method, verified_at, result,
                                    operator_account_id, operator_name)
SELECT subject_id, method, verified_at, result, operator_account_id, operator_name
  FROM verifications
 WHERE subject_type = 'customer' AND real_name = '' AND id_card_no = '';

INSERT INTO customer_real_name_verifications(customer_id, method, real_name, id_card_no, result,
                                             verified_at, operator_account_id, operator_name)
SELECT subject_id, method, real_name, id_card_no, result,
       verified_at, operator_account_id, operator_name
  FROM verifications
 WHERE subject_type = 'customer' AND NOT (real_name = '' AND id_card_no = '');

INSERT INTO worker_real_name_verifications(worker_id, method, real_name, id_card_no, result,
                                           verified_at, operator_account_id, operator_name)
SELECT subject_id, method, real_name, id_card_no, result,
       verified_at, operator_account_id, operator_name
  FROM verifications
 WHERE subject_type = 'worker';

DROP TABLE verifications;

COMMIT;
