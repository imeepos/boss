-- quad_links 三码唯一索引收窄到"活跃链路"(LINKED/CONFLICT)。
-- 事故(2026-08-23,102 验收):同地址第二单 charge → AutoPreScan → applyTag
--   撞 uq_quad_links_address (23505),UNLINKED(拆机/未扫码释放)历史行永久占坑。
-- 根因:000097 以"000086 非空唯一已覆盖 _active 索引"为由删掉 000056 的
--   active 部分唯一索引——该冗余论只对"永远只有一行"成立;000088 customer
--   放开 1:N 后,同一地址/端口/资产出现"历史 UNLINKED + 新预绑定"多行是
--   正常生命周期,非空唯一把释放行也计入,语义与 000097 自述的
--   "至多一条非空活跃链路"矛盾。
-- 裁定:唯一性只约束活跃链路;UNLINKED 行是留痕历史,允许多行共存。
BEGIN;

DROP INDEX IF EXISTS uq_quad_links_asset;
DROP INDEX IF EXISTS uq_quad_links_port;
DROP INDEX IF EXISTS uq_quad_links_address;

CREATE UNIQUE INDEX uq_quad_links_asset
    ON quad_links (asset_id)
    WHERE asset_id IS NOT NULL AND status IN ('LINKED', 'CONFLICT');

CREATE UNIQUE INDEX uq_quad_links_port
    ON quad_links (port_id)
    WHERE port_id IS NOT NULL AND status IN ('LINKED', 'CONFLICT');

CREATE UNIQUE INDEX uq_quad_links_address
    ON quad_links (address_id)
    WHERE address_id IS NOT NULL AND status IN ('LINKED', 'CONFLICT');

COMMIT;
