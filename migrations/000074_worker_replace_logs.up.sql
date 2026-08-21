-- 师傅换件登记流水(POST /tickets/{no}/replace 此前仅审计,GET 返回占位步骤)。
BEGIN;

CREATE TABLE worker_replace_logs (
    id         BIGSERIAL PRIMARY KEY,
    worker_id  BIGINT      NOT NULL REFERENCES workers(id),
    ticket_no  VARCHAR(64) NOT NULL,
    old_epc    VARCHAR(64) NOT NULL DEFAULT '',
    new_epc    VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_worker_replace_ticket ON worker_replace_logs(ticket_no, created_at DESC);

COMMIT;
