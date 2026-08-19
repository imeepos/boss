-- 阶段2 延伸:客户自助注册 → 后台审核 → 实名认证 闭环(对标 000050 师傅 onboarding 子域)。
-- 字段权威:docs/contract/fields.md §7.6(本迁移新增)。
-- 设计:注册申请与正式 customers 主档解耦——待审核客户不入 customers(下单需有效客户主档)。
BEGIN;

-- 客户注册申请:客户自助提交(公开端点),后台 menu:customer 审核队列。
CREATE TABLE customer_registrations (
    id                  BIGSERIAL PRIMARY KEY,
    name                VARCHAR(64)  NOT NULL,                       -- 申报姓名
    phone               VARCHAR(32)  NOT NULL,                       -- 联系电话(脱敏存储)
    id_card_no          VARCHAR(64)  NOT NULL,                       -- 证件号(实名认证凭据,脱敏存储)
    legal_entity_id     BIGINT       NOT NULL REFERENCES legal_entities(id),  -- 归属运营主体
    address_id          BIGINT       NOT NULL REFERENCES addresses(id),       -- 装机地址
    region_id           INTEGER      NOT NULL,                       -- 经营区域(regions 表)
    status              VARCHAR(16)  NOT NULL DEFAULT 'PENDING',     -- PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回
    review_note         TEXT,                                        -- 审核意见(驳回必填)
    reviewer_account_id BIGINT,                                       -- 审核人账号 id(accounts)
    customer_id         BIGINT,                                       -- 审核通过后写入的 customers.id(空=未建主档)
    submitted_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    reviewed_at         TIMESTAMPTZ                                   -- null=未审核
);
CREATE INDEX idx_customer_registrations_status ON customer_registrations(status);

-- 客户实名核验:与 customers 1:1 当前态(对标 worker_real_name_verifications)。
CREATE TABLE customer_real_name_verifications (
    id                   BIGSERIAL PRIMARY KEY,
    customer_id          BIGINT       NOT NULL REFERENCES customers(id),
    method               VARCHAR(32)  NOT NULL,                      -- 人脸/证件OCR/人工/第三方
    real_name            VARCHAR(64)  NOT NULL,                      -- 申报实名(核验基准)
    id_card_no           VARCHAR(64)  NOT NULL,                      -- 证件号(脱敏存储)
    result               VARCHAR(16)  NOT NULL DEFAULT 'PENDING',   -- PENDING 待核验 / PASS 通过 / FAIL 不通过
    verified_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    operator_account_id  BIGINT,                                      -- 核验人账号 id
    operator_name        VARCHAR(64)
);
CREATE INDEX idx_customer_rnv_customer ON customer_real_name_verifications(customer_id, verified_at);

COMMIT;
