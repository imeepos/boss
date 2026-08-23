CREATE TABLE etl_job (
 id BIGSERIAL PRIMARY KEY, job_key VARCHAR(128) NOT NULL UNIQUE, name VARCHAR(128) NOT NULL,
 source_domain VARCHAR(64) NOT NULL, target_domain VARCHAR(64) NOT NULL, cron_expr VARCHAR(128) NOT NULL DEFAULT '',
 refresh_cadence VARCHAR(32) NOT NULL DEFAULT 'DAILY', owner VARCHAR(128) NOT NULL DEFAULT '', enabled BOOLEAN NOT NULL DEFAULT TRUE,
 last_run_at TIMESTAMPTZ, last_status VARCHAR(16) NOT NULL DEFAULT 'RUNNING', last_duration_ms BIGINT NOT NULL DEFAULT 0,
 last_rows_affected BIGINT NOT NULL DEFAULT 0, lateness_threshold INTEGER NOT NULL DEFAULT 60 CHECK (lateness_threshold > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE etl_job_run (
 id BIGSERIAL PRIMARY KEY, job_key VARCHAR(128) NOT NULL REFERENCES etl_job(job_key), started_at TIMESTAMPTZ NOT NULL,
 finished_at TIMESTAMPTZ, status VARCHAR(16) NOT NULL, rows_affected BIGINT NOT NULL DEFAULT 0, error TEXT NOT NULL DEFAULT ''
);
INSERT INTO etl_job (job_key,name,source_domain,target_domain,cron_expr,refresh_cadence,owner,lateness_threshold) VALUES
 ('order_projection','订单投影','order','projection','*/5 * * * *','5MIN','ops',15),
 ('ar_aging_snapshot','AR账龄快照','billing','projection','0 * * * *','HOURLY','finance',120),
 ('metric_quality_scan','指标质量扫描','metric','projection','0 * * * *','HOURLY','analyst',120),
 ('compensation_reconcile','补偿对账','report','projection','*/15 * * * *','15MIN','ops',30),
 ('billing_projection','账务投影','billing','projection','*/10 * * * *','10MIN','finance',30),
 ('customer_projection','客户投影','customer','projection','*/10 * * * *','10MIN','crm',30);
