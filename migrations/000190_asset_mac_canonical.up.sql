-- 000190: MAC 规范化存储与表达式唯一约束(P4-T1,docs/design/asset-tag-p4-plan.md §四)。
-- 应用层自本版起把 mac 写入归一为大写冒号规范形(AA:BB:CC:DD:EE:FF;输入仍接受
-- 冒号/横杠/裸 hex 三形态)。本迁移做两件事:
-- 1. 存量 mac 按与写入口径完全一致的 SQL(去分隔符→大写→两两配冒号)UPDATE 归一;
--    现为 0 行非空,属空转,但必须存在保持向前一致。
-- 2. uq_assets_mac 自列值部分唯一索引升级为规范化表达式唯一索引
--    upper(regexp_replace(mac,'[:. -]','','g'))(大写+去冒号/横杠/点/空格):
--    直写 SQL 绕过应用层也撞同一唯一空间,与应用层归一双保险;
--    归一后非空才占唯一名额(NULL/空串/纯分隔符不占)。

BEGIN;

UPDATE assets
   SET mac = rtrim(regexp_replace(upper(regexp_replace(mac, '[:. -]', '', 'g')), '(..)', '\1:', 'g'), ':')
 WHERE mac IS NOT NULL
   AND mac <> '';

DROP INDEX IF EXISTS uq_assets_mac;

CREATE UNIQUE INDEX uq_assets_mac
    ON assets (upper(regexp_replace(mac, '[:. -]', '', 'g')))
    WHERE upper(regexp_replace(mac, '[:. -]', '', 'g')) <> '';

COMMIT;
