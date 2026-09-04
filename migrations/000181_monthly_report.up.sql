-- 000181: 月度填报事实域(BI 经营分析域,T19)。
-- 三事实表粒度=月×区域,PRIMARY KEY(month,region) 即 UNIQUE(month,region);
-- 区域白名单 51 Barangay 种子提取自 docs/books/模板_月度填报.xlsx _RegionList(权威输入)。
-- 派生列(期末在用/主营总收入)用 GENERATED ALWAYS 存储列在库端闭环:
-- 服务端计算口径,导入/接口无外部写入入口(与模板灰色列「公式-勿填」对齐)。
-- 菜单权限 menu:monthly 沿 000135/000149/000165 先例登记 permissions+sysadmin。

BEGIN;

CREATE TABLE monthly_regions (
    region     TEXT PRIMARY KEY,
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO monthly_regions (region) VALUES
('Anilao'),('Atlag'),('Babatnin'),('Bagna'),('Bagong Bayan'),
('Balayong'),('Balite'),('Bangkal'),('Barihan'),('Bulihan'),
('Bungahan'),('Caingin'),('Calero'),('Caliligawan'),('Canalate'),
('Caniogan'),('Catmon'),('Cofradia'),('Dakila'),('Guinhawa'),
('Ligas'),('Liyang'),('Longos'),('Look 1st'),('Look 2nd'),
('Lugam'),('Mabolo'),('Mambog'),('Masile'),('Matimbo'),
('Mojon'),('Namayan'),('Niugan'),('Pamarawan'),('Panasahan'),
('Pinagbakahan'),('San Agustin'),('San Gabriel'),('San Juan'),('San Pablo'),
('San Vicente'),('Santiago'),('Santisima Trinidad'),('Santo Cristo'),('Santo Niño'),
('Santo Rosario'),('Santol'),('Sumapang Bata'),('Sumapang Matanda'),('Taal'),
('Tikay');

CREATE TABLE monthly_user_revenue (
    month                TEXT NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    region               TEXT NOT NULL REFERENCES monthly_regions(region),
    opening_active       BIGINT NOT NULL DEFAULT 0 CHECK (opening_active >= 0),
    new_users            BIGINT NOT NULL DEFAULT 0 CHECK (new_users >= 0),
    churned_users        BIGINT NOT NULL DEFAULT 0 CHECK (churned_users >= 0),
    adjusted_users       BIGINT NOT NULL DEFAULT 0 CHECK (adjusted_users >= 0),
    broadband_revenue    BIGINT NOT NULL DEFAULT 0 CHECK (broadband_revenue >= 0),
    value_added_revenue  BIGINT NOT NULL DEFAULT 0 CHECK (value_added_revenue >= 0),
    onetime_charge       BIGINT NOT NULL DEFAULT 0 CHECK (onetime_charge >= 0),
    discount_amount      BIGINT NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    refund_reversal      BIGINT NOT NULL DEFAULT 0 CHECK (refund_reversal >= 0),
    closing_active       BIGINT GENERATED ALWAYS AS
                         (opening_active + new_users - churned_users + adjusted_users) STORED,
    total_revenue        BIGINT GENERATED ALWAYS AS
                         (broadband_revenue + value_added_revenue + onetime_charge
                          - discount_amount - refund_reversal) STORED,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (month, region)
);

CREATE TABLE monthly_network_delivery (
    month               TEXT NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    region              TEXT NOT NULL REFERENCES monthly_regions(region),
    install_requests    BIGINT NOT NULL DEFAULT 0 CHECK (install_requests >= 0),
    ontime_completions  BIGINT NOT NULL DEFAULT 0 CHECK (ontime_completions >= 0),
    ports_deployed      BIGINT NOT NULL DEFAULT 0 CHECK (ports_deployed >= 0),
    ports_active        BIGINT NOT NULL DEFAULT 0 CHECK (ports_active >= 0),
    fault_reports       BIGINT NOT NULL DEFAULT 0 CHECK (fault_reports >= 0),
    repair_hours        BIGINT NOT NULL DEFAULT 0 CHECK (repair_hours >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (month, region)
);

CREATE TABLE monthly_finance_cost (
    month              TEXT NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    region             TEXT NOT NULL REFERENCES monthly_regions(region),
    invoiced_amount    BIGINT NOT NULL DEFAULT 0 CHECK (invoiced_amount >= 0),
    collected_amount   BIGINT NOT NULL DEFAULT 0 CHECK (collected_amount >= 0),
    receivable_ending  BIGINT NOT NULL DEFAULT 0 CHECK (receivable_ending >= 0),
    direct_cost        BIGINT NOT NULL DEFAULT 0 CHECK (direct_cost >= 0),
    fixed_cost         BIGINT NOT NULL DEFAULT 0 CHECK (fixed_cost >= 0),
    capex_invest       BIGINT NOT NULL DEFAULT 0 CHECK (capex_invest >= 0),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (month, region)
);

INSERT INTO permissions (code, name)
VALUES ('menu:monthly', '数字孪生与经营·月度填报')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.code = 'menu:monthly'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
