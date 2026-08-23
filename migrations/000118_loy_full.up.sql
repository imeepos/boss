-- 完整 LOY(2028 Q2):等级、任务、缴费自动积分、回滚、过期。
-- 账本 loy_point_ledgers 仍为唯一积分事实源;新增规则表不回写余额语义。

-- 积分等级:按累计获得积分(lifetime earned)升序匹配最高达标档。
CREATE TABLE loy_levels (
    level_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    min_points BIGINT NOT NULL CHECK (min_points >= 0),
    status VARCHAR(16) NOT NULL DEFAULT 'ENABLED' CHECK (status IN ('ENABLED', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 任务定义:周期任务按 period_key 幂等(每日/每月一次),一次性任务终身一次。
CREATE TABLE loy_tasks (
    task_id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    points BIGINT NOT NULL CHECK (points > 0),
    period VARCHAR(16) NOT NULL DEFAULT 'ONE_TIME' CHECK (period IN ('ONE_TIME', 'DAILY', 'MONTHLY')),
    status VARCHAR(16) NOT NULL DEFAULT 'ENABLED' CHECK (status IN ('ENABLED', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loy_task_completions (
    completion_id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES loy_tasks (task_id),
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    period_key VARCHAR(16) NOT NULL, -- ONE_TIME='' / DAILY=YYYY-MM-DD / MONTHLY=YYYY-MM
    points BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (task_id, customer_id, period_key)
);

-- 缴费自动积分规则:单行生效(最新 ENABLED),points_per_yuan=每 1 元(100 分)积分。
CREATE TABLE loy_earn_rules (
    rule_id BIGSERIAL PRIMARY KEY,
    points_per_yuan INT NOT NULL CHECK (points_per_yuan >= 0),
    min_cents BIGINT NOT NULL DEFAULT 0 CHECK (min_cents >= 0),
    expire_days INT NOT NULL DEFAULT 0 CHECK (expire_days >= 0), -- 0=永不过期
    status VARCHAR(16) NOT NULL DEFAULT 'ENABLED' CHECK (status IN ('ENABLED', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 流水补充:过期批次标记 + 获得类流水有效期;幂等唯一约束(缴费送/退冲销)。
ALTER TABLE loy_point_entries
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN expired BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX uq_loy_entries_payearn
    ON loy_point_entries (reason, ref_id)
    WHERE ref_id IS NOT NULL AND reason IN ('PAYMENT_EARN', 'PAYMENT_REVERSAL');
