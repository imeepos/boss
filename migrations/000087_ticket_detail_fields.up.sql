-- 000087: 师傅端工单详情 OpenAPI TicketDetail 缺失字段补列。
-- 背景：schemas.yaml::TicketDetail 定义了 12 个工单头字段，handler 实际只返 6 个。
-- 本轮补列让后端能存值；写入环节的补写见 Step3 (pg_workflow.go)。
-- complaint-type-map.md 定义 complaints.type → faultTypeLabel 映射。
BEGIN;

-- complaints 表：报障时间（迁移时漏列，DEFAULT now() 给存量行补值）、
-- 远程诊断结论（客服/系统填写）、SLA 截止时间（派单时写入，比实时计算更稳）。
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS created_at     TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS remote_diagnosis TEXT        NOT NULL DEFAULT '';
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS sla_deadline    TIMESTAMPTZ;  -- NULL=无 SLA 或已过期

-- dispatch_tickets 表：预约时间段（环节8 派单时由调度填入）、
-- 分光器端口（环节3/5 端口预占时由系统填入）、预绑定标签（环节5 标签预绑定时系统填入）。
ALTER TABLE dispatch_tickets ADD COLUMN IF NOT EXISTS schedule_slot  VARCHAR(64)  NOT NULL DEFAULT '';
ALTER TABLE dispatch_tickets ADD COLUMN IF NOT EXISTS splitter_port  VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE dispatch_tickets ADD COLUMN IF NOT EXISTS pre_bind_tag   VARCHAR(64)  NOT NULL DEFAULT '';

COMMIT;
