-- 指标目录与数据质量规则基础
-- 指标目录：注册指标定义、口径、版本、负责人、刷新频率、状态和血缘占位
-- 数据质量规则：覆盖完整性/唯一性/及时性/跨域一致性，异常进入补偿任务中心（已合并）

CREATE TABLE metric_catalog (
    id              BIGSERIAL PRIMARY KEY,
    key             VARCHAR(64) NOT NULL UNIQUE,
    name            VARCHAR(128) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    formula         TEXT NOT NULL DEFAULT '',
    unit            VARCHAR(32) NOT NULL DEFAULT '',
    dimensions      JSONB NOT NULL DEFAULT '[]'::jsonb,
    refresh_cadence VARCHAR(32) NOT NULL DEFAULT 'DAILY',
    owner           VARCHAR(64) NOT NULL DEFAULT '',
    domain          VARCHAR(32) NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('DRAFT','ACTIVE','DEPRECATED','ARCHIVED')),
    CHECK (refresh_cadence IN ('REALTIME','HOURLY','DAILY','WEEKLY','MONTHLY','ON_DEMAND'))
);
CREATE INDEX idx_metric_catalog_key ON metric_catalog (key);
CREATE INDEX idx_metric_catalog_owner ON metric_catalog (owner);
CREATE INDEX idx_metric_catalog_status ON metric_catalog (status);

CREATE TABLE metric_quality_rules (
    id              BIGSERIAL PRIMARY KEY,
    rule_key        VARCHAR(64) NOT NULL UNIQUE,
    name            VARCHAR(128) NOT NULL,
    scope           VARCHAR(64) NOT NULL,
    check_expr      TEXT NOT NULL,
    threshold_expr  TEXT NOT NULL DEFAULT '',
    severity        VARCHAR(16) NOT NULL DEFAULT 'WARN',
    owner           VARCHAR(64) NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (severity IN ('INFO','WARN','CRITICAL'))
);
CREATE INDEX idx_metric_quality_rules_enabled ON metric_quality_rules (enabled);

-- 种子：把现有 analytics 域的五大核心指标先注册到目录
INSERT INTO metric_catalog (key, name, description, formula, unit, dimensions, refresh_cadence, owner, domain, status)
VALUES
    ('portUtilization', '端口利用率', '已用端口/总端口', 'COUNT(ports WHERE status=''USED'')/COUNT(ports)', 'ratio', '["region"]'::jsonb, 'DAILY', 'sysadmin', 'OSS', 'ACTIVE'),
    ('installConversion', '开通转化率', 'DONE 订单 / 全部订单', 'COUNT(orders WHERE status=''DONE'')/COUNT(orders)', 'ratio', '["region"]'::jsonb, 'DAILY', 'sysadmin', 'ORD', 'ACTIVE'),
    ('maintenanceCostPerUser', '每用户维护成本', '维护总成本/在网客户', 'SUM(cost)/COUNT(customers WHERE service_status=''ACTIVE'')', 'currency', '["region"]'::jsonb, 'MONTHLY', 'sysadmin', 'AMS', 'ACTIVE'),
    ('assetHealth', '资产健康度', '健康资产/总资产', 'COUNT(assets WHERE status IN(''IN_STOCK'',''DEPLOYED''))/COUNT(assets)', 'ratio', '["region"]'::jsonb, 'DAILY', 'sysadmin', 'AMS', 'ACTIVE'),
    ('regionROI', '区域投资回报率', '已收款/扩容端口成本', 'SUM(payments.amount WHERE status=''SUCCESS'')/SUM(capacity_cost)', 'ratio', '["region"]'::jsonb, 'MONTHLY', 'sysadmin', 'BI', 'ACTIVE');