-- 000213: 施工进度记录(P0-B):资源级现场完成量/坐标/照片/上报人;
-- (project_id, facility_code, client_msg_id) 唯一约束实现弱网重传幂等,重复上报不重复计量。
CREATE TABLE construction_progress (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT NOT NULL REFERENCES construction_projects(id),
    facility_code VARCHAR(8) NOT NULL REFERENCES odn_facility(code),
    done_qty      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (done_qty >= 0),
    lat           DOUBLE PRECISION,
    lng           DOUBLE PRECISION,
    note          TEXT,
    photo_ids     BIGINT[] NOT NULL DEFAULT '{}',
    client_msg_id VARCHAR(64) NOT NULL,
    reported_by   BIGINT REFERENCES accounts,
    reported_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, facility_code, client_msg_id)
);
CREATE INDEX idx_construction_progress_project ON construction_progress (project_id, id DESC);
