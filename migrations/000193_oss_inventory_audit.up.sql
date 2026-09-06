-- P5-W3 资源台账稽核:RESERVED 超时阈值参数(biz_params 热更)。
-- 稽核为纯只读 SQL(ports/resources/quad_links/port_change_history),无表结构变更。
BEGIN;
INSERT INTO biz_params(key, value, description) VALUES
 ('resource.audit.reservedStaleHours', '48', '资源台账稽核:RESERVED 态超时阈值(小时,默认48)')
ON CONFLICT (key) DO NOTHING;
COMMIT;