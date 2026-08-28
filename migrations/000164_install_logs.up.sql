-- 000164: 施工回单 + 派单工单到场打卡。
-- 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。
-- 数据分层:install_logs L5(派生);dispatch_tickets L5 增列(派生事实)。

-- 到场打卡事实:派单工单增 3 列(arrived_at / arrive_lat / arrive_lng)。
-- 不写回订单状态;只写事实;GIS 施工实时图层读 arrive_lat/lng(权威点位)。
ALTER TABLE dispatch_tickets
    ADD COLUMN IF NOT EXISTS arrived_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS arrive_lat  DOUBLE PRECISION
        CHECK (arrive_lat IS NULL OR arrive_lat BETWEEN -90 AND 90),
    ADD COLUMN IF NOT EXISTS arrive_lng  DOUBLE PRECISION
        CHECK (arrive_lng IS NULL OR arrive_lng BETWEEN -180 AND 180);

-- 施工回单(install_logs):N:1 dispatch_tickets(同一工单允许多次回单,补录场景);
-- 师傅 ID + 姓名快照(防工人离职后关联断裂);photos JSONB 存 MinIO attachments.id 列表;
-- status:OPEN=已提交未签收、COMPLETED=已签收、REJECTED=拒签。
CREATE TABLE install_logs (
    id              BIGSERIAL PRIMARY KEY,
    ticket_id       BIGINT NOT NULL REFERENCES dispatch_tickets(id),
    order_id        BIGINT NOT NULL REFERENCES orders(id),
    worker_id       BIGINT NOT NULL REFERENCES workers(id),
    worker_name     VARCHAR(64) NOT NULL,
    photos          JSONB NOT NULL DEFAULT '[]'::jsonb,
    sign_name       VARCHAR(64),
    sign_image_url  VARCHAR(255),
    signed_at       TIMESTAMPTZ,
    note            VARCHAR(500),
    status          VARCHAR(16) NOT NULL DEFAULT 'OPEN'
                    CHECK (status IN ('OPEN','COMPLETED','REJECTED')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_install_logs_ticket ON install_logs(ticket_id);
CREATE INDEX idx_install_logs_order ON install_logs(order_id);
CREATE INDEX idx_install_logs_status ON install_logs(status);
-- 同一工单同一状态最多一条 OPEN(防重复提交);COMPLETED/REJECTED 允许多条(补录/驳回重提)
CREATE UNIQUE INDEX uq_install_logs_ticket_open ON install_logs(ticket_id) WHERE status = 'OPEN';

COMMENT ON COLUMN dispatch_tickets.arrived_at IS '师傅到场打卡时间(派生事实,GIS 施工实时图层读 arrive_lat/lng)';
COMMENT ON COLUMN dispatch_tickets.arrive_lat IS '到场打卡纬度(WGS84)';
COMMENT ON COLUMN dispatch_tickets.arrive_lng IS '到场打卡经度(WGS84)';
COMMENT ON TABLE  install_logs                IS '施工回单(现场照片+签收,L5 派生)';
