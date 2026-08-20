-- 附件登记表:三端(admin/user/worker)登录后上传,对象实体存 MinIO。
-- uploader_type 对齐 apikey 主体模型(account/worker/customer),记录上传者身份。
CREATE TABLE attachments (
    id            BIGSERIAL PRIMARY KEY,
    object_key    TEXT        NOT NULL UNIQUE,
    file_name     TEXT        NOT NULL DEFAULT '',
    content_type  TEXT        NOT NULL DEFAULT '',
    size_bytes    BIGINT      NOT NULL DEFAULT 0,
    uploader_type TEXT        NOT NULL CHECK (uploader_type IN ('account', 'worker', 'customer')),
    uploader_id   BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attachments_uploader ON attachments (uploader_type, uploader_id);
