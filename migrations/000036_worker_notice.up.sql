-- 师傅公告(worker.yaml /notices):admin 发布/上下架,active 控制师傅端可见。
BEGIN;

CREATE TABLE worker_notices (
    id           BIGSERIAL PRIMARY KEY,
    title        VARCHAR(128) NOT NULL,
    category     VARCHAR(64) NOT NULL DEFAULT '',
    active       BOOLEAN NOT NULL DEFAULT true,   -- true=上架可见
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notices_active ON worker_notices(active, published_at DESC);

COMMIT;
