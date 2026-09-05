-- 000186: 标签绑定事件流(P1-T2,adopted note 2026-09-06-asset-tag-p1-wave)。
-- 形态依据 R2 调研(Snipe-IT action_logs / bk-cmdb cc_AuditLog):action 短枚举列,
-- 差异 payload 进 changed JSONB,append-only 只增不改;事件仅在状态 UPDATE 命中行时写入,
-- 状态层幂等天然防重放,故不建 request_id/source 列(裁定见 adopted note)。

BEGIN;

CREATE TABLE tag_events (
    id               BIGSERIAL PRIMARY KEY,
    event_id         UUID NOT NULL DEFAULT gen_random_uuid(),
    tag_id           BIGINT NOT NULL,
    asset_id         BIGINT,
    action           VARCHAR(16) NOT NULL,  -- BIND/UNBIND/RECYCLE
    actor_account_id BIGINT,
    detail           VARCHAR(255) NOT NULL DEFAULT '',
    changed          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_tag_events_event_id ON tag_events(event_id);
CREATE INDEX idx_tag_events_tag ON tag_events(tag_id, created_at DESC);
CREATE INDEX idx_tag_events_asset ON tag_events(asset_id, created_at DESC);

COMMIT;
