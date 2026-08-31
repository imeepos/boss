-- 000174: 派单工单站点坐标快照(site_lat/site_lng)。
-- 派单时刻从 orders.address_id → addresses.geom 物化到工单,半径闸门/距离计算
-- 不依赖可变主数据(同 orders.price_snapshot 快照口径);与到场打卡事实
-- arrive_lat/lng(000164,师傅侧)互不混用。空=地址树节点无 geom,闸门跳过。

ALTER TABLE dispatch_tickets
    ADD COLUMN site_lat DOUBLE PRECISION
        CHECK (site_lat IS NULL OR site_lat BETWEEN -90 AND 90),
    ADD COLUMN site_lng DOUBLE PRECISION
        CHECK (site_lng IS NULL OR site_lng BETWEEN -180 AND 180);
