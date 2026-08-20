-- D4(db-design-review): orders 补渠道下钻与数据权限裁剪索引。
-- channel_id 渠道维度下钻、region_path 按 region_scope 子树裁剪此前均全表扫。
BEGIN;

CREATE INDEX idx_orders_channel ON orders(channel_id);
CREATE INDEX idx_orders_region ON orders(region_path);

COMMIT;
