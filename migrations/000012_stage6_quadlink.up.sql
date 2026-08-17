-- 阶段6:四码合一(资产-客户-端口-地址 四码关联,任一码反查单表索引)。
-- 字段权威:docs/contract/fields.md §5.1 + server-ts/src/entities/asset.ts(QuadLink)。
BEGIN;

CREATE TABLE quad_links (
    id                BIGSERIAL PRIMARY KEY,
    asset_id          BIGINT NOT NULL UNIQUE,   -- 四码第1项:资产
    customer_id       BIGINT NOT NULL UNIQUE,   -- 四码第2项:客户(口径裁定,非系统账号)
    port_id           BIGINT NOT NULL UNIQUE,   -- 四码第3项:端口
    address_id        BIGINT NOT NULL UNIQUE,   -- 四码第4项:地址(楼栋级)
    legal_entity_id   BIGINT NOT NULL,          -- 企业归属快照
    legal_entity_name VARCHAR(128) NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'UNLINKED' -- LINKED/CONFLICT/UNLINKED
);

COMMIT;
