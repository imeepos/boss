-- AAA-A2 在线会话表(G3~G6):(loid, session_id) 唯一去重,支撑在线查询/并发限制/CoA 强制下线/僵尸清理。
-- 状态枚举(fields.md 同步):ONLINE 在线 / PENDING_OFFLINE 下线待确认(重试中) / OFFLINE 已下线 / OFFLINE_FAILED 下线失败(重试耗尽)。
BEGIN;

CREATE TABLE aaa_online_sessions (
    id                  BIGSERIAL PRIMARY KEY,
    loid                VARCHAR(32) NOT NULL,           -- 认证账号 LOID
    session_id          VARCHAR(64) NOT NULL,           -- RADIUS Acct-Session-Id
    nas_ip              VARCHAR(64) NOT NULL DEFAULT '',-- NAS IP(CoA 下发目标)
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_update         TIMESTAMPTZ NOT NULL DEFAULT now(), -- Interim/Stop 刷新,僵尸判定依据
    input_octets        BIGINT NOT NULL DEFAULT 0,      -- 累计下行(计账口径)
    output_octets       BIGINT NOT NULL DEFAULT 0,      -- 累计上行
    status              VARCHAR(20) NOT NULL DEFAULT 'ONLINE',
    disconnect_attempts INT NOT NULL DEFAULT 0,         -- Disconnect 已重试次数
    close_reason        VARCHAR(32) NOT NULL DEFAULT '',-- ACCT_STOP/COA_DISCONNECT/ZOMBIE_REAP
    closed_at           TIMESTAMPTZ,
    CONSTRAINT uq_aaa_online_sessions_loid_session UNIQUE (loid, session_id),
    CONSTRAINT ck_aaa_online_sessions_status CHECK (status IN ('ONLINE', 'PENDING_OFFLINE', 'OFFLINE', 'OFFLINE_FAILED'))
);
CREATE INDEX idx_aaa_online_sessions_loid_status ON aaa_online_sessions(loid, status);
CREATE INDEX idx_aaa_online_sessions_last_update ON aaa_online_sessions(last_update);

-- 认证日志失败原因标注(并发超限等),可空兼容存量行。
ALTER TABLE auth_logs ADD COLUMN reason VARCHAR(64);
-- 话单关闭原因标记(僵尸清理补录本地 Stop 等),可空兼容存量行。
ALTER TABLE cdrs ADD COLUMN close_reason VARCHAR(32);

COMMIT;