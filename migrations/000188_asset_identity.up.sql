-- 000188: 资产身份三要素 SN/MAC/LOID(P3-T2,docs/design/asset-tag-p3-plan.md)。
-- 电信 ONU 身份是网络鉴权标识,全网必须唯一(Snipe-IT 弱去重曾致重复序列号竞态
-- issue #14476,社区补救即唯一性加固 PR #18866)。直接 DB 部分唯一索引硬保证:
-- 空值/空串不占唯一名额;存量行不回填不强制(仅新写入生效,应用层空串归一 NULL)。

BEGIN;

ALTER TABLE assets ADD COLUMN sn   TEXT;
ALTER TABLE assets ADD COLUMN mac  TEXT;
ALTER TABLE assets ADD COLUMN loid TEXT;

CREATE UNIQUE INDEX uq_assets_sn   ON assets(sn)   WHERE sn   IS NOT NULL AND sn   <> '';
CREATE UNIQUE INDEX uq_assets_mac  ON assets(mac)  WHERE mac  IS NOT NULL AND mac  <> '';
CREATE UNIQUE INDEX uq_assets_loid ON assets(loid) WHERE loid IS NOT NULL AND loid <> '';

COMMIT;
