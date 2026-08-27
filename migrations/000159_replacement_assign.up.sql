-- 设备更换单接入执行流(adopted note 2026-08-27-replacement-ticket-flow):
-- replacements 只有建单+看单,status PENDING 为吸收态(RPL-20260827-806379389990 实证)。
-- 补派单快照三列:worker FK + name 冗余 + 完成时间;状态机流转由应用层守卫 UPDATE 驱动。
BEGIN;

ALTER TABLE replacements
    ADD COLUMN worker_id   BIGINT REFERENCES workers(id),
    ADD COLUMN worker_name VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN finished_at TIMESTAMPTZ;

CREATE INDEX idx_replacements_worker ON replacements(worker_id);

COMMIT;
