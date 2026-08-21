-- Rollback: 移除 Step1 补列。
ALTER TABLE complaints DROP COLUMN IF EXISTS created_at;
ALTER TABLE complaints DROP COLUMN IF EXISTS remote_diagnosis;
ALTER TABLE complaints DROP COLUMN IF EXISTS sla_deadline;

ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS schedule_slot;
ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS splitter_port;
ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS pre_bind_tag;
