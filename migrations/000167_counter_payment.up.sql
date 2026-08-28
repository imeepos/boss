-- 柜面收款(会议纪要 2026-08-28-柜面现金收款):payments 凭证要素三列 +
-- 柜台日结实点回填表 + 柜面收款/日结权限码。
-- 口径:柜面现金记 cash、扫码记 wechat/alipay、POS 记 card(按资金通道归类,
-- offline 维持师傅个人代收专属语义);method 管资金通道,网点/操作员列管人员归因。
BEGIN;

-- 凭证要素:网点/柜台班次/操作员(收款落账同事务写入;操作员由服务端取登录态,前端不传)。
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS site_name     VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS counter_code  VARCHAR(64)  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS operator_name VARCHAR(64)  NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_payments_site_date ON payments (site_name, created_at);

-- 柜台日结:T+0 只读汇总 + 操作员实点回填;不平差异由服务层输出 [paycheck] DIFF 日志。
CREATE TABLE payment_daily_closings (
    id             BIGSERIAL PRIMARY KEY,
    closing_date   DATE NOT NULL,
    site_name      VARCHAR(128) NOT NULL DEFAULT '',
    operator_name  VARCHAR(64)  NOT NULL DEFAULT '',
    system_amount  NUMERIC(14,2) NOT NULL DEFAULT 0, -- 回填时系统净额快照(SUCCESS-REFUNDED)
    counted_amount NUMERIC(14,2) NOT NULL DEFAULT 0, -- 钱箱实点金额
    created_by     VARCHAR(64) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_pay_closing_date_site_operator UNIQUE (closing_date, site_name, operator_name)
);

-- 权限码:柜面收款登记(按钮级,写操作);ops 不默认授予(不相容岗位分离,按需绑定)。
INSERT INTO permissions (code, name) VALUES
    ('menu:payment:cash', '计费与账务·柜面收款登记'),
    ('menu:daily-close',  '计费与账务·柜台日结')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;

-- 日结报表为查看级:随既有缴费管理可见角色(ops)开放。
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.code = 'menu:daily-close'
WHERE r.code = 'ops'
ON CONFLICT DO NOTHING;

COMMIT;
