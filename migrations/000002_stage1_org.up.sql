-- 阶段1扩展:组织与数据权限模型(地区经营树/子公司/部门/岗位/数据范围)
-- 依据 docs/admin/multi-org-audit.md 五维审查结论;原则:角色=功能权限,组织=数据范围。
BEGIN;

-- 经营区域树(与 addresses 地理树解耦;ltree 表达行政层级,集团/大区/省/城市)
CREATE TABLE regions (
    id          BIGSERIAL PRIMARY KEY,
    path        LTREE NOT NULL,               -- 如 root.luzon.metro_manila
    level       SMALLINT NOT NULL,            -- 冗余列 = nlevel(path):1集团 2大区 3省 4城市
    name        VARCHAR(128) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_regions_path UNIQUE (path),
    CONSTRAINT chk_regions_level_range CHECK (level BETWEEN 1 AND 4),
    CONSTRAINT chk_regions_level_consistent CHECK (nlevel(path) = level)
);
CREATE INDEX idx_regions_gist ON regions USING GIST (path);

-- 子公司/法人(品牌隔离的最小隔离单元)
CREATE TABLE legal_entities (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(64) NOT NULL UNIQUE,  -- LEG-A / LEG-B / LEG-C
    name        VARCHAR(128) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 部门(挂靠子公司);同名部门跨子公司各自成行,名称在子公司内唯一。
CREATE TABLE departments (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    name            VARCHAR(128) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_departments_entity_name UNIQUE (legal_entity_id, name)
);

-- 岗位(与角色分离:岗位是组织编制,角色是功能权限)。
-- 同名岗位(如「客服坐席 agent」)在不同子公司/部门各自成行(数据各自隔离),
-- 故 code 在「部门」内唯一,而非全局唯一(见 docs/admin/org-perm-simulation.md O06)。
CREATE TABLE posts (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(64) NOT NULL,         -- dispatcher / cashier / agent / field_tech
    name        VARCHAR(128) NOT NULL,
    dept_id     BIGINT NOT NULL REFERENCES departments(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_posts_dept_code UNIQUE (dept_id, code)
);

-- 岗位→角色(功能权限)绑定:同名岗位跨子公司复用同一角色码,数据范围另由账号决定
CREATE TABLE post_roles (
    post_id BIGINT NOT NULL REFERENCES posts(id),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY (post_id, role_id)
);

-- 账号增组织归属 + 数据范围;region_scope=NULL 表示全集团(总部/审计)。
ALTER TABLE accounts
    ADD COLUMN legal_entity_id BIGINT REFERENCES legal_entities(id),
    ADD COLUMN dept_id          BIGINT REFERENCES departments(id),
    ADD COLUMN post_id          BIGINT REFERENCES posts(id),
    ADD COLUMN region_scope     LTREE;        -- 空=全集团;非空=限该 path 子树

COMMIT;

-- 种子:最小集团结构(幂等),支撑「同角色跨子公司隔离 / 部门 / 岗位 / 数据范围」可重复演示。
-- 组织基线见 docs/admin/multi-org-audit.md §一:Group → Region → LegalEntity → Department → Post。
BEGIN;

-- 经营区域树:集团(1) → 大区(2) → 省(3) → 城市(4),四层行政层级。
INSERT INTO regions (path, level, name) VALUES
    ('root',                 1, '集团'),
    ('root.luzon',           2, '吕宋大区'),
    ('root.luzon.ncr',       3, '首都大区省'),
    ('root.luzon.ncr.manila',4, '马尼拉市'),
    ('root.luzon.ncr.quezon',4, '奎松市'),
    ('root.visayas',         2, '比萨扬大区'),
    ('root.visayas.cebu',    3, '宿务省'),
    ('root.visayas.cebu.city',4, '宿务市'),
    ('root.mindanao',        2, '棉兰老大区'),
    ('root.mindanao.davao',  3, '达沃省'),
    ('root.mindanao.davao.city',4, '达沃市')
ON CONFLICT (path) DO NOTHING;

-- 子公司/法人(品牌隔离):LEG-B 跨大区经营(吕宋 + 棉兰老),其余单大区。
INSERT INTO legal_entities (code, name) VALUES
    ('LEG-A', '主品牌·企业'),
    ('LEG-B', '家庭宽带'),
    ('LEG-C', '批发品牌')
ON CONFLICT (code) DO NOTHING;

-- 部门(挂靠子公司):总部职能 + 属地职能;同名部门跨子公司重复可表达「同岗位不同公司」。
INSERT INTO departments (legal_entity_id, name)
SELECT le.id, d.name
FROM legal_entities le
CROSS JOIN (VALUES
    ('装维调度部'), ('客服部'), ('财务部'), ('网络运维部'), ('市场经营部')
) AS d(name)
WHERE le.code IN ('LEG-A','LEG-B','LEG-C')
ON CONFLICT (legal_entity_id, name) DO NOTHING;

-- 岗位(组织编制,与角色分离):一人一岗,岗位归属部门;同名岗位跨子公司各成一行(数据隔离)。
INSERT INTO posts (code, name, dept_id)
SELECT p.code, p.name, dep.id
FROM (VALUES
    ('field_tech',  '装维师傅',     '装维调度部'),
    ('dispatcher',  '装维调度员',   '装维调度部'),
    ('agent',       '客服坐席',     '客服部'),
    ('cashier',     '财务收款员',   '财务部'),
    ('credit',      '信用专员',     '财务部'),
    ('region_mgr',  '区域经理',     '市场经营部'),
    ('key_acct',    '大客户经理',   '市场经营部'),
    ('noc_engineer','网络运维工程师','网络运维部')
) AS p(code, name, dept_name)
JOIN departments dep ON dep.name = p.dept_name
ON CONFLICT (dept_id, code) DO NOTHING;

-- 岗位→角色(功能权限)绑定;同名岗位跨子公司复用同一角色码,数据范围由账号 region_scope/legal_entity_id 决定。
INSERT INTO post_roles (post_id, role_id)
SELECT po.id, r.id
FROM posts po
JOIN roles r ON r.code = CASE po.code
    WHEN 'field_tech'  THEN 'technician'
    WHEN 'dispatcher'  THEN 'technician'
    WHEN 'agent'       THEN 'ops'
    WHEN 'cashier'     THEN 'ops'
    WHEN 'credit'      THEN 'analyst'
    WHEN 'region_mgr'  THEN 'analyst'
    WHEN 'key_acct'    THEN 'ops'
    WHEN 'noc_engineer'THEN 'resource_admin'
    END
ON CONFLICT DO NOTHING;

COMMIT;
