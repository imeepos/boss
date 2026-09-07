-- 迁移 000203(存量导入修正):VLAN 四列 SMALLINT→INTEGER。
-- 机理:apply 实跑 68 行 internet_cvlan/tr069_cvlan 达 102765~104052(源 xlsx R216 等原值,非解析错),
-- 超 SMALLINT 上限 32767;000202 列尚无数据,直接改型零代价;svlan/cvlan 一并放宽防同类复发。
-- 契约同步:fields.md §4.2 四列类型同步(同提交);resource.Port 四字段 *int16→*int。
BEGIN;

ALTER TABLE ports ALTER COLUMN svlan          TYPE INTEGER;
ALTER TABLE ports ALTER COLUMN cvlan          TYPE INTEGER;
ALTER TABLE ports ALTER COLUMN internet_cvlan TYPE INTEGER;
ALTER TABLE ports ALTER COLUMN tr069_cvlan    TYPE INTEGER;

COMMIT;
