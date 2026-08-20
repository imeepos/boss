-- D5 裁定落地:三处实名表合并为统一 verifications(subject_type + subject_id 模式)。
-- 吸收:real_name_verifications(000026)/customer_real_name_verifications(000051)/worker_real_name_verifications(000050)。
-- 不设 UNIQUE(subject_type,subject_id,verified_at):000026 语义是 1:N 审计轨迹,同客户同刻多行合法,
-- 当前态由代码取 ORDER BY verified_at DESC LIMIT 1;subject_id 跨 customers/workers 两主档,只能软引用不加 FK。
BEGIN;

CREATE TABLE verifications (
    id                  BIGSERIAL PRIMARY KEY,
    subject_type        VARCHAR(16)  NOT NULL CHECK (subject_type IN ('customer', 'worker')),
    subject_id          BIGINT       NOT NULL,                      -- customers.id / workers.id(软引用)
    method              VARCHAR(32)  NOT NULL,                      -- 人脸/证件OCR/人工/第三方
    real_name           VARCHAR(64)  NOT NULL DEFAULT '',           -- 申报实名(000026 旧行无,补空串)
    id_card_no          VARCHAR(64)  NOT NULL DEFAULT '',           -- 证件号(脱敏存储,000026 旧行无)
    result              VARCHAR(16)  NOT NULL DEFAULT 'PENDING',   -- PENDING 待核验 / PASS 通过 / FAIL 不通过
    verified_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    operator_account_id BIGINT,                                     -- 核验人账号 id(accounts)
    operator_name       VARCHAR(64),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_verifications_subject ON verifications(subject_type, subject_id, verified_at);

-- 000026 客户实名审计轨迹(无 real_name/id_card_no,落空串)。
INSERT INTO verifications(subject_type, subject_id, method, real_name, id_card_no, result,
                          verified_at, operator_account_id, operator_name)
SELECT 'customer', customer_id, method, '', '', result,
       verified_at, operator_account_id, operator_name
  FROM real_name_verifications;

-- 000051 客户 onboarding 实名。
INSERT INTO verifications(subject_type, subject_id, method, real_name, id_card_no, result,
                          verified_at, operator_account_id, operator_name)
SELECT 'customer', customer_id, method, real_name, id_card_no, result,
       verified_at, operator_account_id, operator_name
  FROM customer_real_name_verifications;

-- 000050 师傅 onboarding 实名。
INSERT INTO verifications(subject_type, subject_id, method, real_name, id_card_no, result,
                          verified_at, operator_account_id, operator_name)
SELECT 'worker', worker_id, method, real_name, id_card_no, result,
       verified_at, operator_account_id, operator_name
  FROM worker_real_name_verifications;

DROP TABLE real_name_verifications;
DROP TABLE customer_real_name_verifications;
DROP TABLE worker_real_name_verifications;

COMMIT;
