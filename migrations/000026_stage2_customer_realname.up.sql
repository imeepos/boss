-- 阶段2:客户实名核验记录(承接 customer.html 实名核验表)。
-- 字段权威:server-ts/src/entities/customer.ts(RealNameVerification)。
BEGIN;

CREATE TABLE real_name_verifications (
    id                 BIGSERIAL PRIMARY KEY,
    customer_id        BIGINT NOT NULL REFERENCES customers(id),
    method             VARCHAR(32) NOT NULL,   -- 人脸/证件OCR/人工/第三方
    verified_at        TIMESTAMPTZ NOT NULL,
    result             VARCHAR(16) NOT NULL,   -- PASS通过/FAIL不通过
    operator_account_id BIGINT,
    operator_name      VARCHAR(64)
);
CREATE INDEX idx_real_name_verifications ON real_name_verifications(customer_id, verified_at);

COMMIT;
