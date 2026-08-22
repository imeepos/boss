-- Q2 话单补偿:cdrs 补 Kafka 投递状态,实时链路失败可从 PG 权威侧补投。
-- 既有行默认 PENDING:首轮补偿按批次限速重放,最终一致。
BEGIN;

ALTER TABLE cdrs
    ADD COLUMN kafka_status VARCHAR(16) NOT NULL DEFAULT 'PENDING'; -- PENDING/SENT/FAILED

CREATE INDEX idx_cdrs_kafka_status ON cdrs(kafka_status, id);

COMMIT;
