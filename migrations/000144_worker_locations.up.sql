-- 师傅实时位置：每次上报留存轨迹，最新位置按 reported_at 反查。
CREATE TABLE worker_locations (
    id          BIGSERIAL PRIMARY KEY,
    worker_id   BIGINT NOT NULL REFERENCES workers(id),
    lat         DOUBLE PRECISION NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lng         DOUBLE PRECISION NOT NULL CHECK (lng BETWEEN -180 AND 180),
    accuracy_m  REAL NOT NULL DEFAULT 0 CHECK (accuracy_m >= 0),
    speed_mps   REAL NOT NULL DEFAULT 0 CHECK (speed_mps >= 0),
    bearing     REAL NOT NULL DEFAULT 0 CHECK (bearing >= 0 AND bearing < 360),
    reported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_worker_locations_latest ON worker_locations(worker_id, reported_at DESC, id DESC);
CREATE INDEX idx_worker_locations_time ON worker_locations(reported_at DESC);
