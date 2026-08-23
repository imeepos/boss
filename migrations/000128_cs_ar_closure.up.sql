-- S3 CS/AR business closure: lifecycle writes, rule outputs, approvals and replay audit.
BEGIN;
ALTER TABLE cs_ticket_extensions ADD COLUMN IF NOT EXISTS assigned_to BIGINT;
ALTER TABLE cs_ticket_extensions ADD COLUMN IF NOT EXISTS updated_by BIGINT;
ALTER TABLE cs_callbacks ADD COLUMN IF NOT EXISTS dispatch_status VARCHAR(16) NOT NULL DEFAULT 'PENDING';
ALTER TABLE cs_callbacks ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMPTZ;
ALTER TABLE ar_credit_profiles ADD COLUMN IF NOT EXISTS evaluated_at TIMESTAMPTZ;
ALTER TABLE ar_writeoffs ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'PENDING';
ALTER TABLE ar_writeoffs ADD COLUMN IF NOT EXISTS requested_by BIGINT;
ALTER TABLE ar_writeoffs ADD COLUMN IF NOT EXISTS approved_note TEXT;
CREATE TABLE IF NOT EXISTS ar_replay_events (
 id BIGSERIAL PRIMARY KEY,
 customer_id BIGINT NOT NULL REFERENCES customers(id),
 source_type VARCHAR(32) NOT NULL,
 source_id BIGINT NOT NULL,
 event_type VARCHAR(32) NOT NULL,
 payload JSONB NOT NULL DEFAULT '{}'::jsonb,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(source_type, source_id, event_type)
);
CREATE INDEX IF NOT EXISTS idx_ar_replay_customer ON ar_replay_events(customer_id, created_at);
COMMIT;
