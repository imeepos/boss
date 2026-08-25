-- ODN 核心链路设备补地理坐标(lat/lng),与 GIS 地图点位关联。
-- 决策:用户要求"所有物料都要有地理位置 与地图关联";odn_site/odn_facility 已有坐标,
-- odn_device 缺失(000081 遗漏)。可空列,历史数据无坐标不影响建档,无坐标设备不上地图点位。
BEGIN;

ALTER TABLE odn_device
    ADD COLUMN lat DOUBLE PRECISION,
    ADD COLUMN lng DOUBLE PRECISION;

COMMIT;
