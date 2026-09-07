-- 迁移 000202(存量开户导入建模迁移):VLAN 四元组挂端口 + legacy_path 承接光缆层级 + 合同月数存档。
-- 机理(裁定 notes/adopted/2026-09-07-legacy-vlan-on-ports):
--  1) 线路 VLAN 四元组(svlan/cvlan/internet_cvlan/tr069_cvlan)落 ports:VLAN 属物理线路属性而非订购属性,
--     换端口重装时 VLAN 随新端口重新生效,TL1 下发取数链(resolver 读端口侧)无需账号-端口二跳 join;
--     全列 SMALLINT 可空,存量 Excel 缺失行照常入档留空,禁止编造补值(设计 §5 红线)。
--  2) ports.legacy_path TEXT 可空:OCC/ODB/OBD/Port 四级光缆段层级以单列无损承接
--     (格式 OCC06/ODB040/OBD01/P05);Excel 三位编码不合 ODN 五位编码规范,不入 odn_facility,
--     ODN 正式建模列 P2 待办。
--  3) lo_accounts.contract_months SMALLINT 可空:存量"月数"存档;orders.buy_months 只服务订单态,
--     存量开户不建历史订单,拆机 1 笔以 lo_accounts.status=CLOSED 表达。
-- 契约同步:fields.md §4.2 ports 表 + §3.1 lo_accounts 备注(同提交);
-- domain 结构体 resource.Port / aaa.LoAccount 补字段(可空列用指针,零值与 NULL 可分)。
-- 全列 ADD COLUMN IF NOT EXISTS 幂等可重放;down 逆序对称 DROP。
BEGIN;

ALTER TABLE ports ADD COLUMN IF NOT EXISTS svlan          SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS cvlan          SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS internet_cvlan SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS tr069_cvlan    SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS legacy_path    TEXT;

ALTER TABLE lo_accounts ADD COLUMN IF NOT EXISTS contract_months SMALLINT;

COMMIT;
