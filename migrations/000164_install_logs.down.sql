-- 000164 down:严格逆序回滚
DROP TABLE IF EXISTS install_logs;
ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS arrive_lng;
ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS arrive_lat;
ALTER TABLE dispatch_tickets DROP COLUMN IF EXISTS arrived_at;
