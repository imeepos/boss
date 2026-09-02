-- 迁移 000178(阶段7 TL1/网管北向):NMS 端点 + PON 定位 + ONUNO 分配器(设计 §8)。
-- 机理:TL1 对接(T4)需三层新数据——
--  1) provision_nms:法人级 NMS(U2000)端点配置,legal_entity_id UNIQUE 一法人一端点;
--     取数优先级 表 > 环境变量(设计 §7)。pass_cipher 复用 config_secrets 加解密(T5 实现)。
--  2) resources.nms_oltid:OLT 行的 U2000 侧标识;ports 的 PON 四维(pon_frame/slot/port）
--     + onu_no(NULL=未分配)组成 PONID="NA-<frame>-<slot>-<port>" 与装维定位。
--  3) pon_onu_alloc:每 (OLT,PON) 的 ONUNO 0~127 分配器,幂等自增分配,四列联合主键防重复。
-- 契约同步:fields.md §下发 增列 + domain-map.md provision 域增 tl1 子包/app resolver(同提交)。
BEGIN;

-- 1) 法人级 NMS 端点(一法人一行)
CREATE TABLE provision_nms (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL UNIQUE REFERENCES legal_entities(id),
    host            VARCHAR(128) NOT NULL,
    port            INTEGER NOT NULL DEFAULT 13027,
    protocol        VARCHAR(8) NOT NULL DEFAULT 'tcp',  -- tcp(P2 扩 ssl)
    username        VARCHAR(32) NOT NULL,
    pass_cipher     TEXT NOT NULL,                      -- 复用 config_secrets 加解密 helpers(T5)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2) OLT 行 U2000 侧标识 + 端口 PON 定位
ALTER TABLE resources ADD COLUMN IF NOT EXISTS nms_oltid VARCHAR(128); -- OLT 行:U2000 侧标识
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_frame SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_slot  SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_port  SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS onu_no    SMALLINT;  -- NULL=未分配

-- 3) ONUNO 分配器(每 (OLT,PON) 幂等自增 next_no,首 0 其后 1,2,...)
CREATE TABLE pon_onu_alloc (
    olt_resource_id BIGINT NOT NULL REFERENCES resources(id),
    pon_frame SMALLINT NOT NULL,
    pon_slot  SMALLINT NOT NULL,
    pon_port  SMALLINT NOT NULL,
    next_no   SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (olt_resource_id, pon_frame, pon_slot, pon_port)
);

COMMIT;
