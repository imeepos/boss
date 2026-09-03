-- T17 历史回填:order_stages.finished_at 仅回填可从权威痕迹表精确推导的行。
-- 裁定(2026-09-03 stage-finished-at):
--   环节1(下单)完成时间 = orders.created_at(权威:orders 表,下单即建单);
--   环节7(预下发配置)完成时间 = provision_tasks.created_at(权威:下发任务创建时间,
--     stage_event='preConfigOLT' 一一对应);
--   其余环节无带时间戳的权威痕迹表(scan_logs/quad_links/lo_accounts/activation_callbacks/
--     dispatch_tickets 均无 created_at 列;reserve_records/port_change_history 102 空表),
--     保持 NULL——前端已有降级文案(T16 timelineUnfinished),不猜测。
-- 幂等:全部 UPDATE 以 finished_at IS NULL 为守卫,重复执行零副作用。
BEGIN;

-- 环节1 下单:完成时间 = 订单创建时间(权威 orders.created_at)。
-- 同时把历史 DOING 补正为 DONE(下单即完成,与新写入口径一致)。
UPDATE order_stages os
SET result = 'DONE', finished_at = o.created_at
FROM orders o
WHERE os.order_id = o.id AND os.stage = 1 AND os.finished_at IS NULL;

-- 环节7 预下发配置:完成时间 = 下发任务创建时间(权威 provision_tasks.created_at,
-- stage_event='preConfigOLT');多任务取最早创建(PreConfigOLT 幂等可重复建)。
UPDATE order_stages os
SET finished_at = (
  SELECT min(pt.created_at) FROM provision_tasks pt
  WHERE pt.order_id = os.order_id AND pt.stage_event = 'preConfigOLT'
)
WHERE os.stage = 7 AND os.finished_at IS NULL
  AND EXISTS (SELECT 1 FROM provision_tasks pt2
              WHERE pt2.order_id = os.order_id AND pt2.stage_event = 'preConfigOLT');

COMMIT;