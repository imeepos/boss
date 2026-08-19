-- 阶段2 延伸:师傅注册 → 后台审核 → 实名认证 闭环(业务支撑平台 onboarding 子域)。
-- 字段权威:docs/contract/fields.md §7.5(本迁移新增)。
-- 设计:注册申请与正式 workers 主档解耦——待审核师傅不入 workers(登录主体仅限在职)。
BEGIN;

-- 师傅注册申请:师傅端自助提交,后台 menu:dispatch 审核队列。
CREATE TABLE worker_registrations (
    id                  BIGSERIAL PRIMARY KEY,
    name                VARCHAR(64)  NOT NULL,                       -- 师傅姓名(申报)
    phone               VARCHAR(32)  NOT NULL,                       -- 联系电话(脱敏存储)
    id_card_no          VARCHAR(64)  NOT NULL,                       -- 证件号(实名认证凭据,脱敏存储)
    group_id            BIGINT       NOT NULL REFERENCES worker_groups(id),
    region_id           INTEGER      NOT NULL,                       -- 服务区域(regions 表)
    status              VARCHAR(16)  NOT NULL DEFAULT 'PENDING',     -- PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回
    review_note         TEXT,                                        -- 审核意见(驳回必填)
    reviewer_account_id BIGINT,                                       -- 审核人账号 id(accounts)
    worker_id           BIGINT,                                       -- 审核通过后写入的 workers.id(空=未建主档)
    submitted_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    reviewed_at         TIMESTAMPTZ                                   -- null=未审核
);
CREATE INDEX idx_worker_registrations_status ON worker_registrations(status);

-- 师傅实名核验:对标客户 real_name_verifications(fields.md §5.1),与 workers 1:1 当前态。
CREATE TABLE worker_real_name_verifications (
    id                   BIGSERIAL PRIMARY KEY,
    worker_id            BIGINT       NOT NULL REFERENCES workers(id),
    method               VARCHAR(32)  NOT NULL,                      -- 人脸/证件OCR/人工/第三方
    real_name            VARCHAR(64)  NOT NULL,                      -- 申报实名(核验基准)
    id_card_no           VARCHAR(64)  NOT NULL,                      -- 证件号(脱敏存储)
    result               VARCHAR(16)  NOT NULL DEFAULT 'PENDING',   -- PENDING 待核验 / PASS 通过 / FAIL 不通过
    verified_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    operator_account_id  BIGINT,                                      -- 核验人账号 id
    operator_name        VARCHAR(64)
);
CREATE INDEX idx_worker_rnv_worker ON worker_real_name_verifications(worker_id, verified_at);

COMMIT;
